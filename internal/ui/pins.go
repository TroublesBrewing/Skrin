package ui

import (
	"sort"

	"github.com/lurioso/skrin/internal/vault"
)

// togglePin pins the note you're on, or takes it off the list — Obsidian's
// bookmarks. Recents answer "where was I"; pins answer "where do I live".
func (m *Model) togglePin() {
	rel, ok := m.subject()
	if !ok {
		m.flash = "Select a note to pin"
		return
	}
	for i, p := range m.pinned {
		if p == rel {
			m.pinned = append(m.pinned[:i:i], m.pinned[i+1:]...)
			m.flash = "Unpinned " + displayName(rel)
			return
		}
	}
	m.pinned = append(m.pinned, rel)
	sort.Strings(m.pinned)
	m.flash = "Pinned " + displayName(rel)
	if k := m.keyFor(inMain, actPins); k != "" {
		m.flash += " · " + k + " lists them"
	}
}

// openPins lists the pinned notes, and opens the one you pick the same way
// Go to note does.
func (m *Model) openPins() {
	c := &chooser{title: "Pinned notes", prompt: "name", empty: "No pinned note by that name", verb: "open · alt+←/→ split"}
	for _, rel := range m.pinned {
		if !m.vault.Exists(rel) {
			continue
		}
		c.items = append(c.items, choice{label: displayName(rel), detail: parentOf(rel), rel: rel, do: func() { m.goTo(rel, "") }})
	}
	if len(c.items) == 0 {
		m.flash = "No pinned notes yet"
		if k := m.keyFor(inMain, actPin); k != "" {
			m.flash += " · " + k + " pins the one you're on"
		}
		return
	}
	c.split = m.openSplit
	m.openChooser(c)
}

// keepPins drops pins whose note has gone, so the list never points at
// nothing. The vault changes under us all the time: Obsidian, Sync, E.
func (m *Model) keepPins() {
	out := m.pinned[:0]
	for _, rel := range m.pinned {
		if vault.IsNote(rel) && m.vault.Exists(rel) {
			out = append(out, rel)
		}
	}
	m.pinned = out
}
