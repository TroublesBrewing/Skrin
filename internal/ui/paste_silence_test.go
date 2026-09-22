package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// A terminal paste (Ctrl+Shift+V) refuses out loud where Ctrl+V does: a
// paste that lands nowhere must never be silent, whether it came in as a
// key or as the terminal's own bracketed paste.
func TestATerminalPasteThatLandsNowhereSaysSo(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l") // reading, with no card open and nothing typing
	m.Update(tea.PasteMsg{Content: "zzq"})
	if !strings.Contains(m.flash, "Nothing here takes text") {
		t.Errorf("flash = %q, want it to say the paste landed nowhere", m.flash)
	}
	// The note must be untouched. (The flash itself says "takes text", so
	// the whole render can't be searched for the payload's word.)
	if strings.Contains(read(m, "Welcome.md"), "zzq") {
		t.Error("nothing should have been written to the note")
	}
}

// ...and inside the card, where the only stops that take no text are the
// Save button and the area headings.
func TestATerminalPasteOnTheSaveButtonSaysWhereToGo(t *testing.T) {
	m := newTestModel(t)
	press(m, "B")
	for i := 0; i < 30; i++ {
		press(m, "tab")
		if m.book.area == bookAreaSave {
			break
		}
	}
	if m.book.area != bookAreaSave {
		t.Fatalf("setup: area=%v, want Save", m.book.area)
	}
	m.Update(tea.PasteMsg{Content: "text"})
	if !strings.Contains(m.flash, "Tab to a field") {
		t.Errorf("flash = %q, want it to say where a paste can go", m.flash)
	}
}

// A paste that does land keeps the quiet it always had: no flash, just the
// text in the field.
func TestAPasteThatLandsNeedsNoFlash(t *testing.T) {
	m := newTestModel(t)
	press(m, "B", "tab")
	m.Update(tea.PasteMsg{Content: "gone in"})
	if m.book.title.value() != "gone in" {
		t.Errorf("title = %q", m.book.title.value())
	}
	if m.flash != "" {
		t.Errorf("flash = %q, want silence on a paste that worked", m.flash)
	}
}
