package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"
)

// lineInput is a one-line text field with a cursor and the usual readline
// keys.
type lineInput struct {
	runes []rune
	cur   int
}

func (in *lineInput) set(s string) {
	in.runes = []rune(s)
	in.cur = len(in.runes)
}

func (in *lineInput) value() string { return string(in.runes) }

// insert types s at the cursor. Pasted newlines become spaces.
func (in *lineInput) insert(s string) {
	r := []rune(strings.NewReplacer("\r\n", " ", "\n", " ").Replace(s))
	in.runes = append(in.runes[:in.cur], append(r, in.runes[in.cur:]...)...)
	in.cur += len(r)
}

// handle applies an editing key and reports whether it used it.
func (in *lineInput) handle(k tea.KeyPressMsg) bool {
	switch k.String() {
	case "backspace", "ctrl+h":
		if in.cur > 0 {
			in.runes = append(in.runes[:in.cur-1], in.runes[in.cur:]...)
			in.cur--
		}
	case "delete":
		if in.cur < len(in.runes) {
			in.runes = append(in.runes[:in.cur], in.runes[in.cur+1:]...)
		}
	case "left", "ctrl+b":
		in.cur = max(in.cur-1, 0)
	case "right", "ctrl+f":
		in.cur = min(in.cur+1, len(in.runes))
	case "home", "ctrl+a":
		in.cur = 0
	case "end", "ctrl+e":
		in.cur = len(in.runes)
	case "ctrl+u":
		in.runes = in.runes[in.cur:]
		in.cur = 0
	case "ctrl+w":
		i := in.cur
		for i > 0 && in.runes[i-1] == ' ' {
			i--
		}
		for i > 0 && in.runes[i-1] != ' ' && in.runes[i-1] != '/' {
			i--
		}
		in.runes = append(in.runes[:i], in.runes[in.cur:]...)
		in.cur = i
	default:
		if k.Text == "" || k.Mod&(tea.ModCtrl|tea.ModAlt) != 0 {
			return false
		}
		in.insert(k.Text)
	}
	return true
}

func (in *lineInput) view(text, cursor lipgloss.Style) string {
	at, after := " ", ""
	if in.cur < len(in.runes) {
		at, after = string(in.runes[in.cur]), string(in.runes[in.cur+1:])
	}
	return text.Render(string(in.runes[:in.cur])) + cursor.Render(at) + text.Render(after)
}

type promptKind int

const (
	promptNewNote promptKind = iota
	promptNewFolder
	promptRename
)

// prompt asks for a name in the status line.
type prompt struct {
	kind   promptKind
	label  string // "New note in Filosofi/"
	target string // promptRename: the path being renamed
	in     lineInput
	err    string
}

func (m *Model) promptKey(k tea.KeyPressMsg) {
	switch k.String() {
	case "esc", "ctrl+c":
		m.prompt = nil
	case "enter":
		m.submitPrompt()
	default:
		if m.prompt.in.handle(k) {
			m.prompt.err = ""
		}
	}
}

func (m *Model) promptLine() string {
	p := m.prompt
	names := map[promptKind]string{promptNewNote: " NEW NOTE ", promptNewFolder: " NEW FOLDER ", promptRename: " RENAME "}
	left := m.st.pill.Render(names[p.kind]) + " " + m.st.text.Render(p.label+" ▸ ") + p.in.view(m.st.text, m.st.cursor)
	right := m.st.muted.Render("enter ok · esc cancel")
	if p.err != "" {
		right = m.st.errText.Render(p.err)
	}
	return spread(left, right, m.width)
}

// confirm asks a question in the status line. y runs yes; n runs no, or
// cancels when there is none; esc cancels.
type confirm struct {
	pill     string // " DELETE "
	question string
	keys     string // the answers, shown after the question
	danger   bool
	cancel   string // flash when cancelled
	yes, no  func()
}

func (m *Model) confirmKey(k tea.KeyPressMsg) {
	c := m.confirm
	switch k.String() {
	case "y", "Y":
		m.confirm = nil
		c.yes()
	case "n", "N":
		m.confirm = nil
		if c.no != nil {
			c.no()
		} else {
			m.flash = c.cancel
		}
	case "esc", "q", "ctrl+c":
		m.confirm = nil
		m.flash = c.cancel
	}
}

func (m *Model) confirmLine() string {
	c := m.confirm
	pill := m.st.pill
	if c.danger {
		pill = m.st.dangerPill
	}
	return spread(pill.Render(c.pill)+" "+m.st.text.Render(c.question)+" "+m.st.bold.Render(c.keys), "", m.width)
}

// choice is one row in a chooser.
type choice struct {
	label, detail string
	do            func()
}

// chooser is a filterable list in a floating box: move destinations,
// backlinks, the outline.
type chooser struct {
	title   string
	prompt  string             // before the filter field
	empty   string             // when nothing matches
	verb    string             // what enter does, for the footer
	none    func(query string) // what enter does when nothing matches, if anything
	items   []choice
	in      lineInput
	matches []int // indexes into items, best first
	cur     int
}

func (c *chooser) filter() {
	c.matches, c.cur = c.matches[:0], 0
	q := strings.ToLower(strings.TrimSpace(c.in.value()))
	if q == "" {
		for i := range c.items {
			c.matches = append(c.matches, i)
		}
		return
	}
	labels := make([]string, len(c.items))
	for i, it := range c.items {
		labels[i] = strings.ToLower(it.label)
	}
	for _, mt := range fuzzy.Find(q, labels) {
		c.matches = append(c.matches, mt.Index)
	}
}

func (m *Model) openChooser(c *chooser) {
	c.filter()
	m.chooser = c
}

func (m *Model) chooserKey(k tea.KeyPressMsg) {
	c := m.chooser
	switch k.String() {
	case "esc", "ctrl+c":
		m.chooser = nil
	case "enter":
		q := strings.TrimSpace(c.in.value())
		switch {
		case len(c.matches) > 0:
			it := c.items[c.matches[c.cur]]
			m.chooser = nil
			it.do()
		case c.none != nil && q != "":
			m.chooser = nil
			c.none(q)
		}
	case "up", "ctrl+p", "shift+tab":
		c.cur = max(c.cur-1, 0)
	case "down", "ctrl+n", "tab":
		c.cur = min(c.cur+1, max(len(c.matches)-1, 0))
	default:
		if c.in.handle(k) {
			c.filter()
		}
	}
}

func (m *Model) chooserBox() []string {
	c := m.chooser
	w := min(clamp(m.width*3/5, 40, 80), m.width-4)
	inner := w - 2
	rows := clamp(m.height-12, 3, 14)
	body := []string{" " + m.st.muted.Render(c.prompt+" ▸ ") + c.in.view(m.st.text, m.st.cursor), ""}
	off := max(0, c.cur-rows+1)
	for i := off; i < min(len(c.matches), off+rows); i++ {
		it := c.items[c.matches[i]]
		if i == c.cur {
			body = append(body, m.st.selFocus.Render(fit("  "+it.label+"  "+it.detail, inner)))
			continue
		}
		body = append(body, fit(m.st.text.Render("  "+it.label)+"  "+m.st.muted.Render(it.detail), inner))
	}
	if len(c.matches) == 0 {
		body = append(body, m.st.muted.Render("  "+c.empty))
	}
	body = append(body, "", " "+m.st.muted.Render("↑↓ choose · enter "+c.verb+" · esc cancel"))
	return m.box(c.title, body, w, len(body)+2, true)
}

// overlay draws box over base, centred horizontally and a third of the way
// down.
func (m *Model) overlay(base string, box []string) string {
	if len(box) == 0 {
		return base
	}
	x := max((m.width-ansi.StringWidth(box[0]))/2, 0)
	y := max((m.height-len(box))/3, 0)
	return m.overlayAt(base, box, x, y)
}

// overlayAt draws box over base with its top-left corner at x, y.
func (m *Model) overlayAt(base string, box []string, x, y int) string {
	top := lipgloss.NewLayer(strings.Join(box, "\n")).X(x).Y(y).Z(1)
	// The compositor trims trailing blanks; pad back to the full frame.
	lines := strings.Split(lipgloss.NewCompositor(lipgloss.NewLayer(base), top).Render(), "\n")
	for len(lines) < m.height {
		lines = append(lines, "")
	}
	for i := range lines {
		lines[i] = fit(lines[i], m.width)
	}
	return strings.Join(lines[:m.height], "\n")
}
