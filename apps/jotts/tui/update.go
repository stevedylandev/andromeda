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

	case notesLoadedMsg:
		if msg.Err != nil {
			return m, m.setStatus("load: "+msg.Err.Error(), false)
		}
		cmd := m.list.SetNotes(msg.Notes)
		if n, ok := m.list.Selected(); ok {
			m.cont.SetNote(&n)
		} else {
			m.cont.SetNote(nil)
		}
		return m, cmd

	case noteSavedMsg:
		if msg.Err != nil {
			return m, m.setStatus("save: "+msg.Err.Error(), false)
		}
		if msg.Note != nil {
			m.cont.Invalidate(msg.Note.ShortID)
		}
		m.state = stateList
		m.form.Blur()
		return m, tea.Batch(loadNotesCmd(m.backend), m.setStatus("saved", true))

	case noteDeletedMsg:
		if msg.Err != nil {
			return m, m.setStatus("delete: "+msg.Err.Error(), false)
		}
		m.cont.Invalidate(msg.ShortID)
		m.state = stateList
		return m, tea.Batch(loadNotesCmd(m.backend), m.setStatus("deleted", true))

	case editorFinishedMsg:
		if msg.Err != nil {
			return m, m.setStatus("editor: "+msg.Err.Error(), false)
		}
		if msg.Tag == "" {
			m.form.SetContent(msg.Content)
			return m, nil
		}
		var orig *Note
		for _, it := range m.list.inner.Items() {
			ni, ok := it.(noteItem)
			if ok && ni.note.ShortID == msg.Tag {
				n := ni.note
				orig = &n
				break
			}
		}
		if orig == nil || strings.TrimRight(orig.Content, "\n") == strings.TrimRight(msg.Content, "\n") {
			return m, nil
		}
		return m, saveNoteCmd(m.backend, msg.Tag, orig.Title, msg.Content)

	case submitFormMsg:
		return m, saveNoteCmd(m.backend, msg.ShortID, msg.Title, msg.Content)

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

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.confirmDelete {
		switch msg.String() {
		case "y", "Y":
			m.confirmDelete = false
			n, ok := m.list.Selected()
			if !ok {
				return m, nil
			}
			return m, deleteNoteCmd(m.backend, n.ShortID)
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
	// While the list is in filter-entry mode, route every key to it
	// so typing and esc/enter behave as expected.
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
		if n, ok := m.list.Selected(); ok {
			m.cont.SetNote(&n)
			m.state = stateContent
		}
		return m, nil
	case key.Matches(msg, m.keys.Create):
		m.form.StartCreate()
		m.state = stateForm
		return m, nil
	case key.Matches(msg, m.keys.Edit):
		if n, ok := m.list.Selected(); ok {
			m.form.StartEdit(n)
			m.state = stateForm
		}
		return m, nil
	case key.Matches(msg, m.keys.ExtEdit):
		if n, ok := m.list.Selected(); ok {
			return m, openExternalEditor(n.ShortID, n.Content)
		}
		return m, nil
	case key.Matches(msg, m.keys.Delete):
		if _, ok := m.list.Selected(); ok {
			m.confirmDelete = true
		}
		return m, nil
	case key.Matches(msg, m.keys.Copy):
		if n, ok := m.list.Selected(); ok {
			return m, sharedtui.CopyToClipboardCmd(n.Content, "copied text")
		}
		return m, nil
	case key.Matches(msg, m.keys.CopyLink):
		if !m.isRemote {
			return m, m.setStatus("local mode: no link", false)
		}
		if n, ok := m.list.Selected(); ok {
			return m, sharedtui.CopyToClipboardCmd(noteLinkURL(m.backend.RemoteURL(), n.ShortID), "copied link")
		}
		return m, nil
	case key.Matches(msg, m.keys.OpenBrowser):
		if !m.isRemote {
			return m, nil
		}
		if n, ok := m.list.Selected(); ok {
			return m, sharedtui.OpenURLCmd(noteLinkURL(m.backend.RemoteURL(), n.ShortID))
		}
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		if m.isRemote {
			return m, loadNotesCmd(m.backend)
		}
		return m, nil
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
		if n, ok := m.list.Selected(); ok {
			m.form.StartEdit(n)
			m.state = stateForm
		}
		return m, nil
	case key.Matches(msg, m.keys.ExtEdit):
		if n, ok := m.list.Selected(); ok {
			return m, openExternalEditor(n.ShortID, n.Content)
		}
		return m, nil
	case key.Matches(msg, m.keys.Copy):
		if n, ok := m.list.Selected(); ok {
			return m, sharedtui.CopyToClipboardCmd(n.Content, "copied text")
		}
		return m, nil
	case key.Matches(msg, m.keys.CopyLink):
		if !m.isRemote {
			return m, m.setStatus("local mode: no link", false)
		}
		if n, ok := m.list.Selected(); ok {
			return m, sharedtui.CopyToClipboardCmd(noteLinkURL(m.backend.RemoteURL(), n.ShortID), "copied link")
		}
		return m, nil
	case key.Matches(msg, m.keys.OpenBrowser):
		if !m.isRemote {
			return m, nil
		}
		if n, ok := m.list.Selected(); ok {
			return m, sharedtui.OpenURLCmd(noteLinkURL(m.backend.RemoteURL(), n.ShortID))
		}
		return m, nil
	case key.Matches(msg, m.keys.ToggleWrap):
		m.cont.ToggleWrap()
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
	if n, ok := m.list.Selected(); ok {
		m.cont.SetNote(&n)
	} else {
		m.cont.SetNote(nil)
	}
}

func (m *Model) applyLayout() {
	if !m.ready {
		return
	}
	listW, contentW := splitWidths(m.width)
	bodyH := splitBodyHeight(m.height - 1)

	listInnerW := max(listW-paneFrameWidth(), 1)
	listInnerH := max(bodyH-paneFrameHeight(), 1)
	m.list.SetSize(listInnerW, listInnerH)

	contentInnerW := max(contentW-paneFrameWidth(), 20)
	contentInnerH := max(bodyH-paneFrameHeight(), 3)
	m.cont.SetSize(contentInnerW, max(contentInnerH-1, 1))
	m.form.SetSize(contentInnerW, contentInnerH)
}

func (m *Model) setStatus(text string, ok bool) tea.Cmd {
	m.status = text
	m.statusOK = ok
	m.statusUntil = time.Now().Add(2 * time.Second)
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearStatusMsg{} })
}
