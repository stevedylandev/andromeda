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

	showHelp      bool
	confirmDelete bool

	status      string
	statusOK    bool
	statusUntil time.Time

	help help.Model
	keys keyMap
}

func newModel(backend Backend, notes []Note, width, height int) Model {
	m := Model{
		backend:  backend,
		isRemote: backend.RemoteURL() != "",
		state:    stateList,
		list:     newListModel(notes),
		cont:     newContentModel(),
		form:     newFormModel(),
		help:     help.New(),
		keys:     defaultKeys(),
		width:    width,
		height:   height,
		ready:    true,
	}
	m.applyLayout()
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.RequestWindowSize
}
