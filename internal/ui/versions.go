package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/markdown"
	"github.com/lurioso/skrin/internal/snapshot"
	"github.com/lurioso/skrin/internal/vault"
	"github.com/lurioso/skrin/internal/version"
)

// The time machine (V) is the visible form of something Skrin has been
// doing all along: every time it writes over a note, it keeps what was
// there. u walks back through those versions one step at a time and never
// shows you where you are going. This shows the whole line — oldest on the
// left, the note as it is now on the right — lets you read any version, or
// what changed, and puts one back if you want it.
//
// It only reads. The single write it can make (enter, to put a version
// back) snapshots what was there first and is one journal step, so both u
// and U take it off again.

// versionsView is the open time machine.
type versionsView struct {
	rel   string
	saved []snapshot.Snapshot // oldest first
	now   string              // the note as it was on disk when this opened
	// cur is the version being shown: an index into saved, or len(saved)
	// for the note as it is now — the right-hand end of the line.
	cur  int
	off  int  // how far the text or the diff is scrolled
	diff bool // showing what changed instead of the text itself
}

// versionsOn is the time machine's real state: a beta feature needs the
// build to allow beta, beta mode to be on, and its own switch on.
func (m *Model) versionsOn() bool {
	return version.Beta && m.opts.Beta && m.opts.Versions
}

// openVersions opens the time machine on the note in hand.
func (m *Model) openVersions() {
	if !m.versionsOn() {
		m.flash = "The time machine is off: it's a beta feature, switched on in Settings (" + note(m.keyFor(inMain, actHelp), "?") + " then tab)"
		return
	}
	rel, ok := m.subject()
	if !ok {
		m.flash = "Select a note to see its earlier versions"
		return
	}
	// The one thing it must never do is write over text that is still being
	// typed, so it doesn't open on a note that is open for editing.
	if m.editor != nil && m.edit.rel == rel {
		m.flash = "Save (ctrl+s) or close (esc) first: the time machine writes the note"
		return
	}
	src, err := m.vault.Read(rel)
	if err != nil {
		m.flash = "Can't read " + rel + ": " + err.Error()
		return
	}
	saved := m.snaps.List(rel)
	if len(saved) == 0 {
		m.flash = "No earlier versions of " + displayName(rel) + " yet · Skrin keeps one every time it writes over a note"
		return
	}
	m.versions = &versionsView{
		rel:   rel,
		saved: saved,
		now:   src,
		cur:   len(saved) - 1, // the newest saved version: where u would take you
	}
}

// versionKey handles the time machine's keys.
func (m *Model) versionKey(k tea.KeyPressMsg) {
	v := m.versions
	switch m.actionIn(inVersions, k.String()) {
	case actCancel:
		m.versions = nil
		m.flash = "Time machine closed · nothing changed"
	case actLeft:
		v.step(-1)
	case actRight:
		v.step(1)
	case actUp:
		v.off = max(v.off-1, 0)
	case actDown:
		v.off++
	case actVersionDiff:
		v.diff = !v.diff
		v.off = 0
	case actPick:
		m.putVersionBack()
	}
}

// step moves along the line of versions, oldest on the left, now on the
// right, and starts the new one from its top.
func (v *versionsView) step(dir int) {
	next := clamp(v.cur+dir, 0, len(v.saved))
	if next != v.cur {
		v.cur, v.off = next, 0
	}
}

// text is the version being shown.
func (v *versionsView) text() string {
	if v.cur >= len(v.saved) {
		return v.now
	}
	return v.saved[v.cur].Content
}

// when is what to call the version being shown.
func (v *versionsView) when(now time.Time) string {
	if v.cur >= len(v.saved) {
		return "as it is now"
	}
	return when(v.saved[v.cur].Time, now)
}

// putVersionBack writes the shown version over the note. It reads the note
// again first, so what it keeps is what is really there, and refuses when
// there would be nothing to change. (u's own one-step-back is
// restoreVersion in edit.go; this is the same idea, aimed anywhere.)
func (m *Model) putVersionBack() {
	v := m.versions
	if v.cur >= len(v.saved) {
		m.flash = "That is the note as it is · ← goes back in time"
		return
	}
	snap := v.saved[v.cur]
	cur, err := m.vault.Read(v.rel)
	if err != nil {
		m.flash = "Can't read " + v.rel + ": " + err.Error()
		return
	}
	if cur == snap.Content {
		m.flash = "The note already reads exactly like that version"
		return
	}
	if err := m.snaps.Save(v.rel, cur); err != nil {
		// Refusing is right: without the snapshot there would be no u.
		m.flash = "Nothing restored: couldn't keep the current text first (" + err.Error() + ")"
		return
	}
	if err := m.vault.Write(v.rel, snap.Content); err != nil {
		m.flash = "Couldn't write " + v.rel + ": " + err.Error()
		return
	}
	m.journal.Record(vault.Op{
		Desc:  "restore " + v.rel + " as it was " + when(snap.Time, m.opts.Now()),
		Steps: []vault.Step{{Kind: vault.StepModified, Rel: v.rel, Content: cur}},
	})
	rel, was := v.rel, when(snap.Time, m.opts.Now())
	m.versions = nil
	m.refresh()
	m.reveal(rel)
	m.open(rel)
	m.flash = fmt.Sprintf("Restored %s as it was %s · u or U undoes", displayName(rel), was)
}

// versionsBox draws the time machine: the version above, the line of
// versions below it.
func (m *Model) versionsBox() []string {
	v := m.versions
	if v == nil {
		return nil
	}
	w := min(max(m.width-6, 50), 96)
	inner := w - 4
	rows := max(m.height-12, 4)

	lines := m.versionLines(v, inner-2)
	v.off = clamp(v.off, 0, max(len(lines)-1, 0))
	body := make([]string, 0, rows+3)
	for i := v.off; i < len(lines) && len(body) < rows; i++ {
		body = append(body, "  "+fit(lines[i], inner-2))
	}
	for len(body) < rows {
		body = append(body, "")
	}
	body = append(body, "  "+m.st.border.Render(strings.Repeat("─", max(inner-2, 0))))
	body = append(body, "  "+m.versionStrip(v, inner-2))
	body = append(body, "  "+m.st.muted.Render(m.versionHint()))
	return m.box(" Time machine · "+displayName(v.rel)+" ", body, w, len(body)+2, true)
}

// versionLines is what the panel shows: the version's own text, rendered
// as a note, or the unified diff between it and the note as it is.
func (m *Model) versionLines(v *versionsView, width int) []string {
	if v.diff {
		out := []string{}
		for _, l := range diffLinesNamed("that version", "now", v.text(), v.now) {
			out = append(out, m.diffLine(l))
		}
		return out
	}
	var out []string
	for _, l := range markdown.Render(v.text(), markdown.Options{Width: width, Palette: m.pal}) {
		out = append(out, l.Text)
	}
	return out
}

// versionStrip is the line of versions: one mark each, oldest on the left,
// the note as it is on the right, with the one being shown filled in.
func (m *Model) versionStrip(v *versionsView, width int) string {
	var b strings.Builder
	for i := 0; i <= len(v.saved); i++ {
		mark := "○"
		if i == len(v.saved) {
			mark = "◆" // the note as it is
		}
		if i == v.cur {
			b.WriteString(m.st.titleFocus.Render("●"))
		} else {
			b.WriteString(m.st.muted.Render(mark))
		}
		if i < len(v.saved) {
			b.WriteString(m.st.muted.Render("─"))
		}
	}
	where := fmt.Sprintf("%d of %d · %s", v.cur+1, len(v.saved)+1, v.when(m.opts.Now()))
	if added, removed := countChanges(v.text(), v.now); added+removed > 0 {
		where += fmt.Sprintf(" · +%d −%d against now", added, removed)
	}
	return spread(b.String(), m.st.muted.Render(where), width)
}

func (m *Model) versionHint() string {
	what := "d what changed"
	if m.versions.diff {
		what = "d the text itself"
	}
	return fmt.Sprintf("←→ move in time · j/k scroll · %s · enter restores this version · esc closes", what)
}

// countChanges counts the lines a unified diff adds and takes away, for
// the strip's own summary of how far from now a version is.
func countChanges(from, to string) (added, removed int) {
	for _, l := range diffLinesNamed("from", "to", from, to) {
		switch {
		case strings.HasPrefix(l, "+++"), strings.HasPrefix(l, "---"):
		case strings.HasPrefix(l, "+"):
			added++
		case strings.HasPrefix(l, "-"):
			removed++
		}
	}
	return added, removed
}
