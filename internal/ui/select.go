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
func (m *Model) toggleNoteSel() {
	switch {
	case m.noteSel != nil:
		m.noteSel = nil
	case m.notePath != "" && len(m.lines) > 0:
		m.noteSel = &lineSel{m.noteOff, m.noteOff}
	}
}

// moveNoteSel stretches the selection with a motion, scrolling to keep its
// end in view. It reports whether a was a motion.
func (m *Model) moveNoteSel(a action, vis int) bool {
	s := m.noteSel
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
	default:
		return false
	}
	s.cur = clamp(s.cur, 0, len(m.lines)-1)
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
	t := m.selectionText()
	if t == "" {
		return ""
	}
	n := strings.Count(t, "\n") + 1
	if strings.HasSuffix(t, "\n") {
		n--
	}
	if n > 1 {
		return plural(n, "line") + " selected"
	}
	return plural(countWords(t), "word") + " selected"
}

func (m *Model) clearSelection() {
	m.noteSel = nil
	if m.editor != nil {
		m.editor.ClearSelection()
	}
}
