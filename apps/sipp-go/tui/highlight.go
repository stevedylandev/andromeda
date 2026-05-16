package tui

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

type highlighter struct {
	cache map[string]string
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
	return out
}

func (h *highlighter) invalidate(shortID string) {
	delete(h.cache, shortID)
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
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal256")
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
