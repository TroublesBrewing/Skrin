package spread

import (
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type vkind int

const (
	vNull vkind = iota
	vBool
	vNum
	vDate
	vText
	vLink
	vList
)

// value is a field's value. Properties are stored as text, so each one is
// coerced once: a number if it reads as one, a date if it's ISO, a link if
// it's exactly one wikilink, and text otherwise.
type value struct {
	k   vkind
	b   bool
	n   float64
	t   time.Time
	s   string // text; for a link, the note it leads to (or "?target" when it leads nowhere)
	raw string // how it shows in a table or list
	l   []value
}

var (
	numRE  = regexp.MustCompile(`^[-+]?(\d+\.?\d*|\.\d+)$`)
	linkRE = regexp.MustCompile(`^\[\[([^\]|#]*)(#[^\]|]*)?(\|[^\]]*)?\]\]$`)
)

var dateLayouts = []string{
	"2006-01-02", "2006-01-02T15:04", "2006-01-02T15:04:05", "2006-01-02 15:04",
	"2006-01-02 15:04:05", time.RFC3339,
}

func text(s string) value { return value{k: vText, s: s, raw: s} }

// coerce reads a value as written. A link's target is resolved later, by
// whoever knows which note it was written in.
func coerce(raw string) value {
	s := strings.TrimSpace(raw)
	switch {
	case s == "":
		return value{}
	case strings.EqualFold(s, "true"), strings.EqualFold(s, "false"):
		return value{k: vBool, b: strings.EqualFold(s, "true"), raw: s}
	case numRE.MatchString(s):
		n, _ := strconv.ParseFloat(s, 64)
		return value{k: vNum, n: n, raw: s}
	}
	for _, l := range dateLayouts {
		if t, err := time.ParseInLocation(l, s, time.Local); err == nil {
			return value{k: vDate, t: t, raw: s}
		}
	}
	if m := linkRE.FindStringSubmatch(s); m != nil {
		return value{k: vLink, s: strings.TrimSpace(m[1]), raw: s}
	}
	return text(s)
}

// ctx is one run of a spread.
type ctx struct {
	v       Vault
	from    string
	linksTo map[string]map[string]bool // note → the notes linking to it
	linksOf map[string]map[string]bool // note → the notes it links to
}

// resolve points a link value at the note it leads to, from note owner.
func (c *ctx) resolve(v value, owner string) value {
	if v.k != vLink {
		return v
	}
	if rel, ok := c.v.Resolve(v.s, owner); ok {
		v.s = rel
	} else {
		v.s = "?" + strings.ToLower(v.s)
	}
	return v
}

func set(rels []string) map[string]bool {
	m := make(map[string]bool, len(rels))
	for _, r := range rels {
		m[r] = true
	}
	return m
}

// --- expressions ----------------------------------------------------------

type expr interface{ eval(c *ctx, n *Note) value }

type lit struct{ v value }

func (e lit) eval(*ctx, *Note) value { return e.v }

// linkLit is a [[link]] written in the spread: it leads from the note the
// spread is in.
type linkLit struct{ target string }

func (e linkLit) eval(c *ctx, _ *Note) value {
	return c.resolve(value{k: vLink, s: e.target, raw: "[[" + e.target + "]]"}, c.from)
}

type fieldRef struct{ name string }

func (e fieldRef) eval(c *ctx, n *Note) value {
	switch e.name {
	case "file.name":
		return text(noteName(n.Rel))
	case "file.link":
		return value{k: vLink, s: n.Rel, raw: linkTo(n.Rel)}
	case "file.path":
		return text(n.Rel)
	case "file.folder":
		if d := path.Dir(n.Rel); d != "." {
			return text(d)
		}
		return text("")
	case "file.tags":
		return tagList(n.Tags)
	case "file.mtime":
		return value{k: vDate, t: n.Mod, raw: n.Mod.Format("2006-01-02 15:04")}
	case "file.size":
		return value{k: vNum, n: float64(n.Size), raw: strconv.FormatInt(n.Size, 10)}
	}
	vals := n.Props[e.name]
	switch len(vals) {
	case 0:
		return value{}
	case 1:
		return c.resolve(coerce(vals[0]), n.Rel)
	}
	l := value{k: vList, raw: strings.Join(vals, ", ")}
	for _, s := range vals {
		l.l = append(l.l, c.resolve(coerce(s), n.Rel))
	}
	return l
}

// tagList is file.tags: every tag with its '#', and each parent of a
// nested tag too, the way Dataview lists them — so #books/fiction is also
// #books.
func tagList(tags []string) value {
	seen := map[string]bool{}
	var out []string
	for _, t := range tags {
		parts := strings.Split(t, "/")
		for i := range parts {
			if p := "#" + strings.Join(parts[:i+1], "/"); !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	sort.Strings(out)
	l := value{k: vList, raw: strings.Join(out, ", ")}
	for _, t := range out {
		l.l = append(l.l, text(t))
	}
	return l
}

type cmpE struct {
	op   string
	a, b expr
}

func (e cmpE) eval(c *ctx, n *Note) value {
	d, ok := compare(e.a.eval(c, n), e.b.eval(c, n))
	var r bool
	switch e.op {
	case "=":
		r = ok && d == 0
	case "!=":
		r = !(ok && d == 0)
	case "<":
		r = ok && d < 0
	case "<=":
		r = ok && d <= 0
	case ">":
		r = ok && d > 0
	case ">=":
		r = ok && d >= 0
	}
	return boolean(r)
}

type andE struct{ a, b expr }

func (e andE) eval(c *ctx, n *Note) value {
	return boolean(truthy(e.a.eval(c, n)) && truthy(e.b.eval(c, n)))
}

type orE struct{ a, b expr }

func (e orE) eval(c *ctx, n *Note) value {
	return boolean(truthy(e.a.eval(c, n)) || truthy(e.b.eval(c, n)))
}

type notE struct{ a expr }

func (e notE) eval(c *ctx, n *Note) value { return boolean(!truthy(e.a.eval(c, n))) }

type containsE struct {
	a, b expr
	fold bool
}

func (e containsE) eval(c *ctx, n *Note) value {
	return boolean(contains(e.a.eval(c, n), e.b.eval(c, n), e.fold))
}

func boolean(b bool) value { return value{k: vBool, b: b, raw: strconv.FormatBool(b)} }

func truthy(v value) bool {
	switch v.k {
	case vNull:
		return false
	case vBool:
		return v.b
	case vNum:
		return v.n != 0
	case vText:
		return v.s != ""
	case vList:
		return len(v.l) > 0
	}
	return true
}

// compare orders two values of the same kind, and reports false when they
// can't be compared. null only equals null. Text meeting a number or a
// date is read again as one — `due < "2026-10-01"` means what it says.
func compare(a, b value) (int, bool) {
	if a.k == vNull || b.k == vNull {
		return 0, a.k == b.k
	}
	if a.k != b.k {
		if a.k == vText {
			a = coerce(a.s)
		}
		if b.k == vText {
			b = coerce(b.s)
		}
		if a.k != b.k {
			return 0, false
		}
	}
	switch a.k {
	case vBool:
		switch {
		case a.b == b.b:
			return 0, true
		case b.b:
			return -1, true
		}
		return 1, true
	case vNum:
		switch {
		case a.n < b.n:
			return -1, true
		case a.n > b.n:
			return 1, true
		}
		return 0, true
	case vDate:
		return a.t.Compare(b.t), true
	case vText, vLink:
		return strings.Compare(a.s, b.s), true
	}
	return 0, false
}

// contains is Dataview's: a list contains a value equal to v; text
// contains v as a substring. fold ignores case (icontains).
func contains(a, v value, fold bool) bool {
	switch a.k {
	case vList:
		for _, e := range a.l {
			if fold && e.k == vText && v.k == vText {
				if strings.EqualFold(e.s, v.s) {
					return true
				}
			} else if d, ok := compare(e, v); ok && d == 0 {
				return true
			}
		}
	case vText:
		if v.k != vText {
			return false
		}
		if fold {
			return strings.Contains(strings.ToLower(a.s), strings.ToLower(v.s))
		}
		return strings.Contains(a.s, v.s)
	case vLink:
		d, ok := compare(a, v)
		return ok && d == 0
	}
	return false
}

// --- FROM -----------------------------------------------------------------

type source interface{ match(c *ctx, n *Note) bool }

type tagS struct{ tag string }

func (s tagS) match(_ *ctx, n *Note) bool {
	for _, t := range n.Tags {
		if t == s.tag || strings.HasPrefix(t, s.tag+"/") {
			return true
		}
	}
	return false
}

// folderS is a folder, or one note named by its path without ".md".
// "" is the whole vault.
type folderS struct{ dir string }

func (s folderS) match(_ *ctx, n *Note) bool {
	return s.dir == "" || strings.HasPrefix(n.Rel, s.dir+"/") || n.Rel == s.dir+".md" || n.Rel == s.dir
}

// linksToS is [[note]]: the notes that link to it.
type linksToS struct{ target string }

func (s linksToS) match(c *ctx, n *Note) bool {
	rel, ok := c.v.Resolve(s.target, c.from)
	if !ok {
		return false
	}
	if c.linksTo[rel] == nil {
		c.linksTo[rel] = set(c.v.Backlinks(rel))
	}
	return c.linksTo[rel][n.Rel]
}

// linksFromS is outgoing([[note]]): the notes it links to.
type linksFromS struct{ target string }

func (s linksFromS) match(c *ctx, n *Note) bool {
	rel, ok := c.v.Resolve(s.target, c.from)
	if !ok {
		return false
	}
	if c.linksOf[rel] == nil {
		c.linksOf[rel] = set(c.v.Outgoing(rel))
	}
	return c.linksOf[rel][n.Rel]
}

type notS struct{ s source }

func (s notS) match(c *ctx, n *Note) bool { return !s.s.match(c, n) }

type andS struct{ a, b source }

func (s andS) match(c *ctx, n *Note) bool { return s.a.match(c, n) && s.b.match(c, n) }

type orS struct{ a, b source }

func (s orS) match(c *ctx, n *Note) bool { return s.a.match(c, n) || s.b.match(c, n) }

// --- running it -----------------------------------------------------------

type row struct {
	rel  string
	vals []value // the fields, in order
	keys []value // the sort keys, in order
}

func (q *Query) eval(v Vault, from string) ([]row, error) {
	c := &ctx{v: v, from: from, linksTo: map[string]map[string]bool{}, linksOf: map[string]map[string]bool{}}
	notes := v.Notes()
	sort.Slice(notes, func(i, j int) bool { return notes[i].Rel < notes[j].Rel })
	var rows []row
	for i := range notes {
		n := &notes[i]
		if q.from != nil && !q.from.match(c, n) {
			continue
		}
		if q.where != nil && !truthy(q.where.eval(c, n)) {
			continue
		}
		r := row{rel: n.Rel}
		for _, f := range q.fields {
			r.vals = append(r.vals, f.e.eval(c, n))
		}
		for _, k := range q.sort {
			r.keys = append(r.keys, k.e.eval(c, n))
		}
		rows = append(rows, r)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		for k, key := range q.sort {
			if d := order(rows[i].keys[k], rows[j].keys[k], key.desc); d != 0 {
				return d < 0
			}
		}
		return false
	})
	if q.Limit >= 0 && len(rows) > q.Limit {
		rows = rows[:q.Limit]
	}
	return rows, nil
}

// order sorts two keys. A missing value goes last whichever way the sort
// runs; values that can't be compared keep a fixed order by kind.
func order(a, b value, desc bool) int {
	switch {
	case a.k == vNull && b.k == vNull:
		return 0
	case a.k == vNull:
		return 1
	case b.k == vNull:
		return -1
	}
	d, ok := compare(a, b)
	if !ok {
		d = int(a.k) - int(b.k)
	}
	if desc {
		d = -d
	}
	return d
}

// --- the answer, as markdown --------------------------------------------

func noteName(rel string) string { return strings.TrimSuffix(path.Base(rel), ".md") }

// linkTo is a wikilink to the note at rel, by its full path so it can't
// lead anywhere else, showing just its name.
func linkTo(rel string) string {
	return "[[" + strings.TrimSuffix(rel, ".md") + "|" + noteName(rel) + "]]"
}

func display(v value) string {
	if v.k == vNull {
		return ""
	}
	return strings.ReplaceAll(v.raw, "\n", " ")
}

func (q *Query) markdown(rows []row) string {
	var b strings.Builder
	if q.kind == kList {
		for _, r := range rows {
			b.WriteString("- " + linkTo(r.rel))
			if len(r.vals) > 0 {
				if s := display(r.vals[0]); s != "" {
					b.WriteString(": " + s)
				}
			}
			b.WriteByte('\n')
		}
		return b.String()
	}
	var head []string
	if !q.noID {
		head = append(head, "Note")
	}
	for _, f := range q.fields {
		head = append(head, f.name)
	}
	cellRow := func(cells []string) {
		for i, c := range cells {
			cells[i] = strings.ReplaceAll(c, "|", `\|`)
		}
		b.WriteString("| " + strings.Join(cells, " | ") + " |\n")
	}
	cellRow(head)
	b.WriteString("|" + strings.Repeat(" --- |", len(head)) + "\n")
	for _, r := range rows {
		var cells []string
		if !q.noID {
			cells = append(cells, linkTo(r.rel))
		}
		for _, v := range r.vals {
			cells = append(cells, display(v))
		}
		cellRow(cells)
	}
	return b.String()
}
