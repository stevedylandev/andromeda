package tui

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func ansiListDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.SetSpacing(0)
	d.Styles.NormalTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Padding(0, 0, 0, 2)
	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")).
		Bold(true).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("3")).
		Padding(0, 0, 0, 1)
	d.Styles.DimmedTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Padding(0, 0, 0, 2)
	d.Styles.FilterMatch = lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color("3"))
	return d
}

func ansiListStyles() list.Styles {
	s := list.DefaultStyles(true)
	s.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")).Padding(0, 1)
	s.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 0)
	s.NoItems = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Padding(0, 0, 0, 2)
	s.DefaultFilterCharacterMatch = lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color("3"))
	return s
}

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

	l := list.New(items, ansiListDelegate(), 0, 0)
	l.Title = "notes"
	l.Styles = ansiListStyles()
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
