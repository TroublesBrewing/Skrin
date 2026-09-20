package markdown

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
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

// "# Title ##" closes the heading with markup, not text. Obsidian hides the
// trailing hashes and internal/index already strips them, so a heading must
// not read differently depending on which of them you ask.
func TestClosingHashesAreMarkup(t *testing.T) {
	lines := render(t, "# Stoic ideas ##\nbody\n## Second #\n### Kept #tag\n#### Spaced   \n", 60)
	for src, want := range map[int]string{
		0: "Stoic ideas",
		2: "Second",
		3: "Kept #tag", // a tag isn't a closing hash
		4: "Spaced",
	} {
		if s := plainAt(lines, src); s != want {
			t.Errorf("source line %d renders as %q, want %q", src, s, want)
		}
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

func TestNarrowTableCellsWrap(t *testing.T) {
	src := "| Topic | Decision |\n|---|---|\n| Stack | Go with Bubble Tea, Bubbles and Lip Gloss in one binary |\n"
	lines := render(t, src, 30)
	var rows []Line
	for _, l := range lines {
		if l.Src == 2 {
			rows = append(rows, l)
		}
	}
	if len(rows) < 2 {
		t.Fatalf("long cell should wrap onto several display rows, got %d: %q", len(rows), plainAt(lines, 2))
	}
	full := plainAt(lines, 2)
	if strings.Contains(full, "…") {
		t.Errorf("cell was clipped instead of wrapped: %q", full)
	}
	for _, word := range []string{"Bubble", "Tea,", "Bubbles", "Lip", "Gloss", "binary"} {
		if !strings.Contains(full, word) {
			t.Errorf("wrapped cell lost %q: %q", word, full)
		}
	}
	if !strings.Contains(ansi.Strip(rows[0].Text), "Stack") {
		t.Errorf("first row should still show the short cell: %q", ansi.Strip(rows[0].Text))
	}
	for _, r := range rows[1:] {
		if strings.Contains(ansi.Strip(r.Text), "Stack") {
			t.Errorf("continuation row repeats the short cell instead of blanking it: %q", ansi.Strip(r.Text))
		}
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

func TestTableRowsAreStriped(t *testing.T) {
	src := "| Book | Author |\n|---|---|\n| Dune | Herbert |\n| Kallocain | Karin Boye, who also wrote poetry and more |\n| Thinking | Kahneman |\n| Walden | Thoreau |\n"
	lines := render(t, src, 40)
	bg := lipgloss.NewStyle().Background(theme.Default().LighterBackground).Render("x")
	stripe := bg[:strings.Index(bg, "x")] // the escape that turns the stripe on
	byRow := map[int][]string{}
	for _, l := range lines {
		byRow[l.Src] = append(byRow[l.Src], l.Text)
	}
	for src, want := range map[int]bool{0: false, 2: false, 3: true, 4: false, 5: true} {
		for _, text := range byRow[src] {
			if got := strings.Contains(text, stripe); got != want {
				t.Errorf("source line %d striped = %v, want %v: %q", src, got, want, ansi.Strip(text))
			}
		}
	}
	if len(byRow[3]) < 2 {
		t.Errorf("the long row should wrap: %d display lines", len(byRow[3]))
	}
	w := ansi.StringWidth(lines[0].Text)
	for _, l := range lines {
		if ansi.StringWidth(l.Text) != w {
			t.Errorf("rows don't line up: %q is %d wide, want %d", ansi.Strip(l.Text), ansi.StringWidth(l.Text), w)
		}
	}
}

// Footnotes: Obsidian numbers them in the order the labels first appear,
// and shows the number, never the label.
func TestFootnotesAreNumberedInOrder(t *testing.T) {
	src := "Stoicism[^why] och ödet[^fate].\n\n[^fate]: Om kausalitet.\n[^why]: Därför.\n"
	got := allText(render(t, src, 60))
	for _, want := range []string{"Stoicism[1] och ödet[2].", "[2] Om kausalitet.", "[1] Därför."} {
		if !strings.Contains(got, want) {
			t.Errorf("want %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "why") || strings.Contains(got, "fate") {
		t.Errorf("the label is markup, not text:\n%s", got)
	}
}

func TestTheSameFootnoteTwiceKeepsItsNumber(t *testing.T) {
	src := "Ett[^a] två[^b] ett igen[^a].\n"
	got := allText(render(t, src, 60))
	if !strings.Contains(got, "Ett[1] två[2] ett igen[1].") {
		t.Errorf("got:\n%s", got)
	}
}

func TestAFootnoteWithoutItsPairStillRenders(t *testing.T) {
	src := "En referens utan definition[^lost].\n\n[^orphan]: en definition utan referens.\n"
	got := allText(render(t, src, 60))
	if !strings.Contains(got, "En referens utan definition[1].") ||
		!strings.Contains(got, "[2] en definition utan referens.") {
		t.Errorf("neither half should vanish:\n%s", got)
	}
}

// allText is every display line, stripped, one per row.
func allText(lines []Line) string {
	var parts []string
	for _, l := range lines {
		parts = append(parts, ansi.Strip(l.Text))
	}
	return strings.Join(parts, "\n")
}
