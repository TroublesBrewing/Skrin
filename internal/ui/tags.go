package ui

import (
	"fmt"
	"strings"

	"github.com/lurioso/skrin/internal/tagname"
	"github.com/lurioso/skrin/internal/vault"
)

// openTags lists every tag in the vault with how many notes use it, and
// searches for the one you pick. The index has known the tags since the
// suggestions were built; this is the way in that was missing.
//
// Each spelling stands on its own row, as in the suggestions: "Filosofi"
// and "filosofi" side by side, so you can see the split and pick one
// instead of making a third.
func (m *Model) openTags() {
	tags := m.idx.Tags()
	if len(tags) == 0 {
		m.flash = "No tags in the vault yet · #a-tag in a note makes one"
		return
	}
	c := &chooser{title: "Tags", prompt: "tag", empty: "No tag by that name", verb: "search"}
	for _, t := range tags {
		c.items = append(c.items, choice{
			label:  "#" + t.Text,
			detail: plural2(t.Notes, "note", "notes"),
			do:     func() { m.searchTag(t.Text) },
		})
	}
	m.openChooser(c)
}

// searchTag opens the search panel on #tag, the same query you would type
// yourself — one way to search, not two.
func (m *Model) searchTag(tag string) {
	m.openSearch()
	m.search.in.set("#" + tag)
	m.search.inNote = false
	m.runSearch()
}

// Renaming a tag everywhere is the fix for what the tag list keeps showing
// and Weeds keeps finding: one tag written two ways, or a tag misspelled
// once and never again. Obsidian has it; Skrin's only answer used to be
// search and replace, note by note, which is no answer for something that
// also lives in frontmatter.
//
// It has no key of its own. It is rare and large, so the palette carries
// it, and picking the tag from the list is the same list # already shows.

// startTagRename asks which tag to rename.
func (m *Model) startTagRename() {
	tags := m.idx.Tags()
	if len(tags) == 0 {
		m.flash = "No tags in the vault yet · #a-tag in a note makes one"
		return
	}
	c := &chooser{title: "Rename which tag?", prompt: "tag", empty: "No tag by that name", verb: "rename"}
	for _, t := range tags {
		t := t
		c.items = append(c.items, choice{
			label:  "#" + t.Text,
			detail: plural2(t.Notes, "note", "notes"),
			do:     func() { m.askTagName(t.Text) },
		})
	}
	m.openChooser(c)
}

// askTagName asks what it should be called instead, with the old name
// there to edit.
func (m *Model) askTagName(tag string) {
	p := &prompt{kind: promptRenameTag, label: "Rename #" + tag + " to", target: tag}
	p.in.set(tag)
	m.prompt = p
}

// renameTag rewrites the tag in every note that has it: the #tags in the
// text and the values of a tags property, in one journal step. Every
// spelling of the old tag is gathered into the new one, and a tag nested
// under it comes along. Notes are read fresh as they are written, so
// nothing is done to a note on the strength of a stale reading, and each
// one is snapshotted first.
func (m *Model) renameTag(old, want string) error {
	from, to := tagname.Clean(old), tagname.Clean(want)
	if err := tagname.Valid(to); err != nil {
		return err
	}
	if strings.EqualFold(from, to) && from == to {
		return fmt.Errorf("#%s is already its name", from)
	}
	var steps []vault.Step
	var failed []string
	notes, tags := 0, 0
	for _, rel := range m.idx.Notes() {
		cur, err := m.vault.Read(rel)
		if err != nil {
			continue // a note that isn't readable right now keeps what it has
		}
		next, n := tagname.Rename(cur, from, to)
		if n == 0 || next == cur {
			continue
		}
		_ = m.snaps.Save(rel, cur) // snapshots are what make u safe
		if err := m.vault.Write(rel, next); err != nil {
			failed = append(failed, displayName(rel)+": "+err.Error())
			continue
		}
		steps = append(steps, vault.Step{Kind: vault.StepModified, Rel: rel, Content: cur})
		notes++
		tags += n
	}
	if len(steps) == 0 {
		if len(failed) > 0 {
			return fmt.Errorf("nothing renamed: %s", strings.Join(failed, "; "))
		}
		return fmt.Errorf("no note has #%s", from)
	}
	m.journal.Record(vault.Op{
		Desc:  fmt.Sprintf("rename #%s to #%s", from, to),
		Steps: steps,
	})
	m.refresh()
	out := fmt.Sprintf("Renamed #%s to #%s · %s in %s · U undoes", from, to, plural(tags, "tag"), plural(notes, "note"))
	if len(failed) > 0 {
		out += " · failed: " + strings.Join(failed, "; ")
	}
	m.flash = out
	return nil
}
