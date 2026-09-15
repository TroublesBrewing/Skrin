package logo

import (
	"image/color"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/theme"
)

func TestBitmapsAreWellFormed(t *testing.T) {
	for _, bm := range [][]string{small, large} {
		if len(bm)%2 != 0 {
			t.Errorf("%d pixel rows; they must pair up into cells", len(bm))
		}
		for _, r := range bm {
			if len(r) != len(bm[0]) || strings.Trim(r, ".ADL") != "" {
				t.Errorf("bad row %q", r)
			}
		}
	}
}

func TestSizes(t *testing.T) {
	p := theme.Default()
	check := func(what string, lines []string, w, wantW, wantH int) {
		t.Helper()
		if len(lines) != wantH || w != wantW {
			t.Fatalf("%s: %d×%d cells, want %d×%d", what, w, len(lines), wantW, wantH)
		}
		for _, l := range lines {
			if got := ansi.StringWidth(l); got != w {
				t.Errorf("%s: row is %d cells, want %d", what, got, w)
			}
		}
	}
	lines, w := Header(p)
	check("header", lines, w, 8, 3)
	for scale := 1; scale <= 2; scale++ {
		lines, w := Splash(p, scale)
		sw, sh := SplashSize(scale)
		check("splash", lines, w, sw, sh)
		if sw != 20*scale || sh != 7*scale {
			t.Errorf("SplashSize(%d) = %d×%d", scale, sw, sh)
		}
	}
}

func TestFollowsTheTheme(t *testing.T) {
	a := theme.Default()
	b := a
	b.Accent = color.NRGBA{0xff, 0, 0, 0xff}
	la, _ := Header(a)
	lb, _ := Header(b)
	if strings.Join(la, "\n") == strings.Join(lb, "\n") {
		t.Error("the logo ignored the theme's accent colour")
	}
}
