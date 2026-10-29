package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func loadNotesCmd(b Backend) tea.Cmd {
	return func() tea.Msg {
		notes, err := b.List()
		return notesLoadedMsg{Notes: notes, Err: err}
	}
}

func saveNoteCmd(b Backend, shortID, title, content string, public *bool) tea.Cmd {
	return func() tea.Msg {
		var (
			note *Note
			err  error
		)
		if shortID == "" {
			note, err = b.Create(title, content)
		} else {
			note, err = b.Update(shortID, title, content, public)
		}
		return noteSavedMsg{Note: note, Err: err}
	}
}

func deleteNoteCmd(b Backend, shortID string) tea.Cmd {
	return func() tea.Msg {
		_, err := b.Delete(shortID)
		return noteDeletedMsg{ShortID: shortID, Err: err}
	}
}

func togglePublicCmd(b Backend, n Note) tea.Cmd {
	target := !n.Public
	status := "private"
	if target {
		status = "public"
	}
	return func() tea.Msg {
		note, err := b.Update(n.ShortID, n.Title, n.Content, &target)
		return noteSavedMsg{Note: note, Err: err, Status: status}
	}
}

func noteLinkURL(remoteBase, shortID string) string {
	if remoteBase == "" {
		return ""
	}
	return strings.TrimRight(remoteBase, "/") + "/notes/" + shortID
}
