package habit

import (
	"reflect"
	"strings"
	"testing"
)

const day1 = "### Reflection\n\n### Habits\n- [ ] Meditera 10 min\n- [x] Läsa 30 min\n- [ ] Stretching\n\n### Notes\nSome prose.\n"

func TestParseFindsTheBlock(t *testing.T) {
	b, ok := Parse(day1)
	if !ok {
		t.Fatal("no block found")
	}
	want := []Item{{"Meditera 10 min", false}, {"Läsa 30 min", true}, {"Stretching", false}}
	if !reflect.DeepEqual(b.Items, want) {
		t.Errorf("Items = %v, want %v", b.Items, want)
	}
}

func TestParseEndsAtProseNotAtTags(t *testing.T) {
	// A tag inside the block must not end it (a #tag is not a heading),
	// but prose does.
	note := "### Habits\n- [ ] a\n#journal\n- [ ] b\nSome prose\n- [ ] c\n"
	b, _ := Parse(note)
	if len(b.Items) != 2 {
		t.Errorf("items = %v, want the two around the tag: %v", b.Items, len(b.Items))
	}
}

func TestParseAcceptsObsidensCapitalX(t *testing.T) {
	b, ok := Parse("### Habits\n- [X] done thing\n")
	if !ok || len(b.Items) != 1 || !b.Items[0].Done {
		t.Errorf("capital X not parsed: %v %v", ok, b.Items)
	}
}

func TestParseEmptySectionIsABlock(t *testing.T) {
	// Today's note may have the heading but nothing written under it.
	b, ok := Parse("### Habits\n\n### Notes\n")
	if !ok || len(b.Items) != 0 {
		t.Errorf("empty section: ok=%v items=%v, want ok with none", ok, b.Items)
	}
}

func TestParseAbsent(t *testing.T) {
	if _, ok := Parse("### Todo's\n- [ ] not habits\n"); ok {
		t.Error("a Todo's block is not a Habits block")
	}
}

func TestToggleFlipsOneMarkerInPlace(t *testing.T) {
	out, ok := Toggle(day1, 0)
	if !ok {
		t.Fatal("toggle refused")
	}
	want := "### Reflection\n\n### Habits\n- [x] Meditera 10 min\n- [x] Läsa 30 min\n- [ ] Stretching\n\n### Notes\nSome prose.\n"
	if out != want {
		t.Errorf("toggle:\n%q\nwant\n%q", out, want)
	}
	// And back.
	out, ok = Toggle(out, 0)
	if !ok || out != day1 {
		t.Errorf("toggle back gave %q", out)
	}
}

func TestToggleKeepsSurroundingBytesIntact(t *testing.T) {
	// Indented checkboxes and trailing prose must survive byte for
	// byte outside the marker itself.
	note := "### Habits\n  - [ ] spaced\n\n### Notes\n"
	out, ok := Toggle(note, 0)
	if !ok {
		t.Fatal("toggle refused")
	}
	want := "### Habits\n  - [x] spaced\n\n### Notes\n"
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestToggleRefusals(t *testing.T) {
	if _, ok := Toggle("### Notes\n", 0); ok {
		t.Error("no block, no toggle")
	}
	if _, ok := Toggle(day1, 3); ok {
		t.Error("no such item")
	}
}

func TestInsertBeforeThePageBreak(t *testing.T) {
	tmpl := "### Habits\n- [ ] Meditera 10 min\n- [ ] Stretching\n"
	note := "### Reflection\n\n---\n### Todo's\n"
	out, ok := Insert(note, tmpl)
	if !ok {
		t.Fatal("insert refused")
	}
	want := "### Reflection\n\n### Habits\n- [ ] Meditera 10 min\n- [ ] Stretching\n\n---\n### Todo's\n"
	if out != want {
		t.Errorf("got:\n%q\nwant:\n%q", out, want)
	}
}

func TestInsertAtTheEndWithoutABreak(t *testing.T) {
	out, ok := Insert("### Notes\na line\n", "### Habits\n- [ ] walk\n")
	if !ok {
		t.Fatal("insert refused")
	}
	want := "### Notes\na line\n### Habits\n- [ ] walk\n"
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestInsertRefusals(t *testing.T) {
	if _, ok := Insert(day1, "### Habits\n- [ ] x\n"); ok {
		t.Error("a note with a block must not get a second")
	}
	if _, ok := Insert("### Notes\n", "### Todo's\n- [ ] x\n"); ok {
		t.Error("no starter list in the template, no insert")
	}
}

func TestStreakCountsAndStops(t *testing.T) {
	g := Grid{
		Names: []string{"walk"},
		Days: []Day{
			{"Daily/2026-09-11.md", "09-11", []Item{{"walk", false}}},
			{"Daily/2026-09-12.md", "09-12", []Item{{"walk", true}}},
			{"Daily/2026-09-13.md", "09-13", []Item{{"walk", true}}},
			{"", "09-14", nil}, // unrecorded day
			{"Daily/2026-09-15.md", "09-15", []Item{{"walk", true}}},
		},
		Done: map[string]map[string]bool{
			"Daily/2026-09-11.md": {"walk": false},
			"Daily/2026-09-12.md": {"walk": true},
			"Daily/2026-09-13.md": {"walk": true},
			"Daily/2026-09-15.md": {"walk": true},
		},
	}
	if n := Streak(g, "walk"); n != 1 {
		t.Errorf("streak = %d, want 1 (stops at the unrecorded day)", n)
	}
	if s := Since(g, "walk"); s != "09-15" {
		t.Errorf("since = %q, want 09-15", s)
	}
	// The same week without the gap and without the unticked 09-11:
	// 09-12..09-15 is a 2-day run (09-14 still has no note, so the walk
	// stops there counting back from 09-15... days [09-12, 09-13, none,
	// 09-15]: from 09-15 the unrecorded 09-14 stops it at 1.
	g.Days = g.Days[1:]
	if n := Streak(g, "walk"); n != 1 {
		t.Errorf("streak over the gap = %d, want 1", n)
	}
	// And with every day ticked, the whole range counts.
	g.Days = []Day{
		{"Daily/2026-09-12.md", "09-12", []Item{{"walk", true}}},
		{"Daily/2026-09-13.md", "09-13", []Item{{"walk", true}}},
		{"Daily/2026-09-15.md", "09-15", []Item{{"walk", true}}},
	}
	if n := Streak(g, "walk"); n != 3 {
		t.Errorf("streak = %d, want 3", n)
	}
	if s := Since(g, "walk"); s != "09-12" {
		t.Errorf("since = %q, want 09-12", s)
	}
}

func TestStreakEmptyDayNoteStopsTheCount(t *testing.T) {
	// A day whose note exists but has no block stops the count too.
	g := Grid{
		Names: []string{"walk"},
		Days:  []Day{{"Daily/2026-09-14.md", "09-14", nil}, {"Daily/2026-09-15.md", "09-15", []Item{{"walk", true}}}},
		Done:  map[string]map[string]bool{"Daily/2026-09-15.md": {"walk": true}},
	}
	if n := Streak(g, "walk"); n != 1 {
		t.Errorf("streak = %d, want 1", n)
	}
}

func TestBlockRangeFindsTheSameSpanParseUses(t *testing.T) {
	note := "### Reflection\n\n### Habits\n- [ ] Meditera 10 min\n- [x] Läsa 30 min\n- [ ] Stretching\n\n### Notes\nSome prose.\n"
	lines := strings.Split(note, "\n")
	start, end, ok := BlockRange(lines)
	if !ok {
		t.Fatal("no range found")
	}
	if lines[start] != Heading {
		t.Errorf("start = %d (%q), want the heading line", start, lines[start])
	}
	// The span must cover exactly the heading and the three checkbox
	// lines, and nothing of "### Notes" or its prose — the same
	// boundary Parse itself finds.
	got := strings.Join(lines[start:end], "\n")
	want := "### Habits\n- [ ] Meditera 10 min\n- [x] Läsa 30 min\n- [ ] Stretching\n"
	if got != want {
		t.Errorf("span =\n%q\nwant\n%q", got, want)
	}
}

func TestBlockRangeAbsent(t *testing.T) {
	if _, _, ok := BlockRange(strings.Split("### Todo's\n- [ ] x\n", "\n")); ok {
		t.Error("a Todo's-only note has no Habits range")
	}
}

func TestBlockRangeToTheEndOfTheNote(t *testing.T) {
	lines := strings.Split("### Habits\n- [ ] walk\n", "\n")
	start, end, ok := BlockRange(lines)
	if !ok || start != 0 || end != len(lines) {
		t.Errorf("start=%d end=%d ok=%v, want the whole note", start, end, ok)
	}
}
