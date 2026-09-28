package ui

import (
	"strings"
	"testing"
)

// The UX charter's flow rules promise continuity: "back into a note lands
// where you left it; the editor opens at the line you were reading." Half
// of it was true — the editor opened at the note pane's top line, so
// writing, pressing Esc to read, and coming back put the cursor at the top
// of the screen. The user's report, 2026-09-28: "Markören ska minnas sin
// plats i dokumentet … och inte behöva börja om med markören högst upp."

// longNote is a note tall enough that the top of the window and the line
// the cursor is on are certainly not the same line.
func placeFixture(t *testing.T) *Model {
	t.Helper()
	m := newTestModel(t)
	var b strings.Builder
	b.WriteString("# Long\n\n")
	for i := 1; i <= 60; i++ {
		b.WriteString("line ")
		b.WriteString(string(rune('a' + i%26)))
		b.WriteString("\n")
	}
	writeFile(t, m.vault.Root, "Long.md", b.String())
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	m.open("Long.md")
	return m
}

func TestTheCursorComesBackToWhereItWas(t *testing.T) {
	m := placeFixture(t)
	press(m, "enter") // into the editor
	if m.editor == nil {
		t.Fatal("setup: the editor should be open")
	}
	m.editor.MoveTo(40, 3)
	press(m, "esc") // out to the reading view
	if m.editor != nil {
		t.Fatal("esc should leave the editor")
	}
	press(m, "enter") // back in
	if m.editor == nil {
		t.Fatal("enter should open the editor again")
	}
	row, col := m.editor.Cursor()
	if row != 40 || col != 3 {
		t.Errorf("cursor = %d,%d — want 40,3: the line and column it was left on", row, col)
	}
}

func TestLeavingTheEditorLandsTheReadingViewOnTheCursorsLine(t *testing.T) {
	m := placeFixture(t)
	press(m, "enter")
	m.editor.MoveTo(45, 0)
	press(m, "esc")
	// The reading view is scrolled to that line, not to the top of what the
	// editor's window happened to be showing.
	if m.noteOff == 0 {
		t.Error("the reading view should follow the cursor down the note")
	}
	if m.noteAt != 45 {
		t.Errorf("the note's own cursor = %d, want the line writing stopped on (45)", m.noteAt)
	}
}

func TestEachNoteKeepsItsOwnPlace(t *testing.T) {
	m := placeFixture(t)
	press(m, "enter")
	m.editor.MoveTo(30, 2)
	press(m, "esc")
	// Off to another note, edited somewhere else entirely.
	m.open("Welcome.md")
	if m.editor != nil {
		t.Fatal("setup: the editor should be closed between notes")
	}
	press(m, "enter")
	m.editor.MoveTo(1, 4)
	press(m, "esc")
	// Back to the first: its own place, not the other note's.
	press(m, "esc")
	m.open("Long.md")
	press(m, "enter")
	if row, col := m.editor.Cursor(); row != 30 || col != 2 {
		t.Errorf("cursor = %d,%d, want 30,2", row, col)
	}
	press(m, "esc")
	m.open("Welcome.md")
	press(m, "enter")
	if row, col := m.editor.Cursor(); row != 1 || col != 4 {
		t.Errorf("cursor = %d,%d, want 1,4", row, col)
	}
}

func TestTheFirstVisitStillOpensAtTheLineYouWereReading(t *testing.T) {
	m := placeFixture(t)
	// Scroll the reading view down with the keys, then edit: with no
	// remembered place, the editor opens at the top of what was on screen,
	// as it always has.
	for i := 0; i < 20; i++ {
		press(m, "j")
	}
	if m.noteOff == 0 {
		t.Fatal("setup: the reading view should have scrolled")
	}
	want := m.lines[m.noteOff].Src
	press(m, "enter")
	if m.editor == nil {
		t.Fatal("enter should open the editor")
	}
	if row, _ := m.editor.Cursor(); row != want {
		t.Errorf("cursor line = %d, want the reading view's top line (%d)", row, want)
	}
}

func TestAPlacePastTheEndOfAShrunkenNoteIsClamped(t *testing.T) {
	m := placeFixture(t)
	press(m, "enter")
	m.editor.MoveTo(50, 1)
	press(m, "esc")
	// The note is rewritten much shorter while it isn't open — Obsidian,
	// Sync, another editor.
	writeFile(t, m.vault.Root, "Long.md", "# Long\n\nonly three lines\n")
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	m.open("Long.md")
	press(m, "enter")
	if m.editor == nil {
		t.Fatal("it should still open")
	}
	row, _ := m.editor.Cursor()
	// The shorter note is four lines, counting the empty one at the end:
	// anywhere inside it is right, line 50 is not.
	if lines := strings.Count("# Long\n\nonly three lines\n", "\n"); row > lines {
		t.Errorf("cursor line = %d, want it clamped into the shorter note (at most %d)", row, lines)
	}
}

func TestTheRememberedPlaceFollowsARename(t *testing.T) {
	m := placeFixture(t)
	press(m, "enter")
	m.editor.MoveTo(35, 0)
	press(m, "esc")
	if err := m.rename("Long.md", "Longer"); err != nil {
		t.Fatal(err)
	}
	m.open("Longer.md")
	press(m, "enter")
	if row, _ := m.editor.Cursor(); row != 35 {
		t.Errorf("cursor line = %d, want 35: the place should follow the note", row)
	}
}

func TestSavingWithoutLeavingKeepsTheCursorWhereItIs(t *testing.T) {
	m := placeFixture(t)
	press(m, "enter")
	m.editor.MoveTo(20, 2)
	press(m, "ctrl+s")
	if m.editor == nil {
		t.Fatal("ctrl+s should not close the editor")
	}
	if row, col := m.editor.Cursor(); row != 20 || col != 2 {
		t.Errorf("cursor = %d,%d: saving should not move it", row, col)
	}
}

func TestComingBackLooksLikeNothingHappened(t *testing.T) {
	m := placeFixture(t)
	press(m, "enter")
	m.editor.MoveTo(45, 0)
	m.settle()
	top := m.editor.TopRow()
	if top == 0 {
		t.Fatal("setup: the window should have scrolled to reach line 45")
	}
	press(m, "esc", "enter")
	if got := m.editor.TopRow(); got != top {
		t.Errorf("top line = %d, want %d: the window should come back too, not only the cursor", got, top)
	}
	if row, _ := m.editor.Cursor(); row != 45 {
		t.Errorf("cursor line = %d, want 45", row)
	}
}
