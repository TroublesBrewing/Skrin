// Package datevar turns the date variables of Obsidian's Templates
// plugin into the dates they mean, in a note being written: {{date}},
// {{time}}, {{date:FORMAT}} and offsets like {{date+1d:dddd}}.
//
// It is the same bargain internal/duedate makes for "due:: tomorrow", and
// deliberately so: only lines written or changed during this edit are
// touched, so a {{date}} that has sat in an old note since March still
// says what its author wrote. Fenced code and code spans are left alone,
// because a note about templates has to be able to show the variable
// without it evaporating.
package datevar

import (
	"regexp"
	"strings"
	"time"

	"github.com/lurioso/skrin/internal/daily"
)

// Change is one variable that became a date, for the status line.
type Change struct {
	Var  string // as written: "{{date}}", "{{date+1d:dddd}}"
	Date string // what it became
}

// varRE is a date or time variable, in the shapes daily.ExpandTemplate
// already understands. {{title}} is not one of them: it is the note's
// own name, which is not a thing that happens while you type.
var varRE = regexp.MustCompile(`(?i){{\s*(date|time)\s*(([+-]\d+)([yqmwdhs]))?\s*(:.+?)?}}`)

// Resolve rewrites the date variables in text, as of now. base is the
// note as it was when this edit began; lines that are still in it are
// left alone. dateFormat and timeFormat are the Templates plugin's, so a
// variable means the same in Skrin as in Obsidian.
func Resolve(text, base, dateFormat, timeFormat string, now time.Time) (string, []Change) {
	old := map[string]int{}
	for _, l := range strings.Split(base, "\n") {
		old[l]++
	}
	lines := strings.Split(text, "\n")
	var changes []Change
	fence := ""
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if fence != "" {
			if strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]) == "" {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			fence = t[:len(t)-len(strings.TrimLeft(t, t[:1]))]
			continue
		}
		if old[l] > 0 {
			old[l]-- // a line from before this edit: not ours to rewrite
			continue
		}
		lines[i], changes = resolveLine(l, dateFormat, timeFormat, now, changes)
	}
	return strings.Join(lines, "\n"), changes
}

func resolveLine(l, dateFormat, timeFormat string, now time.Time, changes []Change) (string, []Change) {
	var b strings.Builder
	last := 0
	for _, m := range varRE.FindAllStringIndex(l, -1) {
		if strings.Count(l[:m[0]], "`")%2 == 1 {
			continue // inside a code span: it is being shown, not used
		}
		raw := l[m[0]:m[1]]
		// The one expander, the one used by Insert template and by folder
		// templates. The title it takes can be anything here: a date or
		// time variable never contains one.
		date := daily.ExpandTemplate(raw, "", dateFormat, timeFormat, now)
		if date == raw {
			continue // nothing the expander knows how to fill
		}
		b.WriteString(l[last:m[0]])
		b.WriteString(date)
		last = m[1]
		changes = append(changes, Change{Var: raw, Date: date})
	}
	if last == 0 {
		return l, changes
	}
	b.WriteString(l[last:])
	return b.String(), changes
}
