package tui

import (
	"fmt"
	"os/exec"
	"runtime"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
)

// CopyToClipboardCmd copies text to the OS clipboard and emits a StatusMsg.
func CopyToClipboardCmd(text, okStatus string) tea.Cmd {
	return func() tea.Msg {
		if err := clipboard.WriteAll(text); err != nil {
			return StatusMsg{Text: "clipboard: " + err.Error()}
		}
		return StatusMsg{Text: okStatus, OK: true}
	}
}

// OpenURLCmd opens url in the default browser and emits a StatusMsg.
func OpenURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if err := openURL(url); err != nil {
			return StatusMsg{Text: "open: " + err.Error()}
		}
		return StatusMsg{Text: "opened " + url, OK: true}
	}
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
