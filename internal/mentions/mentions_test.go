package mentions

import (
	"strings"
	"testing"
)

func TestAPlainMentionIsFoundWithItsLineAndLine(t *testing.T) {
	docs := []Doc{{Rel: "Notes/Reading.md", Content: "# Reading\n\nI read Stoic again today.\n"}}
	hits := Find(docs, []string{"Stoic"}, "Filosofi/Stoic.md")
	if len(hits) != 1 {
		t.Fatalf("hits = %+v, want one", hits)
	}
	h := hits[0]
	if h.Rel != "Notes/Reading.md" || h.Line != 2 {
		t.Errorf("hit = %+v, want the note and its third line", h)
	}
	if h.Text != "Stoic" {
		t.Errorf("text = %q, want the words as written", h.Text)
	}
	if h.Context != "I read Stoic again today." {
		t.Errorf("context = %q, want the whole line", h.Context)
	}
}

func TestTheNoteItselfIsNeverAMention(t *testing.T) {
	docs := []Doc{{Rel: "Stoic.md", Content: "# Stoic\n\nStoic is what this note is.\n"}}
	if hits := Find(docs, []string{"Stoic"}, "Stoic.md"); len(hits) != 0 {
		t.Errorf("a note mentioning its own name is not a missing link: %+v", hits)
	}
}

func TestAlreadyLinkedIsNotAMention(t *testing.T) {
	docs := []Doc{{Rel: "A.md", Content: strings.Join([]string{
		"See [[Stoic]] for more.",
		"And [[Stoic|the Stoics]] too.",
		"An embed: ![[Stoic]]",
		"A markdown link: [Stoic](Filosofi/Stoic.md)",
		"But Stoic here is plain.",
	}, "\n")}}
	hits := Find(docs, []string{"Stoic"}, "Stoic.md")
	if len(hits) != 1 {
		t.Fatalf("hits = %d, want only the plain one:\n%+v", len(hits), hits)
	}
	if hits[0].Line != 4 {
		t.Errorf("line = %d, want the last line", hits[0].Line)
	}
}

func TestCodeAndFrontmatterAreNotProse(t *testing.T) {
	docs := []Doc{{Rel: "A.md", Content: strings.Join([]string{
		"---",
		"author: Stoic",
		"---",
		"Inline `Stoic` stays code.",
		"```",
		"Stoic in a fence",
		"```",
		"Stoic in the open.",
	}, "\n")}}
	hits := Find(docs, []string{"Stoic"}, "Stoic.md")
	if len(hits) != 1 {
		t.Fatalf("hits = %d, want only the prose one:\n%+v", len(hits), hits)
	}
	if hits[0].Context != "Stoic in the open." {
		t.Errorf("context = %q", hits[0].Context)
	}
}

func TestAMentionMatchesWholeWordsInAnyCase(t *testing.T) {
	docs := []Doc{{Rel: "A.md", Content: "stoic thinking, Stoicism, unstoic, STOIC.\n"}}
	hits := Find(docs, []string{"Stoic"}, "Stoic.md")
	var texts []string
	for _, h := range hits {
		texts = append(texts, h.Text)
	}
	// "Stoicism" and "unstoic" have a letter against the match, so they
	// are other words and not this note.
	if len(hits) != 2 || texts[0] != "stoic" || texts[1] != "STOIC" {
		t.Errorf("hits = %v, want the two whole words in any case", texts)
	}
}

func TestAnAliasCountsAndOverlappingNamesAreOneRow(t *testing.T) {
	docs := []Doc{{Rel: "A.md", Content: "Stoicism is the school.\n"}}
	hits := Find(docs, []string{"Stoicism", "Stoic"}, "Stoic.md")
	if len(hits) != 1 {
		t.Fatalf("one mention should be one row, got %+v", hits)
	}
	if hits[0].Text != "Stoicism" {
		t.Errorf("text = %q, want the alias that was actually written", hits[0].Text)
	}
}

func TestApplyWrapsTheWordsThatWereThere(t *testing.T) {
	content := "I read Stoic today, and stoic again.\n"
	hits := Find([]Doc{{Rel: "A.md", Content: content}}, []string{"Stoic"}, "Stoic.md")
	if len(hits) != 2 {
		t.Fatalf("setup: hits = %+v", hits)
	}
	got := Apply(content, hits)
	want := "I read [[Stoic]] today, and [[stoic]] again.\n"
	if got != want {
		t.Errorf("Apply:\n got %q\nwant %q", got, want)
	}
}

func TestApplyCanLinkOneOfSeveral(t *testing.T) {
	content := "Stoic once. Stoic twice.\n"
	hits := Find([]Doc{{Rel: "A.md", Content: content}}, []string{"Stoic"}, "Stoic.md")
	got := Apply(content, hits[1:2])
	if want := "Stoic once. [[Stoic]] twice.\n"; got != want {
		t.Errorf("Apply:\n got %q\nwant %q", got, want)
	}
}

func TestApplyLeavesAStaleHitAlone(t *testing.T) {
	// A hit whose offsets no longer fit the text is dropped rather than
	// cutting the note somewhere arbitrary.
	got := Apply("short", []Hit{{Start: 100, End: 105}})
	if got != "short" {
		t.Errorf("Apply with a stale hit = %q, want the text untouched", got)
	}
}

func TestAnEmptyNameFindsNothing(t *testing.T) {
	docs := []Doc{{Rel: "A.md", Content: "anything at all\n"}}
	if hits := Find(docs, []string{"", "   "}, "B.md"); len(hits) != 0 {
		t.Errorf("an empty name should match nothing, got %+v", hits)
	}
}

func TestHitsComeSortedByNoteThenPosition(t *testing.T) {
	docs := []Doc{
		{Rel: "C.md", Content: "Stoic\n"},
		{Rel: "A.md", Content: "one Stoic two Stoic\n"},
	}
	hits := Find(docs, []string{"Stoic"}, "Stoic.md")
	if len(hits) != 3 {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[0].Rel != "A.md" || hits[1].Rel != "A.md" || hits[2].Rel != "C.md" {
		t.Errorf("order = %q, %q, %q", hits[0].Rel, hits[1].Rel, hits[2].Rel)
	}
	if hits[0].Start >= hits[1].Start {
		t.Error("within a note, hits should come in reading order")
	}
}
