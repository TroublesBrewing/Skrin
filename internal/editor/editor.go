// Package editor is Skrin's built-in note editor. It is a soft-wrapping
// text area with light markdown highlighting, list continuation, its own
// keystroke undo, and optional vim-style keys.
package editor

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/theme"
)

// Action tells the host what a key asked for.
type Action int

const (
	None  Action = iota
	Save         // write the note and keep editing
	Close        // write the note and leave the editor
	Copy         // copy the selection to the system clipboard; keep editing
)

// Mode is the vim mode. Without vim keys the editor is always in Insert.
type Mode int

const (
	Insert Mode = iota
	Normal
)

const (
	tabWidth = 4
	maxUndo  = 500
)

type state struct {
	lines    [][]rune
	row, col int
}

// Editor edits one note.
type Editor struct {
	lines       [][]rune
	row, col    int  // cursor; col is a rune index and may equal the line length
	goal        int  // visual column kept by vertical moves; -1 when unset
	top         int  // first visible display line
	w, h        int  // text area size in cells
	vim         bool //
	mode        Mode
	pending     string   // vim: operator waiting for a second key ("d", "y", "g")
	yank        []string // vim: lines yanked by dd or yy
	undo        []state
	redo        []state
	lastKind    string // kind of the last edit, for grouping keystrokes into one undo step
	sel         bool   // a selection runs from (srow, scol) to the cursor
	srow        int
	scol        int
	saved       string
	crlf        bool
	lineNumbers bool
	folds       []Fold
	st          styles
}

// New opens text for editing. With vim it starts in Normal mode.
func New(text string, vim bool, pal theme.Palette) *Editor {
	e := &Editor{goal: -1, w: 40, h: 10, vim: vim}
	e.SetPalette(pal)
	e.load(text)
	e.saved = e.Text()
	if vim {
		e.mode = Normal
	}
	return e
}

func (e *Editor) load(text string) {
	e.crlf = strings.Contains(text, "\r\n") // Text writes the endings back as they came
	parts := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	e.lines = make([][]rune, len(parts))
	for i, p := range parts {
		e.lines[i] = []rune(p)
	}
	e.row = clamp(e.row, 0, len(e.lines)-1)
	e.col = clamp(e.col, 0, len(e.lines[e.row]))
}

// Text is the note as it now reads, with its original line endings.
func (e *Editor) Text() string {
	parts := make([]string, len(e.lines))
	for i, l := range e.lines {
		parts[i] = string(l)
	}
	sep := "\n"
	if e.crlf {
		sep = "\r\n"
	}
	return strings.Join(parts, sep)
}

// Dirty reports unsaved changes.
func (e *Editor) Dirty() bool { return e.Text() != e.saved }

// MarkSaved records the current text as saved.
func (e *Editor) MarkSaved() { e.saved = e.Text() }

// Reset replaces the text (say, with the version on disk), dropping the
// keystroke history.
func (e *Editor) Reset(text string) {
	e.load(text)
	e.saved = e.Text()
	e.undo, e.redo, e.lastKind = nil, nil, ""
	e.scroll()
}

// Mode is the current vim mode.
func (e *Editor) Mode() Mode { return e.mode }

// SetPalette recolours the editor.
func (e *Editor) SetPalette(p theme.Palette) { e.st = newStyles(p) }

// SetLineNumbers turns line numbers on or off.
func (e *Editor) SetLineNumbers(on bool) {
	e.lineNumbers = on
	e.scroll()
}

// LineNumbers reports whether line numbers are enabled.
func (e *Editor) LineNumbers() bool { return e.lineNumbers }

// GutterWidth is the width of the line numbers column (e.g. " 1 │ ").
func (e *Editor) GutterWidth() int {
	if !e.lineNumbers {
		return 0
	}
	digits := max(len(strconv.Itoa(len(e.lines))), 2)
	return digits + 3
}

// SetSize sets the text area in cells.
func (e *Editor) SetSize(w, h int) {
	e.w, e.h = max(w, 4), max(h, 1)
	e.scroll()
}

// GoTo puts the cursor at the start of row and scrolls it to the top.
func (e *Editor) GoTo(row int) {
	e.row, e.col, e.goal = clamp(row, 0, len(e.lines)-1), 0, -1
	e.top = e.displayIndex(e.row, 0)
	e.scroll()
}

// TopRow is the source line shown at the top of the editor.
func (e *Editor) TopRow() int {
	d := 0
	for row := range e.lines {
		d += e.rowsOf(row)
		if d > e.top {
			return row
		}
	}
	return len(e.lines) - 1
}

// Paste inserts pasted text in one undo step, over the selection if there
// is one.
func (e *Editor) Paste(s string) {
	if e.vim && e.mode == Normal {
		e.mode = Insert
	}
	e.push("paste")
	if e.sel {
		e.deleteSel()
	}
	e.insertText(s)
	e.goal = -1
	e.scroll()
}

// HandleKey applies one key press.
func (e *Editor) HandleKey(k tea.KeyPressMsg) Action {
	s := k.String()
	switch s {
	case "ctrl+s":
		return Save
	case "ctrl+c":
		if e.sel {
			return Copy
		}
		return Close
	case "ctrl+l":
		e.push("todo")
		e.toggleTodo()
		e.goal = -1
		e.clampNormal()
		e.scroll()
		return None
	}
	if e.vim && e.mode == Normal {
		return e.normalKey(k)
	}
	// Shift and a cursor key selects, from wherever the cursor was.
	if base := strings.Replace(s, "shift+", "", 1); base != s && s != "shift+tab" {
		row, col := e.row, e.col
		if e.move(base) {
			if !e.sel {
				e.sel, e.srow, e.scol = true, row, col
			}
			if base != "up" && base != "down" && base != "pgup" && base != "pgdown" {
				e.goal = -1
			}
			e.scroll()
			return None
		}
	}
	vertical := false
	switch s {
	case "esc":
		if e.sel {
			e.sel = false
			break
		}
		if !e.vim {
			return Close
		}
		e.mode = Normal
		e.lastKind = ""
		e.clampNormal()
	case "enter":
		e.push("newline")
		e.dropSel()
		e.newline()
	case "backspace", "ctrl+h":
		e.push("delete")
		if !e.dropSel() {
			e.backspace()
		}
	case "delete":
		e.push("delete")
		if !e.dropSel() {
			e.deleteForward()
		}
	case "tab":
		e.push("indent")
		e.indent()
	case "shift+tab":
		e.push("indent")
		e.outdent()
	case "ctrl+z":
		e.undoEdit()
	case "ctrl+y", "ctrl+shift+z":
		e.redoEdit()
	default:
		if e.move(s) {
			e.sel = false
			vertical = s == "up" || s == "down" || s == "pgup" || s == "pgdown"
			break
		}
		if k.Text == "" || k.Mod&(tea.ModCtrl|tea.ModAlt) != 0 {
			return None
		}
		// Typing groups into words; in vim a whole insert is one undo step.
		kind := "type"
		if !e.vim && strings.TrimSpace(k.Text) == "" {
			kind = "space"
		}
		if e.sel {
			kind = "replace" // typing over a selection is its own undo step
		}
		e.push(kind)
		e.dropSel()
		e.insertText(k.Text)
	}
	if !vertical {
		e.goal = -1
	}
	e.scroll()
	return None
}

// move handles the cursor keys shared by both modes. It reports whether s
// was one.
func (e *Editor) move(s string) bool {
	switch s {
	case "left":
		e.left()
	case "right":
		e.right()
	case "up":
		e.moveVert(-1)
	case "down":
		e.moveVert(1)
	case "pgup":
		e.moveVert(-e.h)
	case "pgdown":
		e.moveVert(e.h)
	case "home":
		e.col = 0
	case "end":
		e.col = len(e.lines[e.row])
	case "ctrl+left", "alt+left":
		e.wordLeft()
	case "ctrl+right", "alt+right":
		e.wordRight()
	case "ctrl+home":
		e.row, e.col = 0, 0
	case "ctrl+end":
		e.row = len(e.lines) - 1
		e.col = len(e.lines[e.row])
	default:
		return false
	}
	e.lastKind = ""
	return true
}

func (e *Editor) normalKey(k tea.KeyPressMsg) Action {
	s := k.String()
	line := e.lines[e.row]
	vertical := false
	if p := e.pending; p != "" {
		e.pending = ""
		switch p + s {
		case "dd":
			e.push("cut")
			e.deleteLine()
		case "yy":
			e.yank = []string{string(line)}
		case "gg":
			e.row, e.col = 0, 0
		}
		e.clampNormal()
		e.scroll()
		return None
	}
	// Visual mode: motions stretch the selection, d or x deletes it, and
	// anything else ends it first.
	if e.sel {
		switch s {
		case "esc", "v":
			e.sel = false
			return None
		case "d", "x":
			e.push("cut")
			e.deleteSel()
			e.clampNormal()
			e.scroll()
			return None
		case "h", "l", "j", "k", "ctrl+d", "ctrl+u", "w", "b", "e", "0", "^", "$", "G", "g",
			"left", "right", "up", "down", "pgup", "pgdown", "home", "end":
		default:
			e.sel = false
		}
	}
	switch s {
	case "esc":
		return Close
	case "v":
		e.sel, e.srow, e.scol = true, e.row, e.col
	case "h":
		e.left()
	case "l":
		e.right()
	case "j", "k", "ctrl+d", "ctrl+u":
		n := map[string]int{"j": 1, "k": -1, "ctrl+d": max(e.h/2, 1), "ctrl+u": -max(e.h/2, 1)}[s]
		e.moveVert(n)
		vertical = true
	case "w":
		e.wordRight()
		if e.col >= len(e.lines[e.row]) && e.row < len(e.lines)-1 {
			e.row, e.col = e.row+1, firstNonSpace(e.lines[e.row+1])
		}
	case "b":
		e.wordLeft()
	case "e":
		e.wordEnd()
	case "0":
		e.col = 0
	case "^":
		e.col = firstNonSpace(line)
	case "$":
		e.col = len(line)
	case "G":
		e.row, e.col = len(e.lines)-1, 0
	case "g", "d", "y":
		e.pending = s
	case "i":
		e.insertMode()
	case "a":
		e.col = min(e.col+1, len(line))
		e.insertMode()
	case "I":
		e.col = firstNonSpace(line)
		e.insertMode()
	case "A":
		e.col = len(line)
		e.insertMode()
	case "o":
		e.push("line")
		e.col = len(line)
		e.insertText("\n" + leadingSpace(line))
		e.insertMode()
		e.lastKind = "type" // what's typed next joins this undo step
	case "O":
		e.push("line")
		ind := leadingSpace(line)
		e.insertLines(e.row, []string{ind})
		e.col = utf8.RuneCountInString(ind)
		e.insertMode()
		e.lastKind = "type" // what's typed next joins this undo step
	case "x":
		if len(line) > 0 {
			e.push("cut")
			e.deleteForward()
		}
	case "D":
		e.push("cut")
		e.lines[e.row] = line[:e.col:e.col]
	case "J":
		if e.row < len(e.lines)-1 {
			e.push("join")
			next := strings.TrimLeftFunc(string(e.lines[e.row+1]), unicode.IsSpace)
			e.col = len(line)
			e.lines[e.row] = []rune(strings.TrimRightFunc(string(line), unicode.IsSpace) + " " + next)
			e.lines = append(e.lines[:e.row+1], e.lines[e.row+2:]...)
		}
	case "p":
		if len(e.yank) > 0 {
			e.push("paste")
			e.insertLines(e.row+1, e.yank) // leaves the cursor on the first pasted line
			e.col = firstNonSpace(e.lines[e.row])
		}
	case "P":
		if len(e.yank) > 0 {
			e.push("paste")
			e.insertLines(e.row, e.yank)
			e.col = firstNonSpace(e.lines[e.row])
		}
	case "u":
		e.undoEdit()
	case "ctrl+r":
		e.redoEdit()
	default:
		if e.move(s) {
			vertical = s == "up" || s == "down" || s == "pgup" || s == "pgdown"
		}
	}
	if !vertical {
		e.goal = -1
	}
	e.clampNormal()
	e.scroll()
	return None
}

func (e *Editor) insertMode() {
	e.mode = Insert
	e.lastKind = ""
}

// clampNormal keeps the cursor on a character, as vim's Normal mode does.
func (e *Editor) clampNormal() {
	if e.vim && e.mode == Normal {
		e.col = clamp(e.col, 0, max(len(e.lines[e.row])-1, 0))
	}
}

// --- editing -------------------------------------------------------------

// push saves the state before an edit. Runs of typing (or of deleting)
// form one undo step, broken at spaces.
func (e *Editor) push(kind string) {
	if kind == e.lastKind && (kind == "type" || kind == "delete") {
		return
	}
	e.undo = append(e.undo, e.snapshot())
	if len(e.undo) > maxUndo {
		e.undo = e.undo[1:]
	}
	e.redo = nil
	e.lastKind = kind
}

func (e *Editor) snapshot() state {
	lines := make([][]rune, len(e.lines))
	for i, l := range e.lines {
		lines[i] = append([]rune(nil), l...)
	}
	return state{lines, e.row, e.col}
}

func (e *Editor) restore(s state) {
	e.lines, e.row, e.col = s.lines, s.row, s.col
	e.lastKind, e.sel = "", false
}

// --- selection -----------------------------------------------------------

// Selection is the selected text, or "" when nothing is selected.
func (e *Editor) Selection() string {
	if !e.sel {
		return ""
	}
	r1, c1, r2, c2 := e.selRange()
	if r1 == r2 {
		return string(e.lines[r1][c1:c2])
	}
	parts := []string{string(e.lines[r1][c1:])}
	for r := r1 + 1; r < r2; r++ {
		parts = append(parts, string(e.lines[r]))
	}
	return strings.Join(append(parts, string(e.lines[r2][:c2])), "\n")
}

// ClearSelection ends the selection, keeping the text.
func (e *Editor) ClearSelection() { e.sel = false }

// selRange is the selection in reading order, the end exclusive. In vim's
// Normal mode the character under the cursor is included, as in vim.
func (e *Editor) selRange() (r1, c1, r2, c2 int) {
	r1, c1, r2, c2 = e.srow, e.scol, e.row, e.col
	if r2 < r1 || (r2 == r1 && c2 < c1) {
		r1, c1, r2, c2 = r2, c2, r1, c1
	}
	if e.vim && e.mode == Normal {
		c2 = min(c2+1, len(e.lines[r2]))
	}
	return r1, c1, r2, c2
}

// deleteSel removes the selected text, leaving the cursor where it began.
func (e *Editor) deleteSel() {
	r1, c1, r2, c2 := e.selRange()
	head := append([]rune(nil), e.lines[r1][:c1]...)
	e.lines[r1] = append(head, e.lines[r2][c2:]...)
	e.lines = append(e.lines[:r1+1], e.lines[r2+1:]...)
	e.row, e.col, e.sel = r1, c1, false
}

// dropSel deletes the selection, if there is one, and reports whether it
// did.
func (e *Editor) dropSel() bool {
	if !e.sel {
		return false
	}
	e.deleteSel()
	return true
}

func (e *Editor) undoEdit() {
	if len(e.undo) == 0 {
		return
	}
	e.redo = append(e.redo, e.snapshot())
	e.restore(e.undo[len(e.undo)-1])
	e.undo = e.undo[:len(e.undo)-1]
}

func (e *Editor) redoEdit() {
	if len(e.redo) == 0 {
		return
	}
	e.undo = append(e.undo, e.snapshot())
	e.restore(e.redo[len(e.redo)-1])
	e.redo = e.redo[:len(e.redo)-1]
}

// insertText inserts s at the cursor; newlines split the line.
func (e *Editor) insertText(s string) {
	parts := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	line := e.lines[e.row]
	tail := append([]rune(nil), line[e.col:]...)
	first := append(append([]rune(nil), line[:e.col]...), []rune(parts[0])...)
	if len(parts) == 1 {
		e.lines[e.row] = append(first, tail...)
		e.col = len(first)
		return
	}
	added := [][]rune{first}
	for _, p := range parts[1 : len(parts)-1] {
		added = append(added, []rune(p))
	}
	last := []rune(parts[len(parts)-1])
	e.col = len(last)
	added = append(added, append(last, tail...))
	rest := append(added, e.lines[e.row+1:]...)
	e.lines = append(e.lines[:e.row], rest...)
	e.row += len(parts) - 1
}

// insertLines puts whole lines before line at.
func (e *Editor) insertLines(at int, lines []string) {
	added := make([][]rune, len(lines))
	for i, l := range lines {
		added[i] = []rune(l)
	}
	rest := append(added, e.lines[at:]...)
	e.lines = append(e.lines[:at], rest...)
	e.row = at
}

var listRE = regexp.MustCompile(`^(\s*)([-*+]|(\d{1,9})([.)]))( \[.\])? `)

var (
	taskRE   = regexp.MustCompile(`^(\s*)([-*+]|\d{1,9}[.)]) \[(.)\]`)
	bulletRE = regexp.MustCompile(`^\s*([-*+]|\d{1,9}[.)]) `)
)

// toggleTodo is Ctrl-l, like Obsidian's "Toggle checkbox": a to-do is
// ticked off (or back on), a list item becomes a to-do, and any other line
// becomes one, "- [ ] " in front.
func (e *Editor) toggleTodo() {
	line := string(e.lines[e.row])
	var next string
	if m := taskRE.FindStringSubmatchIndex(line); m != nil {
		mark := "x"
		if line[m[6]:m[7]] != " " {
			mark = " "
		}
		next = line[:m[6]] + mark + line[m[7]:]
	} else if m := bulletRE.FindStringIndex(line); m != nil {
		next = line[:m[1]] + "[ ] " + line[m[1]:]
	} else {
		indent := leadingSpace(e.lines[e.row])
		next = indent + "- [ ] " + line[len(indent):]
	}
	shift := utf8.RuneCountInString(next) - len(e.lines[e.row])
	e.lines[e.row] = []rune(next)
	e.col = clamp(e.col+shift, 0, len(e.lines[e.row]))
}

// newline splits the line. Inside a list item it continues the list, and
// on an empty item it ends the list instead, as Obsidian does.
func (e *Editor) newline() {
	line := string(e.lines[e.row])
	m := listRE.FindStringSubmatch(line)
	if m == nil || e.col < utf8.RuneCountInString(m[0]) {
		e.insertText("\n")
		return
	}
	if strings.TrimSpace(line[len(m[0]):]) == "" && e.col == len(e.lines[e.row]) {
		e.lines[e.row] = nil
		e.col = 0
		return
	}
	marker := m[2]
	if m[3] != "" {
		n, _ := strconv.Atoi(m[3])
		marker = strconv.Itoa(n+1) + m[4]
	}
	box := ""
	if m[5] != "" {
		box = " [ ]"
	}
	e.insertText("\n" + m[1] + marker + box + " ")
}

func (e *Editor) backspace() {
	if e.col > 0 {
		l := e.lines[e.row]
		e.lines[e.row] = append(l[:e.col-1:e.col-1], l[e.col:]...)
		e.col--
		return
	}
	if e.row == 0 {
		return
	}
	prev := e.lines[e.row-1]
	e.col = len(prev)
	e.lines[e.row-1] = append(append([]rune(nil), prev...), e.lines[e.row]...)
	e.lines = append(e.lines[:e.row], e.lines[e.row+1:]...)
	e.row--
}

func (e *Editor) deleteForward() {
	l := e.lines[e.row]
	if e.col < len(l) {
		e.lines[e.row] = append(l[:e.col:e.col], l[e.col+1:]...)
		return
	}
	if e.row < len(e.lines)-1 {
		e.lines[e.row] = append(append([]rune(nil), l...), e.lines[e.row+1]...)
		e.lines = append(e.lines[:e.row+1], e.lines[e.row+2:]...)
	}
}

func (e *Editor) deleteLine() {
	e.yank = []string{string(e.lines[e.row])}
	e.lines = append(e.lines[:e.row], e.lines[e.row+1:]...)
	if len(e.lines) == 0 {
		e.lines = [][]rune{nil}
	}
	e.row = min(e.row, len(e.lines)-1)
	e.col = firstNonSpace(e.lines[e.row])
}

// indent indents a list item (Obsidian uses tabs) or inserts a tab.
func (e *Editor) indent() {
	if listRE.MatchString(string(e.lines[e.row])) {
		e.lines[e.row] = append([]rune{'\t'}, e.lines[e.row]...)
		e.col++
		return
	}
	e.insertText("\t")
}

func (e *Editor) outdent() {
	l := e.lines[e.row]
	n := 0
	switch {
	case len(l) > 0 && l[0] == '\t':
		n = 1
	default:
		for n < len(l) && n < tabWidth && l[n] == ' ' {
			n++
		}
	}
	e.lines[e.row] = l[n:]
	e.col = max(e.col-n, 0)
}

// --- movement ------------------------------------------------------------

func (e *Editor) left() {
	switch {
	case e.col > 0:
		e.col--
	case e.row > 0 && !(e.vim && e.mode == Normal):
		e.row--
		e.col = len(e.lines[e.row])
	}
}

func (e *Editor) right() {
	switch {
	case e.col < len(e.lines[e.row]):
		e.col++
	case e.row < len(e.lines)-1 && !(e.vim && e.mode == Normal):
		e.row, e.col = e.row+1, 0
	}
}

func (e *Editor) wordRight() {
	line := e.lines[e.row]
	if e.col >= len(line) {
		if e.row < len(e.lines)-1 {
			e.row, e.col = e.row+1, 0
		}
		return
	}
	c := e.col
	for c < len(line) && !unicode.IsSpace(line[c]) {
		c++
	}
	for c < len(line) && unicode.IsSpace(line[c]) {
		c++
	}
	e.col = c
}

func (e *Editor) wordLeft() {
	if e.col == 0 {
		if e.row > 0 {
			e.row--
			e.col = len(e.lines[e.row])
		}
		return
	}
	line, c := e.lines[e.row], e.col
	for c > 0 && unicode.IsSpace(line[c-1]) {
		c--
	}
	for c > 0 && !unicode.IsSpace(line[c-1]) {
		c--
	}
	e.col = c
}

func (e *Editor) wordEnd() {
	line := e.lines[e.row]
	c := e.col + 1
	for c < len(line) && unicode.IsSpace(line[c]) {
		c++
	}
	for c+1 < len(line) && !unicode.IsSpace(line[c+1]) {
		c++
	}
	e.col = min(c, max(len(line)-1, 0))
}

// moveVert moves n display lines, keeping the visual column.
func (e *Editor) moveVert(n int) {
	starts := e.segments(e.row)
	s := segOf(starts, e.col)
	if e.goal < 0 {
		e.goal = width(e.lines[e.row][starts[s]:e.col])
	}
	row := e.row
	for ; n > 0; n-- {
		if s+1 < len(starts) {
			s++
		} else if row+1 < len(e.lines) {
			row, s = row+1, 0
			starts = e.segments(row)
		} else {
			break
		}
	}
	for ; n < 0; n++ {
		if s > 0 {
			s--
		} else if row > 0 {
			row--
			starts = e.segments(row)
			s = len(starts) - 1
		} else {
			break
		}
	}
	e.row = row
	e.col = colAt(e.lines[row], starts, s, e.goal)
}

// colAt finds the column in display segment s closest to visual x.
func colAt(line []rune, starts []int, s, x int) int {
	limit := len(line)
	if s+1 < len(starts) {
		limit = starts[s+1] - 1 // stay on this display line
	}
	c, w := starts[s], 0
	for c < limit {
		rw := runeWidth(line[c])
		if w+rw > x {
			break
		}
		w += rw
		c++
	}
	return c
}

// --- layout --------------------------------------------------------------

// segments returns the rune index where each display line of row starts.
// Lines break after the last space that fits, or anywhere in a long word.
// One cell is kept free for the cursor at the end of a line.
func (e *Editor) segments(row int) []int {
	line := e.lines[row]
	limit := e.w - 1
	starts := []int{0}
	x, start, space := 0, 0, -1
	for i, r := range line {
		w := runeWidth(r)
		if x+w > limit && i > start {
			cut := i
			if space >= start && space+1 < i {
				cut = space + 1
			}
			if cut < i && width(line[cut:i])+w > limit {
				cut = i
			}
			starts = append(starts, cut)
			start, x, space = cut, width(line[cut:i]), -1
		}
		x += w
		if r == ' ' || r == '\t' {
			space = i
		}
	}
	return starts
}

// segOf is the display segment holding col.
func segOf(starts []int, col int) int {
	s := 0
	for k, st := range starts {
		if col >= st {
			s = k
		}
	}
	return s
}

func (e *Editor) displayIndex(row, col int) int {
	n := 0
	for r := 0; r < row; r++ {
		n += e.rowsOf(r)
	}
	return n + segOf(e.segments(row), col)
}

func (e *Editor) scroll() {
	d := e.displayIndex(e.row, e.col)
	if d < e.top {
		e.top = d
	}
	if d >= e.top+e.h {
		e.top = d - e.h + 1
	}
	e.top = max(e.top, 0)
}

func runeWidth(r rune) int {
	if r == '\t' {
		return tabWidth
	}
	return ansi.StringWidth(string(r))
}

func width(rs []rune) int {
	w := 0
	for _, r := range rs {
		w += runeWidth(r)
	}
	return w
}

func firstNonSpace(line []rune) int {
	for i, r := range line {
		if !unicode.IsSpace(r) {
			return i
		}
	}
	return 0
}

func leadingSpace(line []rune) string {
	return string(line[:firstNonSpace(line)])
}

// LinkQuery reports what has been typed after an unclosed "[[" before the
// cursor on the current line. Typing an alias (after "|") ends it.
func (e *Editor) LinkQuery() (string, bool) {
	before := string(e.lines[e.row][:e.col])
	i := strings.LastIndex(before, "[[")
	if i < 0 {
		return "", false
	}
	q := before[i+2:]
	if strings.Contains(q, "]]") || strings.Contains(q, "|") {
		return "", false
	}
	return q, true
}

// CompleteLink replaces the link query with text and closes the link,
// leaving the cursor after the "]]".
func (e *Editor) CompleteLink(text string) {
	q, ok := e.LinkQuery()
	if !ok {
		return
	}
	e.push("complete")
	line := e.lines[e.row]
	start := e.col - utf8.RuneCountInString(q)
	rest := append([]rune(nil), line[e.col:]...)
	closing := "]]"
	if strings.HasPrefix(string(rest), "]]") {
		closing = ""
	}
	ins := []rune(text + closing)
	e.lines[e.row] = append(append(append([]rune(nil), line[:start]...), ins...), rest...)
	e.col = start + len(ins)
	if closing == "" {
		e.col += 2
	}
	e.lastKind, e.goal = "", -1
	e.scroll()
}

// CursorPos is where the cursor is drawn: display row from the top of the
// text area, and column.
func (e *Editor) CursorPos() (row, col int) {
	starts := e.segments(e.row)
	s := segOf(starts, e.col)
	return e.displayIndex(e.row, e.col) - e.top, width(e.lines[e.row][starts[s]:e.col])
}

func clamp(v, lo, hi int) int { return max(lo, min(v, hi)) }
