package vault

import (
	"encoding/json"
	"os"
	"path"
	"sort"
	"strings"
)

// OrderFile holds the manual order of Files levels. It sits at the vault
// root, so it travels with the vault, and it's hidden, so neither Skrin nor
// Obsidian lists it. Deleting it puts every level back in the default
// order.
const OrderFile = ".skrin"

// Order is the manual order of Files levels: for each ordered folder
// (vault-relative, "" for the root), its children's names in order. A
// level that isn't in it keeps the default order.
type Order map[string][]string

// LoadOrder reads the order file. A missing or unreadable file, or one that
// isn't the JSON Skrin writes, means no manual order at all.
func (v *Vault) LoadOrder() Order {
	data, err := os.ReadFile(v.Abs(OrderFile))
	if err != nil {
		return Order{}
	}
	var raw map[string][]string
	if json.Unmarshal(data, &raw) != nil {
		return Order{}
	}
	o := Order{}
	for key, names := range raw {
		o[strings.Trim(key, "/")] = names
	}
	return o
}

// Marshal is the order file's text. Folders are keys ending in "/", and
// the root is "/".
func (o Order) Marshal() string {
	raw := make(map[string][]string, len(o))
	for dir, names := range o {
		raw[dir+"/"] = names // the root, "", becomes "/"
	}
	b, _ := json.MarshalIndent(raw, "", "  ")
	return string(b) + "\n"
}

// Arrange puts the entries of folder dir, already in the default order, in
// the level's manual order: the listed names first, as listed, then
// anything new, still in the default order.
func (o Order) Arrange(dir string, entries []Entry) {
	names, ok := o[dir]
	if !ok {
		return
	}
	pos := make(map[string]int, len(names))
	for i, n := range names {
		if _, dup := pos[n]; !dup {
			pos[n] = i
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		pi, iok := pos[entries[i].Name]
		pj, jok := pos[entries[j].Name]
		if iok && jok {
			return pi < pj
		}
		return iok && !jok
	})
}

// Follow updates the order for items that moved: a renamed item keeps its
// place in its level, one that left a level leaves its list, and an ordered
// folder keeps its order under its new path. It reports whether anything
// changed.
func (o Order) Follow(moves [][2]string) bool {
	changed := false
	for _, mv := range moves {
		from, to := mv[0], mv[1]
		if names, ok := o[levelOf(from)]; ok {
			if i := indexOf(names, path.Base(from)); i >= 0 {
				if levelOf(from) == levelOf(to) {
					names[i] = path.Base(to)
				} else {
					o[levelOf(from)] = append(names[:i:i], names[i+1:]...)
				}
				changed = true
			}
		}
		var under []string
		for dir := range o {
			if dir == from || strings.HasPrefix(dir, from+"/") {
				under = append(under, dir)
			}
		}
		for _, dir := range under {
			o[to+strings.TrimPrefix(dir, from)] = o[dir]
			delete(o, dir)
			changed = true
		}
	}
	return changed
}

// levelOf is the folder holding rel, "" for the root.
func levelOf(rel string) string {
	if d := path.Dir(rel); d != "." {
		return d
	}
	return ""
}

func indexOf(names []string, name string) int {
	for i, n := range names {
		if n == name {
			return i
		}
	}
	return -1
}
