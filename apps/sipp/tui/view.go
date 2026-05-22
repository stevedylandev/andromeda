package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	sharedtui "github.com/stevedylandev/andromeda/pkg/tui"
)

var (
	borderStyle    = sharedtui.Border(lipgloss.RoundedBorder())
	borderActive   = sharedtui.BorderActive(lipgloss.RoundedBorder())
	titleStyle     = sharedtui.TitleStyle
	statusOKStyle  = sharedtui.StatusOKStyle
	statusErrStyle = sharedtui.StatusErrStyle
	hintStyle      = sharedtui.HintStyle
	modalStyle       = sharedtui.ModalStyle
	statusModalStyle = sharedtui.StatusModalStyle
)

func (m Model) View() tea.View {
	if !m.ready {
		return tea.View{Content: "loading...", AltScreen: true}
	}

	listW := m.width * 30 / 100
	if listW < 24 {
		listW = 24
	}
	contentW := m.width - listW
	bodyH := m.height - 1

	left := m.renderListPane(listW, bodyH)
	right := m.renderRightPane(contentW, bodyH)

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	footer := m.renderFooter()
	base := lipgloss.JoinVertical(lipgloss.Left, body, footer)

	var overlays []*lipgloss.Layer
	if m.showHelp {
		overlays = append(overlays, centerLayer(m.width, m.height,
			modalStyle.Render(m.help.FullHelpView(m.keys.FullHelp())), 1))
	}
	if m.confirmDelete {
		name := ""
		if s, ok := m.list.Selected(); ok {
			name = s.Name
			if name == "" {
				name = s.ShortID
			}
		}
		overlays = append(overlays, centerLayer(m.width, m.height,
			modalStyle.Render(fmt.Sprintf("Delete %q?\n\ny / n", name)), 2))
	}
	if m.status != "" {
		st := statusOKStyle
		if !m.statusOK {
			st = statusErrStyle
		}
		overlays = append(overlays, bottomCenterLayer(m.width, m.height,
			statusModalStyle.Render(st.Render(m.status)), 3))
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
	return style.Width(w).Height(h).Render(m.list.View())
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
	if m.cont.Wrap() {
		header += " (wrap)"
	}
	if t := m.cont.Header(); t != "" {
		header = t
		if m.cont.Wrap() {
			header += " (wrap)"
		}
	}
	inner := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(header), m.cont.View())
	return style.Width(w).Height(h).Render(inner)
}

func (m Model) renderForm(w, h int) string {
	header := "new snippet"
	if !m.form.IsCreate() {
		header = "edit"
	}

	nameField := m.form.name.View()
	if m.form.ActiveField() == formFieldName {
		nameField = borderActive.Render(nameField)
	} else {
		nameField = borderStyle.Render(nameField)
	}

	body := m.form.content.View()
	if m.form.ActiveField() == formFieldContent {
		body = borderActive.Render(body)
	} else {
		body = borderStyle.Render(body)
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(header), nameField, body)
	return borderActive.Width(w).Height(h).Render(inner)
}

func (m Model) renderFooter() string {
	mode := "local"
	if m.isRemote {
		mode = "remote " + m.backend.RemoteURL()
	}
	help := m.help.ShortHelpView(m.keys.ShortHelp())
	return hintStyle.Render(fmt.Sprintf("[%s] %s", mode, help))
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

func paneFrameWidth() int  { return borderStyle.GetHorizontalFrameSize() }
func paneFrameHeight() int { return borderStyle.GetVerticalFrameSize() }
