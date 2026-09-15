package ui

import (
	"strings"
	"testing"
)

// onWelcome shows Welcome.md, which links to [[Stoic]] and [[Missing]].
func onWelcome(m *Model) { press(m, "2", "G", "enter") }

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

func TestEnterFollowsTheOnlyLinkInView(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "enter", "enter", "enter") // into Antik/, Zeno.md into the note pane, follow
	if m.notePath != "Filosofi/Stoic.md" {
		t.Errorf("enter followed to %q", m.notePath)
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
	press(m, "j", "}")
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
