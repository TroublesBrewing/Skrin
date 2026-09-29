package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// The terminal window closes while a sentence is half typed. What is in
// the editor has to reach the file, because the process is about to stop
// existing.
func TestShutdownWritesWhatTheEditorHolds(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "e") // edit Stoic
	if m.editor == nil {
		t.Fatal("the editor should be open")
	}
	typeText(m, "half a thought")
	if !m.editor.Dirty() {
		t.Fatal("the editor should be holding unsaved text")
	}

	_, cmd := m.Update(ShutdownMsg{})
	if cmd == nil {
		t.Fatal("a shutdown should end the program")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("no message from the shutdown command")
	} else if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("shutdown should quit, got %T", msg)
	}
	disk, err := m.vault.Read("Filosofi/Stoic.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(disk, "half a thought") {
		t.Error("the text never reached the file")
	}
}

// Nothing to save is not a reason to hesitate: it still quits.
func TestShutdownWithNothingOpenJustQuits(t *testing.T) {
	m := newTestModel(t)
	if m.shutdown() {
		t.Error("nothing was open to write")
	}
	_, cmd := m.Update(ShutdownMsg{})
	if cmd == nil {
		t.Fatal("it should still quit")
	}
}

// Being shut down is no reason to overwrite a note that changed on disk
// underneath. The same rule autosave follows holds here: it is the user's
// decision, and it waits for them, even if waiting means this text is
// only in the editor.
func TestShutdownDoesNotOverwriteANoteThatChangedOnDisk(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "e")
	typeText(m, "mine")
	if err := m.vault.Write("Filosofi/Stoic.md", "someone else's\n"); err != nil {
		t.Fatal(err)
	}
	if m.shutdown() {
		t.Error("it should not have written")
	}
	disk, err := m.vault.Read("Filosofi/Stoic.md")
	if err != nil {
		t.Fatal(err)
	}
	if disk != "someone else's\n" {
		t.Errorf("the other writer's text was overwritten: %q", disk)
	}
}
