package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// tableField is which of the table form's fields has the focus.
type tableField int

const (
	tableCols tableField = iota
	tableRows
	tableHeads
	tableFields // how many there are
)

const (
	tableMaxCols = 12
	tableMaxRows = 50
)

// tableForm is Insert table: the size of a new table and, if you like,
// its headings, with the markdown it will write shown underneath. It
// inserts; it doesn't edit a table once it's in the note — the table is
// plain text there, like everything else.
type tableForm struct {
	cols, rows int
	heads      lineInput
	field      tableField
	// typed is set once a digit has gone into the focused number: the
	// first digit after arriving replaces what's there, the next ones
	// add to it, the way a number field in any form behaves.
	typed bool
}

// startTable opens the form from the palette: in the editor, over it; in
// the reading view, by opening the note in the editor first, at the line
// you were reading.
func (m *Model) startTable() {
	if m.editor == nil {
		if _, ok := m.subject(); !ok {
			m.flash = "Select a note to put a table in"
			return
		}
		m.startEdit()
		if m.editor == nil {
			return // startEdit said why
		}
	}
	m.table = &tableForm{cols: 3, rows: 2, field: tableCols}
}

// headings are the headings typed so far: comma-separated, blanks kept,
// so "Name,,Date" leaves the middle one empty.
func (f *tableForm) headings() []string {
	v := f.heads.value()
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// size is the table's columns and rows: headings typed beyond the column
// count widen the table to fit them rather than being dropped.
func (f *tableForm) size() (cols, rows int) {
	return min(max(f.cols, len(f.headings())), tableMaxCols), f.rows
}

// markdown is the table as it goes into the note, columns padded so it
// reads as a table in the editor too. No headings leaves the heading row
// empty, Obsidian's way, for typing them in place.
func (f *tableForm) markdown() []string {
	cols, rows := f.size()
	heads := f.headings()
	cells := make([]string, cols)
	widths := make([]int, cols)
	for i := range cells {
		if i < len(heads) {
			cells[i] = strings.ReplaceAll(heads[i], "|", `\|`)
		}
		widths[i] = max(ansi.StringWidth(cells[i]), 3)
	}
	row := func(cell func(i int) string) string {
		var b strings.Builder
		b.WriteString("|")
		for i := range cells {
			c := cell(i)
			b.WriteString(" " + c + strings.Repeat(" ", widths[i]-ansi.StringWidth(c)) + " |")
		}
		return b.String()
	}
	out := []string{
		row(func(i int) string { return cells[i] }),
		row(func(i int) string { return strings.Repeat("-", widths[i]) }),
	}
	for range rows {
		out = append(out, row(func(int) string { return "" }))
	}
	return out
}

func (m *Model) tableKey(k tea.KeyPressMsg) {
	f := m.table
	key := k.String()
	switch m.actionIn(inTable, key) {
	case actCancel:
		m.table = nil
		return
	case actPick:
		m.insertTable()
		return
	case actNextField:
		f.field, f.typed = (f.field+1)%tableFields, false
		return
	case actPrevField:
		f.field, f.typed = (f.field+tableFields-1)%tableFields, false
		return
	case actRight, actLeft:
		// The ‹ n › arrows: ← one fewer, → one more. In the headings
		// field they move the text cursor instead, below.
		if f.field != tableHeads {
			if m.actionIn(inTable, key) == actRight {
				f.bump(1)
			} else {
				f.bump(-1)
			}
			return
		}
	}
	if f.field == tableHeads {
		f.heads.handle(k)
		return
	}
	switch {
	case key == "+" || key == "=":
		f.bump(1)
	case key == "-":
		f.bump(-1)
	case key == "backspace":
		f.setNum(f.num() / 10)
	case len(key) == 1 && key[0] >= '0' && key[0] <= '9':
		// Digits type the number: the first replaces it, then 1 and 2 make
		// 12, and a digit that would go past the limit starts again.
		d := int(key[0] - '0')
		n := d
		if f.typed && f.num()*10+d <= f.limit() {
			n = f.num()*10 + d
		}
		f.typed = true
		f.setNum(n)
	}
}

func (f *tableForm) num() int {
	if f.field == tableRows {
		return f.rows
	}
	return f.cols
}

func (f *tableForm) limit() int {
	if f.field == tableRows {
		return tableMaxRows
	}
	return tableMaxCols
}

// setNum sets the focused number, kept between 1 and its limit.
func (f *tableForm) setNum(n int) {
	n = clamp(n, 1, f.limit())
	if f.field == tableRows {
		f.rows = n
	} else {
		f.cols = n
	}
}

// bump moves the focused number by d; on the headings field it does nothing.
func (f *tableForm) bump(d int) {
	if f.field != tableHeads {
		f.setNum(f.num() + d)
	}
}

// insertTable puts the table in the note being edited, as a block of its
// own, and leaves the cursor in the first cell to fill: a heading when
// none were typed, otherwise the first cell under them.
func (m *Model) insertTable() {
	f := m.table
	m.table = nil
	if m.editor == nil {
		return
	}
	row := 0
	if len(f.headings()) > 0 {
		row = 2
	}
	m.editor.InsertBlock(f.markdown(), row, 2)
	cols, rows := f.size()
	m.flash = fmt.Sprintf("Table inserted: %s × %s · ctrl+z takes it out", plural(cols, "column"), plural(rows, "row"))
}

// tableBox draws the form: the three fields, then the table it will make.
func (m *Model) tableBox() []string {
	f := m.table
	w := min(clamp(m.width*3/5, 44, 72), m.width-4)
	inner := w - 2
	label := func(fd tableField, s string) string {
		s = fit(s, 10)
		if f.field == fd {
			return m.st.flash.Render(s)
		}
		return m.st.muted.Render(s)
	}
	number := func(fd tableField, n int) string {
		s := fmt.Sprintf("‹ %d ›", n)
		if f.field == fd {
			return m.st.selFocus.Render(s)
		}
		return m.st.text.Render(s)
	}
	cols, _ := f.size()
	colsNote := ""
	if cols > f.cols {
		colsNote = m.st.muted.Render("  widened to fit the headings")
	}
	heads := m.st.muted.Render("optional: Name, Rating, Date")
	if f.field == tableHeads || f.heads.value() != "" {
		heads = f.heads.view(m.st.text, m.st.cursor)
		if f.field != tableHeads {
			heads = m.st.text.Render(f.heads.value())
		}
	}
	body := []string{
		" " + label(tableCols, "Columns") + number(tableCols, cols) + colsNote,
		" " + label(tableRows, "Rows") + number(tableRows, f.rows),
		" " + label(tableHeads, "Headings") + heads,
		"",
	}
	md := f.markdown()
	show := min(len(md), max(m.height-16, 3))
	for _, l := range md[:show] {
		body = append(body, " "+m.st.muted.Render(fit(l, inner-2)))
	}
	if more := len(md) - show; more > 0 {
		body = append(body, " "+m.st.muted.Render(fmt.Sprintf("… and %s more", plural(more, "row"))))
	}
	body = append(body, "", " "+m.st.muted.Render("↑↓ field · ←→ or digits change · enter insert · esc cancel"))
	return m.box("Insert table", body, w, len(body)+2, true)
}
