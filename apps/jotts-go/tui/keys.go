package tui

import sharedtui "github.com/stevedylandev/andromeda/crates-go/tui"

type keyMap = sharedtui.KeyMap

func defaultKeys() keyMap { return sharedtui.DefaultKeys() }
