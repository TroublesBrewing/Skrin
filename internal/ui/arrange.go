package ui

import (
	"slices"

	"github.com/lurioso/skrin/internal/vault"
)

// arranging is arrange mode (A): J and K reorder the Files level under the
// cursor, saving .skrin as they go. before is the order file as the
// session found it, for the one journal step written when the session
// ends.
type arranging struct {
	before  string
	existed bool
}

// enterArrange is A: arrange mode on the Files level under the cursor.
func (m *Model) enterArrange() {
	before, err := m.vault.Read(vault.OrderFile)
	m.arrange = &arranging{before: before, existed: err == nil}
	m.focus, m.zen, m.visual = paneFiles, false, nil
}

// arrangeLevel is the folder whose order J, K and R change: the one holding
// the row under the cursor.
func (m *Model) arrangeLevel() string { return parentOf(m.files.selected().Rel) }

// arrangeKey handles the keys arrange mode adds; the rest work as usual.
func (m *Model) arrangeKey(a action) {
	e := m.files.selected()
	switch a {
	case actOrderUp, actOrderDown:
		m.shiftItem(a == actOrderDown)
	case actOrderReset:
		m.resetLevel()
	case actArrangeIn:
		if e.IsDir {
			m.files.in()
		}
	case actToggleFolder:
		if e.IsDir {
			m.files.toggle()
		}
	case actLeaveArrange:
		m.leaveArrange()
	}
}

// shiftItem is J or K: the item under the cursor trades places with its
// neighbour in the level, and the level's new order is saved.
func (m *Model) shiftItem(down bool) {
	e := m.files.selected()
	if e.Rel == "" {
		m.flash = "The vault row stays at the top"
		return
	}
	dir := parentOf(e.Rel)
	names := m.files.childNames(dir)
	i := slices.Index(names, e.Name)
	j, edge := i-1, "top"
	if down {
		j, edge = i+1, "bottom"
	}
	if i < 0 || j < 0 || j >= len(names) {
		m.flash = "Already at the " + edge + " of " + m.folderLabel(dir)
		return
	}
	names[i], names[j] = names[j], names[i]
	m.order[dir] = names
	m.saveOrder()
}

// resetLevel is R: the level goes back to the default order.
func (m *Model) resetLevel() {
	dir := m.arrangeLevel()
	if _, ok := m.order[dir]; !ok {
		m.flash = m.folderLabel(dir) + " is already in the default order"
		return
	}
	delete(m.order, dir)
	m.saveOrder()
	m.flash = m.folderLabel(dir) + " is back in the default order"
}

// saveOrder writes .skrin and shows the new order, with the cursor staying
// on its item.
func (m *Model) saveOrder() {
	if err := m.vault.Write(vault.OrderFile, m.order.Marshal()); err != nil {
		m.flash = "Couldn't save the order: " + err.Error()
		return
	}
	m.refresh()
}

// leaveArrange ends arrange mode. What the session changed becomes one
// journal step, so one U undoes every move.
func (m *Model) leaveArrange() {
	a := m.arrange
	m.arrange = nil
	after, err := m.vault.Read(vault.OrderFile)
	if err != nil || after == a.before {
		return
	}
	step := vault.Step{Kind: vault.StepModified, Rel: vault.OrderFile, Content: a.before}
	if !a.existed {
		step = vault.Step{Kind: vault.StepCreated, Rel: vault.OrderFile, Content: after}
	}
	m.journal.Record(vault.Op{Desc: "arrange Files", Steps: []vault.Step{step}})
	m.flash = "Arranged · U undoes the whole session"
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
