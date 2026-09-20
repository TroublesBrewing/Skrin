// Package habiton draws Habiton, the creature in the habits view. He is
// wordless: posture and eyes are the whole of what he says. He never
// scolds and never congratulates in words — a missed day makes him sleepy,
// not disappointed, because shame is a poor way to build a habit and a
// worse thing to meet in a terminal.
//
// He is pixel art in half-blocks, the same technique as internal/logo, in
// the theme's own colours.
package habiton

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/lurioso/skrin/internal/habit"
	"github.com/lurioso/skrin/internal/theme"
)

// Mood is what Habiton is doing. It is read from the habits themselves
// and never stored: there is no score file, nothing to sync, nothing to
// lose. Delete Skrin and your checkboxes are all that ever existed.
type Mood int

const (
	// Asleep: nothing ticked today. The day hasn't started, so neither
	// has he.
	Asleep Mood = iota
	// Awake: something ticked today, but not all of it.
	Awake
	// Bright: everything ticked today. Arms up.
	Bright
	// Sleepy: little has been ticked over the last few recorded days. He
	// is low on energy, not cross — the difference is the whole point.
	Sleepy
)

// Read works out the mood from the grid, looking at today and at the few
// days behind it. Days with no note at all are not counted against you:
// unrecorded is not the same as undone.
func Read(g habit.Grid, todayPath string) Mood {
	if len(g.Names) == 0 {
		return Asleep
	}
	done := 0
	for _, n := range g.Names {
		if g.Done[todayPath][n] {
			done++
		}
	}
	switch {
	case done == len(g.Names):
		return Bright
	case recentRate(g, 3) < 0.34:
		return Sleepy
	case done > 0:
		return Awake
	}
	return Asleep
}

// recentRate is the share of ticks over the last n recorded days, today
// included. Days without a note are skipped rather than counted as zero.
func recentRate(g habit.Grid, n int) float64 {
	ticks, slots, seen := 0, 0, 0
	for i := len(g.Days) - 1; i >= 0 && seen < n; i-- {
		day := g.Days[i]
		if day.Path == "" {
			continue
		}
		seen++
		for _, name := range g.Names {
			slots++
			if g.Done[day.Path][name] {
				ticks++
			}
		}
	}
	if slots == 0 {
		return 1 // nothing recorded yet: no reason to look tired
	}
	return float64(ticks) / float64(slots)
}

// Pixels: '.' clear, 'A' the accent, 'D' the accent darkened (outline and
// shadow), 'L' the foreground (eyes). Twelve by twelve, so six cell rows.
var art = map[Mood][]string{
	Asleep: {
		"............",
		"............",
		"...DDDDDD...",
		"..DAAAAAAD..",
		"..DALLAALD..",
		".DAAAAAAAAD.",
		".DAAAAAAAAD.",
		"..DAAAAAAD..",
		"...DDDDDD...",
		"............",
		"............",
		"............",
	},
	Awake: {
		"............",
		"...DDDDDD...",
		"..DAAAAAAD..",
		".DAAAAAAAAD.",
		".DALAAAALAD.",
		".DAAAAAAAAD.",
		".DAAAAAAAAD.",
		"..DAAAAAAD..",
		"...DDDDDD...",
		"....D..D....",
		"...DD..DD...",
		"............",
	},
	Bright: {
		".A........A.",
		".AA.DDDD.AA.",
		"..ADAAAADA..",
		".DAAAAAAAAD.",
		".DALAAAALAD.",
		".DAAAAAAAAD.",
		".DAALAALAAD.",
		"..DAAAAAAD..",
		"...DDDDDD...",
		"....D..D....",
		"...DD..DD...",
		"............",
	},
	Sleepy: {
		"............",
		"............",
		"...DDDDDD...",
		"..DAAAAAAD..",
		"..DALLAALD..",
		".DAAAAAAAAD.",
		".DAAAAAAAAD.",
		".DAAAAAAAAD.",
		"..DDDDDDDD..",
		"...D....D...",
		"..DD....DD..",
		"............",
	},
}

// Draw returns Habiton's rows and his width in cells. frame alternates 0
// and 1: he blinks, which is the whole of the animation. Anything busier
// would pull the eye away from the note you are reading.
func Draw(mood Mood, frame int, p theme.Palette) ([]string, int) {
	bm := art[mood]
	if frame%2 == 1 && (mood == Awake || mood == Bright) {
		bm = blink(bm)
	}
	ink := map[byte]color.Color{
		'A': p.Accent,
		'D': mix(p.Accent, p.Background, 0.45),
		'L': p.Foreground,
	}
	w, h := len(bm[0]), len(bm)
	lines := make([]string, h/2)
	for row := range lines {
		var b strings.Builder
		for x := 0; x < w; x++ {
			b.WriteString(cell(ink[bm[2*row][x]], ink[bm[2*row+1][x]]))
		}
		lines[row] = b.String()
	}
	return lines, w
}

// blink shuts the eyes by painting them in the body's own colour, so they
// disappear for a moment. At one pixel an eye, that reads as a blink far
// better than any drawn lid would. The moods whose eyes are already shut
// don't blink at all: Draw never calls this for them.
func blink(bm []string) []string {
	out := make([]string, len(bm))
	for i, row := range bm {
		out[i] = strings.ReplaceAll(row, "L", "A")
	}
	return out
}

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

func mix(a, b color.Color, t float64) color.Color {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	l := func(x, y uint32) uint8 { return uint8((float64(x)*(1-t) + float64(y)*t) / 257) }
	return color.NRGBA{l(ar, br), l(ag, bg), l(ab, bb), 255}
}
