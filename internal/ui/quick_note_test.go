package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestIOpensTheQuickNoteOverlay(t *testing.T) {
	m := newTestModel(t)
	press(m, "1") // Files focus
	press(m, "i")
	if m.quickNote == nil {
		t.Fatalf("i did not open the overlay; flash %q", m.flash)
	}
	if m.quickNote.area != quickNoteText {
		t.Error("the overlay opens with the text focused")
	}
	press(m, "esc")
	if m.quickNote != nil {
		t.Error("esc closes the overlay")
	}
	if m.flash != "Nothing saved" {
		t.Errorf("esc flash = %q", m.flash)
	}
}

func TestIOpensFromNoteFocusToo(t *testing.T) {
	m := newTestModel(t)
	press(m, "2") // note focus
	press(m, "i")
	if m.quickNote == nil {
		t.Fatal("i must open from anywhere in main, note focus included")
	}
}

func TestQuickNoteSavesNamedFromFirstLine(t *testing.T) {
	m := newTestModel(t)
	press(m, "i")
	typeText(m, "Idea about the thing")
	press(m, "enter")
	if m.quickNote != nil {
		t.Fatal("save should close the overlay")
	}
	rel := "Idea about the thing.md"
	src, err := m.vault.Read(rel)
	if err != nil {
		t.Fatalf("Read(%s): %v", rel, err)
	}
	if src != "Idea about the thing" {
		t.Errorf("content = %q, want the typed text byte for byte", src)
	}
	if !strings.HasPrefix(m.flash, "Saved "+rel) || !strings.Contains(m.flash, "U undoes") {
		t.Errorf("flash = %q", m.flash)
	}
	if m.journal.Len() != 1 {
		t.Errorf("journal.Len() = %d, want 1 (so U undoes the create)", m.journal.Len())
	}
}

func TestQuickNoteMultilineBodyKeepsFirstLineAsName(t *testing.T) {
	m := newTestModel(t)
	press(m, "i")
	typeText(m, "# Big idea")
	press(m, "shift+enter")
	typeText(m, "the body goes here")
	press(m, "enter")
	src, err := m.vault.Read("Big idea.md")
	if err != nil {
		t.Fatalf("Read: %v (hashes should be stripped from the name only, not the body)", err)
	}
	if src != "# Big idea\nthe body goes here" {
		t.Errorf("content = %q, want the whole typed text unchanged", src)
	}
}

func TestQuickNoteAltEnterAlsoNewlines(t *testing.T) {
	m := newTestModel(t)
	press(m, "i")
	typeText(m, "Alt test")
	press(m, "alt+enter")
	typeText(m, "second line")
	press(m, "enter")
	src, err := m.vault.Read("Alt test.md")
	if err != nil {
		t.Fatal(err)
	}
	if src != "Alt test\nsecond line" {
		t.Errorf("content = %q", src)
	}
}

func TestQuickNoteEmptyFirstLineRefuses(t *testing.T) {
	m := newTestModel(t)
	press(m, "i")
	press(m, "enter")
	if m.quickNote == nil {
		t.Fatal("an empty first line must not close the overlay")
	}
	if !strings.Contains(m.quickNote.err, "The first line becomes the note's name") {
		t.Errorf("err = %q", m.quickNote.err)
	}
}

func TestQuickNoteNameCollisionRefuses(t *testing.T) {
	m := newTestModel(t)
	if err := os.WriteFile(filepath.Join(m.vault.Root, "Taken.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	press(m, "i")
	typeText(m, "Taken")
	press(m, "enter")
	if m.quickNote == nil {
		t.Fatal("a name collision must not close the overlay")
	}
	if !strings.Contains(m.quickNote.err, "already exists") {
		t.Errorf("err = %q", m.quickNote.err)
	}
}

func TestQuickNoteNameCutAt50Chars(t *testing.T) {
	m := newTestModel(t)
	press(m, "i")
	long := strings.Repeat("a", 60)
	typeText(m, long)
	press(m, "enter")
	want := strings.Repeat("a", 50)
	if _, err := m.vault.Read(want + ".md"); err != nil {
		t.Errorf("expected a 50-char name, Read(%s): %v", want, err)
	}
}

func TestQuickNoteFolderRowDefaultsToVaultRootWhenUnset(t *testing.T) {
	m := newTestModel(t)
	press(m, "i")
	press(m, "tab") // to the folder row
	if m.quickNote.def != "" {
		t.Errorf("default = %q, want the vault root (empty)", m.quickNote.def)
	}
	press(m, "esc")
}

func TestQuickNoteFolderRowDefaultsToDeclaredFolder(t *testing.T) {
	m := newTestModel(t)
	if err := os.WriteFile(filepath.Join(m.vault.Root, ".obsidian", "app.json"),
		[]byte(`{"newFileLocation":"folder","newFileFolderPath":"Filosofi"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	press(m, "i")
	if m.quickNote.def != "Filosofi" {
		t.Errorf("default = %q, want Filosofi (the declared folder)", m.quickNote.def)
	}
	typeText(m, "Declared folder test")
	press(m, "enter")
	if _, err := m.vault.Read("Filosofi/Declared folder test.md"); err != nil {
		t.Errorf("note wasn't saved into the declared folder: %v", err)
	}
}

func TestQuickNoteFolderRowFallsBackWhenDeclaredFolderIsGone(t *testing.T) {
	m := newTestModel(t)
	if err := os.WriteFile(filepath.Join(m.vault.Root, ".obsidian", "app.json"),
		[]byte(`{"newFileLocation":"folder","newFileFolderPath":"Nonexistent"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	press(m, "i")
	if m.quickNote.def != "" {
		t.Errorf("default = %q, want the vault root: a declared-but-deleted folder falls back", m.quickNote.def)
	}
}

func TestQuickNoteTabFiltersAndPicksAFolder(t *testing.T) {
	m := newTestModel(t)
	press(m, "i")
	typeText(m, "Filed away")
	press(m, "tab") // to the folder row
	typeText(m, "Filosofi")
	press(m, "enter") // pick the highlighted match
	if m.quickNote.area != quickNoteText {
		t.Error("picking a folder returns focus to the text")
	}
	press(m, "enter") // save
	if _, err := m.vault.Read("Filosofi/Filed away.md"); err != nil {
		t.Errorf("note wasn't saved into the picked folder: %v", err)
	}
}

func TestQuickNoteTypedFolderThatIsntOneRefuses(t *testing.T) {
	m := newTestModel(t)
	press(m, "i")
	typeText(m, "Anteckningar test")
	press(m, "tab")
	typeText(m, "Anteckningar")
	press(m, "enter") // no such folder matches: refuse, don't pick garbage
	if m.quickNote == nil || m.quickNote.area != quickNoteFolder {
		t.Fatal("the refusal must keep the overlay open on the folder row")
	}
	if !strings.Contains(m.quickNote.err, "No folder Anteckningar/") {
		t.Errorf("err = %q", m.quickNote.err)
	}
}

func TestQuickNoteUndoRemovesTheSavedNote(t *testing.T) {
	m := newTestModel(t)
	press(m, "i")
	typeText(m, "Undo me")
	press(m, "enter")
	if !m.vault.Exists("Undo me.md") {
		t.Fatal("note wasn't created")
	}
	press(m, "U")
	if m.vault.Exists("Undo me.md") {
		t.Error("U should undo the quick note's creation")
	}
}

func TestQuickNoteDoesNotOpenTheSavedNote(t *testing.T) {
	m := newTestModel(t)
	before := m.notePath
	press(m, "i")
	typeText(m, "Stay put")
	press(m, "enter")
	if m.notePath != before {
		t.Errorf("notePath changed to %q; quick notes must not open after saving", m.notePath)
	}
}

func TestQuickNoteFrameFitsAtEverySize(t *testing.T) {
	m := newTestModel(t)
	press(m, "i")
	typeText(m, "Sizing")
	for _, size := range sizes {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		checkFrame(t, m, "quick note at "+itoa(size[0])+"x"+itoa(size[1]))
	}
}
