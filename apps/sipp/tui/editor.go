package tui

import (
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	sharedtui "github.com/stevedylandev/andromeda/crates-go/tui"
)

func openExternalEditor(shortID, name, content string) tea.Cmd {
	base := name
	if base == "" {
		base = "snippet.txt"
	}
	pattern := "sipp-" + shortID + "-*-" + filepath.Base(base)
	return sharedtui.SpawnEditor(shortID, pattern, content)
}
