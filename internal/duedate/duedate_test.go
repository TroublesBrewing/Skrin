package duedate

import (
	"strings"
	"testing"
	"time"
)

// friday is Friday 2026-09-18, mid-afternoon.
var friday = time.Date(2026, 9, 18, 15, 30, 0, 0, time.Local)

func TestDate(t *testing.T) {
	for word, want := range map[string]string{
		"today": "2026-09-18", "tomorrow": "2026-09-19", "Tomorrow": "2026-09-19",
		"saturday": "2026-09-19", "sunday": "2026-09-20", "monday": "2026-09-21",
		"thursday": "2026-09-24",
		"friday":   "2026-09-25", // on a Friday, the next one
		"FRIDAY":   "2026-09-25",
	} {
		if got := Date(word, friday).Format("2006-01-02"); got != want {
			t.Errorf("%s = %s, want %s", word, got, want)
		}
	}
}

func TestResolveEveryForm(t *testing.T) {
	text := "due:: tomorrow\n" +
		"- [ ] call the printer [due:: friday] [priority:: high]\n" +
		"- [ ] book the room (due:: monday)\n" +
		"**Due**:: today\n" +
		"- due::Sunday\n"
	want := "due:: 2026-09-19\n" +
		"- [ ] call the printer [due:: 2026-09-25] [priority:: high]\n" +
		"- [ ] book the room (due:: 2026-09-21)\n" +
		"**Due**:: 2026-09-18\n" +
		"- due::2026-09-20\n"
	got, changes := Resolve(text, "", friday)
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if len(changes) != 5 || changes[1] != (Change{"friday", "2026-09-25"}) {
		t.Errorf("changes = %+v", changes)
	}
}

func TestResolveLeavesTheRestAlone(t *testing.T) {
	text := strings.Join([]string{
		"due:: tomorrow morning",   // not one word
		"overdue:: tomorrow",       // another key
		"pre-due:: tomorrow",       // another key
		"due:: 2026-10-01",         // already a date
		"start:: tomorrow",         // only due
		"due: tomorrow",            // one colon: not a field
		"`due:: tomorrow` in code", // a code span
		"```",
		"due:: tomorrow", // a code block
		"```",
		"I'm due:: tomorrowish",
	}, "\n")
	if got, changes := Resolve(text, "", friday); got != text || len(changes) != 0 {
		t.Errorf("changed what it shouldn't:\n%s\n%+v", got, changes)
	}
}

// Only what was typed this time: an old "tomorrow" means its author's
// tomorrow, not today's.
func TestResolveOnlyTouchesChangedLines(t *testing.T) {
	base := "# Plan\n- [ ] old task [due:: tomorrow]\n"
	text := "# Plan\n- [ ] old task [due:: tomorrow]\n- [ ] new task [due:: tomorrow]\n"
	got, changes := Resolve(text, base, friday)
	want := "# Plan\n- [ ] old task [due:: tomorrow]\n- [ ] new task [due:: 2026-09-19]\n"
	if got != want || len(changes) != 1 {
		t.Errorf("got\n%s\n%+v", got, changes)
	}
	// An edited line counts as new, even if it was there before.
	got, _ = Resolve("- [ ] old task, reworded [due:: tomorrow]\n", base, friday)
	if !strings.Contains(got, "2026-09-19") {
		t.Errorf("an edited line should resolve: %s", got)
	}
	// Two identical lines, one of them new: one resolves.
	got, changes = Resolve("due:: today\ndue:: today\n", "due:: today\n", friday)
	if len(changes) != 1 || strings.Count(got, "2026-09-18") != 1 {
		t.Errorf("got %q, %+v", got, changes)
	}
}
