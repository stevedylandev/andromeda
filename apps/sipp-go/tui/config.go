package tui

import sharedtui "github.com/stevedylandev/andromeda/crates-go/tui"

const appName = "sipp"

type Config = sharedtui.Config

func ConfigPath() (string, error) { return sharedtui.ConfigPath(appName) }
func LoadConfig() (Config, error) { return sharedtui.LoadConfig(appName) }
func SaveConfig(cfg Config) error { return sharedtui.SaveConfig(appName, cfg) }
