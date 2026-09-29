package ui

import (
	"fmt"

	"github.com/lurioso/skrin/internal/version"
)

// outgoingOn is the feature's real state: a beta feature needs the build
// to allow beta, beta mode to be on, and its own switch on.
func (m *Model) outgoingOn() bool {
	return version.Beta && m.opts.Beta && m.opts.Outgoing
}

// showOutgoing lists the notes this one links to — the other half of
// Obsidian's backlinks pane, and the half Skrin only had as the spread
// function outgoing([[note]]).
//
// It is the mirror of showBacklinks on purpose: same chooser, same way of
// leaving the editor, same wording. A panel that answers the opposite
// question should not have to be learned twice.
//
// There is no key. Every new key is a cost the steering document asks to
// be counted, this is a lookup rather than a move, and what deserves a
// key is precisely what the card "Paletten som gränssnitt" is for. Until
// then it lives in Ctrl+P, beside Merge and Rename a tag.
func (m *Model) showOutgoing() {
	if !m.outgoingOn() {
		m.flash = "Outgoing links is off: it's a beta feature, switched on in Settings (" + note(m.keyFor(inMain, actHelp), "?") + " then tab)"
		return
	}
	rel, ok := m.subject()
	if !ok {
		m.flash = "Select a note to see what it links to"
		return
	}
	out := m.idx.Outgoing(rel)
	if len(out) == 0 {
		m.flash = describe([]string{rel}) + " doesn't link anywhere yet"
		return
	}
	c := &chooser{title: fmt.Sprintf("Links from %s (%d)", displayName(rel), len(out)), prompt: "filter", empty: "No match", verb: "open"}
	for _, target := range out {
		target := target
		c.items = append(c.items, choice{
			label: displayName(target),
			do: func() {
				// Picking one leaves the note being edited, which is the
				// same save-then-go Esc already does, not a special case.
				if m.editor != nil {
					if m.saveEdit(true); m.editor != nil {
						return // a conflict came up; the dialog has focus now
					}
				}
				m.open(target)
			},
		})
	}
	m.openChooser(c)
}
