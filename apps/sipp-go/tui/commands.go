package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func loadSnippetsCmd(b Backend) tea.Cmd {
	return func() tea.Msg {
		list, err := b.List()
		return snippetsLoadedMsg{Snippets: list, Err: err}
	}
}

func saveSnippetCmd(b Backend, shortID, name, content string) tea.Cmd {
	return func() tea.Msg {
		var (
			s   *Snippet
			err error
		)
		if shortID == "" {
			s, err = b.Create(name, content)
		} else {
			s, err = b.Update(shortID, name, content)
		}
		return snippetSavedMsg{Snippet: s, Err: err}
	}
}

func deleteSnippetCmd(b Backend, shortID string) tea.Cmd {
	return func() tea.Msg {
		_, err := b.Delete(shortID)
		return snippetDeletedMsg{ShortID: shortID, Err: err}
	}
}

func shareLinkURL(remoteBase, shortID string) string {
	if remoteBase == "" {
		return ""
	}
	return strings.TrimRight(remoteBase, "/") + "/s/" + shortID
}
