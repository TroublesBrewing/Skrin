// Package daily creates daily notes the way Obsidian does. It uses the
// path and template variables of the core Daily notes plugin, and the todo
// rollover of the Rollover Daily Todos community plugin.
package daily

import (
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/rivo/uniseg"

	"github.com/lurioso/skrin/internal/obsidian"
)

// Path is the vault-relative path of the daily note for day t.
func Path(s obsidian.DailyNotes, t time.Time) string {
	return path.Join(s.Folder, Format(t, s.Format)+".md")
}

// TemplatePath is the vault path of the daily-note template, or "" if none
// is set. Obsidian stores it with or without ".md".
func TemplatePath(s obsidian.DailyNotes) string {
	t := strings.Trim(s.Template, "/")
	if t != "" && !strings.EqualFold(path.Ext(t), ".md") {
		t += ".md"
	}
	return t
}

// Previous finds the most recent daily note before day t, looking back
// about three years.
func Previous(exists func(string) bool, s obsidian.DailyNotes, t time.Time) string {
	for i := 1; i <= 1100; i++ {
		if p := Path(s, t.AddDate(0, 0, -i)); exists(p) {
			return p
		}
	}
	return ""
}

var (
	dateVarRE  = regexp.MustCompile(`(?i){{\s*date\s*}}`)
	timeVarRE  = regexp.MustCompile(`(?i){{\s*time\s*}}`)
	titleVarRE = regexp.MustCompile(`(?i){{\s*title\s*}}`)
	calcVarRE  = regexp.MustCompile(`(?i){{\s*(date|time)\s*(([+-]\d+)([yqmwdhs]))?\s*(:.+?)?}}`)
)

// Expand fills in a daily-note template like Obsidian's Daily notes plugin:
// {{date}} and {{title}} become the note's name, {{time}} the current time,
// and {{date:FORMAT}}, {{time:FORMAT}} and offsets like {{date+1d:dddd}}
// are formatted with moment.js syntax.
func Expand(tmpl, format string, day, now time.Time) string {
	title := Format(day, format)
	s := dateVarRE.ReplaceAllLiteralString(tmpl, title)
	s = timeVarRE.ReplaceAllLiteralString(s, Format(now, "HH:mm"))
	s = titleVarRE.ReplaceAllLiteralString(s, title)
	base := time.Date(day.Year(), day.Month(), day.Day(), now.Hour(), now.Minute(), now.Second(), 0, day.Location())
	return calcVarRE.ReplaceAllStringFunc(s, func(match string) string {
		m := calcVarRE.FindStringSubmatch(match)
		t := base
		if m[2] != "" {
			n, _ := strconv.Atoi(m[3])
			t = shift(t, n, m[4])
		}
		f := format
		if m[5] != "" {
			f = strings.TrimSpace(m[5][1:])
		}
		return Format(t, f)
	})
}

// shift adds n units the way moment's add() reads shorthand units.
func shift(t time.Time, n int, unit string) time.Time {
	switch unit {
	case "y", "Y":
		return t.AddDate(n, 0, 0)
	case "q", "Q":
		return t.AddDate(0, 3*n, 0)
	case "M":
		return t.AddDate(0, n, 0)
	case "w", "W":
		return t.AddDate(0, 0, 7*n)
	case "d", "D":
		return t.AddDate(0, 0, n)
	case "h", "H":
		return t.Add(time.Duration(n) * time.Hour)
	case "m":
		return t.Add(time.Duration(n) * time.Minute)
	case "s", "S":
		return t.Add(time.Duration(n) * time.Second)
	}
	return t
}

// Lines splits note text the way the Rollover plugin does.
func Lines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(strings.ReplaceAll(s, "\r", "\n"), "\n")
}

var todoRE = regexp.MustCompile(`\s*[*+-] \[(.+?)\]`)

// UnfinishedTodos returns the lines the Rollover Daily Todos plugin would
// carry over: every checkbox whose mark is a single character that isn't a
// done marker, plus (with children) the more-indented lines under each.
func UnfinishedTodos(lines []string, children bool, done string) []string {
	var todos []string
	for l := 0; l < len(lines); l++ {
		if !isTodo(lines[l], done) {
			continue
		}
		todos = append(todos, lines[l])
		if !children {
			continue
		}
		parent := indent(lines[l])
		for l+1 < len(lines) && indent(lines[l+1]) > parent {
			l++
			todos = append(todos, lines[l])
		}
	}
	return todos
}

func isTodo(line, done string) bool {
	m := todoRE.FindStringSubmatch(line)
	if m == nil || uniseg.GraphemeClusterCount(m[1]) != 1 {
		return false
	}
	if strings.ContainsAny(m[1], "‮​‌‍") {
		return false
	}
	g := uniseg.NewGraphemes(done)
	for g.Next() {
		if g.Str() == m[1] {
			return false
		}
	}
	return true
}

// indent counts leading whitespace; blank lines are -1, as in the plugin.
func indent(line string) int {
	return strings.IndexFunc(line, func(r rune) bool { return !unicode.IsSpace(r) })
}

// DropEmpty removes todos with no text ("- [ ]").
func DropEmpty(todos []string) []string {
	var out []string
	for _, t := range todos {
		if s := strings.TrimSpace(t); s != "- [ ]" && s != "- [  ]" {
			out = append(out, t)
		}
	}
	return out
}

// AddTodos puts todos right after the first occurrence of heading, or at
// the end of the note when heading is "none" or not found.
func AddTodos(note string, todos []string, heading string) string {
	block := "\n" + strings.Join(todos, "\n")
	if heading != "" && heading != "none" && strings.Contains(note, heading) {
		return strings.Replace(note, heading, heading+block, 1)
	}
	return note + block
}

// RemoveLines deletes every line equal to one of lines, which is what the
// plugin's "delete todos from previous day" does.
func RemoveLines(note string, lines []string) string {
	drop := map[string]bool{}
	for _, l := range lines {
		drop[l] = true
	}
	var kept []string
	for _, l := range strings.Split(note, "\n") {
		if !drop[l] {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "\n")
}
