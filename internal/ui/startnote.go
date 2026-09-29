package ui

import (
	"path"
	"sort"

	"github.com/lurioso/skrin/internal/config"
	"github.com/lurioso/skrin/internal/vault"
	"github.com/lurioso/skrin/internal/version"
)

// startNoteOn is the feature's real state. There is no separate switch to
// turn it on: choosing a note is the switch, and no note chosen is off.
// A row that does nothing until you fill it can't arrive unasked, which
// is what an experiment has to promise.
func (m *Model) startNoteOn() bool {
	return version.Beta && m.opts.Beta
}

// startNote is the note this vault opens on, or "" for none. A note that
// has been deleted or renamed away counts as none rather than as an
// error: Skrin lands on the welcome card, and the setting is still there
// to be pointed at something else.
func (m *Model) startNote() string {
	if !m.startNoteOn() {
		return ""
	}
	rel := m.opts.Vault.StartNote
	if rel == "" || !vault.IsNote(rel) || !m.vault.Exists(rel) {
		return ""
	}
	return rel
}

// startNoteLabel is what the Settings row shows.
func (m *Model) startNoteLabel() string {
	rel := m.opts.Vault.StartNote
	switch {
	case rel == "":
		return "none — the welcome card"
	case !m.vault.Exists(rel):
		return rel + " (gone)"
	}
	return rel
}

// pickStartNote chooses the note this vault opens on. The last row clears
// it, so the welcome card is always one Enter away again — a setting you
// can't undo from the same place you set it is a trap.
func (m *Model) pickStartNote() {
	files, err := m.vault.Files()
	if err != nil {
		m.flash = err.Error()
		return
	}
	setCur := 0
	if m.manual != nil {
		setCur = m.manual.setCur
	}
	back := func() {
		m.openManual()
		m.manualGoTab(manualTabSettings)
		m.manual.setCur = setCur
	}
	var items []choice
	for _, f := range files {
		if !vault.IsNote(f) {
			continue
		}
		rel := f
		items = append(items, choice{
			label: rel,
			do:    func() { back(); m.setStartNote(rel) },
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].label < items[j].label })
	if m.opts.Vault.StartNote != "" {
		items = append([]choice{{
			label: "none — open the welcome card",
			do:    func() { back(); m.setStartNote("") },
		}}, items...)
	}
	if len(items) == 0 {
		m.flash = "No notes in this vault yet"
		return
	}
	m.openChooser(&chooser{
		title:  "Note to open on start",
		prompt: "Note",
		empty:  "No note by that name",
		verb:   "open it on start",
		items:  items,
		cancel: back,
	})
}

// setStartNote saves the choice into the vault's own settings file, not
// config.toml: a note path means nothing in another vault, and this one
// should follow the vault rather than the machine.
func (m *Model) setStartNote(rel string) {
	m.opts.Vault.StartNote = rel
	if err := config.SaveVaultSettings(m.vault.Root, m.opts.Vault); err != nil {
		m.flash = "couldn't save settings: " + err.Error()
		return
	}
	if rel == "" {
		m.flash = "Skrin will open on the welcome card"
		return
	}
	m.flash = "Skrin will open on " + path.Base(rel)
}
