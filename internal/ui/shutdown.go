package ui

// ShutdownMsg says the operating system is taking Skrin away: the
// terminal window was closed (SIGHUP) or something asked the process to
// end (SIGTERM). It is sent by main, which keeps the signals to itself
// rather than letting Bubble Tea turn them into a quit that never reaches
// this model.
//
// Why it exists: after autosave (v0.37.0) the everyday path to lost text
// was closed, but not this one. Closing the terminal window killed the
// process where it stood, and anything typed since the last autosave went
// with it. That is data loss, not a missing feature, so it is fixed and
// the fix is on — [[skrin styrdokument]], *Takten*: "Något som är trasigt,
// förlorar data eller inte går att ta sig ur rättas direkt, och
// rättningen är på."
type ShutdownMsg struct{}

// shutdown writes what the editor is holding, if it safely can, and says
// whether it wrote anything.
//
// It saves by exactly the same rules as an ordinary autosave, deliberately:
// a note that changed on disk underneath is still not overwritten, and the
// HELD state still stands. Being shut down is no reason to decide
// something that is the user's to decide — that case is a card of its own
// (the conflict copy), and this one closes the common path.
func (m *Model) shutdown() bool {
	if m.editor == nil || m.conflict != nil || !m.editor.Dirty() {
		return false
	}
	before := m.edit.base
	m.autosave()
	return m.edit.base != before
}
