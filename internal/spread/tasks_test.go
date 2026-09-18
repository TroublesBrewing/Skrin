package spread

import (
	"strings"
	"testing"
)

// planner has two projects with tasks, and a note whose inline fields
// back up its frontmatter.
var planner = fakeVault{notes: []Note{
	{Rel: "Projects/Alpha.md", Tags: []string{"project"}, Props: map[string][]string{"owner": {"Ann"}}, Tasks: []Task{
		{Line: 3, Status: " ", Text: "call the printer [due:: 2026-09-22] [priority:: high]", Parent: -1,
			Fields: map[string][]string{"due": {"2026-09-22"}, "priority": {"high"}}},
		{Line: 4, Status: "x", Text: "find the number", Parent: 0},
		{Line: 5, Status: " ", Text: "ask about paper", Parent: 0},
		{Line: 7, Status: "x", Text: "send the draft [due:: 2026-09-10]", Parent: -1,
			Fields: map[string][]string{"due": {"2026-09-10"}}},
	}},
	{Rel: "Projects/Beta.md", Tags: []string{"project"}, Props: map[string][]string{"owner": {"Bo"}}, Tasks: []Task{
		{Line: 2, Status: "-", Text: "cancelled idea", Parent: -1},
		{Line: 3, Status: " ", Text: "book the room [due:: 2026-09-20]", Parent: -1,
			Fields: map[string][]string{"due": {"2026-09-20"}}},
	}},
	{Rel: "Reading.md", Props: map[string][]string{"rating": {"3"}}, Fields: map[string][]string{"rating": {"5"}, "mood": {"calm"}}},
}}

func taskRun(t *testing.T, q string) Result {
	t.Helper()
	r := Run(q, planner, "")
	if r.Err != "" {
		t.Fatalf("%s: %s", q, r.Err)
	}
	return r
}

// shown is the answer as its lines, links reduced to what they show.
func shown(md string) string {
	var out []string
	for _, l := range strings.Split(strings.TrimRight(md, "\n"), "\n") {
		for {
			i := strings.Index(l, "[[")
			if i < 0 {
				break
			}
			j := strings.Index(l[i:], "]]")
			_, alias, _ := strings.Cut(l[i+2:i+j], "|")
			l = l[:i] + alias + l[i+j+2:]
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

func TestTaskListsGroupedByNoteWithSubtasks(t *testing.T) {
	r := taskRun(t, `TASK FROM #project`)
	want := "Alpha\n" +
		"- [ ] call the printer [due:: 2026-09-22] [priority:: high] ↗\n" +
		"    - [x] find the number ↗\n" +
		"    - [ ] ask about paper ↗\n" +
		"- [x] send the draft [due:: 2026-09-10] ↗\n" +
		"\n" +
		"Beta\n" +
		"- [-] cancelled idea ↗\n" +
		"- [ ] book the room [due:: 2026-09-20] ↗"
	if got := shown(r.Markdown); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestTaskLinksOpenAtTheLine(t *testing.T) {
	r := taskRun(t, `TASK FROM "Projects/Beta"`)
	if !strings.Contains(r.Markdown, "- [-] cancelled idea [[Projects/Beta#:3|↗]]\n") {
		t.Errorf("got %q", r.Markdown)
	}
}

func TestTaskWhereSeesTheTaskFirst(t *testing.T) {
	cases := map[string]string{
		`TASK WHERE !completed`:            "Alpha|call the printer|find the number|ask about paper|Beta|cancelled idea|book the room",
		`TASK WHERE !checked`:              "Alpha|call the printer|find the number|ask about paper|Beta|book the room",
		`TASK WHERE status = "-"`:          "Beta|cancelled idea",
		`TASK WHERE due <= "2026-09-20"`:   "Alpha|send the draft|Beta|book the room",
		`TASK WHERE priority = "high"`:     "Alpha|call the printer|find the number|ask about paper",
		`TASK WHERE contains(text, "the")`: "Alpha|call the printer|find the number|ask about paper|send the draft|Beta|book the room",
		`TASK WHERE owner = "Bo"`:          "Beta|cancelled idea|book the room",
		`TASK WHERE line = 5`:              "Alpha|ask about paper",
	}
	for q, want := range cases {
		if got := taskNames(taskRun(t, q)); got != want {
			t.Errorf("%s\n got  %s\n want %s", q, got, want)
		}
	}
}

// taskNames is each note and task in an answer, in order, without
// checkboxes, fields or links.
func taskNames(r Result) string {
	var out []string
	for _, l := range strings.Split(shown(r.Markdown), "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if len(l) > 6 && l[0] == '-' {
			l = l[6:]
		}
		if i := strings.Index(l, " ["); i >= 0 {
			l = l[:i]
		}
		out = append(out, strings.TrimSuffix(l, " ↗"))
	}
	return strings.Join(out, "|")
}

func TestAMatchingSubtaskShowsOnItsOwnWhenItsParentDoesnt(t *testing.T) {
	if got := taskNames(taskRun(t, `TASK WHERE text = "ask about paper"`)); got != "Alpha|ask about paper" {
		t.Errorf("got %s", got)
	}
}

func TestTaskSortDecidesWhichNoteLeads(t *testing.T) {
	got := taskNames(taskRun(t, `TASK WHERE due SORT due`))
	if got != "Alpha|send the draft|call the printer|find the number|ask about paper|Beta|book the room" {
		t.Errorf("got %s", got)
	}
	got = taskNames(taskRun(t, `TASK WHERE due SORT due DESC LIMIT 2`))
	if got != "Alpha|call the printer|find the number|ask about paper|Beta|book the room" {
		t.Errorf("got %s", got)
	}
}

func TestNoTasksMatch(t *testing.T) {
	if r := taskRun(t, `TASK WHERE status = "?"`); r.Markdown != "" || r.Note != "No tasks match" {
		t.Errorf("got %+v", r)
	}
}

func TestInlineFieldsJoinFrontmatter(t *testing.T) {
	r := taskRun(t, `TABLE rating, mood FROM "Reading"`)
	if !strings.Contains(r.Markdown, "| 3, 5 | calm |") {
		t.Errorf("got %q", r.Markdown)
	}
	if got := names(taskRun(t, `LIST WHERE mood = "calm"`).Markdown); strings.Join(got, ",") != "Reading" {
		t.Errorf("WHERE on an inline field: %v", got)
	}
	if got := names(taskRun(t, `LIST WHERE contains(rating, 5)`).Markdown); strings.Join(got, ",") != "Reading" {
		t.Errorf("both values are there: %v", got)
	}
}
