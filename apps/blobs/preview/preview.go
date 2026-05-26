// Package preview renders images as half-block ANSI text suitable for
// embedding in a Bubble Tea viewport. Uses charmbracelet/x/mosaic.
package preview

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/charmbracelet/x/mosaic"
	_ "golang.org/x/image/webp"
)

// Protocol is kept for API compatibility with the rest of the TUI.
// Only two states matter now: available or not.
type Protocol int

const (
	ProtoNone Protocol = iota
	ProtoMosaic
)

// Detect always returns ProtoMosaic — mosaic is pure Go and has no
// terminal/runtime requirements beyond truecolor support.
func Detect() Protocol { return ProtoMosaic }

// Render decodes img and returns a half-block ANSI rendering sized to
// cols x rows character cells.
func Render(p Protocol, img []byte, cols, rows int) (string, error) {
	if p != ProtoMosaic {
		return "", fmt.Errorf("preview disabled")
	}
	if cols <= 0 {
		cols = 40
	}
	if rows <= 0 {
		rows = 20
	}
	decoded, _, err := image.Decode(bytes.NewReader(img))
	if err != nil {
		return "", err
	}
	fitCols, fitRows := fitAspect(decoded.Bounds().Dx(), decoded.Bounds().Dy(), cols, rows)
	// mosaic's Width/Height are pixel dims; it emits one cell per 2 pixels.
	m := mosaic.New().
		Width(fitCols).
		Height(fitRows).
		Symbol(mosaic.All).
		Dither(true)
	return m.Render(decoded), nil
}

// fitAspect returns the largest (cols, rows) inside the maxCols x maxRows
// box that preserves the image's aspect ratio. Assumes terminal cells are
// roughly 1:2 (width:height), so one row covers about 2 image-units of
// vertical space per 1 image-unit horizontal per column.
func fitAspect(imgW, imgH, maxCols, maxRows int) (int, int) {
	if imgW <= 0 || imgH <= 0 {
		return maxCols, maxRows
	}
	const cellAspect = 2.0 // cell height / cell width
	imgAspect := float64(imgW) / float64(imgH)
	// rows of "image pixels" per cell column to keep aspect:
	// cols/rows = imgAspect * cellAspect
	cols := maxCols
	rows := int(float64(cols) / imgAspect / cellAspect)
	if rows > maxRows {
		rows = maxRows
		cols = int(float64(rows) * imgAspect * cellAspect)
	}
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return cols, rows
}
