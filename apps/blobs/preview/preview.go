// Package preview renders images inline in supported terminals.
//
// Detection cascade: kitty/ghostty graphics → iTerm2 inline images →
// chafa fallback → metadata-only text.
package preview

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Protocol int

const (
	ProtoNone Protocol = iota
	ProtoKitty
	ProtoITerm
	ProtoChafa
)

// Detect inspects env vars to pick a preview protocol.
// Order: explicit BLOBS_PREVIEW override → kitty/ghostty → iTerm2/wezterm → chafa binary.
func Detect() Protocol {
	if v := strings.ToLower(os.Getenv("BLOBS_PREVIEW")); v != "" {
		switch v {
		case "kitty":
			return ProtoKitty
		case "iterm", "iterm2":
			return ProtoITerm
		case "chafa":
			if _, err := exec.LookPath("chafa"); err == nil {
				return ProtoChafa
			}
		case "none", "off":
			return ProtoNone
		}
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" || os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
		return ProtoKitty
	}
	switch strings.ToLower(os.Getenv("TERM_PROGRAM")) {
	case "iterm.app", "wezterm":
		return ProtoITerm
	}
	if strings.HasPrefix(os.Getenv("TERM"), "xterm-kitty") {
		return ProtoKitty
	}
	if _, err := exec.LookPath("chafa"); err == nil {
		return ProtoChafa
	}
	return ProtoNone
}

// Render returns a string to print into a TUI viewport / pane.
// w, h are character cell dimensions of the target pane.
func Render(p Protocol, img []byte, w, h int) (string, error) {
	switch p {
	case ProtoKitty:
		return kittyEscape(img), nil
	case ProtoITerm:
		return itermEscape(img), nil
	case ProtoChafa:
		return chafaRender(img, w, h)
	default:
		return "", fmt.Errorf("no preview protocol available")
	}
}

func kittyEscape(img []byte) string {
	enc := base64.StdEncoding.EncodeToString(img)
	const chunk = 4096
	var b strings.Builder
	for i := 0; i < len(enc); i += chunk {
		end := i + chunk
		if end > len(enc) {
			end = len(enc)
		}
		more := 1
		if end == len(enc) {
			more = 0
		}
		if i == 0 {
			fmt.Fprintf(&b, "\x1b_Ga=T,f=100,m=%d;%s\x1b\\", more, enc[i:end])
		} else {
			fmt.Fprintf(&b, "\x1b_Gm=%d;%s\x1b\\", more, enc[i:end])
		}
	}
	return b.String()
}

func itermEscape(img []byte) string {
	enc := base64.StdEncoding.EncodeToString(img)
	return fmt.Sprintf("\x1b]1337;File=inline=1;preserveAspectRatio=1:%s\x07", enc)
}

func chafaRender(img []byte, w, h int) (string, error) {
	if w <= 0 {
		w = 40
	}
	if h <= 0 {
		h = 20
	}
	size := fmt.Sprintf("%dx%d", w, h)
	cmd := exec.Command("chafa", "--size", size, "--format", "symbols", "-")
	cmd.Stdin = bytes.NewReader(img)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}
