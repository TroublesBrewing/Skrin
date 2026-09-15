package editor

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	"github.com/lurioso/skrin/internal/theme"
)

// Style slots for highlighting.
const (
	sText = iota
	sMeta
	sCode
	sLink
	sTag
	sBold
	sMarker
	sHeading // sHeading..sHeading+5 are h1..h6
	sSel     = sHeading + 6
)

type styles struct {
	slot                 []lipgloss.Style
	cursor, normalCursor lipgloss.Style
}

func newStyles(p theme.Palette) styles {
	s := lipgloss.NewStyle
	st := styles{slot: make([]lipgloss.Style, sSel+1)}
	st.slot[sSel] = s().Background(p.Selection).Foreground(p.LightForeground)
	st.slot[sText] = s().Foreground(p.Foreground)
	st.slot[sMeta] = s().Foreground(p.DarkForeground)
	st.slot[sCode] = s().Foreground(p.Code)
	st.slot[sLink] = s().Foreground(p.Link)
	st.slot[sTag] = s().Foreground(p.Tag)
	st.slot[sBold] = s().Foreground(p.Foreground).Bold(true)
	st.slot[sMarker] = s().Foreground(p.Accent)
	for i, c := range p.Headings {
		st.slot[sHeading+i] = s().Foreground(c).Bold(true)
	}
	st.cursor = s().Background(p.Accent).Foreground(p.Background)
	st.normalCursor = s().Background(p.Foreground).Foreground(p.Background)
	return st
}

// Line kinds, decided per line from the lines above it.
const (
	kText = iota
	kMeta // frontmatter
	kCode // fenced code, fences included
	kHeading
)

var (
	headingRE = regexp.MustCompile(`^ {0,3}(#{1,6})[ \t]`)
	markerRE  = regexp.MustCompile(`^\s*([-*+]|\d{1,9}[.)])( \[.\])?[ \t]`)
	spans     = []struct {
		re    *regexp.Regexp
		slot  int
		after bool // the match starts with the character before the token
	}{
		{regexp.MustCompile(`\*\*[^*]+\*\*`), sBold, false},
		{regexp.MustCompile(`(^|\s)#[\p{L}\p{N}_/-]*[\p{L}_/-][\p{L}\p{N}_/-]*`), sTag, true},
		{regexp.MustCompile(`!?\[\[[^\[\]]+\]\]`), sLink, false},
		{regexp.MustCompile(`\[[^\[\]]+\]\([^()]*\)`), sLink, false},
		{regexp.MustCompile("`[^`]+`"), sCode, false},
	}
)

// View renders exactly h display lines, each at most w cells wide.
func (e *Editor) View() []string {
	kinds := e.classify()
	out := make([]string, 0, e.h)
	d := 0
	for row := 0; row < len(e.lines) && len(out) < e.h; row++ {
		starts := e.segments(row)
		if d+len(starts) <= e.top {
			d += len(starts)
			continue
		}
		slots := e.highlight(row, kinds[row])
		e.markSel(row, slots)
		for s := range starts {
			if d >= e.top && len(out) < e.h {
				out = append(out, e.renderSegment(row, starts, s, slots))
			}
			d++
		}
	}
	for len(out) < e.h {
		out = append(out, "")
	}
	return out
}

func (e *Editor) renderSegment(row int, starts []int, s int, slots []int) string {
	line := e.lines[row]
	a, b := starts[s], len(line)
	if s+1 < len(starts) {
		b = starts[s+1]
	}
	cursor := e.st.cursor
	if e.vim && e.mode == Normal {
		cursor = e.st.normalCursor
	}
	switch {
	case row != e.row || segOf(starts, e.col) != s:
		return e.renderRange(line, a, b, slots)
	case e.col >= b: // end of the line
		return e.renderRange(line, a, b, slots) + cursor.Render(" ")
	}
	return e.renderWithCursorAt(row, a, b, slots, e.col, cursor)
}

// renderWithCursorAt draws runes a..b with the cursor on rune c.
func (e *Editor) renderWithCursorAt(row, a, b int, slots []int, c int, cursor lipgloss.Style) string {
	line := e.lines[row]
	var out strings.Builder
	out.WriteString(e.renderRange(line, a, c, slots))
	cell := string(line[c])
	rest := ""
	if line[c] == '\t' {
		cell, rest = " ", strings.Repeat(" ", tabWidth-1)
	}
	out.WriteString(cursor.Render(cell) + rest)
	out.WriteString(e.renderRange(line, c+1, b, slots))
	return out.String()
}

func (e *Editor) renderRange(line []rune, a, b int, slots []int) string {
	var out, run strings.Builder
	slot := -1
	for i := a; i < b; i++ {
		if slots[i] != slot {
			if run.Len() > 0 {
				out.WriteString(e.st.slot[slot].Render(run.String()))
				run.Reset()
			}
			slot = slots[i]
		}
		if line[i] == '\t' {
			run.WriteString(strings.Repeat(" ", tabWidth))
		} else {
			run.WriteRune(line[i])
		}
	}
	if run.Len() > 0 {
		out.WriteString(e.st.slot[slot].Render(run.String()))
	}
	return out.String()
}

// classify decides each line's kind: frontmatter, code or heading.
func (e *Editor) classify() []int {
	kinds := make([]int, len(e.lines))
	front := len(e.lines) > 1 && strings.TrimRight(string(e.lines[0]), " ") == "---"
	fence := ""
	for i, l := range e.lines {
		s := string(l)
		t := strings.TrimSpace(s)
		switch {
		case front:
			kinds[i] = kMeta
			if i > 0 && (t == "---" || t == "...") {
				front = false
			}
		case fence != "":
			kinds[i] = kCode
			if strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]) == "" {
				fence = ""
			}
		case strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~"):
			kinds[i] = kCode
			fence = t[:len(t)-len(strings.TrimLeft(t, t[:1]))]
		default:
			if m := headingRE.FindStringSubmatch(s); m != nil {
				kinds[i] = kHeading + len(m[1]) - 1
			}
		}
	}
	return kinds
}

// markSel paints the selected part of row.
func (e *Editor) markSel(row int, slots []int) {
	if !e.sel {
		return
	}
	r1, c1, r2, c2 := e.selRange()
	if row < r1 || row > r2 {
		return
	}
	a, b := 0, len(slots)
	if row == r1 {
		a = c1
	}
	if row == r2 {
		b = c2
	}
	for i := a; i < b && i < len(slots); i++ {
		slots[i] = sSel
	}
}

// highlight gives each rune of a line its style slot.
func (e *Editor) highlight(row, kind int) []int {
	line := e.lines[row]
	slots := make([]int, len(line))
	base := sText
	switch {
	case kind == kMeta:
		base = sMeta
	case kind == kCode:
		base = sCode
	case kind >= kHeading:
		base = sHeading + kind - kHeading
	}
	for i := range slots {
		slots[i] = base
	}
	if kind == kMeta || kind == kCode {
		return slots
	}
	s := string(line)
	mark := func(a, b, slot int) {
		for i := utf8.RuneCountInString(s[:a]); i < utf8.RuneCountInString(s[:b]); i++ {
			slots[i] = slot
		}
	}
	if kind == kText {
		if m := markerRE.FindStringIndex(s); m != nil {
			mark(m[0], m[1], sMarker)
		}
	}
	for _, sp := range spans {
		for _, m := range sp.re.FindAllStringIndex(s, -1) {
			a := m[0]
			if sp.after && a < len(s) && s[a] != '#' {
				_, n := utf8.DecodeRuneInString(s[a:])
				a += n
			}
			mark(a, m[1], sp.slot)
		}
	}
	return slots
}
