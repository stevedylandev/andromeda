package tui

import sharedtui "github.com/stevedylandev/andromeda/pkg/tui"

type notesLoadedMsg struct {
	Notes []Note
	Err   error
}

type noteSavedMsg struct {
	Note   *Note
	Err    error
	Status string // optional status text; defaults to "saved"
}

type noteDeletedMsg struct {
	ShortID string
	Err     error
}

type submitFormMsg struct {
	ShortID string
	Title   string
	Content string
}

type cancelFormMsg struct{}

type (
	statusMsg         = sharedtui.StatusMsg
	clearStatusMsg    = sharedtui.ClearStatusMsg
	editorFinishedMsg = sharedtui.EditorFinishedMsg
)
