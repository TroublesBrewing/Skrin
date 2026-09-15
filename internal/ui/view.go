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
	"github.com/lurioso/skrin/internal/version"
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
	out := strings.Join(rows[:min(len(rows), m.height)], "\n")
	switch {
	case m.chooser != nil:
		out = m.overlay(out, m.chooserBox())
	case m.complete != nil && m.editor != nil:
		box, x, y := m.completionBox()
		out = m.overlayAt(out, box, x, y)
	}
	return out
}

func (m *Model) header() []string {
	left := []string{
		m.st.brand.Render("Skrin") + m.st.muted.Render(" v"+version.Version+" · ") + m.st.bold.Render(m.vault.Name()),
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

// markCell is the one-cell column in front of a row that shows a mark.
func (m *Model) markCell(rel string) string {
	if m.marks[rel] {
		return "●"
	}
	return " "
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
		text := fit(m.markCell(r.path)+strings.Repeat("  ", r.depth)+icon+r.name, inner)
		st := m.st.text
		switch {
		case r.path == "":
			st = m.st.bold
		case m.marks[r.path]:
			st = m.st.marked
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
		if m.marks[e.Rel] {
			st = m.st.marked
		}
		left := fit(m.markCell(e.Rel)+name, max(inner-8, 1))
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
	return m.box(m.folderLabel(m.cwd), body, w, h, m.focus == paneList)
}

func (m *Model) notePane(w, h int) []string {
	if m.editor != nil {
		return m.editorPane(w, h)
	}
	vis := h - 2
	var body []string
	title := ""
	e, ok := m.selected()
	switch {
	case !ok:
		body = []string{"", m.st.muted.Render("  Nothing here yet. n creates a note.")}
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
			line := " " + m.lines[i].Text
			if m.hints != nil {
				for _, ht := range m.hints.hints {
					if ht.row == i {
						line = m.withLabel(line, 1+ht.link.Col, ht.label)
					}
				}
			}
			body = append(body, line)
		}
	}
	return m.box(title, body, w, h, m.focus == paneNote)
}

func (m *Model) statusLine() string {
	switch {
	case m.conflict != nil:
		return m.conflictLine()
	case m.editor != nil:
		return m.editLine()
	case m.hints != nil:
		return spread(m.st.pill.Render(" FOLLOW ")+" "+m.st.text.Render("Type the letters on a link"), m.st.muted.Render("esc cancel"), m.width)
	case m.prompt != nil:
		return m.promptLine()
	case m.confirm != nil:
		return m.confirmLine()
	}
	mode := " VIEW "
	if m.visual != nil {
		mode = " VISUAL "
	}
	left := m.st.pill.Render(mode) + " " + m.st.text.Render(m.location())
	if n := len(m.marks); n > 0 {
		left += m.st.marked.Render(fmt.Sprintf("  ● %d marked", n))
	}
	if m.isNote && len(m.lines) > 0 {
		left += m.st.muted.Render("  " + m.scrollInfo())
	}
	right := m.st.muted.Render("e edit · f follow · b backlinks · o outline · q quit")
	if m.flash != "" {
		right = m.st.flash.Render(m.flash)
	}
	return spread(left, right, m.width)
}

func (m *Model) location() string {
	if m.notePath != "" {
		return m.notePath
	}
	return m.folderLabel(m.cwd)
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

// spread puts left and right on one line of exactly w cells. When both
// don't fit, the right side keeps up to two thirds of the line and the left
// side is clipped.
func spread(left, right string, w int) string {
	lw, rw := ansi.StringWidth(left), ansi.StringWidth(right)
	if lw+rw+2 > w {
		if rw > w*2/3 {
			right = ansi.Truncate(right, w*2/3, "…")
			rw = ansi.StringWidth(right)
		}
		left = ansi.Truncate(left, max(w-rw-2, 0), "…")
		lw = ansi.StringWidth(left)
	}
	gap := w - lw - rw - 1
	if gap < 1 {
		return fit(left+" "+right, w)
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
