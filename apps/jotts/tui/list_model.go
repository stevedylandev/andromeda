package tui

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	sharedtui "github.com/stevedylandev/andromeda/pkg/tui"
)

type noteItem struct {
	note Note
}

func (n noteItem) Title() string       { return n.note.Title }
func (n noteItem) Description() string { return "" }
func (n noteItem) FilterValue() string { return n.note.Title }

type listModel struct {
	inner list.Model
}

func newListModel(notes []Note) listModel {
	items := make([]list.Item, 0, len(notes))
	for _, n := range notes {
		items = append(items, noteItem{note: n})
	}

	l := list.New(items, sharedtui.ANSIListDelegate(), 0, 0)
	l.Title = "notes"
	l.Styles = sharedtui.ANSIListStyles()
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return listModel{inner: l}
}

func (l listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	var cmd tea.Cmd
	l.inner, cmd = l.inner.Update(msg)
	return l, cmd
}

func (l listModel) View() string { return l.inner.View() }

func (l *listModel) SetSize(w, h int) { l.inner.SetSize(w, h) }

func (l *listModel) SetNotes(notes []Note) tea.Cmd {
	items := make([]list.Item, 0, len(notes))
	for _, n := range notes {
		items = append(items, noteItem{note: n})
	}
	return l.inner.SetItems(items)
}

func (l listModel) Selected() (Note, bool) {
	it := l.inner.SelectedItem()
	if it == nil {
		return Note{}, false
	}
	n, ok := it.(noteItem)
	if !ok {
		return Note{}, false
	}
	return n.note, true
}

func (l listModel) IsFiltering() bool {
	return l.inner.SettingFilter()
}
