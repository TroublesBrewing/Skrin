package ui

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/vault"
)

func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.WindowTitle = "Skrin · " + m.vault.Name()
	return v
}

func (m *Model) render() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	l := m.layout()
	rows := m.header()
	var cols [][]string
	if l.treeW > 0 {
		cols = append(cols, m.treePane(l.treeW, l.bodyH))
	}
	cols = append(cols, m.listPane(l.listW, l.bodyH), m.notePane(l.noteW, l.bodyH))
	for i := 0; i < l.bodyH; i++ {
		var b strings.Builder
		for _, c := range cols {
			b.WriteString(c[i])
		}
		rows = append(rows, b.String())
	}
	rows = append(rows, m.statusLine())
	return strings.Join(rows[:min(len(rows), m.height)], "\n")
}

func (m *Model) header() []string {
	left := []string{
		m.st.brand.Render("Skrin") + m.st.muted.Render(" · ") + m.st.bold.Render(m.vault.Name()),
		m.st.muted.Render("unofficial TUI for Obsidian vaults"),
		m.st.muted.Render(tildePath(m.vault.Root)),
	}
	right := []string{m.st.muted.Render("◐ " + m.pal.Name), "", ""}
	out := make([]string, headerHeight)
	for i := range out {
		logoRow := strings.Repeat(" ", m.logoW)
		if i < len(m.logo) {
			logoRow = m.logo[i]
		}
		out[i] = spread(" "+logoRow+"  "+left[i], right[i], m.width)
	}
	return out
}

func (m *Model) treePane(w, h int) []string {
	inner, vis := w-2, h-2
	t := &m.tree
	var body []string
	for i := t.off; i < min(len(t.rows), t.off+vis); i++ {
		r := t.rows[i]
		icon := "  "
		if r.hasKids {
			icon = "▸ "
			if t.expanded[r.path] {
				icon = "▾ "
			}
		}
		text := fit(" "+strings.Repeat("  ", r.depth)+icon+r.name, inner)
		st := m.st.text
		if r.path == "" {
			st = m.st.bold
		}
		switch {
		case i == t.cur && m.focus == paneTree:
			text = m.st.selFocus.Render(text)
		case i == t.cur:
			text = m.st.selBlur.Render(text)
		default:
			text = st.Render(text)
		}
		body = append(body, text)
	}
	return m.box("Folders", body, w, h, m.focus == paneTree)
}

func (m *Model) listPane(w, h int) []string {
	inner, vis := w-2, h-2
	var body []string
	if len(m.entries) == 0 {
		body = append(body, m.st.muted.Render(fit("  (empty folder)", inner)))
	}
	now := time.Now()
	for i := m.listOff; i < min(len(m.entries), m.listOff+vis); i++ {
		e := m.entries[i]
		name, st, date := e.Name, m.st.text, ""
		switch {
		case e.IsDir:
			name, st = "▸ "+name+"/", m.st.dir
		case vault.IsNote(name):
			name, date = "  "+strings.TrimSuffix(name, path.Ext(name)), shortDate(e.ModTime, now)
		default:
			name, st, date = "  "+name, m.st.muted, shortDate(e.ModTime, now)
		}
		left := fit(" "+name, max(inner-8, 1))
		right := fmt.Sprintf(" %6s ", date)
		switch {
		case i == m.listCur && m.focus == paneList:
			body = append(body, m.st.selFocus.Render(fit(left+right, inner)))
		case i == m.listCur:
			body = append(body, m.st.selBlur.Render(fit(left+right, inner)))
		default:
			body = append(body, st.Render(left)+m.st.muted.Render(right))
		}
	}
	title := m.vault.Name() + "/"
	if m.cwd != "" {
		title = m.cwd + "/"
	}
	return m.box(title, body, w, h, m.focus == paneList)
}

func (m *Model) notePane(w, h int) []string {
	vis := h - 2
	var body []string
	title := ""
	e, ok := m.selected()
	switch {
	case !ok:
		body = []string{"", m.st.muted.Render("  Nothing here yet.")}
	case e.IsDir:
		title = e.Name + "/"
		body = []string{
			"",
			"  " + m.st.dir.Render(e.Name+"/"),
			"  " + m.st.muted.Render(m.dirInfo),
			"",
			"  " + m.st.muted.Render("enter opens it"),
		}
	case m.noteErr != nil:
		title = e.Name
		body = []string{"", "  " + m.st.errText.Render("Can't read this note: "+m.noteErr.Error())}
	case !m.isNote:
		title = e.Name
		body = []string{"", "  " + m.st.muted.Render("Not a markdown note, so there's nothing to preview.")}
	default:
		title = strings.TrimSuffix(e.Name, path.Ext(e.Name))
		for i := m.noteOff; i < min(len(m.lines), m.noteOff+vis); i++ {
			body = append(body, " "+m.lines[i].Text)
		}
	}
	return m.box(title, body, w, h, m.focus == paneNote)
}

func (m *Model) statusLine() string {
	left := m.st.pill.Render(" VIEW ") + " " + m.st.text.Render(m.location())
	if m.isNote && len(m.lines) > 0 {
		left += m.st.muted.Render("  " + m.scrollInfo())
	}
	right := m.st.muted.Render("h/l panes · enter open · q quit")
	if m.flash != "" {
		right = m.st.flash.Render(m.flash)
	}
	return spread(left, right, m.width)
}

func (m *Model) location() string {
	switch {
	case m.notePath != "":
		return m.notePath
	case m.cwd != "":
		return m.cwd + "/"
	}
	return m.vault.Name() + "/"
}

// scrollInfo describes the note's scroll position the way vim does.
func (m *Model) scrollInfo() string {
	vis := m.layout().bodyH - 2
	maxOff := len(m.lines) - vis
	switch {
	case maxOff <= 0:
		return "All"
	case m.noteOff == 0:
		return "Top"
	case m.noteOff >= maxOff:
		return "Bot"
	}
	return fmt.Sprintf("%d%%", 100*m.noteOff/maxOff)
}

// box frames body lines in a rounded border of exactly w×h cells, with the
// title set into the top edge.
func (m *Model) box(title string, body []string, w, h int, focused bool) []string {
	bc, tc := m.st.border, m.st.title
	if focused {
		bc, tc = m.st.borderFocus, m.st.titleFocus
	}
	inner := max(w-2, 1)
	t := ""
	if title != "" {
		t = ansi.Truncate(" "+title+" ", inner-1, "…")
	}
	out := make([]string, 0, h)
	out = append(out, bc.Render("╭─")+tc.Render(t)+bc.Render(strings.Repeat("─", max(inner-1-ansi.StringWidth(t), 0))+"╮"))
	for i := 0; i < h-2; i++ {
		s := ""
		if i < len(body) {
			s = body[i]
		}
		out = append(out, bc.Render("│")+fit(s, inner)+bc.Render("│"))
	}
	out = append(out, bc.Render("╰"+strings.Repeat("─", inner)+"╯"))
	return out
}

// fit truncates or pads s to exactly w cells.
func fit(s string, w int) string {
	s = ansi.Truncate(s, w, "…")
	if pad := w - ansi.StringWidth(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

// spread puts left and right on one line of exactly w cells.
func spread(left, right string, w int) string {
	gap := w - ansi.StringWidth(left) - ansi.StringWidth(right) - 1
	if gap < 1 {
		return fit(left, w)
	}
	return left + strings.Repeat(" ", gap) + right + " "
}

func shortDate(t, now time.Time) string {
	switch {
	case t.IsZero():
		return ""
	case t.Year() == now.Year() && t.YearDay() == now.YearDay():
		return t.Format("15:04")
	case t.Year() == now.Year():
		return t.Format("02 Jan")
	}
	return t.Format("2006")
}

func tildePath(p string) string {
	if home, err := os.UserHomeDir(); err == nil {
		if rest, ok := strings.CutPrefix(p, home); ok {
			return "~" + rest
		}
	}
	return p
}
