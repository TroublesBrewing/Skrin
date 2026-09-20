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
	press(m, "G", "l", "v") // Welcome.md, the cursor on
	selTo(m, 0)             // up to the first line
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

// Ctrl+V asks the terminal for its clipboard and waits for the answer.
// Terminals that won't answer get what Ctrl+C copied in Skrin instead.

func TestCtrlVPastesWhatTheTerminalHandsBack(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "e") // the editor, on Welcome.md
	m.editor.MoveTo(1, 0)
	press(m, "ctrl+v")
	if !m.pasting {
		t.Fatal("ctrl+v should be waiting for the terminal")
	}
	m.Update(tea.ClipboardMsg{Content: "from elsewhere", Selection: 'c'})
	if m.pasting {
		t.Error("the answer should end the wait")
	}
	if !strings.Contains(m.editor.Text(), "from elsewhere") {
		t.Errorf("editor = %q", m.editor.Text())
	}
	if m.flash != "Pasted" {
		t.Errorf("flash = %q", m.flash)
	}
}

func TestASilentTerminalGetsWhatYouCopiedHere(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "v")
	selTo(m, 0)
	press(m, "ctrl+c") // copy the first line of Welcome.md
	press(m, "e")
	m.editor.MoveTo(1, 0)
	press(m, "ctrl+v")
	m.Update(pasteTimeoutMsg{})
	if !strings.Contains(m.editor.Text(), "# Welcome") {
		t.Errorf("the line copied here should come back: %q", m.editor.Text())
	}
	if !strings.Contains(m.flash, "won't hand its clipboard over") {
		t.Errorf("the flash should say what happened: %q", m.flash)
	}
}

func TestASilentTerminalAndNothingCopiedSaysSo(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "e")
	press(m, "ctrl+v")
	m.Update(pasteTimeoutMsg{})
	if !strings.Contains(m.flash, "ctrl+shift+v") {
		t.Errorf("the flash should name the way that works: %q", m.flash)
	}
}

func TestAnEmptyClipboardIsNotAFailedRead(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "e")
	press(m, "ctrl+v")
	m.Update(tea.ClipboardMsg{Selection: 'c'})
	if m.flash != "The clipboard is empty" {
		t.Errorf("flash = %q", m.flash)
	}
}

func TestPastingWhereNothingTakesTextSaysSo(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l") // reading, not typing
	press(m, "ctrl+v")
	m.Update(tea.ClipboardMsg{Content: "text", Selection: 'c'})
	if !strings.Contains(m.flash, "Nothing here takes text") {
		t.Errorf("flash = %q", m.flash)
	}
	if strings.Contains(read(m, "Welcome.md"), "text") {
		t.Error("nothing should have been written")
	}
}

func TestPasteReachesEveryFieldThatTakesTyping(t *testing.T) {
	m := newTestModel(t)
	press(m, "/") // the search field
	m.Update(tea.PasteMsg{Content: "stoic"})
	if !strings.Contains(m.statusLine()+m.render(), "stoic") {
		t.Error("a paste should reach the search field")
	}
	press(m, "esc")
	press(m, "G", "l", "e", "ctrl+f") // find in the note
	m.Update(tea.PasteMsg{Content: "Wel\ncome"})
	if got := m.noteFind.in.value(); got != "Wel come" {
		t.Errorf("find field = %q, want the newline as a space", got)
	}
}

// selTo moves the reading view's cursor to line n with plain motions,
// the way a hand would.
func selTo(m *Model, n int) {
	for i := 0; i < 200 && m.noteSel != nil && m.noteSel.cur > n; i++ {
		press(m, "k")
	}
	for i := 0; i < 200 && m.noteSel != nil && m.noteSel.cur < n; i++ {
		press(m, "j")
	}
}
