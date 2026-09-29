package ui

import (
	"strings"
	"testing"

	"github.com/lurioso/skrin/internal/version"
)

func copyPathModel(t *testing.T) *Model {
	t.Helper()
	if !version.Beta {
		t.Skip("a release build leaves every experiment out")
	}
	return newTestModelWith(t, Options{RolloverTodos: true, Beta: true, CopyPath: true})
}

// Off is the default, and the key says where it is switched on rather
// than doing nothing quietly: an experiment still costs its key, and an
// undocumented key is a defect.
func TestCopyPathIsOffUntilAskedFor(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "y")
	if !strings.Contains(m.flash, "beta") || !strings.Contains(m.flash, "Settings") {
		t.Errorf("flash %q: it should say where to switch it on", m.flash)
	}
	if m.copied != "" {
		t.Errorf("nothing should have been copied, got %q", m.copied)
	}
}

func TestCopyPathTakesTheRowUnderTheCursor(t *testing.T) {
	m := copyPathModel(t)
	inFilosofi(m)
	press(m, "j")
	if got := m.files.selected().Rel; got != "Filosofi/Stoic.md" {
		t.Fatalf("cursor on %q, expected Filosofi/Stoic.md", got)
	}
	press(m, "y")
	if m.copied != "Filosofi/Stoic.md" {
		t.Errorf("y copied %q, expected the vault-relative path", m.copied)
	}
	if !strings.Contains(m.flash, "Filosofi/Stoic.md") {
		t.Errorf("flash %q should name what was copied", m.flash)
	}
	press(m, "Y")
	if want := m.vault.Abs("Filosofi/Stoic.md"); m.copied != want {
		t.Errorf("Y copied %q, expected %q", m.copied, want)
	}
}

// A folder is a path like any other, so it copies too.
func TestCopyPathCopiesAFolderToo(t *testing.T) {
	m := copyPathModel(t)
	press(m, "1", "j", "j")
	if got := m.files.selected().Rel; got != "Filosofi" {
		t.Fatalf("cursor on %q, expected the Filosofi folder", got)
	}
	press(m, "y")
	if m.copied != "Filosofi" {
		t.Errorf("copied %q", m.copied)
	}
}

// In the note pane the open note is what's in hand, which is the same
// answer r, m and d give there.
func TestCopyPathTakesTheOpenNoteInTheNotePane(t *testing.T) {
	m := copyPathModel(t)
	inFilosofi(m)
	press(m, "j", "l") // read Stoic: the focus moves to the note pane
	if m.focus != paneNote || m.notePath != "Filosofi/Stoic.md" {
		t.Fatalf("focus %v, note %q", m.focus, m.notePath)
	}
	press(m, "y")
	if m.copied != "Filosofi/Stoic.md" {
		t.Errorf("copied %q", m.copied)
	}
}

// Marks win over the cursor, as they do for every other file action, and
// several paths come out one per line — what a shell expects.
func TestCopyPathCopiesEveryMarkedItemOnItsOwnLine(t *testing.T) {
	m := copyPathModel(t)
	inFilosofi(m)
	press(m, "j", "space", "j", "space") // mark Stoic, then the next row
	if len(m.marks) != 2 {
		t.Fatalf("marks %v", m.marks)
	}
	press(m, "Y")
	lines := strings.Split(m.copied, "\n")
	if len(lines) != 2 {
		t.Fatalf("copied %q, expected one path per line", m.copied)
	}
	for _, l := range lines {
		if !strings.HasPrefix(l, m.vault.Root) {
			t.Errorf("Y should copy whole paths, got %q", l)
		}
	}
	if !strings.Contains(m.flash, "2 paths") {
		t.Errorf("flash %q should count them", m.flash)
	}
}

// The vault itself has no path to copy, and saying so beats copying an
// empty string on top of what the clipboard already held.
func TestCopyPathSaysWhenThereIsNothingInHand(t *testing.T) {
	m := copyPathModel(t)
	press(m, "1") // the vault root row
	if m.files.selected().Rel != "" {
		t.Skip("the cursor didn't land on the vault row")
	}
	press(m, "y")
	if m.copied != "" || !strings.Contains(m.flash, "Nothing in hand") {
		t.Errorf("copied %q, flash %q", m.copied, m.flash)
	}
}
