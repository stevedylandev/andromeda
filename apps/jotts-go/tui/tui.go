package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func Run(opts Options) error {
	backend, err := ResolveBackend(opts)
	if err != nil {
		return err
	}
	defer backend.Close()

	p := tea.NewProgram(newModel(backend), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
