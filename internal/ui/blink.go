package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lurioso/skrin/internal/config"
)

// cursorBlinkMsg toggles the input cursor between drawn and blank, so the
// active field is easy to spot everywhere there is one: the editor, quick
// note, go-to-a-note, the Book Card, every prompt and filter.
type cursorBlinkMsg struct{}

// blinkGap turns the CursorBlink setting's word into a period and whether
// the cursor blinks at all. Anything unrecognised is medium.
func blinkGap(word string) (time.Duration, bool) {
	switch word {
	case "off":
		return 0, false
	case "slow":
		return time.Second, true
	case "fast":
		return 250 * time.Millisecond, true
	default: // "medium" and anything unknown
		return 500 * time.Millisecond, true
	}
}

// armCursorBlink keeps the input cursor blinking while a field is focused.
// Like the autosave and habit clocks it is called at the end of every
// update, and arms one tick only when blinking is wanted and none is
// already on its way. With blink off, the cursor is always drawn.
func (m *Model) armCursorBlink() tea.Cmd {
	gap, on := blinkGap(m.opts.CursorBlink)
	if !on {
		m.cursorOn = true
		return nil
	}
	if m.cursorBlinking {
		return nil
	}
	m.cursorBlinking = true
	return tea.Tick(gap, func(time.Time) tea.Msg { return cursorBlinkMsg{} })
}

// cursorStyle is the style for a single-line field's cursor cell: the
// cursor style while the blink is on-phase, the plain text style while it
// is off — so the cell reads as ordinary text with no block behind it.
func (m *Model) cursorStyle() lipgloss.Style {
	if m.cursorOn {
		return m.st.cursor
	}
	return m.st.text
}

// cursorBlinkChoices are the speeds the Cursor blink setting offers, in
// order, with the word saved to config.toml.
var cursorBlinkChoices = []struct{ word, note string }{
	{"off", "the cursor is steady, always drawn"},
	{"slow", "about a second on, a second off"},
	{"medium", "about half a second each way — the default"},
	{"fast", "about a quarter second each way"},
}

// pickCursorBlink is the Settings row for how fast the input cursor
// blinks. Choosing takes effect at once and saves to config.toml.
func (m *Model) pickCursorBlink() {
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
	for _, c := range cursorBlinkChoices {
		c := c
		items = append(items, choice{label: c.word, detail: c.note, do: func() {
			m.opts.CursorBlink = c.word
			m.opts.Config.General.CursorBlink = c.word
			m.cursorOn = true
			back()
			if err := config.Save(m.opts.Config); err != nil {
				m.flash = "couldn't save settings: " + err.Error()
				return
			}
			m.flash = "Cursor blink: " + c.word
		}})
	}
	m.manual = nil
	m.openChooser(&chooser{
		title:  "Cursor blink",
		prompt: "Speed",
		empty:  "No speed by that name",
		verb:   "choose",
		items:  items,
		cancel: back,
	})
}
