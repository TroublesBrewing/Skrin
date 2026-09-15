package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lurioso/skrin/internal/search"
	"github.com/lurioso/skrin/internal/vault"
)

// replNote is one note in the replace preview.
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

// findForReplace builds the replace preview from the index's copy of each
// note in scope.
func (m *Model) findForReplace(p *searchPanel) {
	needle := p.in.value()
	p.query = needle != ""
	if !p.query {
		return
	}
	for _, rel := range m.scope(p) {
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
		p.addGroup(len(p.notes), len(n.matches))
		p.notes = append(p.notes, n)
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
func (p *searchPanel) active() (matches, notes int) {
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

// toggleSkip skips (or includes again) the match, or the whole note on
// its title row, and moves down.
func (p *searchPanel) toggleSkip() {
	if p.cur >= len(p.rows) || p.rows[p.cur].note < 0 {
		return
	}
	r := p.rows[p.cur]
	n := &p.notes[r.note]
	if r.hit < 0 {
		n.skip = !n.skip
	} else {
		n.matches[r.hit].skip = !n.matches[r.hit].skip
	}
	p.move(1)
}

func (m *Model) askReplace() {
	p := m.search
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
func (m *Model) applyReplace(p *searchPanel) {
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
	m.journal.Record(vault.Op{Desc: fmt.Sprintf("replace %q with %q in %s", p.in.value(), with, plural(notes, "note")), Steps: steps})
	m.lastSearch, m.search = p, nil
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

func (m *Model) replaceRow(p *searchPanel, r resultRow, w int) string {
	n := p.notes[r.note]
	if r.hit < 0 {
		kept := 0
		for _, mt := range n.matches {
			if !mt.skip {
				kept++
			}
		}
		right := fmt.Sprintf("%d of %d", kept, len(n.matches))
		if n.skip {
			right = "skipped"
		}
		return m.groupHeader(n.rel, right, w, n.skip)
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
	return m.hitPrefix(mt.line) +
		snippet(line, [][2]int{{mt.start - ls, min(mt.end-ls, len(line))}}, w-hitPrefixW-9, m.st.text, hl) + tail
}
