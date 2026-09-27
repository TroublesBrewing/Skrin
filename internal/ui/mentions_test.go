package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func mentionsText(m *Model) string {
	var b strings.Builder
	for _, l := range m.mentionsBox() {
		b.WriteString(ansi.Strip(l))
		b.WriteString("\n")
	}
	return b.String()
}

// openOn opens the mentions panel for one note, with a note in the vault
// that talks about it in prose.
func mentionsFixture(t *testing.T) *Model {
	t.Helper()
	m := newTestModel(t)
	writeFile(t, m.vault.Root, "Reading.md",
		"# Reading\n\nI read Stoic again today.\n\nSee [[Stoic]] for the note itself.\n\nStoic twice, then.\n")
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMLingersOnlyOnUnlinkedMentions(t *testing.T) {
	m := mentionsFixture(t)
	press(m, "g")        // Go to note
	typeText(m, "Stoic") //
	press(m, "enter")    // open Filosofi/Stoic.md
	if m.notePath != "Filosofi/Stoic.md" {
		t.Fatalf("setup: open note = %q", m.notePath)
	}
	press(m, "M")
	if m.mentions == nil {
		t.Fatal("M should open the panel when there are mentions")
	}
	if len(m.mentions.hits) != 2 {
		t.Fatalf("hits = %d, want the two plain ones (the [[Stoic]] line doesn't count):\n%+v",
			len(m.mentions.hits), m.mentions.hits)
	}
	out := mentionsText(m)
	if !strings.Contains(out, "I read Stoic again today.") {
		t.Errorf("the panel should show the sentence:\n%s", out)
	}
	if !strings.Contains(out, "Reading") {
		t.Errorf("the panel should say which note it is in:\n%s", out)
	}
}

func TestMSaysSoWhenThereAreNoMentions(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l") // Welcome.md, read
	press(m, "M")
	if m.mentions != nil {
		t.Error("nothing to show should not open an empty panel")
	}
	if !strings.Contains(m.flash, "No unlinked mentions") {
		t.Errorf("flash = %q: it should say there are none, and point at b", m.flash)
	}
}

func TestMWithNoNoteInHandRefusesInTheUsualWords(t *testing.T) {
	m := newTestModel(t)
	m.files.cur = 0 // the vault root row: a folder, not a note
	press(m, "M")
	if m.mentions != nil {
		t.Fatal("a folder has no mentions")
	}
	if !strings.Contains(m.flash, "Select a note to") {
		t.Errorf("flash = %q: refusals share one shape", m.flash)
	}
}

func TestAKeyLinksOneMentionAndLeavesTheProseAlone(t *testing.T) {
	m := mentionsFixture(t)
	m.open("Filosofi/Stoic.md")
	press(m, "M")
	if m.mentions == nil {
		t.Fatal("setup: the panel should be open")
	}
	press(m, "a")
	src, err := m.vault.Read("Reading.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(src, "I read [[Stoic]] again today.") {
		t.Errorf("the first mention should be linked in place:\n%s", src)
	}
	if strings.Contains(src, "[[Stoic]] twice") {
		t.Errorf("only the mention under the cursor should be linked:\n%s", src)
	}
	// The panel stays, with one mention left to deal with.
	if m.mentions == nil || len(m.mentions.hits) != 1 {
		t.Errorf("the list should now hold the one still unlinked: %+v", m.mentions)
	}
	if !strings.Contains(m.flash, "Linked 1 mention") {
		t.Errorf("flash = %q", m.flash)
	}
}

func TestShiftALinksThemAllInOneUndoableStep(t *testing.T) {
	m := mentionsFixture(t)
	m.open("Filosofi/Stoic.md")
	press(m, "M", "A")
	src, _ := m.vault.Read("Reading.md")
	if strings.Count(src, "[[Stoic]]") != 3 { // the two new ones and the one that was there
		t.Errorf("both mentions should be linked:\n%s", src)
	}
	if m.mentions != nil {
		t.Error("with nothing left unlinked the panel should close itself")
	}
	if !strings.Contains(m.flash, "Linked 2 mentions in 1 note") {
		t.Errorf("flash = %q", m.flash)
	}
	press(m, "U")
	back, _ := m.vault.Read("Reading.md")
	if strings.Count(back, "[[Stoic]]") != 1 {
		t.Errorf("U should take the whole linking back in one step:\n%s", back)
	}
}

func TestANoteChangedOnDiskIsNamedRatherThanOverwritten(t *testing.T) {
	m := mentionsFixture(t)
	m.open("Filosofi/Stoic.md")
	press(m, "M")
	// Someone else writes the note between the scan and the keypress.
	writeFile(t, m.vault.Root, "Reading.md", "# Reading\n\nRewritten elsewhere.\n")
	press(m, "A")
	src, _ := m.vault.Read("Reading.md")
	if src != "# Reading\n\nRewritten elsewhere.\n" {
		t.Errorf("the newer text must survive untouched:\n%s", src)
	}
	if !strings.Contains(m.flash, "changed on disk") {
		t.Errorf("flash = %q: it must say why nothing was linked", m.flash)
	}
}

func TestEnterReadsThePlaceTheMentionSits(t *testing.T) {
	m := mentionsFixture(t)
	m.open("Filosofi/Stoic.md")
	press(m, "M", "enter")
	if m.mentions != nil {
		t.Error("going there should close the panel")
	}
	if m.notePath != "Reading.md" {
		t.Errorf("open note = %q, want the note doing the mentioning", m.notePath)
	}
	if !strings.Contains(m.flash, "is mentioned here") {
		t.Errorf("flash = %q", m.flash)
	}
}

func TestAnAliasCountsAsTheNotesOwnName(t *testing.T) {
	m := newTestModel(t)
	writeFile(t, m.vault.Root, "Stoa.md", "---\naliases: [Stoicism]\n---\n# Stoa\n")
	writeFile(t, m.vault.Root, "Elsewhere.md", "# Elsewhere\n\nStoicism, the school.\n")
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	m.open("Stoa.md")
	press(m, "M")
	if m.mentions == nil || len(m.mentions.hits) != 1 {
		t.Fatalf("an alias should be looked for too: %+v", m.mentions)
	}
	if out := mentionsText(m); !strings.Contains(out, "alias") {
		t.Errorf("the title should say aliases were included:\n%s", out)
	}
	press(m, "a")
	src, _ := m.vault.Read("Elsewhere.md")
	if !strings.Contains(src, "[[Stoicism]]") {
		t.Errorf("the alias should be linked as it was written:\n%s", src)
	}
}

func TestMentionsFillsTheFrameExactly(t *testing.T) {
	m := mentionsFixture(t)
	m.open("Filosofi/Stoic.md")
	press(m, "M")
	checkFrame(t, m, "the Mentions panel")
}

func TestEscClosesMentions(t *testing.T) {
	m := mentionsFixture(t)
	m.open("Filosofi/Stoic.md")
	press(m, "M", "esc")
	if m.mentions != nil {
		t.Error("esc should close the panel")
	}
	if m.flash != "Mentions closed" {
		t.Errorf("flash = %q", m.flash)
	}
}
