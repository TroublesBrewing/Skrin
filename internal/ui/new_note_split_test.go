package ui

import (
	"strings"
	"testing"
)

// Tab in a new-note prompt picks the folder the note lands in, instead of
// the one the cursor is standing in — the design miss the user hit reading
// in one folder and wanting to write in another.
func TestNewNotePicksAFolderWithTab(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)      // cursor on Antik/, cwd Filosofi
	press(m, "j", "n") // on Stoic.md, cwd Filosofi
	if m.prompt == nil || m.prompt.folder != "Filosofi" {
		t.Fatalf("prompt = %+v, want folder Filosofi", m.prompt)
	}
	press(m, "tab")
	if m.chooser == nil || m.chooser.title != "New note in" {
		t.Fatalf("tab should open the folder picker, got %+v", m.chooser)
	}
	// Pick the root: filter to it and choose.
	typeText(m, "/")
	press(m, "enter")
	if m.prompt == nil || m.prompt.folder != "" || !strings.Contains(m.prompt.label, m.vault.Name()+"/") {
		t.Fatalf("after picking the root: prompt = %+v", m.prompt)
	}
	typeText(m, "Vid roten")
	press(m, "enter")
	if m.prompt != nil {
		t.Fatalf("prompt not closed: err %q", m.prompt.err)
	}
	if !m.vault.Exists("Vid roten.md") {
		t.Fatalf("the note should land in the root, not Filosofi")
	}
	if m.editor == nil || m.edit.rel != "Vid roten.md" {
		t.Fatalf("the note should open in the editor, rel %q", m.edit.rel)
	}
}

// Esc from the folder picker leaves the name and folder as they were.
func TestNewNoteFolderPickerCancels(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "n", "tab")
	if m.chooser == nil {
		t.Fatal("setup: no picker")
	}
	press(m, "esc")
	if m.prompt == nil || m.prompt.folder != "Filosofi" {
		t.Fatalf("esc should restore the prompt unchanged: %+v", m.prompt)
	}
}

// Alt+n opens a new note in a split beside the one being read, so the
// reference note stays put while the new one is written.
func TestNewNoteSplitKeepsTheReferenceBeside(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l") // Welcome.md open for reading
	if m.notePath != "Welcome.md" {
		t.Fatalf("setup: open %q", m.notePath)
	}
	press(m, "alt+n")
	if m.prompt == nil || !m.prompt.split {
		t.Fatalf("alt+n should ask for a split note, prompt %+v", m.prompt)
	}
	typeText(m, "Läsanteckning")
	press(m, "enter")
	if m.prompt != nil {
		t.Fatalf("prompt not closed: %q", m.prompt.err)
	}
	if m.split == nil || m.split.path != "Welcome.md" {
		t.Fatalf("the note being read should become the split, split = %+v", m.split)
	}
	if m.notePath != "Läsanteckning.md" || m.editor == nil {
		t.Fatalf("the new note should be open in the editor: note %q, editor %v", m.notePath, m.editor != nil)
	}
}

// Alt+n with no note open falls back to an ordinary new note, so the key
// is never a dead end.
func TestNewNoteSplitFallsBackWithoutANote(t *testing.T) {
	m := newTestModel(t) // no note open
	press(m, "alt+n")
	if m.prompt == nil || m.prompt.split {
		t.Fatalf("with no note open, alt+n should be an ordinary new note: %+v", m.prompt)
	}
}

// The split note lands in the folder picked with Tab too, so the two
// features compose.
func TestNewNoteSplitPicksAFolderToo(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l") // Welcome.md
	press(m, "alt+n")
	press(m, "tab")
	typeText(m, "filosofi")
	press(m, "enter")
	typeText(m, "I Filosofi")
	press(m, "enter")
	if m.prompt != nil {
		t.Fatalf("prompt not closed: %q", m.prompt.err)
	}
	if !m.vault.Exists("Filosofi/I Filosofi.md") {
		t.Fatal("the split note should land in the picked folder")
	}
	if m.split == nil || m.split.path != "Welcome.md" {
		t.Fatalf("split = %+v", m.split)
	}
}
