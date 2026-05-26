package main

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	sharedtui "github.com/stevedylandev/andromeda/pkg/tui"
)

var listIDStyle = sharedtui.ListIDStyle

type bucketItem struct{ b BucketInfo }

func (i bucketItem) Title() string {
	if i.b.CreationDate != "" {
		return i.b.Name + listIDStyle.Render(" "+i.b.CreationDate)
	}
	return i.b.Name
}
func (i bucketItem) Description() string { return "" }
func (i bucketItem) FilterValue() string { return i.b.Name }

type folderItemTUI struct {
	prefix string // full S3 prefix (ends in /)
	name   string
}

func (i folderItemTUI) Title() string       { return "📁 " + i.name + "/" }
func (i folderItemTUI) Description() string { return "" }
func (i folderItemTUI) FilterValue() string { return i.name }

type fileItemTUI struct {
	obj ObjectInfo
}

func (i fileItemTUI) Title() string {
	return i.obj.Name + listIDStyle.Render(fmt.Sprintf("  %s  %s", humanSize(i.obj.Size), i.obj.LastModified))
}
func (i fileItemTUI) Description() string { return "" }
func (i fileItemTUI) FilterValue() string { return i.obj.Name }

func newList(title string, items []list.Item) list.Model {
	l := list.New(items, sharedtui.ANSIListDelegate(), 0, 0)
	l.Title = title
	l.Styles = sharedtui.ANSIListStyles()
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()
	return l
}
