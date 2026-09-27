package ui

import (
	"fmt"
	"strings"

	"github.com/lurioso/skrin/internal/frontmatter"
	"github.com/lurioso/skrin/internal/index"
	"github.com/lurioso/skrin/internal/obsidian"
	"github.com/lurioso/skrin/internal/vault"
)

// Merging is the answer to the one thing Weeds finds and can't fix: two
// notes that are the same note. One's text joins the end of the other,
// every link to it follows, and it goes to the trash — one journal step, so
// U puts all three back together.
//
// It has no key. It is a rare, large action, so it lives in the palette
// where it is found by name, and the keymap gains nothing.

// startMerge asks which note this one should join.
func (m *Model) startMerge() {
	rel, ok := m.subject()
	if !ok {
		m.flash = "Select a note to merge into another"
		return
	}
	var items []choice
	for _, other := range m.idx.Notes() {
		if other == rel {
			continue
		}
		other := other
		items = append(items, choice{
			label:  displayName(other),
			detail: m.folderLabel(parentOf(other)),
			rel:    other,
			do:     func() { m.askMerge(rel, other) },
		})
	}
	if len(items) == 0 {
		m.flash = "Nothing to merge into: this is the only note in the vault"
		return
	}
	m.openChooser(&chooser{
		title:  "Merge " + displayName(rel) + " into…",
		prompt: "filter",
		empty:  "No match",
		verb:   "merge into",
		items:  items,
	})
}

// askMerge asks before merging: a note disappears, which is the same stake
// as deleting one, and d asks too. The question names every part of what
// happens, so nobody finds out afterwards.
func (m *Model) askMerge(source, target string) {
	links := len(m.idx.AllLinksTo(source))
	q := fmt.Sprintf("Merge %s into %s? Its text joins the end of %s",
		displayName(source), displayName(target), displayName(target))
	if links > 0 {
		follow := "follow"
		if links == 1 {
			follow = "follows"
		}
		q += fmt.Sprintf(", %s to it %s", plural(links, "link"), follow)
	}
	q += ", and it goes to the trash."
	m.confirm = &confirm{
		pill:     " MERGE ",
		question: q,
		keys:     "y/n",
		cancel:   "Nothing merged",
		yes:      func() { m.mergeNow(source, target) },
	}
}

// mergeNow does it, as one undoable operation: the target gains the
// source's text, links to the source are rewritten to the target, and the
// source goes to the trash. Anything that can't be read stops the whole
// thing before a single write.
func (m *Model) mergeNow(source, target string) {
	srcText, err := m.vault.Read(source)
	if err != nil {
		m.flash = "Can't read " + source + ": " + err.Error()
		return
	}
	tgtText, err := m.vault.Read(target)
	if err != nil {
		m.flash = "Can't read " + target + ": " + err.Error()
		return
	}
	merged := mergedText(tgtText, srcText, displayName(source))
	if merged == tgtText {
		m.flash = displayName(source) + " has nothing in it to merge · delete it with d instead"
		return
	}

	var steps []vault.Step
	_ = m.snaps.Save(target, tgtText) // snapshots are what make u safe
	if err := m.vault.Write(target, merged); err != nil {
		m.flash = "Couldn't write " + target + ": " + err.Error()
		return
	}
	steps = append(steps, vault.Step{Kind: vault.StepModified, Rel: target, Content: tgtText})

	// Links to the source now mean the target. This runs before the source
	// is trashed, so the index still knows what pointed at it.
	relinked, err := m.relinkTo(source, target)
	steps = append(steps, relinked...)
	links := 0
	for _, s := range relinked {
		if s.Kind == vault.StepModified {
			links++
		}
	}
	if err != nil {
		m.journal.Record(vault.Op{Desc: "merge " + source + " into " + target, Steps: steps})
		m.refresh()
		m.flash = "Merged the text but not every link: " + err.Error() + " · U undoes"
		return
	}

	option := obsidian.LoadSettings(m.vault.Root).TrashOption
	trashed := true
	if option == "none" {
		if err := m.vault.Remove(source); err != nil {
			trashed = false
		}
	} else if t, err := m.vault.Trash(source, option); err == nil {
		steps = append(steps, vault.Step{Kind: vault.StepTrashed, Rel: source, Trash: t})
	} else {
		trashed = false
	}
	m.journal.Record(vault.Op{Desc: "merge " + source + " into " + target, Steps: steps})

	m.refresh()
	m.reveal(target)
	m.open(target)
	out := fmt.Sprintf("Merged %s into %s", displayName(source), displayName(target))
	if links > 0 {
		out += fmt.Sprintf(" · %s updated", plural(links, "note's links"))
	}
	if !trashed {
		out += " · " + displayName(source) + " is still there: couldn't trash it"
	}
	if frontmatter.End(strings.Split(srcText, "\n")) > 0 {
		out += " · its properties stayed behind, in the trash copy"
	}
	m.flash = out + " · U undoes"
}

// relinkTo points every link to source at target instead, in whatever form
// each link was written. It reuses the rewriting a rename does, so a
// wikilink stays a wikilink and a markdown link keeps its shape.
func (m *Model) relinkTo(source, target string) ([]vault.Step, error) {
	if err := m.idx.Update(m.vault); err != nil {
		return nil, err
	}
	format := obsidian.LoadSettings(m.vault.Root).NewLinkFormat
	bySource := map[string][]index.Edit{}
	for _, b := range m.idx.AllLinksTo(source) {
		if b.Source == source {
			continue // a link inside the note being merged away
		}
		text := m.idx.LinkText(target, b.Source, format)
		if b.Link.Markdown {
			text = markdownPath(relink{source: b.Source, target: source, link: b.Link}, b.Source, target)
		}
		bySource[b.Source] = append(bySource[b.Source], index.Edit{Link: b.Link, Target: text})
	}
	steps, _, err := m.applyLinkEdits(bySource)
	return steps, err
}

// mergedText is the target note with the source's text added at the end. The
// source's frontmatter is left out — two sets of properties can't be one —
// and the trashed copy keeps it. A source that starts with a heading brings
// its own; one that doesn't gets a heading naming it, so the joined text
// doesn't run into the note above it without a seam.
func mergedText(target, source, name string) string {
	body := strings.TrimSpace(withoutFrontmatter(source))
	if body == "" {
		return target
	}
	out := strings.TrimRight(target, "\n")
	if out != "" {
		out += "\n\n"
	}
	if !strings.HasPrefix(body, "#") {
		out += "## " + name + "\n\n"
	}
	return out + body + "\n"
}

// withoutFrontmatter is a note's text with its property block taken off.
func withoutFrontmatter(src string) string {
	lines := strings.Split(src, "\n")
	end := frontmatter.End(lines)
	if end == 0 {
		return src
	}
	return strings.Join(lines[min(end+1, len(lines)):], "\n")
}
