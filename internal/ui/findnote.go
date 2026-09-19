package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/editor"
)

// noteFind is Ctrl+F in the editor: a field in the status line that finds
// text in the note being edited, as Obsidian's editor does. The match in
// hand shows as the selection, so Esc leaves it selected, ready to type
// over, and a second Esc clears it as it always does.
type noteFind struct {
	in               lineInput
	fromRow, fromCol int // where the search started, to go back to if nothing matches
	matches          []editor.Match
	cur              int
}

func (m *Model) openNoteFind() {
	row, col := m.editor.Cursor()
	m.noteFind = &noteFind{fromRow: row, fromCol: col}
}

// refind searches again after the query changed, from where the search
// started, wrapping round to the top.
func (m *Model) refind() {
	f := m.noteFind
	f.matches = m.editor.Find(f.in.value())
	f.cur = 0
	for i, mt := range f.matches {
		if mt.Row > f.fromRow || (mt.Row == f.fromRow && mt.Col >= f.fromCol) {
			f.cur = i
			break
		}
	}
	m.showFound()
}

func (m *Model) showFound() {
	f := m.noteFind
	if len(f.matches) == 0 {
		m.editor.MoveTo(f.fromRow, f.fromCol)
		return
	}
	m.editor.Show(f.matches[f.cur])
}

func (m *Model) noteFindKey(k tea.KeyPressMsg) {
	f := m.noteFind
	switch m.actionIn(inNoteFind, k.String()) {
	case actCancel:
		if len(f.matches) == 0 {
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
	right := m.st.muted.Render(count + " · enter next · shift+enter previous · esc close")
	return spread(left, right, m.width)
}
