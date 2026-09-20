package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/theme"
)

// The grain paints behind the text, so whatever it does it must not
// change a single character or a single cell of width.
func TestGrainLeavesTheTextExactlyAsItWas(t *testing.T) {
	sh := shades(theme.Default(), 3)
	st := newStyles(theme.Default())
	for _, line := range []string{
		"plain text, nothing special",
		"  " + st.bold.Render("# A heading") + " and then some",
		strings.Repeat(" ", 40),
		"wide runes: 日本語 mixed with latin",
	} {
		line = fit(line, 40)
		got := grain(line, 3, 40, sh)
		if a, b := ansi.Strip(got), ansi.Strip(line); a != b {
			t.Errorf("text changed:\n got %q\nwant %q", a, b)
		}
		if w := ansi.StringWidth(got); w != 40 {
			t.Errorf("width = %d, want 40", w)
		}
	}
}

// It must be the same every frame: paper that shimmered while you read
// would be worse than no paper at all.
func TestTheGrainHoldsStill(t *testing.T) {
	sh := shades(theme.Default(), 3)
	line := fit("the same line", 30)
	if grain(line, 7, 30, sh) != grain(line, 7, 30, sh) {
		t.Error("the same row must come out the same")
	}
	if grain(line, 7, 30, sh) == grain(line, 8, 30, sh) {
		t.Error("but two rows shouldn't be identical, or there is no grain")
	}
}

func TestTheGrainIsOffUnlessAskedFor(t *testing.T) {
	// Other things paint backgrounds too (a selection, the status pill),
	// so the test counts them rather than looking for any at all.
	m := newTestModel(t)
	press(m, "G", "l")
	plain := strings.Count(m.render(), "48;2;")

	m.opts.Paper = true // without beta mode it still stays off
	if m.paperOn() || strings.Count(m.render(), "48;2;") != plain {
		t.Error("an experiment needs beta mode too")
	}
	m.opts.Beta = true
	if !m.paperOn() {
		t.Fatal("with both on, the grain should be on")
	}
	if got := strings.Count(m.render(), "48;2;"); got < plain+50 {
		t.Errorf("backgrounds painted: %d with the grain, %d without", got, plain)
	}
}

// The frame around the note is furniture, not paper.
func TestTheGrainStaysInsideTheBorders(t *testing.T) {
	m := newTestModel(t)
	m.opts.Beta, m.opts.Paper = true, true
	press(m, "G", "l")
	for _, row := range strings.Split(m.render(), "\n") {
		if !strings.Contains(row, "│") {
			continue
		}
		if i := strings.Index(row, "48;2;"); i >= 0 && i < strings.Index(row, "│") {
			t.Errorf("grain before the first border: %q", ansi.Strip(row))
		}
	}
}

// The grain is only worth painting if the shades differ enough to see.
// Three steps of 255 is about one per cent, which is why the first try
// was invisible on a light theme.
func TestTheShadesActuallyDiffer(t *testing.T) {
	light := theme.Builtin("flexoki-light")
	for _, step := range []int{4, 8, 14} {
		sh := shades(light, step)
		if len(sh) < 2 {
			t.Fatal("a grain needs shades")
		}
		seen := map[string]bool{}
		for _, s := range sh {
			seen[s] = true
		}
		if len(seen) != len(sh) {
			t.Errorf("step %d: shades repeat, so there is no grain: %q", step, sh)
		}
	}
}

// A near-white background has nowhere to lighten to, so the grain goes
// darker there — and lighter on a dark theme. Every shade must be its own
// colour either way, or part of the grain is invisible.
func TestTheGrainGoesWhereThereIsRoom(t *testing.T) {
	for _, name := range []string{"flexoki-light", "gruvbox"} {
		p := theme.Builtin(name)
		seen := map[string]bool{}
		for _, s := range shades(p, 8) {
			seen[s] = true
		}
		if len(seen) != grainShades {
			t.Errorf("%s: %d shades of %d are distinct", name, len(seen), grainShades)
		}
	}
}
