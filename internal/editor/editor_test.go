package editor

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/theme"
)

func key(k string) tea.KeyPressMsg {
	named := map[string]rune{
		"enter": tea.KeyEnter, "backspace": tea.KeyBackspace, "tab": tea.KeyTab, "esc": tea.KeyEscape,
		"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight,
		"home": tea.KeyHome, "end": tea.KeyEnd,
	}
	if c, ok := named[k]; ok {
		return tea.KeyPressMsg{Code: c}
	}
	if k == "shift+tab" {
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	}
	if k == "shift+enter" {
		return tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift}
	}
	if c, ok := strings.CutPrefix(k, "ctrl+"); ok {
		return tea.KeyPressMsg{Code: []rune(c)[0], Mod: tea.ModCtrl}
	}
	return tea.KeyPressMsg{Code: []rune(k)[0], Text: k}
}

func keys(e *Editor, ks ...string) {
	for _, k := range ks {
		e.HandleKey(key(k))
	}
}

func typ(e *Editor, s string) {
	for _, r := range s {
		e.HandleKey(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func open(text string) *Editor { return New(text, false, theme.Default()) }

func TestRoundTripKeepsLineEndings(t *testing.T) {
	for _, s := range []string{"a\nb\n", "a\r\nb\r\n", "", "x", "\n\n"} {
		if got := open(s).Text(); got != s {
			t.Errorf("round trip of %q gave %q", s, got)
		}
	}
	e := open("a\r\nb")
	keys(e, "end")
	typ(e, "!")
	if e.Text() != "a!\r\nb" {
		t.Errorf("CRLF not kept after an edit: %q", e.Text())
	}
}

func TestTypingUndoGroupsByWord(t *testing.T) {
	e := open("")
	typ(e, "hello world")
	for _, want := range []string{"hello ", "hello", ""} {
		keys(e, "ctrl+z")
		if e.Text() != want {
			t.Fatalf("after undo: %q, want %q", e.Text(), want)
		}
	}
	keys(e, "ctrl+y")
	if e.Text() != "hello" {
		t.Errorf("redo: %q", e.Text())
	}
	if !e.Dirty() {
		t.Error("changed text should be dirty")
	}
	e.MarkSaved()
	if e.Dirty() {
		t.Error("dirty after MarkSaved")
	}
}

func TestListContinuation(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"- [ ] buy milk", "- [ ] buy milk\n- [ ] "},
		{"- [x] done", "- [x] done\n- [ ] "},
		{"1. one", "1. one\n2. "},
		{"\t* nested", "\t* nested\n\t* "},
		{"plain", "plain\n"},
	} {
		e := open(c.in)
		keys(e, "end", "enter")
		if e.Text() != c.want {
			t.Errorf("enter after %q: %q, want %q", c.in, e.Text(), c.want)
		}
	}
	e := open("- a")
	keys(e, "end", "enter", "enter")
	if e.Text() != "- a\n" {
		t.Errorf("enter on an empty item should end the list: %q", e.Text())
	}
}

func TestBackspaceAndDeleteJoinLines(t *testing.T) {
	e := open("ab\ncd")
	keys(e, "down", "home", "backspace")
	if e.Text() != "abcd" || e.col != 2 {
		t.Errorf("backspace join: %q col %d", e.Text(), e.col)
	}
	keys(e, "end", "enter", "up", "end", "delete")
	if e.Text() != "abcd" {
		t.Errorf("delete join: %q", e.Text())
	}
}

func TestTabIndentsListItems(t *testing.T) {
	e := open("- a")
	keys(e, "tab")
	if e.Text() != "\t- a" {
		t.Fatalf("tab: %q", e.Text())
	}
	keys(e, "shift+tab")
	if e.Text() != "- a" {
		t.Errorf("shift+tab: %q", e.Text())
	}
}

func TestSoftWrapAndVerticalMoves(t *testing.T) {
	e := open("aaaa bbbb cccc\nnext")
	e.SetSize(11, 5) // wraps at 10, keeping a cell for the cursor
	if got := e.segments(0); len(got) != 2 || got[1] != 10 {
		t.Fatalf("segments = %v", got)
	}
	view := e.View()
	if len(view) != 5 {
		t.Fatalf("view has %d lines", len(view))
	}
	if ansi.Strip(view[0]) != "aaaa bbbb " || ansi.Strip(view[1]) != "cccc" {
		t.Errorf("wrapped view = %q", view[:3])
	}
	keys(e, "right", "right", "down")
	if e.row != 0 || e.col != 12 {
		t.Errorf("down within a wrapped line: row %d col %d", e.row, e.col)
	}
	keys(e, "down")
	if e.row != 1 || e.col != 2 {
		t.Errorf("down to next line: row %d col %d", e.row, e.col)
	}
}

func TestViewAlwaysFits(t *testing.T) {
	e := open(strings.Repeat("Stoicism teaches calm under pressure, #stoa [[Seneca]] **bold**\n", 30) + "\tTabbed\tline")
	for _, size := range [][2]int{{12, 4}, {30, 10}, {80, 40}} {
		e.SetSize(size[0], size[1])
		keys(e, "ctrl+end")
		view := e.View()
		if len(view) != size[1] {
			t.Fatalf("%v: %d lines", size, len(view))
		}
		for i, l := range view {
			if w := ansi.StringWidth(l); w > size[0] {
				t.Errorf("%v: line %d is %d wide", size, i, w)
			}
		}
	}
}

func TestPasteIsOneUndoStep(t *testing.T) {
	e := open("ab")
	keys(e, "right")
	e.Paste("X\nY")
	if e.Text() != "aX\nYb" || e.row != 1 || e.col != 1 {
		t.Fatalf("paste: %q at %d:%d", e.Text(), e.row, e.col)
	}
	keys(e, "ctrl+z")
	if e.Text() != "ab" {
		t.Errorf("undo paste: %q", e.Text())
	}
}

func TestGoToAndTopRow(t *testing.T) {
	e := open(strings.Repeat("line\n", 50))
	e.SetSize(20, 10)
	e.GoTo(30)
	if e.TopRow() != 30 {
		t.Errorf("TopRow = %d", e.TopRow())
	}
}

func TestLinkCompletion(t *testing.T) {
	e := open("See [[Sto")
	keys(e, "end")
	if q, ok := e.LinkQuery(); !ok || q != "Sto" {
		t.Fatalf("LinkQuery = %q %v", q, ok)
	}
	e.CompleteLink("Stoic")
	if e.Text() != "See [[Stoic]]" || e.col != 13 {
		t.Errorf("completed %q, cursor %d", e.Text(), e.col)
	}
	if _, ok := e.LinkQuery(); ok {
		t.Error("closed link still offers completion")
	}
	keys(e, "ctrl+z")
	if e.Text() != "See [[Sto" {
		t.Errorf("undo completion: %q", e.Text())
	}

	e = open("[[Sto]] x")
	keys(e, "right", "right", "right", "right", "right")
	e.CompleteLink("Filosofi/Stoic")
	if e.Text() != "[[Filosofi/Stoic]] x" || e.col != 18 {
		t.Errorf("already closed: %q cursor %d", e.Text(), e.col)
	}
	if _, ok := open("[[a|b").LinkQuery(); ok {
		t.Error("typing an alias should not complete")
	}
}

func TestCursorPos(t *testing.T) {
	e := open("aaaa bbbb cccc")
	e.SetSize(11, 5)
	keys(e, "end")
	if r, c := e.CursorPos(); r != 1 || c != 4 {
		t.Errorf("CursorPos = %d,%d, want 1,4", r, c)
	}
}

func TestCtrlLMakesAndTicksTodos(t *testing.T) {
	e := open("")
	keys(e, "ctrl+l")
	typ(e, "buy milk")
	if e.Text() != "- [ ] buy milk" {
		t.Fatalf("empty line: %q", e.Text())
	}
	keys(e, "ctrl+l")
	if e.Text() != "- [x] buy milk" {
		t.Errorf("tick off: %q", e.Text())
	}
	keys(e, "ctrl+l")
	if e.Text() != "- [ ] buy milk" {
		t.Errorf("tick back on: %q", e.Text())
	}
	for in, want := range map[string]string{
		"call mum":      "- [ ] call mum",
		"\t- nested":    "\t- [ ] nested",
		"1. first":      "1. [ ] first",
		"  - [-] maybe": "  - [ ] maybe",
	} {
		e := open(in)
		keys(e, "ctrl+l")
		if e.Text() != want {
			t.Errorf("ctrl+l on %q = %q, want %q", in, e.Text(), want)
		}
	}
	e = open("call mum")
	keys(e, "end", "ctrl+l")
	if e.col != len("- [ ] call mum") {
		t.Errorf("cursor should stay at the end of the text: %d", e.col)
	}
	keys(e, "ctrl+z")
	if e.Text() != "call mum" {
		t.Errorf("ctrl+z: %q", e.Text())
	}
}

func TestEscAndSaveActions(t *testing.T) {
	e := open("x")
	if e.HandleKey(key("ctrl+s")) != Save || e.HandleKey(key("esc")) != Close {
		t.Error("ctrl+s should save, esc should close")
	}
}

func TestVimBasics(t *testing.T) {
	e := New("one\ntwo\nthree", true, theme.Default())
	if e.Mode() != Normal {
		t.Fatal("vim should start in Normal mode")
	}
	keys(e, "j", "d", "d")
	if e.Text() != "one\nthree" {
		t.Fatalf("dd: %q", e.Text())
	}
	keys(e, "p")
	if e.Text() != "one\nthree\ntwo" {
		t.Fatalf("p: %q", e.Text())
	}
	keys(e, "g", "g", "o")
	typ(e, "new line")
	keys(e, "esc")
	if e.Text() != "one\nnew line\nthree\ntwo" || e.Mode() != Normal {
		t.Fatalf("o + typing: %q", e.Text())
	}
	keys(e, "u")
	if e.Text() != "one\nthree\ntwo" {
		t.Errorf("u should undo the whole insert: %q", e.Text())
	}
	keys(e, "A")
	typ(e, "!")
	keys(e, "esc", "x")
	if e.Text() != "one\nthree\ntwo" {
		t.Errorf("A! then x: %q", e.Text())
	}
	if e.HandleKey(key("esc")) != Close {
		t.Error("esc in Normal mode should close")
	}
}

func TestLineNumbersGutter(t *testing.T) {
	e := open("first line\nsecond line\nthird line")
	if e.LineNumbers() {
		t.Error("LineNumbers should default to false")
	}
	if gw := e.GutterWidth(); gw != 0 {
		t.Errorf("GutterWidth() = %d, want 0 when line numbers disabled", gw)
	}

	e.SetLineNumbers(true)
	if !e.LineNumbers() {
		t.Error("LineNumbers() should be true after SetLineNumbers(true)")
	}
	// 3 lines -> digits = 2 -> digits + 3 = 5
	if gw := e.GutterWidth(); gw != 5 {
		t.Errorf("GutterWidth() = %d, want 5 for 3-line file", gw)
	}

	e.SetSize(30, 5)
	view := strings.Join(e.View(), "\n")
	plain := ansi.Strip(view)
	if !strings.Contains(plain, " 1 │ ") || !strings.Contains(plain, " 2 │ ") || !strings.Contains(plain, " 3 │ ") {
		t.Errorf("expected gutter line numbers in view, got:\n%s", plain)
	}

	// Soft wrap continuation test
	eWrapped := open("this is a very long line that should wrap onto multiple display rows in a narrow width")
	eWrapped.SetLineNumbers(true)
	eWrapped.SetSize(20, 5)
	wrappedView := ansi.Strip(strings.Join(eWrapped.View(), "\n"))
	if !strings.Contains(wrappedView, " 1 │ ") {
		t.Errorf("expected source line 1 in wrapped view, got:\n%s", wrappedView)
	}
	if !strings.Contains(wrappedView, "   │ ") {
		t.Errorf("expected continuation padding in wrapped view, got:\n%s", wrappedView)
	}
}

func TestShiftEnterInsertsANewLineLikeEnter(t *testing.T) {
	e := open("")
	typ(e, "one")
	keys(e, "shift+enter")
	typ(e, "two")
	if got := e.Text(); got != "one\ntwo" {
		t.Errorf("got %q", got)
	}
	if e.row != 1 || e.col != 3 {
		t.Errorf("cursor at %d,%d, want 1,3", e.row, e.col)
	}
}

func TestShiftEnterUndoesLikeEnter(t *testing.T) {
	e := open("")
	typ(e, "one")
	keys(e, "shift+enter")
	typ(e, "two")
	// A newline and the typing after it are separate undo steps, the same
	// as plain Enter: Shift+Enter reuses that exact code path, not a
	// special case of its own.
	keys(e, "ctrl+z")
	if got := e.Text(); got != "one\n" {
		t.Fatalf("first undo should drop just the typing: got %q", got)
	}
	keys(e, "ctrl+z")
	if got := e.Text(); got != "one" {
		t.Errorf("second undo should drop the newline: got %q", got)
	}
}

func TestInsertBlockKeepsItsOwnLines(t *testing.T) {
	table := []string{"| A | B |", "| - | - |", "|   |   |"}
	cases := []struct {
		name, text string
		row        int
		want       string
	}{
		{"in the middle of a paragraph line", "one\ntwo", 0, "one\n\n| A | B |\n| - | - |\n|   |   |\n\ntwo"},
		{"on a blank line between paragraphs", "one\n\ntwo", 1, "one\n\n| A | B |\n| - | - |\n|   |   |\n\ntwo"},
		{"in an empty note", "", 0, "| A | B |\n| - | - |\n|   |   |"},
		{"after the last line", "one", 0, "one\n\n| A | B |\n| - | - |\n|   |   |"},
	}
	for _, c := range cases {
		e := open(c.text)
		e.GoTo(c.row)
		e.InsertBlock(table, 0, 2)
		if e.Text() != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, e.Text(), c.want)
		}
		if row := e.row; e.Text() != "" && string(e.lines[row]) != "| A | B |" || e.col != 2 {
			t.Errorf("%s: cursor at %d:%d, want on the block's first line at 2", c.name, e.row, e.col)
		}
		keys(e, "ctrl+z")
		if e.Text() != c.text {
			t.Errorf("%s: one undo should take the whole block back: %q", c.name, e.Text())
		}
	}
	e := open("")
	e.InsertBlock(table, 2, 2)
	if e.row != 2 || e.col != 2 {
		t.Errorf("row 2 should put the cursor in the first body cell: %d:%d", e.row, e.col)
	}
}

func TestArrowsPastTheEdgesGoToTheLineEnds(t *testing.T) {
	e := open("first line\nlast line")
	e.row, e.col = 1, 4
	keys(e, "down")
	if e.row != 1 || e.col != 9 {
		t.Errorf("down on the last row goes to its end: %d:%d", e.row, e.col)
	}
	keys(e, "up")
	if e.row != 0 || e.col != 4 {
		t.Errorf("up from there goes back to the column it came from: %d:%d", e.row, e.col)
	}
	keys(e, "up")
	if e.row != 0 || e.col != 0 {
		t.Errorf("up on the first row goes to its start: %d:%d", e.row, e.col)
	}
	v := New("one\ntwo", true, theme.Default())
	keys(v, "esc") // normal mode
	v.row, v.col = 1, 1
	keys(v, "j")
	if v.col != 1 {
		t.Errorf("vim's j on the last line stays put, as in vim: col %d", v.col)
	}
}
