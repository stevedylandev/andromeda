package tui

import "charm.land/bubbles/v2/key"

import sharedtui "github.com/stevedylandev/andromeda/pkg/tui"

type keyMap = sharedtui.KeyMap

func defaultKeys() keyMap { return sharedtui.DefaultKeys() }

// togglePublic is jotts-specific: publish/unpublish the selected note.
// It lives outside the shared key map because sipp and blobs have no
// concept of public notes.
var togglePublicKey = key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "public"))

// fullHelp is the shared help grid with jotts's public toggle appended
// to the viewing actions column.
func fullHelp(k keyMap) [][]key.Binding {
	cols := k.FullHelp()
	if len(cols) > 2 {
		cols[2] = append(cols[2], togglePublicKey)
		return cols
	}
	return append(cols, []key.Binding{togglePublicKey})
}
