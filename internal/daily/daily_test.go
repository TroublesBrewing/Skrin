package daily

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lurioso/skrin/internal/obsidian"
)

var tuesday = time.Date(2026, 9, 15, 14, 5, 9, 0, time.Local)

func TestFormat(t *testing.T) {
	for f, want := range map[string]string{
		"YYYY-MM-DD":                  "2026-09-15",
		"dddd, Do MMMM YYYY [week] W": "Tuesday, 15th September 2026 week 38",
		"h:mm A":                      "2:05 PM",
		"YY Q DDDD E ddd MMM":         "26 3 258 2 Tue Sep",
		"[Daily]/YYYY/MM/YYYY-MM-DD":  "Daily/2026/09/2026-09-15",
		"GGGG-[W]WW":                  "2026-W38",
	} {
		if got := Format(tuesday, f); got != want {
			t.Errorf("Format(%q) = %q, want %q", f, got, want)
		}
	}
	if got := Format(time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local), "Do hh a"); got != "1st 12 am" {
		t.Errorf("midnight = %q", got)
	}
}

func TestExpand(t *testing.T) {
	tmpl := "# {{date}} / {{title}} at {{time}}\n{{date:dddd}} {{date+1d:DD}} {{time:HH:mm}} {{DATE : YYYY}}"
	want := "# 2026-09-15 / 2026-09-15 at 14:05\nTuesday 16 14:05 2026"
	if got := Expand(tmpl, "YYYY-MM-DD", tuesday, tuesday); got != want {
		t.Errorf("Expand =\n%q\nwant\n%q", got, want)
	}
}

func TestPathAndPrevious(t *testing.T) {
	s := obsidian.DailyNotes{Folder: "Daily", Format: "YYYY-MM-DD"}
	if p := Path(s, tuesday); p != "Daily/2026-09-15.md" {
		t.Errorf("Path = %q", p)
	}
	have := map[string]bool{"Daily/2026-09-11.md": true, "Daily/2026-09-13.md": true, "Daily/2026-09-15.md": true}
	if p := Previous(func(p string) bool { return have[p] }, s, tuesday); p != "Daily/2026-09-13.md" {
		t.Errorf("Previous = %q", p)
	}
	if TemplatePath(obsidian.DailyNotes{Template: "Templates/Day"}) != "Templates/Day.md" {
		t.Error("template path without .md not completed")
	}
}

func TestUnfinishedTodos(t *testing.T) {
	note := strings.Join([]string{
		"### Todo's",
		"- [ ] call mum",
		"  - about sunday",
		"    - and the cake",
		"- [x] done",
		"    - child of done stays",
		"- [-] cancelled",
		"* [ ] star task",
		"+ [>] forwarded",
		"- [ ] ", // empty, with a trailing space
		"- [ ]",  // empty
		"text - [ ] inline todo",
		"- [ab] not a checkbox",
	}, "\n")
	got := UnfinishedTodos(Lines(note), true, "xX-")
	want := []string{"- [ ] call mum", "  - about sunday", "    - and the cake", "* [ ] star task",
		"+ [>] forwarded", "- [ ] ", "- [ ]", "text - [ ] inline todo"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("todos =\n%q\nwant\n%q", got, want)
	}
	if got := UnfinishedTodos(Lines(note), false, "xX-"); len(got) != 6 {
		t.Errorf("without children got %d lines: %q", len(got), got)
	}
	if got := DropEmpty(want); len(got) != 6 {
		t.Errorf("DropEmpty kept %q", got)
	}
}

func TestAddTodosAndRemoveLines(t *testing.T) {
	note := "# Day\n### Todo's\n\n### Notes\n"
	todos := []string{"- [ ] a", "  - b"}
	if got := AddTodos(note, todos, "### Todo's"); got != "# Day\n### Todo's\n- [ ] a\n  - b\n\n### Notes\n" {
		t.Errorf("under heading: %q", got)
	}
	if got := AddTodos(note, todos, "none"); got != note+"\n- [ ] a\n  - b" {
		t.Errorf("no heading: %q", got)
	}
	if got := AddTodos(note, todos, "## Missing"); got != note+"\n- [ ] a\n  - b" {
		t.Errorf("heading not found: %q", got)
	}
	if got := RemoveLines("x\n- [ ] a\ny\n  - b", todos); got != "x\ny" {
		t.Errorf("RemoveLines = %q", got)
	}
}
