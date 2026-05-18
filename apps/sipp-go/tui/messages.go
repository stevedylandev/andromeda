package tui

import sharedtui "github.com/stevedylandev/andromeda/crates-go/tui"

type snippetsLoadedMsg struct {
	Snippets []Snippet
	Err      error
}

type snippetSavedMsg struct {
	Snippet *Snippet
	Err     error
}

type snippetDeletedMsg struct {
	ShortID string
	Err     error
}

type submitFormMsg struct {
	ShortID string
	Name    string
	Content string
}

type cancelFormMsg struct{}

type (
	statusMsg         = sharedtui.StatusMsg
	clearStatusMsg    = sharedtui.ClearStatusMsg
	editorFinishedMsg = sharedtui.EditorFinishedMsg
)
