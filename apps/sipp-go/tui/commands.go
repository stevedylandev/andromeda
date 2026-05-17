package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
)

func loadSnippetsCmd(b Backend) tea.Cmd {
	return func() tea.Msg {
		list, err := b.List()
		return snippetsLoadedMsg{snippets: list, err: err}
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
		return snippetSavedMsg{snippet: s, err: err}
	}
}

func deleteSnippetCmd(b Backend, shortID string) tea.Cmd {
	return func() tea.Msg {
		_, err := b.Delete(shortID)
		return snippetDeletedMsg{shortID: shortID, err: err}
	}
}

func copyToClipboardCmd(text, okStatus string) tea.Cmd {
	return func() tea.Msg {
		if err := clipboard.WriteAll(text); err != nil {
			return statusMsg{text: "clipboard: " + err.Error(), ok: false}
		}
		return statusMsg{text: okStatus, ok: true}
	}
}

func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if err := openURL(url); err != nil {
			return statusMsg{text: "open: " + err.Error(), ok: false}
		}
		return statusMsg{text: "opened " + url, ok: true}
	}
}

func shareLinkURL(remoteBase, shortID string) string {
	if remoteBase == "" {
		return ""
	}
	return strings.TrimRight(remoteBase, "/") + "/s/" + shortID
}
