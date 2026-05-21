package tui

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	sharedtui "github.com/stevedylandev/andromeda/pkg/tui"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.applyLayout()
		return m, nil

	case snippetsLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			return m, m.setStatus("load: "+msg.Err.Error(), false)
		}
		cmd := m.list.SetSnippets(msg.Snippets)
		m.refreshContentFromSelection()
		return m, cmd

	case snippetSavedMsg:
		if msg.Err != nil {
			return m, m.setStatus("save: "+msg.Err.Error(), false)
		}
		if msg.Snippet != nil {
			m.cont.Invalidate(msg.Snippet.ShortID)
		}
		m.state = stateList
		m.form.Blur()
		return m, tea.Batch(loadSnippetsCmd(m.backend), m.setStatus("saved", true))

	case snippetDeletedMsg:
		if msg.Err != nil {
			return m, m.setStatus("delete: "+msg.Err.Error(), false)
		}
		m.cont.Invalidate(msg.ShortID)
		m.state = stateList
		return m, tea.Batch(loadSnippetsCmd(m.backend), m.setStatus("deleted", true))

	case editorFinishedMsg:
		if msg.Err != nil {
			return m, m.setStatus("editor: "+msg.Err.Error(), false)
		}
		if msg.Tag == "" {
			m.form.SetContent(msg.Content)
			return m, nil
		}
		var orig *Snippet
		for _, it := range m.list.inner.Items() {
			si, ok := it.(snippetItem)
			if ok && si.snippet.ShortID == msg.Tag {
				s := si.snippet
				orig = &s
				break
			}
		}
		if orig == nil || strings.TrimRight(orig.Content, "\n") == strings.TrimRight(msg.Content, "\n") {
			return m, nil
		}
		return m, saveSnippetCmd(m.backend, msg.Tag, orig.Name, msg.Content)

	case submitFormMsg:
		return m, saveSnippetCmd(m.backend, msg.ShortID, msg.Name, msg.Content)

	case cancelFormMsg:
		m.state = stateList
		return m, nil

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

	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	if m.confirmDelete {
		switch msg.String() {
		case "y", "Y":
			m.confirmDelete = false
			s, ok := m.list.Selected()
			if !ok {
				return m, nil
			}
			return m, deleteSnippetCmd(m.backend, s.ShortID)
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
	case stateList:
		return m.handleListKey(msg)
	case stateContent:
		return m.handleContentKey(msg)
	case stateForm:
		var cmd tea.Cmd
		m.form, cmd = m.form.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleListKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.list.IsFiltering() {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		m.refreshContentFromSelection()
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Open):
		if s, ok := m.list.Selected(); ok {
			m.cont.SetSnippet(&s)
			m.state = stateContent
		}
		return m, nil
	case key.Matches(msg, m.keys.Create):
		m.form.StartCreate()
		m.state = stateForm
		return m, nil
	case key.Matches(msg, m.keys.Edit):
		if s, ok := m.list.Selected(); ok {
			m.form.StartEdit(s)
			m.state = stateForm
		}
		return m, nil
	case key.Matches(msg, m.keys.ExtEdit):
		if s, ok := m.list.Selected(); ok {
			return m, openExternalEditor(s.ShortID, s.Name, s.Content)
		}
		return m, nil
	case key.Matches(msg, m.keys.Delete):
		if _, ok := m.list.Selected(); ok {
			m.confirmDelete = true
		}
		return m, nil
	case key.Matches(msg, m.keys.Copy):
		if s, ok := m.list.Selected(); ok {
			return m, sharedtui.CopyToClipboardCmd(s.Content, "copied text")
		}
		return m, nil
	case key.Matches(msg, m.keys.CopyLink):
		if !m.isRemote {
			return m, m.setStatus("local mode: no link", false)
		}
		if s, ok := m.list.Selected(); ok {
			return m, sharedtui.CopyToClipboardCmd(shareLinkURL(m.backend.RemoteURL(), s.ShortID), "copied link")
		}
		return m, nil
	case key.Matches(msg, m.keys.OpenBrowser):
		if !m.isRemote {
			return m, nil
		}
		if s, ok := m.list.Selected(); ok {
			return m, sharedtui.OpenURLCmd(shareLinkURL(m.backend.RemoteURL(), s.ShortID))
		}
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		return m, loadSnippetsCmd(m.backend)
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.refreshContentFromSelection()
	return m, cmd
}

func (m Model) handleContentKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.ToggleWrap):
		m.cont.ToggleWrap()
		if m.cont.Wrap() {
			return m, m.setStatus("wrap on", true)
		}
		return m, m.setStatus("wrap off", true)
	case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.Back):
		m.state = stateList
		return m, nil
	case key.Matches(msg, m.keys.ScrollDown):
		m.cont = m.cont.ScrollDown(1)
		return m, nil
	case key.Matches(msg, m.keys.ScrollUp):
		m.cont = m.cont.ScrollUp(1)
		return m, nil
	case key.Matches(msg, m.keys.Edit):
		if s, ok := m.list.Selected(); ok {
			m.form.StartEdit(s)
			m.state = stateForm
		}
		return m, nil
	case key.Matches(msg, m.keys.ExtEdit):
		if s, ok := m.list.Selected(); ok {
			return m, openExternalEditor(s.ShortID, s.Name, s.Content)
		}
		return m, nil
	case key.Matches(msg, m.keys.Copy):
		if s, ok := m.list.Selected(); ok {
			return m, sharedtui.CopyToClipboardCmd(s.Content, "copied text")
		}
		return m, nil
	case key.Matches(msg, m.keys.CopyLink):
		if !m.isRemote {
			return m, m.setStatus("local mode: no link", false)
		}
		if s, ok := m.list.Selected(); ok {
			return m, sharedtui.CopyToClipboardCmd(shareLinkURL(m.backend.RemoteURL(), s.ShortID), "copied link")
		}
		return m, nil
	case key.Matches(msg, m.keys.OpenBrowser):
		if !m.isRemote {
			return m, nil
		}
		if s, ok := m.list.Selected(); ok {
			return m, sharedtui.OpenURLCmd(shareLinkURL(m.backend.RemoteURL(), s.ShortID))
		}
		return m, nil
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil
	}

	var cmd tea.Cmd
	m.cont, cmd = m.cont.Update(msg)
	return m, cmd
}

func (m *Model) refreshContentFromSelection() {
	if s, ok := m.list.Selected(); ok {
		m.cont.SetSnippet(&s)
	} else {
		m.cont.SetSnippet(nil)
	}
}

func (m *Model) applyLayout() {
	if !m.ready {
		return
	}
	listW := m.width * 30 / 100
	if listW < 24 {
		listW = 24
	}
	contentW := m.width - listW - 2
	if contentW < 20 {
		contentW = 20
	}
	bodyH := m.height - 2
	if bodyH < 5 {
		bodyH = 5
	}

	m.list.SetSize(max(listW-paneFrameWidth(), 1), max(bodyH-paneFrameHeight(), 1))
	m.cont.SetSize(max(contentW-paneFrameWidth(), 1), max(bodyH-paneFrameHeight()-1, 1))
	m.form.SetSize(max(contentW-paneFrameWidth(), 1), max(bodyH-paneFrameHeight(), 1))
}

func (m *Model) setStatus(text string, ok bool) tea.Cmd {
	m.status = text
	m.statusOK = ok
	m.statusUntil = time.Now().Add(2 * time.Second)
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearStatusMsg{} })
}
