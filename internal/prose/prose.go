// Package prose says which bytes of a note are prose and which belong to
// something that only looks like it: a fenced or inline code span, a
// wikilink, a markdown link, the frontmatter block.
//
// It exists because two things need the same answer and must not disagree
// about it. Finding where a note is mentioned without a link (internal/
// mentions) and renaming a tag everywhere (internal/tagname) both rewrite
// what someone wrote, and both would be wrong to touch a word inside a
// code sample or inside a link that already works.
package prose

import (
	"strings"

	"github.com/lurioso/skrin/internal/frontmatter"
)

// Mask marks every byte of s that is prose. False means the byte belongs
// to a fenced code block, an inline code span, a wikilink (embeds
// included) or a markdown link. The frontmatter is left as prose: a
// property is prose for a tag and not for a mention, so whoever cares
// calls MaskFrontmatter as well.
func Mask(s string) []bool {
	keep := make([]bool, len(s))
	for i := range keep {
		keep[i] = true
	}
	block := func(from, to int) {
		for i := max(from, 0); i < min(to, len(keep)); i++ {
			keep[i] = false
		}
	}

	// Fenced code, by line, so an unterminated fence protects the rest of
	// the note rather than nothing.
	at, fenced := 0, false
	for _, l := range strings.Split(s, "\n") {
		t := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(t, "```"), strings.HasPrefix(t, "~~~"):
			fenced = !fenced
			block(at, at+len(l)+1)
		case fenced:
			block(at, at+len(l)+1)
		}
		at += len(l) + 1
	}

	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '`':
			if j := strings.IndexByte(s[i+1:], '`'); j >= 0 {
				block(i, i+j+2)
				i += j + 1
			}
		case strings.HasPrefix(s[i:], "[["):
			if j := strings.Index(s[i:], "]]"); j >= 0 {
				block(i, i+j+2)
				i += j + 1
			}
		case s[i] == '[':
			// A markdown link: the text and the target both belong to the
			// link, not to the prose around it.
			if shut := strings.IndexByte(s[i:], ']'); shut >= 0 && strings.HasPrefix(s[i+shut+1:], "(") {
				if end := strings.IndexByte(s[i+shut:], ')'); end >= 0 {
					block(i, i+shut+end+1)
					i += shut + end
				}
			}
		}
	}
	return keep
}

// MaskFrontmatter also blanks the note's property block in a mask Mask
// returned. A property that happens to hold a word is not a sentence
// about it.
func MaskFrontmatter(keep []bool, s string) []bool {
	lines := strings.Split(s, "\n")
	end := frontmatter.End(lines)
	if end == 0 {
		return keep
	}
	at := 0
	for i := 0; i <= end && i < len(lines); i++ {
		at += len(lines[i]) + 1
	}
	for i := 0; i < min(at, len(keep)); i++ {
		keep[i] = false
	}
	return keep
}

// Free reports whether every byte of a span is prose.
func Free(keep []bool, from, to int) bool {
	for i := from; i < to && i < len(keep); i++ {
		if !keep[i] {
			return false
		}
	}
	return true
}
