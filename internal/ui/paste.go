package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// clipboardWait is how long Ctrl+V waits for the terminal to hand over its
// clipboard before falling back to what was copied here.
const clipboardWait = 300 * time.Millisecond

// pasteTimeoutMsg says the terminal never answered the clipboard request.
type pasteTimeoutMsg struct{}

// askPaste is Ctrl+V. A program in a terminal can't reach the system
// clipboard by itself: it asks the terminal for it over OSC 52 and waits
// for the answer. Plenty of terminals write the clipboard but refuse to
// read it back — it would let anything at the far end of an ssh session
// read what you copied — so when no answer comes we fall back to what
// Ctrl+C copied here, and say what happened either way.
//
// Ctrl+Shift+V still works and always did: that one the terminal pastes
// itself, and it arrives as an ordinary paste.
func (m *Model) askPaste() tea.Cmd {
	m.pasting = true
	return tea.Batch(
		tea.Cmd(tea.ReadClipboard),
		tea.Tick(clipboardWait, func(time.Time) tea.Msg { return pasteTimeoutMsg{} }),
	)
}

// pasteFrom puts what the terminal gave back, or what Ctrl+C copied here
// when it gave nothing.
func (m *Model) pasteFrom(clip string, answered bool) {
	switch {
	case clip != "":
		m.pasted(clip, "")
	case answered && m.copied == "":
		m.flash = "The clipboard is empty" + m.noImageToolNote()
	case m.copied != "":
		m.pasted(m.copied, " · your terminal won't hand its clipboard over, so this is what you copied in Skrin")
	default:
		m.flash = "Nothing to paste: your terminal won't hand its clipboard over, and ctrl+shift+v pastes straight into Skrin"
	}
}

func (m *Model) pasted(s, note string) {
	if !m.paste(s) {
		if m.book != nil {
			m.flash = "Nothing to paste into here: Tab to a field first"
			return
		}
		m.flash = "Nothing here takes text: open a note in the editor first"
		return
	}
	m.flash = "Pasted" + note
}

// noImageToolNote owns up, on an empty clipboard, to the one thing that
// can look like an empty clipboard without being one: an image sitting
// there that Skrin can't reach, because the desktop has no program to
// hand it over. Said only when the clipboard came back empty, so it
// never nags a paste that worked.
func (m *Model) noImageToolNote() string {
	if !m.noImageTool {
		return ""
	}
	return " · an image in it can't be reached from here: install wl-clipboard (Wayland), xclip (X11) or pngpaste (macOS)"
}
