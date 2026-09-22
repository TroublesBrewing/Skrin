package ui

import (
	"strings"
	"unicode"

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

// insert types s at the cursor. A one-line field has no lines, so every
// kind of break becomes a space: LF, CRLF and the bare CR a terminal
// paste sends.
func (in *lineInput) insert(s string) {
	r := []rune(strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(s))
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
	promptExtract
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
	rel           string // the note the row stands for, in Go to note
	// run is do for a row whose action hands Bubble Tea a command, as
	// quitting or handing a note to $EDITOR does; the palette's rows.
	run func() tea.Cmd
	// key is shown flush right, the way the palette shows each command's
	// key; also is matched by the filter without being shown, so a
	// command can be found by words other than its name.
	key, also string
}

// chooser is a filterable list in a floating box: move destinations,
// backlinks, the outline.
type chooser struct {
	title   string
	prompt  string                      // before the filter field
	empty   string                      // when nothing matches
	verb    string                      // what enter does, for the footer
	none    func(query string)          // what enter does when nothing matches, if anything
	split   func(rel string, left bool) // Alt+←/→: open the row's note in a split
	cancel  func()                      // Esc: where to go back to, if not just closing
	items   []choice
	in      lineInput
	matches []int // indexes into items, best first
	cur     int
	// byWords ranks whole words ahead of letters scattered through a row,
	// for lists searched by what a row means rather than its name: the
	// palette, where "toc" means the outline, not "op-t-i-o-ns c-onfig".
	byWords bool
	// remove, when set, is what d does to the highlighted row — for
	// lists the user built (folder templates), where removing is a
	// thing. Lists without it never see the key.
	remove func(choice)
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
	if c.byWords {
		c.matches = rankByWords(q, c.items)
		return
	}
	labels := make([]string, len(c.items))
	for i, it := range c.items {
		labels[i] = strings.ToLower(strings.TrimSpace(it.label + " " + it.also))
	}
	for _, mt := range fuzzy.Find(q, labels) {
		c.matches = append(c.matches, mt.Index)
	}
}

// rankByWords orders items for query q in three tiers: every word of q
// starting a word of the row's name; the same in its other words; then
// q's letters in order through the name, for a typo. Letters scattered
// through the other words are not a match: there are enough of them that
// nearly anything would be. Within the first two tiers the list keeps its
// own order.
func rankByWords(q string, items []choice) []int {
	qw := words(q)
	seen := map[int]bool{}
	var out []int
	take := func(i int) {
		if !seen[i] {
			seen[i] = true
			out = append(out, i)
		}
	}
	for i, it := range items {
		if startsWords(qw, it.label) {
			take(i)
		}
	}
	for i, it := range items {
		if startsWords(qw, it.label+" "+it.also) {
			take(i)
		}
	}
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = strings.ToLower(it.label)
	}
	for _, mt := range fuzzy.Find(q, names) {
		take(mt.Index)
	}
	return out
}

// words splits s into lowercase words at anything not a letter or digit.
func words(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// startsWords reports whether every word in q starts some word of s.
func startsWords(q []string, s string) bool {
	if len(q) == 0 {
		return false
	}
	sw := words(s)
	for _, w := range q {
		found := false
		for _, x := range sw {
			if strings.HasPrefix(x, w) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (m *Model) openChooser(c *chooser) {
	c.filter()
	m.chooser = c
}

// chooserKey handles a key in a list. The list's keys come from the keymap
// registry; the rest edit the filter.
func (m *Model) chooserKey(k tea.KeyPressMsg) tea.Cmd {
	c := m.chooser
	if c.remove != nil && k.String() == "d" && len(c.matches) > 0 {
		it := c.items[c.matches[c.cur]]
		m.chooser = nil
		c.remove(it)
		return nil
	}
	switch a := m.actionIn(inList, k.String()); a {
	case actCancel:
		m.chooser = nil
		if c.cancel != nil {
			c.cancel()
		}
	case actSplitLeft, actSplitRight:
		if c.split == nil || len(c.matches) == 0 {
			return nil
		}
		it := c.items[c.matches[c.cur]]
		m.chooser = nil
		c.split(it.rel, a == actSplitLeft)
	case actPick:
		q := strings.TrimSpace(c.in.value())
		switch {
		case len(c.matches) > 0:
			it := c.items[c.matches[c.cur]]
			m.chooser = nil
			if it.run != nil {
				return it.run()
			}
			if it.do != nil { // rows without an action just close the list
				it.do()
			}
		case c.none != nil && q != "":
			m.chooser = nil
			c.none(q)
		}
	case actUp:
		c.cur = max(c.cur-1, 0)
	case actDown:
		c.cur = min(c.cur+1, max(len(c.matches)-1, 0))
	default:
		if c.in.handle(k) {
			c.filter()
		}
	}
	return nil
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
		text := "  " + it.label + "  " + it.detail
		if it.key != "" {
			// The key sits flush right, and the name gives way to it.
			kw := ansi.StringWidth(it.key) + 2
			text = fit(text, max(inner-kw, 0))
			if i == c.cur {
				body = append(body, m.st.selFocus.Render(text+fit(it.key+"  ", kw)))
				continue
			}
			body = append(body, m.st.text.Render(text)+m.st.flash.Render(fit(it.key+"  ", kw)))
			continue
		}
		if i == c.cur {
			body = append(body, m.st.selFocus.Render(fit(text, inner)))
			continue
		}
		body = append(body, fit(m.st.text.Render("  "+it.label)+"  "+m.st.muted.Render(it.detail), inner))
	}
	if len(c.matches) == 0 {
		body = append(body, m.st.muted.Render("  "+c.empty))
	}
	hint := "↑↓ choose · enter " + c.verb
	if c.remove != nil {
		hint += " · d remove"
	}
	hint += " · esc cancel"
	body = append(body, "", " "+m.st.muted.Render(hint))
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
