package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"

	"github.com/stevedylandev/andromeda/pkg/web"
)

var md = goldmark.New(
	goldmark.WithExtensions(extension.Strikethrough, extension.Table, extension.TaskList, extension.Linkify),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
)

func buildTemplates() (map[string]*template.Template, error) {
	funcs := template.FuncMap{
		"shortSHA": func(s string) string {
			if len(s) >= 8 {
				return s[:8]
			}
			return s
		},
		"humanSize": humanSize,
		"timeAgo":   timeAgo,
	}

	pages, err := fs.Glob(appFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	out := make(map[string]*template.Template, len(pages))
	for _, page := range pages {
		if strings.HasSuffix(page, "/base.html") {
			continue
		}
		tmpl, err := template.New("").Funcs(funcs).ParseFS(appFS, "templates/base.html", page)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", page, err)
		}
		out[path.Base(page)] = tmpl
	}
	return out, nil
}

func (a *App) renderPage(w http.ResponseWriter, name string, data any) {
	tmpl, ok := a.Templates[name]
	if !ok {
		a.Log.Error("template missing", "name", name)
		http.Error(w, "template missing", http.StatusInternalServerError)
		return
	}
	web.Render(tmpl, w, name, data, a.Log)
}

func renderMarkdown(source, rawBase string) (template.HTML, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		return "", err
	}
	out := buf.String()
	if rawBase != "" {
		out = rewriteRelativeImages(out, rawBase)
	}
	return template.HTML(out), nil
}

var imgSrcRe = regexp.MustCompile(`(<img\b[^>]*?\bsrc=")([^"]+)(")`)

func rewriteRelativeImages(html, base string) string {
	return imgSrcRe.ReplaceAllStringFunc(html, func(m string) string {
		sub := imgSrcRe.FindStringSubmatch(m)
		src := sub[2]
		if isAbsoluteRef(src) {
			return m
		}
		src = strings.TrimPrefix(src, "./")
		return sub[1] + base + src + sub[3]
	})
}

func isAbsoluteRef(s string) bool {
	if s == "" {
		return true
	}
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "#") {
		return true
	}
	if strings.HasPrefix(s, "data:") || strings.HasPrefix(s, "//") {
		return true
	}
	return strings.Contains(s, "://")
}

func renderBlobLines(source string) template.HTML {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	lines := strings.Split(source, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	var buf bytes.Buffer
	buf.WriteString(`<table class="blob-code"><tbody>`)
	for i, line := range lines {
		n := i + 1
		fmt.Fprintf(&buf,
			`<tr id="L%d"><td class="line-num"><a href="#L%d">%d</a></td><td class="line-code"><pre>%s</pre></td></tr>`,
			n, n, n, template.HTMLEscapeString(line),
		)
	}
	buf.WriteString(`</tbody></table>`)
	return template.HTML(buf.String())
}

func findReadme(tree *object.Tree) (string, bool) {
	names := []string{"README.md", "readme.md", "Readme.md", "README", "readme", "README.txt", "readme.txt"}
	for _, n := range names {
		f, err := tree.File(n)
		if err == nil {
			content, err := f.Contents()
			if err == nil {
				return content, true
			}
		}
	}
	return "", false
}

func convertPatch(patch *object.Patch) ([]FilePatch, DiffStats) {
	var files []FilePatch
	stats := DiffStats{}
	for _, fp := range patch.FilePatches() {
		from, to := fp.Files()
		fromName, toName := "", ""
		if from != nil {
			fromName = from.Path()
		}
		if to != nil {
			toName = to.Path()
		}
		out := FilePatch{From: fromName, To: toName, IsBin: fp.IsBinary()}
		var all []DiffLine
		oldNum, newNum := 1, 1
		for _, ch := range fp.Chunks() {
			kind := ""
			switch ch.Type() {
			case 1: // Add
				kind = "add"
			case 2: // Delete
				kind = "del"
			default:
				kind = "ctx"
			}
			text := ch.Content()
			lines := strings.Split(text, "\n")
			if len(lines) > 0 && lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1]
			}
			for _, l := range lines {
				dl := DiffLine{Kind: kind, Text: l}
				switch kind {
				case "add":
					dl.NewNum = newNum
					newNum++
					out.Added++
				case "del":
					dl.OldNum = oldNum
					oldNum++
					out.Removed++
				default:
					dl.OldNum = oldNum
					dl.NewNum = newNum
					oldNum++
					newNum++
				}
				all = append(all, dl)
			}
		}
		out.Hunks = collapseContext(all, diffContextLines)
		stats.Files++
		stats.Added += out.Added
		stats.Removed += out.Removed
		files = append(files, out)
	}
	return files, stats
}

const diffContextLines = 3

func collapseContext(lines []DiffLine, ctx int) []DiffHunk {
	if len(lines) == 0 {
		return nil
	}
	keep := make([]bool, len(lines))
	for i, l := range lines {
		if l.Kind == "add" || l.Kind == "del" {
			start := i - ctx
			if start < 0 {
				start = 0
			}
			end := i + ctx
			if end >= len(lines) {
				end = len(lines) - 1
			}
			for j := start; j <= end; j++ {
				keep[j] = true
			}
		}
	}
	var hunks []DiffHunk
	var current *DiffHunk
	for i, l := range lines {
		if keep[i] {
			if current == nil {
				current = &DiffHunk{}
			}
			current.Lines = append(current.Lines, l)
		} else if current != nil {
			hunks = append(hunks, *current)
			current = nil
		}
	}
	if current != nil {
		hunks = append(hunks, *current)
	}
	return hunks
}

func humanSize(n int64) string {
	const k = 1024
	if n < k {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(k), 0
	for n2 := n / k; n2 >= k; n2 /= k {
		div *= k
		exp++
	}
	units := []string{"K", "M", "G", "T"}
	return fmt.Sprintf("%.1f %s", float64(n)/float64(div), units[exp])
}

func timeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}
