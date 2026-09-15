package markdown

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/theme"
)

const sample = `---
title: Test
tags: [stoa, filosofi/antik]
---
# Heading one
Some text with [[Note|an alias]], [[Missing]] and ` + "`code **x**`" + `.
## Second
- [ ] open task
- [x] done task
> [!tip] Remember
> Inside the callout
` + "```go" + `
**not bold** [[not a link]]
` + "```" + `
Tagged #idea but not a#tag.
`

func render(t *testing.T, src string, width int) []Line {
	t.Helper()
	lines := Render(src, Options{Width: width, Palette: theme.Default(), Resolve: func(s string) bool { return s == "Note" }})
	for i, l := range lines {
		if w := ansi.StringWidth(l.Text); w > width {
			t.Errorf("line %d is %d cells wide, max %d: %q", i, w, width, ansi.Strip(l.Text))
		}
	}
	return lines
}

// plainAt returns the stripped text of all display lines from source line src.
func plainAt(lines []Line, src int) string {
	var parts []string
	for _, l := range lines {
		if l.Src == src {
			parts = append(parts, ansi.Strip(l.Text))
		}
	}
	return strings.Join(parts, "\n")
}

func TestHeadingsMapToSourceLines(t *testing.T) {
	lines := render(t, sample, 60)
	got := map[int]int{}
	for _, l := range lines {
		if l.Heading > 0 {
			got[l.Heading] = l.Src
		}
	}
	if got[1] != 4 || got[2] != 6 {
		t.Errorf("heading → source line = %v, want h1→4, h2→6", got)
	}
	if s := plainAt(lines, 4); s != "Heading one" {
		t.Errorf("h1 renders as %q", s)
	}
}

func TestFrontmatterBecomesProperties(t *testing.T) {
	lines := render(t, sample, 60)
	if s := plainAt(lines, 2); !strings.Contains(s, "#stoa #filosofi/antik") {
		t.Errorf("tags property renders as %q", s)
	}
	if s := plainAt(lines, 0); s != "" {
		t.Errorf("opening fence should not render, got %q", s)
	}
}

func TestInlineMarkup(t *testing.T) {
	lines := render(t, sample, 80)
	s := plainAt(lines, 5)
	if !strings.Contains(s, "an alias") || strings.Contains(s, "[[") {
		t.Errorf("wikilinks not rendered by display text: %q", s)
	}
	if !strings.Contains(s, "code **x**") {
		t.Errorf("code span should stay literal: %q", s)
	}
	if s := plainAt(lines, 7); !strings.HasPrefix(s, "☐ open task") {
		t.Errorf("open task renders as %q", s)
	}
	if s := plainAt(lines, 9); !strings.Contains(s, "Remember") {
		t.Errorf("callout title renders as %q", s)
	}
	if s := plainAt(lines, 12); !strings.Contains(s, "**not bold** [[not a link]]") {
		t.Errorf("code block content should stay literal: %q", s)
	}
	if s := plainAt(lines, 14); !strings.Contains(s, "#idea") || !strings.Contains(s, "a#tag") {
		t.Errorf("tag line renders as %q", s)
	}
}

func TestLinksKnowTheirColumns(t *testing.T) {
	lines := render(t, "See [[Stoic#Morning|the morning]] and [web](https://x.com) or ![[Zeno]]", 80)
	want := []Link{
		{Col: 4, Target: "Stoic#Morning", Wiki: true},
		{Col: 20, Target: "https://x.com"},
		{Col: 27, Target: "Zeno", Wiki: true},
	}
	if got := lines[0].Links; len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("links = %+v\nwant %+v", got, want)
	}
	table := render(t, "| a | b |\n|---|---|\n| x | [[Note\\|alias]] |", 40)
	if l := table[2].Links; len(l) != 1 || l[0].Target != "Note" || l[0].Col != 6 {
		t.Errorf("table link = %+v", l)
	}
}

func TestBrokenLinksAreStyledDifferently(t *testing.T) {
	good := Render("[[Note]]", Options{Width: 40, Palette: theme.Default(), Resolve: func(string) bool { return true }})
	bad := Render("[[Note]]", Options{Width: 40, Palette: theme.Default(), Resolve: func(string) bool { return false }})
	if good[0].Text == bad[0].Text {
		t.Error("resolved and unresolved links look identical")
	}
}

func TestTableColumnsAlign(t *testing.T) {
	src := "intro\n| Key | Action |\n|---|---|\n| `j` | down |\n| ctrl+d | half page [[Note\\|down]] |\n"
	lines := render(t, src, 60)
	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5", len(lines))
	}
	col := -1
	for k, l := range lines[1:] {
		if l.Src != k+1 {
			t.Errorf("table row %d maps to source line %d", k, l.Src)
		}
		// Position of the column divider between the two cells.
		runes := []rune(ansi.Strip(l.Text))
		pos, seen := -1, 0
		for i, c := range runes {
			if c == '│' || c == '├' || c == '┼' {
				if seen++; seen == 2 {
					pos = i
					break
				}
			}
		}
		if col == -1 {
			col = pos
		} else if pos != col {
			t.Errorf("row %d divider at %d, want %d: %q", k, pos, col, string(runes))
		}
	}
	if s := plainAt(lines, 4); !strings.Contains(s, "half page down") {
		t.Errorf("escaped pipe in wikilink broke the cell: %q", s)
	}
}

func TestWordsCrossingStylesStayTogether(t *testing.T) {
	// "`Daily/`," is one word even though it spans two styles.
	src := "This is the one exception: notes always go in `Daily/`, never elsewhere."
	for width := 20; width <= 60; width++ {
		for _, l := range render(t, src, width) {
			if strings.HasPrefix(ansi.Strip(l.Text), ",") {
				t.Fatalf("width %d: line starts with a comma torn off its word", width)
			}
		}
	}
}

func TestNarrowTableClips(t *testing.T) {
	src := "| Topic | Decision |\n|---|---|\n| Stack | Go with Bubble Tea, Bubbles and Lip Gloss in one binary |\n"
	lines := render(t, src, 30)
	if s := plainAt(lines, 2); !strings.Contains(s, "…") {
		t.Errorf("long cell not clipped: %q", s)
	}
}

func TestWrappingKeepsSourceLine(t *testing.T) {
	src := "intro\n" + strings.Repeat("stoicism teaches calm ", 10) + "\nend"
	lines := render(t, src, 24)
	n := 0
	for _, l := range lines {
		if l.Src == 1 {
			n++
		}
	}
	if n < 5 {
		t.Errorf("long paragraph wrapped into %d lines, want several", n)
	}
	if last := lines[len(lines)-1]; last.Src != 2 || ansi.Strip(last.Text) != "end" {
		t.Errorf("last line = %+v", last)
	}
}
