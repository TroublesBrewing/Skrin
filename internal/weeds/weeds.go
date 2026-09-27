// Package weeds finds what has grown in a vault where nobody wanted it:
// links that lead nowhere, notes nothing points at, notes with a title and
// no note, one name on two files, a tag used once, and notes nobody has
// opened in half a year. Each of those is invisible from inside a single
// note — you only meet them by accident, months later, when a link you
// were sure of turns out to be dimmed.
//
// The user's own rule names the idea: "anything growing where you don't
// want things is a weed, whether it's a rose or a dandelion."
//
// Nothing here reads the disk or changes anything. The caller hands over
// the vault as its index already knows it and gets back a list to show,
// which is what keeps the checks testable without a vault and cheap
// enough to run on a keypress.
package weeds

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"time"
)

// Kind is one sort of loose end. The order is the order the groups are
// shown in: what is broken first, what is merely untidy after.
type Kind int

const (
	DeadLink  Kind = iota // a link whose target isn't there
	Unlinked              // nothing links here, and it links nowhere
	Stub                  // a title and next to no note
	SameName              // two notes with one file name
	LoneTag               // a tag only one note uses
	Untouched             // nobody has opened it in a long time
)

// Defaults for Options. They are values a note-taker would recognise
// rather than round numbers: half a year is long enough that a note has
// genuinely slipped out of mind, and 80 characters is about a title and a
// first thought.
const (
	DefaultUntouched = 182 * 24 * time.Hour
	DefaultStubUnder = 80
	DefaultMax       = 40
)

// Link is one link in a note, as the checks need it.
type Link struct {
	Target string // as written, without #heading or |alias
	Line   int    // 0-based source line
	Dead   bool   // resolves to nothing in the vault
	Embed  bool   // ![[…]] rather than [[…]]
}

// Note is one note as the checks look at it.
type Note struct {
	Rel      string
	Body     string    // the text with frontmatter and headings taken off
	Modified time.Time //
	Links    []Link
	// Linked is how many notes link here. The caller counts it in one
	// pass over the vault rather than asking per note, which would walk
	// every link once per note.
	Linked int
	Tags   []string // as this note spells them, without '#', each once
}

// Options tune the checks. The zero value uses the defaults above.
type Options struct {
	Now       time.Time
	Untouched time.Duration
	StubUnder int
	Max       int // most items shown per group; the group still counts the rest
	// Skip are folders left out of the checks that judge a note as a
	// whole: a daily note is meant to stand alone and is never
	// "forgotten", so a journal folder here keeps a year of days from
	// burying everything else. A dead link is still reported, wherever
	// it is.
	Skip []string
}

func (o Options) with() Options {
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	if o.Untouched <= 0 {
		o.Untouched = DefaultUntouched
	}
	if o.StubUnder <= 0 {
		o.StubUnder = DefaultStubUnder
	}
	if o.Max <= 0 {
		o.Max = DefaultMax
	}
	return o
}

// Item is one loose end: where it is, and what to say about it.
type Item struct {
	Kind Kind
	Rel  string // the note to open
	Line int    // 0-based line, or -1 when the item is the note itself
	What string // the missing target, the tag, the shared name
	Note string // the row's own words: context, age, size
}

// Group is one kind of loose end. Items is capped at Options.Max; Total
// says how many there really are, so a capped group can say so instead of
// quietly lying.
type Group struct {
	Kind  Kind
	Title string
	Hint  string
	Items []Item
	Total int
}

// Find runs every check over the vault. Groups with nothing in them are
// left out, so an empty answer means a tidy vault.
func Find(notes []Note, o Options) []Group {
	o = o.with()
	var out []Group
	for _, g := range []Group{
		deadLinks(notes),
		unlinked(notes, o),
		stubs(notes, o),
		sameName(notes),
		loneTags(notes),
		untouched(notes, o),
	} {
		if g.Total == 0 {
			continue
		}
		g.Total = len(g.Items)
		if len(g.Items) > o.Max {
			g.Items = g.Items[:o.Max]
		}
		out = append(out, g)
	}
	return out
}

// Count is every group's total added up: what the title says.
func Count(groups []Group) int {
	n := 0
	for _, g := range groups {
		n += g.Total
	}
	return n
}

func deadLinks(notes []Note) Group {
	g := Group{
		Kind:  DeadLink,
		Title: "Links that lead nowhere",
		Hint:  "the note isn't there · n makes it",
	}
	for _, n := range notes {
		for _, l := range n.Links {
			if !l.Dead {
				continue
			}
			what := l.Target
			kind := "link"
			if l.Embed {
				kind = "embed"
			}
			g.Items = append(g.Items, Item{
				Kind: DeadLink,
				Rel:  n.Rel,
				Line: l.Line,
				What: what,
				Note: fmt.Sprintf("%s · %s line %d", kind, short(n.Rel), l.Line+1),
			})
		}
	}
	sortItems(g.Items)
	g.Total = len(g.Items)
	return g
}

func unlinked(notes []Note, o Options) Group {
	g := Group{
		Kind:  Unlinked,
		Title: "Notes on their own",
		Hint:  "nothing links here, and they link nowhere",
	}
	for _, n := range notes {
		if skipped(n.Rel, o.Skip) || n.Linked > 0 || n.alive() {
			continue
		}
		g.Items = append(g.Items, Item{
			Kind: Unlinked,
			Rel:  n.Rel,
			Line: -1,
			What: short(n.Rel),
			Note: "in " + folder(n.Rel) + " · " + age(n.Modified, o.Now),
		})
	}
	sortItems(g.Items)
	g.Total = len(g.Items)
	return g
}

func stubs(notes []Note, o Options) Group {
	g := Group{
		Kind:  Stub,
		Title: "A title and no note",
		Hint:  "written down and never written",
	}
	for _, n := range notes {
		if skipped(n.Rel, o.Skip) {
			continue
		}
		body := strings.TrimSpace(n.Body)
		if len([]rune(body)) >= o.StubUnder {
			continue
		}
		note := fmt.Sprintf("%d characters", len([]rune(body)))
		if body == "" {
			note = "empty"
		}
		g.Items = append(g.Items, Item{
			Kind: Stub,
			Rel:  n.Rel,
			Line: -1,
			What: short(n.Rel),
			Note: note + " · " + age(n.Modified, o.Now),
		})
	}
	sortItems(g.Items)
	g.Total = len(g.Items)
	return g
}

func sameName(notes []Note) Group {
	g := Group{
		Kind:  SameName,
		Title: "One name, two notes",
		Hint:  "[[the name]] can only mean one of them",
	}
	byName := map[string][]string{}
	for _, n := range notes {
		name := strings.ToLower(short(n.Rel))
		byName[name] = append(byName[name], n.Rel)
	}
	for _, rels := range byName {
		if len(rels) < 2 {
			continue
		}
		sort.Strings(rels)
		folders := make([]string, 0, len(rels))
		for _, r := range rels {
			folders = append(folders, folder(r))
		}
		g.Items = append(g.Items, Item{
			Kind: SameName,
			Rel:  rels[0],
			Line: -1,
			What: short(rels[0]),
			Note: fmt.Sprintf("%d notes · %s", len(rels), strings.Join(folders, ", ")),
		})
	}
	sortItems(g.Items)
	g.Total = len(g.Items)
	return g
}

func loneTags(notes []Note) Group {
	g := Group{
		Kind:  LoneTag,
		Title: "Tags used once",
		Hint:  "a thought filed under a word only it uses",
	}
	// Two counts: how many notes write a tag exactly that way, and how
	// many write it in any case at all. A tag used once whose other
	// spelling is common is nearly always a typo, and the row says so.
	spelling := map[string][]string{}
	folded := map[string]int{}
	best := map[string]string{}
	bestN := map[string]int{}
	for _, n := range notes {
		for _, t := range n.Tags {
			spelling[t] = append(spelling[t], n.Rel)
			folded[strings.ToLower(t)]++
		}
	}
	for t, rels := range spelling {
		low := strings.ToLower(t)
		if len(rels) > bestN[low] {
			best[low], bestN[low] = t, len(rels)
		}
	}
	for t, rels := range spelling {
		if len(rels) != 1 {
			continue
		}
		low := strings.ToLower(t)
		note := "only " + short(rels[0]) + " uses it"
		if other := best[low]; other != t && bestN[low] > 1 {
			note = fmt.Sprintf("also spelled #%s, in %d notes", other, bestN[low])
		}
		g.Items = append(g.Items, Item{
			Kind: LoneTag,
			Rel:  rels[0],
			Line: -1,
			What: "#" + t,
			Note: note,
		})
	}
	sortItems(g.Items)
	g.Total = len(g.Items)
	return g
}

func untouched(notes []Note, o Options) Group {
	g := Group{
		Kind:  Untouched,
		Title: "Not touched in a long while",
		Hint:  "oldest first · still worth reading?",
	}
	cut := o.Now.Add(-o.Untouched)
	old := make([]Note, 0, len(notes))
	for _, n := range notes {
		if skipped(n.Rel, o.Skip) || n.Modified.IsZero() || n.Modified.After(cut) {
			continue
		}
		old = append(old, n)
	}
	// Oldest first, rather than by path: the point of the group is which
	// note has waited longest. Sorting the notes and then building the
	// rows keeps it one sort rather than a lookup per comparison.
	sort.SliceStable(old, func(i, j int) bool { return old[i].Modified.Before(old[j].Modified) })
	for _, n := range old {
		g.Items = append(g.Items, Item{
			Kind: Untouched,
			Rel:  n.Rel,
			Line: -1,
			What: short(n.Rel),
			Note: age(n.Modified, o.Now) + " · in " + folder(n.Rel),
		})
	}
	g.Total = len(g.Items)
	return g
}

// alive reports whether the note links anywhere that exists.
func (n Note) alive() bool {
	for _, l := range n.Links {
		if !l.Dead {
			return true
		}
	}
	return false
}

// sortItems puts a group in a stable, readable order: by note, then by
// line, so two runs of the same vault list the same things the same way.
func sortItems(items []Item) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Rel != items[j].Rel {
			return items[i].Rel < items[j].Rel
		}
		if items[i].Line != items[j].Line {
			return items[i].Line < items[j].Line
		}
		return items[i].What < items[j].What
	})
}

// skipped reports whether rel is in one of the folders left out.
func skipped(rel string, skip []string) bool {
	for _, s := range skip {
		s = strings.Trim(s, "/")
		if s == "" {
			continue
		}
		if rel == s || strings.HasPrefix(rel, s+"/") {
			return true
		}
	}
	return false
}

// short is a note's name without its folders or extension, the way a
// wikilink writes it.
func short(rel string) string {
	base := path.Base(rel)
	return strings.TrimSuffix(base, path.Ext(base))
}

// folder is the folder a note sits in, or "the root".
func folder(rel string) string {
	d := path.Dir(rel)
	if d == "." || d == "" {
		return "the root"
	}
	return d
}

// age is how long ago a note was last written, in the words someone
// would use: days up to a month, then months, then years.
func age(mod, now time.Time) string {
	if mod.IsZero() {
		return "never touched"
	}
	d := now.Sub(mod)
	switch days := int(d.Hours() / 24); {
	case days < 1:
		return "today"
	case days == 1:
		return "yesterday"
	case days < 31:
		return fmt.Sprintf("%d days ago", days)
	case days < 365:
		return plural(days/30, "month") + " ago"
	default:
		return plural(days/365, "year") + " ago"
	}
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
