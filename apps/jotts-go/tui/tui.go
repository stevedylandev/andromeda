package tui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

func Run(opts Options) error {
	backend, err := ResolveBackend(opts)
	if err != nil {
		return err
	}
	defer backend.Close()

	notes, err := backend.List()
	if err != nil {
		return err
	}

	width, height := 100, 28
	if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 && h > 0 {
		width, height = w, h
	}

	p := tea.NewProgram(newModel(backend, notes, width, height), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
