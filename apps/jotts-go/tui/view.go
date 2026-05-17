package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	statusOKStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true)
	statusErrStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true)
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("3")).
			Padding(1, 2)
)

func (m Model) View() tea.View {
	listW, contentW := splitWidths(m.width)
	bodyH := splitBodyHeight(m.height)

	left := m.renderListPane(listW, bodyH)
	right := m.renderRightPane(contentW, bodyH)

	base := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	var overlays []*lipgloss.Layer
	if m.showHelp {
		overlays = append(overlays, centerLayer(m.width, m.height,
			modalStyle.Render(m.help.FullHelpView(m.keys.FullHelp())), 1))
	}
	if m.confirmDelete {
		title := ""
		if n, ok := m.list.Selected(); ok {
			title = n.Title
		}
		overlays = append(overlays, centerLayer(m.width, m.height,
			modalStyle.Render(fmt.Sprintf("Delete %q?\n\ny / n", title)), 2))
	}
	if m.status != "" {
		st := statusOKStyle
		if !m.statusOK {
			st = statusErrStyle
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

func (m Model) renderListPane(w, h int) string {
	style := borderStyle
	if m.state == stateList {
		style = borderActive
	}
	return style.
		Width(max(w-paneFrameWidth(), 1)).
		Height(max(h-paneFrameHeight(), 1)).
		Render(m.list.View())
}

func (m Model) renderRightPane(w, h int) string {
	if m.state == stateForm {
		return m.renderForm(w, h)
	}
	return m.renderContent(w, h)
}

func (m Model) renderContent(w, h int) string {
	style := borderStyle
	if m.state == stateContent {
		style = borderActive
	}
	header := "preview"
	if t := m.cont.Title(); t != "" {
		header = t
	}
	inner := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(header), m.cont.View())
	return style.
		Width(max(w-paneFrameWidth(), 1)).
		Height(max(h-paneFrameHeight(), 1)).
		Render(inner)
}

func (m Model) renderForm(w, h int) string {
	header := "new note"
	if !m.form.IsCreate() {
		header = "edit"
	}

	titleField := m.form.title.View()
	if m.form.ActiveField() == formFieldTitle {
		titleField = borderActive.Render(titleField)
	} else {
		titleField = borderStyle.Render(titleField)
	}

	body := m.form.content.View()
	if m.form.ActiveField() == formFieldContent {
		body = borderActive.Render(body)
	} else {
		body = borderStyle.Render(body)
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(header), titleField, body)
	return borderActive.
		Width(max(w-paneFrameWidth(), 1)).
		Height(max(h-paneFrameHeight(), 1)).
		Render(inner)
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

func paneFrameWidth() int  { return borderStyle.GetHorizontalFrameSize() }
func paneFrameHeight() int { return borderStyle.GetVerticalFrameSize() }
