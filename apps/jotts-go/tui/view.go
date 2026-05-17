package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("8"))
	borderActive = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("3"))
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("3")).
			Padding(0, 1)
	itemStyle    = lipgloss.NewStyle().Padding(0, 1)
	itemSelected = lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color("3"))
	statusOK = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true)
	statusErr = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true)
	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("3")).
			Padding(1, 2)
)

func (m Model) View() string {
	listW, contentW := splitWidths(m.width)
	bodyH := splitBodyHeight(m.height)

	left := m.renderList(listW, bodyH)
	right := m.renderRight(contentW, bodyH)

	view := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

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
		line := truncate(n.Title, maxInt(w-paneFrameWidth()-4, 1))
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
	return style.
		Width(maxInt(w-paneFrameWidth(), 1)).
		Height(maxInt(h-paneFrameHeight(), 1)).
		Render(content)
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
	return style.
		Width(maxInt(w-paneFrameWidth(), 1)).
		Height(maxInt(h-paneFrameHeight(), 1)).
		Render(inner)
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
	return borderStyle.
		Width(maxInt(w-paneFrameWidth(), 1)).
		Height(maxInt(h-paneFrameHeight(), 1)).
		Render(inner)
}

func splitWidths(total int) (int, int) {
	if total < 44 {
		return total / 2, total - (total / 2)
	}
	list := total * 30 / 100
	if list < 24 {
		list = 24
	}
	if total-list < 20 {
		list = total - 20
	}
	if list < 1 {
		list = 1
	}
	return list, total - list
}

func splitBodyHeight(total int) int {
	if total < 3 {
		return 3
	}
	return total
}

func paneFrameWidth() int {
	return borderStyle.GetHorizontalFrameSize()
}

func paneFrameHeight() int {
	return borderStyle.GetVerticalFrameSize()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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
