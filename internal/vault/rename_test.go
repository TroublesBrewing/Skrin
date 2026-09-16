package vault

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Looking before renaming leaves a window: Obsidian Sync or another app can
// put a file at the destination in between, and os.Rename would overwrite
// it without a trace. Nothing Skrin moves may replace what it finds.
func TestRenameRefusesToReplace(t *testing.T) {
	dir := t.TempDir()
	from, to := filepath.Join(dir, "from.md"), filepath.Join(dir, "to.md")
	if err := os.WriteFile(from, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, []byte("theirs"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := renameNoReplace(from, to); !errors.Is(err, ErrExists) {
		t.Fatalf("renaming over a file = %v, want ErrExists", err)
	}
	if b, _ := os.ReadFile(to); string(b) != "theirs" {
		t.Errorf("the file at the destination was overwritten: %q", b)
	}
	if _, err := os.Stat(from); err != nil {
		t.Errorf("the source should still be here: %v", err)
	}
	if err := renameNoReplace(from, filepath.Join(dir, "free.md")); err != nil {
		t.Errorf("renaming to a free name: %v", err)
	}
}

func TestMoveRefusesToReplace(t *testing.T) {
	v := makeVault(t, "a.md", "b.md")
	if _, err := v.Move("a.md", "b.md"); !errors.Is(err, ErrExists) {
		t.Errorf("moving onto a file = %v, want ErrExists", err)
	}
	if !v.Exists("a.md") || !v.Exists("b.md") {
		t.Error("a refused move should leave both files where they were")
	}
}

func TestRestoreRefusesWhenSomethingTookThePlace(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir()) // keep the real trash out of it
	v := makeVault(t, "note.md")
	trashed, err := v.Trash("note.md", "local")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(v.Abs("note.md"), []byte("a new note by the same name"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := v.Restore(trashed); !errors.Is(err, ErrExists) {
		t.Fatalf("restoring onto a file = %v, want ErrExists", err)
	}
	if got, _ := v.Read("note.md"); got != "a new note by the same name" {
		t.Errorf("restoring overwrote the note that was there: %q", got)
	}
}
