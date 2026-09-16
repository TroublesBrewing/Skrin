package book

import (
	"strings"
	"testing"
)

func TestRenderAndParseRoundTrip(t *testing.T) {
	b := Book{
		Title:        "Meditations",
		Authors:      []string{"Marcus Aurelius"},
		Translators:  []string{"Gregory Hays"},
		OriginalYear: "180",
		EditionYear:  "2003",
		Publisher:    "Modern Library",
		Edition:      "Paperback",
		ISBN:         "0812968255",
		Format:       "paperback",
		Pages:        "256",
		Shelf:        "Office / Philosophy",
		Status:       "reading",
		Rating:       4,
		Started:      "2026-09-16",
		Tags:         []string{"philosophy"},
		Cover:        "Assets/Covers/Meditations-2003.jpg",
		Notes:        "Hays translation is remarkably modern and direct.",
		Quotes: []Quote{
			{Text: "The soul becomes dyed with the color of its thoughts.", Page: "42", Speaker: "Marcus Aurelius"},
			{Text: "Waste no time arguing what a good man should be. Be one.", Page: "", Speaker: ""},
		},
	}
	md := Render(b)
	if !strings.HasPrefix(md, "---\n") {
		t.Fatalf("Render should start with frontmatter fence, got:\n%s", md)
	}
	if !strings.Contains(md, `type: book`) {
		t.Error("missing type: book")
	}
	if !strings.Contains(md, "![Cover|right|200](Assets/Covers/Meditations-2003.jpg)") {
		t.Error("missing cover embed")
	}

	got, ok := Parse(md)
	if !ok {
		t.Fatal("Parse reported ok=false on a book note")
	}
	if got.Title != b.Title || got.Publisher != b.Publisher || got.ISBN != b.ISBN {
		t.Errorf("basic fields didn't round-trip: %+v", got)
	}
	if len(got.Authors) != 1 || got.Authors[0] != "Marcus Aurelius" {
		t.Errorf("Authors = %v", got.Authors)
	}
	if len(got.Translators) != 1 || got.Translators[0] != "Gregory Hays" {
		t.Errorf("Translators = %v", got.Translators)
	}
	if got.Rating != 4 {
		t.Errorf("Rating = %d, want 4", got.Rating)
	}
	if got.Notes != b.Notes {
		t.Errorf("Notes = %q, want %q", got.Notes, b.Notes)
	}
	if len(got.Quotes) != 2 {
		t.Fatalf("Quotes = %v, want 2", got.Quotes)
	}
	if got.Quotes[0].Text != b.Quotes[0].Text || got.Quotes[0].Page != "42" || got.Quotes[0].Speaker != "Marcus Aurelius" {
		t.Errorf("Quotes[0] = %+v", got.Quotes[0])
	}
	// The second quote had no speaker or page: the default speaker is the
	// first author, and re-parsing must not invent a page number.
	if got.Quotes[1].Speaker != "Marcus Aurelius" {
		t.Errorf("Quotes[1].Speaker = %q, want default author", got.Quotes[1].Speaker)
	}
	if got.Quotes[1].Page != "" {
		t.Errorf("Quotes[1].Page = %q, want empty", got.Quotes[1].Page)
	}
}

func TestParseRejectsNonBookNotes(t *testing.T) {
	if _, ok := Parse("# Just a note\n\nNo frontmatter here."); ok {
		t.Error("Parse should reject a note with no frontmatter")
	}
	if _, ok := Parse("---\ntype: daily\n---\n\nbody"); ok {
		t.Error("Parse should reject frontmatter of another type")
	}
}

func TestNoteNameSanitizes(t *testing.T) {
	if got := NoteName(`Weird: Title/With*Bad?Chars`); got != "Weird TitleWithBadChars.md" {
		t.Errorf("NoteName = %q", got)
	}
	if got := NoteName(""); got != "Untitled book.md" {
		t.Errorf("NoteName(\"\") = %q", got)
	}
}

func TestStatusLabelAndStars(t *testing.T) {
	if StatusLabel("reading") == "" || !strings.Contains(StatusLabel("reading"), "Reading") {
		t.Errorf("StatusLabel(reading) = %q", StatusLabel("reading"))
	}
	if StatusLabel("") != "" {
		t.Errorf("StatusLabel(\"\") = %q, want empty", StatusLabel(""))
	}
	if Stars(3) != "★★★☆☆" {
		t.Errorf("Stars(3) = %q", Stars(3))
	}
	if Stars(0) != "" {
		t.Errorf("Stars(0) = %q, want empty", Stars(0))
	}
}
