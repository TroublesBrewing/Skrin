package ui

import (
	"os"
	"strings"
	"testing"
)

// Two marked notes can share a name — a/Plan.md and b/Plan.md. Moving them
// into one folder used to move the first and fail the second, leaving the
// batch half done. The clash is caught before anything moves.
func TestMoveRefusesTwoItemsThatWouldBecomeOne(t *testing.T) {
	m := newTestModel(t)
	if err := os.WriteFile(m.vault.Abs("Templates/Stoic.md"), []byte("# another Stoic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	ops := m.journal.Len()
	m.moveTo("Daily", []string{"Filosofi/Stoic.md", "Templates/Stoic.md"})
	if !strings.Contains(m.flash, "would both become") {
		t.Errorf("the refusal should name the clash: %q", m.flash)
	}
	if m.vault.Exists("Daily/Stoic.md") {
		t.Error("nothing should have moved")
	}
	if !m.vault.Exists("Filosofi/Stoic.md") || !m.vault.Exists("Templates/Stoic.md") {
		t.Error("both notes should still be where they were")
	}
	if m.journal.Len() != ops {
		t.Error("a refused move shouldn't leave a step to undo")
	}
}
