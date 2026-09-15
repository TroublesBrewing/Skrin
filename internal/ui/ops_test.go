package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// inFilosofi puts the cursor in the Filosofi folder's list, on Antik/.
func inFilosofi(m *Model) { press(m, "1", "j", "j", "l") }

func TestCreateNoteInCurrentFolderAndUndo(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "n")
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
	if e, _ := m.selected(); e.Rel != "Filosofi/Ny tanke.md" {
		t.Errorf("cursor on %q, want the new note", e.Rel)
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

func TestNewNoteInNewSubfolders(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "n")
	typeText(m, "Stoa/Seneca/Brev")
	press(m, "enter")
	if !m.vault.Exists("Filosofi/Stoa/Seneca/Brev.md") || m.cwd != "Filosofi/Stoa/Seneca" {
		t.Fatalf("nested create: cwd %q", m.cwd)
	}
	press(m, "esc", "U")
	if m.vault.Exists("Filosofi/Stoa") {
		t.Error("undo should remove the folders it created too")
	}
	if m.cwd != "Filosofi" {
		t.Errorf("after undo cwd = %q, want the nearest surviving folder", m.cwd)
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
	press(m, "N")
	typeText(m, "Arkiv")
	press(m, "enter")
	if !m.vault.IsDir("Filosofi/Arkiv") {
		t.Fatal("folder not created")
	}
	if e, _ := m.selected(); e.Rel != "Filosofi/Arkiv" {
		t.Errorf("cursor on %q", e.Rel)
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
	press(m, "U")
	if !m.vault.Exists("Filosofi/Stoic.md") {
		t.Error("undo didn't rename back")
	}
}

func TestRenameCurrentFolderFromTree(t *testing.T) {
	m := newTestModel(t)
	press(m, "1", "j", "j", "r", "ctrl+u")
	typeText(m, "Philosophy")
	press(m, "enter")
	if !m.vault.IsDir("Philosophy") || m.cwd != "Philosophy" || m.tree.selected() != "Philosophy" {
		t.Errorf("cwd %q, tree on %q", m.cwd, m.tree.selected())
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
	trashed := filepath.Join(os.Getenv("XDG_DATA_HOME"), "Trash", "files", "Stoic.md")
	if _, err := os.Stat(trashed); err != nil {
		t.Fatalf("note not in the system trash: %v", err)
	}
	press(m, "U")
	if !m.vault.Exists("Filosofi/Stoic.md") {
		t.Error("U didn't restore from the trash")
	}
	if e, _ := m.selected(); e.Rel != "Filosofi/Stoic.md" {
		t.Errorf("cursor on %q, want the restored note", e.Rel)
	}
}

func TestDeleteFolderAsksWithNoteCount(t *testing.T) {
	m := newTestModel(t)
	press(m, "1", "j", "j", "d")
	if m.confirm == nil || !strings.Contains(m.confirm.question, "folder \"Filosofi\" and its 2 notes") {
		t.Fatalf("question = %q", m.confirm.question)
	}
	press(m, "y")
	if m.vault.Exists("Filosofi") || m.cwd != "" {
		t.Errorf("folder deleted? cwd = %q", m.cwd)
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
	press(m, "2", "v", "j", "j", "v")
	if len(m.marks) != 3 || m.visual != nil {
		t.Fatalf("visual marks = %v", m.marks)
	}
	press(m, "esc")
	if len(m.marks) != 0 {
		t.Error("esc should clear marks")
	}
	press(m, "ctrl+a")
	if len(m.marks) != len(m.entries) {
		t.Errorf("ctrl+a marked %d of %d", len(m.marks), len(m.entries))
	}
	press(m, "ctrl+a")
	if len(m.marks) != 0 {
		t.Error("second ctrl+a should unmark")
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
