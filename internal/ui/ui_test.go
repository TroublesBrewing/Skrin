package ui

import (
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
	t.Setenv("XDG_DATA_HOME", t.TempDir())  // keep the real trash out of it
	t.Setenv("XDG_STATE_HOME", t.TempDir()) // and the real snapshots
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
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "alt+left":
		return tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt}
	case "alt+right":
		return tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt}
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

func TestFrameFillsTerminalExactly(t *testing.T) {
	m := newTestModel(t)
	for _, size := range [][2]int{{140, 45}, {100, 30}, {80, 24}, {50, 12}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, focus := range []string{"1", "2", "3"} {
			press(m, focus)
			checkFrame(t, m, "focus "+focus)
		}
	}
}

func TestModalsFillTerminalExactly(t *testing.T) {
	m := newTestModel(t)
	for _, size := range [][2]int{{120, 40}, {60, 16}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		press(m, "2", "n")
		typeText(m, "a rather long name for a brand new note")
		checkFrame(t, m, "new-note prompt")
		press(m, "esc", "d")
		checkFrame(t, m, "delete confirm")
		press(m, "n", "m")
		if m.chooser == nil {
			t.Fatal("m did not open the folder picker")
		}
		checkFrame(t, m, "move picker")
		press(m, "esc")
	}
}

func TestNavigateTreeListAndNote(t *testing.T) {
	m := newTestModel(t)

	// Tree rows: vault root, Daily, Filosofi, Templates.
	press(m, "1", "j", "j")
	if m.cwd != "Filosofi" {
		t.Fatalf("cwd = %q, want Filosofi", m.cwd)
	}
	// Contents: Antik/ first, then Stoic.md, which previews on selection.
	press(m, "l", "j")
	if m.notePath != "Filosofi/Stoic.md" || len(m.lines) == 0 {
		t.Fatalf("preview = %q (%d lines)", m.notePath, len(m.lines))
	}
	press(m, "enter", "G")
	if m.focus != paneNote || m.noteOff == 0 {
		t.Errorf("focus %v, offset %d: want note pane scrolled to the bottom", m.focus, m.noteOff)
	}
	// Entering a subfolder from the list moves the tree along with it.
	press(m, "h", "g", "enter")
	if m.cwd != "Filosofi/Antik" || m.tree.selected() != "Filosofi/Antik" {
		t.Errorf("cwd = %q, tree on %q", m.cwd, m.tree.selected())
	}
	press(m, "backspace")
	if m.cwd != "Filosofi" {
		t.Errorf("backspace: cwd = %q", m.cwd)
	}
	if e, _ := m.selected(); e.Rel != "Filosofi/Antik" {
		t.Errorf("after going up, cursor on %q, want the folder we came from", e.Rel)
	}
}

func TestVaultChangeKeepsPlace(t *testing.T) {
	m := newTestModel(t)
	press(m, "1", "j", "j", "l", "j")
	if err := os.WriteFile(m.vault.Abs("Filosofi/Aaa.md"), []byte("# new"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	if e, _ := m.selected(); e.Rel != "Filosofi/Stoic.md" {
		t.Errorf("cursor moved to %q after a new file appeared", e.Rel)
	}
	if len(m.entries) != 3 {
		t.Errorf("new file not listed: %d entries", len(m.entries))
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
