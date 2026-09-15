package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// inFilosofi opens Filosofi in Files and puts the cursor on its first entry,
// Antik/. One j further is Stoic.
func inFilosofi(m *Model) { press(m, "1", "j", "j", "l") }

func TestCreateNoteInCurrentFolderAndUndo(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "n")
	if m.prompt == nil || !strings.Contains(m.prompt.label, "Filosofi/") {
		t.Fatalf("prompt = %+v, want one naming Filosofi/", m.prompt)
	}
	typeText(m, "Ny tanke")
	press(m, "enter")
	if m.prompt != nil || !m.vault.Exists("Filosofi/Ny tanke.md") {
		t.Fatalf("note not created (prompt error %q)", m.prompt.err)
	}
	if m.editor == nil || m.edit.rel != "Filosofi/Ny tanke.md" {
		t.Fatal("a new note should open in the editor")
	}
	press(m, "esc")
	if got := m.files.selected().Rel; got != "Filosofi/Ny tanke.md" {
		t.Errorf("cursor on %q, want the new note", got)
	}

	press(m, "n", "enter", "esc", "n", "enter", "esc")
	if !m.vault.Exists("Filosofi/Untitled.md") || !m.vault.Exists("Filosofi/Untitled 1.md") {
		t.Error("empty names should become Untitled, Untitled 1")
	}
	press(m, "U")
	if m.vault.Exists("Filosofi/Untitled 1.md") || !m.vault.Exists("Filosofi/Untitled.md") {
		t.Error("U should undo only the last create")
	}
}

func TestNewNoteOnAFolderGoesInside(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m) // on Antik/
	press(m, "n")
	if m.prompt == nil || !strings.Contains(m.prompt.label, "Filosofi/Antik/") {
		t.Fatalf("prompt = %+v, want one naming Filosofi/Antik/", m.prompt)
	}
}

func TestNewNoteInNewSubfolders(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "n")
	typeText(m, "Stoa/Seneca/Brev")
	press(m, "enter")
	if !m.vault.Exists("Filosofi/Stoa/Seneca/Brev.md") || m.cwd() != "Filosofi/Stoa/Seneca" {
		t.Fatalf("nested create: cwd %q", m.cwd())
	}
	press(m, "esc", "U")
	if m.vault.Exists("Filosofi/Stoa") {
		t.Error("undo should remove the folders it created too")
	}
	if m.cwd() != "Filosofi" {
		t.Errorf("after undo cwd = %q, want the nearest surviving folder", m.cwd())
	}
}

func TestFolderSlashMakesUntitledNoteInside(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "n")
	typeText(m, "Anteckningar/")
	press(m, "enter")
	if m.prompt != nil || !m.vault.Exists("Filosofi/Anteckningar/Untitled.md") || m.editor == nil {
		t.Fatalf("Anteckningar/ should give Anteckningar/Untitled.md, open in the editor (prompt %+v)", m.prompt)
	}
	press(m, "esc", "n", "enter", "esc")
	if !m.vault.Exists("Filosofi/Anteckningar/Untitled 1.md") {
		t.Error("a second untitled note in the same folder should be Untitled 1")
	}
	press(m, "U", "U")
	if m.vault.Exists("Filosofi/Anteckningar") {
		t.Error("undo should remove the new folder too")
	}
	press(m, "N")
	typeText(m, "Arkiv/")
	press(m, "enter")
	if !m.vault.IsDir("Filosofi/Arkiv") {
		t.Error("N with a trailing slash should still make the folder")
	}
}

func TestBadNameKeepsPromptOpen(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "n")
	typeText(m, "a:b")
	press(m, "enter")
	if m.prompt == nil || m.prompt.err == "" {
		t.Fatal("invalid name accepted or prompt closed")
	}
	press(m, "esc")
	if m.prompt != nil {
		t.Error("esc should cancel")
	}
}

func TestNewFolder(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "N")
	typeText(m, "Arkiv")
	press(m, "enter")
	if !m.vault.IsDir("Filosofi/Arkiv") {
		t.Fatal("folder not created")
	}
	if got := m.files.selected().Rel; got != "Filosofi/Arkiv" {
		t.Errorf("cursor on %q", got)
	}
}

func TestRenameAndUndo(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "r")
	if m.prompt == nil || m.prompt.in.value() != "Stoic" {
		t.Fatalf("rename prompt should be prefilled with the name")
	}
	press(m, "ctrl+u")
	typeText(m, "Stoa")
	press(m, "enter", "y") // y: also update the two links to it
	if !m.vault.Exists("Filosofi/Stoa.md") || m.vault.Exists("Filosofi/Stoic.md") {
		t.Fatal("rename failed")
	}
	if got := m.files.selected().Rel; got != "Filosofi/Stoa.md" {
		t.Errorf("the cursor should follow the rename, got %q", got)
	}
	press(m, "U")
	if !m.vault.Exists("Filosofi/Stoic.md") {
		t.Error("undo didn't rename back")
	}
}

func TestRenameFolderKeepsItOpen(t *testing.T) {
	m := newTestModel(t)
	press(m, "1", "j", "j", "l", "h", "r", "ctrl+u") // open Filosofi, back on it
	typeText(m, "Philosophy")
	press(m, "enter")
	if !m.vault.IsDir("Philosophy") || m.cwd() != "Philosophy" || m.files.selected().Rel != "Philosophy" {
		t.Errorf("cwd %q, cursor on %q", m.cwd(), m.files.selected().Rel)
	}
	if !m.files.expanded["Philosophy"] {
		t.Error("the renamed folder should stay open")
	}
}

func TestDeleteToTrashAndRestore(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "d")
	if m.confirm == nil || !strings.Contains(m.confirm.question, `"Stoic"`) {
		t.Fatalf("confirm = %+v", m.confirm)
	}
	press(m, "n")
	if !m.vault.Exists("Filosofi/Stoic.md") {
		t.Fatal("n should keep the note")
	}
	press(m, "d", "y")
	if m.vault.Exists("Filosofi/Stoic.md") {
		t.Fatal("note not deleted")
	}
	if got := m.files.selected().Rel; got != "Filosofi/Antik" {
		t.Errorf("after the delete the cursor should be on its neighbour, got %q", got)
	}
	trashed := filepath.Join(os.Getenv("XDG_DATA_HOME"), "Trash", "files", "Stoic.md")
	if _, err := os.Stat(trashed); err != nil {
		t.Fatalf("note not in the system trash: %v", err)
	}
	press(m, "U")
	if !m.vault.Exists("Filosofi/Stoic.md") {
		t.Error("U didn't restore from the trash")
	}
	if got := m.files.selected().Rel; got != "Filosofi/Stoic.md" {
		t.Errorf("cursor on %q, want the restored note", got)
	}
}

func TestDeleteOpenNoteClosesIt(t *testing.T) {
	m := newTestModel(t)
	onWelcome(m)
	press(m, "d")
	if m.confirm == nil || !strings.Contains(m.confirm.question, `"Welcome"`) {
		t.Fatalf("in the note, d should ask about the open note: %+v", m.confirm)
	}
	press(m, "y")
	if m.vault.Exists("Welcome.md") || m.notePath != "" {
		t.Fatalf("deleted? open %q", m.notePath)
	}
	checkFrame(t, m, "no note after delete")
	press(m, "U")
	if !m.vault.Exists("Welcome.md") || m.files.selected().Rel != "Welcome.md" {
		t.Errorf("U should restore it and put the cursor on it, cursor on %q", m.files.selected().Rel)
	}
}

func TestDeleteFolderAsksWithNoteCount(t *testing.T) {
	m := newTestModel(t)
	press(m, "1", "j", "j", "d")
	if m.confirm == nil || !strings.Contains(m.confirm.question, "folder \"Filosofi\" and its 2 notes") {
		t.Fatalf("question = %q", m.confirm.question)
	}
	press(m, "y")
	if m.vault.Exists("Filosofi") || m.files.selected().Rel != "Templates" {
		t.Errorf("folder deleted? cursor on %q, want the next folder", m.files.selected().Rel)
	}
}

func TestMarkMoveAndUndoAsOne(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "space", "space")
	if len(m.marks) != 2 {
		t.Fatalf("marks = %v", m.marks)
	}
	press(m, "m")
	typeText(m, "daily")
	if c := m.chooser; c == nil || len(c.matches) == 0 || c.items[c.matches[0]].label != "Daily/" {
		t.Fatalf("picker didn't find Daily: %+v", m.chooser)
	}
	press(m, "enter")
	if !m.vault.Exists("Daily/Stoic.md") || !m.vault.IsDir("Daily/Antik") || len(m.marks) != 0 {
		t.Fatal("bulk move failed")
	}
	press(m, "U")
	if !m.vault.Exists("Filosofi/Stoic.md") || !m.vault.Exists("Filosofi/Antik/Zeno.md") {
		t.Error("one U should move both back")
	}
}

func TestMovingTheOpenNoteKeepsItOpen(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "enter", "m") // in the note, m moves the open note
	typeText(m, "daily")
	press(m, "enter")
	if !m.vault.Exists("Daily/Stoic.md") || m.notePath != "Daily/Stoic.md" {
		t.Fatalf("moved? open %q", m.notePath)
	}
	if got := m.files.selected().Rel; got != "Filosofi/Antik" {
		t.Errorf("the cursor should stay behind on the next row, got %q", got)
	}
}

func TestMoveRefusesNameClash(t *testing.T) {
	m := newTestModel(t)
	if err := os.WriteFile(m.vault.Abs("Daily/Stoic.md"), []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}
	inFilosofi(m)
	press(m, "space", "space", "m")
	typeText(m, "daily")
	press(m, "enter")
	if !m.vault.IsDir("Filosofi/Antik") || !strings.Contains(m.flash, "Nothing moved") {
		t.Errorf("clash should move nothing; flash %q", m.flash)
	}
}

func TestVisualRangeAndMarkAll(t *testing.T) {
	m := newTestModel(t)
	press(m, "j", "v", "j", "j", "v") // Daily, Filosofi, Templates
	if len(m.marks) != 3 || m.visual != nil {
		t.Fatalf("visual marks = %v", m.marks)
	}
	press(m, "esc")
	if len(m.marks) != 0 {
		t.Error("esc should clear marks")
	}
	press(m, "ctrl+a")
	if want := len(m.files.siblings()); len(m.marks) != want {
		t.Errorf("ctrl+a marked %d of %d", len(m.marks), want)
	}
	press(m, "ctrl+a")
	if len(m.marks) != 0 {
		t.Error("second ctrl+a should unmark")
	}
	// A range can cross folders now: both daily notes, then Filosofi.
	press(m, "home", "j", "l", "v", "j", "j", "v")
	for _, p := range []string{"Daily/2026-09-11.md", "Daily/2026-09-13.md", "Filosofi"} {
		if !m.marks[p] {
			t.Errorf("%s not marked: %v", p, m.marks)
		}
	}
}

const wantDaily = "# Tuesday 15 September\n### Todo's\n- [ ] call mum\n  - about sunday\n* [ ] star task\n\n### Notes\n"

func TestDailyNoteWithRollover(t *testing.T) {
	m := newTestModel(t)
	press(m, "t")
	got, err := m.vault.Read("Daily/2026-09-15.md")
	if err != nil {
		t.Fatal(err)
	}
	if got != wantDaily {
		t.Errorf("daily note =\n%q\nwant\n%q", got, wantDaily)
	}
	if m.focus != paneNote || m.notePath != "Daily/2026-09-15.md" {
		t.Errorf("focus %v on %q", m.focus, m.notePath)
	}
	if prev, _ := m.vault.Read("Daily/2026-09-13.md"); !strings.Contains(prev, "call mum") {
		t.Error("previous note changed although deleteOnComplete is off")
	}
	press(m, "t")
	if !strings.HasPrefix(m.flash, "Today's note") {
		t.Errorf("second t should just open it; flash %q", m.flash)
	}
	press(m, "U")
	if m.vault.Exists("Daily/2026-09-15.md") {
		t.Error("U should remove the untouched new daily note")
	}
}

func TestDailyNoteLeavesRolloverToOpenObsidian(t *testing.T) {
	m := newTestModelWith(t, Options{RolloverTodos: true, ObsidianOpen: func() bool { return true }})
	press(m, "t")
	got, _ := m.vault.Read("Daily/2026-09-15.md")
	if got != "# Tuesday 15 September\n### Todo's\n\n### Notes\n" {
		t.Errorf("with Obsidian open the note should only have the template: %q", got)
	}
	if !strings.Contains(m.flash, "Rollover plugin") {
		t.Errorf("flash %q should say why", m.flash)
	}
}

func TestDailyRolloverOff(t *testing.T) {
	m := newTestModelWith(t, Options{RolloverTodos: false})
	press(m, "t")
	if got, _ := m.vault.Read("Daily/2026-09-15.md"); strings.Contains(got, "call mum") {
		t.Error("rollover_todos = false still rolled over")
	}
}
