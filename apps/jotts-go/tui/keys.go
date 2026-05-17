package tui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Open        key.Binding
	Back        key.Binding
	Quit        key.Binding
	Create      key.Binding
	Edit        key.Binding
	ExtEdit     key.Binding
	Delete      key.Binding
	Copy        key.Binding
	CopyLink    key.Binding
	OpenBrowser key.Binding
	Refresh     key.Binding
	Help        key.Binding
	ToggleWrap  key.Binding
	ScrollUp    key.Binding
	ScrollDown  key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Open:        key.NewBinding(key.WithKeys("enter", "l"), key.WithHelp("⏎/l", "open")),
		Back:        key.NewBinding(key.WithKeys("h", "esc"), key.WithHelp("h/esc", "back")),
		Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Create:      key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "create")),
		Edit:        key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		ExtEdit:     key.NewBinding(key.WithKeys("E"), key.WithHelp("E", "$EDITOR")),
		Delete:      key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Copy:        key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy text")),
		CopyLink:    key.NewBinding(key.WithKeys("Y"), key.WithHelp("Y", "copy link")),
		OpenBrowser: key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "browser")),
		Refresh:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		ToggleWrap:  key.NewBinding(key.WithKeys("ctrl+w"), key.WithHelp("⌃w", "wrap")),
		ScrollUp:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		ScrollDown:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Open, k.Create, k.Edit, k.Delete, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Open, k.Back, k.Create, k.Edit},
		{k.ExtEdit, k.Delete, k.Copy, k.CopyLink},
		{k.OpenBrowser, k.Refresh, k.ToggleWrap, k.Help},
		{k.ScrollUp, k.ScrollDown, k.Quit},
	}
}
