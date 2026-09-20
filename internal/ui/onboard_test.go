package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

// paletteLabels is what the open palette lists, in order.
func paletteLabels(m *Model) []string {
	var out []string
	for _, i := range m.chooser.matches {
		out = append(out, m.chooser.items[i].label)
	}
	return out
}

func paletteItem(t *testing.T, m *Model, label string) choice {
	t.Helper()
	for _, it := range m.chooser.items {
		if it.label == label {
			return it
		}
	}
	t.Fatalf("no %q in the palette: %v", label, paletteLabels(m))
	return choice{}
}

// runCommand opens the palette, types q and runs the best match.
func runCommand(m *Model, q string) {
	press(m, "ctrl+p")
	typeText(m, q)
	press(m, "enter")
}

func TestPaletteCoversTheKeymap(t *testing.T) {
	inPalette := map[action]bool{}
	for _, e := range mainCommands {
		if notInPalette[e.act] {
			t.Errorf("%s is both in the palette and left out of it", actionName[e.act])
		}
		inPalette[e.act] = true
	}
	for _, b := range defaultBindings {
		if b.where != inMain || b.act == actNone {
			continue
		}
		if !inPalette[b.act] && !notInPalette[b.act] {
			t.Errorf("%s (%q) is in the keymap but neither in the palette nor left out on purpose", actionName[b.act], b.help)
		}
	}
	inEditorPalette := map[action]bool{}
	for _, c := range editorCommands {
		inEditorPalette[c.act] = true
	}
	for _, b := range defaultBindings {
		if b.where == inEditor && b.act != actNone && b.act != actPalette && !inEditorPalette[b.act] {
			t.Errorf("the editor's %s isn't in its palette", actionName[b.act])
		}
	}
}

func TestPaletteOpensWithCtrlPAndColonWhileGStaysGoToNote(t *testing.T) {
	for _, k := range []string{"ctrl+p", ":"} {
		m := newTestModel(t)
		press(m, k)
		if m.chooser == nil || m.chooser.title != "Commands" {
			t.Fatalf("%s should open the palette", k)
		}
		checkFrame(t, m, "palette open")
	}
	m := newTestModel(t)
	press(m, "g")
	if m.chooser == nil || m.chooser.title != "Go to note" {
		t.Fatal("g should still open Go to note")
	}
}

func TestPaletteFindsCommandsByOtherWords(t *testing.T) {
	cases := map[string]string{
		"trash":     "Delete to the trash",
		"journal":   "Today's daily note",
		"shortcuts": "Keys: every key, and change them",
		"toc":       "Outline: jump to a heading",
		"exit":      "Quit Skrin",
	}
	for q, want := range cases {
		m := newTestModel(t)
		press(m, "ctrl+p")
		typeText(m, q)
		if got := paletteLabels(m); len(got) == 0 || got[0] != want {
			t.Errorf("%q: best match %v, want %q first", q, got, want)
		}
	}
	m := newTestModel(t)
	press(m, "ctrl+p")
	typeText(m, "trash")
	if got := paletteLabels(m); len(got) != 1 {
		t.Errorf("a word that names one command should find just that one: %v", got)
	}
	press(m, "esc", "ctrl+p")
	typeText(m, "zne")
	if got := paletteLabels(m); len(got) == 0 || got[0] != "Zen mode: just the note, centred" {
		t.Errorf("a typo in a name should still find it: %v", got)
	}
}

func TestPaletteRunsTheCommandAndTeachesItsKey(t *testing.T) {
	m := newTestModel(t)
	press(m, "G") // Welcome
	runCommand(m, "zen")
	if !m.zen {
		t.Fatal("running zen from the palette should turn zen on")
	}
	if m.chooser != nil {
		t.Error("the palette should close once a command runs")
	}
	if m.flash != "Next time: z" {
		t.Errorf("the palette should name the key for next time: flash %q", m.flash)
	}
}

func TestPaletteShowsAndTeachesReboundKeys(t *testing.T) {
	m := newTestModel(t)
	m.keys.set(inMain, actZen, "ctrl+g")
	press(m, "G", "ctrl+p")
	if it := paletteItem(t, m, "Zen mode: just the note, centred"); it.key != "ctrl+g" {
		t.Errorf("zen shows key %q, want the rebound ctrl+g", it.key)
	}
	typeText(m, "zen")
	press(m, "enter")
	if m.flash != "Next time: ctrl+g" {
		t.Errorf("flash %q", m.flash)
	}
}

func TestAPaletteRefusalKeepsTheStatusLine(t *testing.T) {
	m := newTestModel(t) // on the vault's root: no note to edit
	runCommand(m, "edit the note")
	if !strings.HasPrefix(m.flash, "Select a note") {
		t.Errorf("the command's own refusal should win over the lesson: flash %q", m.flash)
	}
}

func TestPaletteRecentCommandsComeFirst(t *testing.T) {
	m := newTestModel(t)
	press(m, "G")
	runCommand(m, "zen")
	press(m, "ctrl+p")
	if got := paletteLabels(m); got[0] != "Zen mode: just the note, centred" {
		t.Errorf("the command just run should head the list: %v", got[:3])
	}
}

func TestPaletteTogglesSettings(t *testing.T) {
	m := newTestModel(t)
	press(m, "ctrl+p")
	if it := paletteItem(t, m, "Setting: Line numbers in notes"); it.detail != "off" {
		t.Errorf("the setting should show its state: %q", it.detail)
	}
	press(m, "esc")
	runCommand(m, "setting line numbers")
	if !m.opts.LineNumbers || m.flash != "Line numbers in notes: on" {
		t.Errorf("line numbers %v, flash %q", m.opts.LineNumbers, m.flash)
	}
	b, err := os.ReadFile(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "skrin", "config.toml"))
	if err != nil || !strings.Contains(string(b), "line_numbers = true") {
		t.Errorf("the change should be saved to config.toml: %q, %v", b, err)
	}
}

func TestPaletteOpensTheManualTabs(t *testing.T) {
	m := newTestModel(t)
	runCommand(m, "settings")
	if m.manual == nil || m.manual.tab != manualTabSettings {
		t.Fatal("Settings should open the manual on its Settings tab")
	}
	m = newTestModel(t)
	runCommand(m, "guide")
	if m.manual == nil || m.manual.tab != manualTabGuide {
		t.Fatal("Guide should open the manual on its Guide tab")
	}
}

func TestPaletteQuits(t *testing.T) {
	m := newTestModel(t)
	press(m, "ctrl+p")
	typeText(m, "quit")
	_, cmd := m.Update(key("enter"))
	if !quits(cmd) {
		t.Fatal("Quit Skrin should quit")
	}
}

func TestPaletteLeavesClaudeOutWhenItIsOff(t *testing.T) {
	m := newTestModel(t)
	press(m, "ctrl+p")
	for _, l := range paletteLabels(m) {
		if strings.Contains(l, "Claude") && !strings.HasPrefix(l, "Setting:") {
			t.Errorf("Claude is off, but the palette lists %q", l)
		}
	}
	m = newTestModel(t)
	m.opts.Assistant.Enabled = true
	press(m, "ctrl+p")
	paletteItem(t, m, "Ask Claude")
}

func TestPaletteInTheEditor(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "enter") // Welcome, straight into editing
	if m.editor == nil {
		t.Fatal("setup: no editor")
	}
	typeText(m, "Hello ")
	press(m, "ctrl+p")
	if m.chooser == nil || m.chooser.title != "Commands" {
		t.Fatal("ctrl+p in the editor should open the palette")
	}
	if it := paletteItem(t, m, "Save"); it.key != "ctrl+s" {
		t.Errorf("Save shows key %q", it.key)
	}
	checkFrame(t, m, "palette over the editor")
	typeText(m, "save")
	press(m, "enter")
	if m.editor == nil {
		t.Fatal("Save should keep the editor open")
	}
	b, _ := os.ReadFile(m.vault.Abs("Welcome.md"))
	if !strings.HasPrefix(string(b), "Hello ") {
		t.Errorf("Save from the palette should write the note: %q", b)
	}
	runCommand(m, "to-do")
	if !strings.Contains(m.editor.Text(), "- [ ] Hello") {
		t.Errorf("the to-do command should run the editor's own ctrl+l: %q", m.editor.Text())
	}
}

func TestPaletteRunsTheEditorsOwnKeys(t *testing.T) {
	for _, c := range editorCommands {
		if c.act != actNone || c.key == "" || strings.HasPrefix(c.name, "Save") || strings.HasPrefix(c.name, "Leave") {
			continue
		}
		if got := keyPress(c.key).String(); got != c.key {
			t.Errorf("%s: the palette would press %q, not %q", c.name, got, c.key)
		}
	}
}

func TestStatusHintsFollowWhereYouAre(t *testing.T) {
	m := newTestModel(t)
	m.width, m.height = 200, 40
	hint := func() string { return m.hintLine(200) }
	if h := hint(); !strings.Contains(h, "l open") || !strings.Contains(h, "ctrl+p commands") {
		t.Errorf("on a folder in Files: %q", h)
	}
	press(m, "G") // Welcome, a note
	if h := hint(); !strings.Contains(h, "enter edit") || !strings.Contains(h, "l read") {
		t.Errorf("on a note in Files: %q", h)
	}
	m.opts.InstantOpen = false
	if h := hint(); !strings.HasPrefix(h, "l open · enter edit") {
		t.Errorf("on a note with instant-open off, l opens it: %q", h)
	}
	m.opts.InstantOpen = true
	press(m, "l") // over to the note
	if h := hint(); !strings.Contains(h, "e edit") || !strings.Contains(h, "f follow link") || !strings.Contains(h, "b backlinks") {
		t.Errorf("in the note: %q", h)
	}
	press(m, "v")
	if h := hint(); !strings.HasPrefix(h, "ctrl+c copy") {
		t.Errorf("with lines selected: %q", h)
	}
	press(m, "esc", "1", "space")
	if h := hint(); !strings.HasPrefix(h, "m move · d delete") {
		t.Errorf("with something marked: %q", h)
	}
}

func TestStatusHintsShowReboundKeysAndDropWhole(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l")
	m.keys.set(inMain, actEdit, "w")
	if h := m.hintLine(200); !strings.HasPrefix(h, "w edit") {
		t.Errorf("a rebound key should show as rebound: %q", h)
	}
	h := m.hintLine(30)
	if !strings.Contains(h, "ctrl+p commands") || strings.Contains(h, "…") || ansi.StringWidth(h) > 30 {
		t.Errorf("a narrow line keeps the palette and drops hints whole: %q", h)
	}
	for _, w := range []int{60, 100, 140} {
		m.width = w
		checkFrame(t, m, "status line at a narrow width")
	}
}

func TestStatusHintsLeaveClaudeOutWhenItIsOff(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "v")
	if h := m.hintLine(200); strings.Contains(h, "Claude") {
		t.Errorf("Claude is off: %q", h)
	}
	m.opts.Assistant.Enabled = true
	if h := m.hintLine(200); !strings.Contains(h, "C ask Claude") {
		t.Errorf("Claude is on: %q", h)
	}
}

func TestAKeyThatDoesNothingSaysSo(t *testing.T) {
	m := newTestModel(t)
	press(m, "y")
	if m.flash != "y does nothing here · ctrl+p finds every command" {
		t.Errorf("flash %q", m.flash)
	}
	press(m, "j")
	if m.flash != "" {
		t.Errorf("a key that works clears it: %q", m.flash)
	}
}

func TestWelcomeTellsTheTruthAboutInstantOpen(t *testing.T) {
	m := newTestModel(t)
	frame := ansi.Strip(m.render())
	if !strings.Contains(frame, "notes open as you land") || !strings.Contains(frame, "ctrl+p  every command, by name") {
		t.Errorf("welcome card with instant-open on:\n%s", frame)
	}
	m.opts.InstantOpen = false
	frame = ansi.Strip(m.render())
	if strings.Contains(frame, "notes open as you land") || !strings.Contains(frame, "open the note under the cursor") {
		t.Errorf("welcome card with instant-open off:\n%s", frame)
	}
	for _, size := range [][2]int{{120, 45}, {70, 20}, {50, 12}} {
		m.width, m.height = size[0], size[1]
		checkFrame(t, m, "welcome card")
	}
}

func TestTipOfTheDayHoldsForTheDayAndMovesOn(t *testing.T) {
	m := newTestModel(t)
	day := time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)
	m.opts.Now = func() time.Time { return day }
	k1, w1, ok := m.tipOfTheDay()
	if !ok {
		t.Fatal("no tip")
	}
	m.opts.Now = func() time.Time { return day.Add(8 * time.Hour) }
	if k, w, _ := m.tipOfTheDay(); k != k1 || w != w1 {
		t.Error("the tip should hold still for the day")
	}
	m.opts.Now = func() time.Time { return day.AddDate(0, 0, 1) }
	if _, w, _ := m.tipOfTheDay(); w == w1 {
		t.Error("the tip should move on the next day")
	}
	if !strings.Contains(ansi.Strip(m.render()), "Tip · ") {
		t.Error("the welcome card should show the tip")
	}
}

func TestEditorStatusNamesThePalette(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "enter")
	if s := ansi.Strip(m.editLine()); !strings.Contains(s, "ctrl+p commands") {
		t.Errorf("editor status line: %q", s)
	}
}

func TestTheGuideStartsWithGettingStarted(t *testing.T) {
	m := newTestModel(t)
	runCommand(m, "guide")
	var heads []string
	for _, l := range m.guideText(80) {
		if l.level == 1 {
			heads = append(heads, l.plain)
		}
	}
	if len(heads) == 0 || heads[0] != "Getting started" {
		t.Errorf("the Guide's first section should be Getting started: %v", heads)
	}
}

// One measure, never two: lines when the selection spans more than one,
// words when it sits inside a single line. Two counts side by side got
// truncated, and a truncated count is no count at all.
func TestTheStatusLineCountsWhatIsSelected(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "l") // Stoic, reading
	press(m, "v")
	if s := ansi.Strip(m.statusLine()); !strings.Contains(s, "2 words selected") {
		t.Errorf("one line selected: count its words, since \"1 line\" says nothing: %q", s)
	}
	press(m, "j")
	if s := ansi.Strip(m.statusLine()); !strings.Contains(s, "2 lines selected") {
		t.Errorf("reading view, two lines: %q", s)
	}
	press(m, "esc")
	if s := ansi.Strip(m.statusLine()); strings.Contains(s, "selected") {
		t.Errorf("no selection, no count: %q", s)
	}
	// Shift+Down from the start of a line ends at the start of the next
	// one: that covers the line above it, and nothing of the one below.
	press(m, "e", "shift+down")
	if s := ansi.Strip(m.editLine()); !strings.Contains(s, "words selected") {
		t.Errorf("one line in the editor: %q", s)
	}
	press(m, "shift+down")
	if s := ansi.Strip(m.editLine()); !strings.Contains(s, "2 lines selected") {
		t.Errorf("two lines in the editor: %q", s)
	}
}

func TestTheInstantOpenSettingNamesBothWays(t *testing.T) {
	m := newTestModel(t)
	press(m, "?", "tab")
	if frame := ansi.Strip(m.render()); !strings.Contains(frame, "Open notes on the cursor, not only on l/→ or Enter") {
		t.Errorf("the Settings tab should name both ways of opening:\n%s", frame)
	}
}
