package ui

import (
	"strings"
	"testing"

	"github.com/lurioso/skrin/internal/version"
)

func outgoingModel(t *testing.T) *Model {
	t.Helper()
	if !version.Beta {
		t.Skip("a release build leaves every experiment out")
	}
	return newTestModelWith(t, Options{RolloverTodos: true, Beta: true, Outgoing: true})
}

// Welcome links to Stoic and to a note that doesn't exist yet. The panel
// answers "what does this note point at", the mirror of b.
func TestOutgoingListsWhatTheNoteLinksTo(t *testing.T) {
	m := outgoingModel(t)
	m.open("Welcome.md")
	if m.notePath != "Welcome.md" {
		t.Fatalf("opened %q", m.notePath)
	}
	m.showOutgoing()
	if m.chooser == nil {
		t.Fatalf("no panel; flash %q", m.flash)
	}
	if !strings.Contains(m.chooser.title, "Links from Welcome") {
		t.Errorf("title %q", m.chooser.title)
	}
	var labels []string
	for _, it := range m.chooser.items {
		labels = append(labels, it.label)
	}
	if !strings.Contains(strings.Join(labels, " "), "Stoic") {
		t.Errorf("items %v should hold the note it links to", labels)
	}
}

// A note that links nowhere says so rather than opening an empty panel.
func TestOutgoingSaysSoWhenANoteLinksNowhere(t *testing.T) {
	m := outgoingModel(t)
	m.open("Daily/2026-09-11.md")
	m.showOutgoing()
	if m.chooser != nil {
		t.Fatal("an empty panel is worse than a sentence")
	}
	if !strings.Contains(m.flash, "doesn't link anywhere") {
		t.Errorf("flash %q", m.flash)
	}
}

// Off, the panel says where it is switched on, and the palette doesn't
// carry the row at all.
func TestOutgoingIsOffUntilAskedFor(t *testing.T) {
	m := newTestModel(t)
	m.open("Welcome.md")
	m.showOutgoing()
	if m.chooser != nil {
		t.Fatal("an experiment that is off may not open a panel")
	}
	if !strings.Contains(m.flash, "beta") {
		t.Errorf("flash %q", m.flash)
	}
	for _, it := range m.mainPaletteItems() {
		if strings.Contains(it.label, "Outgoing links") {
			t.Error("the palette shouldn't carry a switched-off experiment")
		}
	}
}

func TestOutgoingIsInThePaletteWhenOn(t *testing.T) {
	m := outgoingModel(t)
	var found bool
	for _, it := range m.mainPaletteItems() {
		if strings.Contains(it.label, "Outgoing links") {
			found = true
		}
	}
	if !found {
		t.Error("switched on, it should be findable by name")
	}
}
