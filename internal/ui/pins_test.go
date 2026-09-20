package ui

import (
	"os"
	"strings"
	"testing"
)

// goToPin opens a note by name and pins it.
func goToPin(m *Model, name string) {
	press(m, "g")
	typeText(m, name)
	press(m, "enter")
	press(m, "p")
}

// Pins are Obsidian's bookmarks: the notes you live in, as opposed to the
// ones you were last in, which is what Go to note already offers.

func TestPinningAndUnpinningTheNoteYouAreOn(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l") // reading Welcome.md
	press(m, "p")
	if len(m.pinned) != 1 || m.pinned[0] != "Welcome.md" {
		t.Fatalf("pinned = %v, flash %q", m.pinned, m.flash)
	}
	if !strings.Contains(m.flash, "Pinned Welcome") || !strings.Contains(m.flash, "P lists them") {
		t.Errorf("flash = %q", m.flash)
	}
	press(m, "p")
	if len(m.pinned) != 0 || !strings.Contains(m.flash, "Unpinned") {
		t.Errorf("the same key should take it off: %v, %q", m.pinned, m.flash)
	}
}

func TestThePinnedListOpensTheNoteYouPick(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "p") // pin Welcome
	goToPin(m, "Zeno")      // and one more
	press(m, "P")
	if m.chooser == nil {
		t.Fatalf("P should list the pins; flash %q", m.flash)
	}
	if n := len(m.chooser.items); n != 2 {
		t.Fatalf("items = %d, want 2", n)
	}
	typeText(m, "Welcome")
	press(m, "enter")
	if m.notePath != "Welcome.md" {
		t.Errorf("picked %q", m.notePath)
	}
}

func TestAnEmptyPinListSaysHowToFillIt(t *testing.T) {
	m := newTestModel(t)
	press(m, "P")
	if m.chooser != nil || !strings.Contains(m.flash, "p pins the one you're on") {
		t.Errorf("chooser %v, flash %q", m.chooser != nil, m.flash)
	}
}

func TestPinsSurviveARestart(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "p")
	goToPin(m, "Zeno")
	s := m.Session()
	if len(s.Pinned) != 2 {
		t.Fatalf("the session should carry them: %v", s.Pinned)
	}
	again := newTestModelWith(t, Options{Session: s})
	if len(again.pinned) != 2 {
		t.Errorf("pins after a restart: %v", again.pinned)
	}
}

func TestAPinToANoteThatIsGoneFallsOut(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "p")
	goToPin(m, "Zeno")
	if err := os.Remove(m.vault.Abs("Welcome.md")); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{}) // Obsidian, Sync or a delete elsewhere
	if len(m.pinned) != 1 || strings.Contains(m.pinned[0], "Welcome") {
		t.Errorf("the list must never point at nothing: %v", m.pinned)
	}
}

func TestPinningNeedsANote(t *testing.T) {
	m := newTestModel(t) // the vault root, a folder under the cursor
	press(m, "p")
	if !strings.Contains(m.flash, "Select a note to pin") {
		t.Errorf("flash = %q", m.flash)
	}
}
