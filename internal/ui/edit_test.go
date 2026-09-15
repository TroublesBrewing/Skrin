package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/lurioso/skrin/internal/editor"
)

const welcome = "# Welcome\nSee [[Stoic]] and [[Missing]] #start\n"

func read(m *Model, rel string) string {
	s, _ := m.vault.Read(rel)
	return s
}

// openWelcome opens Welcome.md, the last row in Files, in the editor.
func openWelcome(t *testing.T, m *Model) {
	t.Helper()
	press(m, "G", "e")
	if m.editor == nil || m.edit.rel != "Welcome.md" {
		t.Fatalf("editor open on %q", m.edit.rel)
	}
}

func TestEditSaveUndoRedo(t *testing.T) {
	m := newTestModel(t)
	openWelcome(t, m)
	checkFrame(t, m, "editor")
	typeText(m, "Hej! ")
	if !m.editor.Dirty() {
		t.Fatal("typing should make the note dirty")
	}
	press(m, "ctrl+s")
	if got := read(m, "Welcome.md"); got != "Hej! "+welcome {
		t.Fatalf("saved %q", got)
	}
	press(m, "esc")
	if m.editor != nil {
		t.Fatal("esc should leave the editor")
	}
	press(m, "u")
	if got := read(m, "Welcome.md"); got != welcome {
		t.Errorf("u should restore the note: %q", got)
	}
	press(m, "ctrl+r")
	if got := read(m, "Welcome.md"); got != "Hej! "+welcome {
		t.Errorf("ctrl+r should redo: %q", got)
	}
}

func TestLeavingEditorSavesAndKeepsPlace(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "enter", "ctrl+d")
	top := m.lines[m.noteOff].Src
	press(m, "e")
	if got := m.editor.TopRow(); got != top {
		t.Errorf("editor opened at line %d, want %d (the top of the view)", got, top)
	}
	typeText(m, "x")
	press(m, "esc")
	if !strings.Contains(read(m, "Filosofi/Stoic.md"), "xline") {
		t.Error("leaving the editor should save")
	}
	if got := m.lines[m.noteOff].Src; got != top {
		t.Errorf("view back at line %d, want %d", got, top)
	}
}

func TestConflictKeepMine(t *testing.T) {
	m := newTestModel(t)
	openWelcome(t, m)
	typeText(m, "mine ")
	if err := os.WriteFile(m.vault.Abs("Welcome.md"), []byte("theirs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	press(m, "ctrl+s")
	if m.conflict == nil {
		t.Fatal("a change on disk went unnoticed")
	}
	press(m, "d")
	if m.conflict.diff == nil {
		t.Fatal("d should show the diff")
	}
	checkFrame(t, m, "conflict diff")
	press(m, "d", "m")
	if m.conflict != nil || m.editor == nil || read(m, "Welcome.md") != "mine "+welcome {
		t.Fatalf("keep mine: conflict %v, disk %q", m.conflict, read(m, "Welcome.md"))
	}
	press(m, "esc", "u")
	if got := read(m, "Welcome.md"); got != "theirs\n" {
		t.Errorf("first u should bring back the version from disk: %q", got)
	}
	press(m, "u")
	if got := read(m, "Welcome.md"); got != welcome {
		t.Errorf("second u should bring back the original: %q", got)
	}
}

func TestConflictTakeTheirsOnLeave(t *testing.T) {
	m := newTestModel(t)
	openWelcome(t, m)
	typeText(m, "mine ")
	if err := os.WriteFile(m.vault.Abs("Welcome.md"), []byte("theirs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	press(m, "esc")
	if m.conflict == nil || !m.conflict.closing {
		t.Fatal("leaving with a change on disk should ask")
	}
	press(m, "t")
	if m.editor != nil || read(m, "Welcome.md") != "theirs\n" {
		t.Fatalf("take theirs: editor %v, disk %q", m.editor != nil, read(m, "Welcome.md"))
	}
	press(m, "u")
	if got := read(m, "Welcome.md"); got != "mine "+welcome {
		t.Errorf("u should bring back your version: %q", got)
	}
}

func TestEscInConflictKeepsEditing(t *testing.T) {
	m := newTestModel(t)
	openWelcome(t, m)
	typeText(m, "mine ")
	os.WriteFile(m.vault.Abs("Welcome.md"), []byte("theirs\n"), 0o644)
	press(m, "ctrl+s", "esc")
	if m.conflict != nil || m.editor == nil || read(m, "Welcome.md") != "theirs\n" {
		t.Error("esc should go back to editing without saving")
	}
}

func TestVimModeInEditor(t *testing.T) {
	m := newTestModelWith(t, Options{RolloverTodos: true, Vim: true})
	openWelcome(t, m)
	if m.editor.Mode() != editor.Normal {
		t.Fatal("vim editor should open in Normal mode")
	}
	press(m, "A")
	typeText(m, "!")
	press(m, "esc")
	if m.editor == nil || m.editor.Mode() != editor.Normal {
		t.Fatal("esc in Insert should go to Normal, not close")
	}
	press(m, "esc")
	if m.editor != nil || !strings.HasPrefix(read(m, "Welcome.md"), "# Welcome!\n") {
		t.Errorf("esc in Normal should save and close: %q", read(m, "Welcome.md"))
	}
}

func TestExternalEditorChangesAreUndoable(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "enter")
	if err := os.WriteFile(m.vault.Abs("Welcome.md"), []byte("changed in nvim\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(externalDoneMsg{rel: "Welcome.md", before: welcome})
	if !strings.Contains(m.flash, "u undoes") {
		t.Errorf("flash = %q", m.flash)
	}
	press(m, "u")
	if got := read(m, "Welcome.md"); got != welcome {
		t.Errorf("u after the external editor: %q", got)
	}
}

func TestExternalEditorCommand(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "hx --vsplit")
	if got := externalEditor(""); strings.Join(got, " ") != "hx --vsplit" {
		t.Errorf("from $EDITOR: %q", got)
	}
	if got := externalEditor("code -w"); strings.Join(got, " ") != "code -w" {
		t.Errorf("configured: %q", got)
	}
}

func TestSnapshotsFollowRename(t *testing.T) {
	m := newTestModel(t)
	openWelcome(t, m)
	typeText(m, "v2 ")
	press(m, "esc", "r", "ctrl+u")
	typeText(m, "Hello")
	press(m, "enter", "u")
	if got := read(m, "Hello.md"); got != welcome {
		t.Errorf("u after a rename: %q", got)
	}
}
