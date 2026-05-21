package tui

import (
	tea "charm.land/bubbletea/v2"
)

func Run(opts Options) error {
	backend, err := ResolveBackend(opts)
	if err != nil {
		return err
	}
	defer backend.Close()

	p := tea.NewProgram(newModel(backend))
	_, err = p.Run()
	return err
}
