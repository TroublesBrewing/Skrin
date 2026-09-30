package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lurioso/skrin/internal/frontmatter"
	"github.com/lurioso/skrin/internal/vault"
	"github.com/lurioso/skrin/internal/version"
)

// propRE is a property at the top level of the frontmatter: a name, a
// colon, and whatever follows on the same line. An indented line belongs
// to the property above it, so it never matches.
var propRE = regexp.MustCompile(`^([A-Za-z0-9_][A-Za-z0-9_ .\-/]*):(.*)$`)

// property is one entry in a note's frontmatter, with the lines it owns.
// Every edit is made on those lines and nowhere else, so anything Skrin
// doesn't understand — a comment, a nested map, an anchor — comes out of a
// write byte for byte as it went in.
type property struct {
	key      string
	value    string // the text after the colon, trimmed
	from, to int    // the lines it owns, inclusive
	// editable is true for a plain one-line property. A block list or a
	// nested map is shown and can be removed, but Skrin will not rewrite
	// it from a single field: there is no way to put it back wrong.
	editable bool
	what     string // what it is, for a row that can't be edited here
}

// propertiesOn is the feature's real state: a beta feature needs the
// build to allow beta, beta mode to be on, and its own switch on.
func (m *Model) propertiesOn() bool {
	return version.Beta && m.opts.Beta && m.opts.Properties
}

// readProperties reads a note's frontmatter into properties. Lines it
// can't read are left out of the list and untouched by every write.
func readProperties(lines []string) []property {
	end := frontmatter.End(lines)
	if end == 0 {
		return nil
	}
	var out []property
	for i := 1; i < end; i++ {
		mt := propRE.FindStringSubmatch(strings.TrimRight(lines[i], " \r"))
		if mt == nil {
			continue // indented, blank, a comment: not ours to touch
		}
		p := property{key: mt[1], value: strings.TrimSpace(mt[2]), from: i, to: i, editable: true}
		if p.value == "" {
			// What belongs to it is what is indented under it, or a
			// bullet: YAML allows a list at column 0. Nothing else does,
			// and that matters — a comment swallowed by the property
			// above would be deleted along with it, which is the one
			// thing this panel must never do.
			for j := i + 1; j < end; j++ {
				t := strings.TrimRight(lines[j], " \r")
				indented := t != strings.TrimLeft(t, " \t")
				bullet := strings.HasPrefix(strings.TrimLeft(t, " \t"), "- ")
				if t == "" || !(indented || bullet) {
					break
				}
				p.to = j
			}
			if p.to > p.from {
				p.editable, p.what = false, "list"
			} else {
				p.what = "empty"
			}
		}
		out = append(out, p)
		i = p.to
	}
	return out
}

// propertyLines rebuilds a note with lines[from:to] replaced by repl.
func propertyLines(lines []string, from, to int, repl []string) string {
	out := make([]string, 0, len(lines)+len(repl))
	out = append(out, lines[:from]...)
	out = append(out, repl...)
	out = append(out, lines[to+1:]...)
	return strings.Join(out, "\n")
}

// showProperties lists the open note's properties, Obsidian's properties
// pane done with what Skrin already has: the chooser lists them, Enter
// edits one in the ordinary name prompt, d removes one, and the first row
// adds one. No new overlay, no new key — the point is a field to type in,
// not a new place to learn.
func (m *Model) showProperties() {
	if !m.propertiesOn() {
		m.flash = "Properties is off: it's a beta feature, switched on in Settings (" + note(m.keyFor(inMain, actHelp), "?") + " then tab)"
		return
	}
	rel, ok := m.subjectOpen()
	if !ok {
		m.flash = "Open a note to see its properties"
		return
	}
	text, err := m.propertySource(rel)
	if err != nil {
		m.flash = err.Error()
		return
	}
	props := readProperties(strings.Split(text, "\n"))
	c := &chooser{
		title:  fmt.Sprintf("Properties of %s (%d)", displayName(rel), len(props)),
		prompt: "filter", empty: "No property by that name", verb: "edit",
		remove: func(ch choice) { m.removeProperty(rel, ch.label) },
	}
	c.items = append(c.items, choice{
		label:  "+ add a property",
		detail: "a name and a value",
		do: func() {
			m.prompt = &prompt{kind: promptPropNew, label: "New property in " + displayName(rel) + ", as name: value", target: rel}
		},
	})
	for _, p := range props {
		p := p
		it := choice{label: p.key, detail: p.value}
		switch {
		case !p.editable:
			it.detail = p.what + " — edit it in the note itself"
			it.do = func() {
				m.flash = p.key + " is a " + p.what + ": edit it in the note, so nothing of it is lost on the way"
			}
		default:
			it.do = func() {
				pr := &prompt{kind: promptPropValue, label: p.key + " in " + displayName(rel), target: rel, propKey: p.key}
				pr.in.set(p.value)
				m.prompt = pr
			}
		}
		c.items = append(c.items, it)
	}
	m.openChooser(c)
}

// propertySource is the note's text as it stands now: the editor's buffer
// when that note is open for editing, otherwise the file.
func (m *Model) propertySource(rel string) (string, error) {
	if m.editor != nil && m.edit.rel == rel {
		return m.editor.Text(), nil
	}
	return m.vault.Read(rel)
}

// writeProperties puts text back where propertySource took it from: into
// the editor's buffer, where it rides the ordinary autosave and snapshot,
// or to the file through the same read-check-snapshot-journal path every
// write outside the editor uses.
func (m *Model) writeProperties(rel, before, text, desc string) error {
	if m.editor != nil && m.edit.rel == rel {
		row, col := m.editor.Cursor()
		m.editor.Rewrite(text, row, col)
		return nil
	}
	cur, err := m.vault.Read(rel)
	if err != nil {
		return err
	}
	if cur != before {
		return fmt.Errorf("%s changed on disk · nothing written", displayName(rel))
	}
	_ = m.snaps.Save(rel, cur) // snapshots are what make u safe
	if err := m.vault.Write(rel, text); err != nil {
		return err
	}
	m.journal.Record(vault.Op{
		Desc:  desc,
		Steps: []vault.Step{{Kind: vault.StepModified, Rel: rel, Content: cur}},
	})
	m.refresh()
	return nil
}

// setProperty writes a new value for one property.
func (m *Model) setProperty(rel, key, value string) error {
	before, err := m.propertySource(rel)
	if err != nil {
		return err
	}
	lines := strings.Split(before, "\n")
	for _, p := range readProperties(lines) {
		if p.key != key {
			continue
		}
		if !p.editable {
			return fmt.Errorf("%s is a %s: edit it in the note", key, p.what)
		}
		text := propertyLines(lines, p.from, p.to, []string{key + ": " + value})
		if err := m.writeProperties(rel, before, text, "set "+key+" in "+displayName(rel)); err != nil {
			return err
		}
		m.flash = key + " is now " + value
		return nil
	}
	return fmt.Errorf("%s has no property %s any more", displayName(rel), key)
}

// removeProperty takes a property out, with the lines it owns.
func (m *Model) removeProperty(rel, key string) {
	before, err := m.propertySource(rel)
	if err != nil {
		m.flash = err.Error()
		return
	}
	lines := strings.Split(before, "\n")
	props := readProperties(lines)
	for _, p := range props {
		if p.key != key {
			continue
		}
		from, to := p.from, p.to
		// The last property leaves an empty frontmatter block behind, and
		// an empty block is noise in every reader. It goes too — but only
		// when nothing else is left inside it, comments included.
		if len(props) == 1 {
			end := frontmatter.End(lines)
			bare := true
			for i := 1; i < end; i++ {
				if (i < from || i > to) && strings.TrimSpace(lines[i]) != "" {
					bare = false
				}
			}
			if bare {
				from, to = 0, end
			}
		}
		text := propertyLines(lines, from, to, nil)
		if err := m.writeProperties(rel, before, text, "remove "+key+" from "+displayName(rel)); err != nil {
			m.flash = err.Error()
			return
		}
		m.flash = key + " removed · U undoes"
		if m.editor != nil && m.edit.rel == rel {
			m.flash = key + " removed"
		}
		m.showProperties() // the list, minus the one that's gone
		return
	}
	m.flash = displayName(rel) + " has no property " + key + " any more"
}

// addProperty writes a new property into the frontmatter, making the block
// if the note hasn't got one. Obsidian only reads frontmatter on the very
// first line, so that is where a new block goes.
func (m *Model) addProperty(rel, line string) error {
	key, value, ok := strings.Cut(line, ":")
	key = strings.TrimSpace(key)
	if !ok || key == "" {
		return fmt.Errorf("write it as name: value")
	}
	if !propRE.MatchString(key + ":") {
		return fmt.Errorf("%q can't be a property name", key)
	}
	before, err := m.propertySource(rel)
	if err != nil {
		return err
	}
	lines := strings.Split(before, "\n")
	for _, p := range readProperties(lines) {
		if p.key == key {
			return fmt.Errorf("%s already has %s", displayName(rel), key)
		}
	}
	entry := key + ": " + strings.TrimSpace(value)
	var text string
	if end := frontmatter.End(lines); end == 0 {
		text = strings.Join(append([]string{"---", entry, "---"}, lines...), "\n")
	} else {
		text = propertyLines(lines, end, end, []string{entry, lines[end]})
	}
	if err := m.writeProperties(rel, before, text, "add "+key+" to "+displayName(rel)); err != nil {
		return err
	}
	m.flash = key + " added"
	return nil
}
