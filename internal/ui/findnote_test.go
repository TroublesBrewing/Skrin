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

// viewFindModel reads a note long enough to scroll, with "stoa" on the
// first line and far down it.
func viewFindModel(t *testing.T) *Model {
	t.Helper()
	m := newTestModel(t)
	body := "Stoa överst\n\n" + strings.Repeat("fyllnad\n\n", 40) + "stoa längst ned\n"
	if err := os.WriteFile(m.vault.Abs("Welcome.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m.reload()
	press(m, "G", "l") // Welcome, being read
	return m
}

func TestFindInTheViewFindsHighlightsAndScrolls(t *testing.T) {
	m := viewFindModel(t)
	press(m, "ctrl+f")
	if m.noteFind == nil || !m.noteFind.inView {
		t.Fatal("ctrl+f while reading should find in the note")
	}
	typeText(m, "stoa")
	if n := len(m.noteFind.matches); n != 2 {
		t.Fatalf("matches: %d, want 2", n)
	}
	if s := ansi.Strip(m.statusLine()); !strings.Contains(s, "1 of 2") {
		t.Errorf("the status line should count: %q", s)
	}
	checkFrame(t, m, "finding while reading")
	if !strings.Contains(m.render(), m.st.selFocus.Render("Stoa")) {
		t.Error("the match should be highlighted where it stands")
	}
	press(m, "enter") // the one far down
	if m.noteOff == 0 {
		t.Error("a match below the fold should scroll the note to it")
	}
	if s := ansi.Strip(m.statusLine()); !strings.Contains(s, "2 of 2") {
		t.Errorf("status after enter: %q", s)
	}
	press(m, "enter") // round to the first
	if s := ansi.Strip(m.statusLine()); !strings.Contains(s, "1 of 2") {
		t.Errorf("forward from the last should go round: %q", s)
	}
	press(m, "shift+enter")
	if s := ansi.Strip(m.statusLine()); !strings.Contains(s, "2 of 2") {
		t.Errorf("shift+enter should go back: %q", s)
	}
}

func TestFindInTheViewCloses(t *testing.T) {
	m := viewFindModel(t)
	press(m, "ctrl+f")
	typeText(m, "stoa")
	press(m, "esc")
	if m.noteFind != nil {
		t.Fatal("esc should close the find field")
	}
	if strings.Contains(m.render(), m.st.selFocus.Render("Stoa")) {
		t.Error("nothing should stay highlighted once it's closed")
	}
	if s := ansi.Strip(m.statusLine()); !strings.Contains(s, "VIEW") {
		t.Errorf("the status line should go back: %q", s)
	}
}

func TestFindInTheViewWithNoMatchStaysPut(t *testing.T) {
	m := viewFindModel(t)
	press(m, "ctrl+f")
	typeText(m, "zzz")
	if s := ansi.Strip(m.statusLine()); !strings.Contains(s, "no match") {
		t.Errorf("status: %q", s)
	}
	if m.noteOff != 0 {
		t.Errorf("nothing found: the note shouldn't move: %d", m.noteOff)
	}
}

func TestFindNeedsANoteToFindIn(t *testing.T) {
	m := newTestModel(t) // the vault root: no note open
	press(m, "ctrl+f")
	if m.noteFind != nil || m.flash != "Select a note to find in" {
		t.Errorf("find %v, flash %q", m.noteFind != nil, m.flash)
	}
}
