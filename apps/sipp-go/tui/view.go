package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
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

func (m Model) View() tea.View {
	if !m.ready {
		return tea.View{Content: "loading...", AltScreen: true}
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

	base := lipgloss.JoinVertical(lipgloss.Left, body, footer)

	var overlays []*lipgloss.Layer
	if m.showHelp {
		overlays = append(overlays, centerLayer(m.width, m.height,
			modalStyle.Render(m.help.FullHelpView(m.keys.FullHelp())), 1))
	}
	if m.confirmDelete {
		s := m.current()
		name := ""
		if s != nil {
			name = s.Name
		}
		overlays = append(overlays, centerLayer(m.width, m.height,
			modalStyle.Render(fmt.Sprintf("Delete %q?\n\ny / n", name)), 2))
	}
	if m.status != "" {
		st := statusOK
		if !m.statusOK {
			st = statusErr
		}
		overlays = append(overlays, bottomCenterLayer(m.width, m.height,
			modalStyle.Render(st.Render(m.status)), 3))
	}

	content := base
	if len(overlays) > 0 {
		layers := append([]*lipgloss.Layer{lipgloss.NewLayer(base)}, overlays...)
		canvas := lipgloss.NewCanvas(m.width, m.height)
		canvas.Compose(lipgloss.NewCompositor(layers...))
		content = canvas.Render()
	}

	return tea.View{Content: content, AltScreen: true}
}

func centerLayer(w, h int, content string, z int) *lipgloss.Layer {
	cw, ch := lipgloss.Width(content), lipgloss.Height(content)
	x := (w - cw) / 2
	y := (h - ch) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return lipgloss.NewLayer(content).X(x).Y(y).Z(z)
}

func bottomCenterLayer(w, h int, content string, z int) *lipgloss.Layer {
	cw, ch := lipgloss.Width(content), lipgloss.Height(content)
	x := (w - cw) / 2
	y := h - ch - 1
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return lipgloss.NewLayer(content).X(x).Y(y).Z(z)
}

func (m Model) renderList(w, h int) string {
	style := borderStyle
	if m.focus == FocusList || m.focus == FocusSearch {
		style = borderActive
	}

	list := m.visible()
	rows := make([]string, 0, len(list)+2)
	rows = append(rows, titleStyle.Render("snippets"))
	if len(list) == 0 {
		rows = append(rows, hintStyle.Render("  (empty — press c)"))
	}
	for i, s := range list {
		label := s.Name
		if label == "" {
			label = s.ShortID
		}
		line := truncate(label, w-6)
		id := dimStyle.Render(" " + s.ShortID)
		if i == m.cursor {
			rows = append(rows, itemSelected.Render("▶ "+line)+id)
		} else {
			rows = append(rows, itemStyle.Render("  "+line)+id)
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
	case FocusCreateName, FocusCreateContent, FocusEditName, FocusEditContent:
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
	if m.wrapContent {
		header += " (wrap)"
	}
	s := m.current()
	if s != nil {
		header = s.Name
		if header == "" {
			header = s.ShortID
		}
	}
	body := m.contentVP.View()
	inner := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(header), body)
	return style.Width(w).Height(h).Render(inner)
}

func (m Model) renderForm(w, h int) string {
	header := "new snippet"
	if m.editShortID != "" {
		header = "edit"
	}
	if m.wrapContent {
		header += " (wrap)"
	}
	name := m.nameInput.View()
	if m.focus == FocusCreateName || m.focus == FocusEditName {
		name = borderActive.Render(name)
	} else {
		name = borderStyle.Render(name)
	}

	body := m.contentArea.View()
	if m.focus == FocusCreateContent || m.focus == FocusEditContent {
		body = borderActive.Render(body)
	} else {
		body = borderStyle.Render(body)
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(header), name, body)
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
