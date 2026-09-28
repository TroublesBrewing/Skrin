// Package spread runs spreads: small queries in a ```spread (or
// ```dataview) block, over what the vault's index knows about each note.
// The query language is a subset of Dataview's DQL. A spread only reads;
// its answer is markdown — a pipe table or a bullet list, with a wikilink
// to each note — for the note renderer to draw like any other.
package spread

import (
	"fmt"
	"time"
)

// Note is what a spread can see of one note.
type Note struct {
	Rel    string
	Tags   []string            // lower-case, without '#'
	Props  map[string][]string // frontmatter; lower-case keys, values as written
	Fields map[string][]string // inline "key:: value" fields in the body
	Tasks  []Task
	Mod    time.Time
	Size   int64
}

// Task is one checkbox item in a note.
type Task struct {
	Line   int    // 0-based
	Status string // the character between the brackets
	Text   string // everything after the checkbox, as written
	Parent int    // index of the task it's nested under, or -1
	Fields map[string][]string
}

// Vault is the index as a spread sees it.
type Vault interface {
	Notes() []Note
	// Resolve finds the note a wikilink target leads to, from note from.
	Resolve(target, from string) (string, bool)
	// Backlinks lists the notes that link to rel.
	Backlinks(rel string) []string
	// Outgoing lists the notes rel links to.
	Outgoing(rel string) []string
}

// Result is a spread's answer. Err, when set, replaces the rest.
type Result struct {
	Markdown string // the table or list
	Note     string // one line for after it: "No notes match", "+N more …"
	Err      string
}

// maxRows is where a spread without LIMIT stops, so a query over a big
// vault can't bury the note it sits in.
const maxRows = 200

// Error is a mistake in a spread, or Dataview syntax it doesn't support
// yet. Line counts from 1 at the first line inside the block.
type Error struct {
	Line int
	Msg  string
}

func (e *Error) Error() string { return fmt.Sprintf("Spread: %s (line %d)", e.Msg, e.Line) }

// Run parses and evaluates src, the text inside a spread block, for the
// note at from.
func Run(src string, v Vault, from string) Result {
	q, err := Parse(src)
	if err != nil {
		return Result{Err: err.Error()}
	}
	if q.kind == kTask {
		return q.runTasks(v, from)
	}
	rows, err := q.eval(v, from)
	if err != nil {
		return Result{Err: err.Error()}
	}
	if len(rows) == 0 {
		return Result{Note: "No notes match" + q.whyNothing(v)}
	}
	var note string
	if q.Limit < 0 && len(rows) > maxRows {
		note = fmt.Sprintf("+%d more · add LIMIT or narrow FROM", len(rows)-maxRows)
		rows = rows[:maxRows]
	}
	return Result{Markdown: q.markdown(rows), Note: note}
}

// whyNothing looks for the reason an answer is empty that a person can't
// see: a property name no note in the vault has, which is nearly always a
// spelling. "No notes match" is true either way, and useless on its own —
// it reads the same whether the query is right and the vault is empty of
// matches, or the query asks for "statuss".
//
// It only ever adds what it is sure of: a name is reported only when *no*
// note has it, so a name that exists and simply didn't match is never
// blamed.
func (q *Query) whyNothing(v Vault) string {
	if len(q.refs) == 0 {
		return ""
	}
	seen := map[string]bool{}
	for _, n := range v.Notes() {
		for k := range n.Props {
			seen[k] = true
		}
		for k := range n.Fields {
			seen[k] = true
		}
		for _, t := range n.Tasks {
			for k := range t.Fields {
				seen[k] = true
			}
		}
	}
	var missing []string
	for _, r := range q.refs {
		if !seen[r] {
			missing = append(missing, r)
		}
	}
	switch len(missing) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf(" · no note has a property called %q", missing[0])
	}
	return fmt.Sprintf(" · no note has a property called %q or %q", missing[0], missing[1])
}
