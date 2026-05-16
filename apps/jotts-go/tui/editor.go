package tui

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

func openExternalEditor(shortID, content string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return func() tea.Msg {
			return statusMsg{text: "$EDITOR not set", ok: false}
		}
	}

	tmp, err := os.CreateTemp("", "jotts-*.md")
	if err != nil {
		return func() tea.Msg {
			return statusMsg{text: "tempfile: " + err.Error(), ok: false}
		}
	}
	path := tmp.Name()
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		_ = os.Remove(path)
		return func() tea.Msg {
			return statusMsg{text: "tempfile: " + err.Error(), ok: false}
		}
	}
	_ = tmp.Close()

	cmd := exec.Command(editor, path)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		defer os.Remove(path)
		if err != nil {
			return editorFinishedMsg{shortID: shortID, err: err}
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return editorFinishedMsg{shortID: shortID, err: rerr}
		}
		return editorFinishedMsg{shortID: shortID, content: string(b)}
	})
}
