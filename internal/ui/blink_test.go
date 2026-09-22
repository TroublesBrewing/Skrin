package ui

import (
	"testing"
	"time"
)

func TestBlinkGapMapsWordsToPeriods(t *testing.T) {
	cases := []struct {
		word string
		gap  time.Duration
		on   bool
	}{
		{"off", 0, false},
		{"slow", time.Second, true},
		{"medium", 500 * time.Millisecond, true},
		{"fast", 250 * time.Millisecond, true},
		{"", 500 * time.Millisecond, true}, // unset and unknown fall back to medium
		{"weird", 500 * time.Millisecond, true},
	}
	for _, c := range cases {
		gap, on := blinkGap(c.word)
		if gap != c.gap || on != c.on {
			t.Errorf("blinkGap(%q) = %v, %v; want %v, %v", c.word, gap, on, c.gap, c.on)
		}
	}
}

func TestCursorBlinkTogglesTheEditorToo(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l", "e") // open the editor on Welcome
	if m.editor == nil {
		t.Fatal("editor should be open")
	}
	if !m.cursorOn {
		t.Fatal("cursor starts drawn")
	}
	m.Update(cursorBlinkMsg{})
	if m.cursorOn || m.editor.CursorOn() {
		t.Error("the blink should blank both the model's and the editor's cursor")
	}
	m.Update(cursorBlinkMsg{})
	if !m.cursorOn || !m.editor.CursorOn() {
		t.Error("the blink should restore both")
	}
}

func TestCursorBlinkOffKeepsCursorOn(t *testing.T) {
	m := newTestModel(t)
	m.opts.CursorBlink = "off"
	// armCursorBlink with "off" must leave the cursor drawn and arm no tick.
	m.cursorOn = false
	cmd := m.armCursorBlink()
	if cmd != nil {
		t.Error("blink off should arm no tick")
	}
	if !m.cursorOn {
		t.Error("blink off should leave the cursor drawn")
	}
}
