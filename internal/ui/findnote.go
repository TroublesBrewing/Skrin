package ui

import (
	"fmt"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/editor"
)

// noteFind is Ctrl+F in the editor: a field in the status line that finds
// text in the note being edited, as Obsidian's editor does. The match in
// hand shows as the selection, so Esc leaves it selected, ready to type
// over, and a second Esc clears it as it always does.
type noteFind struct {
	in               lineInput
	fromRow, fromCol int // where the search started, to go back to if nothing matches
	matches          []findMatch
	cur              int
	// inView is a search of the note being read rather than edited. Its
	// matches are rows of the note as it's drawn, so a match shows where
	// it stands, highlighted in the text.
	inView bool
}

// findMatch is one match: the row it's on, and the cells it covers. While
// editing, the row is a line of the note and the cells are runes; while
// reading, the row is a line as drawn and the cells are screen columns.
type findMatch struct{ row, col, end int }

func (m *Model) openNoteFind() {
	if m.editor != nil {
		row, col := m.editor.Cursor()
		m.noteFind = &noteFind{fromRow: row, fromCol: col}
		return
	}
	if m.notePath == "" {
		m.flash = "Select a note to find in"
		return
	}
	m.noteFind = &noteFind{inView: true, fromRow: m.noteOff}
}

// findInView lists every place q occurs in the note as it's drawn,
// ignoring case. A match split over two lines by wrapping isn't found.
func (m *Model) findInView(q string) []findMatch {
	want := []rune(strings.ToLower(q))
	if len(want) == 0 {
		return nil
	}
	var out []findMatch
	for row, l := range m.lines {
		plain := []rune(ansi.Strip(l.Text))
		low := make([]rune, len(plain))
		for i, r := range plain {
			low[i] = unicode.ToLower(r)
		}
		for c := 0; c+len(want) <= len(low); {
			if string(low[c:c+len(want)]) == string(want) {
				col := ansi.StringWidth(string(plain[:c]))
				out = append(out, findMatch{row, col, col + ansi.StringWidth(string(plain[c:c+len(want)]))})
				c += len(want)
				continue
			}
			c++
		}
	}
	return out
}

// highlight marks the match in hand in a line of the note being read.
func (m *Model) highlight(line string, mt findMatch) string {
	text := ansi.Cut(ansi.Strip(line), mt.col, mt.end)
	return ansi.Cut(line, 0, mt.col) + "\x1b[m" + m.st.selFocus.Render(text) + ansi.TruncateLeft(line, mt.end, "")
}

// refind searches again after the query changed, from where the search
// started, wrapping round to the top.
func (m *Model) refind() {
	f := m.noteFind
	if f.inView {
		f.matches = m.findInView(f.in.value())
	} else {
		f.matches = nil
		for _, mt := range m.editor.Find(f.in.value()) {
			f.matches = append(f.matches, findMatch{mt.Row, mt.Col, mt.End})
		}
	}
	f.cur = 0
	for i, mt := range f.matches {
		if mt.row > f.fromRow || (mt.row == f.fromRow && mt.col >= f.fromCol) {
			f.cur = i
			break
		}
	}
	m.showFound()
}

func (m *Model) showFound() {
	f := m.noteFind
	if f.inView {
		if len(f.matches) == 0 {
			m.noteOff = clamp(f.fromRow, 0, max(len(m.lines)-1, 0))
			return
		}
		m.noteOff = scrollTo(f.matches[f.cur].row, m.noteOff, m.layout().bodyH-2, len(m.lines))
		return
	}
	if len(f.matches) == 0 {
		m.editor.MoveTo(f.fromRow, f.fromCol)
		return
	}
	mt := f.matches[f.cur]
	m.editor.Show(editor.Match{Row: mt.row, Col: mt.col, End: mt.end})
}

func (m *Model) noteFindKey(k tea.KeyPressMsg) {
	f := m.noteFind
	switch m.actionIn(inNoteFind, k.String()) {
	case actCancel:
		if len(f.matches) == 0 && !f.inView {
			m.editor.MoveTo(f.fromRow, f.fromCol)
		}
		m.noteFind = nil
	case actNextMatch:
		if n := len(f.matches); n > 0 {
			f.cur = (f.cur + 1) % n
			m.showFound()
		}
	case actPrevMatch:
		if n := len(f.matches); n > 0 {
			f.cur = (f.cur + n - 1) % n
			m.showFound()
		}
	default:
		if f.in.handle(k) {
			m.refind()
		}
	}
}

// noteFindLine is the status line while finding: the field, and how many
// matches there are and which one is showing.
func (m *Model) noteFindLine() string {
	f := m.noteFind
	left := m.st.pill.Render(" FIND ") + " " + f.in.view(m.st.text, m.st.cursor)
	var count string
	switch {
	case f.in.value() == "":
		count = "type to find in the note"
	case len(f.matches) == 0:
		count = "no match"
	default:
		count = fmt.Sprintf("%d of %d", f.cur+1, len(f.matches))
	}
	how := " · enter next · shift+enter previous · esc close"
	right := m.st.muted.Render(count + how)
	return spread(left, right, m.width)
}
