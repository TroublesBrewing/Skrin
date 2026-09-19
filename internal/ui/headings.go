package ui

import (
	"fmt"

	"github.com/lurioso/skrin/internal/index"
	"github.com/lurioso/skrin/internal/vault"
)

// headingRename is a heading whose text changed while the note was open.
type headingRename struct{ from, to string }

// headingRenames pairs the headings a note had when the editor opened with
// the ones it has now, and reports those that were renamed. Headings that
// are still there, in the same order, hold as anchors first, so adding or
// removing one elsewhere doesn't look like a rename. In the gaps between
// the anchors, one heading gone and one arrived in its place, at the same
// level, is a rename; anything left over is an ordinary add or delete.
//
// A heading whose name still exists elsewhere in the note is left alone:
// the links to it go on resolving, so nothing needs to change.
func headingRenames(before, after string) []headingRename {
	was, now := index.ParseHeadings(before), index.ParseHeadings(after)
	same := func(a, b index.Heading) bool {
		return a.Level == b.Level && index.SameHeading(a.Text, b.Text)
	}
	still := func(text string) bool {
		for _, h := range now {
			if index.SameHeading(h.Text, text) {
				return true
			}
		}
		return false
	}

	// The longest run of headings that didn't change, by the usual
	// backwards table, gives the anchors.
	keep := make([][]int, len(was)+1)
	for i := range keep {
		keep[i] = make([]int, len(now)+1)
	}
	for i := len(was) - 1; i >= 0; i-- {
		for j := len(now) - 1; j >= 0; j-- {
			if same(was[i], now[j]) {
				keep[i][j] = keep[i+1][j+1] + 1
			} else {
				keep[i][j] = max(keep[i+1][j], keep[i][j+1])
			}
		}
	}

	var out []headingRename
	var gone, came []index.Heading
	pair := func() {
		for k := 0; k < len(gone) && k < len(came); k++ {
			if gone[k].Level == came[k].Level && !still(gone[k].Text) {
				out = append(out, headingRename{gone[k].Text, came[k].Text})
			}
		}
		gone, came = nil, nil
	}
	i, j := 0, 0
	for i < len(was) && j < len(now) {
		switch {
		case same(was[i], now[j]):
			pair()
			i, j = i+1, j+1
		case keep[i+1][j] >= keep[i][j+1]:
			gone = append(gone, was[i])
			i++
		default:
			came = append(came, now[j])
			j++
		}
	}
	gone = append(gone, was[i:]...)
	came = append(came, now[j:]...)
	pair()
	return out
}

// subEdit is one link whose #heading has to follow a rename.
type subEdit struct {
	source string
	link   index.Link
	sub    string
}

// offerHeadingRelink asks, as the editor closes on rel, whether the links
// to a heading that changed name should follow it — the same offer moving
// a note makes, and what Obsidian does on a heading rename. Links from the
// note to its own headings count too.
func (m *Model) offerHeadingRelink(rel, before, after string) {
	renames := headingRenames(before, after)
	if len(renames) == 0 {
		return
	}
	if err := m.idx.Update(m.vault); err != nil {
		m.flash = "Renamed, but couldn't check the links: " + err.Error()
		return
	}
	var edits []subEdit
	for _, b := range m.idx.AllLinksTo(rel) {
		if b.Link.Sub == "" {
			continue
		}
		for _, r := range renames {
			if sub, ok := index.RenameSub(b.Link.Sub, r.from, r.to); ok {
				edits = append(edits, subEdit{b.Source, b.Link, sub})
				break
			}
		}
	}
	if len(edits) == 0 {
		return
	}
	notes := map[string]bool{}
	for _, e := range edits {
		notes[e.source] = true
	}
	what := fmt.Sprintf("%q is now %q", renames[0].from, renames[0].to)
	if len(renames) > 1 {
		what = plural(len(renames), "heading") + " renamed"
	}
	m.confirm = &confirm{
		pill: " LINKS ",
		question: fmt.Sprintf("%s: update %s in %s so they keep working?",
			what, plural(len(edits), "link"), plural(len(notes), "note")),
		keys:   "y update · n or esc leaves them",
		cancel: "Links left as they were",
		yes:    func() { m.relinkHeadings(edits) },
		no:     func() { m.flash = "Links left as they were" },
	}
}

// relinkHeadings rewrites the links in edits as one undoable operation.
func (m *Model) relinkHeadings(edits []subEdit) {
	bySource := map[string][]index.Edit{}
	for _, e := range edits {
		bySource[e.source] = append(bySource[e.source],
			index.Edit{Link: e.link, Target: e.link.Target, Sub: e.sub})
	}
	steps, n, err := m.applyLinkEdits(bySource)
	if len(steps) > 0 {
		m.journal.Record(vault.Op{Desc: "Heading links updated", Steps: steps})
	}
	m.refresh()
	if err != nil {
		m.flash = "Updated " + plural(n, "link") + ", then stopped: " + err.Error()
		return
	}
	m.flash = "Updated " + plural(n, "link") + " · U undoes it"
}
