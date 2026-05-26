package main

import (
	sharedtui "github.com/stevedylandev/andromeda/pkg/tui"
)

type bucketsLoadedMsg struct {
	Buckets []BucketInfo
	Err     error
}

type listingLoadedMsg struct {
	Bucket  string
	Prefix  string
	Folders []string
	Files   []ObjectInfo
	Err     error
}

type previewLoadedMsg struct {
	Seq     int
	Bucket  string
	Key     string
	Content string // pre-rendered ANSI (image) or text
	Err     error
}

type previewDebounceMsg struct {
	Seq    int
	Bucket string
	Key    string
	W, H   int
}

type deletedMsg struct {
	Key string
	Err error
}

type uploadedMsg struct {
	Key string
	URL string
	Err error
}

type (
	statusMsg      = sharedtui.StatusMsg
	clearStatusMsg = sharedtui.ClearStatusMsg
)
