package vault

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func makeVault(t *testing.T, files ...string) *Vault {
	t.Helper()
	root := t.TempDir()
	for _, f := range files {
		p := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("# "+f), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	v, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// TestOpenOnAMissingPathSaysSo is a polish-list item: the error should
// name the path and say what's wrong, not just surface a bare "stat"
// error.
func TestOpenOnAMissingPathSaysSo(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	_, err := Open(missing)
	if err == nil {
		t.Fatal("Open on a missing path should fail")
	}
	if !strings.Contains(err.Error(), missing) || !strings.Contains(err.Error(), "can't open the vault") {
		t.Errorf("error %q doesn't name the path and say what's wrong", err.Error())
	}
}

func TestDirsAndFilesSkipHidden(t *testing.T) {
	v := makeVault(t, "Welcome.md", "Daily/2026-09-11.md", "Filosofi/Stoa/Epiktetos.md",
		".obsidian/app.json", ".trash/old.md", "Daily/.draft.md")
	dirs, err := v.Dirs()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"", "Daily", "Filosofi", "Filosofi/Stoa"}; !reflect.DeepEqual(dirs, want) {
		t.Errorf("Dirs = %q, want %q", dirs, want)
	}
	files, err := v.Files()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Daily/2026-09-11.md", "Filosofi/Stoa/Epiktetos.md", "Welcome.md"}; !reflect.DeepEqual(files, want) {
		t.Errorf("Files = %q, want %q", files, want)
	}
}

// TestSkrinJsonStaysOutOfTheTree is Amendment 2: the order file is no
// longer a dotfile (so it can sync), but it's still Skrin's own machinery,
// so both Entries (the Files tree) and List skip it by name.
func TestSkrinJsonStaysOutOfTheTree(t *testing.T) {
	v := makeVault(t, "Welcome.md")
	if err := v.Write(OrderFile, `{"/": ["Welcome.md"]}`); err != nil {
		t.Fatal(err)
	}
	es, err := v.Entries()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range es {
		if e.Rel == OrderFile {
			t.Errorf("Entries shouldn't list %s: %v", OrderFile, es)
		}
	}
	entries, err := v.List("")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name == OrderFile {
			t.Errorf("List shouldn't list %s: %v", OrderFile, entries)
		}
	}
}

func TestEntriesListsFoldersAndFiles(t *testing.T) {
	v := makeVault(t, "Welcome.md", "Filosofi/Stoa/Epiktetos.md", ".obsidian/app.json", "Daily/.draft.md")
	es, err := v.Entries()
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range es {
		s := e.Rel
		if e.IsDir {
			s += "/"
		}
		if e.ModTime.IsZero() || e.Name != filepath.Base(e.Rel) {
			t.Errorf("entry %+v lacks a name or a time", e)
		}
		got = append(got, s)
	}
	if want := []string{"Daily/", "Filosofi/", "Filosofi/Stoa/", "Filosofi/Stoa/Epiktetos.md", "Welcome.md"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Entries = %q, want %q", got, want)
	}
}

func TestRemoveRefusesTheRoot(t *testing.T) {
	v := makeVault(t)
	if err := v.Remove(""); err == nil {
		t.Error("Remove(\"\") should refuse the vault root")
	}
	if _, err := os.Stat(v.Root); err != nil {
		t.Errorf("the vault root is gone: %v", err)
	}
}

func TestListFoldersFirst(t *testing.T) {
	v := makeVault(t, "b.md", "A.md", "zeta/x.md", "Alpha/y.md", ".hidden.md")
	entries, err := v.List("")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name)
	}
	if want := []string{"Alpha", "zeta", "A.md", "b.md"}; !reflect.DeepEqual(names, want) {
		t.Errorf("List = %q, want %q", names, want)
	}
	if entries[0].Rel != "Alpha" || !entries[0].IsDir {
		t.Errorf("first entry = %+v", entries[0])
	}
}

func TestWatchSeesNewNotesInNewFolders(t *testing.T) {
	v := makeVault(t, "Welcome.md")
	changed := make(chan struct{}, 8)
	trouble := make(chan string, 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tell := func(s string) {
		select {
		case trouble <- s:
		default:
		}
	}
	if err := v.Watch(ctx, func() { changed <- struct{}{} }, tell); err != nil {
		t.Fatal(err)
	}
	defer func() {
		select {
		case s := <-trouble:
			t.Errorf("a vault this size should watch cleanly: %q", s)
		default:
		}
	}()
	if err := os.MkdirAll(v.Abs("New/Deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	waitFor(t, changed)
	if err := os.WriteFile(v.Abs("New/Deep/note.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitFor(t, changed)
}

func waitFor(t *testing.T, ch chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("no change notification")
	}
}
