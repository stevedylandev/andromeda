package tui

import (
	"fmt"

	"github.com/charmbracelet/glamour"
)

type mdRenderer struct {
	r     *glamour.TermRenderer
	width int
	cache map[string]string
}

func newRenderer(width int) *mdRenderer {
	if width < 20 {
		width = 80
	}
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-2),
	)
	return &mdRenderer{r: r, width: width, cache: map[string]string{}}
}

func (m *mdRenderer) resize(width int) {
	if width == m.width || width < 20 {
		return
	}
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-2),
	)
	m.r = r
	m.width = width
	m.cache = map[string]string{}
}

func (m *mdRenderer) render(key, body string) string {
	if m.r == nil {
		return body
	}
	if v, ok := m.cache[key]; ok {
		return v
	}
	out, err := m.r.Render(body)
	if err != nil {
		out = fmt.Sprintf("render error: %v\n\n%s", err, body)
	}
	m.cache[key] = out
	return out
}

func (m *mdRenderer) invalidate(key string) {
	delete(m.cache, key)
}
