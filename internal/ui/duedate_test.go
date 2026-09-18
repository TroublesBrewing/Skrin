package ui

import (
	"os"
	"strings"
	"testing"
)

// The test clock is Tuesday 2026-09-15.
func editWelcome(t *testing.T, body string) *Model {
	t.Helper()
	m := newTestModel(t)
	if err := os.WriteFile(m.vault.Abs("Welcome.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	onWelcome(m)
	press(m, "e")
	if m.editor == nil {
		t.Fatal("the editor didn't open")
	}
	return m
}

func welcomeOnDisk(t *testing.T, m *Model) string {
	t.Helper()
	b, err := os.ReadFile(m.vault.Abs("Welcome.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestLeavingTheEditorSetsDueDates(t *testing.T) {
	m := editWelcome(t, "# Welcome\n- [ ] old task [due:: tomorrow]\n")
	typeText(m, "- [ ] new task [due:: friday]")
	press(m, "enter")
	press(m, "esc")
	if m.editor != nil {
		t.Fatal("esc should leave the editor")
	}
	got := welcomeOnDisk(t, m)
	if !strings.Contains(got, "- [ ] new task [due:: 2026-09-18]") {
		t.Errorf("the new task's date wasn't set:\n%s", got)
	}
	if !strings.Contains(got, "- [ ] old task [due:: tomorrow]") {
		t.Errorf("a line from before this edit changed:\n%s", got)
	}
	if !strings.Contains(m.flash, "due:: friday → 2026-09-18") {
		t.Errorf("flash = %q", m.flash)
	}
}

func TestCtrlSAloneLeavesTheWord(t *testing.T) {
	m := editWelcome(t, "# Welcome\n")
	typeText(m, "due:: tomorrow")
	press(m, "enter", "ctrl+s")
	if got := welcomeOnDisk(t, m); !strings.Contains(got, "due:: tomorrow") {
		t.Errorf("Ctrl-s shouldn't rewrite under the cursor:\n%s", got)
	}
	press(m, "esc") // but leaving still does, though a save came between
	if got := welcomeOnDisk(t, m); !strings.Contains(got, "due:: 2026-09-16") {
		t.Errorf("leaving after a Ctrl-s should still set the date:\n%s", got)
	}
}

func TestSeveralDueDatesAreAllNamed(t *testing.T) {
	m := editWelcome(t, "# Welcome\n")
	typeText(m, "[due:: today] and (due:: monday)")
	press(m, "esc")
	if got := welcomeOnDisk(t, m); !strings.Contains(got, "[due:: 2026-09-15] and (due:: 2026-09-21)") {
		t.Errorf("got:\n%s", got)
	}
	if !strings.Contains(m.flash, "2 due dates: today → 2026-09-15, monday → 2026-09-21") {
		t.Errorf("flash = %q", m.flash)
	}
}

func TestUndoTakesTheDatesBackWithTheEdit(t *testing.T) {
	m := editWelcome(t, "# Welcome\n")
	typeText(m, "due:: tomorrow")
	press(m, "enter", "esc", "u")
	if got := welcomeOnDisk(t, m); got != "# Welcome\n" {
		t.Errorf("u should restore the note from before the edit: %q", got)
	}
}
