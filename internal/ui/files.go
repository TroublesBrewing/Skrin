package ui

import (
	"sort"

	"github.com/lurioso/skrin/internal/vault"
)

// fileRow is one visible row in Files.
type fileRow struct {
	vault.Entry // Rel "" is the vault itself
	depth       int
}

// files is the Files pane: the vault's folders and files as one collapsible
// tree, folders first, like Obsidian's file explorer. The top row is the
// vault itself, which is always open.
type files struct {
	kids     map[string][]vault.Entry // folder → its entries, in explorer order
	byRel    map[string]vault.Entry
	expanded map[string]bool
	rows     []fileRow // visible rows
	cur, off int
}

func newFiles() files { return files{expanded: map[string]bool{"": true}} }

// set loads the vault's entries and rebuilds the rows. Folders that are gone
// are forgotten. The cursor stays on its item; see restore for when that
// item is gone.
func (t *files) set(entries []vault.Entry, rootName string, order vault.Order) {
	was, at := t.selected().Rel, t.cur
	t.kids = map[string][]vault.Entry{}
	t.byRel = map[string]vault.Entry{"": {Name: rootName, IsDir: true}}
	for _, e := range entries {
		p := parentOf(e.Rel)
		t.kids[p] = append(t.kids[p], e)
		t.byRel[e.Rel] = e
	}
	for dir, k := range t.kids {
		vault.Sort(k)
		order.Arrange(dir, k) // a level in .skrin keeps the user's order
	}
	for p := range t.expanded {
		if e, ok := t.byRel[p]; !ok || !e.IsDir {
			delete(t.expanded, p)
		}
	}
	t.expanded[""] = true
	t.build()
	t.restore(was, at)
}

func (t *files) build() {
	t.rows = append(t.rows[:0], fileRow{Entry: t.byRel[""]})
	t.walk("", 1)
}

func (t *files) walk(dir string, depth int) {
	for _, e := range t.kids[dir] {
		t.rows = append(t.rows, fileRow{e, depth})
		if e.IsDir && t.expanded[e.Rel] {
			t.walk(e.Rel, depth+1)
		}
	}
}

// restore puts the cursor back on was, which sat at row at. If it's gone, a
// neighbour from the same folder takes its place (the row now at its index,
// or the one above), or else the nearest folder above it that still shows.
func (t *files) restore(was string, at int) {
	if t.selectPath(was) {
		return
	}
	for _, i := range []int{at, at - 1} {
		if i > 0 && i < len(t.rows) && parentOf(t.rows[i].Rel) == parentOf(was) {
			t.cur = i
			return
		}
	}
	for p := parentOf(was); !t.selectPath(p); p = parentOf(p) {
	}
}

// relayout rebuilds the rows after folders opened or closed. The cursor
// stays on its item, or moves up to the nearest folder still showing.
func (t *files) relayout() {
	was := t.selected().Rel
	t.build()
	for p := was; !t.selectPath(p); p = parentOf(p) {
	}
}

// selected is the entry under the cursor. The vault row has Rel "".
func (t *files) selected() vault.Entry {
	if t.cur < len(t.rows) {
		return t.rows[t.cur].Entry
	}
	return vault.Entry{IsDir: true}
}

func (t *files) entry(rel string) (vault.Entry, bool) {
	e, ok := t.byRel[rel]
	return e, ok
}

// selectPath moves the cursor to p if p is showing.
func (t *files) selectPath(p string) bool {
	for i, r := range t.rows {
		if r.Rel == p {
			t.cur = i
			return true
		}
	}
	return false
}

// reveal opens the folders above p and puts the cursor on it.
func (t *files) reveal(p string) {
	for q := parentOf(p); q != ""; q = parentOf(q) {
		t.expanded[q] = true
	}
	t.relayout()
	t.selectPath(p)
}

// toggle opens or closes the folder under the cursor.
func (t *files) toggle() {
	if e := t.selected(); e.IsDir && e.Rel != "" {
		t.expanded[e.Rel] = !t.expanded[e.Rel]
		t.relayout()
	}
}

// in steps into the folder under the cursor, opening it, and reports false
// when the cursor is on a file.
func (t *files) in() bool {
	e := t.selected()
	if !e.IsDir {
		return false
	}
	if !t.expanded[e.Rel] {
		t.expanded[e.Rel] = true
		t.relayout()
	}
	if len(t.kids[e.Rel]) > 0 {
		t.cur++ // the folder's first entry is the next row
	}
	return true
}

// out closes the folder under the cursor if it's open, and otherwise moves
// up to the folder the cursor's item is in.
func (t *files) out() {
	e := t.selected()
	switch {
	case e.Rel == "":
	case e.IsDir && t.expanded[e.Rel]:
		delete(t.expanded, e.Rel)
		t.relayout()
	default:
		t.selectPath(parentOf(e.Rel))
	}
}

// up moves to the folder the cursor's item is in.
func (t *files) up() {
	if e := t.selected(); e.Rel != "" {
		t.selectPath(parentOf(e.Rel))
	}
}

// collapseAll closes every folder, like Obsidian's "Collapse all".
func (t *files) collapseAll() {
	t.expanded = map[string]bool{"": true}
	t.relayout()
}

// folder is the current folder: the folder under the cursor, or the folder
// holding the file under it.
func (t *files) folder() string {
	e := t.selected()
	if e.IsDir {
		return e.Rel
	}
	return parentOf(e.Rel)
}

// siblings are the entries in the same folder as the cursor's row; on the
// vault row, the top-level entries.
func (t *files) siblings() []vault.Entry {
	e := t.selected()
	if e.Rel == "" {
		return t.kids[""]
	}
	return t.kids[parentOf(e.Rel)]
}

// childNames is folder dir's entries by name, in the order Files shows
// them.
func (t *files) childNames(dir string) []string {
	var out []string
	for _, e := range t.kids[dir] {
		out = append(out, e.Name)
	}
	return out
}

// follow carries open folders across moves and renames, and with cursor the
// cursor as well. The next set picks up the new paths.
func (t *files) follow(moves [][2]string, cursor bool) {
	exp := make(map[string]bool, len(t.expanded))
	for p := range t.expanded {
		exp[movedPath(p, moves)] = true
	}
	t.expanded = exp
	if cursor && t.cur < len(t.rows) {
		t.rows[t.cur].Rel = movedPath(t.rows[t.cur].Rel, moves)
	}
}

// openFolders lists the open folders, sorted, for the session.
func (t *files) openFolders() []string {
	var out []string
	for p := range t.expanded {
		if p != "" {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}
