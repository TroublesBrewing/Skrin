package book

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// frontmatter mirrors the YAML block of a book note. Field order here is
// the field order Render writes, matching the note template in the spec.
type frontmatter struct {
	Type         string   `yaml:"type"`
	Title        string   `yaml:"title"`
	Subtitle     string   `yaml:"subtitle"`
	Authors      []string `yaml:"authors"`
	Translators  []string `yaml:"translators,omitempty"`
	OriginalYear string   `yaml:"original_year"`
	EditionYear  string   `yaml:"edition_year"`
	Publisher    string   `yaml:"publisher"`
	Edition      string   `yaml:"edition"`
	ISBN         string   `yaml:"isbn"`
	Format       string   `yaml:"format"`
	Pages        string   `yaml:"pages"`
	Shelf        string   `yaml:"shelf"`
	Status       string   `yaml:"status"`
	Rating       int      `yaml:"rating"`
	Started      string   `yaml:"started"`
	Finished     string   `yaml:"finished"`
	Tags         []string `yaml:"tags"`
	Cover        string   `yaml:"cover"`
}

// NoteName is the note's file name for a title: sanitized, with .md added.
func NoteName(title string) string {
	name := SanitizeFilePart(title)
	if name == "" {
		name = "Untitled book"
	}
	return name + ".md"
}

// SanitizeFilePart strips characters that don't belong in a vault file
// name (Obsidian's own bad-character set, plus a few filesystem extras),
// collapsing whitespace as it goes.
func SanitizeFilePart(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case strings.ContainsRune(`\/:*?"<>|#^[]`, r):
			return -1
		default:
			return r
		}
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

// Render builds the note's Markdown text: YAML frontmatter, the details
// abstract, reflections and quotes.
func Render(b Book) string {
	fm := frontmatter{
		Type:         "book",
		Title:        b.Title,
		Subtitle:     b.Subtitle,
		Authors:      orEmpty(b.Authors),
		Translators:  b.Translators,
		OriginalYear: b.OriginalYear,
		EditionYear:  b.EditionYear,
		Publisher:    b.Publisher,
		Edition:      b.Edition,
		ISBN:         b.ISBN,
		Format:       b.Format,
		Pages:        b.Pages,
		Shelf:        b.Shelf,
		Status:       b.Status,
		Rating:       b.Rating,
		Started:      b.Started,
		Finished:     b.Finished,
		Tags:         orEmpty(dedupTags(b.Tags)),
		Cover:        b.Cover,
	}
	front, err := yaml.Marshal(fm)
	if err != nil {
		front = []byte{}
	}

	var out strings.Builder
	out.WriteString("---\n")
	out.Write(front)
	out.WriteString("---\n\n")
	out.WriteString("# " + display(b.Title, "Untitled") + "\n\n")
	if b.Cover != "" {
		out.WriteString(fmt.Sprintf("![Cover|right|200](%s)\n\n", b.Cover))
	}

	out.WriteString("> [!abstract] **Book details**\n")
	if len(b.Authors) > 0 {
		out.WriteString("> **Author:** " + strings.Join(b.Authors, ", ") + "  \n")
	}
	if len(b.Translators) > 0 {
		out.WriteString("> **Translator:** " + strings.Join(b.Translators, ", ") + "  \n")
	}
	if b.OriginalYear != "" || b.EditionYear != "" {
		out.WriteString(fmt.Sprintf("> **Original Year:** %s · **This Printing:** %s  \n", display(b.OriginalYear, "?"), display(b.EditionYear, "?")))
	}
	if b.Publisher != "" || b.Edition != "" {
		out.WriteString(fmt.Sprintf("> **Publisher:** %s · **Edition:** %s  \n", display(b.Publisher, "?"), display(b.Edition, "?")))
	}
	out.WriteString(fmt.Sprintf("> **Format:** %s · **Pages:** %s · **ISBN:** %s  \n", display(b.Format, "?"), display(b.Pages, "?"), display(b.ISBN, "?")))
	if b.Shelf != "" {
		out.WriteString("> **Shelf:** " + b.Shelf + "  \n")
	}
	status := StatusLabel(b.Status)
	statusLine := "> **Status:** " + display(status, "?")
	if b.Status == "reading" && b.Started != "" {
		statusLine += fmt.Sprintf(" (Started: %s)", b.Started)
	}
	if b.Status == "read" && b.Finished != "" {
		statusLine += fmt.Sprintf(" (Finished: %s)", b.Finished)
	}
	if b.Rating > 0 {
		statusLine += " · **Rating:** " + Stars(b.Rating)
	}
	out.WriteString(statusLine + "\n\n---\n\n")

	out.WriteString("## Reflections & Notes\n\n")
	if strings.TrimSpace(b.Notes) != "" {
		out.WriteString(strings.TrimRight(b.Notes, "\n") + "\n")
	}
	out.WriteString("\n---\n\n## Quotes\n\n")
	for _, q := range b.Quotes {
		if strings.TrimSpace(q.Text) == "" {
			continue
		}
		out.WriteString("> \"" + q.Text + "\"\n")
		attrib := "> — *" + display(q.Speaker, firstOr(b.Authors, "Unknown")) + "*"
		if q.Page != "" {
			attrib += ", p. " + q.Page
		}
		out.WriteString(attrib + "\n\n")
	}
	return strings.TrimRight(out.String(), "\n") + "\n"
}

func display(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func firstOr(list []string, fallback string) string {
	if len(list) > 0 {
		return list[0]
	}
	return fallback
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func dedupTags(tags []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range append(append([]string{}, DefaultTags...), tags...) {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

var frontFence = regexp.MustCompile(`(?s)^---\r?\n(.*?\n)---\r?\n?(.*)$`)

// Parse reads a book note's text back into a Book, for editing or adding
// quotes. It reports ok=false when the note has no "type: book"
// frontmatter (so the caller can fall back to a blank card instead).
func Parse(content string) (b Book, ok bool) {
	m := frontFence.FindStringSubmatch(content)
	if m == nil {
		return Book{}, false
	}
	var fm frontmatter
	if err := yaml.Unmarshal([]byte(m[1]), &fm); err != nil || fm.Type != "book" {
		return Book{}, false
	}
	b = Book{
		Title: fm.Title, Subtitle: fm.Subtitle, Authors: fm.Authors, Translators: fm.Translators,
		OriginalYear: fm.OriginalYear, EditionYear: fm.EditionYear, Publisher: fm.Publisher,
		Edition: fm.Edition, ISBN: fm.ISBN, Format: fm.Format, Pages: fm.Pages, Shelf: fm.Shelf,
		Status: fm.Status, Rating: fm.Rating, Started: fm.Started, Finished: fm.Finished,
		Tags: fm.Tags, Cover: fm.Cover,
	}
	body := m[2]
	b.Notes = parseSection(body, "## Reflections & Notes")
	b.Quotes = parseQuotes(body)
	return b, true
}

// parseSection returns the text between a "## Heading" and the next "---"
// or "## " heading, trimmed.
func parseSection(body, heading string) string {
	i := strings.Index(body, heading)
	if i < 0 {
		return ""
	}
	rest := body[i+len(heading):]
	end := len(rest)
	if j := strings.Index(rest, "\n---"); j >= 0 && j < end {
		end = j
	}
	if j := strings.Index(rest, "\n## "); j >= 0 && j < end {
		end = j
	}
	return strings.TrimSpace(rest[:end])
}

var quoteLine = regexp.MustCompile(`(?m)^>\s*"(.*)"\s*$`)
var attribLine = regexp.MustCompile(`^>\s*—\s*\*(.*?)\*(?:,\s*p\.\s*(\S+))?\s*$`)

// parseQuotes reads the "## Quotes" section back into Quote values.
func parseQuotes(body string) []Quote {
	i := strings.Index(body, "## Quotes")
	if i < 0 {
		return nil
	}
	section := body[i+len("## Quotes"):]
	lines := strings.Split(section, "\n")
	var quotes []Quote
	for idx := 0; idx < len(lines); idx++ {
		qm := quoteLine.FindStringSubmatch(lines[idx])
		if qm == nil {
			continue
		}
		q := Quote{Text: qm[1]}
		if idx+1 < len(lines) {
			if am := attribLine.FindStringSubmatch(strings.TrimSpace(lines[idx+1])); am != nil {
				q.Speaker, q.Page = am[1], am[2]
				idx++
			}
		}
		quotes = append(quotes, q)
	}
	return quotes
}

// ParseYear turns a string into an int for sorting or display, or 0 if it
// isn't one (original years like "180 BCE" are kept as free text and
// simply sort as 0).
func ParseYear(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}
