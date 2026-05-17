package tui

import (
	"time"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
)

type sessionState uint8

const (
	stateList sessionState = iota
	stateContent
	stateForm
)

type Model struct {
	backend  Backend
	isRemote bool

	state sessionState
	list  listModel
	cont  contentModel
	form  formModel

	width, height int
	ready         bool
	loading       bool

	showHelp      bool
	confirmDelete bool

	status      string
	statusOK    bool
	statusUntil time.Time

	help help.Model
	keys keyMap
}

func newModel(backend Backend) Model {
	return Model{
		backend:  backend,
		isRemote: backend.RemoteURL() != "",
		state:    stateList,
		list:     newListModel(nil),
		cont:     newContentModel(),
		form:     newFormModel(),
		help:     help.New(),
		keys:     defaultKeys(),
		loading:  true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.RequestWindowSize, loadSnippetsCmd(m.backend))
}
