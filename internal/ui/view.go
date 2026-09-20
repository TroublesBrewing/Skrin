package ui

import (
	"fmt"
	"os"
	"strconv"
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
	if m.manual != nil {
		return m.manualView()
	}
	out := m.panes()
	if m.zen {
		out = m.zenView()
	}
	if m.book != nil {
		out = m.overlay(out, m.bookCardBox())
	}
	if m.habits != nil {
		out = m.overlay(out, m.habitsBox())
	}
	if m.quickNote != nil {
		out = m.overlay(out, m.quickNoteBox())
	}
	if m.table != nil {
		out = m.overlay(out, m.tableBox())
	}
	switch {
	case m.chooser != nil:
		out = m.overlay(out, m.chooserBox())
	case m.search != nil:
		out = m.overlay(out, m.searchBox())
	case m.complete != nil && m.editor != nil:
		box, x, y := m.completionBox()
		out = m.overlayAt(out, box, x, y)
	}
	return out
}

// panes is the usual screen: the header, Files and the note, and the
// status line.
func (m *Model) panes() string {
	l := m.layout()
	rows := m.header()
	var cols [][]string
	if l.filesW > 0 {
		cols = append(cols, m.filesPane(l.filesW, l.bodyH))
	}
	switch {
	case m.split == nil:
		cols = append(cols, m.notePane(l.noteW, l.bodyH))
	case m.splitLeft:
		cols = append(cols, m.splitPane(l.splitW, l.bodyH), m.notePane(l.noteW, l.bodyH))
	default:
		cols = append(cols, m.notePane(l.noteW, l.bodyH), m.splitPane(l.splitW, l.bodyH))
	}
	if l.drawerW > 0 {
		cols = append(cols, m.drawerPane(l.drawerW, l.bodyH))
	}
	for i := 0; i < l.bodyH; i++ {
		var b strings.Builder
		for _, c := range cols {
			b.WriteString(c[i])
		}
		rows = append(rows, b.String())
	}
	switch {
	case l.drawerH == 1:
		rows = append(rows, m.drawerLine())
	case l.drawerH > 1:
		rows = append(rows, m.drawerPane(m.width, l.drawerH)...)
	}
	rows = append(rows, m.statusLine())
	return strings.Join(rows[:min(len(rows), m.height)], "\n")
}

// zenView is the note alone, centred at a readable width under its name.
// The bottom row shows the status line only when there's something to say
// or to answer.
func (m *Model) zenView() string {
	l := m.layout()
	vis := l.bodyH - 2
	title, body := m.noteBody(l.noteW-2, vis)
	margin := strings.Repeat(" ", max((m.width-l.noteTextW())/2-1, 0))
	if m.opts.LineNumbers {
		margin = strings.Repeat(" ", max((m.width-m.noteTextW()-m.gutterWidth())/2, 0))
	}
	rows := []string{fit(strings.Repeat(" ", max((m.width-ansi.StringWidth(title))/2, 0))+m.st.muted.Render(title), m.width)}
	for i := 0; i < vis; i++ {
		s := ""
		if i < len(body) {
			s = body[i]
		}
		rows = append(rows, fit(margin+s, m.width))
	}
	bottom := strings.Repeat(" ", m.width)
	if m.prompt != nil || m.confirm != nil || m.hints != nil || m.conflict != nil || m.flash != "" || len(m.proposals) > 0 {
		bottom = m.statusLine()
	}
	return strings.Join(append(rows, bottom), "\n")
}

func (m *Model) header() []string {
	left := []string{
		m.st.brand.Render("Skrin") + m.st.muted.Render(" v"+version.Version+" · ") + m.st.bold.Render(m.vault.Name()),
		m.st.muted.Render("a terminal home for your vault"),
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

// filesPane draws the tree. The open note is highlighted, as Obsidian
// highlights the active file; the cursor bar shows while Files has focus.
func (m *Model) filesPane(w, h int) []string {
	inner, vis := w-2, h-2
	f := &m.files
	var body []string
	for i := f.off; i < min(len(f.rows), f.off+vis); i++ {
		r := f.rows[i]
		name, icon, st := r.Name, "  ", m.st.text
		switch {
		case r.Rel == "":
			icon, st = "", m.st.bold
		case r.IsDir:
			icon, st = "▸ ", m.st.dir
			if f.expanded[r.Rel] {
				icon = "▾ "
			}
		case vault.IsNote(r.Name):
			name = displayName(r.Rel)
		default:
			st = m.st.muted
		}
		switch {
		case m.marks[r.Rel]:
			st = m.st.marked
		case r.Rel == m.notePath && r.Rel != "":
			st = m.st.open
		}
		text := fit(m.markCell(r.Rel)+strings.Repeat("  ", r.depth)+icon+name, inner)
		if i == f.cur && m.focus == paneFiles {
			text = m.st.selFocus.Render(text)
		} else {
			text = st.Render(text)
		}
		body = append(body, text)
	}
	return m.box("Files", body, w, h, m.focus == paneFiles)
}

func (m *Model) notePane(w, h int) []string {
	title, body := m.noteBody(w-2, h-2)
	if pad := m.noteMargin(); pad > 0 && (m.notePath != "" || m.editor != nil || len(m.proposals) > 0) {
		margin := strings.Repeat(" ", pad)
		for i := range body {
			body[i] = margin + body[i]
		}
	}
	return m.box(title, body, w, h, m.focus == paneNote || (m.editor != nil && m.focus != paneClaude))
}

// noteBody is what the note pane shows, in w cells by at most vis rows,
// and its title: the editor, the open note, or the splash when nothing is
// open.
func (m *Model) noteBody(w, vis int) (string, []string) {
	switch {
	case len(m.proposals) > 0:
		p := m.proposals[0]
		var body []string
		for i := p.off; i < min(len(p.diff), p.off+vis); i++ {
			body = append(body, " "+m.diffLine(p.diff[i]))
		}
		return p.title, body
	case m.editor != nil:
		return m.editorBody(vis)
	case m.notePath == "":
		return "", m.splashBody(w, vis)
	case m.noteErr != nil:
		return displayName(m.notePath), []string{"", "  " + m.st.errText.Render("Can't read this note: "+m.noteErr.Error())}
	}
	var body []string
	totalLines := m.totalSrcLines()
	digits := max(len(strconv.Itoa(totalLines)), 2)
	for i := m.noteOff; i < min(len(m.lines), m.noteOff+vis); i++ {
		line := m.lines[i].Text
		if m.noteSel.covers(i) {
			line = m.st.selFocus.Render(ansi.Strip(m.lines[i].Text))
		}
		var prefix string
		var prefixLen int
		if m.opts.LineNumbers {
			if i == 0 || m.lines[i].Src != m.lines[i-1].Src {
				prefix = m.st.muted.Render(fmt.Sprintf("%*d", digits, m.lines[i].Src+1)) + m.st.muted.Render(" │ ")
			} else {
				prefix = m.st.muted.Render(fmt.Sprintf("%*s", digits, "")) + m.st.muted.Render(" │ ")
			}
			prefixLen = digits + 3
		} else {
			prefix = " "
			prefixLen = 1
		}
		line = prefix + line
		if m.hints != nil {
			for _, ht := range m.hints.hints {
				if ht.row == i {
					line = m.withLabel(line, prefixLen+ht.link.Col, ht.label)
				}
			}
		}
		if f := m.noteFind; f != nil && f.inView && len(f.matches) > 0 {
			if mt := f.matches[f.cur]; mt.row == i {
				line = m.highlight(line, findMatch{i, mt.col + prefixLen, mt.end + prefixLen})
			}
		}
		body = append(body, line)
	}
	return displayName(m.notePath), body
}

func (m *Model) statusLine() string {
	switch {
	case m.conflict != nil:
		return m.conflictLine()
	case len(m.proposals) > 0:
		return m.proposalLine()
	case m.editor != nil:
		return m.editLine()
	case m.hints != nil:
		return spread(m.st.pill.Render(" FOLLOW ")+" "+m.st.text.Render("Type the letters on a link · Alt+ opens it beside"), m.st.muted.Render("esc cancel"), m.width)
	case m.prompt != nil:
		return m.promptLine()
	case m.confirm != nil:
		return m.confirmLine()
	}
	if m.noteFind != nil {
		return m.noteFindLine()
	}
	mode := " VIEW "
	switch {
	case m.visual != nil:
		mode = " MARK " // a range of items in Files
	case m.noteSel != nil:
		mode = " VISUAL " // selected text in the note
	}
	left := m.st.pill.Render(mode) + " " + m.st.text.Render(m.location())
	if sel := m.selectedNote(); sel != "" && m.noteSel != nil {
		left += m.st.marked.Render("  " + sel)
	}
	if n := len(m.marks); n > 0 {
		left += m.st.marked.Render(fmt.Sprintf("  ● %d marked", n))
	}
	if m.notePath != "" && len(m.lines) > 0 {
		left += m.st.muted.Render("  " + m.scrollInfo())
	}
	if w := m.wordCount(); w != "" {
		left += m.st.muted.Render("  " + w)
	}
	right := m.st.muted.Render(m.hintLine(m.width - min(ansi.StringWidth(left), m.width/2) - 3))
	if m.flash != "" {
		right = m.st.flash.Render(m.flash)
	}
	return spread(left, right, m.width)
}

// location is the open note's path and modified date, or the current
// folder when no note is open.
func (m *Model) location() string {
	if m.notePath == "" {
		return m.folderLabel(m.cwd())
	}
	s := m.notePath
	if e, ok := m.files.entry(m.notePath); ok && !e.ModTime.IsZero() {
		s += " · " + shortDate(e.ModTime, m.opts.Now())
	}
	if m.split != nil {
		s += " · split"
	}
	return s
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
