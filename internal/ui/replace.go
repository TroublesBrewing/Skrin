package ui

import (
	"fmt"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/search"
	"github.com/lurioso/skrin/internal/vault"
)

// replacePanel is R: find and replace in one note or the whole vault, with
// a live preview.
type replacePanel struct {
	find, with           lineInput
	focus                int // 0 find, 1 replace with, 2 the list
	vault                bool
	rel                  string // the note for "this note"; "" when none is selected
	matchCase, wholeWord bool
	notes                []replNote
	rows                 []resultRow
	cur                  int
}

type replNote struct {
	rel     string
	content string // the note as the preview saw it
	matches []replMatch
	skip    bool
}

type replMatch struct {
	start, end int // byte offsets in content
	line       int
	inLink     bool
	skip       bool
}

var wikiLinkRE = regexp.MustCompile(`\[\[[^\[\]]*\]\]`)

func (m *Model) openReplace() {
	p := &replacePanel{vault: true}
	if rel, ok := m.editTarget(); ok {
		p.rel, p.vault = rel, false
	}
	if last := m.lastReplace; last != nil {
		p.find.set(last.find.value())
		p.with.set(last.with.value())
		p.matchCase, p.wholeWord = last.matchCase, last.wholeWord
	}
	m.replace = p
	m.runReplace()
}

// runReplace rebuilds the preview from the index's copy of each note.
func (m *Model) runReplace() {
	p := m.replace
	p.notes, p.rows, p.cur = nil, nil, 0
	needle := p.find.value()
	if needle == "" {
		return
	}
	rels := m.idx.Notes()
	if !p.vault {
		rels = []string{p.rel}
	}
	for _, rel := range rels {
		content, ok := m.idx.Content(rel)
		if !ok {
			continue
		}
		spans := search.Find(content, needle, p.matchCase, p.wholeWord)
		if len(spans) == 0 {
			continue
		}
		links := wikiLinkRE.FindAllStringIndex(content, -1)
		n := replNote{rel: rel, content: content}
		line, pos := 0, 0
		for _, s := range spans {
			line += strings.Count(content[pos:s[0]], "\n")
			pos = s[0]
			n.matches = append(n.matches, replMatch{start: s[0], end: s[1], line: line, inLink: overlaps(links, s)})
		}
		ni := len(p.notes)
		p.notes = append(p.notes, n)
		p.rows = append(p.rows, resultRow{ni, -1})
		for mi := range n.matches {
			p.rows = append(p.rows, resultRow{ni, mi})
		}
	}
}

func overlaps(spans [][]int, s [2]int) bool {
	for _, l := range spans {
		if s[0] < l[1] && s[1] > l[0] {
			return true
		}
	}
	return false
}

// active counts the matches, and the notes, a replace would change.
func (p *replacePanel) active() (matches, notes int) {
	for _, n := range p.notes {
		if n.skip {
			continue
		}
		c := 0
		for _, mt := range n.matches {
			if !mt.skip {
				c++
			}
		}
		if c > 0 {
			matches += c
			notes++
		}
	}
	return matches, notes
}

func (m *Model) replaceKey(k tea.KeyPressMsg) {
	p := m.replace
	s := k.String()
	last := max(len(p.rows)-1, 0)
	switch s {
	case "esc", "ctrl+c":
		m.lastReplace, m.replace = p, nil
		return
	case "tab":
		p.focus = (p.focus + 1) % 3
		return
	case "shift+tab":
		p.focus = (p.focus + 2) % 3
		return
	case "alt+c":
		p.matchCase = !p.matchCase
		m.runReplace()
		return
	case "alt+w":
		p.wholeWord = !p.wholeWord
		m.runReplace()
		return
	case "ctrl+t":
		if p.rel == "" {
			m.flash = "Select a note to replace in just that one"
			return
		}
		p.vault = !p.vault
		m.runReplace()
		return
	case "ctrl+s":
		m.askReplace()
		return
	case "up", "ctrl+p":
		p.cur = max(p.cur-1, 0)
		return
	case "down", "ctrl+n":
		p.cur = min(p.cur+1, last)
		return
	}
	switch p.focus {
	case 0:
		if s == "enter" {
			p.focus = 1
		} else if p.find.handle(k) {
			m.runReplace()
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
			p.cur = min(p.cur+1, last)
		case "k":
			p.cur = max(p.cur-1, 0)
		case "space", " ":
			p.toggleSkip()
		case "enter":
			if p.cur < len(p.rows) {
				r := p.rows[p.cur]
				n := p.notes[r.note]
				line := 0
				if r.hit >= 0 {
					line = n.matches[r.hit].line
				}
				m.lastReplace, m.replace = p, nil
				m.goToLine(n.rel, line)
			}
		}
	}
}

// toggleSkip skips (or includes again) the match, or the whole note on
// its title row, and moves down.
func (p *replacePanel) toggleSkip() {
	if p.cur >= len(p.rows) {
		return
	}
	r := p.rows[p.cur]
	n := &p.notes[r.note]
	if r.hit < 0 {
		n.skip = !n.skip
	} else {
		n.matches[r.hit].skip = !n.matches[r.hit].skip
	}
	p.cur = min(p.cur+1, len(p.rows)-1)
}

func (p *replacePanel) paste(s string) {
	switch p.focus {
	case 0:
		p.find.insert(s)
	case 1:
		p.with.insert(s)
	}
}

func (m *Model) askReplace() {
	p := m.replace
	n, notes := p.active()
	if n == 0 {
		m.flash = "Nothing to replace"
		return
	}
	m.confirm = &confirm{
		pill:     " REPLACE ",
		question: fmt.Sprintf("Replace %s in %s with %q?", plural2(n, "match", "matches"), plural(notes, "note"), p.with.value()),
		keys:     "y/n",
		cancel:   "Nothing replaced",
		yes:      func() { m.applyReplace(p) },
	}
}

// applyReplace writes the replacements. A note that changed on disk since
// the preview is skipped, never overwritten. Each changed note is
// snapshotted, and the whole replace is one journal entry.
func (m *Model) applyReplace(p *replacePanel) {
	with := p.with.value()
	var steps []vault.Step
	var changed, failed []string
	notes, count := 0, 0
	for _, n := range p.notes {
		if n.skip {
			continue
		}
		var spans [][2]int
		for _, mt := range n.matches {
			if !mt.skip {
				spans = append(spans, [2]int{mt.start, mt.end})
			}
		}
		if len(spans) == 0 {
			continue
		}
		cur, err := m.vault.Read(n.rel)
		if err != nil || cur != n.content {
			changed = append(changed, displayName(n.rel))
			continue
		}
		_ = m.snaps.Save(n.rel, cur) // snapshots only help u
		if err := m.vault.Write(n.rel, search.Replace(cur, spans, with)); err != nil {
			failed = append(failed, displayName(n.rel)+": "+err.Error())
			continue
		}
		steps = append(steps, vault.Step{Kind: vault.StepModified, Rel: n.rel, Content: cur})
		notes++
		count += len(spans)
	}
	m.journal.Record(vault.Op{Desc: fmt.Sprintf("replace %q with %q in %s", p.find.value(), with, plural(notes, "note")), Steps: steps})
	m.lastReplace, m.replace = p, nil
	if err := m.reload(); err != nil {
		m.flash = err.Error()
		return
	}
	m.flash = fmt.Sprintf("Replaced %s in %s · U undoes", plural2(count, "match", "matches"), plural(notes, "note"))
	if len(changed) > 0 {
		m.flash += " · skipped " + strings.Join(changed, ", ") + " (changed on disk since the preview)"
	}
	if len(failed) > 0 {
		m.flash += " · failed: " + strings.Join(failed, "; ")
	}
}

func (m *Model) replaceBox() []string {
	p := m.replace
	w, h := min(m.width-4, 110), max(m.height-4, 10)
	inner := w - 2
	field := func(label string, in *lineInput, focused bool) string {
		cursor := m.st.text
		if focused {
			cursor = m.st.cursor
		}
		return " " + m.st.muted.Render(label) + in.view(m.st.text, cursor)
	}
	scope := m.toggle("whole vault", true)
	if p.rel != "" {
		scope = m.toggle(displayName(p.rel), !p.vault) + m.toggle("whole vault", p.vault) + m.st.muted.Render(" ctrl+t")
	}
	opts := " " + m.st.muted.Render("in ") + scope + "   " +
		m.toggle("Aa", p.matchCase) + m.st.muted.Render(" alt+c") + "   " +
		m.toggle("whole words", p.wholeWord) + m.st.muted.Render(" alt+w")
	body := []string{
		field("find     ▸ ", &p.find, p.focus == 0),
		field("replace  ▸ ", &p.with, p.focus == 1),
		opts, "",
	}
	body = append(body, m.windowRows(len(p.rows), p.cur, h-len(body)-4, inner, p.focus == 2, func(i int) string {
		return m.replaceRow(p, i, inner)
	})...)
	if p.find.value() != "" && len(p.rows) == 0 {
		body = append(body, m.st.muted.Render("  No matches"))
	}
	for len(body) < h-3 {
		body = append(body, "")
	}
	n, notes := p.active()
	summary := plural2(n, "match", "matches") + " in " + plural(notes, "note")
	body = append(body, " "+m.st.muted.Render(summary+" · tab next · space skip · ctrl+s replace · esc close"))
	return m.box("Search & replace", body, w, h, true)
}

func (m *Model) replaceRow(p *replacePanel, i, w int) string {
	r := p.rows[i]
	n := p.notes[r.note]
	if r.hit < 0 {
		if n.skip {
			return "▸ " + m.st.muted.Render(displayName(n.rel)+"  (skipped)")
		}
		kept := 0
		for _, mt := range n.matches {
			if !mt.skip {
				kept++
			}
		}
		title := "▸ " + m.st.bold.Render(displayName(n.rel)) + "  " + m.st.muted.Render(parentOf(n.rel))
		return spread(title, m.st.muted.Render(fmt.Sprintf("%d of %d", kept, len(n.matches))), w)
	}
	mt := n.matches[r.hit]
	ls := strings.LastIndex(n.content[:mt.start], "\n") + 1
	le := len(n.content)
	if i := strings.IndexByte(n.content[mt.start:], '\n'); i >= 0 {
		le = mt.start + i
	}
	line := strings.TrimSuffix(n.content[ls:le], "\r")
	hl := func(old string) string {
		return m.st.diffDel.Strikethrough(true).Render(old) + m.st.muted.Render("→") + m.st.diffAdd.Render(p.with.value())
	}
	tail := ""
	switch {
	case mt.skip || n.skip:
		hl = func(old string) string { return m.st.found.Render(old) }
		tail = m.st.muted.Render(" skipped")
	case mt.inLink:
		tail = m.st.errText.Render(" ⚠ link")
	}
	return m.st.muted.Render(fmt.Sprintf("%6d  ", mt.line+1)) +
		snippet(line, [][2]int{{mt.start - ls, min(mt.end-ls, len(line))}}, w-8-9, m.st.text, hl) + tail
}
