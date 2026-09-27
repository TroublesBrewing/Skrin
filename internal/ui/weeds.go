package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/frontmatter"
	"github.com/lurioso/skrin/internal/obsidian"
	"github.com/lurioso/skrin/internal/version"
	"github.com/lurioso/skrin/internal/weeds"
)

// Weeds is W: one page that answers "what needs tending?" about the whole
// vault. Everything it shows is invisible from inside a single note — a
// link whose note was renamed away, a thought that ended up filed under a
// tag only it uses, a note nothing points at any more — so it is found by
// accident, months later, if at all.
//
// It never changes anything by itself. Enter goes to the place, and the one
// fix it offers (n, the note a dead link wanted) is the same create that
// following that link would do, undoable with U like any other.

// weedRow is one line of the panel: a group's heading, or an item under it.
// Headings are rows so that scrolling and drawing stay one list; the cursor
// skips them.
type weedRow struct {
	head string      // the group's title, when this row is a heading
	hint string      // the group's own line under its title
	item *weeds.Item // nil on a heading
	last bool        // the group's last item, for the blank line after it
}

// weedsView is the open panel.
type weedsView struct {
	groups []weeds.Group
	rows   []weedRow
	cur    int // the cursor's row; always an item when there is one
	top    int // first row drawn
	total  int // loose ends found, capped groups included
}

// weedsOn is the panel's real state: a beta feature needs the build to
// allow beta, beta mode to be on, and its own switch on.
func (m *Model) weedsOn() bool {
	return version.Beta && m.opts.Beta && m.opts.Weeds
}

// openWeeds gathers the vault and opens the panel. The gather reads no
// files: everything comes from the index Skrin already keeps current, so
// this costs a walk over notes in memory rather than a disk pass.
func (m *Model) openWeeds() {
	if !m.weedsOn() {
		m.flash = "Weeds is off: it's a beta feature, switched on in Settings (" + note(m.keyFor(inMain, actHelp), "?") + " then tab)"
		return
	}
	groups := weeds.Find(m.gatherWeeds(), weeds.Options{
		Now:  m.opts.Now(),
		Skip: m.weedsSkip(),
	})
	v := &weedsView{groups: groups, total: weeds.Count(groups)}
	for _, g := range groups {
		title := fmt.Sprintf("%s (%d)", g.Title, g.Total)
		if len(g.Items) < g.Total {
			title = fmt.Sprintf("%s (%d, showing %d)", g.Title, g.Total, len(g.Items))
		}
		v.rows = append(v.rows, weedRow{head: title, hint: g.Hint})
		for i := range g.Items {
			v.rows = append(v.rows, weedRow{item: &g.Items[i], last: i == len(g.Items)-1})
		}
	}
	v.cur = v.nextItem(-1, 1)
	m.weeds = v
}

// weedsSkip are the folders left out of the judgements about notes as a
// whole. A daily note is meant to stand alone, so a year of days would
// otherwise bury everything worth seeing; a template is unlinked, thin and
// old on purpose, and it being all three is not a loose end. A dead link
// inside either is still reported: that one is broken wherever it sits.
func (m *Model) weedsSkip() []string {
	skip := []string{obsidian.LoadSettings(m.vault.Root).Daily.Folder}
	if f, _ := m.templatesFolder(); f != "" {
		skip = append(skip, f)
	}
	return skip
}

// gatherWeeds turns the index into what the checks read. The backlink
// count is done here in one pass over every link in the vault, rather than
// asking the index per note, which would walk them all again each time.
func (m *Model) gatherWeeds() []weeds.Note {
	rels := m.idx.Notes()
	linked := map[string]int{}
	links := make(map[string][]weeds.Link, len(rels))
	for _, rel := range rels {
		for _, l := range m.idx.Links(rel) {
			target, ok := m.idx.ResolveLink(l, rel)
			if ok && target != rel {
				linked[target]++
			}
			links[rel] = append(links[rel], weeds.Link{
				Target: l.Target,
				Line:   l.Line,
				Dead:   !ok,
				Embed:  l.Embed,
			})
		}
	}
	out := make([]weeds.Note, 0, len(rels))
	for _, rel := range rels {
		mod, _, _ := m.idx.Stat(rel)
		src, _ := m.idx.Content(rel)
		out = append(out, weeds.Note{
			Rel:      rel,
			Body:     noteBody(src),
			Modified: mod,
			Links:    links[rel],
			Linked:   linked[rel],
			Tags:     m.idx.WrittenTags(rel),
		})
	}
	return out
}

// noteBody is what is left of a note once its frontmatter and its
// headings are taken off: what someone would call the note itself. A note
// that is all title and properties has an empty body, which is exactly
// what makes it a stub.
func noteBody(src string) string {
	lines := strings.Split(src, "\n")
	if end := frontmatter.End(lines); end > 0 {
		lines = lines[min(end+1, len(lines)):]
	}
	var b strings.Builder
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue // a heading, or a line of tags
		}
		b.WriteString(l)
		b.WriteString("\n")
	}
	return b.String()
}

// nextItem is the next row holding an item, searching from row i in
// direction dir. It returns i itself when there is nowhere to go, so the
// cursor never lands on a heading.
func (v *weedsView) nextItem(i, dir int) int {
	for j := i + dir; j >= 0 && j < len(v.rows); j += dir {
		if v.rows[j].item != nil {
			return j
		}
	}
	if i >= 0 && i < len(v.rows) && v.rows[i].item != nil {
		return i
	}
	// Opening: land on the first item there is.
	for j := 0; j < len(v.rows); j++ {
		if v.rows[j].item != nil {
			return j
		}
	}
	return 0
}

// selected is the item under the cursor, if any.
func (v *weedsView) selected() *weeds.Item {
	if v.cur < 0 || v.cur >= len(v.rows) {
		return nil
	}
	return v.rows[v.cur].item
}

// weedKey handles the panel's keys.
func (m *Model) weedKey(k tea.KeyPressMsg) {
	v := m.weeds
	switch m.actionIn(inWeeds, k.String()) {
	case actCancel:
		m.weeds = nil
		m.flash = "Weeds closed"
	case actUp:
		v.cur = v.nextItem(v.cur, -1)
	case actDown:
		v.cur = v.nextItem(v.cur, 1)
	case actPick:
		m.goToWeed()
	case actMakeMissing:
		m.makeMissingNote()
	}
}

// goToWeed opens the place the row points at and closes the panel: the
// note, and for a link that leads nowhere the line the link is on.
func (m *Model) goToWeed() {
	it := m.weeds.selected()
	if it == nil {
		return
	}
	m.weeds = nil
	m.reveal(it.Rel)
	if it.Line >= 0 {
		m.goToLine(it.Rel, it.Line)
		m.flash = fmt.Sprintf("%s, line %d: the link to %s leads nowhere", it.Rel, it.Line+1, it.What)
		return
	}
	m.open(it.Rel)
}

// makeMissingNote is n on a link that leads nowhere: it makes the note the
// link wanted, exactly as following that link would — where Obsidian would
// put it, with no question in the way and U to take it back. The panel
// closes, because the note it just made opens for writing.
func (m *Model) makeMissingNote() {
	it := m.weeds.selected()
	if it == nil {
		return
	}
	if it.Kind != weeds.DeadLink {
		m.flash = "n makes the note a link wanted — this row isn't a link"
		return
	}
	m.weeds = nil
	// The note holding the link first, so a vault that puts new notes in
	// "the current folder" puts this one beside it, as following the link
	// from inside the note would.
	m.reveal(it.Rel)
	m.showNote(it.Rel)
	m.offerCreate(it.What)
}

// weedsBox draws the panel.
func (m *Model) weedsBox() []string {
	v := m.weeds
	if v == nil {
		return nil
	}
	w := min(max(m.width-6, 50), 96)
	inner := w - 4
	// The panel is as tall as it needs to be, up to what the terminal has
	// left for it once its own border and the status line are counted.
	maxRows := max(m.height-8, 6)
	body := make([]string, 0, maxRows)

	if len(v.rows) == 0 {
		body = append(body, "")
		body = append(body, "  "+m.st.bold.Render("Nothing to tend.")+m.st.muted.Render(" No dead links, no notes on their own, no stubs."))
		body = append(body, "")
		return m.box(" Weeds ", body, w, len(body)+2, true)
	}

	v.scroll(maxRows)
	for i := v.top; i < len(v.rows) && len(body) < maxRows; i++ {
		r := v.rows[i]
		switch {
		case r.head != "":
			if len(body) > 0 {
				body = append(body, "")
			}
			body = append(body, "  "+m.st.title.Render(r.head)+"  "+m.st.muted.Render(r.hint))
		default:
			body = append(body, m.weedLine(r, i == v.cur, inner))
		}
	}
	title := fmt.Sprintf(" Weeds — %s to tend ", plural(v.total, "loose end"))
	return m.box(title, body, w, len(body)+2, true)
}

// weedLine is one item's row: what it is on the left, where it is on the
// right, the cursor's row highlighted whole.
func (m *Model) weedLine(r weedRow, focused bool, inner int) string {
	it := r.item
	left := "  " + it.What
	if it.Kind == weeds.DeadLink {
		// As the note writes it, embed mark and all, so the row reads as
		// the line it came from.
		left = "  [[" + it.What + "]]"
		if strings.HasPrefix(it.Note, "embed") {
			left = "  ![[" + it.What + "]]"
		}
	}
	line := spread(left, it.Note+" ", inner-2)
	if focused {
		return "  " + m.st.selFocus.Render(line)
	}
	return "  " + line
}

// scroll keeps the cursor's row in view, with its group's heading above it
// when the group starts just off the top — a row whose heading has
// scrolled away says less than it should.
func (v *weedsView) scroll(rows int) {
	if v.cur < v.top {
		v.top = v.cur
	}
	if v.cur >= v.top+rows {
		v.top = v.cur - rows + 1
	}
	if v.top > 0 && v.top < len(v.rows) && v.rows[v.top].item != nil {
		// Pull the heading in when it is the row just above.
		if v.rows[v.top-1].head != "" {
			v.top--
		}
	}
	v.top = clamp(v.top, 0, max(len(v.rows)-1, 0))
}
