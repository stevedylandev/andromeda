package main

import (
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/stevedylandev/andromeda/apps/blobs/preview"
)

type tuiState uint8

const (
	stateBuckets tuiState = iota
	stateBrowse
)

type uploadPromptState uint8

const (
	uploadPromptOff uploadPromptState = iota
	uploadPromptActive
)

type tuiModel struct {
	s3   *S3Client
	cfg  ClientConfig
	opts tuiOptions

	state tuiState

	bucketsList list.Model
	browseList  list.Model

	preview      viewport.Model
	previewProto preview.Protocol
	showPreview  bool
	previewSeq   int

	currentBucket string
	currentPrefix string

	width, height int
	ready         bool
	loading       bool

	// modals
	confirmDelete bool
	uploadPrompt  uploadPromptState
	uploadInput   textinput.Model

	// help overlay
	showHelp bool
	help     help.Model
	keys     tuiKeyMap

	// status
	status      string
	statusOK    bool
	statusUntil time.Time
}

func newTUIModel(s3 *S3Client, cfg ClientConfig, opts tuiOptions) tuiModel {
	bl := newList("buckets", nil)
	brl := newList("/", nil)
	ti := textinput.New()
	ti.Placeholder = "/path/to/file"
	ti.Prompt = "upload: "

	m := tuiModel{
		s3:           s3,
		cfg:          cfg,
		opts:         opts,
		state:        stateBuckets,
		bucketsList:  bl,
		browseList:   brl,
		preview:      viewport.New(),
		previewProto: preview.Detect(),
		showPreview:  true,
		help:         help.New(),
		keys:         defaultTUIKeys(),
		uploadInput:  ti,
		loading:      true,
	}
	if cfg.DefaultBucket != "" {
		m.state = stateBrowse
		m.currentBucket = cfg.DefaultBucket
		m.currentPrefix = opts.Prefix
	}
	return m
}

func (m tuiModel) Init() tea.Cmd {
	cmds := []tea.Cmd{tea.RequestWindowSize}
	if m.state == stateBrowse {
		cmds = append(cmds, loadListingCmd(m.s3, m.currentBucket, m.currentPrefix))
	} else {
		cmds = append(cmds, loadBucketsCmd(m.s3))
	}
	return tea.Batch(cmds...)
}

func (m *tuiModel) setStatus(text string, ok bool) tea.Cmd {
	m.status = text
	m.statusOK = ok
	m.statusUntil = time.Now().Add(2 * time.Second)
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearStatusMsg{} })
}

func (m *tuiModel) selectedFile() (ObjectInfo, bool) {
	if m.state != stateBrowse {
		return ObjectInfo{}, false
	}
	it := m.browseList.SelectedItem()
	if it == nil {
		return ObjectInfo{}, false
	}
	f, ok := it.(fileItemTUI)
	if !ok {
		return ObjectInfo{}, false
	}
	return f.obj, true
}

func (m *tuiModel) selectedFolderPrefix() (string, bool) {
	if m.state != stateBrowse {
		return "", false
	}
	it := m.browseList.SelectedItem()
	if it == nil {
		return "", false
	}
	f, ok := it.(folderItemTUI)
	if !ok {
		return "", false
	}
	return f.prefix, true
}

func (m *tuiModel) selectedBucket() (string, bool) {
	if m.state != stateBuckets {
		return "", false
	}
	it := m.bucketsList.SelectedItem()
	if it == nil {
		return "", false
	}
	b, ok := it.(bucketItem)
	if !ok {
		return "", false
	}
	return b.b.Name, true
}

func (m *tuiModel) applyLayout() {
	if !m.ready {
		return
	}
	bodyH := m.height - 2
	if bodyH < 5 {
		bodyH = 5
	}

	switch m.state {
	case stateBuckets:
		fw, fh := paneFrameW(), paneFrameH()
		m.bucketsList.SetSize(max(m.width-fw, 1), max(bodyH-fh, 1))
	case stateBrowse:
		listW := m.width
		if m.showPreview {
			listW = m.width / 2
		}
		fw, fh := paneFrameW(), paneFrameH()
		m.browseList.SetSize(max(listW-fw, 1), max(bodyH-fh, 1))
		if m.showPreview {
			pw := m.width - listW
			m.preview.SetWidth(max(pw-fw, 1))
			m.preview.SetHeight(max(bodyH-fh-1, 1))
		}
	}
}

func setItemsFromListing(l *list.Model, prefix string, folders []string, files []ObjectInfo) tea.Cmd {
	items := make([]list.Item, 0, len(folders)+len(files))
	for _, fp := range folders {
		name := fp
		if len(prefix) <= len(fp) {
			name = fp[len(prefix):]
		}
		name = trimTrailingSlash(name)
		items = append(items, folderItemTUI{prefix: fp, name: name})
	}
	for _, f := range files {
		items = append(items, fileItemTUI{obj: f})
	}
	return l.SetItems(items)
}

func trimTrailingSlash(s string) string {
	if len(s) > 0 && s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}
	return s
}
