package ui

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/lurioso/skrin/internal/theme"
)

// Files says what a row is with colour, so no two of its meanings may wear
// the same one. Until v0.11 the open note was the accent colour, and
// gruvbox gives folders that very hex, so on the default theme the open
// note couldn't be told from a folder.
func TestFilesColoursTellItsRowsApart(t *testing.T) {
	st := newStyles(theme.Default())
	for _, c := range []struct {
		what string
		a, b lipgloss.Style
	}{
		{"the open note and a folder", st.open, st.dir},
		{"the open note and a marked row", st.open, st.marked},
		{"the open note and an ordinary note", st.open, st.text},
		{"a folder and a marked row", st.dir, st.marked},
	} {
		if sameColour(c.a.GetForeground(), c.b.GetForeground()) {
			t.Errorf("%s share a colour: %v", c.what, c.a.GetForeground())
		}
	}
}

func sameColour(a, b color.Color) bool {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	return ar == br && ag == bg && ab == bb
}
