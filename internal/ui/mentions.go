package ui

import (
	"fmt"
	"path"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/mentions"
	"github.com/lurioso/skrin/internal/vault"
)

// Mentions is M: the other half of backlinks. b says which notes link
// here; M says which notes talk about this one without linking — its name
// or an alias written in plain prose, because the sentence was written
// before the note existed and nobody went back. Enter reads the place, a
// links it, A links them all, and U undoes the lot in one step.

// mentionsView is the open panel.
type mentionsView struct {
	rel   string   // the note being talked about
	names []string // its name and its aliases
	hits  []mentions.Hit
	// src is each note's text as it was when the panel scanned it, so a
	// note changed on disk since then is refused rather than overwritten.
	src map[string]string
	cur int
	top int
}

// openMentions gathers the mentions of the note in hand.
func (m *Model) openMentions() {
	rel, ok := m.subject()
	if !ok {
		m.flash = "Select a note to see where it's mentioned"
		return
	}
	v := m.scanMentions(rel)
	if len(v.hits) == 0 {
		m.flash = "No unlinked mentions of " + displayName(rel) + " · b shows what links here"
		return
	}
	m.mentions = v
}

// scanMentions reads the vault as the index has it and looks for the note's
// name and aliases in everyone else's prose.
func (m *Model) scanMentions(rel string) *mentionsView {
	base := path.Base(rel)
	names := []string{strings.TrimSuffix(base, path.Ext(base))}
	names = append(names, m.idx.Aliases(rel)...)
	docs := make([]mentions.Doc, 0, len(m.idx.Notes()))
	src := map[string]string{}
	for _, r := range m.idx.Notes() {
		content, ok := m.idx.Content(r)
		if !ok {
			continue
		}
		docs = append(docs, mentions.Doc{Rel: r, Content: content})
		src[r] = content
	}
	return &mentionsView{
		rel:   rel,
		names: names,
		hits:  mentions.Find(docs, names, rel),
		src:   src,
	}
}

// mentionKey handles the panel's keys.
func (m *Model) mentionKey(k tea.KeyPressMsg) {
	v := m.mentions
	switch m.actionIn(inMentions, k.String()) {
	case actCancel:
		m.mentions = nil
		m.flash = "Mentions closed"
	case actUp:
		v.cur = max(v.cur-1, 0)
	case actDown:
		v.cur = min(v.cur+1, len(v.hits)-1)
	case actPick:
		m.goToMention()
	case actLinkMention:
		m.linkMentions(false)
	case actLinkAllMentions:
		m.linkMentions(true)
	}
}

// goToMention opens the note the mention sits in, at its line, and closes
// the panel.
func (m *Model) goToMention() {
	v := m.mentions
	if v.cur >= len(v.hits) {
		return
	}
	h := v.hits[v.cur]
	m.mentions = nil
	m.reveal(h.Rel)
	m.goToLine(h.Rel, h.Line)
	m.flash = fmt.Sprintf("%s, line %d — %s is mentioned here", h.Rel, h.Line+1, displayName(v.rel))
}

// linkMentions writes the links in: the one under the cursor, or every
// mention in the list. The words already in the note become the link text,
// so the prose is unchanged and only the brackets are new. A note that
// changed on disk since the panel opened is left alone and named, never
// written over; the rest still go through, and the whole lot is one
// journal step that U takes back.
func (m *Model) linkMentions(all bool) {
	v := m.mentions
	if v.cur >= len(v.hits) {
		return
	}
	todo := v.hits[v.cur : v.cur+1]
	if all {
		todo = v.hits
	}
	byNote := map[string][]mentions.Hit{}
	order := []string{}
	for _, h := range todo {
		if _, seen := byNote[h.Rel]; !seen {
			order = append(order, h.Rel)
		}
		byNote[h.Rel] = append(byNote[h.Rel], h)
	}

	var steps []vault.Step
	var stale, failed []string
	notes, links := 0, 0
	for _, rel := range order {
		cur, err := m.vault.Read(rel)
		if err != nil || cur != v.src[rel] {
			stale = append(stale, displayName(rel))
			continue
		}
		_ = m.snaps.Save(rel, cur) // snapshots are what make u safe
		if err := m.vault.Write(rel, mentions.Apply(cur, byNote[rel])); err != nil {
			failed = append(failed, displayName(rel)+": "+err.Error())
			continue
		}
		steps = append(steps, vault.Step{Kind: vault.StepModified, Rel: rel, Content: cur})
		notes++
		links += len(byNote[rel])
	}
	if len(steps) > 0 {
		m.journal.Record(vault.Op{
			Desc:  fmt.Sprintf("link %s to %s", plural(links, "mention"), displayName(v.rel)),
			Steps: steps,
		})
	}
	m.refresh()

	// Scan again, so what is left in the list is what is still unlinked.
	again := m.scanMentions(v.rel)
	again.cur = min(v.cur, max(len(again.hits)-1, 0))
	if len(again.hits) == 0 {
		m.mentions = nil
	} else {
		m.mentions = again
	}
	m.flash = mentionFlash(links, notes, stale, failed)
}

// mentionFlash says what happened, naming anything that didn't.
func mentionFlash(links, notes int, stale, failed []string) string {
	switch {
	case links == 0 && len(stale) > 0:
		return "Nothing linked: " + strings.Join(stale, ", ") + " changed on disk · press M again"
	case links == 0 && len(failed) > 0:
		return "Nothing linked: " + strings.Join(failed, "; ")
	case links == 0:
		return "Nothing to link"
	}
	out := fmt.Sprintf("Linked %s in %s · u or U undoes", plural(links, "mention"), plural(notes, "note"))
	if len(stale) > 0 {
		out += " · left alone: " + strings.Join(stale, ", ") + " (changed on disk)"
	}
	if len(failed) > 0 {
		out += " · failed: " + strings.Join(failed, "; ")
	}
	return out
}

// mentionsBox draws the panel: the sentence on the left, where it is on
// the right, so the eye reads the prose and not the paths.
func (m *Model) mentionsBox() []string {
	v := m.mentions
	if v == nil {
		return nil
	}
	w := min(max(m.width-6, 50), 96)
	inner := w - 4
	rows := max(m.height-10, 4)
	if v.cur < v.top {
		v.top = v.cur
	}
	if v.cur >= v.top+rows {
		v.top = v.cur - rows + 1
	}
	var body []string
	for i := v.top; i < len(v.hits) && len(body) < rows; i++ {
		h := v.hits[i]
		where := fmt.Sprintf("%s · %d ", displayName(h.Rel), h.Line+1)
		line := spread("  "+h.Context, where, inner-2)
		if i == v.cur {
			body = append(body, "  "+m.st.selFocus.Render(line))
			continue
		}
		body = append(body, "  "+line)
	}
	body = append(body, "")
	body = append(body, "  "+m.st.muted.Render(fmt.Sprintf("%s · %s links this one · %s links all · enter reads it",
		plural(len(v.hits), "mention"),
		note(m.keyFor(inMentions, actLinkMention), "a"),
		note(m.keyFor(inMentions, actLinkAllMentions), "A"))))
	title := " Mentions of " + displayName(v.rel) + " "
	if len(v.names) > 1 {
		title = fmt.Sprintf(" Mentions of %s (and %s) ", displayName(v.rel), plural(len(v.names)-1, "alias"))
	}
	return m.box(title, body, w, len(body)+2, true)
}
