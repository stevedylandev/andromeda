package tui

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	sharedtui "github.com/stevedylandev/andromeda/crates-go/tui"
)

var listIDStyle = sharedtui.ListIDStyle

type snippetItem struct {
	snippet Snippet
}

func (s snippetItem) Title() string {
	label := s.snippet.Name
	if label == "" {
		label = s.snippet.ShortID
		return label
	}
	return label + listIDStyle.Render(" "+s.snippet.ShortID)
}
func (s snippetItem) Description() string { return "" }
func (s snippetItem) FilterValue() string {
	if s.snippet.Name != "" {
		return s.snippet.Name + " " + s.snippet.ShortID
	}
	return s.snippet.ShortID
}

type listModel struct {
	inner list.Model
}

func newListModel(snippets []Snippet) listModel {
	items := make([]list.Item, 0, len(snippets))
	for _, s := range snippets {
		items = append(items, snippetItem{snippet: s})
	}

	l := list.New(items, sharedtui.ANSIListDelegate(), 0, 0)
	l.Title = "snippets"
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

func (l *listModel) SetSnippets(snippets []Snippet) tea.Cmd {
	items := make([]list.Item, 0, len(snippets))
	for _, s := range snippets {
		items = append(items, snippetItem{snippet: s})
	}
	return l.inner.SetItems(items)
}

func (l listModel) Selected() (Snippet, bool) {
	it := l.inner.SelectedItem()
	if it == nil {
		return Snippet{}, false
	}
	s, ok := it.(snippetItem)
	if !ok {
		return Snippet{}, false
	}
	return s.snippet, true
}

func (l listModel) IsFiltering() bool {
	return l.inner.SettingFilter()
}
