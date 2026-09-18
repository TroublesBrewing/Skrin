package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// withSpread puts a spread into Welcome.md, then opens it. The fixture's
// Daily/ holds 2026-09-11.md and 2026-09-13.md.
func withSpread(t *testing.T, opts Options, query string) *Model {
	t.Helper()
	m := newTestModelWith(t, opts)
	if err := os.WriteFile(m.vault.Abs("Welcome.md"), []byte("# Welcome\n```spread\n"+query+"\n```\nafter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	onWelcome(m)
	return m
}

func screen(m *Model) string { return ansi.Strip(m.render()) }

func TestSpreadShowsWhatItFinds(t *testing.T) {
	m := withSpread(t, Options{Spreads: true}, `LIST FROM "Daily"`)
	s := screen(m)
	for _, want := range []string{"2026-09-11", "2026-09-13", "after"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "```") || strings.Contains(s, `FROM "Daily"`) {
		t.Errorf("the query is still showing:\n%s", s)
	}
	checkFrame(t, m, "a spread in the note")
}

func TestSpreadResultsAreLinksToFollow(t *testing.T) {
	m := withSpread(t, Options{Spreads: true}, `LIST FROM "Daily"`)
	press(m, "f")
	if m.hints == nil || len(m.hints.hints) != 2 {
		t.Fatalf("hints = %+v", m.hints)
	}
	press(m, "a")
	if m.notePath != "Daily/2026-09-11.md" {
		t.Errorf("followed to %q", m.notePath)
	}
}

func TestSpreadStaysCurrentWhenTheVaultChanges(t *testing.T) {
	m := withSpread(t, Options{Spreads: true}, `LIST FROM "Daily"`)
	if err := os.WriteFile(m.vault.Abs("Daily/2026-09-14.md"), []byte("new day"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	if s := screen(m); !strings.Contains(s, "2026-09-14") {
		t.Errorf("a new note didn't show up in the spread:\n%s", s)
	}
}

func TestSpreadTableAndErrors(t *testing.T) {
	m := withSpread(t, Options{Spreads: true}, "TABLE file.folder\nFROM \"Daily\"\nSORT file.name DESC")
	s := screen(m)
	if i, j := strings.Index(s, "2026-09-13"), strings.Index(s, "2026-09-11"); i < 0 || j < 0 || i > j {
		t.Errorf("want 2026-09-13 before 2026-09-11:\n%s", s)
	}
	if !strings.Contains(s, "file.folder") || !strings.Contains(s, "Daily") {
		t.Errorf("the table's header and folder column are missing:\n%s", s)
	}
	m = withSpread(t, Options{Spreads: true}, "LIST FROM \"Daily\"\nGROUP BY file.folder")
	if s := screen(m); !strings.Contains(s, "⚠ Spread: GROUP BY isn't supported yet (line 2)") {
		t.Errorf("no message for unsupported syntax:\n%s", s)
	}
}

func TestSpreadsOffShowTheQuery(t *testing.T) {
	m := withSpread(t, Options{}, `LIST FROM "Daily"`)
	if s := screen(m); !strings.Contains(s, `LIST FROM "Daily"`) || strings.Contains(s, "2026-09-11") {
		t.Errorf("with spreads off, the query should show as code:\n%s", s)
	}
}

func TestShowSpreadsSetting(t *testing.T) {
	m := withSpread(t, Options{Spreads: true}, `LIST FROM "Daily"`)
	for _, it := range settingsItems() {
		if it.label == "Show spreads" {
			it.set(m, false)
		}
	}
	m.settle()
	if s := screen(m); !strings.Contains(s, `LIST FROM "Daily"`) {
		t.Errorf("turning spreads off should show the query again:\n%s", s)
	}
	if m.opts.Config.RenderSpreads() {
		t.Error("the setting should be kept in config")
	}
}
