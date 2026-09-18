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
		return Result{Note: "No notes match"}
	}
	var note string
	if q.Limit < 0 && len(rows) > maxRows {
		note = fmt.Sprintf("+%d more · add LIMIT or narrow FROM", len(rows)-maxRows)
		rows = rows[:maxRows]
	}
	return Result{Markdown: q.markdown(rows), Note: note}
}
