package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/version"
)

// copyPathOn is the feature's real state: a beta feature needs the build
// to allow beta, beta mode to be on, and its own switch on.
func (m *Model) copyPathOn() bool {
	return version.Beta && m.opts.Beta && m.opts.CopyPath
}

// copyPath is y and Y: the path of whatever is in hand goes to the system
// clipboard over OSC 52, the same way Ctrl+C sends a selection — so it
// works locally, over ssh and inside tmux with passthrough on, with no
// clipboard program of its own.
//
// What "in hand" means is not a new rule: it's targets(), the same answer
// r, m and d use — the marked items if there are any, otherwise the Files
// row under the cursor, or the open note when the note pane has the
// focus. A folder is a path like any other, so it copies too.
//
// y copies the vault-relative path (Books/Bilbo.md), Y the whole path as
// the disk spells it. The case grammar decides which is which: lowercase
// is the smaller thing, uppercase the larger. Several marked items come
// out one per line, which is what a shell expects.
func (m *Model) copyPath(full bool) tea.Cmd {
	if !m.copyPathOn() {
		m.flash = "Copying paths is off: it's a beta feature, switched on in Settings (" + note(m.keyFor(inMain, actHelp), "?") + " then tab)"
		return nil
	}
	ts := m.targets()
	if len(ts) == 0 {
		m.flash = "Nothing in hand: put the cursor on a note or a folder first"
		return nil
	}
	out := make([]string, len(ts))
	for i, rel := range ts {
		out[i] = rel
		if full {
			out[i] = m.vault.Abs(rel)
		}
	}
	s := strings.Join(out, "\n")
	if len(out) == 1 {
		m.flash = "Copied " + out[0]
	} else {
		m.flash = fmt.Sprintf("Copied %d paths", len(out))
	}
	// The same field Ctrl+C fills, so a terminal that won't hand its
	// clipboard back still has something for Ctrl+V to paste.
	m.copied = s
	return tea.SetClipboard(s)
}
