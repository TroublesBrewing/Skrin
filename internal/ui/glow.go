package ui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lurioso/skrin/internal/version"
)

// Glow is a writing heatmap: the last year of the vault as a grid of small
// squares, one per day, green for the days you wrote and dim for the rest.
// It reads only what the index already knows — every note's modification
// time — so it costs a walk over notes in memory, never a disk pass, and it
// changes nothing. It is the shape of your practice at a glance, which is
// exactly what a single note can't show you.
//
// There is no key: it is a look, not a move, and lives in Ctrl+P beside
// Outgoing links. Off, the row isn't even in the palette.

// glowDay is one day of the heatmap.
type glowDay struct {
	date  time.Time
	notes []string // the notes last written that day
	col   int      // its week column, Monday-first, oldest leftmost
}

// glowView is the open panel.
type glowView struct {
	days []glowDay   // oldest → newest, one per day, contiguous
	cols [][]glowDay // cols[col] is that week's days, Monday first
	cur  int         // the cursor's day, an index into days
	left int         // the first visible column
}

// glowOn is the feature's real state: a beta feature needs the build to
// allow beta, beta mode to be on, and its own switch on.
func (m *Model) glowOn() bool {
	return version.Beta && m.opts.Beta && m.opts.Glow
}

// openGlow gathers the vault and opens the panel.
func (m *Model) openGlow() {
	if !m.glowOn() {
		m.flash = "Glow is off: it's a beta feature, switched on in Settings (" + note(m.keyFor(inMain, actHelp), "?") + " then tab)"
		return
	}
	days, cols := buildGlowDays(m.opts.Now(), m.glowCounts())
	v := &glowView{days: days, cols: cols}
	if len(days) > 0 {
		v.cur = len(days) - 1 // today
		// Start near the end, so today is in view on a narrow terminal.
		v.left = max(days[len(days)-1].col-10, 0)
	}
	m.glow = v
}

// glowCounts buckets every note by the day it was last written.
func (m *Model) glowCounts() map[string][]string {
	out := map[string][]string{}
	for _, rel := range m.idx.Notes() {
		mod, _, ok := m.idx.Stat(rel)
		if !ok {
			continue
		}
		k := dateKey(mod).Format("2006-01-02")
		out[k] = append(out[k], rel)
	}
	return out
}

// buildGlowDays is the pure core: a column per week, Monday-first, ending
// at the week that holds now, oldest leftmost. The final week stops at
// today, so a year in progress shows a year in progress.
func buildGlowDays(now time.Time, counts map[string][]string) ([]glowDay, [][]glowDay) {
	end := dateKey(now)
	start := mondayOf(end).AddDate(0, 0, -51*7)
	var days []glowDay
	var cols [][]glowDay
	for wm := start; !wm.After(end); wm = wm.AddDate(0, 0, 7) {
		col := len(cols)
		var week []glowDay
		for i := 0; i < 7; i++ {
			d := wm.AddDate(0, 0, i)
			if d.After(end) {
				break
			}
			week = append(week, glowDay{date: d, notes: counts[d.Format("2006-01-02")], col: col})
		}
		cols = append(cols, week)
		days = append(days, week...)
	}
	return days, cols
}

// dateKey is a day with the clock stripped, in local time: the day a note
// was written, wherever the machine is.
func dateKey(t time.Time) time.Time {
	y, mo, d := t.Date()
	return time.Date(y, mo, d, 0, 0, 0, 0, time.Local)
}

// mondayOf is the Monday that begins the week holding d.
func mondayOf(d time.Time) time.Time {
	return d.AddDate(0, 0, -monIndex(d))
}

// monIndex is Monday → 0, Sunday → 6.
func monIndex(d time.Time) int {
	return (int(d.Weekday()) + 6) % 7
}

// glowKey handles the panel's keys.
func (m *Model) glowKey(k tea.KeyPressMsg) {
	v := m.glow
	switch m.actionIn(inGlow, k.String()) {
	case actCancel:
		m.glow = nil
		m.flash = "Glow closed"
	case actLeft:
		v.cur = max(v.cur-1, 0)
	case actRight:
		v.cur = min(v.cur+1, len(v.days)-1)
	case actUp:
		v.cur = max(v.cur-7, 0)
	case actDown:
		v.cur = min(v.cur+7, len(v.days)-1)
	case actPick:
		m.openGlowDay()
	}
}

// openGlowDay lists what was written on the cursor's day, or says the day
// was quiet. It never creates: a quiet day stays a quiet day.
func (m *Model) openGlowDay() {
	d := m.glow.days[m.glow.cur]
	if len(d.notes) == 0 {
		m.flash = "Nothing written on " + d.date.Format("Mon 02 Jan 2006")
		return
	}
	m.glow = nil
	c := &chooser{
		title:  fmt.Sprintf("Written on %s (%d)", d.date.Format("02 Jan 2006"), len(d.notes)),
		prompt: "filter",
		empty:  "No match",
		verb:   "open",
	}
	for _, rel := range d.notes {
		rel := rel
		c.items = append(c.items, choice{
			label: displayName(rel),
			do: func() {
				if m.editor != nil {
					if m.saveEdit(true); m.editor != nil {
						return // a conflict came up; the dialog has focus now
					}
				}
				m.open(rel)
			},
		})
	}
	m.openChooser(c)
}

// glowLevels is the colour ramp: the empty day, then four greens, from the
// page's own tint up to the theme's green.
func (m *Model) glowLevels() []color.Color {
	base, top := m.pal.LighterBackground, m.pal.Green
	lv := make([]color.Color, 5)
	lv[0] = base
	for i := 1; i < 5; i++ {
		lv[i] = lerp(base, top, float64(i)/4.0)
	}
	return lv
}

// levelFor maps a day's note count onto the ramp.
func levelFor(count int) int {
	switch {
	case count <= 0:
		return 0
	case count == 1:
		return 1
	case count <= 3:
		return 2
	case count <= 6:
		return 3
	default:
		return 4
	}
}

// lerp mixes two colours towards b by t (0..1).
func lerp(a, b color.Color, t float64) color.Color {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	return color.RGBA{
		R: uint8(lerp16(ar, br, t) >> 8),
		G: uint8(lerp16(ag, bg, t) >> 8),
		B: uint8(lerp16(ab, bb, t) >> 8),
		A: 0xff,
	}
}

func lerp16(a, b uint32, t float64) uint32 {
	return uint32(float64(a) + (float64(b)-float64(a))*t)
}

// glowBox draws the panel: a month label line, seven weekday rows, and the
// cursor's day underneath. Columns off to the left scroll away as the
// cursor moves right, the way the grid keeps the year within any terminal.
func (m *Model) glowBox() []string {
	v := m.glow
	if v == nil {
		return nil
	}
	if len(v.days) == 0 {
		return m.box(" Glow ", []string{"", "  Nothing here yet."}, 50, 4, true)
	}
	w := min(max(m.width-6, 50), 96)
	inner := w - 2
	const (
		labelW = 2 // "Mo" … "Su"
		stride = 3 // a 2-cell day and its gap
	)
	maxCols := max((inner-labelW)/stride, 1)

	// Keep the cursor's column in view.
	curCol := v.days[v.cur].col
	if curCol < v.left {
		v.left = curCol
	}
	if curCol >= v.left+maxCols {
		v.left = curCol - maxCols + 1
	}
	v.left = clamp(v.left, 0, max(len(v.cols)-1, 0))

	levels := m.glowLevels()
	body := make([]string, 0, 11)

	// Month labels, one at each column where the month changes.
	var lbl strings.Builder
	lbl.WriteString(strings.Repeat(" ", labelW))
	prev := time.Month(0)
	for c := v.left; c < v.left+maxCols && c < len(v.cols); c++ {
		mo := v.cols[c][0].date.Month()
		if c == v.left || mo != prev {
			lbl.WriteString(fit(mo.String()[:3], stride))
		} else {
			lbl.WriteString(strings.Repeat(" ", stride))
		}
		prev = mo
	}
	body = append(body, strings.TrimRight(lbl.String(), " "))

	// The seven weekday rows.
	wd := []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
	for r := 0; r < 7; r++ {
		var line strings.Builder
		line.WriteString(fit(wd[r], labelW))
		for c := v.left; c < v.left+maxCols && c < len(v.cols); c++ {
			col := v.cols[c]
			if r >= len(col) {
				line.WriteString(strings.Repeat(" ", stride))
				continue
			}
			d := col[r]
			cell := lipgloss.NewStyle().Background(levels[levelFor(len(d.notes))]).Render("  ")
			if v.days[v.cur].date.Equal(d.date) {
				cell = lipgloss.NewStyle().Background(m.pal.Foreground).Render("  ")
			}
			line.WriteString(cell)
			line.WriteString(" ")
		}
		body = append(body, strings.TrimRight(line.String(), " "))
	}

	body = append(body, "")

	// The cursor's day, with how much was written there.
	sel := v.days[v.cur]
	written := "nothing written"
	switch len(sel.notes) {
	case 1:
		written = "1 note"
	default:
		written = fmt.Sprintf("%d notes", len(sel.notes))
	}
	body = append(body, fmt.Sprintf("  %s · %s · enter open · esc close",
		sel.date.Format("Mon 02 Jan 2006"), written))

	return m.box(" Glow — your year of writing ", body, w, len(body)+2, true)
}
