package ui

import (
	"testing"
)

// Ctrl+↑ / Ctrl+↓, per [[skrin two panels]] Amendment 1: the cursor jumps
// to the previous/next folder row among the visible rows (the vault row
// counts), skipping file rows. Silent on success; a flash on refusal.
//
// The fixture's tree, folders first: vault row, Daily/, Filosofi/,
// Templates/, Welcome.md.

func TestFolderJumpMovesBetweenFolders(t *testing.T) {
	m := newTestModel(t)
	press(m, "ctrl+down") // vault -> Daily/
	if m.files.selected().Rel != "Daily" {
		t.Fatalf("cursor on %q", m.files.selected().Rel)
	}
	press(m, "ctrl+down") // Daily/ -> Filosofi/
	if m.files.selected().Rel != "Filosofi" {
		t.Fatalf("cursor on %q", m.files.selected().Rel)
	}
	if m.flash != "" {
		t.Errorf("flash %q: a successful jump is silent", m.flash)
	}
	press(m, "ctrl+up") // back to Daily/
	if m.files.selected().Rel != "Daily" {
		t.Fatalf("cursor on %q", m.files.selected().Rel)
	}
}

func TestFolderJumpStartsFromAFileRow(t *testing.T) {
	m := newTestModel(t)
	press(m, "G")       // Welcome.md, a file row
	press(m, "ctrl+up") // up to the nearest folder row, Templates/
	if m.files.selected().Rel != "Templates" {
		t.Fatalf("cursor on %q", m.files.selected().Rel)
	}
}

func TestFolderJumpEdgeFlashes(t *testing.T) {
	m := newTestModel(t)
	press(m, "ctrl+up") // already on the vault row, the top
	if m.flash != "Already at the tree's top" {
		t.Errorf("flash %q", m.flash)
	}
	if m.files.selected().Rel != "" {
		t.Errorf("a refused jump should not move the cursor: on %q", m.files.selected().Rel)
	}
	press(m, "G", "ctrl+down") // the bottom of the tree
	if m.flash != "Already at the tree's bottom" {
		t.Errorf("flash %q", m.flash)
	}
}

func TestFolderJumpFromTheNotePanePointsAtFiles(t *testing.T) {
	m := newTestModel(t)
	press(m, "2") // the note pane
	press(m, "ctrl+down")
	if m.flash != "Go to Files (1) to jump between folders" {
		t.Errorf("flash %q: the note pane should get the pointer refusal", m.flash)
	}
}

func TestFolderRightNoOpsWhenAlreadyOpen(t *testing.T) {
	m := newTestModel(t)
	press(m, "j", "j") // Filosofi/, closed
	press(m, "l")      // opens it
	if !m.files.expanded["Filosofi"] {
		t.Fatal("l should have opened Filosofi")
	}
	if m.flash != "" {
		t.Errorf("flash %q: opening a folder is silent", m.flash)
	}
	press(m, "l") // already open: a no-op
	if m.files.selected().Rel != "Filosofi" {
		t.Errorf("a no-op right shouldn't move the cursor: on %q", m.files.selected().Rel)
	}
	if m.flash != "Already open — Enter toggles it closed" {
		t.Errorf("flash %q", m.flash)
	}
}
