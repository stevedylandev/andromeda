package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type formField uint8

const (
	formFieldTitle formField = iota
	formFieldContent
)

type formModel struct {
	title   textinput.Model
	content textarea.Model

	field    formField
	shortID  string
	isCreate bool

	keys formKeys
}

type formKeys struct {
	Save        key.Binding
	Cancel      key.Binding
	SwitchField key.Binding
}

func defaultFormKeys() formKeys {
	return formKeys{
		Save:        key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("⌃s", "save")),
		Cancel:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		SwitchField: key.NewBinding(key.WithKeys("tab"), key.WithHelp("⇥", "switch field")),
	}
}

func newFormModel() formModel {
	ti := textinput.New()
	ti.Placeholder = "Title"
	ti.Prompt = ""
	ti.CharLimit = 200

	ta := textarea.New()
	ta.Placeholder = "Write markdown..."
	ta.ShowLineNumbers = false
	ta.Prompt = ""

	return formModel{title: ti, content: ta, keys: defaultFormKeys()}
}

func (f *formModel) StartCreate() {
	f.shortID = ""
	f.isCreate = true
	f.title.SetValue("")
	f.content.SetValue("")
	f.field = formFieldTitle
	f.applyFocus()
}

func (f *formModel) StartEdit(n Note) {
	f.shortID = n.ShortID
	f.isCreate = false
	f.title.SetValue(n.Title)
	f.content.SetValue(n.Content)
	f.field = formFieldTitle
	f.applyFocus()
}

func (f *formModel) SetContent(s string) { f.content.SetValue(s) }

func (f *formModel) Blur() {
	f.title.Blur()
	f.content.Blur()
}

func (f *formModel) applyFocus() {
	switch f.field {
	case formFieldTitle:
		f.title.Focus()
		f.content.Blur()
	case formFieldContent:
		f.content.Focus()
		f.title.Blur()
	}
}

func (f *formModel) SetSize(w, h int) {
	f.title.SetWidth(max(w-4, 1))
	f.content.SetWidth(max(w-2, 1))
	f.content.SetHeight(max(h-6, 1))
}

func (f formModel) Update(msg tea.Msg) (formModel, tea.Cmd) {
	if km, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(km, f.keys.Cancel):
			f.Blur()
			return f, func() tea.Msg { return cancelFormMsg{} }
		case key.Matches(km, f.keys.Save):
			title := strings.TrimSpace(f.title.Value())
			if title == "" {
				return f, func() tea.Msg { return statusMsg{Text: "title required"} }
			}
			return f, func() tea.Msg {
				return submitFormMsg{ShortID: f.shortID, Title: title, Content: f.content.Value()}
			}
		case key.Matches(km, f.keys.SwitchField):
			if f.field == formFieldTitle {
				f.field = formFieldContent
			} else {
				f.field = formFieldTitle
			}
			f.applyFocus()
			return f, nil
		}
	}

	var cmd tea.Cmd
	switch f.field {
	case formFieldTitle:
		f.title, cmd = f.title.Update(msg)
	case formFieldContent:
		f.content, cmd = f.content.Update(msg)
	}
	return f, cmd
}

func (f formModel) ActiveField() formField { return f.field }
func (f formModel) IsCreate() bool         { return f.isCreate }
