package tui

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

const highlightCacheMax = 64

type highlighter struct {
	cache map[string]string
	order []string
}

func newHighlighter() *highlighter {
	return &highlighter{cache: map[string]string{}}
}

func (h *highlighter) render(shortID, name, content string) string {
	if v, ok := h.cache[shortID]; ok {
		return v
	}
	out := highlightCode(name, content)
	h.cache[shortID] = out
	h.order = append(h.order, shortID)
	if len(h.order) > highlightCacheMax {
		drop := h.order[0]
		h.order = h.order[1:]
		delete(h.cache, drop)
	}
	return out
}

func (h *highlighter) invalidate(shortID string) {
	if _, ok := h.cache[shortID]; !ok {
		return
	}
	delete(h.cache, shortID)
	for i, k := range h.order {
		if k == shortID {
			h.order = append(h.order[:i], h.order[i+1:]...)
			break
		}
	}
}

func highlightCode(name, content string) string {
	var lexer chroma.Lexer
	if name != "" {
		lexer = lexers.Match(name)
	}
	if lexer == nil {
		lexer = lexers.Analyse(content)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	style := styles.Get("vim")
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal16")
	if formatter == nil {
		formatter = formatters.Fallback
	}
	iter, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content
	}
	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iter); err != nil {
		return content
	}
	return strings.TrimRight(buf.String(), "\n")
}
