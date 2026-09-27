// Package tagname renames a tag everywhere it is written: the #tags in a
// note's prose and the values of its tags property. Obsidian has this and
// Skrin didn't, which left a misspelled tag — the thing Weeds keeps
// finding — with no way to mend it but search and replace, one note at a
// time.
//
// The rewrite is deliberately narrow. It touches a tag and the tags nested
// under it, never a word that merely starts the same way, and never
// anything inside code or a link.
package tagname

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/lurioso/skrin/internal/frontmatter"
	"github.com/lurioso/skrin/internal/prose"
)

// Clean is a tag as this package handles it: no leading '#', no
// surrounding space, no trailing slash.
func Clean(tag string) string {
	return strings.Trim(strings.TrimPrefix(strings.TrimSpace(tag), "#"), "/")
}

// Valid checks a tag name the way Obsidian does: letters, digits,
// underscore, hyphen and '/' for nesting, at least one character, and not
// all digits (Obsidian reads #2026 as a number, not a tag).
func Valid(tag string) error {
	name := Clean(tag)
	switch {
	case name == "":
		return fmt.Errorf("a tag needs a name")
	case strings.ContainsAny(name, " \t"):
		return fmt.Errorf("a tag can't contain spaces — use - or _")
	case strings.Contains(name, "//"):
		return fmt.Errorf("a tag can't have an empty level")
	}
	digits := true
	for _, r := range name {
		switch {
		case unicode.IsLetter(r):
			digits = false
		case unicode.IsDigit(r):
		case r == '_' || r == '-' || r == '/':
			digits = false
		default:
			return fmt.Errorf("a tag can't contain %q", r)
		}
	}
	if digits {
		return fmt.Errorf("a tag needs a letter in it: #%s reads as a number", name)
	}
	return nil
}

// Rename rewrites every occurrence of tag old as new in one note's text and
// reports how many it changed. Matching ignores case, so renaming #Filosofi
// to #filosofi gathers every spelling into one — which is the usual reason
// for doing this at all. A tag nested under the old one comes along:
// #old/child becomes #new/child.
func Rename(content, old, new string) (string, int) {
	from, to := Clean(old), Clean(new)
	if from == "" || to == "" || strings.EqualFold(from, to) && from == to {
		return content, 0
	}
	body, n := renameInProse(content, from, to)
	body, m := renameInProperties(body, from, to)
	return body, n + m
}

// renameInProse rewrites the #tags written in the note's text. It leaves
// code and links alone, and a tag is only a tag when what follows it can't
// be part of a name — so #old doesn't match inside #oldest.
func renameInProse(content, from, to string) (string, int) {
	keep := prose.Mask(content)
	var b strings.Builder
	n, i := 0, 0
	for i < len(content) {
		if content[i] != '#' || !prose.Free(keep, i, i+1) {
			b.WriteByte(content[i])
			i++
			continue
		}
		name, end := tagAt(content, i)
		rest, ok := under(name, from)
		if !ok {
			b.WriteByte(content[i])
			i++
			continue
		}
		b.WriteString("#" + to + rest)
		n++
		i = end
	}
	return b.String(), n
}

// tagAt reads the tag name of the '#' at i, and where the tag ends.
func tagAt(s string, i int) (name string, end int) { return readName(s, i+1) }

// readName reads a tag name starting at at — letters, digits, underscore,
// hyphen, '/' and anything non-ASCII, which is how Obsidian takes "#idé" —
// and says where it ends.
func readName(s string, at int) (name string, end int) {
	j := at
	for j < len(s) {
		c := s[j]
		if c >= 0x80 || unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c)) || c == '_' || c == '-' || c == '/' {
			j++
			continue
		}
		break
	}
	return s[at:j], j
}

// under reports whether name is the tag from, or a tag nested under it,
// and returns what hangs below: "/child" of "parent/child". Case is
// ignored, so every spelling of a tag is gathered by one rename.
func under(name, from string) (rest string, ok bool) {
	switch {
	case strings.EqualFold(name, from):
		return "", true
	case len(name) > len(from) && strings.EqualFold(name[:len(from)], from) && name[len(from)] == '/':
		return name[len(from):], true
	}
	return "", false
}

// renameInProperties rewrites the tag in a note's frontmatter: the values of
// a tags or tag property, in any of the shapes Obsidian writes them —
// "tags: [a, b]", a block list of "- a" lines, or one value on the key's
// own line. Only those lines are touched; another property that happens to
// hold the same word is left alone.
func renameInProperties(content, from, to string) (string, int) {
	lines := strings.Split(content, "\n")
	end := frontmatter.End(lines)
	if end == 0 {
		return content, 0
	}
	n, inTags := 0, false
	for i := 1; i < end; i++ {
		l := lines[i]
		key, _, isKey := cutKey(l)
		switch {
		case isKey && (key == "tags" || key == "tag"):
			inTags = true
		case isKey:
			inTags = false
			continue
		case !inTags:
			continue
		}
		changed, k := renameValues(l, from, to)
		lines[i], n = changed, n+k
	}
	return strings.Join(lines, "\n"), n
}

// cutKey reads "key: value" at the start of a frontmatter line. An indented
// line is a value of the key above it, never a key of its own.
func cutKey(l string) (key, value string, ok bool) {
	if l == "" || l[0] == ' ' || l[0] == '\t' || l[0] == '-' {
		return "", "", false
	}
	k, v, found := strings.Cut(l, ":")
	if !found {
		return "", "", false
	}
	return strings.ToLower(strings.TrimSpace(k)), v, true
}

// renameValues rewrites whole tag values on one frontmatter line, with or
// without a leading '#', nested tags included.
func renameValues(l, from, to string) (string, int) {
	var b strings.Builder
	n, i := 0, 0
	for i < len(l) {
		if !tagStart(l, i) {
			b.WriteByte(l[i])
			i++
			continue
		}
		hash := l[i] == '#'
		at := i
		if hash {
			at++
		}
		name, end := readName(l, at)
		rest, ok := under(name, from)
		if !ok {
			b.WriteByte(l[i])
			i++
			continue
		}
		if hash {
			b.WriteByte('#')
		}
		b.WriteString(to + rest)
		n++
		i = end
	}
	return b.String(), n
}

// tagStart reports whether a tag value could begin at i: the start of the
// line or after a space, quote, comma or bracket, so "ux" inside "unix"
// isn't a value of its own.
func tagStart(l string, i int) bool {
	if i > 0 {
		switch l[i-1] {
		case ' ', '\t', ',', '[', '"', '\'', ':', '-':
		default:
			return false
		}
	}
	c := l[i]
	return c == '#' || unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c)) || c >= 0x80
}
