package ui

import (
	"os"
	"strings"
	"testing"
)

// level lists folder dir's entries in the order Files shows them.
func level(m *Model, dir string) string { return strings.Join(m.files.childNames(dir), ",") }

func TestShiftMovesTheItemUnderTheCursor(t *testing.T) {
	m := newTestModel(t)
	press(m, "1", "j", "shift+down") // on Daily, at the top level
	if got := level(m, ""); got != "Filosofi,Daily,Templates,Welcome.md" {
		t.Fatalf("shift+down: %s", got)
	}
	if m.files.selected().Rel != "Daily" {
		t.Error("the cursor should stay on the item it moved")
	}
	if !strings.Contains(m.flash, "Moved Daily down") || !strings.Contains(m.flash, "U undoes") {
		t.Errorf("a move should say so and say how to take it back: %q", m.flash)
	}
	checkFrame(t, m, "after a move")
	press(m, "G", "shift+up", "shift+up") // Welcome up past Templates and Daily
	if got := level(m, ""); got != "Filosofi,Welcome.md,Daily,Templates" {
		t.Errorf("two shift+up: %s", got)
	}
	press(m, "home", "shift+up")
	if !strings.Contains(m.flash, "vault row") {
		t.Errorf("the vault row can't move: flash %q", m.flash)
	}
	press(m, "j", "shift+up")
	if !strings.Contains(m.flash, "Already at the top") {
		t.Errorf("shift+up at the top of a level: flash %q", m.flash)
	}
	press(m, "2", "shift+down")
	if !strings.Contains(m.flash, "Files") {
		t.Errorf("moving an item is a Files key, and should say so: %q", m.flash)
	}
}

func TestFileKeysKeepWorkingWhileArranging(t *testing.T) {
	m := newTestModel(t)
	press(m, "1", "j", "shift+down", "n") // no mode to pause them any more
	if m.prompt == nil {
		t.Fatal("n should still start a new note")
	}
	press(m, "esc")
	press(m, "shift+up")
	if got := level(m, ""); got != "Daily,Filosofi,Templates,Welcome.md" {
		t.Errorf("shift+up should still move the item: %s", got)
	}
}

func TestEachMoveIsItsOwnUndoStep(t *testing.T) {
	m := newTestModel(t)
	press(m, "1", "j", "shift+down")
	if !strings.Contains(read(m, ".skrin"), `"/": [`) {
		t.Fatalf(".skrin = %q", read(m, ".skrin"))
	}
	press(m, "shift+down")
	if got := level(m, ""); got != "Filosofi,Templates,Daily,Welcome.md" {
		t.Fatalf("a second shift+down: %s", got)
	}
	press(m, "U")
	if got := level(m, ""); got != "Filosofi,Daily,Templates,Welcome.md" {
		t.Errorf("U should take back the last move only: %s", got)
	}
	press(m, "U")
	if got := level(m, ""); got != "Daily,Filosofi,Templates,Welcome.md" || m.vault.Exists(".skrin") {
		t.Errorf("undoing the first move should take .skrin away again: %s", got)
	}
}

func TestOrderKeepsThroughChangesAndResets(t *testing.T) {
	m := newTestModel(t)
	press(m, "1", "j", "j", "l", "j", "shift+down") // in Filosofi: Antik below Stoic
	if got := level(m, "Filosofi"); got != "Stoic.md,Antik" {
		t.Fatalf("shift+down: %s", got)
	}
	if err := os.WriteFile(m.vault.Abs("Filosofi/Aaa.md"), []byte("# new"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	if got := level(m, "Filosofi"); got != "Stoic.md,Antik,Aaa.md" {
		t.Errorf("a new note should come last in an ordered level: %s", got)
	}
	press(m, "k", "r", "ctrl+u") // Stoic
	typeText(m, "Stoa")
	press(m, "enter", "y")
	if got := level(m, "Filosofi"); got != "Stoa.md,Antik,Aaa.md" {
		t.Errorf("a renamed note should keep its place: %s", got)
	}
	press(m, "U")
	if got := level(m, "Filosofi"); got != "Stoic.md,Antik,Aaa.md" {
		t.Errorf("U should undo the rename, place and all: %s", got)
	}
	press(m, "R")
	if got := level(m, "Filosofi"); got != "Antik,Aaa.md,Stoic.md" {
		t.Errorf("R should put the level back in the default order: %s", got)
	}
	if !strings.Contains(m.flash, "default order") {
		t.Errorf("R should say what it did: %q", m.flash)
	}
	press(m, "R")
	if !strings.Contains(m.flash, "already in the default order") {
		t.Errorf("R on an unordered level: %q", m.flash)
	}
	press(m, "U")
	if got := level(m, "Filosofi"); got != "Stoic.md,Antik,Aaa.md" {
		t.Errorf("U should bring the order back after R: %s", got)
	}
	if err := os.WriteFile(m.vault.Abs(".skrin"), []byte(`{"Filosofi/": ["Stoic.md", "Aaa.md", "Antik"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	if got := level(m, "Filosofi"); got != "Stoic.md,Aaa.md,Antik" {
		t.Errorf("an order written by hand should show: %s", got)
	}
	if err := os.WriteFile(m.vault.Abs(".skrin"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	if got := level(m, "Filosofi"); got != "Antik,Aaa.md,Stoic.md" {
		t.Errorf("a broken .skrin should mean the default order: %s", got)
	}
}
