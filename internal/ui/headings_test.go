package ui

import (
	"os"
	"strings"
	"testing"
)

func TestHeadingRenamesSeesARenameAndNothingElse(t *testing.T) {
	for _, tc := range []struct {
		name          string
		before, after string
		want          []headingRename
	}{
		{"renamed", "# A\n## Morning\n", "# A\n## Sunrise\n", []headingRename{{"Morning", "Sunrise"}}},
		{"added", "# A\n", "# A\n## New\n", nil},
		{"removed", "# A\n## Gone\n", "# A\n", nil},
		{"only spacing and case", "# A\n## the  Morning\n", "# A\n## The Morning\n", nil},
		{"another level is a new heading", "# A\n## Morning\n", "# A\n### Morning walk\n", nil},
		{"the name lives on elsewhere", "# A\n## Morning\n### Morning\n", "# A\n## Dawn\n### Morning\n", nil},
		{"renamed with one added after it", "# A\n## Morning\n", "# A\n## Sunrise\n## Later\n", []headingRename{{"Morning", "Sunrise"}}},
		{"in a code fence", "# A\n```\n## Morning\n```\n", "# A\n```\n## Sunrise\n```\n", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := headingRenames(tc.before, tc.after)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

// headingLinkModel links to Stoic's "## Morning" from another note and from
// Stoic itself, and puts the cursor in the editor on the heading.
func headingLinkModel(t *testing.T) *Model {
	t.Helper()
	m := newTestModel(t)
	write := func(rel, body string) {
		if err := os.WriteFile(m.vault.Abs(rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("Welcome.md", "# Welcome\nSee [[Stoic#Morning]] and [[Stoic#Morning|dawn]].\n")
	write("Filosofi/Antik/Zeno.md", "# Zeno\nRead [[Stoic#Morning]] first.\n")
	write("Filosofi/Stoic.md", "# Stoic\n## Morning\nBack to [[#Morning]].\n")
	m.reload()
	m.openEditor("Filosofi/Stoic.md")
	m.editor.GoTo(1) // ## Morning
	press(m, "end")
	for range len("Morning") {
		press(m, "backspace")
	}
	typeText(m, "Sunrise")
	return m
}

func TestRenamingAHeadingOffersToBringTheLinksAlong(t *testing.T) {
	m := headingLinkModel(t)
	press(m, "esc") // save and close
	if m.confirm == nil || m.confirm.pill != " LINKS " {
		t.Fatalf("confirm = %+v, flash %q", m.confirm, m.flash)
	}
	if q := m.confirm.question; !strings.Contains(q, `"Morning" is now "Sunrise"`) ||
		!strings.Contains(q, "4 links in 3 notes") {
		t.Errorf("question = %q", q)
	}
	checkFrame(t, m, "heading rename offer")
	press(m, "y")
	for rel, want := range map[string]string{
		"Welcome.md":             "[[Stoic#Sunrise]] and [[Stoic#Sunrise|dawn]]",
		"Filosofi/Antik/Zeno.md": "[[Stoic#Sunrise]]",
		"Filosofi/Stoic.md":      "[[#Sunrise]]",
	} {
		if got := read(m, rel); !strings.Contains(got, want) {
			t.Errorf("%s = %q, want %q in it", rel, got, want)
		}
	}
	if !strings.Contains(m.flash, "4 links") {
		t.Errorf("flash = %q", m.flash)
	}
	press(m, "U")
	if got := read(m, "Welcome.md"); !strings.Contains(got, "[[Stoic#Morning]]") {
		t.Errorf("one U should put the links back: %q", got)
	}
	if got := read(m, "Filosofi/Stoic.md"); !strings.Contains(got, "## Sunrise") {
		t.Error("undoing the links shouldn't undo the heading you typed")
	}
}

func TestLeavingTheLinksIsAnAnswerToo(t *testing.T) {
	m := headingLinkModel(t)
	press(m, "esc")
	ops := m.journal.Len()
	press(m, "n")
	if got := read(m, "Welcome.md"); !strings.Contains(got, "[[Stoic#Morning]]") {
		t.Errorf("n should leave every link: %q", got)
	}
	if m.journal.Len() != ops {
		t.Error("nothing was changed, so there's nothing to undo")
	}
	if !strings.Contains(m.flash, "left as they were") {
		t.Errorf("flash = %q", m.flash)
	}
}

func TestAHeadingNobodyLinksToAsksNothing(t *testing.T) {
	m := newTestModel(t)
	m.openEditor("Filosofi/Stoic.md")
	m.editor.GoTo(4) // ## Morning, under the frontmatter
	press(m, "end")
	typeText(m, " walk")
	press(m, "esc")
	if m.confirm != nil {
		t.Fatalf("no link points at it: %+v", m.confirm)
	}
	if !strings.Contains(read(m, "Filosofi/Stoic.md"), "## Morning walk") {
		t.Error("the rename itself should still be saved")
	}
}
