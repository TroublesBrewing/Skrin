package ui

import (
	"strings"
	"testing"

	"github.com/lurioso/skrin/internal/version"
)

func blockRefModel(t *testing.T) *Model {
	t.Helper()
	if !version.Beta {
		t.Skip("a release build leaves every experiment out")
	}
	return newTestModelWith(t, Options{RolloverTodos: true, Beta: true, BlockRef: true})
}

// The id goes at the end of the block, not on the line the cursor happens
// to sit on: the anchor has to point at the whole paragraph.
func TestBlockEndIsTheEndOfTheParagraph(t *testing.T) {
	lines := []string{"first", "second", "third", "", "another block"}
	for _, row := range []int{0, 1, 2} {
		if got := blockEnd(lines, row); got != 2 {
			t.Errorf("from row %d: block ends at %d, want 2", row, got)
		}
	}
	if got := blockEnd(lines, 4); got != 4 {
		t.Errorf("the last block ends at %d, want 4", got)
	}
	if got := blockEnd(lines, 3); got != -1 {
		t.Errorf("a blank line is no block, got %d", got)
	}
}

// A heading ends the block above it and is a block of its own, or a link
// to a paragraph would point past the next section's title.
func TestAHeadingEndsTheBlock(t *testing.T) {
	lines := []string{"intro", "## Morning", "body", "more body"}
	if got := blockEnd(lines, 0); got != 0 {
		t.Errorf("the paragraph before a heading ends at %d, want 0", got)
	}
	if got := blockEnd(lines, 1); got != 1 {
		t.Errorf("a heading is its own block, got %d", got)
	}
	if got := blockEnd(lines, 2); got != 3 {
		t.Errorf("the paragraph after it ends at %d, want 3", got)
	}
}

// Each list item is its own block, so a link can point at one line of a
// list. An item's indented continuation belongs to it.
func TestEachListItemIsItsOwnBlock(t *testing.T) {
	lines := []string{"- one", "  still one", "- two", "- three"}
	if got := blockEnd(lines, 0); got != 1 {
		t.Errorf("the first item ends at %d, want 1", got)
	}
	if got := blockEnd(lines, 2); got != 2 {
		t.Errorf("the second item ends at %d, want 2", got)
	}
}

func TestCopyBlockLinkWritesAnIDAndCopiesTheLink(t *testing.T) {
	m := blockRefModel(t)
	inFilosofi(m)
	press(m, "j", "e") // edit Stoic
	if m.editor == nil {
		t.Fatal("the editor should be open")
	}
	m.editor.MoveTo(5, 0) // inside the note's body, below both headings
	m.copyBlockLink()

	if !strings.HasPrefix(m.copied, "[[") || !strings.Contains(m.copied, "#^") {
		t.Fatalf("copied %q", m.copied)
	}
	id := m.copied[strings.Index(m.copied, "#^")+2 : len(m.copied)-2]
	if len(id) != blockIDLen {
		t.Errorf("id %q", id)
	}
	if !strings.Contains(m.editor.Text(), "^"+id) {
		t.Error("the id was never written into the note")
	}
	if !strings.Contains(m.flash, m.copied) {
		t.Errorf("flash %q should name the link", m.flash)
	}
}

// The same block gives the same link every time, or the links already
// written down would stop meaning anything.
func TestABlockThatAlreadyHasAnIDKeepsIt(t *testing.T) {
	m := blockRefModel(t)
	inFilosofi(m)
	press(m, "j", "e")
	m.editor.MoveTo(5, 0)
	m.copyBlockLink()
	first, text := m.copied, m.editor.Text()

	m.copyBlockLink()
	if m.copied != first {
		t.Errorf("second ask gave %q, first gave %q", m.copied, first)
	}
	if m.editor.Text() != text {
		t.Error("nothing more should have been written")
	}
}

// A heading is already an anchor, and Obsidian puts no block id on one.
func TestAHeadingGetsNoBlockID(t *testing.T) {
	m := blockRefModel(t)
	inFilosofi(m)
	press(m, "j", "e")
	text := m.editor.Text()
	lines := strings.Split(text, "\n")
	var head int
	for i, l := range lines {
		if strings.HasPrefix(l, "#") {
			head = i
			break
		}
	}
	m.editor.MoveTo(head, 0)
	m.copyBlockLink()
	if m.copied != "" {
		t.Errorf("copied %q for a heading", m.copied)
	}
	if m.editor.Text() != text {
		t.Error("a heading should not be written to")
	}
	if !strings.Contains(m.flash, "already an anchor") {
		t.Errorf("flash %q should say what to do instead", m.flash)
	}
}

// Off, it says where it is switched on, writes nothing, and isn't in the
// palette either.
func TestBlockLinksAreOffUntilAskedFor(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "e")
	text := m.editor.Text()
	m.copyBlockLink()
	if m.copied != "" || m.editor.Text() != text {
		t.Error("an experiment that is off may not write or copy")
	}
	if !strings.Contains(m.flash, "beta") {
		t.Errorf("flash %q", m.flash)
	}
	for _, it := range m.editorPaletteItems() {
		if strings.Contains(it.label, "Copy a link to this block") {
			t.Error("a switched-off experiment shouldn't be findable by name")
		}
	}
}

// Outside the editor there is no line to point at, and it says so rather
// than guessing.
func TestABlockLinkNeedsTheEditor(t *testing.T) {
	m := blockRefModel(t)
	m.open("Welcome.md")
	m.copyBlockLink()
	if m.copied != "" {
		t.Errorf("copied %q with no editor open", m.copied)
	}
	if !strings.Contains(m.flash, "editing") {
		t.Errorf("flash %q", m.flash)
	}
}
