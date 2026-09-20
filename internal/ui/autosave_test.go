package ui

import (
	"os"
	"strings"
	"testing"
)

// The editor writes by itself a moment after you type, without saying
// anything, and leaves every question that belongs to leaving the note
// where it was.

func TestTypingStartsTheClockAndTheTickWrites(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "e")
	m.editor.MoveTo(1, 0)
	typeText(m, "fresh")
	if !m.autosaving {
		t.Fatal("typing should start the clock")
	}
	if strings.Contains(read(m, "Welcome.md"), "fresh") {
		t.Fatal("nothing should be on disk before the tick")
	}
	m.Update(autosaveMsg{})
	if !strings.Contains(read(m, "Welcome.md"), "fresh") {
		t.Errorf("the tick should write: %q", read(m, "Welcome.md"))
	}
	if m.flash != "" {
		t.Errorf("saving by itself should say nothing: %q", m.flash)
	}
	if m.editor == nil {
		t.Error("it should still be your editor, open where you were")
	}
	if m.editor.Dirty() {
		t.Error("the ● should be gone once it's on disk")
	}
}

func TestTheClockRunsAgainWhileThereIsMoreToWrite(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "e")
	m.editor.MoveTo(1, 0)
	typeText(m, "one")
	m.Update(autosaveMsg{})
	if m.autosaving {
		t.Error("with everything on disk, no clock should be running")
	}
	typeText(m, " two")
	if !m.autosaving {
		t.Fatal("typing again should start it again")
	}
	m.Update(autosaveMsg{})
	if got := read(m, "Welcome.md"); !strings.Contains(got, "one two") {
		t.Errorf("Welcome = %q", got)
	}
}

func TestSavingByItselfNeverRaisesTheConflict(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "e")
	m.editor.MoveTo(1, 0)
	typeText(m, "mine")
	// Obsidian, Sync or another editor gets there first.
	if err := os.WriteFile(m.vault.Abs("Welcome.md"), []byte("theirs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(autosaveMsg{})
	if m.conflict != nil {
		t.Fatal("a save by itself mustn't interrupt with the conflict")
	}
	if got := read(m, "Welcome.md"); got != "theirs\n" {
		t.Errorf("it should write nothing and wait: %q", got)
	}
	if !m.editor.Dirty() {
		t.Error("what you typed still isn't on disk, so the ● stays")
	}
	if !m.edit.held {
		t.Error("the hold should be on the record, not only in this moment")
	}
	if !strings.Contains(m.flash, "changed on disk") {
		t.Errorf("it should say so at once: %q", m.flash)
	}
	if line := m.editLine(); !strings.Contains(line, "HELD") ||
		!strings.Contains(line, "nothing of yours is written") {
		t.Errorf("and go on saying so while you type: %q", line)
	}
	press(m, "ctrl+s") // the moment you chose
	if m.conflict == nil {
		t.Error("an explicit save is where the question belongs")
	}
	press(m, "m") // keep mine
	if m.edit.held {
		t.Error("answering ends the hold")
	}
	if got := read(m, "Welcome.md"); !strings.Contains(got, "mine") {
		t.Errorf("and writes what you typed: %q", got)
	}
}

func TestOneSnapshotHoldsForTheWholeSession(t *testing.T) {
	m := newTestModel(t)
	before := read(m, "Welcome.md")
	press(m, "G", "l", "e")
	m.editor.MoveTo(1, 0)
	for _, s := range []string{"a", "b", "c"} {
		typeText(m, s)
		m.Update(autosaveMsg{})
	}
	press(m, "esc")
	press(m, "u")
	if got := read(m, "Welcome.md"); got != before {
		t.Errorf("u should reach past every save by itself, to %q, got %q", before, got)
	}
}

func TestLeavingTheNoteKeepsItsOwnWork(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "e")
	m.editor.MoveTo(1, 0)
	typeText(m, "- [ ] call back [due:: tomorrow]")
	m.Update(autosaveMsg{})
	if got := read(m, "Welcome.md"); !strings.Contains(got, "[due:: tomorrow]") {
		t.Errorf("a save by itself shouldn't resolve dates: %q", got)
	}
	press(m, "esc")
	if got := read(m, "Welcome.md"); !strings.Contains(got, "[due:: 2026-09-16]") {
		t.Errorf("leaving the note should resolve it: %q", got)
	}
}

func TestSavingByItselfAsksNothingAboutHeadings(t *testing.T) {
	m := headingLinkModel(t) // "## Morning" renamed to "## Sunrise", links elsewhere
	m.Update(autosaveMsg{})
	if m.confirm != nil {
		t.Fatal("the link question belongs to leaving the note")
	}
	if !strings.Contains(read(m, "Filosofi/Stoic.md"), "## Sunrise") {
		t.Error("the text itself should be on disk")
	}
	if got := read(m, "Welcome.md"); !strings.Contains(got, "[[Stoic#Morning]]") {
		t.Error("and no link should have moved yet")
	}
	press(m, "esc")
	if m.confirm == nil {
		t.Error("leaving the note should still ask")
	}
}

func TestTheClockIsHarmlessOnceTheEditorIsClosed(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "e")
	m.editor.MoveTo(1, 0)
	typeText(m, "typed")
	press(m, "esc") // saves and closes
	m.Update(autosaveMsg{})
	if m.flash == "Couldn't save by itself" || m.editor != nil {
		t.Errorf("a late tick should do nothing: flash %q", m.flash)
	}
	if !strings.Contains(read(m, "Welcome.md"), "typed") {
		t.Error("closing saved it, as it always did")
	}
}
