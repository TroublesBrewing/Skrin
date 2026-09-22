package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// A paste doesn't arrive as a key press: it comes in as its own message,
// tea.PasteMsg, which the model hands to m.paste. The Book Card used to
// handle its keys in bookCardKey alone, so typed text landed in a card
// field and pasted text went nowhere — the card was never named in
// m.paste's switch. These tests drive the reported direction: focus a
// card field, paste, look at the field.

func TestPasteIntoTheBookCardTitle(t *testing.T) {
	m := newTestModel(t)
	press(m, "B", "tab") // search bar -> Title
	if m.book.area != bookAreaStatic || m.book.staticIdx != 0 {
		t.Fatalf("setup: area=%v idx=%d, want the Title field", m.book.area, m.book.staticIdx)
	}
	m.Update(tea.PasteMsg{Content: "Meditations"})
	if got := m.book.title.value(); got != "Meditations" {
		t.Errorf("title = %q, want the pasted text", got)
	}
}

// Every stop that takes typing has to take a paste, not just the first
// one: the report was about the card as a whole, and a field that accepts
// a keystroke but not a paste is the same half-function that v0.36.0
// fixed everywhere else.
func TestPasteReachesEveryCardFieldInTabOrder(t *testing.T) {
	m := newTestModel(t)
	press(m, "B", "tab") // Title

	type field struct {
		what  string
		value func() string
		want  string // what the field holds after the paste
	}
	// The Tab order after the search bar: the static fields in order, then
	// the first quote row's three fields, then Notes. Status starts at the
	// card's default, "reading", so its paste shows as a suffix.
	fields := []field{
		{"subtitle", m.book.subtitle.value, "y"},
		{"authors", m.book.authors.value, "y"},
		{"translators", m.book.translators.value, "y"},
		{"orig year", m.book.origYear.value, "y"},
		{"print year", m.book.editionYear.value, "y"},
		{"pages", m.book.pages.value, "y"},
		{"publisher", m.book.publisher.value, "y"},
		{"edition", m.book.edition.value, "y"},
		{"isbn", m.book.isbn.value, "y"},
		{"format", m.book.format.value, "y"},
		{"shelf", m.book.shelf.value, "y"},
		{"rating", m.book.rating.value, "y"},
		{"status", m.book.status.value, "readingy"},
		{"started", m.book.started.value, "y"},
		{"finished", m.book.finished.value, "y"},
	}
	if got := m.book.title.value(); got != "" {
		t.Fatalf("setup: title should be empty, got %q", got)
	}
	m.Update(tea.PasteMsg{Content: "x"})
	if m.book.title.value() != "x" {
		t.Fatalf("the first field should have taken the paste")
	}

	for _, f := range fields {
		press(m, "tab")
		m.Update(tea.PasteMsg{Content: "y"})
		if got := f.value(); got != f.want {
			t.Errorf("paste never reached %s: %q, want %q", f.what, got, f.want)
		}
	}

	// The quote row's text, page and speaker.
	press(m, "tab") // quote text
	m.Update(tea.PasteMsg{Content: "z"})
	if got := m.book.quotes[0].text.value(); got != "z" {
		t.Errorf("paste never reached the quote text: %q", got)
	}
	press(m, "tab") // quote page
	m.Update(tea.PasteMsg{Content: "z"})
	if got := m.book.quotes[0].page.value(); got != "z" {
		t.Errorf("paste never reached the quote page: %q", got)
	}
	press(m, "tab") // quote speaker
	m.Update(tea.PasteMsg{Content: "z"})
	if got := m.book.quotes[0].speaker.value(); got != "z" {
		t.Errorf("paste never reached the quote speaker: %q", got)
	}

	// Notes & Reflections is a textArea: the paste should land there too,
	// and keep its line breaks.
	press(m, "tab")
	if m.book.area != bookAreaNotes {
		t.Fatalf("setup: area=%v, want the Notes area", m.book.area)
	}
	m.Update(tea.PasteMsg{Content: "one\ntwo"})
	if got := m.book.notes.value(); got != "one\ntwo" {
		t.Errorf("notes = %q, want the paste with its line break", got)
	}
}

// A one-line card field turns a pasted newline into a space, exactly as
// every other one-line field in Skrin does. Notes keeps it.
func TestPasteNewlineInACardFieldBecomesASpace(t *testing.T) {
	m := newTestModel(t)
	press(m, "B", "tab")
	m.Update(tea.PasteMsg{Content: "Stoic\nMeditations"})
	if got := m.book.title.value(); got != "Stoic Meditations" {
		t.Errorf("title = %q, want the newline as a space", got)
	}
}

// Ctrl+V inside the card goes through the same path and lands in the
// focused field, rather than answering "open a note in the editor first"
// at a card that is open and focused.
func TestCtrlVPastesIntoTheOpenBookCard(t *testing.T) {
	m := newTestModel(t)
	press(m, "B", "tab")
	press(m, "ctrl+v")
	if !m.pasting {
		t.Fatal("ctrl+v should be waiting for the terminal")
	}
	m.Update(tea.ClipboardMsg{Content: "from elsewhere", Selection: 'c'})
	if got := m.book.title.value(); got != "from elsewhere" {
		t.Errorf("title = %q, want what the terminal handed back", got)
	}
}

// The Save button and the card's area headings take no text: a paste there
// neither vanishes silently nor claims to have landed.
func TestPasteOnTheSaveButtonRefusesAndSaysSo(t *testing.T) {
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
	press(m, "ctrl+v")
	m.Update(tea.ClipboardMsg{Content: "text", Selection: 'c'})
	if !strings.Contains(m.flash, "Tab to a field") {
		t.Errorf("flash = %q, want it to say where a paste can go", m.flash)
	}
	for _, f := range m.book.staticFields() {
		if f.value() != "" && f.value() != "reading" {
			t.Errorf("nothing should have been pasted: %q", f.value())
		}
	}
	if m.book.notes.value() != "" {
		t.Errorf("notes = %q, want nothing", m.book.notes.value())
	}
}

// The card keeps its own meaning when it is closed: the editor's paste
// must not be routed through a card that isn't there.
func TestPasteWithNoCardOpenStillReachesTheEditor(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "e")
	m.Update(tea.PasteMsg{Content: "hello"})
	if !strings.Contains(m.editor.Text(), "hello") {
		t.Errorf("editor = %q, want the paste", m.editor.Text())
	}
}
