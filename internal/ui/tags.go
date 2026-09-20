package ui

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
