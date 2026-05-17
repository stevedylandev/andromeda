package tui

import (
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

// SpawnEditor opens the user's $EDITOR on a temp file seeded with content.
// pattern is the os.CreateTemp pattern (e.g. "jotts-*.md"); empty falls back
// to a generic ".txt" pattern. tag is echoed in the resulting message so
// callers can correlate the result with the record being edited.
func SpawnEditor(tag, pattern, content string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return func() tea.Msg {
			return StatusMsg{Text: "$EDITOR not set"}
		}
	}
	if pattern == "" {
		pattern = "editor-*.txt"
	}

	tmp, err := os.CreateTemp("", pattern)
	if err != nil {
		return func() tea.Msg {
			return StatusMsg{Text: "tempfile: " + err.Error()}
		}
	}
	path := tmp.Name()
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		_ = os.Remove(path)
		return func() tea.Msg {
			return StatusMsg{Text: "tempfile: " + err.Error()}
		}
	}
	_ = tmp.Close()

	cmd := exec.Command(editor, path)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		defer os.Remove(path)
		if err != nil {
			return EditorFinishedMsg{Tag: tag, Err: err}
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return EditorFinishedMsg{Tag: tag, Err: rerr}
		}
		return EditorFinishedMsg{Tag: tag, Content: string(b)}
	})
}
