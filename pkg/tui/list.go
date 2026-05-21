package tui

import (
	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
)

// ListIDStyle dims a trailing short-id segment in list item titles.
var ListIDStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

// ANSIListDelegate returns the andromeda default list delegate (no description,
// no spacing, accent color "3").
func ANSIListDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.SetSpacing(0)
	d.Styles.NormalTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Padding(0, 0, 0, 2)
	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")).
		Bold(true).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("3")).
		Padding(0, 0, 0, 1)
	d.Styles.DimmedTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Padding(0, 0, 0, 2)
	d.Styles.FilterMatch = lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color("3"))
	return d
}

// ANSIListStyles returns the andromeda default list.Styles.
func ANSIListStyles() list.Styles {
	s := list.DefaultStyles(true)
	s.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")).Padding(0, 1)
	s.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 0)
	s.NoItems = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Padding(0, 0, 0, 2)
	s.DefaultFilterCharacterMatch = lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color("3"))
	return s
}
