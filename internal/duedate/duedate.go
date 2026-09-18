// Package duedate turns a word in a due field into the date it means:
// "due:: tomorrow" becomes "due:: 2026-09-19". It runs once, as the note
// is saved, so the note only ever holds real dates — which spreads,
// sorting and Obsidian all read the same way.
package duedate

import (
	"regexp"
	"strings"
	"time"
)

// Change is one word turned into a date.
type Change struct {
	Word string // as written: "tomorrow", "Friday"
	Date string // 2026-09-19
}

// A due field in any of Dataview's three forms — "due:: x" on its own
// line, "[due:: x]" or "(due:: x)" — whose whole value is one word this
// package knows. "due:: tomorrow morning" is left alone.
var dueRE = regexp.MustCompile(`(?i)(^|[\s\[(*_>])(\**due\**::[ \t]*)(today|tomorrow|monday|tuesday|wednesday|thursday|friday|saturday|sunday)([ \t]*)([\])]|$)`)

var weekdays = map[string]time.Weekday{
	"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday, "wednesday": time.Wednesday,
	"thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday,
}

// Resolve rewrites the due words in text, as of now. Only lines that
// aren't in base — the note as it was before this edit — are touched, so
// an old "tomorrow" written elsewhere keeps meaning what its author meant,
// rather than whatever tomorrow is today. Fenced code and code spans are
// left alone.
func Resolve(text, base string, now time.Time) (string, []Change) {
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
			old[l]--
			continue
		}
		lines[i], changes = resolveLine(l, now, changes)
	}
	return strings.Join(lines, "\n"), changes
}

func resolveLine(l string, now time.Time, changes []Change) (string, []Change) {
	var b strings.Builder
	last := 0
	for _, m := range dueRE.FindAllStringSubmatchIndex(l, -1) {
		wordStart, wordEnd := m[6], m[7]
		if strings.Count(l[:wordStart], "`")%2 == 1 {
			continue // inside a code span
		}
		word := l[wordStart:wordEnd]
		date := Date(word, now).Format("2006-01-02")
		b.WriteString(l[last:wordStart])
		b.WriteString(date)
		last = wordEnd
		changes = append(changes, Change{Word: word, Date: date})
	}
	b.WriteString(l[last:])
	return b.String(), changes
}

// Date is the day a word means, as of now: today, tomorrow, or a weekday's
// next date — always after today, so "friday" on a Friday is next week.
func Date(word string, now time.Time) time.Time {
	y, mo, d := now.Date()
	today := time.Date(y, mo, d, 0, 0, 0, 0, now.Location())
	w := strings.ToLower(word)
	switch w {
	case "today":
		return today
	case "tomorrow":
		return today.AddDate(0, 0, 1)
	}
	days := (int(weekdays[w]) - int(today.Weekday()) + 7) % 7
	if days == 0 {
		days = 7
	}
	return today.AddDate(0, 0, days)
}
