// Package mentions finds where a note is talked about without being
// linked: its name, or one of its aliases, written in the plain text of
// another note. Obsidian calls these unlinked mentions, and they are the
// other half of backlinks — the half you can't see. A vault gathers them
// by itself, because a thought is usually written before the note it
// belongs to exists, and nobody goes back.
//
// What is skipped is as important as what is found: an occurrence already
// inside a link, inside code, or in a note's frontmatter is not a missed
// link, and offering to "fix" it would be wrong.
package mentions

import (
	"sort"
	"strings"

	"github.com/lurioso/skrin/internal/prose"
	"github.com/lurioso/skrin/internal/search"
)

// Doc is one note to scan.
type Doc struct {
	Rel     string
	Content string
}

// Hit is one place a note is mentioned without a link.
type Hit struct {
	Rel        string // the note the mention is in
	Line       int    // 0-based source line
	Start, End int    // byte offsets of the mention in that note's content
	Text       string // the words that matched, exactly as they are written
	Context    string // the whole line, trimmed
}

// Find looks through docs for mentions of any of names — a note's own name
// and its aliases — leaving out the note itself. The hits come sorted by
// note and then by position, each occurrence once even when two names
// would match the same words.
func Find(docs []Doc, names []string, self string) []Hit {
	var out []Hit
	for _, d := range docs {
		if d.Rel == self {
			continue
		}
		out = append(out, findIn(d, names)...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Rel != out[j].Rel {
			return out[i].Rel < out[j].Rel
		}
		return out[i].Start < out[j].Start
	})
	return out
}

func findIn(d Doc, names []string) []Hit {
	// Prose only, and not the frontmatter either: a property holding the
	// name is not a sentence about the note.
	keep := prose.MaskFrontmatter(prose.Mask(d.Content), d.Content)
	lines := lineStarts(d.Content)
	var hits []Hit
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			continue
		}
		for _, sp := range search.Find(d.Content, name, false, true) {
			if !prose.Free(keep, sp[0], sp[1]) {
				continue
			}
			line := lineOf(lines, sp[0])
			hits = append(hits, Hit{
				Rel:     d.Rel,
				Line:    line,
				Start:   sp[0],
				End:     sp[1],
				Text:    d.Content[sp[0]:sp[1]],
				Context: strings.TrimSpace(lineAt(d.Content, lines, line)),
			})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Start < hits[j].Start })
	// Two names can cover the same words — a note called "Stoic" with the
	// alias "Stoicism" both match inside "Stoicism". One mention is one
	// row, so the first (the longest match at that spot wins by coming
	// first in Find's order per name, then by position) keeps the place.
	var out []Hit
	end := -1
	for _, h := range hits {
		if h.Start < end {
			continue
		}
		out = append(out, h)
		end = h.End
	}
	return out
}

// Apply writes the links in: every hit in one note, turned into a wikilink
// around the words that were already there, so the prose reads exactly as
// it did. Hits must be that note's own, sorted and not overlapping, as
// Find returns them. Obsidian resolves a link whatever its capitals, so
// wrapping "stoic" as [[stoic]] finds Stoic.md and leaves the sentence
// alone.
func Apply(content string, hits []Hit) string {
	var b strings.Builder
	pos := 0
	for _, h := range hits {
		if h.Start < pos || h.End > len(content) {
			continue
		}
		b.WriteString(content[pos:h.Start])
		b.WriteString("[[")
		b.WriteString(content[h.Start:h.End])
		b.WriteString("]]")
		pos = h.End
	}
	b.WriteString(content[pos:])
	return b.String()
}

// lineStarts is the byte offset each line begins at.
func lineStarts(s string) []int {
	out := []int{0}
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, i+1)
		}
	}
	return out
}

// lineOf is the 0-based line an offset sits on.
func lineOf(starts []int, at int) int {
	i := sort.SearchInts(starts, at+1) - 1
	return max(i, 0)
}

// lineAt is one line's text, without its newline.
func lineAt(s string, starts []int, line int) string {
	if line < 0 || line >= len(starts) {
		return ""
	}
	from := starts[line]
	to := len(s)
	if line+1 < len(starts) {
		to = starts[line+1] - 1
	}
	return s[from:min(to, len(s))]
}
