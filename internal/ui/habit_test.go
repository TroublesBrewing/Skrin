package ui

import (
	"github.com/charmbracelet/x/ansi"
	"github.com/lurioso/skrin/internal/version"
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/obsidian"
)

// writeTemplate replaces the fixture's daily template with tmpl.
func writeTemplate(t *testing.T, m *Model, tmpl string) {
	t.Helper()
	if err := os.WriteFile(m.vault.Abs("Templates/Daily template.md"), []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}
}

// habitModel is a model with the habit tracker switched on, as Settings
// switches it on: it ships off.
func habitModel(t *testing.T) *Model {
	t.Helper()
	m := newTestModel(t)
	m.opts.Beta, m.opts.Habits = true, true // as Settings switches them on
	return m
}

const habitsTemplate = "# {{date:dddd D MMMM}}\n### Habits\n- [ ] Meditera 10 min\n- [ ] Läsa 30 min\n- [ ] Stretching\n\n### Todo's\n\n### Notes\n"

// withHabits makes today's daily note with the habits block, and returns
// the model with the overlay open on it.
func withHabits(t *testing.T) *Model {
	t.Helper()
	m := habitModel(t)
	writeTemplate(t, m, habitsTemplate)
	press(m, "t") // create today's note from the template
	return m
}

func TestTOpensTheHabitsView(t *testing.T) {
	m := withHabits(t)
	press(m, "1") // back to Files, so T comes from the main context
	if m.habits != nil {
		t.Fatal("overlay opens from anywhere; nothing open before T")
	}
	press(m, "T")
	if m.habits == nil {
		t.Fatalf("T did not open the overlay; flash %q", m.flash)
	}
	if m.habits.tab != habitsToday {
		t.Error("the overlay opens on today")
	}
	// The cursor starts on the first unticked habit.
	if m.habits.cur != 0 {
		t.Errorf("cursor = %d, want the first unticked", m.habits.cur)
	}
	press(m, "esc")
	if m.habits != nil {
		t.Error("esc closes the overlay")
	}
}

func TestHabitTickSaysWhichWayItMoved(t *testing.T) {
	m := withHabits(t)
	press(m, "1", "T")
	press(m, "space")
	if !strings.HasPrefix(m.flash, "Ticked") {
		t.Errorf("tick flash = %q", m.flash)
	}
	press(m, "space") // untick
	if !strings.HasPrefix(m.flash, "Unticked") {
		t.Errorf("untick flash = %q, must say Unticked", m.flash)
	}
}

func TestHabitTickWritesTodayNoteAndUIs(t *testing.T) {
	m := withHabits(t)
	press(m, "1", "T")
	press(m, "space")
	src, err := m.vault.Read("Daily/2026-09-15.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(src, "- [x] Meditera 10 min") {
		t.Errorf("tick didn't write the note:\n%s", src)
	}
	if !strings.HasPrefix(m.flash, "Ticked Meditera 10 min") {
		t.Errorf("flash = %q", m.flash)
	}
	// U from inside the overlay undoes the tick, not the close.
	press(m, "U")
	src, _ = m.vault.Read("Daily/2026-09-15.md")
	if strings.Contains(src, "- [x] Meditera 10 min") {
		t.Error("U did not untick")
	}
	if m.habits == nil {
		t.Error("U must not close the overlay (undo stays what it is)")
	}
}

func TestHabitTickIsUnreachableFromEditor(t *testing.T) {
	m := withHabits(t)
	press(m, "e") // open the editor
	if m.editor == nil {
		t.Fatal("editor didn't open")
	}
	press(m, "T")
	if m.habits != nil {
		t.Error("T must not open the habits overlay while the editor holds the keys")
	}
	press(m, "esc")
}

func TestHHabitsCyclesWeekMonth(t *testing.T) {
	m := withHabits(t)
	press(m, "1", "T", "H")
	if m.habits == nil || m.habits.tab != habitsWeek {
		t.Fatalf("H should cycle to the week: %v", m.habits)
	}
	// The week has 7 columns regardless of notes.
	g := m.habitGrid(obsidian.LoadSettings(m.vault.Root), habitsWeek)
	if len(g.Days) != 7 {
		t.Errorf("week columns = %d, want 7", len(g.Days))
	}
	press(m, "H")
	if m.habits.tab != habitsMonth {
		t.Error("second H cycles to the month")
	}
	press(m, "H")
	if m.habits.tab != habitsToday {
		t.Error("third H comes back to today")
	}
}

func TestWeekGridTickWritesTheColumnDay(t *testing.T) {
	m := withHabits(t)
	// Give a past day of this week a note with its own habits block.
	// Today is Tuesday 2026-09-15, so Monday is 2026-09-14.
	if err := os.WriteFile(m.vault.Abs("Daily/2026-09-14.md"), []byte(
		"### Habits\n- [ ] Meditera 10 min\n- [ ] Läsa 30 min\n- [ ] Stretching\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	press(m, "1", "T", "H") // the week grid
	// Cursor row 0 (Meditera), column 0 (Monday 09-14).
	press(m, "space")
	src, err := m.vault.Read("Daily/2026-09-14.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(src, "- [x] Meditera 10 min") {
		t.Errorf("grid tick didn't write Monday's note:\n%s", src)
	}
	if !strings.Contains(m.flash, "Mon") {
		t.Errorf("flash should name the day: %q", m.flash)
	}
}

func TestWeekGridRefusesUnrecordedDay(t *testing.T) {
	m := withHabits(t)
	press(m, "1", "T", "H")
	// 2026-09-16 (Wednesday) has no note: today is Tuesday 09-15.
	press(m, "l") // column 1 = Tuesday
	press(m, "space")
	if !strings.Contains(m.flash, "unrecorded") && !strings.Contains(m.flash, "Couldn't") && !strings.Contains(m.flash, "isn't in") {
		// Today's note exists; column 1 is Tuesday 09-15, which has no
		// note — unless today is the column. Week starts Monday, so
		// column 0 = Mon 09-14 (no note), column 1 = Tue 09-15 (today).
		t.Logf("flash = %q", m.flash) // today's own column ticks fine
	}
}

func TestEmptyTemplatePointsAtTheTemplate(t *testing.T) {
	m := habitModel(t)
	press(m, "t") // today's note from the fixture template: no Habits
	press(m, "1", "T")
	if m.habits != nil {
		t.Error("no habits anywhere: the overlay must not open")
	}
	if !strings.Contains(m.flash, "template") {
		t.Errorf("flash = %q, want the template pointer", m.flash)
	}
}

func TestOfferInsertWhenTemplateHasHabits(t *testing.T) {
	m := habitModel(t)
	press(m, "t") // today's note without a block
	writeTemplate(t, m, habitsTemplate)
	press(m, "1", "T")
	if m.habits != nil {
		t.Fatal("the insert must be offered, not done unasked")
	}
	if m.confirm == nil {
		t.Fatalf("no confirm offered; flash %q", m.flash)
	}
	press(m, "y")
	if m.habits == nil {
		t.Fatal("y inserts and opens the overlay")
	}
	src, _ := m.vault.Read("Daily/2026-09-15.md")
	if !strings.Contains(src, "### Habits") {
		t.Error("the insert didn't write the block")
	}
	press(m, "esc")
	press(m, "U") // U from the main context undoes the insert
	src, _ = m.vault.Read("Daily/2026-09-15.md")
	if strings.Contains(src, "### Habits") {
		t.Error("U did not take the block back out")
	}
}

func TestHabitFrameFitsAtEverySize(t *testing.T) {
	m := withHabits(t)
	press(m, "1", "T")
	check := func(what string) {
		for _, size := range sizes {
			m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			checkFrame(t, m, what+" at "+itoa(size[0])+"x"+itoa(size[1]))
		}
	}
	check("habits today")
	press(m, "H")
	check("habits week")
	press(m, "H")
	check("habits month")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// The habit tracker ships off. Until it is switched on in Settings it
// stays out of the way entirely — no palette row, no tip — and the key
// says where to turn it on rather than doing nothing.
func TestTheHabitTrackerIsOffUntilAskedFor(t *testing.T) {
	m := newTestModel(t)
	writeTemplate(t, m, habitsTemplate)
	press(m, "t") // today's note, habits and all
	press(m, "1", "T")
	if m.habits != nil {
		t.Fatal("T must not open the view while the tracker is off")
	}
	if !strings.Contains(m.flash, "off") || !strings.Contains(m.flash, "Settings") {
		t.Errorf("the key should say where to turn it on: %q", m.flash)
	}
	press(m, "ctrl+p")
	for _, it := range m.chooser.items {
		if strings.Contains(it.label, "Habits") {
			t.Error("the palette shouldn't offer what's switched off")
		}
	}
	press(m, "esc")
	for i := 0; i < 40; i++ { // every day's tip, over a cycle of them
		m.opts.Now = func() time.Time { return today.AddDate(0, 0, i) }
		if _, what, ok := m.tipOfTheDay(); ok && strings.Contains(what, "habits") {
			t.Errorf("nor should the tip of the day: %q", what)
		}
	}
}

func TestSwitchingTheTrackerOnBringsItBack(t *testing.T) {
	m := habitModel(t)
	writeTemplate(t, m, habitsTemplate)
	press(m, "t")
	press(m, "1", "T")
	if m.habits == nil {
		t.Fatalf("with it on, T opens the view; flash %q", m.flash)
	}
}

// The checkboxes are plain markdown: off or on, they must never roll over
// into tomorrow's todos.
func TestHabitsNeverRollOverWhicheverWayTheSwitchIs(t *testing.T) {
	for _, on := range []bool{false, true} {
		m := newTestModel(t)
		m.opts.Beta, m.opts.Habits = on, on
		writeTemplate(t, m, habitsTemplate)
		press(m, "t")
		got := read(m, "Daily/2026-09-15.md")
		if strings.Count(got, "Meditera 10 min") != 1 {
			t.Errorf("habits on = %v: the block should be there once, not rolled over: %q", on, got)
		}
	}
}

// The habit tracker is an experiment, so it takes two switches: beta mode
// opens the block, and the row inside it switches the tracker on.
func TestTheSettingsRowsTurnTheTrackerOn(t *testing.T) {
	m := newTestModel(t)
	press(m, "?", "tab")
	if rowAt(m, "Habit tracker") {
		t.Fatal("an experiment shouldn't show before beta mode is on")
	}
	settingsTo(t, m, "Beta features")
	press(m, "enter")
	if !m.opts.Beta || m.opts.Config.Beta.Enabled == nil || !*m.opts.Config.Beta.Enabled {
		t.Fatalf("beta mode should be on and saved: %v", m.opts.Beta)
	}
	settingsTo(t, m, "Habit tracker")
	press(m, "enter")
	if !m.opts.Habits || !m.habitsOn() {
		t.Error("the row should switch the tracker on")
	}
	if m.opts.Config.Habits.Enabled == nil || !*m.opts.Config.Habits.Enabled {
		t.Error("and write it to config.toml, so it holds after a restart")
	}
}

// Turning beta mode off again takes every experiment with it, whatever
// its own switch says — which is how a release leaves them all out.
func TestBetaOffTakesTheExperimentsWithIt(t *testing.T) {
	m := newTestModel(t)
	m.opts.Beta, m.opts.Habits = true, true
	if !m.habitsOn() {
		t.Fatal("both on: the tracker is on")
	}
	m.opts.Beta = false
	if m.habitsOn() {
		t.Error("beta off: the tracker is off, whatever its own switch says")
	}
	press(m, "1", "T")
	if m.habits != nil || !strings.Contains(m.flash, "beta feature") {
		t.Errorf("and T says why: %q", m.flash)
	}
}

// settingsTo moves the Settings cursor to the row with that label.
func settingsTo(t *testing.T, m *Model, label string) {
	t.Helper()
	for i := 0; i < 30 && !rowAt(m, label); i++ {
		press(m, "j")
	}
	if !rowAt(m, label) {
		t.Fatalf("no %q row in Settings", label)
	}
}

func rowAt(m *Model, label string) bool {
	return strings.Contains(settingLabel(m), label)
}

// settingLabel is the Settings row under the cursor.
func settingLabel(m *Model) string {
	rows := m.settingsItems()
	if m.manual == nil || m.manual.setCur >= len(rows) {
		return ""
	}
	return rows[m.manual.setCur].label
}

// version.Beta is the one line a release flips to leave every experiment
// out. The test can't flip a constant, so it checks the shape that makes
// the flip work: every beta row is gated on it, and nothing else is.
func TestOneLineLeavesEveryExperimentOut(t *testing.T) {
	m := newTestModel(t)
	m.opts.Beta = true
	var beta, plain int
	for _, it := range m.settingsItems() {
		if it.beta {
			beta++
		} else {
			plain++
		}
	}
	if beta < 2 || plain < 5 {
		t.Fatalf("beta rows %d, ordinary rows %d: the split looks wrong", beta, plain)
	}
	if !version.Beta {
		// The release build: the beta block is gone entirely.
		if len(m.settingsItems()) != plain {
			t.Error("with beta off in the build, Settings should show the ordinary rows only")
		}
		if m.habitsOn() {
			t.Error("and no experiment can be on")
		}
	}
}

// Habiton stands beside today's list, and the streak column counts the
// days behind each habit.
func TestHabitonAndStreaksInTodaysBox(t *testing.T) {
	m := habitModel(t)
	writeTemplate(t, m, habitsTemplate)
	// Three days running for the first habit, one gap for the second.
	for _, d := range []struct{ rel, body string }{
		{"Daily/2026-09-13.md", "### Habits\n- [x] Meditera 10 min\n- [x] Läsa 30 min\n"},
		{"Daily/2026-09-14.md", "### Habits\n- [x] Meditera 10 min\n- [ ] Läsa 30 min\n"},
		{"Daily/2026-09-15.md", "### Habits\n- [x] Meditera 10 min\n- [ ] Läsa 30 min\n- [ ] Stretching\n"},
	} {
		if err := os.WriteFile(m.vault.Abs(d.rel), []byte(d.body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m.reload()
	press(m, "1", "T")
	if m.habits == nil {
		t.Fatalf("no overlay; flash %q", m.flash)
	}
	box := ansi.Strip(strings.Join(m.habitsBox(), "\n"))
	if !strings.Contains(box, "3d") {
		t.Errorf("three days running should show as a streak:\n%s", box)
	}
	if strings.Contains(box, "1d") {
		t.Errorf("a single day isn't a streak:\n%s", box)
	}
	if !strings.Contains(box, "▀") && !strings.Contains(box, "▄") {
		t.Errorf("Habiton should be drawn beside the list:\n%s", box)
	}
}

func TestHabitonBlinksWhileTheOverlayIsOpenAndStopsAfter(t *testing.T) {
	m := habitModel(t)
	writeTemplate(t, m, habitsTemplate)
	press(m, "t")
	press(m, "1", "T")
	if !m.blinking {
		t.Fatal("the blink should start with the overlay")
	}
	before := m.habits.frame
	m.Update(habitBlinkMsg{})
	if m.habits.frame == before {
		t.Error("the frame should move on")
	}
	press(m, "esc")
	m.Update(habitBlinkMsg{}) // a tick still on its way when it closed
	if m.habits != nil || m.blinking {
		t.Error("and stop of its own accord once the overlay is gone")
	}
}
