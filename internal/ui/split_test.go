package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestNotesFollowTheCursor(t *testing.T) {
	m := newTestModel(t)
	press(m, "G")
	if m.notePath != "Welcome.md" || m.focus != paneFiles {
		t.Fatalf("open %q, focus %v: the note under the cursor should open", m.notePath, m.focus)
	}
	press(m, "k") // Templates/, a folder: the note stays
	if m.notePath != "Welcome.md" {
		t.Errorf("a folder under the cursor closed the note: open %q", m.notePath)
	}
	press(m, "l", "j") // open Templates, then onto its first entry, a note
	if m.notePath != "Templates/Daily template.md" {
		t.Errorf("stepping onto a note should open it: %q", m.notePath)
	}
	if len(m.back) != 0 {
		t.Error("skimming shouldn't fill the history")
	}
}

// splitWith opens the Go to note row matching q in a split with key.
func splitWith(m *Model, q, key string) {
	press(m, "g")
	typeText(m, q)
	press(m, key)
}

func TestSplitView(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "G", "enter")
	splitWith(m, "zeno", "shift+right")
	if m.split == nil || m.notePath != "Filosofi/Antik/Zeno.md" || m.split.path != "Welcome.md" || !m.splitLeft {
		t.Fatalf("split %+v, focused %q, other on the left %v", m.split, m.notePath, m.splitLeft)
	}
	if m.files.selected().Rel != "Filosofi/Antik/Zeno.md" {
		t.Error("the Files cursor should be on the focused note")
	}
	frame := ansi.Strip(m.render())
	if !strings.Contains(frame, "─ Welcome ─") || !strings.Contains(frame, "─ Zeno ─") || !strings.Contains(frame, "split") {
		t.Errorf("both panes should show:\n%s", frame)
	}
	for _, size := range [][2]int{{140, 45}, {100, 30}, {81, 24}, {80, 24}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, k := range []string{"shift+left", "shift+right"} {
			press(m, k)
			checkFrame(t, m, fmt.Sprintf("split at %dx%d after %s", size[0], size[1], k))
		}
	}
	press(m, "shift+left")
	if m.notePath != "Welcome.md" || m.splitLeft {
		t.Errorf("shift+left should focus Welcome, on the left: focused %q", m.notePath)
	}
	press(m, "shift+left")
	if m.notePath != "Welcome.md" {
		t.Error("shift+left with nothing further left should do nothing")
	}
	m.Update(tea.WindowSizeMsg{Width: 79, Height: 24})
	if m.split != nil || !strings.Contains(m.flash, "No room for a split") {
		t.Errorf("below 80 columns the split should close with a message: %q", m.flash)
	}
	checkFrame(t, m, "split closed at 79")

	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	splitWith(m, "stoic", "shift+left")
	if m.notePath != "Filosofi/Stoic.md" || m.splitLeft || m.split.path != "Welcome.md" {
		t.Fatalf("shift+left should open Stoic on the left: focused %q", m.notePath)
	}
	press(m, "esc")
	if m.split != nil || m.notePath != "Welcome.md" {
		t.Errorf("esc should close the pane you're in: focused %q", m.notePath)
	}
	splitWith(m, "zeno", "shift+right")
	press(m, "z")
	if !m.zen || m.split != nil || !strings.Contains(m.flash, "closed Welcome") {
		t.Errorf("zen should close the other pane: zen %v, split %v, flash %q", m.zen, m.split != nil, m.flash)
	}
	press(m, "esc")
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	splitWith(m, "stoic", "shift+right")
	if m.split != nil || !strings.Contains(m.flash, "No room") {
		t.Errorf("a split below 80 columns should be refused: %q", m.flash)
	}
}

func TestSplitFollowsRenames(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "G", "enter")
	splitWith(m, "zeno", "shift+right")
	press(m, "shift+left", "r", "ctrl+u") // rename Welcome, the focused pane
	typeText(m, "Hello")
	press(m, "enter")
	if m.notePath != "Hello.md" || m.split == nil || m.split.path != "Filosofi/Antik/Zeno.md" {
		t.Fatalf("focused %q, split %+v", m.notePath, m.split)
	}
	press(m, "shift+right", "r", "ctrl+u") // now Zeno
	typeText(m, "Zenon")
	press(m, "enter")
	if m.split.path != "Hello.md" || m.notePath != "Filosofi/Antik/Zenon.md" {
		t.Errorf("after renaming Zeno: focused %q, other %q", m.notePath, m.split.path)
	}
}

// TestEditingTheSkimmedNoteSwapsPanes covers the PO's first review finding:
// e, E, u, Ctrl-r and o (all behind subjectOpen) had the same hole the
// split-focus bugfix cured for l/→/Enter — the cursor landing back on the
// split's own note and pressing e used to call showNote on it, discarding
// the reference note. subjectOpen now swaps panes first, same as openRow.
func TestEditingTheSkimmedNoteSwapsPanes(t *testing.T) {
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
	press(m, "e") // cursor is still on Stoic.md, the split's own note
	if m.notePath != "Filosofi/Stoic.md" || m.editor == nil {
		t.Fatalf("open %q, editor set %v: e should edit the split's note", m.notePath, m.editor != nil)
	}
	if m.split == nil || m.split.path != "Welcome.md" {
		t.Fatalf("split = %+v: the reference note should have moved there, not been dropped", m.split)
	}
}

// TestOutlineOfTheSkimmedNoteSwapsPanes is the same guard for o.
func TestOutlineOfTheSkimmedNoteSwapsPanes(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "G")     // Welcome.md is the reference
	press(m, "k")     // Templates/
	press(m, "k")     // Filosofi/
	press(m, "l")     // opens Filosofi/, cursor stays
	press(m, "alt+j") // onto Antik/, a folder: moves only
	press(m, "alt+j") // onto Stoic.md: the split opens on it
	press(m, "o")     // cursor is still on Stoic.md, the split's own note
	if m.notePath != "Filosofi/Stoic.md" {
		t.Fatalf("open %q: o should focus the split's note", m.notePath)
	}
	if m.split == nil || m.split.path != "Welcome.md" {
		t.Fatalf("split = %+v: the reference note should have moved there, not been dropped", m.split)
	}
}
