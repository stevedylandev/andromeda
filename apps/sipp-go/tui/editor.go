package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	tea "charm.land/bubbletea/v2"
)

func openExternalEditor(shortID, name, content string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return func() tea.Msg {
			return statusMsg{text: "$EDITOR not set", ok: false}
		}
	}

	base := name
	if base == "" {
		base = "snippet.txt"
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("sipp-%s-%s", shortID, filepath.Base(base)))
	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return func() tea.Msg {
			return statusMsg{text: "tempfile: " + err.Error(), ok: false}
		}
	}

	cmd := exec.Command(editor, tmp)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		defer os.Remove(tmp)
		if err != nil {
			return editorFinishedMsg{shortID: shortID, err: err}
		}
		b, rerr := os.ReadFile(tmp)
		if rerr != nil {
			return editorFinishedMsg{shortID: shortID, err: rerr}
		}
		return editorFinishedMsg{shortID: shortID, content: string(b)}
	})
}

func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}
	return cmd.Start()
}
