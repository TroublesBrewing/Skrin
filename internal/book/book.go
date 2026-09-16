// Package book is Skrin's Library & Book Card: metadata lookups (Open
// Library, with a Libris fallback for Swedish ISBNs), cover image
// downloads into the vault, and the note template a Book Card reads and
// writes. Nothing here touches the UI; internal/ui/book_card.go is the
// Bubble Tea front end.
package book

import "strings"

// Quote is one passage caught while reading, with where it came from.
type Quote struct {
	Text    string
	Page    string
	Speaker string
}

// Book is everything a Book Card holds about one book: the bibliographic
// fields, the reading state, and its quotes and reflections. It is the
// single struct both the template writer and its parser work from, so a
// note round-trips through the card without losing anything.
type Book struct {
	Title       string
	Subtitle    string
	Authors     []string
	Translators []string

	OriginalYear string // free text: 180 BCE, "c. 1200", etc.
	EditionYear  string
	Publisher    string
	Edition      string
	ISBN         string
	Format       string
	Pages        string

	Shelf    string
	Status   string // "reading", "read", "want", "paused", "dnf"
	Rating   int    // 0-5 stars; 0 means unrated
	Started  string // YYYY-MM-DD, or ""
	Finished string

	Tags  []string
	Cover string // vault-relative path, e.g. Assets/Covers/Meditations-2003.jpg

	Notes  string
	Quotes []Quote
}

// DefaultTags is what a new book note is tagged with, ahead of anything
// else the user adds.
var DefaultTags = []string{"book"}

// Statuses are the values the Status field cycles through, in the order
// they're offered.
var Statuses = []string{"want", "reading", "read", "paused", "dnf"}

// StatusLabel is how a status reads in the rendered card ("📖 Reading").
func StatusLabel(status string) string {
	labels := map[string]string{
		"want":    "📚 Want to read",
		"reading": "📖 Reading",
		"read":    "✅ Read",
		"paused":  "⏸ Paused",
		"dnf":     "✋ Did not finish",
	}
	if l, ok := labels[strings.ToLower(status)]; ok {
		return l
	}
	if status == "" {
		return ""
	}
	return status
}

// Stars renders a 0-5 rating as ★★★☆☆.
func Stars(rating int) string {
	if rating <= 0 {
		return ""
	}
	if rating > 5 {
		rating = 5
	}
	return strings.Repeat("★", rating) + strings.Repeat("☆", 5-rating)
}
