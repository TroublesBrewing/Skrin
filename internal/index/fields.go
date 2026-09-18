package index

import (
	"regexp"
	"strings"
)

// Task is one checkbox list item: "- [ ] text", "* [x] text", "1. [-] text".
type Task struct {
	Line   int    // 0-based source line
	Status string // the character between the brackets: " ", "x", "-", "/", …
	Text   string // everything after the checkbox, as written
	Parent int    // index of the task it's nested under, or -1
	Fields map[string][]string
}

var (
	listItemRE = regexp.MustCompile(`^([ \t]*)(?:[-*+]|\d+[.)])(?:[ \t]|$)`)
	taskRE     = regexp.MustCompile(`^[ \t]*(?:[-*+]|\d+[.)]) \[(.)\](?:[ \t]+(.*))?$`)
	// A whole-line field, "key:: value" — also inside a list item or a
	// quote, but never on a task's own line (that's the bracketed kind).
	lineFieldRE = regexp.MustCompile("^[ \\t]*(?:(?:[-*+]|\\d+[.)])[ \\t]+|>[ \\t]*)*([^\\s\\[\\]():`][^\\[\\]():`]*?)::[ \\t]*(.*?)[ \\t]*$")
	// A field inside a sentence: "[key:: value]" or "(key:: value)". The
	// value may hold a [[wikilink]].
	inlineFieldRE = regexp.MustCompile("[\\[(]([^\\[\\]():`]+?)::[ \\t]*((?:\\[\\[[^\\]]*\\]\\]|[^\\[\\]()])*?)[ \\t]*[\\])]")
)

// fieldKey is the name a field is queried by, Dataview's way: lower-case,
// markdown emphasis dropped, spaces as dashes — "**Due Date**" is due-date.
func fieldKey(k string) string {
	k = strings.Trim(strings.TrimSpace(k), "*_")
	return strings.ToLower(strings.Join(strings.Fields(k), "-"))
}

// lineFields finds the inline fields on one line of a note's body.
func lineFields(l string) [][2]string {
	l = codeSpanRE.ReplaceAllString(l, "")
	var out [][2]string
	for _, m := range inlineFieldRE.FindAllStringSubmatch(l, -1) {
		if k := fieldKey(m[1]); k != "" {
			out = append(out, [2]string{k, strings.TrimSpace(m[2])})
		}
	}
	if len(out) == 0 && !taskRE.MatchString(l) {
		if m := lineFieldRE.FindStringSubmatch(l); m != nil {
			if k := fieldKey(m[1]); k != "" && m[2] != "" {
				out = append(out, [2]string{k, m[2]})
			}
		}
	}
	return out
}

// tasker follows list nesting through a note, so each task knows the task
// it sits under.
type tasker struct {
	open []openItem // the list items still open, outermost first
}

type openItem struct {
	indent int
	task   int // index into the note's tasks, or -1 for a plain item
}

func indentWidth(s string) int { return len(strings.ReplaceAll(s, "\t", "    ")) }

// line takes one body line. It returns the task found on it, if any.
func (tk *tasker) line(l string, i int, tasks []Task) (Task, bool) {
	m := listItemRE.FindStringSubmatch(l)
	if m == nil {
		// A blank line keeps a list going; an indented line continues the
		// item above it; anything else at the margin ends the list.
		if strings.TrimSpace(l) != "" && indentWidth(l[:len(l)-len(strings.TrimLeft(l, " \t"))]) == 0 {
			tk.open = nil
		}
		return Task{}, false
	}
	ind := indentWidth(m[1])
	for len(tk.open) > 0 && tk.open[len(tk.open)-1].indent >= ind {
		tk.open = tk.open[:len(tk.open)-1]
	}
	parent := -1
	for j := len(tk.open) - 1; j >= 0; j-- {
		if tk.open[j].task >= 0 {
			parent = tk.open[j].task
			break
		}
	}
	t := taskRE.FindStringSubmatch(l)
	if t == nil {
		tk.open = append(tk.open, openItem{ind, -1})
		return Task{}, false
	}
	tk.open = append(tk.open, openItem{ind, len(tasks)})
	return Task{Line: i, Status: t[1], Text: strings.TrimSpace(t[2]), Parent: parent}, true
}
