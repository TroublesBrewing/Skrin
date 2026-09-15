package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/theme"
	"github.com/lurioso/skrin/internal/vault"
)

func newTestModel(t *testing.T) *Model {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"Welcome.md":               "# Welcome\nSee [[Stoic]] and [[Missing]] #start\n",
		"Filosofi/Stoic.md":        "---\ntags: [stoa]\n---\n# Stoic\n## Morning\n" + strings.Repeat("line\n", 80),
		"Filosofi/Antik/Zeno.md":   "# Zeno\n",
		"Daily/2026-09-11.md":      "### Reflection\n",
		".obsidian/workspace.json": "{}",
	}
	for f, body := range files {
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
	m, err := New(v, theme.Default())
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func press(m *Model, keys ...string) {
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "backspace":
			msg = tea.KeyPressMsg{Code: tea.KeyBackspace}
		case "tab":
			msg = tea.KeyPressMsg{Code: tea.KeyTab}
		default:
			r := []rune(k)[0]
			msg = tea.KeyPressMsg{Code: r, Text: k}
		}
		m.Update(msg)
	}
}

func TestFrameFillsTerminalExactly(t *testing.T) {
	m := newTestModel(t)
	for _, size := range [][2]int{{140, 45}, {100, 30}, {80, 24}, {50, 12}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, focus := range []string{"1", "2", "3"} {
			press(m, focus)
			lines := strings.Split(m.render(), "\n")
			if len(lines) != size[1] {
				t.Fatalf("%dx%d: %d lines", size[0], size[1], len(lines))
			}
			for i, l := range lines {
				if w := ansi.StringWidth(l); w != size[0] {
					t.Errorf("%dx%d focus %s: line %d is %d wide: %q", size[0], size[1], focus, i, w, ansi.Strip(l))
				}
			}
		}
	}
}

func TestNavigateTreeListAndNote(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Tree rows: vault root, Daily, Filosofi. Moving the cursor changes folder.
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
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
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
