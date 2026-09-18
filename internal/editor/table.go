package editor

import (
	"errors"
	"slices"
	"strconv"
	"strings"
)

// Tables work the way Obsidian's Advanced Tables plugin made familiar: in a
// table, Tab and Shift+Tab move between cells and Enter to the start of the
// row below,
// and every move lines the table up again. Typing "| Name | Age" and Tab
// is enough to start one — the separator row is added. Everything else
// (rows, columns, alignment, sorting) is a TableOp, run from the palette.

// align is a column's alignment, from its separator cell.
type align int

const (
	alignNone align = iota
	alignLeft
	alignCenter
	alignRight
)

// mdTable is a markdown table as cells: rows[0] is the heading row, the
// separator isn't a row, and every row has one cell per column.
type mdTable struct {
	indent string
	rows   [][]string
	align  []align
}

func (t *mdTable) cols() int { return len(t.align) }

// TableOp is something to do to the table under the cursor.
type TableOp int

const (
	TableFormat TableOp = iota
	TableRowBelow
	TableRowAbove
	TableRowDelete
	TableRowUp
	TableRowDown
	TableColRight
	TableColLeft
	TableColDelete
	TableColMoveLeft
	TableColMoveRight
	TableAlignLeft
	TableAlignCenter
	TableAlignRight
	TableSortAsc
	TableSortDesc
)

// Errors a TableOp answers with, worded for the status line.
var (
	ErrNotInTable  = errors.New("Put the cursor in a table first")
	ErrHeadingRow  = errors.New("The heading row stays: move to a row below it")
	ErrLastColumn  = errors.New("A table needs one column: delete its lines to remove it")
	ErrTableEdge   = errors.New("Already at the table's edge")
	ErrNoBodyRows  = errors.New("No rows under the headings to sort")
	errNotTableKey = errors.New("not a table key")
)

// isTableLine reports whether l looks like a row of a table.
func isTableLine(l []rune) bool {
	return strings.HasPrefix(strings.TrimSpace(string(l)), "|")
}

// inFence reports whether row sits inside a ``` or ~~~ code block, where a
// line starting with | is code, not a table.
func (e *Editor) inFence(row int) bool {
	in := false
	for r := 0; r < row; r++ {
		t := strings.TrimSpace(string(e.lines[r]))
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			in = !in
		}
	}
	return in
}

// tableBounds finds the table around row: its first line and one past its
// last.
func (e *Editor) tableBounds(row int) (start, end int, ok bool) {
	if row >= len(e.lines) || !isTableLine(e.lines[row]) || e.inFence(row) {
		return 0, 0, false
	}
	start, end = row, row+1
	for start > 0 && isTableLine(e.lines[start-1]) {
		start--
	}
	for end < len(e.lines) && isTableLine(e.lines[end]) {
		end++
	}
	return start, end, true
}

// InTable reports whether the cursor is in a table.
func (e *Editor) InTable() bool {
	_, _, ok := e.tableBounds(e.row)
	return ok
}

// splitCells splits a table line into its cells, trimmed. An escaped \| is
// part of a cell, not a border.
func splitCells(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	if strings.HasSuffix(s, "|") && !strings.HasSuffix(s, `\|`) {
		s = s[:len(s)-1]
	}
	var cells []string
	var cur strings.Builder
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		switch {
		case rs[i] == '\\' && i+1 < len(rs) && rs[i+1] == '|':
			cur.WriteString(`\|`)
			i++
		case rs[i] == '|':
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteRune(rs[i])
		}
	}
	return append(cells, strings.TrimSpace(cur.String()))
}

func isSepCell(c string) bool {
	c = strings.TrimSpace(c)
	c = strings.TrimPrefix(c, ":")
	c = strings.TrimSuffix(c, ":")
	return c != "" && strings.Trim(c, "-") == ""
}

func isSepRow(cells []string) bool {
	for _, c := range cells {
		if !isSepCell(c) {
			return false
		}
	}
	return len(cells) > 0
}

func alignOf(c string) align {
	c = strings.TrimSpace(c)
	l, r := strings.HasPrefix(c, ":"), strings.HasSuffix(c, ":")
	switch {
	case l && r:
		return alignCenter
	case r:
		return alignRight
	case l:
		return alignLeft
	}
	return alignNone
}

// parseTable reads lines start..end as a table. hadSep says whether the
// second line was a separator; it's added when it wasn't.
func (e *Editor) parseTable(start, end int) (t *mdTable, hadSep bool) {
	line0 := string(e.lines[start])
	t = &mdTable{indent: line0[:len(line0)-len(strings.TrimLeft(line0, " \t"))]}
	var rows [][]string
	for r := start; r < end; r++ {
		rows = append(rows, splitCells(string(e.lines[r])))
	}
	var seps []string
	if len(rows) >= 2 && isSepRow(rows[1]) {
		hadSep, seps = true, rows[1]
		rows = append(rows[:1], rows[2:]...)
	}
	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r))
	}
	cols = max(cols, len(seps), 1)
	for i := range rows {
		for len(rows[i]) < cols {
			rows[i] = append(rows[i], "")
		}
	}
	t.rows = rows
	t.align = make([]align, cols)
	for c := range seps {
		t.align[c] = alignOf(seps[c])
	}
	return t, hadSep
}

// render lays the table out with its columns lined up, and says where each
// cell's text starts and ends on its line, in runes, for the cursor.
func (t *mdTable) render() (lines []string, at func(r, c int) (line, start, end int)) {
	cols := t.cols()
	w := make([]int, cols)
	for c := range w {
		w[c] = 3
		for _, r := range t.rows {
			w[c] = max(w[c], width([]rune(r[c])))
		}
	}
	type span struct{ start, end int }
	spans := make([][]span, len(t.rows))
	row := func(r int) string {
		var b strings.Builder
		b.WriteString(t.indent + "|")
		pos := len([]rune(t.indent)) + 1
		for c, cell := range t.rows[r] {
			gap := w[c] - width([]rune(cell))
			left := 0
			switch t.align[c] {
			case alignRight:
				left = gap
			case alignCenter:
				left = gap / 2
			}
			s := " " + strings.Repeat(" ", left)
			b.WriteString(s + cell + strings.Repeat(" ", gap-left) + " |")
			start := pos + len([]rune(s))
			spans[r] = append(spans[r], span{start, start + len([]rune(cell))})
			pos += len([]rune(s)) + len([]rune(cell)) + gap - left + 2
		}
		return b.String()
	}
	sep := func() string {
		var b strings.Builder
		b.WriteString(t.indent + "|")
		for c := range w {
			d := strings.Repeat("-", w[c])
			switch t.align[c] {
			case alignLeft:
				d = ":" + d[1:]
			case alignRight:
				d = d[1:] + ":"
			case alignCenter:
				d = ":" + d[2:] + ":"
			}
			b.WriteString(" " + d + " |")
		}
		return b.String()
	}
	lines = append(lines, row(0), sep())
	for r := 1; r < len(t.rows); r++ {
		lines = append(lines, row(r))
	}
	at = func(r, c int) (int, int, int) {
		line := r
		if r > 0 {
			line = r + 1
		}
		s := spans[r][c]
		return line, s.start, s.end
	}
	return lines, at
}

// cell is the cell the cursor is in: its row among t.rows and its column.
// On the separator line it counts as the heading row.
func (e *Editor) cell(start int, hadSep bool, t *mdTable) (r, c int) {
	r = e.row - start
	if hadSep && r >= 1 {
		r = max(r-1, 0)
	}
	r = clamp(r, 0, len(t.rows)-1)
	line := e.lines[e.row]
	lead := len(line) - len([]rune(strings.TrimLeft(string(line), " \t")))
	pipes := 0
	for i := lead; i < min(e.col, len(line)); i++ {
		if line[i] == '|' && (i == 0 || line[i-1] != '\\') {
			pipes++
		}
	}
	return r, clamp(pipes-1, 0, t.cols()-1)
}

// writeTable puts t back over lines start..end and the cursor at the end of
// cell r, c's text.
func (e *Editor) writeTable(start, end int, t *mdTable, r, c int) {
	lines, at := t.render()
	e.replaceLines(start, end, lines)
	line, _, cend := at(r, c)
	e.row, e.col, e.goal = start+line, cend, -1
}

// replaceLines swaps lines start..end for lines.
func (e *Editor) replaceLines(start, end int, lines []string) {
	added := make([][]rune, len(lines))
	for i, l := range lines {
		added[i] = []rune(l)
	}
	rest := append(added, e.lines[end:]...)
	e.lines = append(e.lines[:start], rest...)
}

// tableKey is Tab, Shift+Tab or Enter in a table: line it up and move.
func (e *Editor) tableKey(s string) error {
	start, end, ok := e.tableBounds(e.row)
	if !ok {
		return errNotTableKey
	}
	e.push("table")
	e.sel = false
	t, hadSep := e.parseTable(start, end)
	r, c := e.cell(start, hadSep, t)
	blank := make([]string, t.cols())
	switch s {
	case "tab":
		c++
		if c == t.cols() {
			r, c = r+1, 0
		}
		if r == len(t.rows) {
			t.rows = append(t.rows, slices.Clone(blank))
		}
	case "shift+tab":
		c--
		if c < 0 {
			r, c = r-1, t.cols()-1
		}
		if r < 0 {
			r, c = 0, 0
		}
	case "enter":
		last := r == len(t.rows)-1
		if last && r > 0 && strings.Join(t.rows[r], "") == "" {
			// Enter on an empty last row leaves the table: the row goes,
			// and the cursor lands on a line of its own under it — the way
			// Enter on an empty list item ends the list. A blank line is
			// kept between, since a line straight under a table is read
			// as one more row of it.
			t.rows = t.rows[:r]
			e.writeTable(start, end, t, r-1, c)
			after := start + len(t.rows) + 1
			blank := func(i int) bool { return i < len(e.lines) && strings.TrimSpace(string(e.lines[i])) == "" }
			if !blank(after) {
				e.replaceLines(after, after, []string{""})
			}
			if !blank(after + 1) {
				e.replaceLines(after+1, after+1, []string{""})
			}
			e.row, e.col = after+1, 0
			return nil
		}
		// The first cell of the row below: a row is filled with Tab, and
		// Enter starts the next one, the way a spreadsheet goes back to
		// where the run of Tabs began.
		r, c = r+1, 0
		if r == len(t.rows) {
			t.rows = append(t.rows, slices.Clone(blank))
		}
	}
	e.writeTable(start, end, t, r, c)
	return nil
}

// Table runs op on the table under the cursor, as one undo step, and
// lines the table up while it's at it.
func (e *Editor) Table(op TableOp) error {
	start, end, ok := e.tableBounds(e.row)
	if !ok {
		return ErrNotInTable
	}
	t, hadSep := e.parseTable(start, end)
	r, c := e.cell(start, hadSep, t)
	blank := make([]string, t.cols())
	switch op {
	case TableFormat:
	case TableRowBelow:
		t.rows = slices.Insert(t.rows, r+1, slices.Clone(blank))
		r++
	case TableRowAbove:
		if r == 0 {
			return ErrHeadingRow
		}
		t.rows = slices.Insert(t.rows, r, slices.Clone(blank))
	case TableRowDelete:
		if r == 0 {
			return ErrHeadingRow
		}
		t.rows = slices.Delete(t.rows, r, r+1)
		r = min(r, len(t.rows)-1)
	case TableRowUp, TableRowDown:
		d := -1
		if op == TableRowDown {
			d = 1
		}
		switch {
		case r == 0:
			return ErrHeadingRow
		case r+d < 1 || r+d >= len(t.rows):
			return ErrTableEdge
		}
		t.rows[r], t.rows[r+d] = t.rows[r+d], t.rows[r]
		r += d
	case TableColRight, TableColLeft:
		at := c
		if op == TableColRight {
			at = c + 1
		}
		for i := range t.rows {
			t.rows[i] = slices.Insert(t.rows[i], at, "")
		}
		t.align = slices.Insert(t.align, at, alignNone)
		c = at
	case TableColDelete:
		if t.cols() == 1 {
			return ErrLastColumn
		}
		for i := range t.rows {
			t.rows[i] = slices.Delete(t.rows[i], c, c+1)
		}
		t.align = slices.Delete(t.align, c, c+1)
		c = min(c, t.cols()-1)
	case TableColMoveLeft, TableColMoveRight:
		d := -1
		if op == TableColMoveRight {
			d = 1
		}
		if c+d < 0 || c+d >= t.cols() {
			return ErrTableEdge
		}
		for i := range t.rows {
			t.rows[i][c], t.rows[i][c+d] = t.rows[i][c+d], t.rows[i][c]
		}
		t.align[c], t.align[c+d] = t.align[c+d], t.align[c]
		c += d
	case TableAlignLeft:
		t.align[c] = alignLeft
	case TableAlignCenter:
		t.align[c] = alignCenter
	case TableAlignRight:
		t.align[c] = alignRight
	case TableSortAsc, TableSortDesc:
		if len(t.rows) < 3 {
			return ErrNoBodyRows
		}
		body := t.rows[1:]
		slices.SortStableFunc(body, func(a, b []string) int {
			n := compareCells(a[c], b[c])
			if op == TableSortDesc && a[c] != "" && b[c] != "" {
				n = -n // empty cells stay last either way
			}
			return n
		})
		r = 1
	}
	e.push("table")
	e.sel = false
	e.writeTable(start, end, t, r, c)
	e.scroll()
	return nil
}

// compareCells orders two cells: numbers as numbers, the rest as text
// without regard to case, and empty cells last.
func compareCells(a, b string) int {
	switch {
	case a == "" && b == "":
		return 0
	case a == "":
		return 1
	case b == "":
		return -1
	}
	fa, ea := strconv.ParseFloat(a, 64)
	fb, eb := strconv.ParseFloat(b, 64)
	if ea == nil && eb == nil {
		switch {
		case fa < fb:
			return -1
		case fa > fb:
			return 1
		}
		return 0
	}
	return strings.Compare(strings.ToLower(a), strings.ToLower(b))
}
