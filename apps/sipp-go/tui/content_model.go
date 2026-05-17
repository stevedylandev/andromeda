package tui

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type contentModel struct {
	vp          viewport.Model
	highlighter *highlighter
	wrap        bool

	shortID string
	name    string
	body    string
}

func newContentModel() contentModel {
	return contentModel{vp: viewport.New(), highlighter: newHighlighter(), wrap: true}
}

func (c contentModel) Update(msg tea.Msg) (contentModel, tea.Cmd) {
	var cmd tea.Cmd
	c.vp, cmd = c.vp.Update(msg)
	return c, cmd
}

func (c *contentModel) SetSize(w, h int) {
	c.vp.SetWidth(w)
	c.vp.SetHeight(h)
	c.refresh()
}

func (c *contentModel) SetSnippet(s *Snippet) {
	if s == nil {
		c.shortID, c.name, c.body = "", "", ""
		c.vp.SetContent("")
		c.vp.GotoTop()
		return
	}
	c.shortID = s.ShortID
	c.name = s.Name
	c.body = s.Content
	c.vp.GotoTop()
	c.refresh()
}

func (c *contentModel) ToggleWrap() {
	c.wrap = !c.wrap
	c.vp.GotoTop()
	c.refresh()
}

func (c *contentModel) Wrap() bool { return c.wrap }

func (c *contentModel) Invalidate(shortID string) {
	if c.highlighter != nil {
		c.highlighter.invalidate(shortID)
	}
}

func (c *contentModel) refresh() {
	if c.body == "" {
		c.vp.SetContent("")
		return
	}
	body := c.body
	if c.highlighter != nil {
		body = c.highlighter.render(c.shortID, c.name, c.body)
	}
	if c.wrap && c.vp.Width() > 0 {
		body = lipgloss.NewStyle().Width(c.vp.Width()).Render(body)
	}
	c.vp.SetContent(body)
}

func (c contentModel) Header() string {
	if c.name != "" {
		return c.name
	}
	return c.shortID
}

func (c contentModel) View() string { return c.vp.View() }

func (c contentModel) ScrollUp(n int) contentModel {
	c.vp.ScrollUp(n)
	return c
}

func (c contentModel) ScrollDown(n int) contentModel {
	c.vp.ScrollDown(n)
	return c
}
