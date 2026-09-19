package vault

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCheckName(t *testing.T) {
	for _, ok := range []string{"Stoic notes", "2026-09-15", "Å ä ö", "v1.2"} {
		if err := CheckName(ok); err != nil {
			t.Errorf("CheckName(%q) = %v", ok, err)
		}
	}
	for _, bad := range []string{"", "  ", ".hidden", "..", "a:b", "x#y", "[[x]]", "a|b", "what?"} {
		if CheckName(bad) == nil {
			t.Errorf("CheckName(%q) accepted", bad)
		}
	}
}

func TestCreateFileMakesParentsAndNeverOverwrites(t *testing.T) {
	v := makeVault(t, "Welcome.md")
	dirs, err := v.CreateFile("A/B/note.md", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(dirs, []string{"A", "A/B"}) {
		t.Errorf("created dirs = %q", dirs)
	}
	if _, err := v.CreateFile("A/B/note.md", "again"); !errors.Is(err, ErrExists) {
		t.Errorf("overwrite: err = %v", err)
	}
	if s, _ := v.Read("A/B/note.md"); s != "hi" {
		t.Errorf("content = %q", s)
	}
}

func TestMoveRefusesOverwriteAndSelfNesting(t *testing.T) {
	v := makeVault(t, "a.md", "b.md", "F/x.md")
	if _, err := v.Move("a.md", "b.md"); !errors.Is(err, ErrExists) {
		t.Errorf("overwrite: err = %v", err)
	}
	if _, err := v.Move("F", "F/Sub/F"); err == nil {
		t.Error("moved a folder into itself")
	}
	dirs, err := v.Move("a.md", "New/a.md")
	if err != nil || !v.Exists("New/a.md") || !reflect.DeepEqual(dirs, []string{"New"}) {
		t.Errorf("move into new folder: dirs %q, err %v", dirs, err)
	}
}

func TestSystemTrashAndRestore(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	v := makeVault(t, "Filosofi/My note.md", "Other/My note.md")

	first, err := v.Trash("Filosofi/My note.md", "system")
	if err != nil {
		t.Fatal(err)
	}
	if v.Exists("Filosofi/My note.md") || first.Stored != filepath.Join(data, "Trash", "files", "My note.md") {
		t.Fatalf("trashed to %q", first.Stored)
	}
	info, err := os.ReadFile(first.Info)
	if err != nil || !strings.Contains(string(info), "Path="+filepath.ToSlash(v.Root)+"/Filosofi/My%20note.md") {
		t.Errorf("trashinfo = %q (%v)", info, err)
	}
	second, err := v.Trash("Other/My note.md", "system")
	if err != nil || filepath.Base(second.Stored) != "My note 2.md" {
		t.Errorf("name clash: %q, %v", second.Stored, err)
	}

	if err := v.Restore(first); err != nil {
		t.Fatal(err)
	}
	if !v.Exists("Filosofi/My note.md") {
		t.Error("not restored")
	}
	if _, err := os.Stat(first.Info); !errors.Is(err, os.ErrNotExist) {
		t.Error("trashinfo left behind")
	}
}

func TestLocalTrash(t *testing.T) {
	v := makeVault(t, "Welcome.md")
	tr, err := v.Trash("Welcome.md", "local")
	if err != nil || tr.Stored != v.Abs(".trash/Welcome.md") {
		t.Fatalf("trashed to %q, %v", tr.Stored, err)
	}
	if err := v.Restore(tr); err != nil || !v.Exists("Welcome.md") {
		t.Errorf("restore: %v", err)
	}
}

func TestJournalUndoesInReverse(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	v := makeVault(t, "a.md")
	var j Journal

	dirs, _ := v.CreateFile("New/n.md", "")
	j.Record(Op{Desc: "create", Steps: []Step{{Kind: StepCreated, Rel: dirs[0]}, {Kind: StepCreated, Rel: "New/n.md"}}})
	v.Move("a.md", "New/a.md")
	j.Record(Op{Desc: "move", Steps: []Step{{Kind: StepMoved, Rel: "New/a.md", From: "a.md"}}})
	tr, _ := v.Trash("New/a.md", "system")
	j.Record(Op{Desc: "delete", Steps: []Step{{Kind: StepTrashed, Rel: "New/a.md", Trash: tr}}})
	j.Record(Op{Desc: "nothing"}) // ignored: no steps

	for _, want := range []string{"delete", "move", "create"} {
		op, ok, err := j.Undo(v, "system")
		if !ok || err != nil || op.Desc != want {
			t.Fatalf("undo: %q %v %v, want %q", op.Desc, ok, err, want)
		}
	}
	if !v.Exists("a.md") || v.Exists("New") {
		t.Error("vault not back to its starting state")
	}
	if _, ok, _ := j.Undo(v, "system"); ok {
		t.Error("undo with an empty journal")
	}
}

func TestUndoCreateKeepsEditedFiles(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	v := makeVault(t, "Welcome.md")
	var j Journal
	v.CreateFile("n.md", "")
	j.Record(Op{Desc: "create", Steps: []Step{{Kind: StepCreated, Rel: "n.md"}}})
	v.Write("n.md", "words worth keeping")
	if _, _, err := j.Undo(v, "system"); err != nil {
		t.Fatal(err)
	}
	if v.Exists("n.md") {
		t.Error("undo left the note in place")
	}
	if b, err := os.ReadFile(filepath.Join(data, "Trash", "files", "n.md")); err != nil || string(b) != "words worth keeping" {
		t.Errorf("edited note should be in the trash: %q, %v", b, err)
	}
}

func TestMacTrashUsesTheFinderTrashAndComesBack(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	v := makeVault(t, "Filosofi/My note.md", "Other/My note.md")

	first, err := v.trashOn("darwin", "Filosofi/My note.md", "system")
	if err != nil {
		t.Fatal(err)
	}
	if v.Exists("Filosofi/My note.md") || first.Stored != filepath.Join(home, ".Trash", "My note.md") || first.Info != "" {
		t.Fatalf("a Mac should trash into ~/.Trash, with no freedesktop info file: %+v", first)
	}
	second, err := v.trashOn("darwin", "Other/My note.md", "system")
	if err != nil || filepath.Base(second.Stored) != "My note 2.md" {
		t.Errorf("name clash in ~/.Trash: %q, %v", second.Stored, err)
	}
	if err := v.Restore(first); err != nil || !v.Exists("Filosofi/My note.md") {
		t.Errorf("U should bring it back from ~/.Trash: %v", err)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("XDG_DATA_HOME"), "Trash")); !errors.Is(err, os.ErrNotExist) {
		t.Error("a Mac shouldn't touch the freedesktop trash at all")
	}
}

func TestLinuxTrashIsUnchanged(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	v := makeVault(t, "Welcome.md")
	tr, err := v.trashOn("linux", "Welcome.md", "system")
	if err != nil || tr.Stored != filepath.Join(data, "Trash", "files", "Welcome.md") || tr.Info == "" {
		t.Errorf("Linux should keep the freedesktop trash: %+v, %v", tr, err)
	}
}
