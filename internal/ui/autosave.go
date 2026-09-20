package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// autosaveWait is how long text may sit in the editor before Skrin writes
// it out by itself. It's a ceiling, not a pause: typing on doesn't put it
// off, since the whole point is that nothing you've written is more than a
// moment away from disk.
const autosaveWait = 1500 * time.Millisecond

// autosaveMsg says the editor has held unsaved text for autosaveWait.
type autosaveMsg struct{}

// armAutosave starts the clock when the editor has text that isn't on disk
// yet, and does nothing when a clock is already running or there's nothing
// to write. It's called at the end of every update, so every way of
// changing the text — typing, pasting, undo, a template — is covered.
func (m *Model) armAutosave() tea.Cmd {
	if m.editor == nil || m.autosaving || !m.editor.Dirty() {
		return nil
	}
	m.autosaving = true
	return tea.Tick(autosaveWait, func(time.Time) tea.Msg { return autosaveMsg{} })
}

// autosave writes the note being edited without saying anything and
// without interrupting. It is deliberately more timid than Ctrl+S: if the
// note changed on disk meanwhile it writes nothing and waits, because the
// question of whose version wins belongs to a moment you chose — an
// explicit save, or leaving the editor — and never to the middle of a
// sentence. Ctrl+S and Esc still raise it, exactly as before.
//
// Due dates and the offer to move a heading's links along stay where they
// are too: they belong to leaving the note, not to writing it out.
func (m *Model) autosave() {
	s := &m.edit
	if m.editor == nil || m.conflict != nil || !m.editor.Dirty() {
		return
	}
	text := m.editor.Text()
	if text == s.base {
		return
	}
	disk, err := m.vault.Read(s.rel)
	if err != nil || disk != s.base {
		// Gone or changed under us. Writing would decide something that
		// is yours to decide, so it waits for Ctrl+S or Esc — but it says
		// so, and goes on saying so, because from here until you answer
		// what you type is only in the editor.
		if !s.held {
			s.held = true
			m.flash = displayName(s.rel) + " changed on disk · nothing of yours is written until you answer · ctrl+s decides"
		}
		return
	}
	s.held = false
	if !s.snapshotted {
		if err := m.snaps.Save(s.rel, s.base); err != nil {
			m.flash = "Saving by itself, but u won't undo this edit: " + err.Error()
		}
		s.snapshotted = true
	}
	if err := m.vault.Write(s.rel, text); err != nil {
		// Rare, and too important to swallow: a full disk or a read-only
		// vault means the text is only in the editor.
		m.flash = "Couldn't save by itself: " + err.Error() + " · ctrl+s to try again"
		return
	}
	s.base = text
	m.editor.MarkSaved()
}
