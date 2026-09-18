package markdown

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/theme"
)

// answer is a stand-in spread: it records the query it was given and
// replies with a two-row table.
type answer struct {
	got []string
	res SpreadResult
}

func (a *answer) run(q string) SpreadResult {
	a.got = append(a.got, q)
	return a.res
}

var twoBooks = SpreadResult{Markdown: "| Note | author |\n| --- | --- |\n| [[Books/Dune\\|Dune]] | Frank Herbert |\n| [[Books/Kallocain\\|Kallocain]] | Karin Boye |\n"}

func renderSpread(t *testing.T, src string, a *answer) []Line {
	t.Helper()
	return Render(src, Options{Width: 60, Palette: theme.Default(), Spreads: SpreadOptions{Run: a.run}})
}

func TestSpreadBlockShowsItsAnswer(t *testing.T) {
	a := &answer{res: twoBooks}
	src := "intro\n```spread\nTABLE author\nFROM #books\n```\nafter"
	lines := renderSpread(t, src, a)
	if len(a.got) != 1 || a.got[0] != "TABLE author\nFROM #books" {
		t.Fatalf("query passed = %q", a.got)
	}
	all := plainAll(lines)
	for _, s := range []string{"Dune", "Frank Herbert", "Kallocain", "Karin Boye", "after"} {
		if !strings.Contains(all, s) {
			t.Errorf("missing %q in\n%s", s, all)
		}
	}
	for _, s := range []string{"```", "TABLE author", "FROM #books"} {
		if strings.Contains(all, s) {
			t.Errorf("the query text %q is still showing:\n%s", s, all)
		}
	}
}

func TestSpreadRowsMapToTheOpeningFence(t *testing.T) {
	a := &answer{res: twoBooks}
	lines := renderSpread(t, "intro\n```spread\nTABLE author\n```\nafter", a)
	if lines[0].Src != 0 || lines[len(lines)-1].Src != 4 {
		t.Fatalf("intro/after lines map to %d/%d", lines[0].Src, lines[len(lines)-1].Src)
	}
	for _, l := range lines[1 : len(lines)-1] {
		if l.Src != 1 {
			t.Errorf("a result row maps to line %d, want the fence's line 1: %q", l.Src, ansi.Strip(l.Text))
		}
	}
}

func TestSpreadResultLinksCanBeFollowed(t *testing.T) {
	a := &answer{res: twoBooks}
	var targets []string
	for _, l := range renderSpread(t, "```spread\nTABLE author\n```", a) {
		for _, k := range l.Links {
			targets = append(targets, k.Target)
		}
	}
	if strings.Join(targets, ",") != "Books/Dune,Books/Kallocain" {
		t.Errorf("links = %v", targets)
	}
}

func TestDataviewFenceIsASpreadToo(t *testing.T) {
	a := &answer{res: SpreadResult{Markdown: "- [[Dune]]\n"}}
	renderSpread(t, "```dataview\nLIST\n```", a)
	renderSpread(t, "~~~ Spread\nLIST\n~~~", a)
	if len(a.got) != 2 {
		t.Errorf("ran %d times, want 2", len(a.got))
	}
}

func TestOtherFencesStayCode(t *testing.T) {
	a := &answer{res: twoBooks}
	for _, src := range []string{
		"```query\ntag:#books\n```",
		"```dataviewjs\ndv.list()\n```",
		"```spreadsheet\nx\n```",
		"```spread\nLIST\n", // never closed: code to the end, as in Obsidian
	} {
		lines := renderSpread(t, src, a)
		if !strings.Contains(plainAll(lines), "```") {
			t.Errorf("%q didn't render as code", src)
		}
	}
	if len(a.got) != 0 {
		t.Errorf("ran for %q", a.got)
	}
}

func TestSpreadsOffLeaveTheCode(t *testing.T) {
	lines := Render("```spread\nLIST\n```", Options{Width: 60, Palette: theme.Default()})
	if !strings.Contains(plainAll(lines), "LIST") {
		t.Error("with no Run, the block should show as code")
	}
}

func TestSpreadErrorsAndNotes(t *testing.T) {
	a := &answer{res: SpreadResult{Err: `Spread: expected TABLE or LIST, found "SHOW" (line 1)`}}
	lines := renderSpread(t, "```spread\nSHOW\n```", a)
	if len(lines) != 1 || ansi.Strip(lines[0].Text) != `⚠ Spread: expected TABLE or LIST, found "SHOW" (line 1)` {
		t.Errorf("got %q", plainAll(lines))
	}
	a = &answer{res: SpreadResult{Note: "No notes match"}}
	lines = renderSpread(t, "```spread\nLIST FROM #none\n```", a)
	if len(lines) != 1 || ansi.Strip(lines[0].Text) != "No notes match" || lines[0].Src != 0 {
		t.Errorf("got %q", plainAll(lines))
	}
	a = &answer{res: SpreadResult{Markdown: "- [[Dune]]\n", Note: "+3 more · add LIMIT or narrow FROM"}}
	lines = renderSpread(t, "```spread\nLIST\n```", a)
	if got := plainAll(lines); !strings.HasSuffix(got, "+3 more · add LIMIT or narrow FROM") {
		t.Errorf("got %q", got)
	}
}

func TestSpreadRunsInsideAnEmbeddedNote(t *testing.T) {
	a := &answer{res: SpreadResult{Markdown: "- [[Dune]]\n"}}
	lines := Render("![[Reading]]", Options{Width: 60, Palette: theme.Default(),
		Spreads: SpreadOptions{Run: a.run},
		Embeds:  EmbedOptions{Content: func(string) (string, bool) { return "```spread\nLIST\n```", true }}})
	if len(a.got) != 1 || !strings.Contains(plainAll(lines), "Dune") {
		t.Errorf("got %q (ran %d)", plainAll(lines), len(a.got))
	}
}

func plainAll(lines []Line) string {
	var b []string
	for _, l := range lines {
		b = append(b, ansi.Strip(l.Text))
	}
	return strings.Join(b, "\n")
}
