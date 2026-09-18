package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// quits reports whether cmd is Bubble Tea's quit.
func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// Ctrl+C copies a selection to the system clipboard (OSC 52, via
// tea.SetClipboard) instead of its usual quit/close meaning, exactly the
// way Esc already treats "clears a selection first" as the bigger meaning
// of the same key. With no selection, the first Ctrl+C says how to quit
// and a second one right after quits.

func TestCtrlCCopiesTheReadingViewSelection(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "v") // Welcome.md, one line selected
	if m.noteSel == nil {
		t.Fatal("v should start a selection")
	}
	_, cmd := m.Update(key("ctrl+c"))
	if m.noteSel == nil {
		t.Error("copying should leave the selection in place")
	}
	if m.flash != "Copied 1 line" {
		t.Errorf("flash = %q", m.flash)
	}
	if cmd == nil {
		t.Fatal("expected a clipboard command")
	}
	if got := fmt.Sprint(cmd()); got != "# Welcome" {
		t.Errorf("clipboard content = %q, want %q", got, "# Welcome")
	}
}

func TestCtrlCWithNoSelectionAsksBeforeQuitting(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(key("ctrl+c"))
	if quits(cmd) {
		t.Fatal("the first ctrl+c with nothing selected shouldn't quit: it's so often a reach for copy")
	}
	if !strings.Contains(m.flash, "ctrl+c again quits") || !strings.Contains(m.flash, "q") {
		t.Errorf("the first ctrl+c should say how to quit: flash %q", m.flash)
	}
	_, cmd = m.Update(key("ctrl+c"))
	if !quits(cmd) {
		t.Fatal("a second ctrl+c right after should quit")
	}
}

func TestCtrlCThenAnotherKeyStartsOver(t *testing.T) {
	m := newTestModel(t)
	press(m, "ctrl+c", "j")
	_, cmd := m.Update(key("ctrl+c"))
	if quits(cmd) {
		t.Fatal("ctrl+c, another key, ctrl+c isn't a double press")
	}
}

func TestQStillQuitsAtOnce(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(key("q"))
	if !quits(cmd) {
		t.Fatal("q should quit straight away")
	}
}

func TestCtrlCCopiesTheEditorSelection(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "enter", "shift+right", "shift+right", "shift+right") // enter opens straight into editing now
	if m.editor == nil || m.editor.Selection() != "# W" {
		t.Fatalf("setup: selection = %q", m.editor.Selection())
	}
	_, cmd := m.Update(key("ctrl+c"))
	if m.editor == nil {
		t.Error("copying a selection should not close the editor")
	}
	if m.flash != "Copied 1 line" {
		t.Errorf("flash = %q", m.flash)
	}
	if got := fmt.Sprint(cmd()); got != "# W" {
		t.Errorf("clipboard content = %q, want %q", got, "# W")
	}
}

func TestCtrlCWithNoEditorSelectionStillCloses(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "enter") // opens straight into editing now
	if m.editor == nil {
		t.Fatal("setup: editor should be open")
	}
	press(m, "ctrl+c")
	if m.editor != nil {
		t.Error("ctrl+c with nothing selected should still close the editor")
	}
}

func TestQuickNotePasteGoesIntoTheText(t *testing.T) {
	m := newTestModel(t)
	press(m, "i")
	if m.quickNote == nil {
		t.Fatal("i should open the quick-note overlay")
	}
	m.paste("pasted text")
	if got := m.quickNote.text.value(); got != "pasted text" {
		t.Errorf("quick note text = %q", got)
	}
}

func TestQuickNotePasteGoesIntoTheFolderField(t *testing.T) {
	m := newTestModel(t)
	press(m, "i", "tab")
	if m.quickNote.area != quickNoteFolder {
		t.Fatal("tab should move focus to the folder row")
	}
	m.paste("Daily")
	if got := m.quickNote.folder.value(); got != "Daily" {
		t.Errorf("folder field = %q", got)
	}
}
