package ui

import (
	"math/rand"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/version"
)

// blockIDRE matches the id at the end of a block, Obsidian's way: a
// caret, then letters, digits and dashes, at the very end of the line.
var blockIDRE = regexp.MustCompile(`\s\^([A-Za-z0-9-]+)$`)

// blockIDLen is how many characters a new id gets. Six of this alphabet
// is Obsidian's own shape, and enough that two blocks in a vault are not
// going to collide by accident.
const blockIDLen = 6

const blockIDAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// blockRefOn is the feature's real state: a beta feature needs the build
// to allow beta, beta mode to be on, and its own switch on.
func (m *Model) blockRefOn() bool {
	return version.Beta && m.opts.Beta && m.opts.BlockRef
}

// newBlockID makes an id for a block that hasn't got one.
func newBlockID() string {
	b := make([]byte, blockIDLen)
	for i := range b {
		b[i] = blockIDAlphabet[rand.Intn(len(blockIDAlphabet))]
	}
	return string(b)
}

// headingRE is a markdown heading, hashes and then a space. "#tag" at
// the start of a line is a tag, not a heading, which is why the space is
// required.
var headingRE = regexp.MustCompile(`^#{1,6}\s`)

// listItemRE is a bullet or numbered list item, at any indent.
var listItemRE = regexp.MustCompile(`^\s*([-*+]\s|\d+[.)]\s)`)

// blockEnd is the last line of the block the cursor is in, which is where
// Obsidian puts the id: an anchor has to point at the whole block, not at
// the line the cursor happened to rest on.
//
// What ends a block, in Obsidian's reading and now in Skrin's: a blank
// line, a heading, or the start of the next list item. The last one is
// what makes a list useful — each item is its own block, so a link can
// point at one line of a list rather than at all of it. An item's own
// indented continuation lines are part of it.
func blockEnd(lines []string, row int) int {
	if row < 0 || row >= len(lines) || strings.TrimSpace(lines[row]) == "" {
		return -1
	}
	if headingRE.MatchString(strings.TrimSpace(lines[row])) {
		return row // its own block, and the caller turns it down
	}
	end := row
	for end+1 < len(lines) {
		nxt := lines[end+1]
		if strings.TrimSpace(nxt) == "" || headingRE.MatchString(strings.TrimSpace(nxt)) || listItemRE.MatchString(nxt) {
			break
		}
		end++
	}
	return end
}

// copyBlockLink is Obsidian's "Copy link to block": the block the cursor
// is in gets an id if it hasn't got one, and a link to it goes to the
// clipboard.
//
// The id is written into the editor's buffer rather than to the file, so
// it rides the ordinary autosave, the ordinary snapshot and the ordinary
// u — a link shouldn't be the one thing in Skrin that writes by its own
// rules.
func (m *Model) copyBlockLink() tea.Cmd {
	if !m.blockRefOn() {
		m.flash = "Block links are off: it's a beta feature, switched on in Settings (" + note(m.keyFor(inMain, actHelp), "?") + " then tab)"
		return nil
	}
	if m.editor == nil {
		m.flash = "Open the note for editing first: a block link needs a line to point at"
		return nil
	}
	row, col := m.editor.Cursor()
	lines := strings.Split(m.editor.Text(), "\n")
	end := blockEnd(lines, row)
	if end < 0 {
		m.flash = "Put the cursor in a paragraph: a blank line isn't a block"
		return nil
	}
	if headingRE.MatchString(strings.TrimSpace(lines[end])) {
		// Obsidian doesn't put block ids on headings, and it doesn't need
		// to: a heading is already an anchor of its own.
		m.flash = "A heading is already an anchor — link to it as [[note#" + strings.TrimLeft(strings.TrimSpace(lines[end]), "# ") + "]]"
		return nil
	}

	id := ""
	if mt := blockIDRE.FindStringSubmatch(lines[end]); mt != nil {
		id = mt[1] // the block already has one: the same block, the same link
	} else {
		id = newBlockID()
		lines[end] += " ^" + id
		m.editor.Rewrite(strings.Join(lines, "\n"), row, col)
	}

	link := "[[" + m.idx.LinkText(m.edit.rel, m.edit.rel, m.edit.linkFormat) + "#^" + id + "]]"
	m.copied = link
	m.flash = "Copied " + link
	return tea.SetClipboard(link)
}
