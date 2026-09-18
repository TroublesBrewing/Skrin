package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTableMarkdownPadsAndLeavesEmptyHeadingsToFill(t *testing.T) {
	f := &tableForm{cols: 2, rows: 1}
	want := []string{"|     |     |", "| --- | --- |", "|     |     |"}
	if got := f.markdown(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("no headings:\n%s", strings.Join(got, "\n"))
	}
	f.heads.set("Name, Rating, Date | time")
	got := f.markdown()
	if got[0] != `| Name | Rating | Date \| time |` || got[1] != "| ---- | ------ | ------------ |" {
		t.Errorf("headings widen the table and pad the columns, a | escaped:\n%s", strings.Join(got, "\n"))
	}
	if c, r := f.size(); c != 3 || r != 1 {
		t.Errorf("three headings make three columns: %d × %d", c, r)
	}
}

func TestInsertTableFromTheEditorPalette(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "enter") // Welcome, straight into editing
	runCommand(m, "table")
	if m.table == nil {
		t.Fatal("Insert table should open the form")
	}
	checkFrame(t, m, "table form")
	press(m, "left")      // 2 columns
	press(m, "down", "4") // 4 rows
	press(m, "tab")       // headings
	typeText(m, "Book, Grade")
	if frame := ansi.Strip(m.render()); !strings.Contains(frame, "| Book | Grade |") {
		t.Errorf("the form should preview the table:\n%s", frame)
	}
	press(m, "enter")
	if m.table != nil || m.editor == nil {
		t.Fatal("enter should insert and go back to editing")
	}
	text := m.editor.Text()
	if !strings.Contains(text, "| Book | Grade |\n| ---- | ----- |\n|      |       |") || strings.Count(text, "|      |       |") != 4 {
		t.Errorf("inserted:\n%s", text)
	}
	if !strings.HasPrefix(m.flash, "Table inserted: 2 columns × 4 rows") {
		t.Errorf("flash %q", m.flash)
	}
	typeText(m, "Meditations")
	if !strings.Contains(m.editor.Text(), "| ---- | ----- |\n| Meditations     |       |") {
		t.Errorf("typing should land in the first cell under the headings:\n%s", m.editor.Text())
	}
}

func TestInsertTableFromReadingOpensTheEditorFirst(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l") // Welcome, reading
	runCommand(m, "insert table")
	if m.editor == nil || m.table == nil {
		t.Fatalf("from reading, the note should open in the editor under the form: editor %v, form %v", m.editor != nil, m.table != nil)
	}
	press(m, "enter")
	press(m, "ctrl+s")
	b, _ := os.ReadFile(m.vault.Abs("Welcome.md"))
	if !strings.Contains(string(b), "|     |     |     |\n| --- | --- | --- |") {
		t.Errorf("the default 3 × 2 table should be saved in the note:\n%s", b)
	}
}

func TestInsertTableRefusesWithoutANote(t *testing.T) {
	m := newTestModel(t) // on the vault root
	runCommand(m, "insert table")
	if m.table != nil || m.flash != "Select a note to put a table in" {
		t.Errorf("form %v, flash %q", m.table != nil, m.flash)
	}
}

func TestTableFormNumbersTypeLikeANumberField(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "enter")
	runCommand(m, "table")
	f := m.table
	press(m, "5")
	if f.cols != 5 {
		t.Errorf("the first digit replaces the 3 that was there: %d", f.cols)
	}
	press(m, "1")
	if f.cols != 1 {
		t.Errorf("51 is past the limit of %d, so 1 starts again: %d", tableMaxCols, f.cols)
	}
	press(m, "0")
	if f.cols != 10 {
		t.Errorf("1 then 0 is 10: %d", f.cols)
	}
	press(m, "right", "right", "right", "right")
	if f.cols != tableMaxCols {
		t.Errorf("columns stop at %d: %d", tableMaxCols, f.cols)
	}
	press(m, "down")
	for range 5 {
		press(m, "left")
	}
	if f.rows != 1 {
		t.Errorf("rows stop at 1: %d", f.rows)
	}
	press(m, "backspace")
	if f.rows != 1 {
		t.Errorf("backspace can't empty a number: %d", f.rows)
	}
}

func TestEscClosesTheTableFormAndInsertsNothing(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "enter")
	before := m.editor.Text()
	runCommand(m, "table")
	press(m, "esc")
	if m.table != nil || m.editor == nil || m.editor.Text() != before {
		t.Error("esc should close the form, keep editing, and change nothing")
	}
}

func TestTableCommandsShowOnlyInATable(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "enter") // Welcome, editing, on "# Welcome"
	press(m, "ctrl+p")
	for _, l := range paletteLabels(m) {
		if strings.HasPrefix(l, "Table:") {
			t.Fatalf("outside a table the palette shouldn't list %q", l)
		}
	}
	press(m, "esc")
	runCommand(m, "insert table")
	press(m, "enter") // a 3 × 2 table, cursor in the first heading
	press(m, "ctrl+p")
	paletteItem(t, m, "Table: add a column to the right")
	if s := ansi.Strip(m.editLine()); !strings.Contains(s, "tab next cell") || !strings.Contains(s, "ctrl+p table commands") {
		t.Errorf("in a table the status line names its keys: %q", s)
	}
}

func TestTableCommandFromThePalette(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "enter")
	runCommand(m, "insert table")
	press(m, "enter")
	typeText(m, "Title")
	press(m, "tab")
	typeText(m, "Year")
	runCommand(m, "table delete column")
	if m.flash != "Column deleted · ctrl+z brings it back" {
		t.Errorf("flash %q", m.flash)
	}
	if !strings.Contains(m.editor.Text(), "| Title |     |\n| ----- | --- |") {
		t.Errorf("the Year column should be gone:\n%s", m.editor.Text())
	}
	runCommand(m, "table delete this row")
	if m.flash != "The heading row stays: move to a row below it" {
		t.Errorf("a refusal should say why: %q", m.flash)
	}
}

func TestTableFormArrowsMatchWhatTheyShow(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "enter")
	runCommand(m, "table")
	f := m.table
	press(m, "right")
	if f.cols != 4 || f.field != tableCols {
		t.Errorf("→ on ‹ 3 › should make it 4 and stay on the field: %d, field %d", f.cols, f.field)
	}
	press(m, "left", "left")
	if f.cols != 2 {
		t.Errorf("← should make it one fewer: %d", f.cols)
	}
	press(m, "down")
	if f.field != tableRows || f.cols != 2 {
		t.Errorf("↓ should move to the next field and leave the number alone: field %d, cols %d", f.field, f.cols)
	}
	press(m, "down")
	typeText(m, "Ab")
	press(m, "left")
	typeText(m, "x")
	if f.heads.value() != "Axb" {
		t.Errorf("← in the headings field moves the text cursor: %q", f.heads.value())
	}
	press(m, "up")
	if f.field != tableRows {
		t.Errorf("↑ should go back a field: %d", f.field)
	}
}
