package ui

import (
	"path"
	"sort"
	"strings"
)

type treeRow struct {
	path    string // vault-relative; "" is the root
	name    string
	depth   int
	hasKids bool
}

// tree is the folder pane: all vault folders, shown as a collapsible tree.
type tree struct {
	children map[string][]string // parent → child folders, sorted
	expanded map[string]bool
	rootName string
	rows     []treeRow // visible rows
	cur, off int
}

func newTree() tree { return tree{expanded: map[string]bool{"": true}} }

func (t *tree) setDirs(dirs []string, rootName string) {
	sel := t.selected()
	t.rootName = rootName
	t.children = map[string][]string{}
	for _, d := range dirs {
		if d != "" {
			p := parentOf(d)
			t.children[p] = append(t.children[p], d)
		}
	}
	for _, kids := range t.children {
		sort.Slice(kids, func(i, j int) bool {
			return strings.ToLower(path.Base(kids[i])) < strings.ToLower(path.Base(kids[j]))
		})
	}
	t.build()
	t.selectPath(sel)
}

func (t *tree) build() {
	t.rows = t.rows[:0]
	t.walk("", 0)
}

func (t *tree) walk(p string, depth int) {
	name := t.rootName
	if p != "" {
		name = path.Base(p)
	}
	kids := t.children[p]
	t.rows = append(t.rows, treeRow{p, name, depth, len(kids) > 0})
	if t.expanded[p] {
		for _, k := range kids {
			t.walk(k, depth+1)
		}
	}
}

func (t *tree) selected() string {
	if t.cur < len(t.rows) {
		return t.rows[t.cur].path
	}
	return ""
}

// has reports whether folder p exists in the vault.
func (t *tree) has(p string) bool {
	if p == "" {
		return true
	}
	for _, k := range t.children[parentOf(p)] {
		if k == p {
			return true
		}
	}
	return false
}

// selectPath moves the cursor to folder p, or to the root if p isn't visible.
func (t *tree) selectPath(p string) {
	t.cur = 0
	for i, r := range t.rows {
		if r.path == p {
			t.cur = i
			return
		}
	}
}

func (t *tree) toggle() {
	if t.cur >= len(t.rows) || !t.rows[t.cur].hasKids {
		return
	}
	p := t.rows[t.cur].path
	t.expanded[p] = !t.expanded[p]
	t.build()
	t.selectPath(p)
}

// expandTo opens every ancestor of p so that p is visible.
func (t *tree) expandTo(p string) {
	for q := p; q != ""; q = parentOf(q) {
		t.expanded[parentOf(q)] = true
	}
	t.build()
}
