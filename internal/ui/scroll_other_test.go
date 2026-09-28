package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Scrolling the other half of a split, without leaving the note you are in.
// The user's own proposal, 2026-09-28: "Något som hade varit guld hade varit
// om man kunde skrolla den panel som inte var i fokus … så att man aldrig
// behövde lämna anteckningen man skriver i bara för att skrolla ner på den
// andra." Alt+↑/↓, because Alt already means the other half.

// splitFixture opens a long note beside a short one, with the focus in the
// short one, and the feature switched on.
func splitFixture(t *testing.T) *Model {
	t.Helper()
	m := newTestModelWith(t, Options{Beta: true, ScrollOther: true})
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	var b strings.Builder
	b.WriteString("# Reference\n\n")
	for i := 1; i <= 200; i++ {
		b.WriteString("reference line\n")
	}
	writeFile(t, m.vault.Root, "Reference.md", b.String())
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	m.open("Welcome.md")
	m.openSplit("Reference.md", false)
	// openSplit gives the new note the focus; swap so that the note being
	// written in is focused and Reference is the other half.
	m.swapPanes()
	m.settle()
	if m.split == nil || m.split.path != "Reference.md" {
		t.Fatalf("setup: the other half should be Reference.md, got %+v", m.split)
	}
	if m.notePath != "Welcome.md" {
		t.Fatalf("setup: the focused note should be Welcome.md, got %q", m.notePath)
	}
	return m
}

func TestAltDownScrollsTheOtherHalfAndLeavesYouWhereYouAre(t *testing.T) {
	m := splitFixture(t)
	was, wasOff, wasPath := m.split.off, m.noteOff, m.notePath
	press(m, "alt+down", "alt+down", "alt+down")
	if m.split.off <= was {
		t.Errorf("the other half didn't scroll: off = %d", m.split.off)
	}
	if m.noteOff != wasOff || m.notePath != wasPath {
		t.Errorf("the focused note moved: %q at %d", m.notePath, m.noteOff)
	}
	if m.focus != paneNote {
		t.Error("the focus should stay where it was")
	}
	press(m, "alt+up")
	if m.split.off != 2 {
		t.Errorf("off = %d, want alt+up to come back a line", m.split.off)
	}
}

func TestItWorksWhileWritingWithoutTypingAnything(t *testing.T) {
	m := splitFixture(t)
	press(m, "enter") // into the editor on Welcome.md
	if m.editor == nil {
		t.Fatal("setup: the editor should be open")
	}
	text, row, col := m.editor.Text(), 0, 0
	row, col = m.editor.Cursor()
	press(m, "alt+down", "alt+down")
	if m.split.off == 0 {
		t.Error("the other half should scroll while the editor has the focus")
	}
	if m.editor == nil {
		t.Fatal("it must not leave the editor")
	}
	if m.editor.Text() != text {
		t.Errorf("it typed something:\n%s", m.editor.Text())
	}
	if r, c := m.editor.Cursor(); r != row || c != col {
		t.Errorf("the cursor moved to %d,%d — it must stay put", r, c)
	}
}

func TestAltPageMovesTheOtherHalfAScreen(t *testing.T) {
	m := splitFixture(t)
	press(m, "alt+pgdown")
	if m.split.off < 10 {
		t.Errorf("off = %d, want a screen's worth", m.split.off)
	}
	page := m.split.off
	press(m, "alt+pgup")
	if m.split.off != 0 {
		t.Errorf("off = %d, want alt+pgup to come back a screen (was %d)", m.split.off, page)
	}
}

func TestTheOtherHalfStopsAtItsEndsAndSaysSo(t *testing.T) {
	m := splitFixture(t)
	for i := 0; i < 400; i++ {
		press(m, "alt+down")
	}
	end := m.split.off
	if end == 0 {
		t.Fatal("setup: it should have scrolled")
	}
	press(m, "alt+down")
	if m.split.off != end {
		t.Errorf("off = %d, want it to stop at the end (%d)", m.split.off, end)
	}
	if !strings.Contains(m.flash, "The end of Reference") {
		t.Errorf("flash = %q: an edge should say which edge", m.flash)
	}
	for i := 0; i < 400; i++ {
		press(m, "alt+up")
	}
	press(m, "alt+up")
	if m.split.off != 0 {
		t.Errorf("off = %d, want the top", m.split.off)
	}
	if !strings.Contains(m.flash, "The top of Reference") {
		t.Errorf("flash = %q", m.flash)
	}
}

func TestWithTheExperimentOffTheKeySaysWhereToSwitchItOn(t *testing.T) {
	m := newTestModel(t) // no beta, no scroll-other
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	writeFile(t, m.vault.Root, "Reference.md", "# Reference\n\n"+strings.Repeat("line\n", 200))
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	m.open("Welcome.md")
	m.openSplit("Reference.md", false)
	m.swapPanes()
	press(m, "alt+down")
	if m.split.off != 0 {
		t.Error("nothing should scroll while the experiment is off")
	}
	if !strings.Contains(m.flash, "beta") || !strings.Contains(m.flash, "Settings") {
		t.Errorf("flash = %q: an off experiment says where it is switched on", m.flash)
	}
}

func TestWithNoSplitTheKeyKeepsItsOldMeaning(t *testing.T) {
	// Nothing changed where there is no other half: the key still points at
	// Files, where skimming happens.
	m := newTestModelWith(t, Options{Beta: true, ScrollOther: true})
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "2") // the note pane, no split open
	press(m, "alt+down")
	if m.flash != "Go to Files (1) to skim" {
		t.Errorf("flash = %q, want the old pointer refusal", m.flash)
	}
}

func TestSkimmingInFilesIsUntouched(t *testing.T) {
	m := newTestModelWith(t, Options{Beta: true, ScrollOther: true})
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	// The same path skim_test.go walks: a folder row moves the cursor only,
	// a note opens beside. None of that changed.
	press(m, "G", "alt+k", "alt+k", "l", "alt+j", "alt+j")
	if m.split == nil {
		t.Fatal("in Files, alt+j should still open the note it lands on beside this one")
	}
	if m.split.path != "Filosofi/Stoic.md" {
		t.Errorf("the split holds %q, want the note skimmed onto", m.split.path)
	}
	if m.focus != paneFiles {
		t.Error("skimming keeps the focus in Files, as it always has")
	}
}
