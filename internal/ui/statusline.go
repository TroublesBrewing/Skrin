package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/version"
)

// statusBlinkFor is how long the line's answer stands out before settling
// into its ordinary colour. Long enough to be caught out of the corner of
// an eye, short enough that it is over before it can annoy.
const statusBlinkFor = 700 * time.Millisecond

// statusBlinkMsg only asks for a redraw: the blink is over, so the flash
// should be drawn in its ordinary colour from now on.
type statusBlinkMsg struct{}

// statusSaysWhereOn is the feature's real state: a beta feature needs the
// build to allow beta, beta mode to be on, and its own switch on.
func (m *Model) statusSaysWhereOn() bool {
	return version.Beta && m.opts.Beta && m.opts.StatusSaysWhere
}

// paneWord is the mode pill in the status line. Files and the note both
// read VIEW today, though d, r and n mean different things in each, and
// the charter asks for a context to be visible rather than guessed. With
// the experiment on, the pill says which pane the keys will land in.
func (m *Model) paneWord() string {
	if !m.statusSaysWhereOn() {
		return " VIEW "
	}
	if m.focus == paneNote {
		return " NOTE "
	}
	return " FILES "
}

// armStatusBlink starts the blink when a key has just put a message in the
// status line. The report behind it: "tangenten finns, men ledtråden om
// den sitter i en rad jag inte tittade på" — so the line moves when it
// answers, because movement is seen in peripheral vision and a word
// swapped for another word is not.
//
// It is armed from the key path alone. A message that arrives on its own
// — a theme change, a watcher's complaint — hasn't just been asked for by
// a keystroke, and a line that blinks at things nobody did would be the
// nagging this is meant to avoid.
func (m *Model) armStatusBlink() tea.Cmd {
	if !m.statusSaysWhereOn() || m.flash == "" {
		return nil
	}
	m.statusBlinkAt = time.Now()
	return tea.Tick(statusBlinkFor, func(time.Time) tea.Msg { return statusBlinkMsg{} })
}

// statusBlinking reports whether the flash is still inside its blink.
func (m *Model) statusBlinking() bool {
	return m.statusSaysWhereOn() && !m.statusBlinkAt.IsZero() && time.Since(m.statusBlinkAt) < statusBlinkFor
}
