package editor

import (
	"strings"
	"testing"
)

// at puts the cursor on row, col.
func at(e *Editor, row, col int) { e.row, e.col = row, col }

func lines(e *Editor) string { return e.Text() }

func TestTabStartsATableFromAHeadingLine(t *testing.T) {
	e := open("| Name | Age")
	at(e, 0, 12)
	keys(e, "tab")
	want := "| Name | Age |\n| ---- | --- |\n|      |     |"
	if e.Text() != want {
		t.Fatalf("got\n%s\nwant\n%s", e.Text(), want)
	}
	if e.row != 2 || e.col != 2 {
		t.Errorf("tab from the last heading goes to a new row's first cell: %d:%d", e.row, e.col)
	}
	typ(e, "Ada")
	keys(e, "tab")
	typ(e, "36")
	keys(e, "enter")
	want = "| Name | Age |\n| ---- | --- |\n| Ada  | 36  |\n|      |     |"
	if e.Text() != want {
		t.Fatalf("typing, tab, enter:\ngot\n%s\nwant\n%s", e.Text(), want)
	}
	if e.row != 3 || e.col != 2 {
		t.Errorf("enter goes to the first cell of the row below: %d:%d", e.row, e.col)
	}
}

func TestTabAndShiftTabWalkTheCells(t *testing.T) {
	e := open("| a | b |\n|---|---|\n| c | d |")
	at(e, 0, 2)
	keys(e, "tab")
	if e.row != 0 || string(e.lines[0][e.col-1]) != "b" {
		t.Errorf("tab: cursor at %d:%d", e.row, e.col)
	}
	keys(e, "tab")
	if e.row != 2 || string(e.lines[2][e.col-1]) != "c" {
		t.Errorf("tab past the last heading skips the separator: %d:%d", e.row, e.col)
	}
	keys(e, "shift+tab", "shift+tab")
	if e.row != 0 || string(e.lines[0][e.col-1]) != "a" {
		t.Errorf("shift+tab walks back: %d:%d", e.row, e.col)
	}
	keys(e, "shift+tab")
	if e.row != 0 || string(e.lines[0][e.col-1]) != "a" {
		t.Errorf("shift+tab stops at the first cell: %d:%d", e.row, e.col)
	}
}

func TestEnterOnAnEmptyLastRowLeavesTheTable(t *testing.T) {
	e := open("| a |\n| --- |\n| b |\n\nafter")
	at(e, 2, 3)
	keys(e, "enter")
	if !strings.HasSuffix(strings.Split(e.Text(), "\n\n")[0], "|     |") {
		t.Fatalf("enter on the last row adds one: %q", e.Text())
	}
	keys(e, "enter")
	want := "| a   |\n| --- |\n| b   |\n\n\nafter"
	if e.Text() != want || e.row != 4 || e.col != 0 {
		t.Errorf("enter on the empty row should drop it and leave the table:\n%q\nwant\n%q, cursor %d:%d", e.Text(), want, e.row, e.col)
	}
	typ(e, "text")
	if string(e.lines[4]) != "text" || string(e.lines[3]) != "" {
		t.Error("typing after leaving the table should be ordinary text, a blank line under the table")
	}
	e = open("| a |\n| --- |\n| b |")
	at(e, 2, 3)
	keys(e, "enter", "enter")
	if e.Text() != "| a   |\n| --- |\n| b   |\n\n" || e.row != 4 {
		t.Errorf("leaving a table at the end of the note: %q, cursor on %d", e.Text(), e.row)
	}
}

func TestTableKeysLeaveOtherLinesAlone(t *testing.T) {
	e := open("- item")
	at(e, 0, 6)
	keys(e, "tab")
	if e.Text() != "\t- item" && !strings.HasPrefix(e.Text(), " ") && !strings.HasPrefix(e.Text(), "\t") {
		t.Errorf("tab outside a table still indents: %q", e.Text())
	}
	e = open("```\n| not | a table\n```")
	at(e, 1, 5)
	keys(e, "tab")
	if strings.Contains(e.Text(), "---") {
		t.Errorf("a | line inside a code block isn't a table: %q", e.Text())
	}
}

func TestFormattingKeepsAlignmentWideCharsAndEscapes(t *testing.T) {
	e := open("| a | Number | x\\|y |\n|:-|--:|:-:|\n| ö | 7 | z |")
	at(e, 2, 2)
	if err := e.Table(TableFormat); err != nil {
		t.Fatal(err)
	}
	want := "| a   | Number | x\\|y |\n| :-- | -----: | :--: |\n| ö   |      7 |  z   |"
	if e.Text() != want {
		t.Errorf("got\n%s\nwant\n%s", e.Text(), want)
	}
}

func TestTableOps(t *testing.T) {
	base := "| n | v |\n|---|---|\n| b | 2 |\n| a | 10 |\n| c |  |"
	cases := []struct {
		op   TableOp
		row  int
		want string
	}{
		{TableRowBelow, 2, "| n   | v   |\n| --- | --- |\n| b   | 2   |\n|     |     |\n| a   | 10  |\n| c   |     |"},
		{TableRowDelete, 2, "| n   | v   |\n| --- | --- |\n| a   | 10  |\n| c   |     |"},
		{TableRowDown, 2, "| n   | v   |\n| --- | --- |\n| a   | 10  |\n| b   | 2   |\n| c   |     |"},
		{TableColRight, 2, "| n   |     | v   |\n| --- | --- | --- |\n| b   |     | 2   |\n| a   |     | 10  |\n| c   |     |     |"},
		{TableColDelete, 2, "| v   |\n| --- |\n| 2   |\n| 10  |\n|     |"},
		{TableColMoveRight, 2, "| v   | n   |\n| --- | --- |\n| 2   | b   |\n| 10  | a   |\n|     | c   |"},
		{TableAlignRight, 2, "|   n | v   |\n| --: | --- |\n|   b | 2   |\n|   a | 10  |\n|   c |     |"},
		{TableSortAsc, 2, "| n   | v   |\n| --- | --- |\n| a   | 10  |\n| b   | 2   |\n| c   |     |"},
		{TableSortDesc, 2, "| n   | v   |\n| --- | --- |\n| c   |     |\n| b   | 2   |\n| a   | 10  |"},
	}
	for _, c := range cases {
		e := open(base)
		at(e, c.row, 2)
		if err := e.Table(c.op); err != nil {
			t.Errorf("op %d: %v", c.op, err)
			continue
		}
		if e.Text() != c.want {
			t.Errorf("op %d:\ngot\n%s\nwant\n%s", c.op, e.Text(), c.want)
		}
		keys(e, "ctrl+z")
		if e.Text() != base {
			t.Errorf("op %d: one undo should take it back: %q", c.op, e.Text())
		}
	}
	e := open(base)
	at(e, 3, 6) // column v
	e.Table(TableSortDesc)
	if !strings.Contains(e.Text(), "| a   | 10  |\n| b   | 2   |\n| c   |     |") {
		t.Errorf("numbers sort as numbers, empty last:\n%s", e.Text())
	}
}

func TestTableOpsRefuseWhatTheyCant(t *testing.T) {
	e := open("plain")
	if err := e.Table(TableRowBelow); err != ErrNotInTable {
		t.Errorf("outside a table: %v", err)
	}
	e = open("| a |\n|---|\n| b |")
	at(e, 0, 2)
	for _, op := range []TableOp{TableRowDelete, TableRowUp, TableRowAbove} {
		if err := e.Table(op); err != ErrHeadingRow {
			t.Errorf("op %d on the heading row: %v", op, err)
		}
	}
	if err := e.Table(TableColDelete); err != ErrLastColumn {
		t.Errorf("deleting the only column: %v", err)
	}
	if err := e.Table(TableColMoveLeft); err != ErrTableEdge {
		t.Errorf("moving the first column left: %v", err)
	}
	if e.Text() != "| a |\n|---|\n| b |" {
		t.Errorf("a refusal changes nothing: %q", e.Text())
	}
}
