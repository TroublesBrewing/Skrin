package ui

import (
	"slices"

	"github.com/lurioso/skrin/internal/vault"
)

// Files can be kept in your own order, level by level, in .skrin at the
// vault root. There is no mode for it: Shift+↑ and Shift+↓ move the item
// under the cursor within its level, and R puts a level back in the
// default order. Every move is its own journal step, so U takes back the
// last one.

// cursorLevel is the folder whose order the move keys change: the one
// holding the row under the cursor.
func (m *Model) cursorLevel() string { return parentOf(m.files.selected().Rel) }

// inFiles reports whether the arranging keys apply, and says what would
// unblock them when they don't.
func (m *Model) inFiles(what string) bool {
	if m.focus == paneFiles {
		return true
	}
	m.flash = "Go to Files (1) to " + what
	return false
}

// shiftItem is Shift+↓ and Shift+↑: the item under the cursor trades
// places with its neighbour in the level.
func (m *Model) shiftItem(down bool) {
	if !m.inFiles("move an item") {
		return
	}
	e := m.files.selected()
	if e.Rel == "" {
		m.flash = "The vault row stays put"
		return
	}
	dir := parentOf(e.Rel)
	names := m.files.childNames(dir)
	i := slices.Index(names, e.Name)
	j, edge, way := i-1, "top", "up"
	if down {
		j, edge, way = i+1, "bottom", "down"
	}
	if i < 0 || j < 0 || j >= len(names) {
		m.flash = "Already at the " + edge + " of " + m.folderLabel(dir)
		return
	}
	names[i], names[j] = names[j], names[i]
	m.order[dir] = names
	if m.saveOrder("move " + displayName(e.Rel) + " " + way) {
		m.flash = "Moved " + displayName(e.Rel) + " " + way + " · U undoes"
	}
}

// resetLevel is R: the level the cursor is in goes back to the default
// order.
func (m *Model) resetLevel() {
	if !m.inFiles("put a level back in the default order") {
		return
	}
	dir := m.cursorLevel()
	if _, ok := m.order[dir]; !ok {
		m.flash = m.folderLabel(dir) + " is already in the default order"
		return
	}
	delete(m.order, dir)
	if m.saveOrder("reset " + m.folderLabel(dir)) {
		m.flash = m.folderLabel(dir) + " is back in the default order · U undoes"
	}
}

// saveOrder writes .skrin and shows the new order, with the cursor staying
// on its item. The change is one journal step, so U takes back this move
// and nothing else. It reports whether the order was saved.
func (m *Model) saveOrder(desc string) bool {
	before, err := m.vault.Read(vault.OrderFile)
	existed := err == nil
	if err := m.vault.Write(vault.OrderFile, m.order.Marshal()); err != nil {
		m.flash = "Couldn't save the order: " + err.Error()
		return false
	}
	step := vault.Step{Kind: vault.StepModified, Rel: vault.OrderFile, Content: before}
	if !existed {
		// The first move in this vault: undoing it takes the file away.
		after, _ := m.vault.Read(vault.OrderFile)
		step = vault.Step{Kind: vault.StepCreated, Rel: vault.OrderFile, Content: after}
	}
	m.journal.Record(vault.Op{Desc: desc, Steps: []vault.Step{step}})
	m.refresh()
	return true
}

// orderFollows keeps Files' manual order through moves and renames: a
// renamed item keeps its place, and an ordered folder keeps its order under
// its new path. It returns the journal step for the change, if any.
func (m *Model) orderFollows(moved [][2]string) (vault.Step, bool) {
	before, err := m.vault.Read(vault.OrderFile)
	if err != nil {
		return vault.Step{}, false
	}
	o := m.vault.LoadOrder()
	if !o.Follow(moved) || m.vault.Write(vault.OrderFile, o.Marshal()) != nil {
		return vault.Step{}, false
	}
	return vault.Step{Kind: vault.StepModified, Rel: vault.OrderFile, Content: before}, true
}
