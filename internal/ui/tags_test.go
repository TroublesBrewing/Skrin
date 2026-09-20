package ui

import (
	"os"
	"strings"
	"testing"
)

func TestTheTagListShowsEverySpellingWithItsCount(t *testing.T) {
	m := newTestModel(t)
	if err := os.WriteFile(m.vault.Abs("Filosofi/Antik/Zeno.md"),
		[]byte("# Zeno\n#Stoa and #stoa and #start\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.reload()
	press(m, "#")
	if m.chooser == nil {
		t.Fatalf("# should open the tag list; flash %q", m.flash)
	}
	var labels, details []string
	for _, it := range m.chooser.items {
		labels = append(labels, it.label)
		details = append(details, it.detail)
	}
	for _, want := range []string{"#Stoa", "#stoa", "#start"} {
		if !contains(labels, want) {
			t.Errorf("%q missing from %v", want, labels)
		}
	}
	if !contains(details, "2 notes") {
		t.Errorf("#start is in two notes; details = %v", details)
	}
}

func TestPickingATagSearchesForIt(t *testing.T) {
	m := newTestModel(t)
	press(m, "#")
	typeText(m, "start")
	press(m, "enter")
	if m.search == nil {
		t.Fatal("picking a tag should search for it")
	}
	if got := m.search.in.value(); got != "#start" {
		t.Errorf("query = %q", got)
	}
	if len(m.search.results) == 0 {
		t.Error("the search should have run, not just opened")
	}
}

func TestNoTagsSaysSo(t *testing.T) {
	m := newTestModel(t)
	for _, rel := range []string{"Welcome.md", "Filosofi/Stoic.md"} {
		if err := os.WriteFile(m.vault.Abs(rel), []byte("# plain\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m.reload()
	press(m, "#")
	if m.chooser != nil || !strings.Contains(m.flash, "No tags in the vault yet") {
		t.Errorf("chooser %v, flash %q", m.chooser != nil, m.flash)
	}
}

func contains(all []string, want string) bool {
	for _, s := range all {
		if s == want {
			return true
		}
	}
	return false
}
