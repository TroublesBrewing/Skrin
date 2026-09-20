package ui

import (
	"path"
	"strings"

	"github.com/lurioso/skrin/internal/obsidian"
	"github.com/lurioso/skrin/internal/vault"
)

// Pulling text out into a note of its own is the other half of capture:
// what was caught quickly, in a daily note or a quick note, becomes
// something that can be found. Obsidian's Note Refactor plugin does the
// same; this is the part of it that earns its place.
//
// It works on whole lines, in the editor and in the reading view alike. A
// block of text is what you pull out — half a sentence isn't a note.

// startExtract pulls the selected lines into a note of their own, named
// after their first line, and leaves a link where they were.
func (m *Model) startExtract() {
	rel, first, last, lines, ok := m.selectedLines()
	if !ok {
		m.flash = "Select the lines to pull out first: v in the note, shift+arrows in the editor"
		return
	}
	// The lines are held as they are now: what happens next — a question,
	// a name typed — must not depend on the selection still being there.
	m.pulling = &pull{rel: rel, first: first, last: last, lines: lines}
	name := noteName(lines[first : last+1])
	if name == "" {
		m.askExtractName("", "")
		return
	}
	if err := m.extractTo(name); err != nil {
		m.askExtractName(name, err.Error())
	}
}

// askExtractName asks what the new note should be called, with what went
// wrong when something did. The lines are already held, so the question
// can be answered at leisure.
func (m *Model) askExtractName(name, why string) {
	p := &prompt{kind: promptExtract, label: "Pull these lines into", err: why}
	p.in.set(name)
	m.prompt = p
}

// extractTo does the pull, into a note called name beside the one the
// lines came from. Both halves — the new note and the hole left behind —
// are one entry in the journal, so one U undoes the whole thing. That is
// the difference between a tool you use and one you daren't.
func (m *Model) extractTo(name string) error {
	p := m.pulling
	if p == nil {
		return errString("nothing is waiting to be pulled out")
	}
	rel, first, last, lines := p.rel, p.first, p.last, p.lines
	name = cleanName(name)
	if name == "" {
		return errString("a note needs a name")
	}
	target := path.Join(path.Dir(rel), name+".md")
	if target == rel {
		return errString("that is this note's own name")
	}
	if m.vault.Exists(target) {
		return errString(target + " already exists")
	}

	// With the editor open its buffer is the truth, so the file is made to
	// match before anything else: the journal has to record what was
	// really there, or U would take the new note back and leave the hole.
	if m.editor != nil {
		m.saveEdit(false)
		if m.conflict != nil {
			return errString("the note changed on disk: answer that first")
		}
	}
	before, err := m.vault.Read(rel)
	if err != nil {
		return err
	}

	body := strings.Join(lines[first:last+1], "\n")
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	dirs, err2 := m.vault.CreateFile(target, body)
	err = err2
	steps := createdSteps(dirs)
	if err != nil {
		m.journal.Record(vault.Op{Desc: "pull out " + target, Steps: steps})
		return err
	}
	steps = append(steps, vault.Step{Kind: vault.StepCreated, Rel: target, Content: body})

	// The link takes the lines' place.
	format := obsidian.LoadSettings(m.vault.Root).NewLinkFormat
	link := "[[" + m.idx.LinkText(target, rel, format) + "]]"
	after := strings.Join(append(append([]string{}, lines[:first]...), append([]string{link}, lines[last+1:]...)...), "\n")

	if err := m.snaps.Save(rel, before); err != nil {
		m.flash = "Pulled out " + name + ", but u won't undo the hole: " + err.Error()
	}
	if err := m.vault.Write(rel, after); err != nil {
		m.journal.Record(vault.Op{Desc: "pull out " + target, Steps: steps})
		return err
	}
	steps = append(steps, vault.Step{Kind: vault.StepModified, Rel: rel, Content: before})
	if m.editor != nil {
		// The editor is showing the note the hole was cut in, so it takes
		// the new text as its own — with the cursor on the link.
		m.editor.Rewrite(after, first, 0)
		m.editor.ClearSelection()
		m.edit.base = after
	}
	m.noteSel = nil
	m.journal.Record(vault.Op{Desc: "pull out " + target, Steps: steps})
	m.pulling = nil
	m.refresh()
	m.flash = "Pulled " + plural(last-first+1, "line") + " into " + name + " · U undoes it"
	return nil
}

// pull is a set of lines waiting to become a note, held from the moment
// you asked so that answering a question can't change what moves.
type pull struct {
	rel         string
	first, last int
	lines       []string
}

// selectedLines is the note the selection is in, the first and last line
// it covers, and the note's lines. Whole lines either way: in the editor
// the rows the selection touches, in the reading view the source lines
// under it.
func (m *Model) selectedLines() (rel string, first, last int, lines []string, ok bool) {
	if m.editor != nil {
		a, b, has := m.editor.SelectedRows()
		if !has {
			return "", 0, 0, nil, false
		}
		return m.edit.rel, a, b, strings.Split(m.editor.Text(), "\n"), true
	}
	s := m.noteSel
	if s == nil || m.notePath == "" || len(m.lines) == 0 {
		return "", 0, 0, nil, false
	}
	src := strings.Split(m.noteSrc, "\n")
	a := clamp(m.lines[min(s.anchor, s.cur)].Src, 0, len(src)-1)
	b := clamp(m.lines[max(s.anchor, s.cur)].Src, 0, len(src)-1)
	return m.notePath, min(a, b), max(a, b), src, true
}

// noteName is the name the pulled-out lines give themselves: their first
// line with its markup taken off. Empty when the first line has nothing
// to go on, and then Skrin asks instead of inventing a name — a note
// called "2026-09-20 1423" is one you never find again.
func noteName(lines []string) string {
	for _, l := range lines {
		t := strings.TrimSpace(l)
		t = strings.TrimLeft(t, "#>-*+ \t")
		if i := strings.Index(t, "] "); i >= 0 && i <= 3 && strings.HasPrefix(t, "[") {
			t = t[i+2:] // a task's "[ ] " or "[x] "
		}
		if n := cleanName(t); n != "" {
			return n
		}
	}
	return ""
}

// cleanName drops what a file name can't hold, the way Obsidian does, and
// keeps it to a length you can still read in a list.
func cleanName(s string) string {
	s = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`#^[]|\/:*?"<>`, r) {
			return -1
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 80 {
		s = strings.TrimSpace(s[:80])
	}
	return strings.Trim(s, ". ")
}

type errString string

func (e errString) Error() string { return string(e) }
