package ui

import (
	"strings"
	"testing"
)

// Merging joins one note onto another: the text, the links to it, and the
// note itself, in one step that U takes back.

func TestMergedTextJoinsTheBodiesAndLeavesPropertiesOut(t *testing.T) {
	target := "---\ntags: [a]\n---\n# Target\n\nFirst thought.\n"
	source := "---\ntags: [b]\n---\nA second thought, with no heading.\n"
	got := mergedText(target, source, "Second")
	want := "---\ntags: [a]\n---\n# Target\n\nFirst thought.\n\n## Second\n\nA second thought, with no heading.\n"
	if got != want {
		t.Errorf("mergedText:\n got %q\nwant %q", got, want)
	}
}

func TestASourceWithItsOwnHeadingBringsIt(t *testing.T) {
	got := mergedText("# Target\n", "# Source\n\nIts text.\n", "Source")
	want := "# Target\n\n# Source\n\nIts text.\n"
	if got != want {
		t.Errorf("mergedText:\n got %q\nwant %q", got, want)
	}
}

func TestAnEmptySourceChangesNothing(t *testing.T) {
	target := "# Target\n\nText.\n"
	if got := mergedText(target, "---\ntags: [x]\n---\n\n\n", "Empty"); got != target {
		t.Errorf("an empty source should leave the target alone, got %q", got)
	}
}

func TestMergeJoinsTheTextAndCarriesTheLinksOver(t *testing.T) {
	m := newTestModel(t)
	writeFile(t, m.vault.Root, "Keep.md", "# Keep\n\nThe one that stays.\n")
	writeFile(t, m.vault.Root, "Gone.md", "# Gone\n\nThe one that joins it.\n")
	writeFile(t, m.vault.Root, "Elsewhere.md", "# Elsewhere\n\nSee [[Gone]] and [[Gone|that one]].\n")
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	m.open("Gone.md")
	runCommand(m, "Merge this note")
	if m.chooser == nil {
		t.Fatal("the palette command should ask which note to merge into")
	}
	// Pick Keep from the chooser.
	typeText(m, "Keep")
	press(m, "enter")
	if m.confirm == nil {
		t.Fatal("a note disappearing should be confirmed, as deleting one is")
	}
	if !strings.Contains(m.confirm.question, "trash") || !strings.Contains(m.confirm.question, "2 links") {
		t.Errorf("question = %q: it should name the trash and the links", m.confirm.question)
	}
	press(m, "y")

	keep, err := m.vault.Read("Keep.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(keep, "The one that stays.") || !strings.Contains(keep, "The one that joins it.") {
		t.Errorf("both texts should be in the note that stays:\n%s", keep)
	}
	if m.vault.Exists("Gone.md") {
		t.Error("the note merged away should be gone from the vault")
	}
	other, _ := m.vault.Read("Elsewhere.md")
	if strings.Contains(other, "[[Gone") {
		t.Errorf("links to the merged note should follow it:\n%s", other)
	}
	if !strings.Contains(other, "[[Keep]]") || !strings.Contains(other, "[[Keep|that one]]") {
		t.Errorf("an alias should survive the rewrite:\n%s", other)
	}
	if m.notePath != "Keep.md" {
		t.Errorf("open note = %q, want the note that now holds both", m.notePath)
	}
	if !strings.Contains(m.flash, "Merged") || !strings.Contains(m.flash, "U undoes") {
		t.Errorf("flash = %q", m.flash)
	}
}

func TestUndoPutsAMergeBackTogether(t *testing.T) {
	m := newTestModel(t)
	writeFile(t, m.vault.Root, "Keep.md", "# Keep\n\nThe one that stays.\n")
	writeFile(t, m.vault.Root, "Gone.md", "# Gone\n\nThe one that joins it.\n")
	writeFile(t, m.vault.Root, "Elsewhere.md", "# Elsewhere\n\nSee [[Gone]].\n")
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	m.open("Gone.md")
	runCommand(m, "Merge this note")
	typeText(m, "Keep")
	press(m, "enter", "y")
	press(m, "U")

	if !m.vault.Exists("Gone.md") {
		t.Error("U should bring the merged-away note back")
	}
	keep, _ := m.vault.Read("Keep.md")
	if strings.Contains(keep, "The one that joins it.") {
		t.Errorf("U should take the joined text back out:\n%s", keep)
	}
	other, _ := m.vault.Read("Elsewhere.md")
	if !strings.Contains(other, "[[Gone]]") {
		t.Errorf("U should put the links back as they were:\n%s", other)
	}
}

func TestMergeRefusesWhenThereIsNoNoteInHand(t *testing.T) {
	m := newTestModel(t)
	m.files.cur = 0 // the vault root: a folder
	m.do(actMergeNote)
	if m.chooser != nil {
		t.Fatal("a folder can't be merged")
	}
	if !strings.Contains(m.flash, "Select a note to") {
		t.Errorf("flash = %q: refusals share one shape", m.flash)
	}
}

func TestMergingAnEmptyNoteSaysToDeleteItInstead(t *testing.T) {
	m := newTestModel(t)
	writeFile(t, m.vault.Root, "Blank.md", "---\ntags: [x]\n---\n")
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	m.open("Blank.md")
	runCommand(m, "Merge this note")
	typeText(m, "Welcome")
	press(m, "enter", "y")
	if !strings.Contains(m.flash, "delete it with d") {
		t.Errorf("flash = %q: an empty note has nothing to merge", m.flash)
	}
	if !m.vault.Exists("Blank.md") {
		t.Error("nothing should have been trashed")
	}
}

func TestTheChooserDoesNotOfferTheNoteItself(t *testing.T) {
	m := newTestModel(t)
	m.open("Welcome.md")
	m.do(actMergeNote)
	if m.chooser == nil {
		t.Fatal("the chooser should open")
	}
	for _, it := range m.chooser.items {
		if it.rel == "Welcome.md" {
			t.Error("a note can't be merged into itself")
		}
	}
}
