package tui

import (
	"strings"

	"github.com/atotto/clipboard"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func loadNotesCmd(b Backend) tea.Cmd {
	return func() tea.Msg {
		notes, err := b.List()
		return notesLoadedMsg{notes: notes, err: err}
	}
}

func saveNoteCmd(b Backend, shortID, title, content string) tea.Cmd {
	return func() tea.Msg {
		var (
			note *Note
			err  error
		)
		if shortID == "" {
			note, err = b.Create(title, content)
		} else {
			note, err = b.Update(shortID, title, content)
		}
		return noteSavedMsg{note: note, err: err}
	}
}

func deleteNoteCmd(b Backend, shortID string) tea.Cmd {
	return func() tea.Msg {
		_, err := b.Delete(shortID)
		return noteDeletedMsg{shortID: shortID, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.resizePanes()
		return m, nil

	case notesLoadedMsg:
		if msg.err != nil {
			cmd := m.setStatus("load: "+msg.err.Error(), false)
			return m, cmd
		}
		m.notes = msg.notes
		m.applyFilter(m.searchInput.Value())
		m.refreshPreview()
		return m, nil

	case noteSavedMsg:
		if msg.err != nil {
			return m, m.setStatus("save: "+msg.err.Error(), false)
		}
		if m.renderer != nil && msg.note != nil {
			m.renderer.invalidate(msg.note.ShortID)
		}
		m.focus = FocusList
		m.titleInput.Reset()
		m.contentArea.Reset()
		m.editShortID = ""
		return m, loadNotesCmd(m.backend)

	case noteDeletedMsg:
		if msg.err != nil {
			return m, m.setStatus("delete: "+msg.err.Error(), false)
		}
		if m.renderer != nil {
			m.renderer.invalidate(msg.shortID)
		}
		return m, tea.Batch(loadNotesCmd(m.backend), m.setStatus("deleted", true))

	case editorFinishedMsg:
		if msg.err != nil {
			return m, m.setStatus("editor: "+msg.err.Error(), false)
		}
		if msg.shortID == "" {
			m.contentArea.SetValue(msg.content)
			return m, nil
		}
		var orig *Note
		for i := range m.notes {
			if m.notes[i].ShortID == msg.shortID {
				orig = &m.notes[i]
				break
			}
		}
		if orig == nil || strings.TrimRight(orig.Content, "\n") == strings.TrimRight(msg.content, "\n") {
			return m, nil
		}
		return m, saveNoteCmd(m.backend, msg.shortID, orig.Title, msg.content)

	case statusMsg:
		return m, m.setStatus(msg.text, msg.ok)

	case clearStatusMsg:
		m.status = ""
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *Model) resizePanes() {
	if !m.ready {
		return
	}

	_, contentOuterW := splitWidths(m.width)
	bodyOuterH := splitBodyHeight(m.height)
	contentInnerW := maxInt(contentOuterW-paneFrameWidth(), 20)
	contentInnerH := maxInt(bodyOuterH-paneFrameHeight(), 3)

	m.contentVP.SetWidth(maxInt(contentInnerW, 1))
	m.contentVP.SetHeight(maxInt(contentInnerH-1, 1))

	m.titleInput.SetWidth(maxInt(contentInnerW-4, 1))
	m.contentArea.SetWidth(maxInt(contentInnerW-2, 1))
	m.contentArea.SetHeight(maxInt(contentInnerH-6, 1))

	listOuterW, _ := splitWidths(m.width)
	listInnerW := maxInt(listOuterW-paneFrameWidth(), 1)
	m.searchInput.SetWidth(maxInt(listInnerW-2, 1))

	if m.renderer == nil {
		m.renderer = newRenderer(contentInnerW)
	} else {
		m.renderer.resize(contentInnerW)
	}
	m.refreshPreview()
}

func (m *Model) refreshPreview() {
	if m.renderer == nil {
		return
	}
	n := m.currentNote()
	if n == nil {
		m.contentVP.SetContent("")
		return
	}
	body := n.Content
	if !m.wrap {
		// raw view: no rendering
		m.contentVP.SetContent(body)
		return
	}
	m.contentVP.SetContent(m.renderer.render(n.ShortID, body))
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.confirmDelete {
		switch msg.String() {
		case "y", "Y":
			n := m.currentNote()
			m.confirmDelete = false
			if n == nil {
				return m, nil
			}
			return m, deleteNoteCmd(m.backend, n.ShortID)
		case "n", "N", "esc", "q":
			m.confirmDelete = false
			return m, nil
		}
		return m, nil
	}

	if m.showHelp {
		if key.Matches(msg, m.keys.Help) || msg.String() == "esc" || msg.String() == "q" {
			m.showHelp = false
		}
		return m, nil
	}

	switch m.focus {
	case FocusList:
		return m.keyList(msg)
	case FocusContent:
		return m.keyContent(msg)
	case FocusCreateTitle, FocusCreateContent, FocusEditTitle, FocusEditContent:
		return m.keyForm(msg)
	case FocusSearch:
		return m.keySearch(msg)
	}
	return m, nil
}

func (m Model) keyList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	notes := m.visibleNotes()
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(notes)-1 {
			m.cursor++
			m.refreshPreview()
		}
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			m.refreshPreview()
		}
	case key.Matches(msg, m.keys.Open):
		if len(notes) > 0 {
			m.focus = FocusContent
			m.contentVP.GotoTop()
		}
	case key.Matches(msg, m.keys.Create):
		m.focus = FocusCreateTitle
		m.editShortID = ""
		m.titleInput.SetValue("")
		m.contentArea.SetValue("")
		m.titleInput.Focus()
		m.contentArea.Blur()
	case key.Matches(msg, m.keys.Edit):
		n := m.currentNote()
		if n != nil {
			m.focus = FocusEditTitle
			m.editShortID = n.ShortID
			m.titleInput.SetValue(n.Title)
			m.contentArea.SetValue(n.Content)
			m.titleInput.Focus()
			m.contentArea.Blur()
		}
	case key.Matches(msg, m.keys.ExtEdit):
		n := m.currentNote()
		if n != nil {
			return m, openExternalEditor(n.ShortID, n.Content)
		}
	case key.Matches(msg, m.keys.Delete):
		if m.currentNote() != nil {
			m.confirmDelete = true
		}
	case key.Matches(msg, m.keys.Copy):
		n := m.currentNote()
		if n != nil {
			if err := clipboard.WriteAll(n.Content); err != nil {
				return m, m.setStatus("clipboard: "+err.Error(), false)
			}
			return m, m.setStatus("copied text", true)
		}
	case key.Matches(msg, m.keys.CopyLink):
		n := m.currentNote()
		if n != nil && m.isRemote {
			link := strings.TrimRight(m.backend.RemoteURL(), "/") + "/notes/" + n.ShortID
			if err := clipboard.WriteAll(link); err != nil {
				return m, m.setStatus("clipboard: "+err.Error(), false)
			}
			return m, m.setStatus("copied link", true)
		}
		return m, m.setStatus("local mode: no link", false)
	case key.Matches(msg, m.keys.OpenBrowser):
		n := m.currentNote()
		if n != nil && m.isRemote {
			link := strings.TrimRight(m.backend.RemoteURL(), "/") + "/notes/" + n.ShortID
			if err := openURL(link); err != nil {
				return m, m.setStatus("open: "+err.Error(), false)
			}
			return m, m.setStatus("opened "+link, true)
		}
	case key.Matches(msg, m.keys.Search):
		m.focus = FocusSearch
		m.searchInput.Focus()
	case key.Matches(msg, m.keys.Refresh):
		if m.isRemote {
			return m, loadNotesCmd(m.backend)
		}
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
	}
	return m, nil
}

func (m Model) keyContent(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.Back):
		m.focus = FocusList
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.contentVP.ScrollDown(1)
	case key.Matches(msg, m.keys.Up):
		m.contentVP.ScrollUp(1)
	case key.Matches(msg, m.keys.Edit):
		n := m.currentNote()
		if n != nil {
			m.focus = FocusEditTitle
			m.editShortID = n.ShortID
			m.titleInput.SetValue(n.Title)
			m.contentArea.SetValue(n.Content)
			m.titleInput.Focus()
		}
	case key.Matches(msg, m.keys.ExtEdit):
		n := m.currentNote()
		if n != nil {
			return m, openExternalEditor(n.ShortID, n.Content)
		}
	case key.Matches(msg, m.keys.Copy):
		n := m.currentNote()
		if n != nil {
			clipboard.WriteAll(n.Content)
			return m, m.setStatus("copied text", true)
		}
	case key.Matches(msg, m.keys.CopyLink):
		n := m.currentNote()
		if n != nil && m.isRemote {
			link := strings.TrimRight(m.backend.RemoteURL(), "/") + "/notes/" + n.ShortID
			clipboard.WriteAll(link)
			return m, m.setStatus("copied link", true)
		}
	case key.Matches(msg, m.keys.OpenBrowser):
		n := m.currentNote()
		if n != nil && m.isRemote {
			openURL(strings.TrimRight(m.backend.RemoteURL(), "/") + "/notes/" + n.ShortID)
		}
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
	case key.Matches(msg, m.keys.ToggleWrap):
		m.wrap = !m.wrap
		m.refreshPreview()
	}
	var cmd tea.Cmd
	m.contentVP, cmd = m.contentVP.Update(msg)
	return m, cmd
}

func (m Model) keyForm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.focus = FocusList
		m.titleInput.Blur()
		m.contentArea.Blur()
		return m, nil
	case key.Matches(msg, m.keys.Save):
		title := strings.TrimSpace(m.titleInput.Value())
		if title == "" {
			return m, m.setStatus("title required", false)
		}
		return m, saveNoteCmd(m.backend, m.editShortID, title, m.contentArea.Value())
	case key.Matches(msg, m.keys.SwitchField):
		switch m.focus {
		case FocusCreateTitle:
			m.focus = FocusCreateContent
		case FocusCreateContent:
			m.focus = FocusCreateTitle
		case FocusEditTitle:
			m.focus = FocusEditContent
		case FocusEditContent:
			m.focus = FocusEditTitle
		}
		m.applyFormFocus()
		return m, nil
	case key.Matches(msg, m.keys.ToggleWrap):
		m.wrap = !m.wrap
		return m, nil
	}

	var cmd tea.Cmd
	switch m.focus {
	case FocusCreateTitle, FocusEditTitle:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case FocusCreateContent, FocusEditContent:
		m.contentArea, cmd = m.contentArea.Update(msg)
	}
	return m, cmd
}

func (m *Model) applyFormFocus() {
	switch m.focus {
	case FocusCreateTitle, FocusEditTitle:
		m.titleInput.Focus()
		m.contentArea.Blur()
	case FocusCreateContent, FocusEditContent:
		m.contentArea.Focus()
		m.titleInput.Blur()
	}
}

func (m Model) keySearch(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchInput.SetValue("")
		m.searchInput.Blur()
		m.focus = FocusList
		m.applyFilter("")
		m.refreshPreview()
		return m, nil
	case "enter":
		m.searchInput.Blur()
		m.focus = FocusList
		return m, nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.applyFilter(m.searchInput.Value())
	m.refreshPreview()
	return m, cmd
}
