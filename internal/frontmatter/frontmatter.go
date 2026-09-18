// Package frontmatter finds a note's YAML frontmatter the way Obsidian
// does: a "---" on the first line, closed by the next "---" or "...".
// The editor's live highlighting keeps its own reading, since it has to
// colour a block that is still being typed and isn't closed yet.
package frontmatter

import "strings"

// End is the index of the line that closes the note's frontmatter, or 0
// when the note has none. Trailing spaces, and a carriage return from a
// Windows line ending, don't count.
func End(lines []string) int {
	if len(lines) < 2 || strings.TrimRight(lines[0], " \r") != "---" {
		return 0
	}
	for i := 1; i < len(lines); i++ {
		if t := strings.TrimRight(lines[i], " \r"); t == "---" || t == "..." {
			return i
		}
	}
	return 0
}
