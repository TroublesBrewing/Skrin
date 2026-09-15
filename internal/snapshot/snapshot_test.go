package snapshot

import (
	"fmt"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	return Open("/vaults/test")
}

func TestUndoRedoWalkThroughVersions(t *testing.T) {
	s := newStore(t)
	// Three edits: v1 → v2 → v3 → v4 (current).
	for _, v := range []string{"v1", "v2", "v3"} {
		if err := s.Save("Note.md", v); err != nil {
			t.Fatal(err)
		}
	}
	cur := "v4"
	for _, want := range []string{"v3", "v2", "v1"} {
		snap, ok, err := s.Undo("Note.md", cur)
		if err != nil || !ok || snap.Content != want {
			t.Fatalf("undo from %q: %q %v %v, want %q", cur, snap.Content, ok, err, want)
		}
		cur = snap.Content
	}
	if _, ok, _ := s.Undo("Note.md", cur); ok {
		t.Error("undo past the oldest version")
	}
	for _, want := range []string{"v2", "v3", "v4"} {
		snap, ok, err := s.Redo("Note.md", cur)
		if err != nil || !ok || snap.Content != want {
			t.Fatalf("redo from %q: %q %v %v, want %q", cur, snap.Content, ok, err, want)
		}
		cur = snap.Content
	}
	if _, ok, _ := s.Redo("Note.md", cur); ok {
		t.Error("redo past the newest version")
	}
}

func TestSaveClearsRedo(t *testing.T) {
	s := newStore(t)
	s.Save("n.md", "a")
	s.Undo("n.md", "b") // now at a, b is redoable
	s.Save("n.md", "a") // a new edit from a
	if _, ok, _ := s.Redo("n.md", "c"); ok {
		t.Error("redo survived a new edit")
	}
}

func TestUndoSkipsVersionsEqualToCurrent(t *testing.T) {
	s := newStore(t)
	s.Save("n.md", "old")
	s.Save("n.md", "same")
	snap, ok, _ := s.Undo("n.md", "same")
	if !ok || snap.Content != "old" {
		t.Errorf("got %q, want the first version that differs", snap.Content)
	}
}

func TestPruneKeepsNewest(t *testing.T) {
	s := newStore(t)
	for i := 0; i < keep+5; i++ {
		s.Save("n.md", fmt.Sprint(i))
	}
	if n := len(versions(s.stack("n.md", "undo"))); n != keep {
		t.Errorf("kept %d versions, want %d", n, keep)
	}
	snap, _, _ := s.Undo("n.md", "now")
	if snap.Content != fmt.Sprint(keep+4) {
		t.Errorf("newest version = %q", snap.Content)
	}
}

func TestMoveFollowsNotesAndFolders(t *testing.T) {
	s := newStore(t)
	s.Save("Filosofi/Stoic.md", "before")
	if err := s.Move("Filosofi", "Philosophy"); err != nil {
		t.Fatal(err)
	}
	if err := s.Move("Philosophy/Stoic.md", "Philosophy/Stoa.md"); err != nil {
		t.Fatal(err)
	}
	snap, ok, _ := s.Undo("Philosophy/Stoa.md", "after")
	if !ok || snap.Content != "before" {
		t.Errorf("snapshot didn't follow the moves: %q %v", snap.Content, ok)
	}
	if err := s.Move("nothing/here.md", "x.md"); err != nil {
		t.Errorf("moving a note without snapshots: %v", err)
	}
}
