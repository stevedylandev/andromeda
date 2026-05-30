package main

import (
	"context"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	sharedtui "github.com/stevedylandev/andromeda/pkg/tui"
)

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.applyLayout()
		return m, nil

	case bucketsLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			return m, m.setStatus("buckets: "+msg.Err.Error(), false)
		}
		items := make([]list.Item, 0, len(msg.Buckets))
		for _, b := range msg.Buckets {
			items = append(items, bucketItem{b: b})
		}
		cmd := m.bucketsList.SetItems(items)
		return m, cmd

	case listingLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			return m, m.setStatus("list: "+msg.Err.Error(), false)
		}
		m.currentBucket = msg.Bucket
		m.currentPrefix = msg.Prefix
		m.browseList.Title = msg.Bucket + ":/" + msg.Prefix
		cmd := setItemsFromListing(&m.browseList, msg.Prefix, msg.Folders, msg.Files)
		return m, tea.Batch(cmd, m.maybePreviewCmd())

	case previewDebounceMsg:
		if msg.Seq != m.previewSeq || msg.Bucket != m.currentBucket {
			return m, nil
		}
		return m, loadPreviewCmd(m.s3, m.previewProto, msg.Seq, msg.Bucket, msg.Key, msg.W, msg.H)

	case previewLoadedMsg:
		if msg.Seq != m.previewSeq || msg.Bucket != m.currentBucket {
			return m, nil
		}
		if msg.Err != nil {
			m.preview.SetContent("preview error: " + msg.Err.Error())
			return m, nil
		}
		m.preview.SetContent(msg.Content)
		m.preview.GotoTop()
		return m, nil

	case deletedMsg:
		if msg.Err != nil {
			return m, m.setStatus("delete: "+msg.Err.Error(), false)
		}
		return m, tea.Batch(
			loadListingCmd(m.s3, m.currentBucket, m.currentPrefix),
			m.setStatus("deleted "+msg.Key, true),
		)

	case uploadedMsg:
		if msg.Err != nil {
			return m, m.setStatus("upload: "+msg.Err.Error(), false)
		}
		cmds := []tea.Cmd{
			loadListingCmd(m.s3, m.currentBucket, m.currentPrefix),
			m.setStatus("uploaded "+msg.Key, true),
		}
		if msg.URL != "" {
			cmds = append(cmds, sharedtui.CopyToClipboardCmd(msg.URL, "copied url"))
		}
		return m, tea.Batch(cmds...)

	case statusMsg:
		return m, m.setStatus(msg.Text, msg.OK)

	case clearStatusMsg:
		if time.Now().Before(m.statusUntil) {
			return m, nil
		}
		m.status = ""
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	switch m.state {
	case stateBuckets:
		m.bucketsList, cmd = m.bucketsList.Update(msg)
	case stateBrowse:
		m.browseList, cmd = m.browseList.Update(msg)
	}
	return m, cmd
}

func (m tuiModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	if m.uploadPrompt == uploadPromptActive {
		switch msg.String() {
		case "esc":
			m.uploadPrompt = uploadPromptOff
			m.uploadInput.Blur()
			return m, nil
		case "enter":
			path := m.uploadInput.Value()
			m.uploadPrompt = uploadPromptOff
			m.uploadInput.Blur()
			m.uploadInput.SetValue("")
			if path == "" {
				return m, nil
			}
			return m, uploadCmd(m.s3, m.cfg, m.currentBucket, m.currentPrefix, path)
		}
		var cmd tea.Cmd
		m.uploadInput, cmd = m.uploadInput.Update(msg)
		return m, cmd
	}

	if m.confirmDelete {
		switch msg.String() {
		case "y", "Y":
			m.confirmDelete = false
			f, ok := m.selectedFile()
			if !ok {
				return m, nil
			}
			return m, deleteCmd(m.s3, m.currentBucket, f.Key)
		case "n", "N", "esc", "q":
			m.confirmDelete = false
		}
		return m, nil
	}

	if m.showHelp {
		if key.Matches(msg, m.keys.Help) || msg.String() == "esc" || msg.String() == "q" {
			m.showHelp = false
		}
		return m, nil
	}

	switch m.state {
	case stateBuckets:
		return m.handleBucketsKey(msg)
	case stateBrowse:
		return m.handleBrowseKey(msg)
	}
	return m, nil
}

func (m tuiModel) handleBucketsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.bucketsList.SettingFilter() {
		var cmd tea.Cmd
		m.bucketsList, cmd = m.bucketsList.Update(msg)
		return m, cmd
	}
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Open):
		b, ok := m.selectedBucket()
		if !ok {
			return m, nil
		}
		m.state = stateBrowse
		m.currentBucket = b
		m.currentPrefix = ""
		m.loading = true
		m.applyLayout()
		return m, loadListingCmd(m.s3, b, "")
	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		return m, loadBucketsCmd(m.s3)
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil
	}
	var cmd tea.Cmd
	m.bucketsList, cmd = m.bucketsList.Update(msg)
	return m, cmd
}

func (m tuiModel) handleBrowseKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.browseList.SettingFilter() {
		var cmd tea.Cmd
		m.browseList, cmd = m.browseList.Update(msg)
		return m, cmd
	}
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Buckets):
		m.state = stateBuckets
		m.loading = true
		m.applyLayout()
		return m, loadBucketsCmd(m.s3)
	case key.Matches(msg, m.keys.Back):
		if m.currentPrefix == "" {
			if m.cfg.DefaultBucket == "" {
				m.state = stateBuckets
				m.loading = true
				m.applyLayout()
				return m, loadBucketsCmd(m.s3)
			}
			return m, nil
		}
		parent := parentPrefix(m.currentPrefix)
		m.loading = true
		return m, loadListingCmd(m.s3, m.currentBucket, parent)
	case key.Matches(msg, m.keys.Open):
		if pfx, ok := m.selectedFolderPrefix(); ok {
			m.loading = true
			return m, loadListingCmd(m.s3, m.currentBucket, pfx)
		}
		if f, ok := m.selectedFile(); ok {
			url, err := ResolveURL(context.Background(), m.s3, m.currentBucket, f.Key)
			if err != nil {
				return m, m.setStatus("url: "+err.Error(), false)
			}
			return m, sharedtui.OpenURLCmd(url)
		}
		return m, nil
	case key.Matches(msg, m.keys.Copy):
		f, ok := m.selectedFile()
		if !ok {
			return m, nil
		}
		url, err := ResolveURL(context.Background(), m.s3, m.currentBucket, f.Key)
		if err != nil {
			return m, m.setStatus("url: "+err.Error(), false)
		}
		return m, sharedtui.CopyToClipboardCmd(url, "copied url")
	case key.Matches(msg, m.keys.CopyLink):
		f, ok := m.selectedFile()
		if !ok {
			return m, nil
		}
		u, ok := m.s3.PublicURL(m.currentBucket, f.Key)
		if !ok {
			return m, m.setStatus("no public URL for bucket "+m.currentBucket, false)
		}
		return m, sharedtui.CopyToClipboardCmd(u, "copied public url")
	case key.Matches(msg, m.keys.CopyKey):
		f, ok := m.selectedFile()
		if !ok {
			return m, nil
		}
		return m, sharedtui.CopyToClipboardCmd(f.Key, "copied key")
	case key.Matches(msg, m.keys.OpenBrowser):
		f, ok := m.selectedFile()
		if !ok {
			return m, nil
		}
		url, err := ResolveURL(context.Background(), m.s3, m.currentBucket, f.Key)
		if err != nil {
			return m, m.setStatus("url: "+err.Error(), false)
		}
		return m, sharedtui.OpenURLCmd(url)
	case key.Matches(msg, m.keys.Delete):
		if _, ok := m.selectedFile(); ok {
			m.confirmDelete = true
		}
		return m, nil
	case key.Matches(msg, m.keys.Upload):
		m.uploadPrompt = uploadPromptActive
		m.uploadInput.Focus()
		return m, nil
	case key.Matches(msg, m.keys.Preview):
		m.showPreview = !m.showPreview
		m.applyLayout()
		if m.showPreview {
			return m, m.maybePreviewCmd()
		}
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		return m, loadListingCmd(m.s3, m.currentBucket, m.currentPrefix)
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil
	}
	var cmd tea.Cmd
	m.browseList, cmd = m.browseList.Update(msg)
	if m.showPreview {
		return m, tea.Batch(cmd, m.maybePreviewCmd())
	}
	return m, cmd
}

func (m *tuiModel) maybePreviewCmd() tea.Cmd {
	if !m.showPreview {
		return nil
	}
	f, ok := m.selectedFile()
	if !ok {
		m.preview.SetContent("")
		return nil
	}
	m.previewSeq++
	return debouncePreviewCmd(m.previewSeq, m.currentBucket, f.Key, m.preview.Width(), m.preview.Height())
}
