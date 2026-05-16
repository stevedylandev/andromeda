package tui

import (
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func loadSnippetsCmd(b Backend) tea.Cmd {
	return func() tea.Msg {
		list, err := b.List()
		return snippetsLoadedMsg{snippets: list, err: err}
	}
}

func saveSnippetCmd(b Backend, shortID, name, content string) tea.Cmd {
	return func() tea.Msg {
		var (
			s   *Snippet
			err error
		)
		if shortID == "" {
			s, err = b.Create(name, content)
		} else {
			s, err = b.Update(shortID, name, content)
		}
		return snippetSavedMsg{snippet: s, err: err}
	}
}

func deleteSnippetCmd(b Backend, shortID string) tea.Cmd {
	return func() tea.Msg {
		_, err := b.Delete(shortID)
		return snippetDeletedMsg{shortID: shortID, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.resizePanes()
		return m, nil

	case snippetsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			return m, m.setStatus("load: "+msg.err.Error(), false)
		}
		m.snippets = msg.snippets
		m.applyFilter(m.searchInput.Value())
		m.refreshPreview()
		return m, nil

	case snippetSavedMsg:
		if msg.err != nil {
			return m, m.setStatus("save: "+msg.err.Error(), false)
		}
		if msg.snippet != nil && m.highlighter != nil {
			m.highlighter.invalidate(msg.snippet.ShortID)
		}
		m.focus = FocusList
		m.nameInput.Reset()
		m.contentArea.Reset()
		m.editShortID = ""
		return m, tea.Batch(loadSnippetsCmd(m.backend), m.setStatus("saved", true))

	case snippetDeletedMsg:
		if msg.err != nil {
			return m, m.setStatus("delete: "+msg.err.Error(), false)
		}
		if m.highlighter != nil {
			m.highlighter.invalidate(msg.shortID)
		}
		return m, tea.Batch(loadSnippetsCmd(m.backend), m.setStatus("deleted", true))

	case editorFinishedMsg:
		if msg.err != nil {
			return m, m.setStatus("editor: "+msg.err.Error(), false)
		}
		if msg.shortID == "" {
			m.contentArea.SetValue(msg.content)
			return m, nil
		}
		var orig *Snippet
		for i := range m.snippets {
			if m.snippets[i].ShortID == msg.shortID {
				orig = &m.snippets[i]
				break
			}
		}
		if orig == nil || strings.TrimRight(orig.Content, "\n") == strings.TrimRight(msg.content, "\n") {
			return m, nil
		}
		return m, saveSnippetCmd(m.backend, msg.shortID, orig.Name, msg.content)

	case statusMsg:
		return m, m.setStatus(msg.text, msg.ok)

	case clearStatusMsg:
		m.status = ""
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *Model) resizePanes() {
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

	m.contentVP.Width = contentW - 2
	m.contentVP.Height = bodyH - 2

	m.nameInput.Width = contentW - 4
	if m.wrapContent {
		m.contentArea.SetWidth(contentW - 2)
	} else {
		m.contentArea.SetWidth(10000)
	}
	m.contentArea.SetHeight(bodyH - 5)

	m.searchInput.Width = listW - 4

	m.refreshPreview()
}

func (m *Model) refreshPreview() {
	s := m.current()
	if s == nil {
		m.contentVP.SetContent("")
		return
	}
	body := s.Content
	if m.highlighter != nil {
		body = m.highlighter.render(s.ShortID, s.Name, s.Content)
	}
	if m.wrapContent && m.contentVP.Width > 0 {
		body = lipgloss.NewStyle().Width(m.contentVP.Width).Render(body)
	}
	m.contentVP.SetContent(body)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	if m.confirmDelete {
		switch msg.String() {
		case "y", "Y":
			s := m.current()
			m.confirmDelete = false
			if s == nil {
				return m, nil
			}
			return m, deleteSnippetCmd(m.backend, s.ShortID)
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
	case FocusCreateName, FocusCreateContent, FocusEditName, FocusEditContent:
		return m.keyForm(msg)
	case FocusSearch:
		return m.keySearch(msg)
	}
	return m, nil
}

func (m Model) keyList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	list := m.visible()
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(list)-1 {
			m.cursor++
			m.refreshPreview()
		}
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			m.refreshPreview()
		}
	case key.Matches(msg, m.keys.Open):
		if len(list) > 0 {
			m.focus = FocusContent
			m.contentVP.GotoTop()
		}
	case key.Matches(msg, m.keys.Create):
		m.focus = FocusCreateName
		m.editShortID = ""
		m.nameInput.SetValue("")
		m.contentArea.SetValue("")
		m.nameInput.Focus()
		m.contentArea.Blur()
	case key.Matches(msg, m.keys.Edit):
		s := m.current()
		if s != nil {
			m.focus = FocusEditName
			m.editShortID = s.ShortID
			m.nameInput.SetValue(s.Name)
			m.contentArea.SetValue(s.Content)
			m.nameInput.Focus()
			m.contentArea.Blur()
		}
	case key.Matches(msg, m.keys.ExtEdit):
		s := m.current()
		if s != nil {
			return m, openExternalEditor(s.ShortID, s.Name, s.Content)
		}
	case key.Matches(msg, m.keys.Delete):
		if m.current() != nil {
			m.confirmDelete = true
		}
	case key.Matches(msg, m.keys.Copy):
		s := m.current()
		if s != nil {
			if err := clipboard.WriteAll(s.Content); err != nil {
				return m, m.setStatus("clipboard: "+err.Error(), false)
			}
			return m, m.setStatus("copied text", true)
		}
	case key.Matches(msg, m.keys.CopyLink):
		s := m.current()
		if s != nil && m.isRemote {
			link := m.shareURL(s.ShortID)
			if err := clipboard.WriteAll(link); err != nil {
				return m, m.setStatus("clipboard: "+err.Error(), false)
			}
			return m, m.setStatus("copied link", true)
		}
		return m, m.setStatus("local mode: no link", false)
	case key.Matches(msg, m.keys.OpenBrowser):
		s := m.current()
		if s != nil && m.isRemote {
			link := m.shareURL(s.ShortID)
			if err := openURL(link); err != nil {
				return m, m.setStatus("open: "+err.Error(), false)
			}
			return m, m.setStatus("opened "+link, true)
		}
	case key.Matches(msg, m.keys.Search):
		m.focus = FocusSearch
		m.searchInput.Focus()
	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		return m, loadSnippetsCmd(m.backend)
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
	}
	return m, nil
}

func (m Model) keyContent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.WrapToggle):
		m.wrapContent = !m.wrapContent
		m.contentVP.GotoTop()
		m.refreshPreview()
		if m.wrapContent {
			return m, m.setStatus("wrap on", true)
		}
		return m, m.setStatus("wrap off", true)
	case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.Back):
		m.focus = FocusList
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.contentVP.ScrollDown(1)
	case key.Matches(msg, m.keys.Up):
		m.contentVP.ScrollUp(1)
	case key.Matches(msg, m.keys.Edit):
		s := m.current()
		if s != nil {
			m.focus = FocusEditName
			m.editShortID = s.ShortID
			m.nameInput.SetValue(s.Name)
			m.contentArea.SetValue(s.Content)
			m.nameInput.Focus()
		}
	case key.Matches(msg, m.keys.ExtEdit):
		s := m.current()
		if s != nil {
			return m, openExternalEditor(s.ShortID, s.Name, s.Content)
		}
	case key.Matches(msg, m.keys.Copy):
		s := m.current()
		if s != nil {
			clipboard.WriteAll(s.Content)
			return m, m.setStatus("copied text", true)
		}
	case key.Matches(msg, m.keys.CopyLink):
		s := m.current()
		if s != nil && m.isRemote {
			clipboard.WriteAll(m.shareURL(s.ShortID))
			return m, m.setStatus("copied link", true)
		}
	case key.Matches(msg, m.keys.OpenBrowser):
		s := m.current()
		if s != nil && m.isRemote {
			openURL(m.shareURL(s.ShortID))
		}
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
	}
	var cmd tea.Cmd
	m.contentVP, cmd = m.contentVP.Update(msg)
	return m, cmd
}

func (m Model) keyForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.WrapToggle):
		m.wrapContent = !m.wrapContent
		if m.wrapContent {
			m.contentArea.SetWidth(m.contentVP.Width)
			return m, m.setStatus("wrap on", true)
		}
		m.contentArea.SetWidth(10000)
		return m, m.setStatus("wrap off", true)
	case key.Matches(msg, m.keys.Cancel):
		m.focus = FocusList
		m.nameInput.Blur()
		m.contentArea.Blur()
		return m, nil
	case key.Matches(msg, m.keys.Save):
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			return m, m.setStatus("name required", false)
		}
		content := m.contentArea.Value()
		if strings.TrimSpace(content) == "" {
			return m, m.setStatus("content required", false)
		}
		return m, saveSnippetCmd(m.backend, m.editShortID, name, content)
	case key.Matches(msg, m.keys.SwitchField):
		switch m.focus {
		case FocusCreateName:
			m.focus = FocusCreateContent
		case FocusCreateContent:
			m.focus = FocusCreateName
		case FocusEditName:
			m.focus = FocusEditContent
		case FocusEditContent:
			m.focus = FocusEditName
		}
		m.applyFormFocus()
		return m, nil
	}

	var cmd tea.Cmd
	switch m.focus {
	case FocusCreateName, FocusEditName:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case FocusCreateContent, FocusEditContent:
		m.contentArea, cmd = m.contentArea.Update(msg)
	}
	return m, cmd
}

func (m *Model) applyFormFocus() {
	switch m.focus {
	case FocusCreateName, FocusEditName:
		m.nameInput.Focus()
		m.contentArea.Blur()
	case FocusCreateContent, FocusEditContent:
		m.contentArea.Focus()
		m.nameInput.Blur()
	}
}

func (m Model) keySearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
