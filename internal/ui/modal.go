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
		r := []rune(k.Text)
		in.runes = append(in.runes[:in.cur], append(r, in.runes[in.cur:]...)...)
		in.cur += len(r)
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

// confirm asks a yes/no question in the status line.
type confirm struct {
	question string
	yes      func()
}

func (m *Model) confirmKey(k tea.KeyPressMsg) {
	switch k.String() {
	case "y", "Y":
		c := m.confirm
		m.confirm = nil
		c.yes()
	case "n", "N", "esc", "q", "ctrl+c":
		m.confirm = nil
		m.flash = "Nothing deleted"
	}
}

// picker chooses the destination folder for a move, with fuzzy filtering.
type picker struct {
	sources []string
	folders []string // candidate destinations
	labels  []string // how each folder is shown, and matched
	in      lineInput
	matches []int // indexes into folders, best first
	cur     int
}

func (p *picker) filter() {
	p.matches, p.cur = p.matches[:0], 0
	q := strings.ToLower(strings.TrimSpace(p.in.value()))
	if q == "" {
		for i := range p.folders {
			p.matches = append(p.matches, i)
		}
		return
	}
	lower := make([]string, len(p.labels))
	for i, l := range p.labels {
		lower[i] = strings.ToLower(l)
	}
	for _, mt := range fuzzy.Find(q, lower) {
		p.matches = append(p.matches, mt.Index)
	}
}

func (m *Model) pickerKey(k tea.KeyPressMsg) {
	p := m.picker
	switch k.String() {
	case "esc", "ctrl+c":
		m.picker = nil
	case "enter":
		if len(p.matches) > 0 {
			dest := p.folders[p.matches[p.cur]]
			m.picker = nil
			m.moveTo(dest, p.sources)
		}
	case "up", "ctrl+p", "shift+tab":
		p.cur = max(p.cur-1, 0)
	case "down", "ctrl+n", "tab":
		p.cur = min(p.cur+1, max(len(p.matches)-1, 0))
	default:
		if p.in.handle(k) {
			p.filter()
		}
	}
}

func (m *Model) pickerBox() []string {
	p := m.picker
	w := min(clamp(m.width/2, 40, 64), m.width-4)
	inner := w - 2
	rows := clamp(m.height-12, 3, 12)
	body := []string{" " + m.st.muted.Render("to ▸ ") + p.in.view(m.st.text, m.st.cursor), ""}
	off := max(0, p.cur-rows+1)
	for i := off; i < min(len(p.matches), off+rows); i++ {
		label := fit("  "+p.labels[p.matches[i]], inner)
		if i == p.cur {
			label = m.st.selFocus.Render(label)
		} else {
			label = m.st.dir.Render(label)
		}
		body = append(body, label)
	}
	if len(p.matches) == 0 {
		body = append(body, m.st.muted.Render("  No folder matches"))
	}
	body = append(body, "", " "+m.st.muted.Render("↑↓ choose · enter move · esc cancel"))
	return m.box("Move "+describe(p.sources), body, w, len(body)+2, true)
}

// overlay draws box over base, centred horizontally and a third of the way
// down.
func (m *Model) overlay(base string, box []string) string {
	if len(box) == 0 {
		return base
	}
	x := max((m.width-ansi.StringWidth(box[0]))/2, 0)
	y := max((m.height-len(box))/3, 0)
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
