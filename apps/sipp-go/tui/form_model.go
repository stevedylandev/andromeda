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
	formFieldName formField = iota
	formFieldContent
)

type formModel struct {
	name    textinput.Model
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
	ti.Placeholder = "name.ext"
	ti.Prompt = ""
	ti.CharLimit = 200

	ta := textarea.New()
	ta.Placeholder = "Paste code..."
	ta.ShowLineNumbers = true
	ta.Prompt = ""

	return formModel{name: ti, content: ta, keys: defaultFormKeys()}
}

func (f *formModel) StartCreate() {
	f.shortID = ""
	f.isCreate = true
	f.name.SetValue("")
	f.content.SetValue("")
	f.field = formFieldName
	f.applyFocus()
}

func (f *formModel) StartEdit(s Snippet) {
	f.shortID = s.ShortID
	f.isCreate = false
	f.name.SetValue(s.Name)
	f.content.SetValue(s.Content)
	f.field = formFieldName
	f.applyFocus()
}

func (f *formModel) SetContent(s string) { f.content.SetValue(s) }

func (f *formModel) Blur() {
	f.name.Blur()
	f.content.Blur()
}

func (f *formModel) applyFocus() {
	switch f.field {
	case formFieldName:
		f.name.Focus()
		f.content.Blur()
	case formFieldContent:
		f.content.Focus()
		f.name.Blur()
	}
}

func (f *formModel) SetSize(w, h int) {
	f.name.SetWidth(max(w-4, 1))
	f.content.SetWidth(max(w-2, 1))
	f.content.SetHeight(max(h-5, 1))
}

func (f formModel) Update(msg tea.Msg) (formModel, tea.Cmd) {
	if km, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(km, f.keys.Cancel):
			f.Blur()
			return f, func() tea.Msg { return cancelFormMsg{} }
		case key.Matches(km, f.keys.Save):
			name := strings.TrimSpace(f.name.Value())
			if name == "" {
				return f, func() tea.Msg { return statusMsg{text: "name required", ok: false} }
			}
			content := f.content.Value()
			if strings.TrimSpace(content) == "" {
				return f, func() tea.Msg { return statusMsg{text: "content required", ok: false} }
			}
			return f, func() tea.Msg {
				return submitFormMsg{shortID: f.shortID, name: name, content: content}
			}
		case key.Matches(km, f.keys.SwitchField):
			if f.field == formFieldName {
				f.field = formFieldContent
			} else {
				f.field = formFieldName
			}
			f.applyFocus()
			return f, nil
		}
	}

	var cmd tea.Cmd
	switch f.field {
	case formFieldName:
		f.name, cmd = f.name.Update(msg)
	case formFieldContent:
		f.content, cmd = f.content.Update(msg)
	}
	return f, cmd
}

func (f formModel) ActiveField() formField { return f.field }
func (f formModel) IsCreate() bool         { return f.isCreate }
