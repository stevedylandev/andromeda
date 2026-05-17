package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
	borderActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("214"))
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214")).
			Padding(0, 1)
	itemStyle    = lipgloss.NewStyle().Padding(0, 1)
	itemSelected = lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color("214"))
	statusOK = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)
	statusErr = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("214")).
			Padding(1, 2).
			Background(lipgloss.Color("236"))
)

func (m Model) View() string {
	if !m.ready {
		return "loading..."
	}

	listW := m.width * 30 / 100
	if listW < 24 {
		listW = 24
	}
	contentW := m.width - listW - 2
	bodyH := m.height - 2

	left := m.renderList(listW, bodyH)
	right := m.renderRight(contentW, bodyH)

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	footer := m.renderFooter()

	view := lipgloss.JoinVertical(lipgloss.Left, body, footer)

	if m.showHelp {
		view = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			modalStyle.Render(m.help.FullHelpView(m.keys.FullHelp())),
			lipgloss.WithWhitespaceChars(" "))
	}
	if m.confirmDelete {
		n := m.currentNote()
		title := ""
		if n != nil {
			title = n.Title
		}
		view = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			modalStyle.Render(fmt.Sprintf("Delete %q?\n\ny / n", title)),
			lipgloss.WithWhitespaceChars(" "))
	}
	if m.status != "" {
		st := statusOK
		if !m.statusOK {
			st = statusErr
		}
		view = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Bottom,
			modalStyle.Render(st.Render(m.status)),
			lipgloss.WithWhitespaceChars(" "))
	}
	return view
}

func (m Model) renderList(w, h int) string {
	style := borderStyle
	if m.focus == FocusList || m.focus == FocusSearch {
		style = borderActive
	}

	notes := m.visibleNotes()
	rows := make([]string, 0, len(notes)+2)
	rows = append(rows, titleStyle.Render("notes"))
	if len(notes) == 0 {
		rows = append(rows, hintStyle.Render("  (empty — press c)"))
	}
	for i, n := range notes {
		line := truncate(n.Title, w-6)
		if i == m.cursor {
			rows = append(rows, itemSelected.Render("▶ "+line))
		} else {
			rows = append(rows, itemStyle.Render("  "+line))
		}
	}

	if m.focus == FocusSearch || m.searchInput.Value() != "" {
		rows = append(rows, "", hintStyle.Render(m.searchInput.View()))
	}

	content := strings.Join(rows, "\n")
	return style.Width(w).Height(h).Render(content)
}

func (m Model) renderRight(w, h int) string {
	switch m.focus {
	case FocusCreateTitle, FocusCreateContent, FocusEditTitle, FocusEditContent:
		return m.renderForm(w, h)
	}
	return m.renderContent(w, h)
}

func (m Model) renderContent(w, h int) string {
	style := borderStyle
	if m.focus == FocusContent {
		style = borderActive
	}
	header := "preview"
	n := m.currentNote()
	if n != nil {
		header = n.Title
	}
	body := m.contentVP.View()
	inner := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(header), body)
	return style.Width(w).Height(h).Render(inner)
}

func (m Model) renderForm(w, h int) string {
	header := "new note"
	if m.editShortID != "" {
		header = "edit"
	}
	title := m.titleInput.View()
	if m.focus == FocusCreateTitle || m.focus == FocusEditTitle {
		title = borderActive.Render(title)
	} else {
		title = borderStyle.Render(title)
	}

	body := m.contentArea.View()
	if m.focus == FocusCreateContent || m.focus == FocusEditContent {
		body = borderActive.Render(body)
	} else {
		body = borderStyle.Render(body)
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(header), title, body)
	return borderStyle.Width(w).Height(h).Render(inner)
}

func (m Model) renderFooter() string {
	mode := "local"
	if m.isRemote {
		mode = "remote " + m.backend.RemoteURL()
	}
	help := m.help.ShortHelpView(m.keys.ShortHelp())
	return hintStyle.Render(fmt.Sprintf("[%s] %s", mode, help))
}

func truncate(s string, n int) string {
	if n < 1 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return s[:n-1] + "…"
}
