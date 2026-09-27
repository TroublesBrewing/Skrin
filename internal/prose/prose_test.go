package prose

import (
	"strings"
	"testing"
)

// at reports whether the substring want is prose in s.
func at(t *testing.T, s, want string, frontmatter bool) bool {
	t.Helper()
	keep := Mask(s)
	if frontmatter {
		keep = MaskFrontmatter(keep, s)
	}
	i := strings.Index(s, want)
	if i < 0 {
		t.Fatalf("setup: %q is not in the text", want)
	}
	return Free(keep, i, i+len(want))
}

func TestPlainTextIsProse(t *testing.T) {
	if !at(t, "A sentence about Stoic.\n", "Stoic", false) {
		t.Error("a word in a sentence should be prose")
	}
}

func TestLinksAreNotProse(t *testing.T) {
	for _, s := range []string{
		"See [[Stoic]].\n",
		"An embed ![[Stoic]].\n",
		"An alias [[Stoic|the school]].\n",
		"A markdown link [Stoic](Filosofi/Stoic.md).\n",
	} {
		if at(t, s, "Stoic", false) {
			t.Errorf("%q: what is already a link is not prose", s)
		}
	}
}

func TestCodeIsNotProse(t *testing.T) {
	if at(t, "Inline `Stoic` here.\n", "Stoic", false) {
		t.Error("an inline code span is not prose")
	}
	if at(t, "```\nStoic\n```\n", "Stoic", false) {
		t.Error("a fenced block is not prose")
	}
	// An unterminated fence protects the rest of the note, rather than
	// nothing: guessing the writer's intent the safe way round.
	if at(t, "```\nStoic\n", "Stoic", false) {
		t.Error("an unclosed fence should still protect what follows it")
	}
}

func TestFrontmatterIsProseUntilAskedOtherwise(t *testing.T) {
	s := "---\nauthor: Stoic\n---\nText.\n"
	if !at(t, s, "Stoic", false) {
		t.Error("Mask alone leaves properties as prose: a tag lives there too")
	}
	if at(t, s, "Stoic", true) {
		t.Error("MaskFrontmatter should take the property block out")
	}
}

func TestFreeIsTrueForAnEmptySpanAndSafePastTheEnd(t *testing.T) {
	keep := Mask("abc")
	if !Free(keep, 1, 1) {
		t.Error("an empty span is free")
	}
	if !Free(keep, 2, 99) {
		t.Error("a span past the end should not panic or report false")
	}
}
