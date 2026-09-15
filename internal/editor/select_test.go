package editor

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/theme"
)

func selKey(e *Editor, keys ...tea.KeyPressMsg) {
	for _, k := range keys {
		e.HandleKey(k)
	}
}

var (
	shiftRight = tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift}
	shiftDown  = tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift}
	shiftEnd   = tea.KeyPressMsg{Code: tea.KeyEnd, Mod: tea.ModShift}
	plainRight = tea.KeyPressMsg{Code: tea.KeyRight}
	escKey     = tea.KeyPressMsg{Code: tea.KeyEscape}
)

func letter(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func TestShiftArrowsSelect(t *testing.T) {
	e := New("one two\nthree four", false, theme.Default())
	selKey(e, shiftRight, shiftRight, shiftRight)
	if got := e.Selection(); got != "one" {
		t.Fatalf("selection %q, want one", got)
	}
	selKey(e, shiftDown)
	if got := e.Selection(); got != "one two\nthr" {
		t.Errorf("selection over two lines %q", got)
	}
	selKey(e, plainRight)
	if e.Selection() != "" {
		t.Error("a plain arrow should end the selection")
	}
}

func TestTypingReplacesSelection(t *testing.T) {
	e := New("one two", false, theme.Default())
	selKey(e, shiftRight, shiftRight, shiftRight, letter('1'))
	if got := e.Text(); got != "1 two" {
		t.Fatalf("text %q, want the selection replaced", got)
	}
	selKey(e, tea.KeyPressMsg{Code: tea.KeyEnd}, tea.KeyPressMsg{Code: tea.KeyHome})
	selKey(e, shiftEnd, tea.KeyPressMsg{Code: tea.KeyBackspace})
	if got := e.Text(); got != "" {
		t.Errorf("backspace over a selection left %q", got)
	}
	selKey(e, tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	if got := e.Text(); got != "1 two" {
		t.Errorf("undo should bring the text back: %q", got)
	}
}

func TestEscClearsSelectionBeforeClosing(t *testing.T) {
	e := New("one", false, theme.Default())
	selKey(e, shiftRight)
	if e.HandleKey(escKey) != None || e.Selection() != "" {
		t.Fatal("the first esc should only clear the selection")
	}
	if e.HandleKey(escKey) != Close {
		t.Error("the second esc should close")
	}
}

func TestVimVisualMode(t *testing.T) {
	e := New("one two", true, theme.Default())
	selKey(e, letter('v'), letter('e'))
	if got := e.Selection(); got != "one" {
		t.Fatalf("v e selected %q, want one", got)
	}
	selKey(e, letter('d'))
	if got := e.Text(); got != " two" || e.Selection() != "" {
		t.Errorf("d in visual mode left %q", got)
	}
	selKey(e, letter('v'), letter('l'))
	if e.HandleKey(escKey) != None || e.Selection() != "" {
		t.Error("esc in visual mode should only end the selection")
	}
}
