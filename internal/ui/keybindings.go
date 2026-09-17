package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/config"
)

// bindRef names one row of the Keys tab: an action and the context it
// works in. That pair is what an override is stored against, so it stays
// the same row whatever keys it ends up with.
type bindRef struct {
	where string
	act   action
}

// keysMode is what the Keys tab is doing: reading it, waiting for the key
// to give a binding, or waiting for a y/n.
type keysMode int

const (
	keysBrowse keysMode = iota
	keysCapture
	keysAsking
)

// keysAsk is a question the Keys tab is waiting on, with what to do if
// the answer is yes.
type keysAsk struct {
	text string
	yes  func()
}

// protected are the ways back: the manual itself, and quitting. Every
// other binding can be left without a key and given a new one later, but
// these two are how you would reach the Keys tab to fix it, so they are
// never allowed to become unreachable from inside Skrin.
var protected = map[action]bool{actHelp: true, actQuit: true}

// keyLines is the Keys tab at text width w: every key Skrin knows,
// grouped as the registry groups them, showing the keys in force rather
// than the ones it ships with.
func (m *Model) keyLines(w int) []manualLine {
	var out []manualLine
	group := ""
	descW := max(w-manualKeyW-4, 10)
	for _, b := range defaultBindings {
		if b.group != group {
			group = b.group
			if len(out) > 0 {
				out = append(out, manualLine{})
			}
			out = append(out, manualLine{plain: group, styled: m.st.titleFocus.Render(group), level: 1})
		}
		keys := b.keys
		rebindable := b.act != actNone
		if rebindable {
			keys = m.keys.bound(b.where, b.act)
		}
		ref := bindRef{b.where, b.act}
		cur := rebindable && m.manual != nil && m.manual.cur == ref
		label := keyLabel(keys)
		if label == "" {
			label = "—"
		}
		mark, markPlain := "  ", "  "
		switch {
		case !rebindable:
			mark, markPlain = m.st.muted.Render(" ·"), " ·"
		case m.keys.changed(b.where, b.act):
			mark, markPlain = m.st.brand.Render(" •"), " •"
		}
		for i, d := range wrap(b.help, descW) {
			shown := label
			if i > 0 {
				shown, mark, markPlain = "", "  ", "  "
			}
			pad := strings.Repeat(" ", max(manualKeyW-ansi.StringWidth(shown), 0))
			plain := markPlain + " " + shown + pad + " " + d
			styled := mark + " " + m.st.flash.Render(shown+pad) + " " + m.st.text.Render(d)
			if cur {
				// The row under the cursor is one solid bar, the way a
				// picked row looks everywhere else in Skrin.
				styled = m.st.selFocus.Render(fit(plain, w))
			}
			l := manualLine{
				plain:  plain,
				styled: styled,
				// The filter matches the keys as config.toml writes
				// them too, so looking for "ctrl+k" finds the row that
				// shows Ctrl-k.
				search: plain + " " + strings.Join(keys, " ") + " " + b.where,
			}
			if rebindable && i == 0 {
				l.bind = &ref
			}
			out = append(out, l)
		}
	}
	return out
}

// keysTabKey handles a key press in the Keys tab.
func (m *Model) keysTabKey(k tea.KeyPressMsg) {
	h := m.manual
	switch h.mode {
	case keysCapture:
		m.captureBinding(k)
		return
	case keysAsking:
		m.keysAskKey(k)
		return
	}
	if h.filtering {
		switch k.String() {
		case "esc":
			h.filtering = false
			h.in.set("")
		case "enter":
			h.filtering = false
		default:
			h.in.handle(k)
		}
		h.off = 0
		m.firstBinding()
		return
	}
	switch k.String() {
	case "esc":
		if h.in.value() != "" {
			h.in.set("")
			m.firstBinding()
			return
		}
		m.manual = nil
		return
	case "q":
		m.manual = nil
		return
	case "/":
		h.filtering, h.note = true, ""
	case "j", "down":
		m.moveBinding(1)
	case "k", "up":
		m.moveBinding(-1)
	case "ctrl+d", "pgdown":
		m.moveBinding(max(m.manualVis()/2, 1))
	case "ctrl+u", "pgup":
		m.moveBinding(-max(m.manualVis()/2, 1))
	case "g", "home":
		m.firstBinding()
	case "G", "end":
		m.lastBinding()
	case "enter":
		if _, ok := m.curBinding(); ok {
			h.mode, h.note = keysCapture, ""
		}
	case "r":
		m.resetBinding()
	case "R":
		m.askResetAll()
	}
}

// bindingRows is the rows of the Keys tab that can be selected, in the
// order they show — which a filter narrows, so j and k walk only what you
// are looking at.
func (m *Model) bindingRows() []bindRef {
	var out []bindRef
	for _, l := range m.manualLines(m.manualWidth()) {
		if l.bind != nil {
			out = append(out, *l.bind)
		}
	}
	return out
}

// curBinding is the binding under the cursor, if the cursor is on one.
func (m *Model) curBinding() (bindRef, bool) {
	for _, b := range m.bindingRows() {
		if b == m.manual.cur {
			return b, true
		}
	}
	return bindRef{}, false
}

func (m *Model) firstBinding() {
	if rows := m.bindingRows(); len(rows) > 0 {
		m.manual.cur = rows[0]
	}
	m.showCursor()
}

func (m *Model) lastBinding() {
	if rows := m.bindingRows(); len(rows) > 0 {
		m.manual.cur = rows[len(rows)-1]
	}
	m.showCursor()
}

// moveBinding walks the cursor n rows on, stopping at either end.
func (m *Model) moveBinding(n int) {
	rows := m.bindingRows()
	if len(rows) == 0 {
		return
	}
	at := 0
	for i, b := range rows {
		if b == m.manual.cur {
			at = i
			break
		}
	}
	m.manual.cur = rows[clamp(at+n, 0, len(rows)-1)]
	m.showCursor()
}

// showCursor scrolls so the binding under the cursor is on screen, with
// its group heading above it where there is room.
func (m *Model) showCursor() {
	h := m.manual
	lines := m.manualLines(m.manualWidth())
	vis := m.manualVis()
	at := -1
	for i, l := range lines {
		if l.bind != nil && *l.bind == h.cur {
			at = i
			break
		}
	}
	if at < 0 {
		h.off = clamp(h.off, 0, max(len(lines)-vis, 0))
		return
	}
	switch {
	case at < h.off+1:
		h.off = max(at-1, 0)
	case at >= h.off+vis:
		h.off = at - vis + 1
	}
	h.off = clamp(h.off, 0, max(len(lines)-vis, 0))
}

// captureBinding takes the next key pressed as the new key for the
// binding under the cursor.
func (m *Model) captureBinding(k tea.KeyPressMsg) {
	h := m.manual
	s := k.String()
	if s == "esc" {
		h.mode, h.note = keysBrowse, ""
		return
	}
	b, ok := m.curBinding()
	if !ok {
		h.mode = keysBrowse
		return
	}
	label := keyLabel([]string{s})
	help := bindingHelp(b)
	// Already its own: nothing to do, and saying so beats silence.
	for _, cur := range m.keys.bound(b.where, b.act) {
		if cur == s {
			h.mode, h.note = keysBrowse, label+" is already "+quote(help)+"."
			return
		}
	}
	switch holder, rebindable := m.keys.holder(b.where, s); {
	case holder != "" && !rebindable:
		// A key the component types or handles itself. Refusing beats
		// taking it and having it half-work.
		h.note = label + " is " + quote(holder) + " here, which can't be changed. Try another key."
		return
	case holder != "" && m.keys.act(b.where, s) != b.act:
		other := m.keys.act(b.where, s)
		h.mode = keysBrowse
		if protected[other] && len(m.keys.bound(b.where, other)) == 1 {
			h.note = label + " is " + quote(holder) + ", the way back to this manual. Give that one another key first."
			return
		}
		m.askTake(b, other, s, label, help, holder)
		return
	}
	m.bindKey(b, s)
	h.mode, h.note = keysBrowse, label+" is now "+quote(help)+". r puts it back."
}

// askTake asks whether to take a key from the binding that has it, and if
// so moves the cursor there so it can be given a new one on the spot.
func (m *Model) askTake(b bindRef, other action, s, label, help, holder string) {
	h := m.manual
	left := len(m.keys.bound(b.where, other)) - 1
	tail := "It would be left with no key."
	if left > 0 {
		tail = "It would keep " + plural(left, "other key") + "."
	}
	h.mode = keysAsking
	h.ask = &keysAsk{
		text: label + " is already " + quote(holder) + " here. Give it to " + quote(help) + "? " + tail,
		yes: func() {
			m.bindKey(b, s)
			m.manual.note = label + " is now " + quote(help) + "."
			if len(m.keys.bound(b.where, other)) == 0 {
				m.manual.cur = bindRef{b.where, other}
				m.showCursor()
				m.manual.note = quote(holder) + " has no key now — enter gives it one, r puts the default back."
			}
		},
	}
}

// keysAskKey answers the y/n the Keys tab is waiting on.
func (m *Model) keysAskKey(k tea.KeyPressMsg) {
	h := m.manual
	switch k.String() {
	case "y", "Y", "enter":
		yes := h.ask.yes
		h.mode, h.ask = keysBrowse, nil
		yes()
	case "n", "N", "esc", "q":
		h.mode, h.ask, h.note = keysBrowse, nil, "Left as it was."
	}
}

// bindKey gives b the key s and writes the keymap to config.toml.
func (m *Model) bindKey(b bindRef, s string) {
	m.keys.set(b.where, b.act, s)
	m.saveKeys()
}

// resetBinding puts the binding under the cursor back to the keys Skrin
// ships with.
func (m *Model) resetBinding() {
	h := m.manual
	b, ok := m.curBinding()
	if !ok {
		return
	}
	if !m.keys.changed(b.where, b.act) {
		h.note = quote(bindingHelp(b)) + " is already on its own default."
		return
	}
	m.keys.reset(b.where, b.act)
	m.saveKeys()
	h.note = quote(bindingHelp(b)) + " is back to " + keyLabel(m.keys.bound(b.where, b.act)) + "."
}

// askResetAll offers to drop every key the user changed. There is no undo
// for it, which is why it asks.
func (m *Model) askResetAll() {
	h := m.manual
	n := 0
	for _, acts := range m.keys.over {
		n += len(acts)
	}
	if n == 0 {
		h.note = "Every key is already as Skrin ships it."
		return
	}
	h.mode = keysAsking
	h.ask = &keysAsk{
		text: "Put every key back to Skrin's own? " + plural(n, "changed key") + " would be forgotten.",
		yes: func() {
			m.keys.resetAll()
			m.saveKeys()
			m.manual.note = "Every key is back to Skrin's own."
			m.firstBinding()
		},
	}
}

// saveKeys writes the keymap to config.toml, so a key you changed is
// still yours next time.
func (m *Model) saveKeys() {
	m.opts.Config.Keys = m.keys.overrides()
	if err := config.Save(m.opts.Config); err != nil {
		m.manual.note = "couldn't save the keymap: " + err.Error()
	}
}

// keysStatus is the status line under the Keys tab: what it is waiting
// for, or what you can do from here.
func (m *Model) keysStatus() string {
	h := m.manual
	switch {
	case h.mode == keysCapture:
		b, _ := m.curBinding()
		return spread(m.st.pill.Render(" PRESS A KEY ")+" "+m.st.text.Render(note(h.note, "the new key for "+quote(bindingHelp(b)))), m.st.muted.Render("esc cancel"), m.width)
	case h.mode == keysAsking:
		return spread(m.st.dangerPill.Render(" KEYS ")+" "+m.st.text.Render(h.ask.text), m.st.bold.Render("y / n"), m.width)
	case h.filtering:
		return spread(m.st.pill.Render(" FILTER ")+" "+h.in.view(m.st.text, m.st.cursor), m.st.muted.Render("enter keep · esc clear"), m.width)
	}
	left := m.st.pill.Render(" KEYS ")
	switch q := h.in.value(); {
	case h.note != "":
		left += " " + m.st.text.Render(h.note)
	case q != "":
		left += " " + m.st.text.Render(plural(len(m.bindingRows()), "key")+" matching “"+q+"” · esc shows everything")
	default:
		left += " " + m.st.muted.Render("• is one you changed · · is the component's own")
	}
	return spread(left, m.st.muted.Render("enter rebind · r/R default · / find · tab settings · esc close"), m.width)
}

// bindingHelp is what a binding says it does, from the registry.
func bindingHelp(b bindRef) string {
	for _, d := range defaultBindings {
		if d.where == b.where && d.act == b.act {
			return d.help
		}
	}
	return ""
}

func quote(s string) string { return "“" + s + "”" }

// note is s, or fallback when s is empty.
func note(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
