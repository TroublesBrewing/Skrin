package ui

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// onWelcome opens Welcome.md, the last row in Files, which links to
// [[Stoic]] and [[Missing]], for reading — l, not Enter, since Enter now
// opens straight into editing (v0.24.0).
func onWelcome(m *Model) { press(m, "G", "l") }

func TestFollowLinkWithHintsAndGoBack(t *testing.T) {
	m := newTestModel(t)
	onWelcome(m)
	press(m, "f")
	if m.hints == nil || len(m.hints.hints) != 2 {
		t.Fatalf("hints = %+v", m.hints)
	}
	checkFrame(t, m, "hints")
	press(m, "a")
	if m.notePath != "Filosofi/Stoic.md" || m.focus != paneNote {
		t.Fatalf("followed to %q", m.notePath)
	}
	press(m, "backspace")
	if m.notePath != "Welcome.md" {
		t.Fatalf("back went to %q", m.notePath)
	}
	press(m, "alt+right")
	if m.notePath != "Filosofi/Stoic.md" {
		t.Errorf("forward went to %q", m.notePath)
	}
	press(m, "alt+left")
	if m.notePath != "Welcome.md" {
		t.Errorf("alt+left went to %q", m.notePath)
	}
}

func TestAltLeftRightToggleFoldersWhenFilesFocused(t *testing.T) {
	m := newTestModel(t)
	press(m, "1", "j", "j") // Files focus, cursor on Filosofi (a collapsed folder)
	if m.files.expanded["Filosofi"] {
		t.Fatal("Filosofi starts collapsed in the fixture vault")
	}
	// Held-Alt skimming (alt+up/down) can't release Alt to expand a folder
	// with plain h/l — alt+left/right must do it instead of history
	// back/forward while Files has focus.
	press(m, "alt+right")
	if !m.files.expanded["Filosofi"] {
		t.Error("alt+right did not open the folder under the cursor in Files")
	}
	press(m, "alt+left")
	if m.files.expanded["Filosofi"] {
		t.Error("alt+left did not close the folder under the cursor in Files")
	}
}

// The frozen opening model: Enter enters, to write. Following a link is
// f's job, and only f's — building the habit of Enter must never take you
// away from the note you meant to write in.
func TestEnterEditsTheNoteItIsOn(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "l", "j", "l") // open Antik/, onto Zeno, read it
	if m.notePath != "Filosofi/Antik/Zeno.md" {
		t.Fatalf("reading %q", m.notePath)
	}
	press(m, "enter") // Zeno has one link in view: [[Stoic|the Stoics]]
	if m.editor == nil || m.edit.rel != "Filosofi/Antik/Zeno.md" {
		t.Fatalf("enter should edit the note, not follow its link: editor %v, note %q", m.editor != nil, m.notePath)
	}
}

func TestFFollowsTheOnlyLinkInView(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "l", "j", "l", "f") // Zeno, then follow its one link straight there
	if m.notePath != "Filosofi/Stoic.md" {
		t.Errorf("f followed to %q", m.notePath)
	}
}

func TestLInTheNoteSaysWhereToGo(t *testing.T) {
	m := newTestModel(t)
	onWelcome(m)
	press(m, "l") // over to the note
	press(m, "l") // already there
	if !strings.Contains(m.flash, "enter edits") || !strings.Contains(m.flash, "f follows") {
		t.Errorf("no key in the note is silent: %q", m.flash)
	}
}

func TestMissingLinkOffersToCreate(t *testing.T) {
	m := newTestModel(t)
	onWelcome(m)
	press(m, "f", "s")
	if m.confirm == nil || !strings.Contains(m.confirm.question, "Missing.md") {
		t.Fatalf("confirm = %+v", m.confirm)
	}
	press(m, "y")
	if !m.vault.Exists("Missing.md") || m.editor == nil {
		t.Fatal("note not created and opened")
	}
	press(m, "esc", "backspace")
	if m.notePath != "Welcome.md" {
		t.Errorf("back from the new note went to %q", m.notePath)
	}
}

func TestBacklinks(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "b")
	c := m.chooser
	if c == nil || len(c.items) != 2 {
		t.Fatalf("backlinks chooser = %+v", c)
	}
	if c.items[0].label != "Zeno" || !strings.Contains(c.items[0].detail, "Teacher of") {
		t.Errorf("first backlink = %+v", c.items[0])
	}
	checkFrame(t, m, "backlinks")
	press(m, "enter")
	if m.notePath != "Filosofi/Antik/Zeno.md" {
		t.Errorf("opened %q", m.notePath)
	}
}

func TestOutlineAndHeadingJumps(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "l", "}")
	if m.lines[m.noteOff].Heading != 1 {
		t.Fatalf("} landed on %+v", m.lines[m.noteOff])
	}
	press(m, "}")
	if m.lines[m.noteOff].Heading != 2 {
		t.Errorf("second } landed on %+v", m.lines[m.noteOff])
	}
	press(m, "{")
	if m.lines[m.noteOff].Heading != 1 {
		t.Errorf("{ landed on %+v", m.lines[m.noteOff])
	}
	press(m, "home", "o")
	if m.chooser == nil || len(m.chooser.items) != 2 {
		t.Fatalf("outline = %+v", m.chooser)
	}
	typeText(m, "morn")
	press(m, "enter")
	if l := m.lines[m.noteOff]; l.Heading != 2 {
		t.Errorf("outline jump landed on %+v", l)
	}
}

func TestRenameUpdatesLinksAndUndoesTogether(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "r", "ctrl+u")
	typeText(m, "Stoa")
	press(m, "enter")
	if m.confirm == nil || m.confirm.pill != " LINKS " || !strings.Contains(m.confirm.question, "2 links in 2 notes") {
		t.Fatalf("confirm = %+v", m.confirm)
	}
	press(m, "y")
	if got := read(m, "Welcome.md"); !strings.Contains(got, "[[Stoa]]") || !strings.Contains(got, "[[Missing]]") {
		t.Errorf("Welcome = %q", got)
	}
	if got := read(m, "Filosofi/Antik/Zeno.md"); !strings.Contains(got, "[[Stoa|the Stoics]]") {
		t.Errorf("Zeno = %q", got)
	}
	if !strings.Contains(m.flash, "2 links updated") {
		t.Errorf("flash = %q", m.flash)
	}
	press(m, "U")
	if !m.vault.Exists("Filosofi/Stoic.md") || !strings.Contains(read(m, "Welcome.md"), "[[Stoic]]") ||
		!strings.Contains(read(m, "Filosofi/Antik/Zeno.md"), "[[Stoic|the Stoics]]") {
		t.Error("one U should undo the rename and the link edits")
	}
}

func TestRenameCanLeaveLinksAlone(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "r", "ctrl+u")
	typeText(m, "Stoa")
	press(m, "enter", "n")
	if !m.vault.Exists("Filosofi/Stoa.md") || !strings.Contains(read(m, "Welcome.md"), "[[Stoic]]") {
		t.Error("n should rename but leave the links")
	}
}

func TestMoveKeepsPlainLinksThatStillWork(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "m")
	typeText(m, "daily")
	press(m, "enter")
	if m.confirm != nil {
		t.Fatalf("[[Stoic]] still works after a move, so nothing to ask: %+v", m.confirm)
	}
	if !m.vault.Exists("Daily/Stoic.md") || !strings.Contains(read(m, "Welcome.md"), "[[Stoic]]") {
		t.Error("move went wrong")
	}
}

func TestLinksMoveTheFilesCursor(t *testing.T) {
	m := newTestModel(t)
	onWelcome(m)
	press(m, "f", "a") // to Stoic
	if m.notePath != "Filosofi/Stoic.md" || m.files.selected().Rel != "Filosofi/Stoic.md" || !m.files.expanded["Filosofi"] {
		t.Fatalf("open %q, cursor on %q: the cursor should follow to the note", m.notePath, m.files.selected().Rel)
	}
	press(m, "backspace")
	if m.notePath != "Welcome.md" || m.files.selected().Rel != "Welcome.md" {
		t.Errorf("back: open %q, cursor on %q", m.notePath, m.files.selected().Rel)
	}
}

func TestFWithNoNoteSaysSo(t *testing.T) {
	m := newTestModel(t)
	press(m, "f")
	if m.hints != nil || !strings.Contains(m.flash, "Select a note to follow its links") {
		t.Errorf("f with no note open: hints %v, flash %q", m.hints != nil, m.flash)
	}
}

func TestLinkCompletionInEditor(t *testing.T) {
	m := newTestModel(t)
	openWelcome(t, m)
	press(m, "end")
	typeText(m, " [[Sto")
	if m.complete == nil || m.complete.items[0].label != "Stoic" {
		t.Fatalf("completion = %+v", m.complete)
	}
	checkFrame(t, m, "completion popup")
	press(m, "enter")
	if m.complete != nil || m.editor == nil {
		t.Fatal("enter should accept the suggestion and keep editing")
	}
	press(m, "esc")
	if got := read(m, "Welcome.md"); !strings.HasPrefix(got, "# Welcome [[Stoic]]\n") {
		t.Errorf("saved %q", got)
	}
}

func TestHeadingCompletion(t *testing.T) {
	m := newTestModel(t)
	openWelcome(t, m)
	press(m, "end")
	typeText(m, " [[Stoic#mor")
	if m.complete == nil || m.complete.items[0].insert != "Stoic#Morning" {
		t.Fatalf("heading completion = %+v", m.complete)
	}
	press(m, "esc")
	if m.complete != nil || m.editor == nil {
		t.Error("esc should close the popup, not the editor")
	}
}

func TestAltHintOpensTheLinkInASplit(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	onWelcome(m)
	press(m, "f", "alt+a") // to Stoic
	if m.split == nil || m.split.path != "Filosofi/Stoic.md" {
		t.Fatalf("split = %+v", m.split)
	}
	if m.notePath != "Welcome.md" || m.focus != paneNote {
		t.Errorf("the note being read should stay open and focused: %q, focus %v", m.notePath, m.focus)
	}
	if !strings.Contains(m.flash, "Opened beside") {
		t.Errorf("flash = %q", m.flash)
	}
}

func TestAltHintNeedsRoomToSplit(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 79, Height: 24})
	onWelcome(m)
	press(m, "f", "alt+a")
	if m.split != nil || !strings.Contains(m.flash, "No room to split") {
		t.Errorf("split = %+v, flash = %q", m.split, m.flash)
	}
}

func TestAltHintToAHeadingScrollsTheSplit(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	// Filosofi/Stoic.md has "# Stoic\n## Morning\n" then 80 filler lines: a
	// heading far enough down that landing on it, not the top, is obvious.
	if err := os.WriteFile(m.vault.Abs("Welcome.md"), []byte("# Welcome\nSee [[Filosofi/Stoic#Morning]].\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	onWelcome(m)
	press(m, "alt+f") // one link in view: alt+f takes it straight into a split
	if m.split == nil || m.split.path != "Filosofi/Stoic.md" {
		t.Fatalf("split = %+v", m.split)
	}
	if m.split.off == 0 {
		t.Errorf("the split should scroll to #Morning, not sit at the top")
	}
}

// The charter's flow rule: no silent anything. Files keys pressed while
// the note has focus used to do nothing at all; now they say where the
// thing they act on lives.
func TestNoFilesKeyIsSilentInTheNote(t *testing.T) {
	for _, k := range []string{"space", "ctrl+a", "H"} {
		m := newTestModel(t)
		press(m, "G", "l") // reading Welcome.md
		press(m, k)
		if !strings.Contains(m.flash, "Files") || !strings.Contains(m.flash, "goes back there") {
			t.Errorf("%q in the note: %q", k, m.flash)
		}
	}
}
