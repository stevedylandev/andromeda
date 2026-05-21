package tui

import sharedtui "github.com/stevedylandev/andromeda/pkg/tui"

type keyMap = sharedtui.KeyMap

func defaultKeys() keyMap { return sharedtui.DefaultKeys() }
