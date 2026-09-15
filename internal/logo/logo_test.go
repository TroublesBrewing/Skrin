package logo

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestHalfBlockSize(t *testing.T) {
	for _, rows := range []int{3, 8} {
		lines, w, err := HalfBlock(rows)
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) != rows {
			t.Fatalf("got %d rows, want %d", len(lines), rows)
		}
		if w < rows/2 || w > rows*3 {
			t.Errorf("width %d looks wrong for %d rows", w, rows)
		}
		drawn := false
		for _, l := range lines {
			if got := ansi.StringWidth(l); got != w {
				t.Errorf("row is %d cells, want %d", got, w)
			}
			if ansi.Strip(l) != "" && len(ansi.Strip(l)) > 0 {
				drawn = true
			}
		}
		if !drawn {
			t.Error("logo rendered nothing")
		}
	}
}
