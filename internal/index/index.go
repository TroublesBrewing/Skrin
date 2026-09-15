// Package index keeps what Skrin knows about every note in the vault
// (links, headings, aliases, block ids) and resolves links the way Obsidian
// does.
package index

import (
	"path"
	"sort"
	"strings"

	"github.com/lurioso/skrin/internal/vault"
)

// Heading is one heading in a note.
type Heading struct {
	Level int
	Text  string
	Line  int // 0-based source line
}

// Link is one link in a note: a wikilink, or a markdown link to a file in
// the vault.
type Link struct {
	Target   string // note name or path as written, without #sub and alias
	Sub      string // heading, or ^block id
	Alias    string // wikilink alias, or a markdown link's text
	Sep      string // wikilink alias separator: "|" or, in tables, `\|`
	HasAlias bool
	Embed    bool
	Markdown bool
	Line     int    // 0-based source line
	Start    int    // byte offsets of the whole link in the note
	End      int    //
	Context  string // the line the link is on, trimmed
}

// Backlink is a link from Source to Target.
type Backlink struct {
	Source string
	Target string
	Link   Link
}

type entry struct {
	note
	mod  int64
	size int64
}

// Index is the vault's link and heading index.
type Index struct {
	notes  map[string]*entry   // markdown notes by vault path
	paths  map[string]string   // lower-cased vault path → vault path, for every file
	byName map[string][]string // lower-cased file name → vault paths
}

// New returns an empty index; call Update to fill it.
func New() *Index {
	return &Index{notes: map[string]*entry{}, paths: map[string]string{}, byName: map[string][]string{}}
}

// Update rescans the vault, re-reading only notes that changed.
func (x *Index) Update(v *vault.Vault) error {
	infos, err := v.FileInfos()
	if err != nil {
		return err
	}
	notes := make(map[string]*entry, len(x.notes))
	paths := make(map[string]string, len(infos))
	byName := make(map[string][]string, len(infos))
	for _, f := range infos {
		paths[strings.ToLower(f.Rel)] = f.Rel
		name := strings.ToLower(path.Base(f.Rel))
		byName[name] = append(byName[name], f.Rel)
		if !vault.IsNote(f.Rel) {
			continue
		}
		mod := f.ModTime.UnixNano()
		if old := x.notes[f.Rel]; old != nil && old.mod == mod && old.size == f.Size {
			notes[f.Rel] = old
			continue
		}
		content, err := v.Read(f.Rel)
		if err != nil {
			continue
		}
		notes[f.Rel] = &entry{note: parse(content), mod: mod, size: f.Size}
	}
	x.notes, x.paths, x.byName = notes, paths, byName
	return nil
}

// Notes lists every note, sorted.
func (x *Index) Notes() []string {
	out := make([]string, 0, len(x.notes))
	for rel := range x.notes {
		out = append(out, rel)
	}
	sort.Strings(out)
	return out
}

// Aliases are a note's aliases property.
func (x *Index) Aliases(rel string) []string {
	if n := x.notes[rel]; n != nil {
		return n.aliases
	}
	return nil
}

// Headings are a note's headings, in order.
func (x *Index) Headings(rel string) []Heading {
	if n := x.notes[rel]; n != nil {
		return n.headings
	}
	return nil
}

// Anchor finds the line of a #heading or #^block in a note. A nested
// heading path like "Day#Morning" matches its last part.
func (x *Index) Anchor(rel, sub string) (int, bool) {
	n := x.notes[rel]
	if n == nil || sub == "" {
		return 0, false
	}
	if id, ok := strings.CutPrefix(sub, "^"); ok {
		line, found := n.blocks[id]
		return line, found
	}
	if i := strings.LastIndex(sub, "#"); i >= 0 {
		sub = sub[i+1:]
	}
	want := normHeading(sub)
	for _, h := range n.headings {
		if normHeading(h.Text) == want {
			return h.Line, true
		}
	}
	return 0, false
}

func normHeading(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// NameCount is how many files in the vault share a file name.
func (x *Index) NameCount(name string) int { return len(x.byName[strings.ToLower(name)]) }

// Resolve finds the file a wikilink target points to, as seen from the
// note at from. Like Obsidian it tries the vault path, then a path relative
// to the linking note, then the file name. Among several files with that
// name, the one closest to the linking note wins.
func (x *Index) Resolve(target, from string) (string, bool) {
	t := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(target), "/"))
	if t == "" {
		_, ok := x.notes[from]
		return from, ok
	}
	for _, p := range []string{t, t + ".md"} {
		if rel, ok := x.paths[p]; ok {
			return rel, true
		}
	}
	if dir := path.Dir(strings.ToLower(from)); dir != "." {
		for _, p := range []string{path.Join(dir, t), path.Join(dir, t) + ".md"} {
			if rel, ok := x.paths[p]; ok {
				return rel, true
			}
		}
	}
	var cands []string
	for _, name := range []string{path.Base(t), path.Base(t) + ".md"} {
		for _, rel := range x.byName[name] {
			// "folder/note" must match the end of the path.
			lr := strings.ToLower(rel)
			if strings.Contains(t, "/") && !strings.HasSuffix(lr, "/"+t) && !strings.HasSuffix(lr, "/"+t+".md") {
				continue
			}
			cands = append(cands, rel)
		}
	}
	if len(cands) == 0 {
		return "", false
	}
	fromDir := path.Dir(from)
	sort.Slice(cands, func(i, j int) bool {
		a, b := cands[i], cands[j]
		if (path.Dir(a) == fromDir) != (path.Dir(b) == fromDir) {
			return path.Dir(a) == fromDir
		}
		if da, db := strings.Count(a, "/"), strings.Count(b, "/"); da != db {
			return da < db
		}
		return a < b
	})
	return cands[0], true
}

// ResolveLink resolves any link found in the note at from.
func (x *Index) ResolveLink(l Link, from string) (string, bool) {
	if !l.Markdown {
		return x.Resolve(l.Target, from)
	}
	t := strings.TrimPrefix(l.Target, "./")
	if t == "" {
		return from, true
	}
	var tries []string
	if dir := path.Dir(from); dir != "." && !strings.HasPrefix(t, "/") {
		tries = append(tries, path.Clean(path.Join(dir, t)))
	}
	tries = append(tries, path.Clean(strings.TrimPrefix(t, "/")))
	for _, p := range tries {
		for _, c := range []string{p, p + ".md"} {
			if rel, ok := x.paths[strings.ToLower(c)]; ok {
				return rel, true
			}
		}
	}
	return "", false
}

// Backlinks lists the links from other notes to rel.
func (x *Index) Backlinks(rel string) []Backlink {
	return x.linksTo(func(target string) bool { return target == rel }, rel)
}

// Referrers lists every link that leads to rel or, when rel is a folder,
// to anything inside it.
func (x *Index) Referrers(rel string, isDir bool) []Backlink {
	if !isDir {
		return x.linksTo(func(t string) bool { return t == rel }, "")
	}
	return x.linksTo(func(t string) bool { return strings.HasPrefix(t, rel+"/") }, "")
}

func (x *Index) linksTo(match func(string) bool, skip string) []Backlink {
	var out []Backlink
	for _, src := range x.Notes() {
		if src == skip {
			continue
		}
		for _, l := range x.notes[src].links {
			if t, ok := x.ResolveLink(l, src); ok && match(t) {
				out = append(out, Backlink{Source: src, Target: t, Link: l})
			}
		}
	}
	return out
}

// LinkText is what a link to rel from the note at from should say, in
// Obsidian's newLinkFormat: "shortest" (the file name when that is unique),
// "relative" or "absolute". Notes lose their .md.
func (x *Index) LinkText(rel, from, format string) string {
	full := rel
	if vault.IsNote(rel) {
		full = strings.TrimSuffix(rel, path.Ext(rel))
	}
	switch format {
	case "absolute":
		return full
	case "relative":
		return RelPath(path.Dir(from), full)
	}
	if x.NameCount(path.Base(rel)) <= 1 {
		return path.Base(full)
	}
	return full
}

// RelPath is the path to `to` from folder dir, both vault-relative.
func RelPath(dir, to string) string {
	if dir == "." || dir == "" {
		return to
	}
	from, target := strings.Split(dir, "/"), strings.Split(to, "/")
	common := 0
	for common < len(from) && common < len(target)-1 && from[common] == target[common] {
		common++
	}
	up := strings.Repeat("../", len(from)-common)
	return up + strings.Join(target[common:], "/")
}

// Edit replaces one link's target; heading, alias and form are kept.
type Edit struct {
	Link   Link
	Target string // new wikilink target, or new markdown path (with extension)
}

// Apply rewrites the given links in a note's content.
func Apply(content string, edits []Edit) string {
	sort.Slice(edits, func(i, j int) bool { return edits[i].Link.Start > edits[j].Link.Start })
	for _, e := range edits {
		if e.Link.Start < 0 || e.Link.End > len(content) || e.Link.Start > e.Link.End {
			continue
		}
		content = content[:e.Link.Start] + render(e) + content[e.Link.End:]
	}
	return content
}

func render(e Edit) string {
	l := e.Link
	bang := ""
	if l.Embed {
		bang = "!"
	}
	sub := ""
	if l.Sub != "" {
		sub = "#" + l.Sub
	}
	if l.Markdown {
		return bang + "[" + l.Alias + "](" + strings.ReplaceAll(e.Target+sub, " ", "%20") + ")"
	}
	alias := ""
	if l.HasAlias {
		alias = l.Sep + l.Alias
	}
	return bang + "[[" + e.Target + sub + alias + "]]"
}
