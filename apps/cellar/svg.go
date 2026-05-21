package main

import (
	"fmt"
	"math"
	"strings"
)

func buildPentagonSVG(sweetness, acidity, tannin, alcohol, body int, size float64, showLabels bool) string {
	cx, cy := size/2.0, size/2.0
	margin := 5.0
	if showLabels {
		margin = 30.0
	}
	r := size/2.0 - margin
	scores := []int{sweetness, acidity, tannin, alcohol, body}
	labels := []string{"Sweetness", "Acidity", "Tannin", "Alcohol", "Body"}
	angles := make([]float64, 5)
	for i := range angles {
		angles[i] = (-90.0 + 72.0*float64(i)) * math.Pi / 180.0
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg viewBox="0 0 %g %g" width="100%%" xmlns="http://www.w3.org/2000/svg">`, size, size)

	for _, pct := range []float64{0.2, 0.4, 0.6, 0.8} {
		parts := make([]string, 5)
		for i, a := range angles {
			parts[i] = fmt.Sprintf("%.1f,%.1f", cx+r*pct*math.Cos(a), cy+r*pct*math.Sin(a))
		}
		fmt.Fprintf(&b, `<polygon points="%s" fill="none" stroke="white" stroke-opacity="0.12" stroke-width="0.75"/>`, strings.Join(parts, " "))
	}

	outline := make([]string, 5)
	for i, a := range angles {
		outline[i] = fmt.Sprintf("%.1f,%.1f", cx+r*math.Cos(a), cy+r*math.Sin(a))
	}
	fmt.Fprintf(&b, `<polygon points="%s" fill="none" stroke="white" stroke-opacity="0.25" stroke-width="1"/>`, strings.Join(outline, " "))

	for _, a := range angles {
		fmt.Fprintf(&b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="white" stroke-opacity="0.12" stroke-width="0.75"/>`,
			cx, cy, cx+r*math.Cos(a), cy+r*math.Sin(a))
	}

	dataPoints := make([][2]float64, 5)
	for i, s := range scores {
		d := float64(s) / 5.0 * r
		dataPoints[i] = [2]float64{cx + d*math.Cos(angles[i]), cy + d*math.Sin(angles[i])}
	}
	parts := make([]string, 5)
	for i, p := range dataPoints {
		parts[i] = fmt.Sprintf("%.1f,%.1f", p[0], p[1])
	}
	fmt.Fprintf(&b, `<polygon points="%s" fill="white" fill-opacity="0.08" stroke="white" stroke-width="1.5"/>`, strings.Join(parts, " "))

	for _, p := range dataPoints {
		fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="2.5" fill="white"/>`, p[0], p[1])
	}

	if showLabels {
		for i, label := range labels {
			a := angles[i]
			labelDist := r + 18.0
			lx := cx + labelDist*math.Cos(a)
			ly := cy + labelDist*math.Sin(a) + 3.5
			fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" fill="white" fill-opacity="0.5" font-size="9" font-family="Commit Mono, monospace" text-anchor="middle">%s</text>`, lx, ly, label)
		}
	}
	b.WriteString("</svg>")
	return b.String()
}

func buildBarsSVG(clarity, colorIntensity, aromaIntensity, noseComplexity int, width float64) string {
	barHeight := 4.0
	rowHeight := 22.0
	sectionGap := 14.0
	labelWidth := 100.0
	trackLeft := labelWidth + 4.0
	trackWidth := width - trackLeft - 10.0
	headerSize := 9.0

	type attr struct {
		Label string
		Score int
	}
	sections := []struct {
		Name  string
		Attrs []attr
	}{
		{"Appearance", []attr{{"Clarity", clarity}, {"Intensity", colorIntensity}}},
		{"Nose", []attr{{"Aroma", aromaIntensity}, {"Complexity", noseComplexity}}},
	}
	totalRows := 0
	for _, s := range sections {
		totalRows += len(s.Attrs)
	}
	totalHeight := float64(len(sections))*(headerSize+8.0) + float64(totalRows)*rowHeight + sectionGap

	var b strings.Builder
	fmt.Fprintf(&b, `<svg viewBox="0 0 %g %g" width="100%%" xmlns="http://www.w3.org/2000/svg">`, width, totalHeight)
	y := 4.0
	for si, sec := range sections {
		if si > 0 {
			y += sectionGap
		}
		fmt.Fprintf(&b, `<text x="0" y="%.1f" fill="white" fill-opacity="0.4" font-size="%g" font-family="Commit Mono, monospace" text-transform="uppercase" letter-spacing="1">%s</text>`,
			y+headerSize, headerSize, sec.Name)
		y += headerSize + 8.0
		for _, a := range sec.Attrs {
			barY := y + (rowHeight-barHeight)/2.0
			fillW := float64(a.Score) / 5.0 * trackWidth
			fmt.Fprintf(&b, `<text x="0" y="%.1f" fill="white" fill-opacity="0.5" font-size="9" font-family="Commit Mono, monospace">%s</text>`,
				y+rowHeight/2.0+3.0, a.Label)
			fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="2" fill="white" fill-opacity="0.08"/>`,
				trackLeft, barY, trackWidth, barHeight)
			if fillW > 0 {
				fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="2" fill="white" fill-opacity="0.6"/>`,
					trackLeft, barY, fillW, barHeight)
			}
			y += rowHeight
		}
	}
	b.WriteString("</svg>")
	return b.String()
}
