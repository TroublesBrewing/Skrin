package ui

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/theme"
	"github.com/lurioso/skrin/internal/version"
)

// Paper gives the note a faint grain, so reading in Skrin feels a little
// more like reading off a page than off a screen.
//
// A terminal has no texture of its own: the only thing a program can
// change is a cell's background. So the grain is painted cell by cell —
// but in runs, not one cell at a time. Every run has to repeat whatever
// styling the text under it carries, and at one run per cell a full
// screen costs some hundred kilobytes a frame, which a terminal feels
// over ssh and inside tmux. In runs of a few cells it costs a fraction of
// that, and at this size a cell is already a coarse pixel: the grain
// reads the same either way.
const (
	grainRun    = 4 // cells per run of one shade
	grainShades = 3 // how many shades the grain moves between
)

// paperOn is the grain's real state: a beta feature, so the build must
// allow beta, beta mode must be on, and its own switch too.
func (m *Model) paperOn() bool {
	return version.Beta && m.opts.Beta && m.opts.Paper
}

// shades are the grain's colours: the background itself and a step or two
// away from it. They go the way there is room to go — darker on a light
// theme, lighter on a dark one — because a near-white background has
// nowhere to lighten to, and a shade that clips is a shade you can't see.
// Worked out once per palette, not per frame.
func shades(p theme.Palette, step int) []string {
	dir := -1.0 // a light background: the room is downward
	if p.Dark {
		dir = 1.0
	}
	out := make([]string, grainShades)
	for i := range out {
		out[i] = bgSeq(nudge(p.Background, dir*float64(i*step)))
	}
	return out
}

// nudge moves a colour d steps toward white (d > 0) or black (d < 0).
func nudge(c color.Color, d float64) color.Color {
	r, g, b, _ := c.RGBA()
	at := func(v uint32) uint8 {
		f := float64(v/257) + d
		return uint8(min(max(f, 0), 255))
	}
	return color.NRGBA{at(r), at(g), at(b), 255}
}

func bgSeq(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r/257, g/257, b/257)
}

// grain lays the paper under one line, which must already be w cells
// wide. The pattern is worked out from the row and column, so it holds
// still while you scroll rather than crawling about.
func grain(line string, row, w int, sh []string) string {
	if w <= 0 || len(sh) == 0 {
		return line
	}
	var b strings.Builder
	b.Grow(len(line) + w/grainRun*24)
	for x := 0; x < w; x += grainRun {
		end := min(x+grainRun, w)
		part := ansi.Cut(line, x, end)
		b.WriteString(sh[shadeAt(x/grainRun, row)] + part)
	}
	b.WriteString("\x1b[49m")
	return b.String()
}

// shadeAt picks a run's shade. It is a cheap hash rather than a random
// number: the same cell must come out the same every frame, or the paper
// would shimmer.
func shadeAt(run, row int) int {
	h := run*2654435761 + row*40503
	h ^= h >> 13
	if h < 0 {
		h = -h
	}
	return h % grainShades
}

// onPaper lays the grain inside an already-drawn box, between its
// borders, leaving the frame and the title alone. Working on the finished
// box means box itself stays one thing that every panel shares.
func (m *Model) onPaper(box []string, w int) []string {
	if !m.paperOn() || w < 4 || len(box) < 3 {
		return box
	}
	sh := shades(m.pal, m.opts.Config.PaperStrength())
	for i := 1; i < len(box)-1; i++ {
		row := box[i]
		left, right := ansi.Cut(row, 0, 1), ansi.Cut(row, w-1, w)
		box[i] = left + grain(ansi.Cut(row, 1, w-1), i, w-2, sh) + right
	}
	return box
}
