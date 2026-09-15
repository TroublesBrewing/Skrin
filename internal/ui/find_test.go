package ui

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestVaultSearch(t *testing.T) {
	m := newTestModel(t)
	press(m, "/")
	typeText(m, "stoic")
	p := m.search
	var got []string
	for _, r := range p.results {
		got = append(got, r.rel)
	}
	if strings.Join(got, ",") != "Filosofi/Antik/Zeno.md,Filosofi/Stoic.md,Welcome.md" {
		t.Fatalf("results = %q", got)
	}
	checkFrame(t, m, "search")
	press(m, "down", "enter") // Zeno's matching line
	if m.search != nil || m.notePath != "Filosofi/Antik/Zeno.md" {
		t.Fatalf("enter opened %q", m.notePath)
	}
	press(m, "/")
	if m.search == nil || m.search.in.value() != "stoic" {
		t.Error("/ should come back with the last query")
	}
	press(m, "esc")
}

func TestResultsAreGroupedWithGaps(t *testing.T) {
	m := newTestModel(t)
	press(m, "/")
	typeText(m, "stoic")
	p := m.search
	gaps := 0
	for _, r := range p.rows {
		if r.note < 0 {
			gaps++
		}
	}
	if gaps != 2 {
		t.Errorf("%d gaps between 3 notes, want 2", gaps)
	}
	for i := 0; i < len(p.rows); i++ {
		press(m, "down")
		if p.rows[p.cur].note < 0 {
			t.Fatal("the cursor stopped on a gap")
		}
	}
	for i := 0; i < len(p.rows); i++ {
		press(m, "up")
		if p.rows[p.cur].note < 0 {
			t.Fatal("the cursor stopped on a gap going up")
		}
	}
}

func TestSearchTagsPropertiesAndCase(t *testing.T) {
	m := newTestModel(t)
	press(m, "/")
	typeText(m, "#start")
	if r := m.search.results; len(r) != 1 || r[0].rel != "Welcome.md" {
		t.Errorf("#start found %+v", r)
	}
	press(m, "ctrl+u")
	typeText(m, "[tags:stoa]")
	if r := m.search.results; len(r) != 1 || r[0].rel != "Filosofi/Stoic.md" {
		t.Errorf("[tags:stoa] found %+v", r)
	}
	press(m, "ctrl+u")
	typeText(m, "stoic")
	n := len(m.search.results)
	press(m, "alt+c")
	if len(m.search.results) >= n {
		t.Errorf("match case should find fewer notes for stoic: %d vs %d", len(m.search.results), n)
	}
}

func TestSearchInThisNote(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "/", "alt+t")
	typeText(m, "line")
	if r := m.search.results; len(r) != 1 || r[0].rel != "Filosofi/Stoic.md" {
		t.Errorf("searching just this note found %+v", r)
	}
}

func TestQuickSwitcher(t *testing.T) {
	m := newTestModel(t)
	press(m, "ctrl+p")
	typeText(m, "zen")
	press(m, "enter")
	if m.notePath != "Filosofi/Antik/Zeno.md" {
		t.Fatalf("switcher opened %q", m.notePath)
	}
	press(m, "ctrl+p")
	typeText(m, "Brand new idea")
	press(m, "enter")
	if m.confirm == nil {
		t.Fatal("an unknown name should offer to create the note")
	}
	press(m, "y")
	if !m.vault.Exists("Brand new idea.md") || m.editor == nil {
		t.Error("note not created")
	}
}

func TestGIsInstantAndGGGoesToTop(t *testing.T) {
	m := newTestModel(t)
	press(m, "g")
	if m.chooser == nil || m.chooser.title != "Go to note" {
		t.Fatal("g should open Go to note at once")
	}
	press(m, "esc")
	inFilosofi(m)
	last := len(m.entries) - 1
	press(m, "G")
	if m.listCur != last {
		t.Fatalf("G should go to the bottom: %d", m.listCur)
	}
	press(m, "G")
	if m.listCur != 0 {
		t.Errorf("GG should go to the top: %d", m.listCur)
	}
	press(m, "end", "home")
	if m.listCur != 0 {
		t.Errorf("home should go to the top: %d", m.listCur)
	}
	m.lastG = time.Now().Add(-time.Second) // a slow second G is just G
	press(m, "G")
	if m.listCur != last {
		t.Errorf("a slow second G went to %d", m.listCur)
	}
}

func TestReplaceInThisNote(t *testing.T) {
	m := newTestModel(t)
	press(m, "2", "G", "/", "alt+r", "alt+t")
	p := m.search
	if !p.replacing || !p.inNote || p.rel != "Welcome.md" {
		t.Fatalf("alt+r, alt+t should replace in the selected note: %+v", p)
	}
	typeText(m, "Welcome")
	press(m, "tab")
	typeText(m, "Hello")
	checkFrame(t, m, "replace")
	press(m, "ctrl+s")
	if m.confirm == nil || !strings.Contains(m.confirm.question, `1 match in 1 note with "Hello"`) {
		t.Fatalf("confirm = %+v", m.confirm)
	}
	press(m, "y")
	if got := read(m, "Welcome.md"); got != "# Hello\nSee [[Stoic]] and [[Missing]] #start\n" {
		t.Fatalf("replaced: %q", got)
	}
	press(m, "U")
	if got := read(m, "Welcome.md"); got != welcome {
		t.Errorf("U: %q", got)
	}
}

func TestReplaceAcrossVaultWithOptionsAndSkips(t *testing.T) {
	m := newTestModel(t)
	press(m, "/", "alt+r")
	typeText(m, "stoic")
	p := m.search
	if n, notes := p.active(); n != 4 || notes != 3 {
		t.Fatalf("any case: %d matches in %d notes, want 4 in 3", n, notes)
	}
	press(m, "alt+w")
	if n, _ := p.active(); n != 3 {
		t.Fatalf("whole words should drop Stoics: %d", n)
	}
	var inLink int
	for _, n := range p.notes {
		for _, mt := range n.matches {
			if mt.inLink {
				inLink++
			}
		}
	}
	if inLink != 2 {
		t.Errorf("%d matches marked as inside a link, want 2", inLink)
	}
	press(m, "tab")
	typeText(m, "Stoa")
	press(m, "tab", "space") // skip Zeno, the first note
	if n, notes := p.active(); n != 2 || notes != 2 {
		t.Fatalf("after skipping Zeno: %d in %d", n, notes)
	}
	checkFrame(t, m, "replace with skips")
	press(m, "ctrl+s", "y")
	if !strings.HasPrefix(read(m, "Filosofi/Stoic.md"), "---\ntags: [stoa]\n---\n# Stoa\n") ||
		!strings.Contains(read(m, "Welcome.md"), "[[Stoa]]") ||
		!strings.Contains(read(m, "Filosofi/Antik/Zeno.md"), "[[Stoic|the Stoics]]") {
		t.Fatal("replace went wrong")
	}
	press(m, "U")
	if !strings.Contains(read(m, "Welcome.md"), "[[Stoic]]") || !strings.Contains(read(m, "Filosofi/Stoic.md"), "# Stoic\n") {
		t.Error("one U should undo the replace in every note")
	}
}

func TestReplaceSkipsNotesChangedOnDisk(t *testing.T) {
	m := newTestModel(t)
	press(m, "/", "alt+r")
	typeText(m, "Welcome")
	if err := os.WriteFile(m.vault.Abs("Welcome.md"), []byte("# Welcome, changed elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	press(m, "tab")
	typeText(m, "Hi")
	press(m, "ctrl+s", "y")
	if got := read(m, "Welcome.md"); got != "# Welcome, changed elsewhere\n" {
		t.Errorf("a note changed on disk was overwritten: %q", got)
	}
	if !strings.Contains(m.flash, "changed on disk") {
		t.Errorf("flash = %q", m.flash)
	}
}
