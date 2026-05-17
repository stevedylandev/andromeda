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
	FocusCreateTitle
	FocusCreateContent
	FocusEditTitle
	FocusEditContent
	FocusSearch
)

type Model struct {
	backend  Backend
	isRemote bool

	notes    []Note
	filtered []int
	cursor   int

	focus         Focus
	showHelp      bool
	confirmDelete bool

	titleInput  textinput.Model
	contentArea textarea.Model
	searchInput textinput.Model
	contentVP   viewport.Model
	help        help.Model
	keys        keyMap

	renderer *mdRenderer
	wrap     bool

	editShortID string

	status      string
	statusOK    bool
	statusUntil time.Time

	width, height int
	ready         bool
	err           error
}

func newModel(backend Backend, notes []Note, width, height int) Model {
	ti := textinput.New()
	ti.Placeholder = "Title"
	ti.Prompt = ""
	ti.CharLimit = 200

	ta := textarea.New()
	ta.Placeholder = "Write markdown..."
	ta.ShowLineNumbers = false
	ta.Prompt = ""

	si := textinput.New()
	si.Placeholder = "search titles"
	si.Prompt = "/ "

	vp := viewport.New(0, 0)

	m := Model{
		backend:     backend,
		isRemote:    backend.RemoteURL() != "",
		notes:       notes,
		focus:       FocusList,
		titleInput:  ti,
		contentArea: ta,
		searchInput: si,
		contentVP:   vp,
		help:        help.New(),
		keys:        defaultKeys(),
		wrap:        true,
		width:       width,
		height:      height,
		ready:       true,
	}
	m.resizePanes()
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.WindowSize()
}

func (m *Model) visibleNotes() []Note {
	if m.filtered == nil {
		return m.notes
	}
	out := make([]Note, 0, len(m.filtered))
	for _, i := range m.filtered {
		out = append(out, m.notes[i])
	}
	return out
}

func (m *Model) currentNote() *Note {
	notes := m.visibleNotes()
	if m.cursor < 0 || m.cursor >= len(notes) {
		return nil
	}
	return &notes[m.cursor]
}

func (m *Model) applyFilter(q string) {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" {
		m.filtered = nil
		if m.cursor >= len(m.notes) {
			m.cursor = 0
		}
		return
	}
	idx := []int{}
	for i, n := range m.notes {
		if strings.Contains(strings.ToLower(n.Title), q) {
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
