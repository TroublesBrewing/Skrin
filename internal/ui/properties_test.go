package ui

import (
	"strings"
	"testing"

	"github.com/lurioso/skrin/internal/version"
)

func propsModel(t *testing.T) *Model {
	t.Helper()
	if !version.Beta {
		t.Skip("a release build leaves every experiment out")
	}
	return newTestModelWith(t, Options{RolloverTodos: true, Beta: true, Properties: true})
}

func TestReadPropertiesTellsScalarsFromLists(t *testing.T) {
	lines := strings.Split(`---
title: A note
tags:
  - one
  - two
empty:
# a comment Skrin didn't write
nested:
  deep: value
---
body`, "\n")
	props := readProperties(lines)
	got := map[string]property{}
	for _, p := range props {
		got[p.key] = p
	}
	if len(props) != 4 {
		t.Fatalf("found %d properties: %+v", len(props), props)
	}
	if p := got["title"]; !p.editable || p.value != "A note" {
		t.Errorf("title: %+v", p)
	}
	if p := got["tags"]; p.editable || p.from != 2 || p.to != 4 {
		t.Errorf("tags should own its bullets and not be editable: %+v", p)
	}
	if p := got["empty"]; !p.editable || p.value != "" {
		t.Errorf("an empty property is still a field: %+v", p)
	}
	if p := got["nested"]; p.editable {
		t.Errorf("a nested block isn't a field: %+v", p)
	}
}

// The heart of it: an edit touches only the lines the property owns, so
// anything Skrin doesn't understand comes out as it went in.
func TestSettingAPropertyLeavesEverythingElseByteForByte(t *testing.T) {
	m := propsModel(t)
	before := `---
title: Old
tags:
  - one
# a comment Skrin didn't write
---
# Body

Text.
`
	if err := m.vault.Write("Welcome.md", before); err != nil {
		t.Fatal(err)
	}
	m.open("Welcome.md")
	if err := m.setProperty("Welcome.md", "title", "New"); err != nil {
		t.Fatal(err)
	}
	after, err := m.vault.Read("Welcome.md")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(before, "title: Old", "title: New", 1)
	if after != want {
		t.Errorf("after:\n%q\nwant:\n%q", after, want)
	}
}

// A list is shown and can be removed, but never rewritten from one field.
func TestAListPropertyIsNotEditedFromAField(t *testing.T) {
	m := propsModel(t)
	before := "---\ntags:\n  - one\n  - two\n---\nbody\n"
	if err := m.vault.Write("Welcome.md", before); err != nil {
		t.Fatal(err)
	}
	m.open("Welcome.md")
	err := m.setProperty("Welcome.md", "tags", "three")
	if err == nil {
		t.Fatal("a list should refuse to be set from a field")
	}
	if !strings.Contains(err.Error(), "list") {
		t.Errorf("error %q should say why", err)
	}
	if got, _ := m.vault.Read("Welcome.md"); got != before {
		t.Errorf("the note changed: %q", got)
	}
}

func TestRemovingAPropertyTakesItsLinesAndNothingElse(t *testing.T) {
	m := propsModel(t)
	before := "---\ntitle: A\ntags:\n  - one\n  - two\nstatus: read\n---\nbody\n"
	if err := m.vault.Write("Welcome.md", before); err != nil {
		t.Fatal(err)
	}
	m.open("Welcome.md")
	m.removeProperty("Welcome.md", "tags")
	got, err := m.vault.Read("Welcome.md")
	if err != nil {
		t.Fatal(err)
	}
	if got != "---\ntitle: A\nstatus: read\n---\nbody\n" {
		t.Errorf("after removing tags:\n%q", got)
	}
}

// The last property takes the empty block with it: an empty frontmatter
// is noise in every reader.
func TestRemovingTheLastPropertyTakesTheBlock(t *testing.T) {
	m := propsModel(t)
	if err := m.vault.Write("Welcome.md", "---\ntitle: A\n---\nbody\n"); err != nil {
		t.Fatal(err)
	}
	m.open("Welcome.md")
	m.removeProperty("Welcome.md", "title")
	got, _ := m.vault.Read("Welcome.md")
	if got != "body\n" {
		t.Errorf("got %q", got)
	}
}

// ...but not when something else is still in there.
func TestAnEmptyBlockStaysWhenAnythingElseIsInIt(t *testing.T) {
	m := propsModel(t)
	if err := m.vault.Write("Welcome.md", "---\ntitle: A\n# a comment\n---\nbody\n"); err != nil {
		t.Fatal(err)
	}
	m.open("Welcome.md")
	m.removeProperty("Welcome.md", "title")
	got, _ := m.vault.Read("Welcome.md")
	if got != "---\n# a comment\n---\nbody\n" {
		t.Errorf("got %q", got)
	}
}

func TestAddingAPropertyToANoteWithoutFrontmatter(t *testing.T) {
	m := propsModel(t)
	if err := m.vault.Write("Welcome.md", "# Body\n"); err != nil {
		t.Fatal(err)
	}
	m.open("Welcome.md")
	if err := m.addProperty("Welcome.md", "status: reading"); err != nil {
		t.Fatal(err)
	}
	got, _ := m.vault.Read("Welcome.md")
	if got != "---\nstatus: reading\n---\n# Body\n" {
		t.Errorf("got %q", got)
	}
}

func TestAddingAPropertyToANoteThatHasFrontmatter(t *testing.T) {
	m := propsModel(t)
	if err := m.vault.Write("Welcome.md", "---\ntitle: A\n---\nbody\n"); err != nil {
		t.Fatal(err)
	}
	m.open("Welcome.md")
	if err := m.addProperty("Welcome.md", "status: reading"); err != nil {
		t.Fatal(err)
	}
	got, _ := m.vault.Read("Welcome.md")
	if got != "---\ntitle: A\nstatus: reading\n---\nbody\n" {
		t.Errorf("got %q", got)
	}
	if err := m.addProperty("Welcome.md", "status: again"); err == nil {
		t.Error("the same property twice should be refused")
	}
	if err := m.addProperty("Welcome.md", "no colon here"); err == nil {
		t.Error("a line that isn't name: value should be refused")
	}
}

// A note changed on disk under the panel is named, never overwritten.
func TestAPropertyWriteRefusesANoteThatChangedOnDisk(t *testing.T) {
	m := propsModel(t)
	if err := m.vault.Write("Welcome.md", "---\ntitle: A\n---\nbody\n"); err != nil {
		t.Fatal(err)
	}
	m.open("Welcome.md")
	props := readProperties(strings.Split("---\ntitle: A\n---\nbody\n", "\n"))
	if len(props) != 1 {
		t.Fatal("setup")
	}
	if err := m.writeProperties("Welcome.md", "something else entirely", "---\ntitle: B\n---\nbody\n", "test"); err == nil {
		t.Error("it should refuse")
	}
	if got, _ := m.vault.Read("Welcome.md"); got != "---\ntitle: A\n---\nbody\n" {
		t.Errorf("the note was written anyway: %q", got)
	}
}

// With the note open for editing, the edit goes into the buffer, so it
// rides the ordinary autosave and snapshot rather than writing behind the
// editor's back.
func TestAPropertyEditGoesIntoTheEditorWhenItIsOpen(t *testing.T) {
	m := propsModel(t)
	if err := m.vault.Write("Welcome.md", "---\ntitle: A\n---\nbody\n"); err != nil {
		t.Fatal(err)
	}
	m.open("Welcome.md")
	press(m, "e")
	if m.editor == nil {
		t.Fatal("the editor should be open")
	}
	if err := m.setProperty("Welcome.md", "title", "B"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.editor.Text(), "title: B") {
		t.Errorf("the editor still holds %q", m.editor.Text())
	}
}

func TestPropertiesAreOffUntilAskedFor(t *testing.T) {
	m := newTestModel(t)
	m.open("Welcome.md")
	m.showProperties()
	if m.chooser != nil {
		t.Fatal("an experiment that is off may not open a panel")
	}
	if !strings.Contains(m.flash, "beta") {
		t.Errorf("flash %q", m.flash)
	}
	for _, it := range m.mainPaletteItems() {
		if strings.Contains(it.label, "Properties") {
			t.Error("the palette shouldn't carry a switched-off experiment")
		}
	}
}

func TestThePropertiesPanelListsThemWithAnAddRow(t *testing.T) {
	m := propsModel(t)
	if err := m.vault.Write("Welcome.md", "---\ntitle: A\nstatus: read\n---\nbody\n"); err != nil {
		t.Fatal(err)
	}
	m.open("Welcome.md")
	m.showProperties()
	if m.chooser == nil {
		t.Fatalf("no panel; flash %q", m.flash)
	}
	if len(m.chooser.items) != 3 {
		t.Fatalf("rows: %+v", m.chooser.items)
	}
	if !strings.HasPrefix(m.chooser.items[0].label, "+ add") {
		t.Errorf("first row %q", m.chooser.items[0].label)
	}
	if m.chooser.remove == nil {
		t.Error("d should remove a property")
	}
}
