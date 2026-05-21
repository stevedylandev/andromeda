package main

import (
	"strings"
	"time"
)

type parsedAttributes struct {
	Title           string
	Slug            string
	Alias           string
	PublishedDate   string
	MetaDescription string
	MetaImage       string
	Lang            string
	Tags            string
	Status          string
}

func parseAttributes(text string) parsedAttributes {
	var a parsedAttributes
	for _, line := range strings.Split(text, "\n") {
		i := strings.Index(line, ":")
		if i < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:i]))
		value := strings.TrimSpace(line[i+1:])
		switch key {
		case "title":
			a.Title = value
		case "slug":
			a.Slug = value
		case "alias":
			a.Alias = value
		case "published_date":
			a.PublishedDate = value
		case "description", "meta_description":
			a.MetaDescription = value
		case "meta_image":
			a.MetaImage = value
		case "lang":
			a.Lang = value
		case "tags":
			a.Tags = value
		case "status":
			a.Status = value
		}
	}
	return a
}

type parsedPageAttributes struct {
	Title       string
	Slug        string
	IsPublished bool
}

func parsePageAttributes(text string) parsedPageAttributes {
	var a parsedPageAttributes
	for _, line := range strings.Split(text, "\n") {
		i := strings.Index(line, ":")
		if i < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:i]))
		value := strings.TrimSpace(line[i+1:])
		switch key {
		case "title":
			a.Title = value
		case "slug":
			a.Slug = value
		case "published":
			a.IsPublished = value == "true"
		}
	}
	return a
}

func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	parts := strings.Split(b.String(), "-")
	out := parts[:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "-")
}

func optStr(s string) *string {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	return &t
}

func deriveSlug(title, slug string) string {
	if slug != "" {
		return slug
	}
	if from := slugify(title); from != "" {
		return from
	}
	id, _ := generateID()
	return id
}

func generateID() (string, error) {
	// Imported from auth crate at call site; this is a fallback path.
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, 10)
	for i := range buf {
		buf[i] = alphabet[time.Now().UnixNano()%int64(len(alphabet))]
		time.Sleep(time.Nanosecond)
	}
	return string(buf), nil
}

var reservedPageSlugs = map[string]bool{
	"posts": true, "admin": true, "feed.xml": true,
	"custom-styles.css": true, "static": true, "files": true,
}

func isReservedPageSlug(slug string) bool {
	return reservedPageSlugs[slug]
}

func parseNavLinks(input string) []NavLink {
	var out []NavLink
	rest := input
	for {
		open := strings.Index(rest, "[")
		if open < 0 {
			break
		}
		close := strings.Index(rest[open:], "]")
		if close < 0 {
			break
		}
		close += open
		label := rest[open+1 : close]
		if close+1 >= len(rest) || rest[close+1] != '(' {
			rest = rest[close+1:]
			continue
		}
		urlEnd := strings.Index(rest[close+2:], ")")
		if urlEnd < 0 {
			break
		}
		urlEnd += close + 2
		url := rest[close+2 : urlEnd]
		if label != "" && url != "" {
			out = append(out, NavLink{Label: label, URL: url})
		}
		rest = rest[urlEnd+1:]
	}
	return out
}

var pubDateLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// parsePubDate accepts RFC3339, naive datetime, or date-only input and returns
// the value normalized to RFC3339 UTC. Returns ok=false if no layout matches.
func parsePubDate(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	for _, l := range pubDateLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC().Format(time.RFC3339), true
		}
	}
	return "", false
}

func toRFC2822(ts string) string {
	for _, l := range pubDateLayouts {
		if t, err := time.Parse(l, ts); err == nil {
			return t.UTC().Format(time.RFC1123Z)
		}
	}
	return ts
}

func xmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(s)
}

func mimeFromPath(path string) string {
	i := strings.LastIndex(path, ".")
	if i < 0 {
		return "application/octet-stream"
	}
	switch strings.ToLower(path[i+1:]) {
	case "css":
		return "text/css"
	case "js":
		return "application/javascript"
	case "html":
		return "text/html"
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "ico":
		return "image/x-icon"
	case "svg":
		return "image/svg+xml"
	case "woff", "woff2":
		return "font/woff2"
	case "ttf":
		return "font/ttf"
	case "otf":
		return "font/otf"
	case "json", "webmanifest":
		return "application/json"
	case "pdf":
		return "application/pdf"
	case "mp4":
		return "video/mp4"
	case "webm":
		return "video/webm"
	}
	return "application/octet-stream"
}

func postToMarkdown(p *Post) string {
	var b strings.Builder
	b.WriteString("---")
	if p.Title != nil {
		b.WriteString("\ntitle: " + *p.Title)
	}
	b.WriteString("\nslug: " + p.Slug)
	b.WriteString("\nstatus: " + p.Status)
	if p.PublishedDate != nil {
		b.WriteString("\npublished_date: " + *p.PublishedDate)
	}
	if p.Tags != nil {
		b.WriteString("\ntags: " + *p.Tags)
	}
	b.WriteString("\nlang: " + p.Lang)
	if p.Alias != nil {
		b.WriteString("\nalias: " + *p.Alias)
	}
	if p.MetaImage != nil {
		b.WriteString("\nmeta_image: " + *p.MetaImage)
	}
	if p.MetaDescription != nil {
		b.WriteString("\ndescription: " + *p.MetaDescription)
	}
	b.WriteString("\n---\n\n")
	b.WriteString(p.Content)
	return b.String()
}

func splitFrontmatter(content string) (string, string) {
	trimmed := strings.TrimPrefix(content, "\ufeff")
	var afterOpen string
	if strings.HasPrefix(trimmed, "---\n") {
		afterOpen = trimmed[4:]
	} else if strings.HasPrefix(trimmed, "---\r\n") {
		afterOpen = trimmed[5:]
	} else {
		return "", content
	}
	for _, sep := range []string{"\r\n---\r\n", "\r\n---\n", "\n---\r\n", "\n---\n"} {
		if i := strings.Index(afterOpen, sep); i >= 0 {
			body := afterOpen[i+len(sep):]
			body = strings.TrimLeft(body, "\r\n")
			return afterOpen[:i], body
		}
	}
	if strings.HasSuffix(afterOpen, "\n---") {
		return strings.TrimSuffix(afterOpen, "\n---"), ""
	}
	if strings.HasSuffix(afterOpen, "\r\n---") {
		return strings.TrimSuffix(afterOpen, "\r\n---"), ""
	}
	return "", content
}

func titleFromFilename(name string) string {
	stem := name
	if i := strings.LastIndex(name, "."); i > 0 {
		stem = name[:i]
	}
	cleaned := strings.Map(func(r rune) rune {
		if r == '-' || r == '_' {
			return ' '
		}
		return r
	}, stem)
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return ""
	}
	return strings.ToUpper(cleaned[:1]) + cleaned[1:]
}
