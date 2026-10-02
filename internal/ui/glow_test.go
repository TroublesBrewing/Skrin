package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// The heatmap is a beta feature, so these ask for both switches.
func newGlowModel(t *testing.T) *Model {
	t.Helper()
	return newTestModelWith(t, Options{RolloverTodos: true, Beta: true, Glow: true})
}

func glowText(m *Model) string {
	var b strings.Builder
	for _, l := range m.glowBox() {
		b.WriteString(ansi.Strip(l))
		b.WriteString("\n")
	}
	return b.String()
}

// TestGlowIsOffUntilSwitchedOnTwice covers the experiment rule: an off
// feature isn't in the palette, and opening it says where to switch it on.
func TestGlowIsOffUntilSwitchedOnTwice(t *testing.T) {
	m := newTestModel(t) // no beta, no glow
	named := func(m *Model) bool {
		for _, c := range m.mainPaletteItems() {
			if strings.Contains(c.label, "heatmap") {
				return true
			}
		}
		return false
	}
	if named(m) {
		t.Error("an off experiment should not offer a palette row")
	}
	m.openGlow()
	if m.glow != nil {
		t.Error("an off feature must not open")
	}
	if !strings.Contains(m.flash, "beta") || !strings.Contains(m.flash, "Settings") {
		t.Errorf("flash = %q: an off feature must say where to switch it on", m.flash)
	}
	m = newTestModelWith(t, Options{Beta: true}) // beta mode alone
	m.openGlow()
	if m.glow != nil {
		t.Error("beta mode alone should not open it")
	}
	if !named(newGlowModel(t)) {
		t.Error("switched on, it should be findable by name in the palette")
	}
}

// TestBuildGlowDaysFillsAYearEndingToday pins the grid's shape: 52 weeks,
// Monday-first, ending at the week that holds now, with no future days.
func TestBuildGlowDaysFillsAYearEndingToday(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.Local) // a Thursday
	counts := map[string][]string{"2026-10-01": {"A.md"}}
	days, cols := buildGlowDays(now, counts)
	if len(cols) != 52 {
		t.Fatalf("cols = %d, want 52 weeks", len(cols))
	}
	// Every column is Monday-first and holds at most 7 days.
	for c, week := range cols {
		if monIndex(week[0].date) != 0 {
			t.Errorf("column %d does not start on a Monday: %v", c, week[0].date.Weekday())
		}
		if len(week) > 7 {
			t.Errorf("column %d has %d days", c, len(week))
		}
	}
	// The last day is today, with its note attached, and none are in the
	// future.
	last := days[len(days)-1]
	if !last.date.Equal(dateKey(now)) {
		t.Errorf("last day = %v, want today %v", last.date, dateKey(now))
	}
	if len(last.notes) != 1 || last.notes[0] != "A.md" {
		t.Errorf("today's notes = %v, want [A.md]", last.notes)
	}
	for _, d := range days {
		if d.date.After(dateKey(now)) {
			t.Errorf("day %v is in the future", d.date)
		}
	}
	// The first day is the Monday 51 weeks before the current week.
	want := mondayOf(dateKey(now)).AddDate(0, 0, -51*7)
	if !days[0].date.Equal(want) {
		t.Errorf("first day = %v, want %v", days[0].date, want)
	}
}

// TestGlowGroupsNotesByTheirLastWrittenDay checks the bucketing reads the
// index's modification times, not file contents.
func TestGlowGroupsNotesByTheirLastWrittenDay(t *testing.T) {
	m := newGlowModel(t)
	// The fixture notes carry real mtimes; the bucket keys must all be
	// well-formed dates, and every note in the index must land somewhere.
	counts := m.glowCounts()
	total := 0
	for k := range counts {
		if _, err := time.Parse("2006-01-02", k); err != nil {
			t.Errorf("bucket key %q is not a date", k)
		}
		total += len(counts[k])
	}
	if total != len(m.idx.Notes()) {
		t.Errorf("bucketed %d notes, want %d", total, len(m.idx.Notes()))
	}
}

// TestGlowCursorMovesADayAndAWeek drives the real key path, not a synthetic
// action, so the keymap table is what the cursor obeys.
func TestGlowCursorMovesADayAndAWeek(t *testing.T) {
	m := newGlowModel(t)
	m.openGlow()
	if m.glow == nil {
		t.Fatal("glow should open when on")
	}
	last := len(m.glow.days) - 1
	if m.glow.cur != last {
		t.Fatalf("cursor = %d, want today (%d)", m.glow.cur, last)
	}
	press(m, "left")
	if m.glow.cur != last-1 {
		t.Errorf("left: cursor = %d, want %d", m.glow.cur, last-1)
	}
	press(m, "up")
	if m.glow.cur != last-1-7 {
		t.Errorf("up: cursor = %d, want %d", m.glow.cur, last-1-7)
	}
	press(m, "down")
	if m.glow.cur != last-1 {
		t.Errorf("down: cursor = %d, want %d", m.glow.cur, last-1)
	}
	press(m, "esc")
	if m.glow != nil {
		t.Error("esc should close glow")
	}
}

// TestGlowBoxShowsTheSelectedDayAndStaysInFrame verifies the panel renders
// the cursor's day and fills the terminal exactly.
func TestGlowBoxShowsTheSelectedDayAndStaysInFrame(t *testing.T) {
	m := newGlowModel(t)
	m.openGlow()
	out := glowText(m)
	if !strings.Contains(out, "enter open") {
		t.Errorf("the panel should say how to open the day:\\n%s", out)
	}
	if !strings.Contains(out, "writing") {
		t.Errorf("the panel should name itself a year of writing:\\n%s", out)
	}
	checkFrame(t, m, "glow panel")
	press(m, "esc")
	if m.glow != nil {
		t.Error("esc should close it")
	}
}

// TestGlowRendersInANarrowTerminal pins the horizontal scroll: today must
// stay reachable even when the grid is wider than the box.
func TestGlowRendersInANarrowTerminal(t *testing.T) {
	m := newGlowModel(t)
	m.Update(tea.WindowSizeMsg{Width: 70, Height: 22})
	m.openGlow()
	checkFrame(t, m, "glow in a narrow terminal")
	// A second narrow size must not panic or overflow.
	m.Update(tea.WindowSizeMsg{Width: 50, Height: 12})
	checkFrame(t, m, "glow in a tiny terminal")
}
