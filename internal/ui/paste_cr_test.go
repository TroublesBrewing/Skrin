package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// A terminal paste sends CR for a line break — the Enter key, not LF.
// Treating CR as anything but a line break collapses a pasted paragraph
// onto one line, so the tests use CR, the way a terminal actually sends
// it; an LF-only test passes while the real thing is broken.
func TestACRTerminalPasteIntoNotesKeepsItsLines(t *testing.T) {
	m := newTestModel(t)
	press(m, "B")
	for i := 0; i < 20; i++ {
		press(m, "tab")
		if m.book.area == bookAreaNotes {
			break
		}
	}
	if m.book.area != bookAreaNotes {
		t.Fatalf("setup: area=%v, want Notes", m.book.area)
	}
	m.Update(tea.PasteMsg{Content: "LINE-ONE\rLINE-TWO"})
	if got, want := m.book.notes.value(), "LINE-ONE\nLINE-TWO"; got != want {
		t.Errorf("notes = %q, want %q", got, want)
	}
}

// The same for CRLF, which is what a paste from another application often
// carries: one break, not two.
func TestACRLFTerminalPasteIntoNotesKeepsItsLines(t *testing.T) {
	m := newTestModel(t)
	press(m, "B")
	for i := 0; i < 20; i++ {
		press(m, "tab")
		if m.book.area == bookAreaNotes {
			break
		}
	}
	m.Update(tea.PasteMsg{Content: "ONE\r\nTWO"})
	if got, want := m.book.notes.value(), "ONE\nTWO"; got != want {
		t.Errorf("notes = %q, want %q", got, want)
	}
}

// In a one-line card field a CR becomes a space, like every other line
// break there.
func TestACRInAOneLineCardFieldBecomesASpace(t *testing.T) {
	m := newTestModel(t)
	press(m, "B", "tab")
	m.Update(tea.PasteMsg{Content: "Stoic\rMeditations"})
	if got := m.book.title.value(); got != "Stoic Meditations" {
		t.Errorf("title = %q, want the break as a space", got)
	}
}
