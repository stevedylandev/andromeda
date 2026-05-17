package tui

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

type contentModel struct {
	vp       viewport.Model
	renderer *mdRenderer
	wrap     bool

	shortID string
	title   string
	body    string
}

func newContentModel() contentModel {
	return contentModel{vp: viewport.New(), wrap: true}
}

func (c contentModel) Update(msg tea.Msg) (contentModel, tea.Cmd) {
	var cmd tea.Cmd
	c.vp, cmd = c.vp.Update(msg)
	return c, cmd
}

func (c *contentModel) SetSize(w, h int) {
	c.vp.SetWidth(w)
	c.vp.SetHeight(h)
	if c.renderer == nil {
		c.renderer = newRenderer(w)
	} else {
		c.renderer.resize(w)
	}
	c.refresh()
}

func (c *contentModel) SetNote(n *Note) {
	if n == nil {
		c.shortID, c.title, c.body = "", "", ""
		c.vp.SetContent("")
		c.vp.GotoTop()
		return
	}
	c.shortID = n.ShortID
	c.title = n.Title
	c.body = n.Content
	c.vp.GotoTop()
	c.refresh()
}

func (c *contentModel) ToggleWrap() {
	c.wrap = !c.wrap
	c.refresh()
}

func (c *contentModel) Wrap() bool { return c.wrap }

func (c *contentModel) Invalidate(shortID string) {
	if c.renderer != nil {
		c.renderer.invalidate(shortID)
	}
}

func (c *contentModel) refresh() {
	if c.body == "" {
		c.vp.SetContent("")
		return
	}
	if !c.wrap || c.renderer == nil {
		c.vp.SetContent(c.body)
		return
	}
	c.vp.SetContent(c.renderer.render(c.shortID, c.body))
}

func (c contentModel) Title() string { return c.title }
func (c contentModel) View() string  { return c.vp.View() }

func (c contentModel) ScrollUp(n int) contentModel {
	c.vp.ScrollUp(n)
	return c
}

func (c contentModel) ScrollDown(n int) contentModel {
	c.vp.ScrollDown(n)
	return c
}
