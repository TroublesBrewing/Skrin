package ui

import "strings"

// lineSel is v in the reading view: whole rendered lines from anchor to
// cur; the motion keys move cur.
type lineSel struct{ anchor, cur int }

func (s *lineSel) covers(i int) bool {
	return s != nil && i >= min(s.anchor, s.cur) && i <= max(s.anchor, s.cur)
}

// toggleNoteSel starts a line selection at the top line in view, or ends
// it.
// toggleNoteSel lights the cursor, or puts it out. Reading has no cursor
// — j and k scroll, and nothing is picked out — so v is what asks for
// one. It lands where you last left it in this note, or in the middle of
// what you are looking at, which is where your eye already is.
func (m *Model) toggleNoteSel() {
	switch {
	case m.noteSel != nil:
		m.noteSel = nil
	case m.notePath != "" && len(m.lines) > 0:
		at := m.selStart(m.layout().bodyH - 2)
		m.noteSel = &lineSel{at, at}
	}
}

// selStart is where the cursor comes back: the line it was on, if that is
// still in view, else the middle of the view.
func (m *Model) selStart(vis int) int {
	if m.noteAt >= m.noteOff && m.noteAt < min(m.noteOff+vis, len(m.lines)) {
		return m.noteAt
	}
	// The middle of what is actually on screen, not of the pane: a note
	// shorter than the pane would otherwise light its last line.
	shown := min(vis, len(m.lines)-m.noteOff)
	return clamp(m.noteOff+shown/2, 0, len(m.lines)-1)
}

// moveNoteSel stretches the selection with a motion, scrolling to keep its
// end in view. It reports whether a was a motion.
// moveNoteSel moves the cursor, or grows the selection from it, and
// scrolls to keep the end in view. It reports whether a was a motion.
//
// Plain motions move the cursor and take the selection back to that one
// line, the way an arrow key drops a selection in any editor. J and K are
// the bigger version of j and k, as the case grammar says: they leave the
// far end where it is and stretch.
func (m *Model) moveNoteSel(a action, vis int) bool {
	s := m.noteSel
	grow := false
	switch a {
	case actDown:
		s.cur++
	case actUp:
		s.cur--
	case actHalfDown:
		s.cur += max(vis/2, 1)
	case actHalfUp:
		s.cur -= max(vis/2, 1)
	case actTop:
		s.cur = 0
	case actBottom:
		s.cur = len(m.lines) - 1
	case actSelDown:
		s.cur, grow = s.cur+1, true
	case actSelUp:
		s.cur, grow = s.cur-1, true
	default:
		return false
	}
	s.cur = clamp(s.cur, 0, len(m.lines)-1)
	if !grow {
		s.anchor = s.cur
	}
	m.noteAt = s.cur
	if s.cur < m.noteOff {
		m.noteOff = s.cur
	}
	if s.cur >= m.noteOff+vis {
		m.noteOff = s.cur - vis + 1
	}
	return true
}

// selectionText is the highlighted text: the editor's selection, or the
// note's source lines under a line selection in the reading view.
func (m *Model) selectionText() string {
	if m.editor != nil {
		return m.editor.Selection()
	}
	s := m.noteSel
	if s == nil || len(m.lines) == 0 {
		return ""
	}
	src := strings.Split(m.noteSrc, "\n")
	a := clamp(m.lines[min(s.anchor, s.cur)].Src, 0, len(src)-1)
	b := clamp(m.lines[max(s.anchor, s.cur)].Src, 0, len(src)-1)
	return strings.Join(src[min(a, b):max(a, b)+1], "\n")
}

// selectedNote is the one measure of a selection worth showing: lines
// when it spans more than one, words when it sits inside a single line,
// where "1 line" would say nothing. One measure, never two, because the
// status line's room is finite and a truncated count is no count at all
// (the user's call, 2026-09-20: "Kapad information riskerar ju att ändå
// inte vara till någon nytta").
//
// A selection that ends at the start of a line covers the lines above it,
// not that one: Shift+Down once, from the start of a line, is one line.
func (m *Model) selectedNote() string {
	n, ok := m.selectedLineCount()
	if !ok {
		return ""
	}
	t := m.selectionText()
	if n > 1 {
		return plural(n, "line") + " selected"
	}
	return plural(countWords(t), "word") + " selected"
}

// selectedLineCount is how many lines the selection covers, and whether
// there is one at all. A selection of one blank line is still a
// selection: the status line must not fall back to counting the whole
// note behind your back.
func (m *Model) selectedLineCount() (int, bool) {
	if m.editor != nil {
		a, b, ok := m.editor.SelectedRows()
		return b - a + 1, ok
	}
	if s := m.noteSel; s != nil {
		return max(s.anchor, s.cur) - min(s.anchor, s.cur) + 1, true
	}
	return 0, false
}

func (m *Model) clearSelection() {
	m.noteSel = nil
	if m.editor != nil {
		m.editor.ClearSelection()
	}
}
