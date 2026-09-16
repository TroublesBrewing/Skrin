package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// The skim chords, per [[skrin split view]] Amendment 1: Alt+↓ / Alt+↑
// (alias Alt+j / Alt+k) in Files move the cursor and open the note it
// lands on in the split beside the one already open. Focus stays in
// Files; a skim with a split open replaces its note; folders, other
// files and the reference note itself move the cursor and nothing else;
// the note pane gets the pointer refusal.
//
// The fixture's tree, folders first: vault row, Daily/, Filosofi/,
// Templates/, Welcome.md — Filosofi/ holds Antik/ and Stoic.md, and
// Daily/ holds 2026-09-11.md and 2026-09-13.md.

func TestSkimOpensTheSplit(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "G")     // the bottom note: Welcome.md opens under the cursor
	press(m, "alt+k") // Templates/, a folder: moves, opens nothing
	press(m, "alt+k") // Filosofi/
	press(m, "l")     // opens Filosofi/, cursor stays
	press(m, "alt+j") // onto Antik/, a folder: moves only
	press(m, "alt+j") // onto Stoic.md
	if m.split == nil {
		t.Fatal("a skim with no split should open one")
	}
	if m.split.path != "Filosofi/Stoic.md" {
		t.Errorf("the split shows %q, want Filosofi/Stoic.md", m.split.path)
	}
	if m.notePath != "Welcome.md" {
		t.Errorf("the main pane moved to %q: the reference note stays put", m.notePath)
	}
	if m.focus != paneFiles {
		t.Errorf("focus is %v: a skim keeps it in Files", m.focus)
	}
	if m.flash != "Opened beside · Shift+→ focuses" {
		t.Errorf("flash %q: a skim-open should say what happened and how to focus", m.flash)
	}
}

// TestFocusingTheSkimmedNoteSwapsPanes reproduces the reported bug: after a
// skim, the cursor sits on the split's own note; l/right (or Enter) used to
// call showNote on it, which quietly threw away the reference note instead
// of just moving the focus to the pane that already shows it.
func TestFocusingTheSkimmedNoteSwapsPanes(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "G")     // Welcome.md is the reference, in the main pane
	press(m, "k")     // Templates/
	press(m, "k")     // Filosofi/
	press(m, "l")     // opens Filosofi/, cursor stays
	press(m, "alt+j") // onto Antik/, a folder: moves only
	press(m, "alt+j") // onto Stoic.md: the split opens on it
	if m.split == nil || m.split.path != "Filosofi/Stoic.md" {
		t.Fatalf("split = %+v, want Filosofi/Stoic.md open beside", m.split)
	}
	press(m, "l") // cursor is still on Stoic.md, the split's own note
	if m.notePath != "Filosofi/Stoic.md" || m.focus != paneNote {
		t.Fatalf("open %q, focus %v: l should focus the split's note", m.notePath, m.focus)
	}
	if m.split == nil || m.split.path != "Welcome.md" {
		t.Fatalf("split = %+v: the reference note should have moved there, not been dropped", m.split)
	}
}

func TestSkimReplacesTheSplitNote(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "G")     // Welcome.md is the reference
	press(m, "k")     // Templates/
	press(m, "k")     // Filosofi/
	press(m, "k")     // Daily/
	press(m, "l")     // opens Daily/, cursor stays
	press(m, "j")     // onto 2026-09-11.md: it opens under the cursor
	press(m, "alt+j") // onto 2026-09-13.md: the split opens
	if m.split.path != "Daily/2026-09-13.md" {
		t.Fatalf("the split shows %q before the second skim", m.split.path)
	}
	press(m, "j", "l", "j") // onto Filosofi/, open it, onto Antik/
	press(m, "alt+j")       // onto Stoic.md: the split is replaced
	if m.split.path != "Filosofi/Stoic.md" {
		t.Errorf("the split shows %q: the last note skimmed should win", m.split.path)
	}
	if m.notePath != "Daily/2026-09-11.md" {
		t.Errorf("the main pane moved to %q: the reference note stays put", m.notePath)
	}
}

func TestSkimAliasJandK(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "G")
	press(m, "alt+k", "alt+k") // Templates/, then Filosofi/, both folders
	press(m, "l")              // opens Filosofi/, cursor stays
	press(m, "alt+j")          // onto Antik/, a folder: moves only
	press(m, "alt+j")          // onto Stoic.md: the alias skims down
	if m.split == nil || m.split.path != "Filosofi/Stoic.md" {
		t.Fatalf("alt+j should skim like alt+down: split %v", m.split)
	}
	cur := m.files.cur
	press(m, "alt+k") // back onto Antik/, a folder: the split keeps Stoic
	if m.files.cur == cur {
		t.Error("alt+k didn't move the cursor")
	}
	if m.split.path != "Filosofi/Stoic.md" {
		t.Errorf("a folder row changed the split to %q", m.split.path)
	}
}

func TestSkimFolderMovesOnly(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "alt+j") // from the vault row onto Daily/, a folder
	if m.files.cur != 1 {
		t.Fatalf("the cursor is at %d: a folder row should still move it", m.files.cur)
	}
	if m.split != nil {
		t.Error("a folder row opened a split")
	}
	if m.flash != "" {
		t.Errorf("flash %q: a folder row is not an event", m.flash)
	}
}

func TestSkimReferenceNoteIsSilent(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "G")     // Welcome.md opens under the cursor and is the reference
	press(m, "k")     // Templates/
	press(m, "k")     // Filosofi/
	press(m, "k")     // Daily/
	press(m, "l")     // opens Daily/, cursor stays
	press(m, "j")     // onto 2026-09-11.md: it opens under the cursor
	press(m, "j")     // plain j onto 2026-09-13.md: it becomes the reference
	press(m, "k")     // back onto 2026-09-11.md — the same note as the main pane
	press(m, "alt+j") // onto 2026-09-13.md again: the split opens on it
	splitPath := m.split.path
	press(m, "alt+k") // onto the reference note itself: silent, panes untouched
	if m.split == nil || m.split.path != splitPath {
		t.Errorf("landing on the reference note changed the split: %v", m.split)
	}
	if m.flash != "" {
		t.Errorf("flash %q: the reference note row is not an event", m.flash)
	}
}

func TestSkimRefusesBelow80Columns(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 79, Height: 40})
	press(m, "G") // Welcome.md opens under the cursor
	press(m, "k", "k")
	press(m, "l")     // opens Filosofi/, cursor stays
	press(m, "alt+j") // onto Antik/, a folder: moves only
	press(m, "alt+j") // onto Stoic.md: refused, but the cursor moved
	if m.split != nil {
		t.Error("a split opened below the minimum width")
	}
	if m.flash != "No room to split" {
		t.Errorf("flash %q: the width refusal should be the split family's own", m.flash)
	}
	if m.files.selected().Name != "Stoic.md" {
		t.Error("the cursor didn't move below the split width: movement is never hostage to layout")
	}
}

func TestSkimFromTheNotePanePointsAtFiles(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "2") // the note pane
	press(m, "alt+down")
	if m.flash != "Go to Files (1) to skim" {
		t.Errorf("flash %q: the note pane should get the pointer refusal", m.flash)
	}
}

func TestSkimEdgeDoesNothingLoud(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "G")     // the bottom of the tree
	press(m, "alt+j") // past the end
	if m.flash != "" {
		t.Errorf("flash %q at the tree's edge: moving past the end is not an event", m.flash)
	}
	if m.split != nil {
		t.Error("a skim at the tree's edge opened a split")
	}
}

// TestShiftRightFocusesTheSplitFromFiles covers the PO's second review
// finding: Shift+→ from Files used to be a silent no-op, breaking the skim
// flash's own promise ("Opened beside · Shift+→ focuses"). It should swap
// the split's note into focus, exactly like the cursor landing back on it
// and pressing l/→ already does.
func TestShiftRightFocusesTheSplitFromFiles(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "G")     // Welcome.md is the reference, in the main pane
	press(m, "k")     // Templates/
	press(m, "k")     // Filosofi/
	press(m, "l")     // opens Filosofi/, cursor stays
	press(m, "alt+j") // onto Antik/, a folder: moves only
	press(m, "alt+j") // onto Stoic.md: the split opens on it
	if m.split == nil || m.split.path != "Filosofi/Stoic.md" {
		t.Fatalf("split = %+v, want Filosofi/Stoic.md open beside", m.split)
	}
	press(m, "shift+right")
	if m.notePath != "Filosofi/Stoic.md" || m.focus != paneNote {
		t.Fatalf("open %q, focus %v: Shift+→ should focus the split's note", m.notePath, m.focus)
	}
	if m.split == nil || m.split.path != "Welcome.md" {
		t.Fatalf("split = %+v: the reference note should have moved there, not been dropped", m.split)
	}
}

// TestShiftRightFromFilesIsANoOpWithoutASplit makes sure the new handler
// doesn't do anything when there's nothing to focus.
func TestShiftRightFromFilesIsANoOpWithoutASplit(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "G")
	cur := m.notePath
	press(m, "shift+right")
	if m.focus != paneFiles || m.notePath != cur || m.split != nil {
		t.Errorf("Shift+→ with no split changed something: focus %v, note %q, split %+v", m.focus, m.notePath, m.split)
	}
}
