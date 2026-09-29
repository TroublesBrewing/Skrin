package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/version"
)

func statusModel(t *testing.T) *Model {
	t.Helper()
	if !version.Beta {
		t.Skip("a release build leaves every experiment out")
	}
	return newTestModelWith(t, Options{RolloverTodos: true, Beta: true, StatusSaysWhere: true})
}

// Off, the line reads what it has always read. An experiment may not
// change something that was there before.
func TestStatusLineSaysViewUntilAskedFor(t *testing.T) {
	m := newTestModel(t)
	if got := m.paneWord(); got != " VIEW " {
		t.Errorf("pane word %q with the experiment off", got)
	}
	press(m, "l")
	if got := m.paneWord(); got != " VIEW " {
		t.Errorf("pane word %q in the note pane with the experiment off", got)
	}
}

// On, it names the pane the next key lands in: d, r and n mean different
// things in Files and in the note.
func TestStatusLineNamesThePane(t *testing.T) {
	m := statusModel(t)
	if m.focus != paneFiles {
		t.Fatalf("focus %v at the start", m.focus)
	}
	if got := m.paneWord(); got != " FILES " {
		t.Errorf("pane word %q in Files", got)
	}
	inFilosofi(m)
	press(m, "j", "l") // read the note: the focus moves to the note pane
	if m.focus != paneNote {
		t.Fatalf("focus %v after opening a note", m.focus)
	}
	if got := m.paneWord(); got != " NOTE " {
		t.Errorf("pane word %q in the note pane", got)
	}
	if line := ansi.Strip(m.statusLine()); !strings.Contains(line, "NOTE") {
		t.Errorf("status line %q", line)
	}
}

// A key whose only answer is the status line makes the line move, since a
// word swapped for another word is what the report says goes unseen.
func TestStatusLineBlinksWhenAKeyAnswersThere(t *testing.T) {
	m := statusModel(t)
	if m.statusBlinking() {
		t.Fatal("nothing has answered yet")
	}
	press(m, "s") // an unbound key: its whole answer is the status line
	if m.flash == "" {
		t.Fatalf("the key should have answered in the line")
	}
	if !m.statusBlinking() {
		t.Error("the line should blink while the answer is fresh")
	}
	// The blink ends by itself: the timer only brings the redraw.
	m.statusBlinkAt = m.statusBlinkAt.Add(-2 * statusBlinkFor)
	if m.statusBlinking() {
		t.Error("the blink should be over by now")
	}
	if line := ansi.Strip(m.statusLine()); !strings.Contains(line, "does nothing here") {
		t.Errorf("the answer should still stand after the blink: %q", line)
	}
}

// Off, the line never moves, whatever answers in it.
func TestStatusLineDoesNotBlinkWhenTheExperimentIsOff(t *testing.T) {
	m := newTestModel(t)
	press(m, "s")
	if m.flash == "" {
		t.Fatal("the key should still answer")
	}
	if m.statusBlinking() || !m.statusBlinkAt.IsZero() {
		t.Error("an experiment that is off may not move the line")
	}
}

// A message nobody pressed a key for doesn't blink: a line that moves at
// things you didn't do is the nagging this is meant to avoid.
func TestStatusLineDoesNotBlinkForAMessageNobodyAskedFor(t *testing.T) {
	m := statusModel(t)
	m.Flash("the theme changed under you")
	if m.statusBlinking() {
		t.Error("only a key's own answer blinks")
	}
}
