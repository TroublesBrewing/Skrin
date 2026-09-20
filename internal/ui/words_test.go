package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestCountingWords(t *testing.T) {
	for _, c := range []struct {
		name, src string
		want      int
	}{
		{"plain", "one two three\n", 3},
		{"frontmatter doesn't count", "---\ntitle: A long title here\ntags: [a, b]\n---\none two\n", 2},
		{"markup isn't words", "# A heading\n- a bullet\n> a quote\n", 6},
		{"empty", "", 0},
		{"punctuation alone isn't a word", "hello — world\n", 2},
	} {
		if got := countWords(c.src); got != c.want {
			t.Errorf("%s: %d words, want %d", c.name, got, c.want)
		}
	}
}

func TestTheStatusLineCountsTheOpenNote(t *testing.T) {
	m := newTestModel(t)
	if err := os.WriteFile(m.vault.Abs("Welcome.md"), []byte("# Welcome\none two three four\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.reload()
	press(m, "G", "l")
	if s := ansi.Strip(m.statusLine()); !strings.Contains(s, "5 words") {
		t.Errorf("reading: %q", s)
	}
	press(m, "e")
	typeText(m, " five")
	if s := ansi.Strip(m.statusLine()); !strings.Contains(s, "6 words") {
		t.Errorf("the editor should count what's there now, not what's saved: %q", s)
	}
}

func TestASelectionIsCountedInstead(t *testing.T) {
	m := newTestModel(t)
	if err := os.WriteFile(m.vault.Abs("Welcome.md"), []byte("# Welcome\none two three four\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.reload()
	press(m, "G", "l", "v") // one line selected: the heading, one word
	if s := ansi.Strip(m.statusLine()); !strings.Contains(s, "1 word selected") {
		t.Errorf("status = %q", s)
	}
}
