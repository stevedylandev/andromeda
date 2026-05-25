package main

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	sharedtui "github.com/stevedylandev/andromeda/pkg/tui"
)

var (
	paneBorder       = sharedtui.Border(lipgloss.RoundedBorder())
	paneBorderActive = sharedtui.BorderActive(lipgloss.RoundedBorder())
	tuiTitleStyle    = sharedtui.TitleStyle
	tuiOKStyle       = sharedtui.StatusOKStyle
	tuiErrStyle      = sharedtui.StatusErrStyle
	tuiHintStyle     = sharedtui.HintStyle
	tuiModalStyle    = sharedtui.ModalStyle
	tuiStatusModal   = sharedtui.StatusModalStyle
)

func paneFrameW() int { return paneBorder.GetHorizontalFrameSize() }
func paneFrameH() int { return paneBorder.GetVerticalFrameSize() }

func (m tuiModel) View() tea.View {
	if !m.ready {
		return tea.View{Content: "loading...", AltScreen: true}
	}

	var body string
	switch m.state {
	case stateBuckets:
		body = paneBorderActive.Width(m.width - paneFrameW()).Height(m.height - 2 - paneFrameH()).Render(m.bucketsList.View())
	case stateBrowse:
		body = m.renderBrowse()
	}

	footer := m.renderFooter()
	base := lipgloss.JoinVertical(lipgloss.Left, body, footer)

	var overlays []*lipgloss.Layer
	if m.showHelp {
		overlays = append(overlays, centerLay(m.width, m.height,
			tuiModalStyle.Render(m.help.FullHelpView(m.keys.FullHelp())), 1))
	}
	if m.confirmDelete {
		name := ""
		if f, ok := m.selectedFile(); ok {
			name = f.Name
		}
		overlays = append(overlays, centerLay(m.width, m.height,
			tuiModalStyle.Render(fmt.Sprintf("Delete %q?\n\ny / n", name)), 2))
	}
	if m.uploadPrompt == uploadPromptActive {
		dest := m.currentBucket + ":/" + m.currentPrefix
		overlays = append(overlays, centerLay(m.width, m.height,
			tuiModalStyle.Render("Upload to "+dest+"\n\n"+m.uploadInput.View()+"\n\nenter=upload  esc=cancel"), 2))
	}
	if m.status != "" {
		st := tuiOKStyle
		if !m.statusOK {
			st = tuiErrStyle
		}
		overlays = append(overlays, bottomLay(m.width, m.height,
			tuiStatusModal.Render(st.Render(m.status)), 3))
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

func (m tuiModel) renderBrowse() string {
	bodyH := m.height - 2
	if !m.showPreview {
		return paneBorderActive.Width(m.width - paneFrameW()).Height(bodyH - paneFrameH()).Render(m.browseList.View())
	}
	listW := m.width / 2
	previewW := m.width - listW
	left := paneBorderActive.Width(listW - paneFrameW()).Height(bodyH - paneFrameH()).Render(m.browseList.View())
	header := tuiTitleStyle.Render("preview")
	previewBody := m.preview.View()
	inner := lipgloss.JoinVertical(lipgloss.Left, header, previewBody)
	right := paneBorder.Width(previewW - paneFrameW()).Height(bodyH - paneFrameH()).Render(inner)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m tuiModel) renderFooter() string {
	label := m.currentBucket
	if m.state == stateBrowse {
		label += ":/" + m.currentPrefix
	} else {
		label = "buckets"
	}
	help := m.help.ShortHelpView(m.keys.ShortHelp())
	return tuiHintStyle.Render(fmt.Sprintf("[%s] %s", label, help))
}

func centerLay(w, h int, content string, z int) *lipgloss.Layer {
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

func bottomLay(w, h int, content string, z int) *lipgloss.Layer {
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
