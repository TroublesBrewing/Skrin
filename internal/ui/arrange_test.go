package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// level lists folder dir's entries in the order Files shows them.
func level(m *Model, dir string) string { return strings.Join(m.files.childNames(dir), ",") }

func TestArrangeMode(t *testing.T) {
	m := newTestModel(t)
	press(m, "1", "j", "A") // on Daily, at the top level
	if m.arrange == nil || !strings.Contains(ansi.Strip(m.render()), "ARRANGE") {
		t.Fatal("A should enter arrange mode, and say so")
	}
	checkFrame(t, m, "arrange mode")
	press(m, "J")
	if got := level(m, ""); got != "Filosofi,Daily,Templates,Welcome.md" {
		t.Fatalf("J: %s", got)
	}
	if m.files.selected().Rel != "Daily" {
		t.Error("the cursor should stay on the item it moved")
	}
	press(m, "G", "K", "K") // Welcome up past Templates and Daily: a file above folders
	if got := level(m, ""); got != "Filosofi,Welcome.md,Daily,Templates" {
		t.Errorf("K K: %s", got)
	}
	press(m, "home", "j", "K")
	if !strings.Contains(m.flash, "Already at the top") {
		t.Errorf("K at the top: flash %q", m.flash)
	}
	press(m, "n")
	if m.prompt != nil || !strings.Contains(m.flash, "about order") {
		t.Error("n should be paused in arrange mode")
	}
	ops := m.journal.Len()
	press(m, "esc")
	if m.arrange != nil || m.journal.Len() != ops+1 {
		t.Fatalf("esc should leave arrange mode as one journal step (%d → %d)", ops, m.journal.Len())
	}
	if !strings.Contains(read(m, ".skrin"), `"/": [`) {
		t.Errorf(".skrin = %q", read(m, ".skrin"))
	}
	press(m, "U")
	if got := level(m, ""); got != "Daily,Filosofi,Templates,Welcome.md" || m.vault.Exists(".skrin") {
		t.Errorf("one U should undo the whole session: %s", got)
	}
}

func TestArrangedOrderKeepsThroughChanges(t *testing.T) {
	m := newTestModel(t)
	press(m, "1", "j", "j", "l", "A", "J", "esc") // in Filosofi: Antik/ below Stoic
	if got := level(m, "Filosofi"); got != "Stoic.md,Antik" {
		t.Fatalf("J: %s", got)
	}
	if err := os.WriteFile(m.vault.Abs("Filosofi/Aaa.md"), []byte("# new"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	if got := level(m, "Filosofi"); got != "Stoic.md,Antik,Aaa.md" {
		t.Errorf("a new note should come last in an ordered level: %s", got)
	}
	press(m, "k", "r", "ctrl+u") // Stoic
	typeText(m, "Stoa")
	press(m, "enter", "y")
	if got := level(m, "Filosofi"); got != "Stoa.md,Antik,Aaa.md" {
		t.Errorf("a renamed note should keep its place: %s", got)
	}
	press(m, "U")
	if got := level(m, "Filosofi"); got != "Stoic.md,Antik,Aaa.md" {
		t.Errorf("U should undo the rename, place and all: %s", got)
	}
	press(m, "A", "R", "esc")
	if got := level(m, "Filosofi"); got != "Antik,Aaa.md,Stoic.md" {
		t.Errorf("R should put the level back in the default order: %s", got)
	}
	if err := os.WriteFile(m.vault.Abs(".skrin"), []byte(`{"Filosofi/": ["Stoic.md", "Aaa.md", "Antik"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	if got := level(m, "Filosofi"); got != "Stoic.md,Aaa.md,Antik" {
		t.Errorf("an order written by hand should show: %s", got)
	}
	if err := os.WriteFile(m.vault.Abs(".skrin"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	if got := level(m, "Filosofi"); got != "Antik,Aaa.md,Stoic.md" {
		t.Errorf("a broken .skrin should mean the default order: %s", got)
	}
}

func TestArrangeModeEndsWhenFilesLosesFocus(t *testing.T) {
	m := newTestModel(t)
	ops := m.journal.Len()
	press(m, "1", "j", "A", "J", "2")
	if m.arrange != nil || m.journal.Len() != ops+1 {
		t.Errorf("leaving Files should end arrange mode and record it: arranging %v, ops %d → %d", m.arrange != nil, ops, m.journal.Len())
	}
}
