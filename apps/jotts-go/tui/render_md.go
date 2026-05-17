package tui

import (
	"fmt"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
)

type mdRenderer struct {
	r     *glamour.TermRenderer
	width int
	cache map[string]string
}

func sp(s string) *string { return &s }
func bp(b bool) *bool     { return &b }
func up(u uint) *uint     { return &u }

// ansiStyle uses only base ANSI palette indexes (0-7 for normal, 8-15 bright)
// so colors follow the terminal theme instead of hardcoded hex/256 values.
func ansiStyle() ansi.StyleConfig {
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{BlockPrefix: "\n", BlockSuffix: "\n"},
			Margin:         up(2),
		},
		BlockQuote: ansi.StyleBlock{
			Indent:      up(1),
			IndentToken: sp("│ "),
		},
		Paragraph: ansi.StyleBlock{},
		List: ansi.StyleList{
			LevelIndent: 2,
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{BlockSuffix: "\n", Color: sp("4"), Bold: bp(true)},
		},
		H1: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "# ", Color: sp("4"), Bold: bp(true)}},
		H2: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "## ", Color: sp("4"), Bold: bp(true)}},
		H3: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "### ", Color: sp("6"), Bold: bp(true)}},
		H4: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "#### ", Color: sp("6")}},
		H5: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "##### ", Color: sp("6")}},
		H6: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "###### ", Color: sp("6")}},
		Strikethrough: ansi.StylePrimitive{CrossedOut: bp(true)},
		Emph:          ansi.StylePrimitive{Italic: bp(true)},
		Strong:        ansi.StylePrimitive{Bold: bp(true)},
		HorizontalRule: ansi.StylePrimitive{
			Color:  sp("8"),
			Format: "\n--------\n",
		},
		Item:        ansi.StylePrimitive{BlockPrefix: "• "},
		Enumeration: ansi.StylePrimitive{BlockPrefix: ". "},
		Task: ansi.StyleTask{
			Ticked:   "[✓] ",
			Unticked: "[ ] ",
		},
		Link:     ansi.StylePrimitive{Color: sp("6"), Underline: bp(true)},
		LinkText: ansi.StylePrimitive{Color: sp("2"), Bold: bp(true)},
		Image:    ansi.StylePrimitive{Color: sp("5"), Underline: bp(true)},
		ImageText: ansi.StylePrimitive{
			Color:  sp("8"),
			Format: "Image: {{.text}} →",
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: sp("1"), Prefix: "`", Suffix: "`"},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: sp("7")},
				Margin:         up(2),
			},
		},
		Table: ansi.StyleTable{
			CenterSeparator: sp("┼"),
			ColumnSeparator: sp("│"),
			RowSeparator:    sp("─"),
		},
		DefinitionDescription: ansi.StylePrimitive{BlockPrefix: "\n* "},
	}
}

func newRenderer(width int) *mdRenderer {
	if width < 20 {
		width = 80
	}
	style := ansiStyle()
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(style),
		glamour.WithWordWrap(width-2),
	)
	return &mdRenderer{r: r, width: width, cache: map[string]string{}}
}

func (m *mdRenderer) resize(width int) {
	if width == m.width || width < 20 {
		return
	}
	style := ansiStyle()
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(style),
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
