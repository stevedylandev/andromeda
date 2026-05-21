package tui

import (
	tea "charm.land/bubbletea/v2"
	sharedtui "github.com/stevedylandev/andromeda/pkg/tui"
)

func openExternalEditor(shortID, content string) tea.Cmd {
	return sharedtui.SpawnEditor(shortID, "jotts-*.md", content)
}
