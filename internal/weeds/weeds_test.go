package weeds

import (
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 9, 28, 9, 0, 0, 0, time.Local)

func ago(d time.Duration) time.Time { return now.Add(-d) }

// group finds one kind in an answer, so a test can be about one check
// without caring which order the groups came in.
func group(t *testing.T, groups []Group, k Kind) Group {
	t.Helper()
	for _, g := range groups {
		if g.Kind == k {
			return g
		}
	}
	t.Fatalf("no group of kind %d in %d groups", k, len(groups))
	return Group{}
}

func has(g Group, what string) bool {
	for _, it := range g.Items {
		if it.What == what {
			return true
		}
	}
	return false
}

func TestATidyVaultHasNoGroupsAtAll(t *testing.T) {
	notes := []Note{
		{Rel: "A.md", Body: strings.Repeat("real writing. ", 20), Modified: ago(time.Hour),
			Links: []Link{{Target: "B", Line: 3}}, Linked: 1, Tags: []string{"idea"}},
		{Rel: "B.md", Body: strings.Repeat("also real writing. ", 20), Modified: ago(time.Hour),
			Links: []Link{{Target: "A", Line: 1}}, Linked: 1, Tags: []string{"idea"}},
	}
	if got := Find(notes, Options{Now: now}); len(got) != 0 {
		t.Errorf("a tidy vault should answer with nothing, got %d groups: %+v", len(got), got)
	}
	if n := Count(nil); n != 0 {
		t.Errorf("Count(nil) = %d", n)
	}
}

func TestADeadLinkIsFoundWithItsLineAndItsNote(t *testing.T) {
	notes := []Note{{
		Rel:      "Filosofi/Stoic.md",
		Body:     strings.Repeat("x", 200),
		Modified: ago(time.Hour),
		Linked:   2,
		Links: []Link{
			{Target: "Zeno", Line: 4},
			{Target: "Chrysippus", Line: 9, Dead: true},
			{Target: "cover.png", Line: 11, Dead: true, Embed: true},
		},
	}}
	g := group(t, Find(notes, Options{Now: now}), DeadLink)
	if g.Total != 2 {
		t.Fatalf("total = %d, want the two dead ones: %+v", g.Total, g.Items)
	}
	var dead Item
	for _, it := range g.Items {
		if it.What == "Chrysippus" {
			dead = it
		}
	}
	if dead.Rel != "Filosofi/Stoic.md" || dead.Line != 9 {
		t.Errorf("item = %+v, want the note and the 0-based line", dead)
	}
	if !strings.Contains(dead.Note, "line 10") {
		t.Errorf("the row should count lines from 1 as an editor does: %q", dead.Note)
	}
	// An embed that leads nowhere says so rather than reading as a link.
	for _, it := range g.Items {
		if it.What == "cover.png" && !strings.Contains(it.Note, "embed") {
			t.Errorf("a dead embed should say embed: %q", it.Note)
		}
	}
}

func TestANoteNobodyLinksToAndThatLinksNowhereIsOnItsOwn(t *testing.T) {
	notes := []Note{
		{Rel: "Alone.md", Body: strings.Repeat("x", 200), Modified: ago(time.Hour)},
		{Rel: "Linked.md", Body: strings.Repeat("x", 200), Modified: ago(time.Hour), Linked: 1},
		{Rel: "Links out.md", Body: strings.Repeat("x", 200), Modified: ago(time.Hour),
			Links: []Link{{Target: "Linked", Line: 1}}},
		// A note whose only link is dead is still on its own: the link
		// goes nowhere, so it joins nothing to nothing.
		{Rel: "Dead end.md", Body: strings.Repeat("x", 200), Modified: ago(time.Hour),
			Links: []Link{{Target: "Gone", Line: 1, Dead: true}}},
	}
	g := group(t, Find(notes, Options{Now: now}), Unlinked)
	if !has(g, "Alone") || !has(g, "Dead end") {
		t.Errorf("items = %+v, want Alone and Dead end", g.Items)
	}
	if has(g, "Linked") || has(g, "Links out") {
		t.Errorf("a note that is linked, or links somewhere real, is not on its own: %+v", g.Items)
	}
}

func TestAStubIsATitleAndNextToNothing(t *testing.T) {
	notes := []Note{
		{Rel: "Stub.md", Body: "A thought.", Modified: ago(48 * time.Hour), Linked: 1},
		{Rel: "Empty.md", Body: "\n\n  \n", Modified: ago(48 * time.Hour), Linked: 1},
		{Rel: "Real.md", Body: strings.Repeat("written out at length. ", 10), Modified: ago(time.Hour), Linked: 1},
	}
	g := group(t, Find(notes, Options{Now: now}), Stub)
	if !has(g, "Stub") || !has(g, "Empty") {
		t.Errorf("items = %+v, want both thin notes", g.Items)
	}
	if has(g, "Real") {
		t.Error("a note with writing in it is not a stub")
	}
	for _, it := range g.Items {
		if it.What == "Empty" && !strings.Contains(it.Note, "empty") {
			t.Errorf("an empty note should say so: %q", it.Note)
		}
		if it.What == "Stub" && !strings.Contains(it.Note, "10 characters") {
			t.Errorf("a stub should say how little there is: %q", it.Note)
		}
	}
}

func TestOneNameOnTwoNotesIsReportedOnce(t *testing.T) {
	notes := []Note{
		{Rel: "Filosofi/Stoic.md", Body: strings.Repeat("x", 200), Modified: ago(time.Hour), Linked: 1},
		{Rel: "Books/stoic.md", Body: strings.Repeat("x", 200), Modified: ago(time.Hour), Linked: 1},
		{Rel: "Only once.md", Body: strings.Repeat("x", 200), Modified: ago(time.Hour), Linked: 1},
	}
	g := group(t, Find(notes, Options{Now: now}), SameName)
	if g.Total != 1 {
		t.Fatalf("total = %d, want one row for the clash: %+v", g.Total, g.Items)
	}
	it := g.Items[0]
	// Case doesn't save you: Obsidian resolves [[stoic]] to one of them
	// whatever the capitals.
	if !strings.Contains(it.Note, "2 notes") || !strings.Contains(it.Note, "Books") || !strings.Contains(it.Note, "Filosofi") {
		t.Errorf("the row should name both folders: %q", it.Note)
	}
}

func TestATagUsedOnceSaysWhenItIsProbablyATypo(t *testing.T) {
	notes := []Note{
		{Rel: "A.md", Body: strings.Repeat("x", 200), Modified: ago(time.Hour), Linked: 1, Tags: []string{"filosofi"}},
		{Rel: "B.md", Body: strings.Repeat("x", 200), Modified: ago(time.Hour), Linked: 1, Tags: []string{"filosofi"}},
		{Rel: "C.md", Body: strings.Repeat("x", 200), Modified: ago(time.Hour), Linked: 1, Tags: []string{"Filosofi"}},
		{Rel: "D.md", Body: strings.Repeat("x", 200), Modified: ago(time.Hour), Linked: 1, Tags: []string{"once-only"}},
	}
	g := group(t, Find(notes, Options{Now: now}), LoneTag)
	var capital, lone Item
	for _, it := range g.Items {
		switch it.What {
		case "#Filosofi":
			capital = it
		case "#once-only":
			lone = it
		}
	}
	if !strings.Contains(capital.Note, "also spelled #filosofi") || !strings.Contains(capital.Note, "2 notes") {
		t.Errorf("a one-off spelling of a common tag should point at the common one: %q", capital.Note)
	}
	if capital.Rel != "C.md" {
		t.Errorf("the row should open the note that uses it: %q", capital.Rel)
	}
	if !strings.Contains(lone.Note, "only D uses it") {
		t.Errorf("a genuinely single tag should say which note has it: %q", lone.Note)
	}
	// A tag two notes share isn't a weed at all.
	if has(g, "#filosofi") {
		t.Error("a tag used by two notes should not be listed")
	}
}

func TestUntouchedNotesComeOldestFirstAndSayHowOld(t *testing.T) {
	notes := []Note{
		{Rel: "Recent.md", Body: strings.Repeat("x", 200), Modified: ago(30 * 24 * time.Hour), Linked: 1},
		{Rel: "Old.md", Body: strings.Repeat("x", 200), Modified: ago(300 * 24 * time.Hour), Linked: 1},
		{Rel: "Oldest.md", Body: strings.Repeat("x", 200), Modified: ago(800 * 24 * time.Hour), Linked: 1},
	}
	g := group(t, Find(notes, Options{Now: now}), Untouched)
	if len(g.Items) != 2 {
		t.Fatalf("items = %+v, want the two past half a year", g.Items)
	}
	if g.Items[0].What != "Oldest" {
		t.Errorf("oldest first: got %q", g.Items[0].What)
	}
	if !strings.Contains(g.Items[0].Note, "2 years ago") {
		t.Errorf("the row should say how long it has waited: %q", g.Items[0].Note)
	}
	if !strings.Contains(g.Items[1].Note, "10 months ago") {
		t.Errorf("months for a note under a year: %q", g.Items[1].Note)
	}
}

func TestASkippedFolderIsLeftOutOfTheWholeNoteChecksButNotOfDeadLinks(t *testing.T) {
	notes := []Note{{
		Rel: "Daily/2026-01-01.md", Body: "", Modified: ago(800 * 24 * time.Hour),
		Links: []Link{{Target: "Nowhere", Line: 2, Dead: true}},
	}}
	got := Find(notes, Options{Now: now, Skip: []string{"Daily"}})
	if len(got) != 1 || got[0].Kind != DeadLink {
		t.Fatalf("a skipped folder should still report its dead links and nothing else: %+v", got)
	}
}

func TestAGroupSaysHowManyThereReallyAreWhenItIsCapped(t *testing.T) {
	var notes []Note
	for i := 0; i < 10; i++ {
		notes = append(notes, Note{Rel: string(rune('A'+i)) + ".md", Body: "", Modified: ago(time.Hour), Linked: 1})
	}
	g := group(t, Find(notes, Options{Now: now, Max: 3}), Stub)
	if len(g.Items) != 3 {
		t.Errorf("shown = %d, want the cap", len(g.Items))
	}
	if g.Total != 10 {
		t.Errorf("total = %d, want all ten: a capped group must not understate itself", g.Total)
	}
	if n := Count([]Group{g}); n != 10 {
		t.Errorf("Count = %d, want 10", n)
	}
}

func TestTheAnswerIsTheSameTwiceOverForTheSameVault(t *testing.T) {
	notes := []Note{
		{Rel: "B.md", Body: "", Modified: ago(time.Hour), Tags: []string{"one", "two"}},
		{Rel: "A.md", Body: "", Modified: ago(time.Hour), Tags: []string{"three"}},
		{Rel: "C.md", Body: "", Modified: ago(time.Hour), Links: []Link{{Target: "X", Dead: true}, {Target: "Y", Dead: true}}},
	}
	first := Find(notes, Options{Now: now})
	for i := 0; i < 20; i++ {
		again := Find(notes, Options{Now: now})
		if len(again) != len(first) {
			t.Fatalf("group count changed between runs: %d then %d", len(first), len(again))
		}
		for gi := range first {
			if len(first[gi].Items) != len(again[gi].Items) {
				t.Fatalf("group %d changed size between runs", gi)
			}
			for ii := range first[gi].Items {
				if first[gi].Items[ii] != again[gi].Items[ii] {
					t.Fatalf("row %d of group %d changed between runs:\n%+v\n%+v",
						ii, gi, first[gi].Items[ii], again[gi].Items[ii])
				}
			}
		}
	}
}

func TestTheGroupsComeInTheOrderTheKindsAreDeclared(t *testing.T) {
	notes := []Note{
		{Rel: "A.md", Body: "", Modified: ago(800 * 24 * time.Hour), Tags: []string{"lone"},
			Links: []Link{{Target: "Gone", Dead: true}}},
		{Rel: "sub/A.md", Body: "", Modified: ago(800 * 24 * time.Hour)},
	}
	got := Find(notes, Options{Now: now})
	want := []Kind{DeadLink, Unlinked, Stub, SameName, LoneTag, Untouched}
	if len(got) != len(want) {
		t.Fatalf("groups = %d, want %d: %+v", len(got), len(want), got)
	}
	for i, k := range want {
		if got[i].Kind != k {
			t.Errorf("group %d is kind %d, want %d — broken before untidy", i, got[i].Kind, k)
		}
	}
}
