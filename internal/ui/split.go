package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/lurioso/skrin/internal/markdown"
	"github.com/lurioso/skrin/internal/vault"
)

// noteView is a note in the pane that doesn't have focus: the other half of
// a split view.
type noteView struct {
	path      string
	src       string
	err       error
	lines     []markdown.Line
	renderedW int
	off       int
}

// openSplit opens rel beside the open note, on the left or the right, and
// gives it the focus. The note that had it becomes the other pane; any
// earlier split is replaced, since two is the most there are.
func (m *Model) openSplit(rel string, left bool) {
	switch {
	case rel == "" || m.notePath == "":
		m.flash = "Select a note to split beside"
		return
	case m.width < splitMinWidth:
		m.flash = "No room to split"
		return
	}
	m.split = &noteView{path: m.notePath, src: m.noteSrc, err: m.noteErr, off: m.noteOff}
	m.splitLeft = !left
	m.showNote(rel)
	m.zen, m.focus = false, paneNote
	m.files.reveal(rel)
}

// skimSplit is Alt+↓ / Alt+↑ in Files: the cursor moves like j/k, and the
// note it lands on opens in the split beside the one already open — the
// tree's own way to say "this one, and this one beside it." Focus stays
// in Files: you're browsing, not done. Skimming with a split open
// replaces its note, so two panes is still the most there are. Folder
// rows and the reference note itself move the cursor and nothing else.
func (m *Model) skimSplit(down bool) {
	if !m.inFiles("skim") {
		return
	}
	f := &m.files
	c := step(f.cur, len(f.rows), upDown(down), m.layout().bodyH-2)
	if c == f.cur {
		return // at the top or the bottom of the tree: nothing moved
	}
	f.cur = c
	e := f.selected()
	if e.IsDir || !vault.IsNote(e.Name) || e.Rel == m.notePath {
		return
	}
	switch {
	case m.notePath == "":
		m.flash = "Select a note to split beside"
	case m.width < splitMinWidth:
		m.flash = "No room to split"
	default:
		if m.split == nil {
			m.split = &noteView{path: m.notePath, src: m.noteSrc, err: m.noteErr, off: m.noteOff}
			m.splitLeft = false
		}
		src, err := m.vault.Read(e.Rel)
		if errors.Is(err, fs.ErrNotExist) {
			m.flash = displayName(e.Rel) + " is gone"
			return
		}
		*m.split = noteView{path: e.Rel, src: src, err: err}
		m.flash = "Opened beside · Shift+→ focuses"
	}
}

// upDown turns a direction into the action step() understands.
func upDown(down bool) action {
	if down {
		return actDown
	}
	return actUp
}

// swapPanes moves the focus to the other pane of a split.
func (m *Model) swapPanes() {
	s := m.split
	cur := noteView{m.notePath, m.noteSrc, m.noteErr, m.lines, m.renderedW, m.noteOff}
	m.notePath, m.noteSrc, m.noteErr, m.lines, m.renderedW, m.noteOff = s.path, s.src, s.err, s.lines, s.renderedW, s.off
	*s = cur
	m.splitLeft = !m.splitLeft
	m.hints, m.noteSel, m.jumpSrc = nil, nil, -1
	m.files.reveal(m.notePath)
}

// closePane is Esc in a split: the focused pane closes and the other takes
// the width.
func (m *Model) closePane() {
	s, name := m.split, displayName(m.notePath)
	m.split = nil
	m.notePath, m.noteSrc, m.noteErr, m.noteOff = s.path, s.src, s.err, s.off
	m.lines, m.renderedW, m.hints, m.noteSel = nil, 0, nil, nil
	m.files.reveal(m.notePath)
	m.flash = "Closed " + name
}

// loadSplit reads the other pane's note again; if it's gone, the pane
// closes.
func (m *Model) loadSplit() {
	s := m.split
	if s == nil {
		return
	}
	src, err := m.vault.Read(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		m.split = nil
		m.flash = displayName(s.path) + " is gone, so its pane closed"
		return
	}
	s.src, s.err, s.lines, s.renderedW = src, err, nil, 0
}

// splitPane draws the other note of a split, without focus.
func (m *Model) splitPane(w, h int) []string {
	s := m.split
	var body []string
	if s.err != nil {
		body = []string{"", "  " + m.st.errText.Render("Can't read this note: "+s.err.Error())}
	}
	totalLines := 1
	if s.src != "" {
		totalLines = strings.Count(s.src, "\n") + 1
	}
	digits := max(len(strconv.Itoa(totalLines)), 2)
	for i := s.off; s.err == nil && i < min(len(s.lines), s.off+h-2); i++ {
		var prefix string
		if m.opts.LineNumbers {
			if i == 0 || s.lines[i].Src != s.lines[i-1].Src {
				prefix = m.st.muted.Render(fmt.Sprintf("%*d", digits, s.lines[i].Src+1)) + m.st.muted.Render(" │ ")
			} else {
				prefix = m.st.muted.Render(fmt.Sprintf("%*s", digits, "")) + m.st.muted.Render(" │ ")
			}
		} else {
			prefix = " "
		}
		body = append(body, prefix+s.lines[i].Text)
	}
	if _, pad := m.splitTextW(); pad > 0 {
		margin := strings.Repeat(" ", pad)
		for i := range body {
			body[i] = margin + body[i]
		}
	}
	return m.box(displayName(s.path), body, w, h, false)
}
