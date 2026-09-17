// Package habit works on the habits block in a daily note: a "### Habits"
// section of checkbox lines. The block lives in the day's note and nowhere
// else — no definitions file, no stored state — so Obsidian renders and
// edits the same lines Skrin does.
package habit

import (
	"strings"
)

// Heading is the section marker the block lives under. It is plain
// markdown; Obsidian and Skrin both render it as a heading.
const Heading = "### Habits"

// Item is one habit of one day, exactly as the note spells it.
type Item struct {
	Text string // without the checkbox, trimmed
	Done bool
}

// Unchecked and Checked render i as its markdown line, the same shape
// Obsidian writes when it toggles a checkbox.
func (i Item) Unchecked() string { return "- [ ] " + i.Text }
func (i Item) Checked() string   { return "- [x] " + i.Text }

// isHeading reports whether a trimmed line is a markdown heading: one or
// more '#' then a space (or a bare '#' line). A "#tag" is not a heading,
// so tags inside the block don't end it.
func isHeading(t string) bool {
	if !strings.HasPrefix(t, "#") {
		return false
	}
	h := strings.TrimLeft(t, "#")
	return h == "" || strings.HasPrefix(h, " ")
}

// isTag reports whether a trimmed line is nothing but markdown tags, the
// way Obsidian renders "#journal" — not a heading (no space after the #).
// Pure tag lines are decoration; they don't end the block.
func isTag(t string) bool {
	if !strings.HasPrefix(t, "#") || isHeading(t) {
		return false
	}
	for _, f := range strings.Fields(t) {
		if !strings.HasPrefix(f, "#") {
			return false
		}
	}
	return true
}

// parseCheckbox reads "- [ ] text" or "- [x] text" (Obsidian also writes
// "- [X]"; anything else is not a checkbox).
func parseCheckbox(t string) (Item, bool) {
	if s, ok := strings.CutPrefix(t, "- ["); ok {
		if len(s) < 2 || s[1] != ']' {
			return Item{}, false
		}
		return Item{Text: strings.TrimSpace(s[2:]), Done: s[0] == 'x' || s[0] == 'X'}, true
	}
	return Item{}, false
}

// Parse finds the "### Habits" section of note and returns its items, or
// ok=false when the note has no such section. The section runs from its
// heading to the next heading, the first line inside that is prose (not a
// checkbox, not blank), or the end of the note — history is what the note
// says, and the block is never guessed at. A heading with no lines under
// it yet parses as an empty block: today's note may simply not be written.
func Parse(note string) (Block, bool) {
	var b Block
	in := false
	for _, l := range strings.Split(note, "\n") {
		t := strings.TrimSpace(l)
		if isHeading(t) {
			if in {
				break // the next heading ends the block
			}
			if t == Heading {
				in = true
			}
			continue
		}
		if !in || t == "" {
			continue // before the block, or a blank line inside it
		}
		if isTag(t) {
			continue // tags decorate the block; they don't end it
		}
		item, ok := parseCheckbox(t)
		if !ok {
			break // prose, not a checkbox: the block ends there
		}
		b.Items = append(b.Items, item)
	}
	return b, in
}

// BlockRange finds the "### Habits" section within lines (as Lines
// already splits a note) and returns its span [start, end): start is the
// heading's own line, end is one past the section's last content line —
// exactly the boundary Parse uses. ok is false when there is no section.
// This is how callers outside this package (the rollover mirror) can
// exclude the block's lines from a scan without duplicating Parse's own
// boundary rule.
func BlockRange(lines []string) (start, end int, ok bool) {
	in := false
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if isHeading(t) {
			if in {
				return start, i, true // the next heading ends the block
			}
			if t == Heading {
				in, start = true, i
			}
			continue
		}
		if !in {
			continue // before the block
		}
		if t == "" || isTag(t) {
			continue // a blank or tag line inside the block
		}
		if _, ok := parseCheckbox(t); !ok {
			return start, i, true // prose ends the block
		}
	}
	if in {
		return start, len(lines), true
	}
	return 0, 0, false
}

// Block is the "### Habits" section of one note, parsed.
type Block struct {
	Items []Item
}

// Lines renders b as the note section: the heading, then one checkbox per
// item. A block with no items is just the heading.
func (b Block) Lines() []string {
	out := []string{Heading}
	for _, it := range b.Items {
		if it.Done {
			out = append(out, it.Checked())
		} else {
			out = append(out, it.Unchecked())
		}
	}
	return out
}

// Toggle flips item idx of note's block and returns the new note text, or
// ok=false when there is no block or no such item. The edit is surgical:
// the checkbox's marker is rewritten in place on its own line — nothing
// else in the note moves, so the block's spacing and everything around it
// survive byte for byte.
func Toggle(note string, idx int) (string, bool) {
	lines := strings.Split(note, "\n")
	in := false
	n := 0
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if isHeading(t) {
			if in {
				break
			}
			if t == Heading {
				in = true
			}
			continue
		}
		if !in || t == "" {
			continue
		}
		item, ok := parseCheckbox(t)
		if !ok {
			break
		}
		if n == idx {
			switch {
			case item.Done:
				if j := strings.Index(l, "- [x]"); j >= 0 {
					lines[i] = l[:j] + "- [ ]" + l[j+5:]
				} else if j := strings.Index(l, "- [X]"); j >= 0 {
					lines[i] = l[:j] + "- [ ]" + l[j+5:]
				}
			default:
				if j := strings.Index(l, "- [ ]"); j >= 0 {
					lines[i] = l[:j] + "- [x]" + l[j+5:]
				}
			}
			return strings.Join(lines, "\n"), true
		}
		n++
	}
	return note, false
}

// Insert puts a fresh Habits block — the template's starter list — into a
// note that has none, before the first "---" page break (the daily
// template's section divider) or at the end when there is no break. The
// block comes from the template's own Habits section, so today's habits
// are exactly what the template says; if the template has none, the insert
// is refused (ok=false) and the caller points at the template instead.
func Insert(note string, template string) (string, bool) {
	if _, ok := Parse(note); ok {
		return note, false // already has a block
	}
	block, ok := Parse(template)
	if !ok || len(block.Items) == 0 {
		return note, false // nothing to start from
	}
	section := strings.Join(block.Lines(), "\n")
	if i := strings.Index(note, "\n---\n"); i >= 0 {
		// Between the section above the break and the break itself,
		// with a blank line of its own on each side.
		return note[:i+1] + section + "\n\n" + note[i+1:], true
	}
	if note == "" {
		return section + "\n", true
	}
	return strings.TrimRight(note, "\n") + "\n" + section + "\n", true
}

// Day is one column of the week/month grid: a day's note and its habits.
type Day struct {
	Path  string // vault-relative, "" when the day has no note
	Name  string // how the day shows in headers and flashes (the caller formats the date)
	Items []Item
}

// Grid is habits × days: one row per habit name (today's names, in the
// note's own order), one column per day, oldest first.
type Grid struct {
	Names []string
	Days  []Day
	Done  map[string]map[string]bool // day path → habit name → ticked
}

// Streak counts the consecutive ticked days for name, walking back from
// the grid's last day. A day with no note, or no block, stops the count
// honestly — unrecorded is not broken, but it is not a tick either.
func Streak(g Grid, name string) int {
	n := 0
	for i := len(g.Days) - 1; i >= 0; i-- {
		day := g.Days[i]
		if day.Path == "" || !g.Done[day.Path][name] {
			break
		}
		n++
	}
	return n
}

// Since is the "since" date of name's streak: the first day of the run
// ending at the grid's last day, "" when there is no run.
func Since(g Grid, name string) string {
	n := Streak(g, name)
	if n == 0 || n > len(g.Days) {
		return ""
	}
	return g.Days[len(g.Days)-n].Name
}
