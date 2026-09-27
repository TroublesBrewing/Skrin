package ui

import (
	"strings"
	"testing"
)

// Renaming a tag everywhere: the palette asks which tag, then what it
// should be called, and writes every note that has it in one step.

func tagFixture(t *testing.T) *Model {
	t.Helper()
	m := newTestModel(t)
	writeFile(t, m.vault.Root, "One.md", "---\ntags: [stoa]\n---\n# One\n\nA thought #stoa and #stoa/praktik.\n")
	writeFile(t, m.vault.Root, "Two.md", "# Two\n\nAnother #Stoa, spelled with a capital.\n")
	writeFile(t, m.vault.Root, "Three.md", "# Three\n\nNo tag here. `#stoa` is code.\n")
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestRenamingATagRewritesTextAndPropertiesEverywhere(t *testing.T) {
	m := tagFixture(t)
	runCommand(m, "Rename a tag")
	if m.chooser == nil {
		t.Fatal("the command should ask which tag")
	}
	typeText(m, "stoa")
	press(m, "enter")
	if m.prompt == nil {
		t.Fatal("it should then ask what to call it")
	}
	if !strings.Contains(m.prompt.label, "#stoa") {
		t.Errorf("label = %q: it should name the tag being renamed", m.prompt.label)
	}
	// The old name is offered to edit, so clear it and type the new one.
	press(m, "ctrl+u")
	typeText(m, "stoicism")
	press(m, "enter")

	one, _ := m.vault.Read("One.md")
	if !strings.Contains(one, "tags: [stoicism]") {
		t.Errorf("the property should be renamed:\n%s", one)
	}
	if !strings.Contains(one, "#stoicism and #stoicism/praktik") {
		t.Errorf("the text and its nested tag should be renamed:\n%s", one)
	}
	two, _ := m.vault.Read("Two.md")
	if !strings.Contains(two, "#stoicism") {
		t.Errorf("every spelling should be gathered into the new name:\n%s", two)
	}
	three, _ := m.vault.Read("Three.md")
	if !strings.Contains(three, "`#stoa`") {
		t.Errorf("a tag in code is not a tag:\n%s", three)
	}
	if !strings.Contains(m.flash, "Renamed #stoa to #stoicism") || !strings.Contains(m.flash, "U undoes") {
		t.Errorf("flash = %q", m.flash)
	}
}

func TestUndoPutsEveryRenamedTagBack(t *testing.T) {
	m := tagFixture(t)
	runCommand(m, "Rename a tag")
	typeText(m, "stoa")
	press(m, "enter", "ctrl+u")
	typeText(m, "x")
	press(m, "enter")
	press(m, "U")
	one, _ := m.vault.Read("One.md")
	two, _ := m.vault.Read("Two.md")
	if !strings.Contains(one, "#stoa") || !strings.Contains(one, "tags: [stoa]") {
		t.Errorf("U should put the first note back:\n%s", one)
	}
	if !strings.Contains(two, "#Stoa") {
		t.Errorf("U should put the capital back as it was:\n%s", two)
	}
}

func TestANameThatWontDoIsRefusedInThePromptWithoutWriting(t *testing.T) {
	m := tagFixture(t)
	runCommand(m, "Rename a tag")
	typeText(m, "stoa")
	press(m, "enter", "ctrl+u")
	typeText(m, "two words")
	press(m, "enter")
	if m.prompt == nil {
		t.Fatal("a bad name should leave the prompt open to fix")
	}
	if !strings.Contains(m.prompt.err, "spaces") {
		t.Errorf("err = %q: it should say what's wrong", m.prompt.err)
	}
	one, _ := m.vault.Read("One.md")
	if !strings.Contains(one, "#stoa") {
		t.Error("nothing should be written until the name is usable")
	}
}

func TestRenamingATagToItsOwnNameSaysSo(t *testing.T) {
	m := tagFixture(t)
	runCommand(m, "Rename a tag")
	typeText(m, "stoa")
	press(m, "enter", "enter") // the offered name, unchanged
	if m.prompt == nil || !strings.Contains(m.prompt.err, "already its name") {
		t.Errorf("prompt = %+v: renaming to the same name should say so", m.prompt)
	}
}

func TestWithNoTagsThePaletteRowSaysThereAreNone(t *testing.T) {
	m := newTestModelWith(t, Options{})
	// The fixture vault's only tag is #start in Welcome.md; take it out.
	writeFile(t, m.vault.Root, "Welcome.md", "# Welcome\nSee [[Stoic]] and [[Missing]]\n")
	writeFile(t, m.vault.Root, "Filosofi/Stoic.md", "# Stoic\n")
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	m.do(actRenameTag)
	if m.chooser != nil {
		t.Fatal("there is nothing to choose from")
	}
	if !strings.Contains(m.flash, "No tags in the vault") {
		t.Errorf("flash = %q", m.flash)
	}
}
