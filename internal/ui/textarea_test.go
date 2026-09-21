package ui

import "testing"

// The bug report: "Jag kan inte navigera mellan rader i Quick Note eftersom
// (gissar jag) rutan tolkar min input som 1 lång rad istället för 2 rader
// vilket är vad jag ser iom wrappingen." The arrows must follow the rows
// the eye sees, which are the wrapped rows, and keep the column.

func TestTextAreaMovesThroughWrappedRows(t *testing.T) {
	var a textArea
	a.wrapWidth = 20
	a.set("en lång rad som wrappar över flera synliga rader i rutan")

	// One raw line, wrapped into several display rows.
	if got := countRows(wrapRowAt(&a)); got < 3 {
		t.Fatalf("expected the text to wrap into several rows, got %d", got)
	}

	// From the start, down moves to the second display row, staying in
	// the field — the whole point of the fix.
	a.cur = 0
	second := len([]rune("en lång rad som ")) // the first wrapped row's extent, in runes
	if !a.moveVert(1) {
		t.Fatal("down should move within a wrapped line")
	}
	if a.cur == 0 {
		t.Error("down didn't move the cursor")
	}
	if a.cur < second {
		t.Errorf("down moved to %d, still on the first display row", a.cur)
	}
	// And up again returns to where it was.
	if !a.moveVert(-1) || a.cur != 0 {
		t.Errorf("up should return the cursor to 0, got %d", a.cur)
	}

	// Down past the last display row leaves the field (returns false)…
	for i := 0; i < 50; i++ {
		if !a.moveVert(1) {
			return // reached the end: correct
		}
	}
	t.Fatal("down never left the text")
}

func TestTextAreaKeepsTheEyeColumn(t *testing.T) {
	var a textArea
	a.wrapWidth = 10
	// Two raw lines: the first wraps to two rows of 10, the second is
	// shorter than the column the eye is in.
	a.set("abcdefghijklmno\nxy")

	// Cursor on the first raw line, column 12 (second display row).
	a.cur = 12
	if !a.moveVert(1) {
		t.Fatal("down from a wrapped row should reach the next line")
	}
	// The column can't survive a 2-rune line; the cursor clamps to it.
	if a.cur != 18 { // start of line 2 (17) + min(12, 2) = 19… but the line's end is 19; clamp to 19-1
		t.Errorf("cursor = %d, want the short line clamped", a.cur)
	}
	if !a.moveVert(-1) {
		t.Fatal("up should re-enter the wrapped line")
	}
	// Coming back up, the column is measured in the row it lands on.
	if a.cur < 10 || a.cur > 15 {
		t.Errorf("up returned to %d, want a position on the second display row", a.cur)
	}
}

func TestTextAreaRawLinesWhenNoWrap(t *testing.T) {
	var a textArea
	a.set("first\nsecond")
	a.cur = 2
	if !a.moveVert(1) || a.cur != 8 { // line 2 starts at 6; column 2 → 8
		t.Errorf("down without wrap should go by raw lines, cur = %d", a.cur)
	}
	if a.moveVert(1) {
		t.Error("down past the last raw line should leave the field")
	}
	if !a.moveVert(-1) || a.cur != 2 {
		t.Errorf("up should return to the first line, cur = %d", a.cur)
	}
}

// wrapRowAt exposes the display-row count for the cursor's wrap width.
func wrapRowAt(a *textArea) []int {
	_, rowAt, _ := wrapPlain(a.runes, a.wrapWidth)
	return rowAt
}
