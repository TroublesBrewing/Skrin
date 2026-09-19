package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// findModel edits a note where "stoa" occurs four times, the cursor on
// the second line.
func findModel(t *testing.T) *Model {
	t.Helper()
	m := newTestModel(t)
	body := "Stoa ett\nmitten\nStoa två och stoa tre\nslut på Stoa"
	if err := os.WriteFile(m.vault.Abs("Welcome.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m.reload()
	m.openEditor("Welcome.md")
	m.editor.MoveTo(1, 0)
	return m
}

func TestFindInTheNoteStartsFromTheCursor(t *testing.T) {
	m := findModel(t)
	press(m, "ctrl+f")
	if m.noteFind == nil {
		t.Fatal("ctrl+f in the editor should open the find field")
	}
	typeText(m, "stoa")
	if m.editor.Text() != "Stoa ett\nmitten\nStoa två och stoa tre\nslut på Stoa" {
		t.Fatal("typing in the find field shouldn't change the note")
	}
	if r, c := m.editor.Cursor(); r != 2 || c != 4 || m.editor.Selection() != "Stoa" {
		t.Errorf("the first match after the cursor should be selected: %d:%d %q", r, c, m.editor.Selection())
	}
	if s := ansi.Strip(m.editLine()); !strings.Contains(s, "2 of 4") {
		t.Errorf("the status line should count: %q", s)
	}
	checkFrame(t, m, "finding in the note")
}

func TestFindInTheNoteMovesAndWraps(t *testing.T) {
	m := findModel(t)
	press(m, "ctrl+f")
	typeText(m, "stoa")
	press(m, "enter") // third
	press(m, "down")  // fourth
	if r, c := m.editor.Cursor(); r != 3 || c != 12 {
		t.Errorf("enter and down go forward: %d:%d", r, c)
	}
	press(m, "enter") // round to the first
	if r, c := m.editor.Cursor(); r != 0 || c != 4 {
		t.Errorf("forward from the last goes round to the first: %d:%d", r, c)
	}
	press(m, "shift+enter") // back to the fourth
	if r, _ := m.editor.Cursor(); r != 3 {
		t.Errorf("shift+enter goes back, round the other way: row %d", r)
	}
	press(m, "up") // third
	if r, c := m.editor.Cursor(); r != 2 || c != 17 {
		t.Errorf("up goes back too: %d:%d", r, c)
	}
}

func TestFindInTheNoteEscLeavesTheMatchSelected(t *testing.T) {
	m := findModel(t)
	press(m, "ctrl+f")
	typeText(m, "mitten")
	press(m, "esc")
	if m.noteFind != nil || m.editor == nil || m.editor.Selection() != "mitten" {
		t.Fatalf("esc closes the field, the match still selected: %q", m.editor.Selection())
	}
	typeText(m, "mitt")
	if !strings.Contains(m.editor.Text(), "\nmitt\n") {
		t.Errorf("typing replaces the selected match: %q", m.editor.Text())
	}
}

func TestFindInTheNoteWithNoMatchGoesBack(t *testing.T) {
	m := findModel(t)
	press(m, "ctrl+f")
	typeText(m, "zzz")
	if s := ansi.Strip(m.editLine()); !strings.Contains(s, "no match") {
		t.Errorf("status: %q", s)
	}
	press(m, "esc")
	if r, c := m.editor.Cursor(); r != 1 || c != 0 || m.editor.Selection() != "" {
		t.Errorf("no match: the cursor goes back to where it was: %d:%d", r, c)
	}
}

func TestFindInTheNoteTakesAPaste(t *testing.T) {
	m := findModel(t)
	press(m, "ctrl+f")
	m.paste("två och")
	if m.editor.Selection() != "två och" {
		t.Errorf("a paste should go into the find field and search: %q", m.editor.Selection())
	}
}
