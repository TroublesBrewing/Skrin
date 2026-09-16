package ui

import (
	"path"
	"strings"
	"testing"

	"github.com/lurioso/skrin/internal/vault"
)

// entries turns paths into vault entries; a trailing / makes a folder.
func entries(paths ...string) []vault.Entry {
	var out []vault.Entry
	for _, p := range paths {
		rel := strings.TrimSuffix(p, "/")
		out = append(out, vault.Entry{Name: path.Base(rel), Rel: rel, IsDir: strings.HasSuffix(p, "/")})
	}
	return out
}

var sample = entries("Welcome.md", "b.png", "Templates/", "Daily/", "Daily/2026-09-11.md",
	"Filosofi/", "Filosofi/Stoic.md", "Filosofi/Antik/", "Filosofi/Antik/Zeno.md")

func newSample() *files {
	t := newFiles()
	t.set(sample, "vault", nil)
	return &t
}

// shown lists the visible rows, indented by depth, folders ending in /.
func (t *files) shown() string {
	var out []string
	for _, r := range t.rows {
		s := strings.Repeat(" ", r.depth) + r.Name
		if r.IsDir && r.Rel != "" {
			s += "/"
		}
		out = append(out, s)
	}
	return strings.Join(out, ",")
}

func TestFilesOrderFoldersFirst(t *testing.T) {
	f := newSample()
	if got, want := f.shown(), "vault, Daily/, Filosofi/, Templates/, b.png, Welcome.md"; got != want {
		t.Errorf("rows = %q, want %q", got, want)
	}
	if f.selected().Rel != "" || f.folder() != "" {
		t.Errorf("a fresh tree should start on the vault row, got %q", f.selected().Rel)
	}
}

func TestFilesInAndOut(t *testing.T) {
	f := newSample()
	f.selectPath("Filosofi")
	f.in() // opens Filosofi and steps onto Antik/
	if f.selected().Rel != "Filosofi/Antik" || !f.expanded["Filosofi"] {
		t.Fatalf("in: on %q", f.selected().Rel)
	}
	f.in()
	if f.selected().Rel != "Filosofi/Antik/Zeno.md" || f.folder() != "Filosofi/Antik" {
		t.Fatalf("second in: on %q", f.selected().Rel)
	}
	if f.in() {
		t.Error("in on a file should report false")
	}
	f.out() // a file: up to its folder
	if f.selected().Rel != "Filosofi/Antik" || !f.expanded["Filosofi/Antik"] {
		t.Fatalf("out from a file: on %q", f.selected().Rel)
	}
	f.out() // an open folder: close it
	if f.selected().Rel != "Filosofi/Antik" || f.expanded["Filosofi/Antik"] {
		t.Fatalf("out on an open folder should close it")
	}
	f.out() // a closed folder: up
	if f.selected().Rel != "Filosofi" {
		t.Errorf("out on a closed folder: on %q", f.selected().Rel)
	}
}

func TestFilesRevealAndCollapseAll(t *testing.T) {
	f := newSample()
	f.reveal("Filosofi/Antik/Zeno.md")
	if f.selected().Rel != "Filosofi/Antik/Zeno.md" {
		t.Fatalf("reveal: on %q", f.selected().Rel)
	}
	if got := strings.Join(f.openFolders(), ","); got != "Filosofi,Filosofi/Antik" {
		t.Errorf("open folders = %q", got)
	}
	f.collapseAll()
	if f.selected().Rel != "Filosofi" || len(f.openFolders()) != 0 {
		t.Errorf("collapse all: on %q, open %q", f.selected().Rel, f.openFolders())
	}
}

func TestFilesKeepPlaceWhenThingsGo(t *testing.T) {
	f := newSample()
	f.reveal("Filosofi/Stoic.md")
	f.set(entries("Welcome.md", "b.png", "Templates/", "Daily/", "Daily/2026-09-11.md",
		"Filosofi/", "Filosofi/Antik/", "Filosofi/Antik/Zeno.md"), "vault", nil)
	if f.selected().Rel != "Filosofi/Antik" {
		t.Errorf("after deleting Stoic the cursor should go to its neighbour, got %q", f.selected().Rel)
	}
	f.selectPath("Filosofi")
	f.set(entries("Welcome.md", "b.png", "Templates/", "Daily/", "Daily/2026-09-11.md"), "vault", nil)
	if f.selected().Rel != "Templates" || f.expanded["Filosofi"] {
		t.Errorf("after deleting Filosofi: on %q", f.selected().Rel)
	}
	f.reveal("Daily/2026-09-11.md")
	f.set(entries("Welcome.md", "Templates/"), "vault", nil)
	if f.selected().Rel != "" {
		t.Errorf("with its folder gone the cursor should climb to the vault row, got %q", f.selected().Rel)
	}
}

func TestFilesFollowRename(t *testing.T) {
	f := newSample()
	f.reveal("Filosofi/Antik/Zeno.md") // opens Filosofi and Antik
	f.selectPath("Filosofi")
	f.follow([][2]string{{"Filosofi", "Philosophy"}}, true)
	f.set(entries("Welcome.md", "Philosophy/", "Philosophy/Stoic.md", "Philosophy/Antik/", "Philosophy/Antik/Zeno.md"), "vault", nil)
	if f.selected().Rel != "Philosophy" || !f.expanded["Philosophy"] || !f.expanded["Philosophy/Antik"] {
		t.Errorf("after the rename: on %q, open %q", f.selected().Rel, f.openFolders())
	}
}

func TestFilesSiblings(t *testing.T) {
	f := newSample()
	if n := len(f.siblings()); n != 5 {
		t.Errorf("on the vault row: %d siblings, want the 5 top-level entries", n)
	}
	f.reveal("Filosofi/Stoic.md")
	if n := len(f.siblings()); n != 2 {
		t.Errorf("in Filosofi: %d siblings, want 2", n)
	}
}
