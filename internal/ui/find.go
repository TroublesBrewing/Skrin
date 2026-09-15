package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/search"
)

// resultRow is one row of a result list: a note's title row (hit -1), or
// one of its hits.
type resultRow struct{ note, hit int }

type noteHits struct {
	rel  string
	hits []search.Hit
}

// searchPanel is the / vault search. Results update as you type.
type searchPanel struct {
	in        lineInput
	matchCase bool
	query     bool // the query has something to look for
	results   []noteHits
	rows      []resultRow
	cur       int
	hits      int
}

func (m *Model) openSearch() {
	p := &searchPanel{}
	last := m.lastSearch
	if last != nil {
		p.in.set(last.in.value())
		p.matchCase = last.matchCase
	}
	m.search = p
	m.runSearch()
	if last != nil {
		p.cur = min(last.cur, max(len(p.rows)-1, 0))
	}
}

func (m *Model) runSearch() {
	p := m.search
	p.results, p.rows, p.cur, p.hits = nil, nil, 0, 0
	q := search.Parse(p.in.value(), p.matchCase)
	p.query = !q.Empty()
	if !p.query {
		return
	}
	for _, rel := range m.idx.Notes() {
		d, _ := m.idx.Doc(rel)
		ok, hits := q.Match(d)
		if !ok {
			continue
		}
		n := len(p.results)
		p.results = append(p.results, noteHits{rel, hits})
		p.rows = append(p.rows, resultRow{n, -1})
		for h := range hits {
			p.rows = append(p.rows, resultRow{n, h})
		}
		p.hits += len(hits)
	}
}

func (m *Model) searchKey(k tea.KeyPressMsg) {
	p := m.search
	last := max(len(p.rows)-1, 0)
	switch k.String() {
	case "esc", "ctrl+c":
		m.lastSearch, m.search = p, nil
	case "enter":
		if p.cur < len(p.rows) {
			m.lastSearch, m.search = p, nil
			r := p.rows[p.cur]
			nh := p.results[r.note]
			if r.hit < 0 {
				m.goTo(nh.rel, "")
			} else {
				m.goToLine(nh.rel, nh.hits[r.hit].Line)
			}
		}
	case "up", "ctrl+p", "shift+tab":
		p.cur = max(p.cur-1, 0)
	case "down", "ctrl+n", "tab":
		p.cur = min(p.cur+1, last)
	case "pgup":
		p.cur = max(p.cur-10, 0)
	case "pgdown":
		p.cur = min(p.cur+10, last)
	case "alt+c":
		p.matchCase = !p.matchCase
		m.runSearch()
	default:
		if p.in.handle(k) {
			m.runSearch()
		}
	}
}

func (m *Model) searchBox() []string {
	p := m.search
	w, h := min(m.width-4, 110), max(m.height-4, 8)
	inner := w - 2
	summary := `#tag  [property:value]  path:  file:  "phrase"  -not  OR`
	if p.query {
		summary = plural2(p.hits, "match", "matches") + " in " + plural(len(p.results), "note")
	}
	right := m.toggle("Aa", p.matchCase) + "  " + m.st.muted.Render(summary)
	body := []string{spread(" "+m.st.muted.Render("search ▸ ")+p.in.view(m.st.text, m.st.cursor), right, inner), ""}
	body = append(body, m.windowRows(len(p.rows), p.cur, h-len(body)-4, inner, true, func(i int) string {
		return m.searchRow(p, i, inner)
	})...)
	if p.query && len(p.rows) == 0 {
		body = append(body, m.st.muted.Render("  No notes match"))
	}
	for len(body) < h-3 {
		body = append(body, "")
	}
	body = append(body, " "+m.st.muted.Render("↑↓ choose · enter open · alt+c match case · esc close"))
	return m.box("Search", body, w, h, true)
}

func (m *Model) searchRow(p *searchPanel, i, w int) string {
	r := p.rows[i]
	nh := p.results[r.note]
	if r.hit < 0 {
		count := ""
		if len(nh.hits) > 0 {
			count = fmt.Sprint(len(nh.hits))
		}
		return spread("▸ "+m.st.bold.Render(displayName(nh.rel))+"  "+m.st.muted.Render(parentOf(nh.rel)), m.st.muted.Render(count), w)
	}
	h := nh.hits[r.hit]
	return m.st.muted.Render(fmt.Sprintf("%6d  ", h.Line+1)) +
		snippet(m.idx.Line(nh.rel, h.Line), h.Spans, w-8, m.st.text, func(s string) string { return m.st.found.Render(s) })
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

// openSwitcher is Ctrl-p: jump to a note by name or alias, or create one.
func (m *Model) openSwitcher() {
	c := &chooser{title: "Go to note", prompt: "name", empty: "No note by that name · enter creates it", verb: "open"}
	for _, rel := range m.idx.Notes() {
		c.items = append(c.items, choice{label: displayName(rel), detail: parentOf(rel), do: func() { m.goTo(rel, "") }})
		for _, a := range m.idx.Aliases(rel) {
			c.items = append(c.items, choice{label: a, detail: "alias of " + displayName(rel), do: func() { m.goTo(rel, "") }})
		}
	}
	c.none = m.offerCreate
	m.openChooser(c)
}

func plural2(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}
