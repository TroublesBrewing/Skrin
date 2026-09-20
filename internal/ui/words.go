package ui

import (
	"strings"
	"unicode"

	"github.com/lurioso/skrin/internal/frontmatter"
)

// countWords counts the words of a note the way a writer counts them: runs of
// anything that isn't a space. Frontmatter doesn't count — it's the note's
// filing, not its prose — and neither do the hashes of a heading or the
// bullet of a list item, since they aren't words either.
func countWords(src string) int {
	lines := strings.Split(src, "\n")
	if end := frontmatter.End(lines); end > 0 {
		lines = lines[end+1:]
	}
	n := 0
	for _, l := range lines {
		t := strings.TrimSpace(l)
		t = strings.TrimLeft(t, "#>-*+ ")
		for _, f := range strings.FieldsFunc(t, unicode.IsSpace) {
			if strings.ContainsFunc(f, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) {
				n++
			}
		}
	}
	return n
}

// wordCount is what the status line says about the note you have open: the
// whole note, or the selection when there is one, as Obsidian does.
func (m *Model) wordCount() string {
	if m.selectionText() != "" {
		return "" // the selection's own count says it, in one measure
	}
	switch {
	case m.editor != nil:
		return plural(countWords(m.editor.Text()), "word")
	case m.notePath != "":
		return plural(countWords(m.noteSrc), "word")
	}
	return ""
}
