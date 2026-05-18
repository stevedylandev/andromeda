package tui

import "charm.land/lipgloss/v2"

// Standard styles shared across andromeda TUIs.
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("3")).
			Padding(0, 1)
	StatusOKStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true)
	StatusErrStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true)
	HintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
	ModalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("3")).
			Padding(1, 2)
	StatusModalStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("3")).
				Padding(0, 1)
)

// Border returns the inactive pane border style using the given border.
func Border(b lipgloss.Border) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(b).
		BorderForeground(lipgloss.Color("8"))
}

// BorderActive returns the focused pane border style using the given border.
func BorderActive(b lipgloss.Border) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(b).
		BorderForeground(lipgloss.Color("3"))
}
