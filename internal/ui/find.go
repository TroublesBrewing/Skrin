package ui

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/search"
)

// resultRow is one row of a result list: a note's title row (hit -1), one
// of its hits, or (note -1) the gap between two notes.
type resultRow struct{ note, hit int }

type noteHits struct {
	rel  string
	hits []search.Hit
}

// searchPanel is /: search that updates as you type. Alt-r turns it into
// search & replace, which looks for the typed text as plain text.
type searchPanel struct {
	in, with  lineInput
	replacing bool
	focus     int    // 0 the query, 1 the replacement (when replacing), 2 the results
	inNote    bool   // look only in rel
	rel       string // the note selected when / was pressed, if any
	matchCase bool
	wholeWord bool // replace mode only

	query   bool       // there is something to look for
	results []noteHits // search mode
	notes   []replNote // replace mode
	rows    []resultRow
	cur     int
	hits    int
}

func (m *Model) openSearch() {
	p := &searchPanel{}
	if rel, ok := m.subject(); ok {
		p.rel = rel
	}
	last := m.lastSearch
	if last != nil {
		p.in.set(last.in.value())
		p.with.set(last.with.value())
		p.replacing, p.matchCase, p.wholeWord = last.replacing, last.matchCase, last.wholeWord
		p.inNote = last.inNote && p.rel != ""
	}
	m.search = p
	m.runSearch()
	if last != nil {
		p.cur = min(last.cur, max(len(p.rows)-1, 0))
		p.move(0)
	}
}

// scope lists the notes to look in.
func (m *Model) scope(p *searchPanel) []string {
	if p.inNote && p.rel != "" {
		return []string{p.rel}
	}
	return m.idx.Notes()
}

func (m *Model) runSearch() {
	p := m.search
	p.results, p.notes, p.rows, p.cur, p.hits = nil, nil, nil, 0, 0
	if p.replacing {
		m.findForReplace(p)
		return
	}
	q := search.Parse(p.in.value(), p.matchCase)
	p.query = !q.Empty()
	if !p.query {
		return
	}
	for _, rel := range m.scope(p) {
		d, _ := m.idx.Doc(rel)
		ok, hits := q.Match(d)
		if !ok {
			continue
		}
		p.addGroup(len(p.results), len(hits))
		p.results = append(p.results, noteHits{rel, hits})
		p.hits += len(hits)
	}
}

// addGroup adds one note's rows: a gap before every note but the first,
// its title row, and a row per hit.
func (p *searchPanel) addGroup(note, hits int) {
	if len(p.rows) > 0 {
		p.rows = append(p.rows, resultRow{-1, -1})
	}
	p.rows = append(p.rows, resultRow{note, -1})
	for h := 0; h < hits; h++ {
		p.rows = append(p.rows, resultRow{note, h})
	}
}

// move steps the cursor d rows, never resting on a gap between notes.
func (p *searchPanel) move(d int) {
	last := max(len(p.rows)-1, 0)
	c := clamp(p.cur+d, 0, last)
	if c < len(p.rows) && p.rows[c].note < 0 {
		if d < 0 {
			c--
		} else {
			c++
		}
	}
	p.cur = clamp(c, 0, last)
}

// fields are what Tab cycles through.
func (p *searchPanel) nextField(d int) int {
	fields := []int{0, 2}
	if p.replacing {
		fields = []int{0, 1, 2}
	}
	i := max(slices.Index(fields, p.focus), 0)
	return fields[(i+d+len(fields))%len(fields)]
}

func (m *Model) searchKey(k tea.KeyPressMsg) {
	p := m.search
	s := k.String()
	switch s {
	case "esc", "ctrl+c":
		m.lastSearch, m.search = p, nil
		return
	case "alt+r":
		p.replacing = !p.replacing
		if !p.replacing && p.focus == 1 {
			p.focus = 0
		}
		m.runSearch()
		return
	case "alt+c":
		p.matchCase = !p.matchCase
		m.runSearch()
		return
	case "alt+w":
		if p.replacing {
			p.wholeWord = !p.wholeWord
			m.runSearch()
		}
		return
	case "alt+t":
		if p.rel == "" {
			m.flash = "Select a note first to look in just that one"
			return
		}
		p.inNote = !p.inNote
		m.runSearch()
		return
	case "tab":
		p.focus = p.nextField(1)
		return
	case "shift+tab":
		p.focus = p.nextField(-1)
		return
	case "up", "ctrl+p":
		p.move(-1)
		return
	case "down", "ctrl+n":
		p.move(1)
		return
	case "pgup":
		p.move(-10)
		return
	case "pgdown":
		p.move(10)
		return
	case "ctrl+s":
		if p.replacing {
			m.askReplace()
		}
		return
	}
	switch p.focus {
	case 0:
		switch {
		case s == "enter" && p.replacing:
			p.focus = 1
		case s == "enter":
			m.openSelected()
		case p.in.handle(k):
			m.runSearch()
		}
	case 1:
		if s == "enter" {
			p.focus = 2
		} else {
			p.with.handle(k)
		}
	case 2:
		switch s {
		case "j":
			p.move(1)
		case "k":
			p.move(-1)
		case "space", " ":
			if p.replacing {
				p.toggleSkip()
			}
		case "enter":
			m.openSelected()
		}
	}
}

func (p *searchPanel) paste(s string) {
	if p.focus == 1 {
		p.with.insert(s)
	} else {
		p.in.insert(s)
	}
}

// openSelected closes the panel and opens the selected note, at the
// selected line.
func (m *Model) openSelected() {
	p := m.search
	if p.cur >= len(p.rows) || p.rows[p.cur].note < 0 {
		return
	}
	r := p.rows[p.cur]
	var rel string
	line := -1
	if p.replacing {
		n := p.notes[r.note]
		rel = n.rel
		if r.hit >= 0 {
			line = n.matches[r.hit].line
		}
	} else {
		nh := p.results[r.note]
		rel = nh.rel
		if r.hit >= 0 {
			line = nh.hits[r.hit].Line
		}
	}
	m.lastSearch, m.search = p, nil
	if line < 0 {
		m.goTo(rel, "")
	} else {
		m.goToLine(rel, line)
	}
}

func (m *Model) searchBox() []string {
	p := m.search
	w, h := min(m.width-4, 110), max(m.height-4, 10)
	inner := w - 2
	field := func(label string, in *lineInput, focused bool) string {
		cursor := m.st.text
		if focused {
			cursor = m.st.cursor
		}
		return " " + m.st.muted.Render(label) + in.view(m.st.text, cursor)
	}
	title, footer := "Search", "↑↓ choose · enter open · tab results · alt+r replace · esc close"
	var body []string
	if p.replacing {
		title, footer = "Search & replace", "tab next field · space skip · ctrl+s replace · alt+r back to search · esc close"
		body = append(body, field("find     ▸ ", &p.in, p.focus == 0), field("replace  ▸ ", &p.with, p.focus == 1))
	} else {
		body = append(body, field("search ▸ ", &p.in, p.focus == 0))
	}
	body = append(body, m.searchOptions(p, inner), "")
	body = append(body, m.windowRows(len(p.rows), p.cur, h-len(body)-4, inner, p.focus == 2 || !p.replacing, func(i int) string {
		return m.resultText(p, i, inner)
	})...)
	if p.query && len(p.rows) == 0 {
		body = append(body, m.st.muted.Render("  Nothing matches"))
	}
	for len(body) < h-3 {
		body = append(body, "")
	}
	body = append(body, " "+m.st.muted.Render(footer))
	return m.box(title, body, w, h, true)
}

func (m *Model) searchOptions(p *searchPanel, w int) string {
	scope := m.toggle("whole vault", true)
	if p.rel != "" {
		scope = m.toggle(displayName(p.rel), p.inNote) + m.toggle("whole vault", !p.inNote) + m.st.muted.Render(" alt+t")
	}
	opts := " " + scope + "   " + m.toggle("Aa", p.matchCase) + m.st.muted.Render(" alt+c")
	if p.replacing {
		opts += "   " + m.toggle("whole words", p.wholeWord) + m.st.muted.Render(" alt+w")
	}
	opts += "   " + m.toggle("replace", p.replacing) + m.st.muted.Render(" alt+r")
	return spread(opts, m.st.muted.Render(m.searchSummary(p)), w)
}

func (m *Model) searchSummary(p *searchPanel) string {
	switch {
	case p.replacing && !p.query:
		return "plain text"
	case p.replacing:
		n, notes := p.active()
		return plural2(n, "match", "matches") + " in " + plural(notes, "note")
	case !p.query:
		return `#tag [prop:value] path: file: "phrase" -not OR`
	}
	return plural2(p.hits, "match", "matches") + " in " + plural(len(p.results), "note")
}

func (m *Model) resultText(p *searchPanel, i, w int) string {
	r := p.rows[i]
	switch {
	case r.note < 0:
		return ""
	case p.replacing:
		return m.replaceRow(p, r, w)
	}
	nh := p.results[r.note]
	if r.hit < 0 {
		count := ""
		if n := len(nh.hits); n > 0 {
			count = plural2(n, "match", "matches")
		}
		return m.groupHeader(nh.rel, count, w, false)
	}
	h := nh.hits[r.hit]
	return m.hitPrefix(h.Line) + snippet(m.idx.Line(nh.rel, h.Line), h.Spans, w-hitPrefixW, m.st.text, func(s string) string { return m.st.found.Render(s) })
}

const hitPrefixW = 10

// hitPrefix is the line-number gutter in front of a hit.
func (m *Model) hitPrefix(line int) string {
	return m.st.muted.Render(fmt.Sprintf("%7d │ ", line+1))
}

// groupHeader is a note's title row in a result list: a bar across the
// list, so each note's results stand apart from the next.
func (m *Model) groupHeader(rel, right string, w int, skipped bool) string {
	bar := lipgloss.NewStyle().Background(m.pal.LighterBackground)
	name := bar.Foreground(m.pal.Accent).Bold(true)
	if skipped {
		name = bar.Foreground(m.pal.DarkForeground).Strikethrough(true)
	}
	muted := bar.Foreground(m.pal.DarkForeground)
	left := bar.Render(" ") + name.Render(displayName(rel)) + muted.Render("  "+m.folderLabel(parentOf(rel)))
	rt := muted.Render(right + " ")
	gap := w - ansi.StringWidth(left) - ansi.StringWidth(rt)
	if gap < 1 {
		return fit(left, w)
	}
	return left + bar.Render(strings.Repeat(" ", gap)) + rt
}

// windowRows renders the rows around the cursor that fit in vis lines.
func (m *Model) windowRows(total, cur, vis, width int, focused bool, row func(int) string) []string {
	if vis <= 0 {
		return nil
	}
	off := clamp(cur-vis/2, 0, max(total-vis, 0))
	var out []string
	for i := off; i < min(total, off+vis); i++ {
		s := row(i)
		if i == cur && focused {
			s = m.st.selFocus.Render(fit(ansi.Strip(s), width))
		} else {
			s = fit(s, width)
		}
		out = append(out, s)
	}
	return out
}

// snippet shows a line with its spans drawn by hl, starting a little before
// the first span so it's in view. Tabs become spaces.
func snippet(line string, spans [][2]int, width int, base lipgloss.Style, hl func(string) string) string {
	indent := len(line) - len(strings.TrimLeft(line, " \t"))
	start := indent
	if len(spans) > 0 && spans[0][0] > start {
		cut := spans[0][0]
		for n := 0; n < 20 && cut > start; n++ {
			_, size := utf8.DecodeLastRuneInString(line[:cut])
			cut -= size
		}
		start = cut
	}
	clean := func(s string) string { return strings.ReplaceAll(s, "\t", " ") }
	var b strings.Builder
	if start > indent {
		b.WriteString(base.Render("…"))
	}
	pos := start
	for _, s := range spans {
		if s[1] <= pos || s[0] > len(line) {
			continue
		}
		a := max(s[0], pos)
		b.WriteString(base.Render(clean(line[pos:a])))
		b.WriteString(hl(clean(line[a:min(s[1], len(line))])))
		pos = min(s[1], len(line))
	}
	b.WriteString(base.Render(clean(line[pos:])))
	return ansi.Truncate(b.String(), max(width, 1), "…")
}

// toggle draws an option that is on (a filled pill) or off.
func (m *Model) toggle(label string, on bool) string {
	if on {
		return m.st.pill.Render(" " + label + " ")
	}
	return m.st.muted.Render(" " + label + " ")
}

// openSwitcher is Go to note: jump to a note by name or alias, or create
// one. The first row is the note already open (or an empty row when none
// is), so Enter straight away just closes the list.
func (m *Model) openSwitcher() {
	c := &chooser{title: "Go to note", prompt: "name", empty: "No note by that name · enter creates it", verb: "open · shift+←/→ split"}
	open := ""
	if m.notePath != "" {
		open = m.notePath
		c.items = append(c.items, choice{label: displayName(open), detail: "open now", rel: open})
	} else {
		c.items = append(c.items, choice{detail: "stay here"})
	}
	for _, rel := range m.idx.Notes() {
		if rel != open {
			c.items = append(c.items, choice{label: displayName(rel), detail: parentOf(rel), rel: rel, do: func() { m.goTo(rel, "") }})
		}
		for _, a := range m.idx.Aliases(rel) {
			c.items = append(c.items, choice{label: a, detail: "alias of " + displayName(rel), rel: rel, do: func() { m.goTo(rel, "") }})
		}
	}
	c.none = m.offerCreate
	c.split = m.openSplit
	m.openChooser(c)
}

func plural2(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}
