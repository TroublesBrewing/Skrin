package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/theme"
	"github.com/lurioso/skrin/internal/vault"
)

// today is the pinned clock for tests: a Tuesday.
var today = time.Date(2026, 9, 15, 9, 30, 0, 0, time.Local)

var fixture = map[string]string{
	"Welcome.md":                       "# Welcome\nSee [[Stoic]] and [[Missing]] #start\n",
	"Filosofi/Stoic.md":                "---\ntags: [stoa]\n---\n# Stoic\n## Morning\n" + strings.Repeat("line\n", 80),
	"Filosofi/Antik/Zeno.md":           "# Zeno\nTeacher of [[Stoic|the Stoics]].\n",
	"Daily/2026-09-11.md":              "### Reflection\n",
	"Daily/2026-09-13.md":              "### Todo's\n- [ ] call mum\n  - about sunday\n- [x] done thing\n- [ ] \n- [-] cancelled\n* [ ] star task\n",
	"Templates/Daily template.md":      "# {{date:dddd D MMMM}}\n### Todo's\n\n### Notes\n",
	".obsidian/daily-notes.json":       `{"folder":"Daily","template":"Templates/Daily template.md"}`,
	".obsidian/community-plugins.json": `["obsidian-rollover-daily-todos"]`,
	".obsidian/plugins/obsidian-rollover-daily-todos/data.json": `{"templateHeading":"### Todo's","deleteOnComplete":false,"removeEmptyTodos":true,"rolloverChildren":true,"doneStatusMarkers":"xX-"}`,
}

func newTestModel(t *testing.T) *Model {
	return newTestModelWith(t, Options{RolloverTodos: true})
}

func newTestModelWith(t *testing.T, opts Options) *Model {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())   // keep the real trash out of it
	t.Setenv("XDG_STATE_HOME", t.TempDir())  // and the real snapshots
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // and the real config.toml
	root := t.TempDir()
	for f, body := range fixture {
		p := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if opts.Now == nil {
		opts.Now = func() time.Time { return today }
	}
	if opts.Open == nil {
		opts.Open = func(string) error { return nil } // never start real apps
	}
	// InstantOpen defaults on here, unlike a zero Options{} in production —
	// nearly every test relies on the cursor opening notes as it moves.
	// A test after the off behaviour sets m.opts.InstantOpen = false itself,
	// once it has a model, rather than threading a way to ask for it here.
	opts.InstantOpen = true
	m, err := New(v, theme.Default(), opts)
	if err != nil {
		t.Fatal(err)
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return m
}

func key(k string) tea.KeyPressMsg {
	switch k {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "ctrl+up":
		return tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl}
	case "ctrl+down":
		return tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModCtrl}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "alt+left":
		return tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt}
	case "alt+right":
		return tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt}
	case "alt+down":
		return tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModAlt}
	case "alt+up":
		return tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModAlt}
	case "shift+right":
		return tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift}
	case "shift+left":
		return tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift}
	case "alt+enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModAlt}
	case "shift+enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift}
	case "pgup":
		return tea.KeyPressMsg{Code: tea.KeyPgUp}
	case "shift+down":
		return tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift}
	case "shift+up":
		return tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}
	}
	if c, ok := strings.CutPrefix(k, "alt+"); ok {
		return tea.KeyPressMsg{Code: []rune(c)[0], Mod: tea.ModAlt}
	}
	if c, ok := strings.CutPrefix(k, "ctrl+"); ok {
		return tea.KeyPressMsg{Code: []rune(c)[0], Mod: tea.ModCtrl}
	}
	return tea.KeyPressMsg{Code: []rune(k)[0], Text: k}
}

func press(m *Model, keys ...string) {
	for _, k := range keys {
		m.Update(key(k))
	}
}

func typeText(m *Model, s string) {
	for _, r := range s {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func checkFrame(t *testing.T, m *Model, what string) {
	t.Helper()
	lines := strings.Split(m.render(), "\n")
	if len(lines) != m.height {
		t.Fatalf("%s: %d lines, want %d", what, len(lines), m.height)
	}
	for i, l := range lines {
		if w := ansi.StringWidth(l); w != m.width {
			t.Errorf("%s: line %d is %d wide, want %d: %q", what, i, w, m.width, ansi.Strip(l))
		}
	}
}

var sizes = [][2]int{{140, 45}, {100, 30}, {80, 24}, {79, 24}, {50, 12}}

func TestFrameFillsTerminalExactly(t *testing.T) {
	m := newTestModel(t)
	check := func(what string) {
		for _, size := range sizes {
			m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			for _, focus := range []string{"1", "2"} {
				press(m, focus)
				checkFrame(t, m, fmt.Sprintf("%s, %dx%d, focus %s", what, size[0], size[1], focus))
			}
		}
	}
	check("no note open")
	press(m, "1", "G", "enter") // Welcome
	check("Welcome open")
}

func TestFilesHideBelow80ColumnsUnlessFocused(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	press(m, "2")
	if m.layout().filesW == 0 {
		t.Error("at 80 columns Files should still show")
	}
	m.Update(tea.WindowSizeMsg{Width: 79, Height: 24})
	if m.layout().filesW != 0 {
		t.Error("below 80 columns Files should hide while the note has focus")
	}
	press(m, "1")
	if m.layout().filesW == 0 {
		t.Error("focusing Files should bring it back")
	}
}

func TestModalsFillTerminalExactly(t *testing.T) {
	m := newTestModel(t)
	for _, size := range [][2]int{{120, 40}, {60, 16}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		press(m, "1", "home", "j", "n") // on Daily/
		typeText(m, "a rather long name for a brand new note")
		checkFrame(t, m, "new-note prompt")
		press(m, "esc", "d")
		if m.confirm == nil {
			t.Fatal("d did not ask")
		}
		checkFrame(t, m, "delete confirm")
		press(m, "n", "m")
		if m.chooser == nil {
			t.Fatal("m did not open the folder picker")
		}
		checkFrame(t, m, "move picker")
		press(m, "esc")
	}
}

func TestNavigateFilesAndNote(t *testing.T) {
	m := newTestModel(t)
	// Rows: the vault, Daily/, Filosofi/, Templates/, Welcome.
	press(m, "j", "j")
	if m.cwd() != "Filosofi" {
		t.Fatalf("cwd = %q, want Filosofi", m.cwd())
	}
	if m.notePath != "" {
		t.Fatalf("a folder under the cursor opened %q", m.notePath)
	}
	// l opens Filosofi (cursor stays); j steps onto Antik/, then Stoic,
	// which opens at once.
	press(m, "l", "j", "j")
	if got := m.files.selected().Rel; got != "Filosofi/Stoic.md" {
		t.Fatalf("cursor on %q", got)
	}
	if m.notePath != "Filosofi/Stoic.md" || m.focus != paneFiles {
		t.Fatalf("open %q, focus %v: the note under the cursor should open, focus staying in Files", m.notePath, m.focus)
	}
	press(m, "l", "G")
	if m.notePath != "Filosofi/Stoic.md" || m.focus != paneNote || m.noteOff == 0 {
		t.Fatalf("open %q, focus %v, offset %d: want Stoic open, scrolled to the bottom", m.notePath, m.focus, m.noteOff)
	}
	// h goes back to Files, where the cursor was left; moving it keeps the note.
	press(m, "h", "k")
	if m.focus != paneFiles || m.files.selected().Rel != "Filosofi/Antik" || m.notePath != "Filosofi/Stoic.md" {
		t.Errorf("focus %v, cursor on %q, open %q", m.focus, m.files.selected().Rel, m.notePath)
	}
	// h climbs out of a closed folder, then closes the open one.
	press(m, "h", "h")
	if m.files.selected().Rel != "Filosofi" || m.files.expanded["Filosofi"] {
		t.Errorf("h h: cursor on %q, Filosofi open %v", m.files.selected().Rel, m.files.expanded["Filosofi"])
	}
	press(m, "l", "j", "l", "H") // open Filosofi, onto Antik, open it, then close everything
	if got := m.files.openFolders(); len(got) != 0 || m.files.selected().Rel != "Filosofi" {
		t.Errorf("H: open %q, cursor on %q", got, m.files.selected().Rel)
	}
	press(m, "l", "j", "backspace") // open Filosofi, onto Antik, then up to the parent
	if m.files.selected().Rel != "Filosofi" {
		t.Errorf("backspace should go up to the folder, got %q", m.files.selected().Rel)
	}
}

func TestEnterOnOtherFileOpensItsApp(t *testing.T) {
	var opened string
	m := newTestModelWith(t, Options{Open: func(target string) error { opened = target; return nil }})
	if err := os.WriteFile(m.vault.Abs("zz.pdf"), []byte("%PDF"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	press(m, "G", "enter")
	if opened != m.vault.Abs("zz.pdf") || m.notePath != "" || m.focus != paneFiles {
		t.Errorf("opened %q, note %q, focus %v", opened, m.notePath, m.focus)
	}
}

func TestVaultChangeKeepsPlace(t *testing.T) {
	m := newTestModel(t)
	press(m, "1", "j", "j", "l", "j", "j")
	if err := os.WriteFile(m.vault.Abs("Filosofi/Aaa.md"), []byte("# new"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	if got := m.files.selected().Rel; got != "Filosofi/Stoic.md" {
		t.Errorf("cursor moved to %q after a new file appeared", got)
	}
	if n := len(m.files.siblings()); n != 3 {
		t.Errorf("new file not listed: %d entries in Filosofi", n)
	}
}

func TestOpenNoteGoneOutsideSkrin(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "enter")
	if err := os.Remove(m.vault.Abs("Welcome.md")); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	if m.notePath != "" || !strings.Contains(m.flash, "Welcome is gone") {
		t.Errorf("open %q, flash %q", m.notePath, m.flash)
	}
	checkFrame(t, m, "note gone")
}

func TestSessionIsRestored(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "l", "ctrl+d")
	s := m.Session()
	if s.Open != "Filosofi/Stoic.md" || s.Cursor != "Filosofi/Stoic.md" || s.Offset == 0 || strings.Join(s.Expanded, ",") != "Filosofi" {
		t.Fatalf("session = %+v", s)
	}
	again := newTestModelWith(t, Options{Session: s, RestoreLastNote: true})
	if again.notePath != s.Open || again.files.selected().Rel != s.Cursor || again.noteOff != s.Offset {
		t.Errorf("restored: open %q, cursor %q, offset %d; want %+v", again.notePath, again.files.selected().Rel, again.noteOff, s)
	}
}

// TestSessionNoteNotRestoredByDefault covers the privacy fix: unless
// RestoreLastNote is turned on, a fresh run starts at the welcome screen
// even though the session remembers what was open, so a vault with
// private notes never opens one by surprise.
func TestSessionNoteNotRestoredByDefault(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "enter")
	s := m.Session()
	if s.Open == "" {
		t.Fatal("the session should still remember the open note")
	}
	again := newTestModelWith(t, Options{Session: s})
	if again.notePath != "" {
		t.Errorf("with RestoreLastNote off, a fresh run should start with no note open, got %q", again.notePath)
	}
	if again.files.selected().Rel != s.Cursor {
		t.Errorf("Files' cursor should still be restored: got %q, want %q", again.files.selected().Rel, s.Cursor)
	}

	// Background messages (theme reload, window resize, opening/closing ?)
	// must NOT reopen the note under the cursor.
	again.Update(ThemeMsg{Palette: theme.Default()})
	if again.notePath != "" {
		t.Errorf("ThemeMsg reopened note: %q", again.notePath)
	}
	again.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if again.notePath != "" {
		t.Errorf("WindowSizeMsg reopened note: %q", again.notePath)
	}
	press(again, "?", "tab", "esc")
	if again.notePath != "" {
		t.Errorf("Opening/closing settings reopened note: %q", again.notePath)
	}

	// Navigating with j/k peeks notes as expected once cursor moves to a note.
	// k moves to Filosofi/Antik (a folder), so no note peeks yet:
	press(again, "k")
	if again.notePath != "" {
		t.Errorf("Moving cursor onto folder should not open note, got %q", again.notePath)
	}
	// j moves back to Filosofi/Stoic.md (a note), so it peeks:
	press(again, "j")
	if again.notePath != "Filosofi/Stoic.md" {
		t.Errorf("Moving cursor onto Stoic.md should peek it, got %q", again.notePath)
	}
	// G moves to Welcome.md (a note):
	press(again, "G")
	if again.notePath != "Welcome.md" {
		t.Errorf("Moving cursor to bottom should peek Welcome.md, got %q", again.notePath)
	}
}

func TestSessionNoteEnterOpensDirectly(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "enter")
	s := m.Session()

	again := newTestModelWith(t, Options{Session: s})
	if again.notePath != "" {
		t.Fatalf("note should not be open on start: got %q", again.notePath)
	}
	// Pressing Enter on the restored cursor row should open it and focus note pane:
	press(again, "enter")
	if again.notePath != "Filosofi/Stoic.md" || again.focus != paneNote {
		t.Errorf("Enter on restored cursor row: note %q, focus %v", again.notePath, again.focus)
	}
}

func TestSkrinsOwnBrandingAndSplash(t *testing.T) {
	m := newTestModel(t)
	frame := ansi.Strip(m.render())
	if strings.Contains(frame, "Obsidian") || strings.Contains(frame, "unofficial") {
		t.Errorf("Obsidian branding on screen:\n%s", frame)
	}
	if !strings.Contains(frame, "a terminal home for your vault") || !strings.Contains(frame, "every command, by name") {
		t.Fatalf("header tagline or splash missing:\n%s", frame)
	}
	art := 0
	for _, l := range strings.Split(frame, "\n") {
		if strings.ContainsAny(l, "▀▄") {
			art++
		}
	}
	if art <= headerHeight {
		t.Errorf("only %d rows of logo; the splash chest is missing", art)
	}
	press(m, "G", "enter")
	if strings.Contains(ansi.Strip(m.render()), "every command, by name") {
		t.Error("an open note should replace the splash")
	}
}

func TestUnresolvedLinksUseIndex(t *testing.T) {
	m := newTestModel(t)
	if !m.resolve("Stoic") || !m.resolve("filosofi/stoic") {
		t.Error("existing note not resolved by name or path")
	}
	if m.resolve("Missing") {
		t.Error("missing note resolved")
	}
}

func TestInstantOpenOffKeepsTheNotePaneStillOnCursorMovement(t *testing.T) {
	m := newTestModel(t)
	m.opts.InstantOpen = false
	// Rows: the vault, Daily/, Filosofi/, Templates/, Welcome.
	press(m, "j", "j")
	if m.notePath != "" {
		t.Fatalf("moving the cursor should not open anything: notePath = %q", m.notePath)
	}
	press(m, "l") // open Filosofi/, cursor stays
	press(m, "j") // onto Antik/, a folder
	if m.notePath != "" {
		t.Fatalf("still nothing opened: notePath = %q", m.notePath)
	}
	press(m, "j") // onto Stoic.md — with instant-open off, this alone shouldn't open it
	if m.notePath != "" {
		t.Fatalf("landing on a note shouldn't open it with instant-open off: notePath = %q", m.notePath)
	}
	press(m, "l") // explicit open, for reading
	if m.notePath != "Filosofi/Stoic.md" || m.editor != nil {
		t.Fatalf("l should still open a note for reading: notePath %q, editor open %v", m.notePath, m.editor != nil)
	}
}

func TestInstantOpenOffKeepsThePreviousNoteAsTheCursorMoves(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "1") // Welcome.md open for reading; focus back on Files, cursor still on it
	m.opts.InstantOpen = false
	press(m, "k", "k") // up from Welcome (the last row) onto Templates/, then Filosofi/
	if m.notePath != "Welcome.md" {
		t.Errorf("the previously open note should stay open until a new one is: got %q", m.notePath)
	}
	press(m, "enter") // Filosofi/ is a folder: toggles, still doesn't touch the open note
	if m.notePath != "Welcome.md" || !m.files.expanded["Filosofi"] {
		t.Errorf("notePath %q, expanded %v", m.notePath, m.files.expanded["Filosofi"])
	}
}

func TestTurningInstantOpenBackOnCatchesUpToTheCursor(t *testing.T) {
	m := newTestModel(t)
	m.opts.InstantOpen = false
	inFilosofi(m)
	press(m, "j") // onto Stoic.md, a note — instant-open off, so nothing opens yet
	if m.notePath != "" {
		t.Fatalf("setup: notePath = %q, want none yet", m.notePath)
	}
	m.opts.InstantOpen = true
	m.settle()
	if m.notePath != "Filosofi/Stoic.md" {
		t.Errorf("turning instant-open back on should sync to wherever the cursor already is: got %q", m.notePath)
	}
}

func TestInstantOpenSettingsToggle(t *testing.T) {
	m := newTestModel(t)
	if !m.opts.InstantOpen {
		t.Fatal("setup: should default on")
	}
	for _, it := range m.settingsItems() {
		if it.label == "Open notes on the cursor, not only on l/→ or Enter" {
			it.set(m, false)
		}
	}
	if m.opts.InstantOpen || m.opts.Config.InstantOpen() {
		t.Error("the toggle should turn it off in Options and in config")
	}
	press(m, "j", "j") // Filosofi/, then a note under it
	if m.notePath != "" {
		t.Errorf("off via Settings should behave like off via config: notePath = %q", m.notePath)
	}
}
