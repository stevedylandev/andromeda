package main

import (
	"charm.land/bubbles/v2/key"
	sharedtui "github.com/stevedylandev/andromeda/pkg/tui"
)

type tuiKeyMap struct {
	sharedtui.KeyMap
	Upload    key.Binding
	Buckets   key.Binding
	Preview   key.Binding
	CopyKey   key.Binding
}

func defaultTUIKeys() tuiKeyMap {
	return tuiKeyMap{
		KeyMap: sharedtui.DefaultKeys(),
		Upload: key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "upload")),
		Buckets: key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "buckets")),
		Preview: key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle preview")),
		CopyKey: key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "copy key")),
	}
}

func (k tuiKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Open, k.Back, k.Copy, k.CopyLink, k.OpenBrowser, k.Upload, k.Delete, k.Quit, k.Help}
}

func (k tuiKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Open, k.Back, k.Buckets, k.Refresh},
		{k.Copy, k.CopyLink, k.CopyKey, k.OpenBrowser},
		{k.Upload, k.Delete, k.Preview, k.Help},
		{k.ScrollUp, k.ScrollDown, k.Quit},
	}
}
