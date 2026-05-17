package tui

type notesLoadedMsg struct {
	notes []Note
	err   error
}

type noteSavedMsg struct {
	note *Note
	err  error
}

type noteDeletedMsg struct {
	shortID string
	err     error
}

type editorFinishedMsg struct {
	shortID string
	content string
	err     error
}

type statusMsg struct {
	text string
	ok   bool
}

type clearStatusMsg struct{}

type submitFormMsg struct {
	shortID string
	title   string
	content string
}

type cancelFormMsg struct{}
