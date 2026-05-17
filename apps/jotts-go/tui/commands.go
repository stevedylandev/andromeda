package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
)

func loadNotesCmd(b Backend) tea.Cmd {
	return func() tea.Msg {
		notes, err := b.List()
		return notesLoadedMsg{notes: notes, err: err}
	}
}

func saveNoteCmd(b Backend, shortID, title, content string) tea.Cmd {
	return func() tea.Msg {
		var (
			note *Note
			err  error
		)
		if shortID == "" {
			note, err = b.Create(title, content)
		} else {
			note, err = b.Update(shortID, title, content)
		}
		return noteSavedMsg{note: note, err: err}
	}
}

func deleteNoteCmd(b Backend, shortID string) tea.Cmd {
	return func() tea.Msg {
		_, err := b.Delete(shortID)
		return noteDeletedMsg{shortID: shortID, err: err}
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

func noteLinkURL(remoteBase, shortID string) string {
	if remoteBase == "" {
		return ""
	}
	return strings.TrimRight(remoteBase, "/") + "/notes/" + shortID
}
