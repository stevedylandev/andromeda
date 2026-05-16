package tui

type snippetsLoadedMsg struct {
	snippets []Snippet
	err      error
}

type snippetSavedMsg struct {
	snippet *Snippet
	err     error
}

type snippetDeletedMsg struct {
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
