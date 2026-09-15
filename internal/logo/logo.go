// Package logo draws Skrin's logo, a small chest, in the terminal. It is
// pixel art painted in the theme's colours and drawn with half-block
// characters: each cell stacks two pixels (▀ / ▄), so the pixels come out
// square and sharp in any terminal.
package logo

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/lurioso/skrin/internal/theme"
)

// The same chest at two sizes. Pixels: '.' clear, 'A' the theme's accent,
// 'D' the accent darkened toward the background, 'L' the text colour (the
// clasp). Heights are even, so pixel rows pair up into cells.
var (
	small = []string{
		".DDDDDD.",
		"DAAAAAAD",
		"DDDLLDDD",
		"DAALLAAD",
		"DAAAAAAD",
		"DDDDDDDD",
	}
	large = []string{
		"....DDDDDDDDDDDD....",
		"..DDDAAAAAAAAAADDD..",
		".DAADAAAAAAAAAADAAD.",
		".DAADAAAAAAAAAADAAD.",
		"DDDDDDDDDDDDDDDDDDDD",
		"DAAADAAALLLLAAADAAAD",
		"DAAADAAALDDLAAADAAAD",
		"DAAADAAALDDLAAADAAAD",
		"DAAADAAALLLLAAADAAAD",
		"DAAADAAAAAAAAAADAAAD",
		"DAAADAAAAAAAAAADAAAD",
		"DAAADAAAAAAAAAADAAAD",
		"DDDDDDDDDDDDDDDDDDDD",
		".DD..............DD.",
	}
)

// Header is the small chest for the header: three rows, and its width.
func Header(p theme.Palette) ([]string, int) { return draw(small, 1, p) }

// Splash is the large chest, each pixel scale×scale, and its width. At
// scale 1 it is seven rows tall.
func Splash(p theme.Palette, scale int) ([]string, int) { return draw(large, scale, p) }

// SplashSize is the width and height in cells of Splash at scale.
func SplashSize(scale int) (int, int) { return len(large[0]) * scale, len(large) * scale / 2 }

func draw(bm []string, scale int, p theme.Palette) ([]string, int) {
	ink := map[byte]color.Color{'A': p.Accent, 'D': mix(p.Accent, p.Background, 0.45), 'L': p.Foreground}
	px := func(x, y int) color.Color { return ink[bm[y/scale][x/scale]] } // nil when clear
	w, h := len(bm[0])*scale, len(bm)*scale
	lines := make([]string, h/2)
	for row := range lines {
		var b strings.Builder
		for x := 0; x < w; x++ {
			b.WriteString(cell(px(x, 2*row), px(x, 2*row+1)))
		}
		lines[row] = b.String()
	}
	return lines, w
}

// cell draws two stacked pixels; a clear one shows the terminal background.
func cell(top, bot color.Color) string {
	st := lipgloss.NewStyle()
	switch {
	case top != nil && bot != nil:
		return st.Foreground(top).Background(bot).Render("▀")
	case top != nil:
		return st.Foreground(top).Render("▀")
	case bot != nil:
		return st.Foreground(bot).Render("▄")
	}
	return " "
}

// mix blends a toward b by t.
func mix(a, b color.Color, t float64) color.Color {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	l := func(x, y uint32) uint8 { return uint8((float64(x)*(1-t) + float64(y)*t) / 257) }
	return color.NRGBA{l(ar, br), l(ag, bg), l(ab, bb), 255}
}
