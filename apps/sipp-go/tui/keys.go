package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up          key.Binding
	Down        key.Binding
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
	Search      key.Binding
	Refresh     key.Binding
	Help        key.Binding
	Save        key.Binding
	SwitchField key.Binding
	Cancel      key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Up:          key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Open:        key.NewBinding(key.WithKeys("enter", "l"), key.WithHelp("⏎/l", "open")),
		Back:        key.NewBinding(key.WithKeys("h", "esc"), key.WithHelp("h/esc", "back")),
		Quit:        key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		Create:      key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "create")),
		Edit:        key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		ExtEdit:     key.NewBinding(key.WithKeys("E"), key.WithHelp("E", "$EDITOR")),
		Delete:      key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Copy:        key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy text")),
		CopyLink:    key.NewBinding(key.WithKeys("Y"), key.WithHelp("Y", "copy link")),
		OpenBrowser: key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "browser")),
		Search:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Refresh:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Save:        key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("⌃s", "save")),
		SwitchField: key.NewBinding(key.WithKeys("tab"), key.WithHelp("⇥", "switch field")),
		Cancel:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Open, k.Create, k.Edit, k.Delete, k.Search, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Open, k.Back},
		{k.Create, k.Edit, k.ExtEdit, k.Delete},
		{k.Copy, k.CopyLink, k.OpenBrowser, k.Search},
		{k.Refresh, k.Help, k.Save, k.SwitchField},
		{k.Cancel, k.Quit},
	}
}
