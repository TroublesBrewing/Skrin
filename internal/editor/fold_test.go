package editor

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/theme"
)

// folded is a note with a spread in the middle: lines 1–3 fold into three
// rows while the cursor is outside them.
func folded(t *testing.T) *Editor {
	t.Helper()
	e := New("above\n```spread\nLIST\n```\nbelow", false, theme.Default())
	e.SetSize(40, 10)
	e.SetFolds([]Fold{{Start: 1, End: 3, Rows: []string{"hint", "row one", "row two"}}})
	return e
}

func view(e *Editor) string {
	var out []string
	for _, l := range e.View() {
		if l = ansi.Strip(l); l != "" {
			out = append(out, l)
		}
	}
	return strings.Join(out, "|")
}

func TestAFoldShowsItsRowsWhileTheCursorIsOutside(t *testing.T) {
	e := folded(t)
	if got := view(e); got != "above|hint|row one|row two|below" {
		t.Errorf("view = %q", got)
	}
	if got := e.displayIndex(4, 0); got != 4 {
		t.Errorf("below sits on display row %d, want 4", got)
	}
}

func TestSteppingOntoAFoldOpensIt(t *testing.T) {
	e := folded(t)
	e.HandleKey(key("down"))
	if e.row != 1 || view(e) != "above|```spread|LIST|```|below" {
		t.Errorf("down from above: row %d, view %q", e.row, view(e))
	}
	e.HandleKey(key("down"))
	e.HandleKey(key("down"))
	e.HandleKey(key("down")) // out of it, onto below
	if e.row != 4 || view(e) != "above|hint|row one|row two|below" {
		t.Errorf("leaving: row %d, view %q", e.row, view(e))
	}
	e.HandleKey(key("up")) // up into it lands on the closing fence
	if e.row != 3 || !strings.Contains(view(e), "LIST") {
		t.Errorf("up from below: row %d, view %q", e.row, view(e))
	}
}

func TestScrollingCountsFoldedRows(t *testing.T) {
	e := New("```spread\nLIST\n```\n"+strings.Repeat("line\n", 20)+"end", false, theme.Default())
	e.SetSize(40, 5)
	e.SetFolds([]Fold{{Start: 0, End: 2, Rows: []string{"h", "1", "2", "3", "4", "5", "6"}}})
	e.GoTo(23) // "end", line 23: 7 folded rows + 20 lines before it
	if e.top != e.displayIndex(23, 0) || e.displayIndex(23, 0) != 27 {
		t.Errorf("top %d, end at %d, want 27", e.top, e.displayIndex(23, 0))
	}
	if e.TopRow() != 23 {
		t.Errorf("TopRow = %d", e.TopRow())
	}
	if r, _ := e.CursorPos(); r != 0 {
		t.Errorf("the cursor should be on the top row, got %d", r)
	}
}

func TestFoldRowsGetOneLineNumber(t *testing.T) {
	e := folded(t)
	e.SetLineNumbers(true)
	got := strings.Split(view(e), "|")
	if !strings.HasPrefix(got[1], " 2 │ hint") || !strings.HasPrefix(got[2], "   │ row one") || !strings.HasPrefix(got[4], " 5 │ below") {
		t.Errorf("gutter: %q", got)
	}
}

func TestStaleFoldsAreDropped(t *testing.T) {
	e := New("one\ntwo", false, theme.Default())
	e.SetFolds([]Fold{{Start: 1, End: 5, Rows: []string{"x"}}, {Start: 0, End: 0}})
	if len(e.folds) != 0 {
		t.Errorf("folds past the end, or with no rows, should be dropped: %+v", e.folds)
	}
}
