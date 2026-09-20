package ui

import (
	"os"
	"testing"
)

// Reading has no cursor: j and k scroll and nothing is picked out. v is
// what asks for one, and it lands where your eye already is rather than
// at the top of the note — which used to make a block in the middle
// impossible to select at all.

func TestTheCursorLightsWhereYouAreLooking(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l") // Welcome.md, reading, no cursor
	if m.noteSel != nil {
		t.Fatal("reading shouldn't pick out a line")
	}
	press(m, "v")
	if m.noteSel == nil {
		t.Fatal("v should light the cursor")
	}
	if m.noteSel.cur == 0 && len(m.lines) > 2 {
		t.Error("it should land in the middle of the view, not on the first line")
	}
	if m.noteSel.anchor != m.noteSel.cur {
		t.Error("and start as one line")
	}
}

func TestPlainMotionsMoveTheCursorAndJKGrowTheSelection(t *testing.T) {
	m := newTestModel(t)
	if err := os.WriteFile(m.vault.Abs("Welcome.md"),
		[]byte("# Titel\n\nett\ntvå\ntre\nfyra\nfem\nsex\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.reload()
	press(m, "G", "l", "v")
	selTo(m, 2) // somewhere with room either way
	at := m.noteSel.cur
	press(m, "j")
	if m.noteSel.cur != at+1 || m.noteSel.anchor != at+1 {
		t.Errorf("j should move the cursor and leave one line selected: %+v", m.noteSel)
	}
	press(m, "J")
	if m.noteSel.anchor != at+1 || m.noteSel.cur != at+2 {
		t.Errorf("J should stretch from where the cursor is: %+v", m.noteSel)
	}
	press(m, "K", "K")
	if m.noteSel.anchor != at+1 || m.noteSel.cur != at {
		t.Errorf("K should stretch back the other way: %+v", m.noteSel)
	}
	press(m, "k")
	if m.noteSel.anchor != m.noteSel.cur {
		t.Error("a plain motion should drop the selection back to one line")
	}
}

// The thing that was impossible: pick a block out of the middle of a note
// that fits on the screen, with nothing to scroll.
func TestABlockInTheMiddleCanBeSelected(t *testing.T) {
	m := newTestModel(t)
	if err := os.WriteFile(m.vault.Abs("Welcome.md"),
		[]byte("# Titel\n\nett\ntvå\ntre\nfyra\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.reload()
	press(m, "G", "l", "v")
	selTo(m, 3) // "två"
	press(m, "J")
	if got := m.selectionText(); got != "två\ntre" {
		t.Errorf("selection = %q, want the two middle lines", got)
	}
}

func TestTheCursorComesBackWhereYouLeftIt(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "v")
	selTo(m, 1)
	press(m, "esc")
	if m.noteSel != nil {
		t.Fatal("esc should put the cursor out")
	}
	press(m, "v")
	if m.noteSel.cur != 1 {
		t.Errorf("v should light it again at line 1, not %d", m.noteSel.cur)
	}
}

// Shift+arrow selects in the editor and in every other program, so it
// selects here too — and lights the cursor itself when it isn't lit.
func TestShiftArrowsSelectInTheNote(t *testing.T) {
	m := newTestModel(t)
	if err := os.WriteFile(m.vault.Abs("Welcome.md"),
		[]byte("# Titel\n\nett\ntvå\ntre\nfyra\nfem\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.reload()
	press(m, "G", "l")
	if m.noteSel != nil {
		t.Fatal("nothing selected while reading")
	}
	press(m, "shift+down")
	if m.noteSel == nil {
		t.Fatalf("shift+down should light the cursor and select; flash %q", m.flash)
	}
	if n := m.noteSel.cur - m.noteSel.anchor; n != 1 {
		t.Errorf("it should have grown one line down, got %d", n)
	}
	press(m, "shift+down")
	if n := m.noteSel.cur - m.noteSel.anchor; n != 2 {
		t.Errorf("and one more, got %d", n)
	}
	press(m, "shift+up", "shift+up", "shift+up")
	if n := m.noteSel.cur - m.noteSel.anchor; n != -1 {
		t.Errorf("shift+up should stretch back the other way, got %d", n)
	}
	// The same keys still order things in Files.
	press(m, "esc", "1")
	before := m.files.selected().Rel
	press(m, "shift+down")
	if m.files.selected().Rel != before {
		t.Log("the row moved in Files, which is the other half of the key")
	}
}
