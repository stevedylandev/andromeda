package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Focus int

const (
	FocusList Focus = iota
	FocusContent
	FocusCreateName
	FocusCreateContent
	FocusEditName
	FocusEditContent
	FocusSearch
)

type Model struct {
	backend  Backend
	isRemote bool

	snippets []Snippet
	filtered []int
	cursor   int

	focus         Focus
	showHelp      bool
	confirmDelete bool

	nameInput   textinput.Model
	contentArea textarea.Model
	searchInput textinput.Model
	contentVP   viewport.Model
	help        help.Model
	keys        keyMap

	highlighter *highlighter

	editShortID string

	status      string
	statusOK    bool
	statusUntil time.Time

	width, height int
	ready         bool
	loading       bool
}

func newModel(backend Backend) Model {
	ti := textinput.New()
	ti.Placeholder = "name.ext"
	ti.Prompt = ""
	ti.CharLimit = 200

	ta := textarea.New()
	ta.Placeholder = "Paste code..."
	ta.ShowLineNumbers = true
	ta.Prompt = ""

	si := textinput.New()
	si.Placeholder = "search names"
	si.Prompt = "/ "

	vp := viewport.New(0, 0)

	return Model{
		backend:     backend,
		isRemote:    backend.RemoteURL() != "",
		focus:       FocusList,
		nameInput:   ti,
		contentArea: ta,
		searchInput: si,
		contentVP:   vp,
		help:        help.New(),
		keys:        defaultKeys(),
		highlighter: newHighlighter(),
	}
}

func (m Model) Init() tea.Cmd {
	return loadSnippetsCmd(m.backend)
}

func (m *Model) visible() []Snippet {
	if m.filtered == nil {
		return m.snippets
	}
	out := make([]Snippet, 0, len(m.filtered))
	for _, i := range m.filtered {
		out = append(out, m.snippets[i])
	}
	return out
}

func (m *Model) current() *Snippet {
	list := m.visible()
	if m.cursor < 0 || m.cursor >= len(list) {
		return nil
	}
	return &list[m.cursor]
}

func (m *Model) applyFilter(q string) {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" {
		m.filtered = nil
		if m.cursor >= len(m.snippets) {
			m.cursor = 0
		}
		return
	}
	idx := []int{}
	for i, s := range m.snippets {
		if strings.Contains(strings.ToLower(s.Name), q) || strings.Contains(strings.ToLower(s.ShortID), q) {
			idx = append(idx, i)
		}
	}
	m.filtered = idx
	if m.cursor >= len(idx) {
		m.cursor = 0
	}
}

func (m *Model) setStatus(text string, ok bool) tea.Cmd {
	m.status = text
	m.statusOK = ok
	m.statusUntil = time.Now().Add(2 * time.Second)
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearStatusMsg{} })
}

func (m *Model) shareURL(shortID string) string {
	if m.backend.RemoteURL() == "" {
		return ""
	}
	return strings.TrimRight(m.backend.RemoteURL(), "/") + "/s/" + shortID
}
