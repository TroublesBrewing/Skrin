package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	udiff "github.com/aymanbagabas/go-udiff"

	"github.com/lurioso/skrin/internal/editor"
	"github.com/lurioso/skrin/internal/snapshot"
	"github.com/lurioso/skrin/internal/vault"
)

// editSession is the note open in the built-in editor.
type editSession struct {
	rel         string
	base        string // the note as it was on disk when last loaded or saved
	snapshotted bool   // the version from before this session is in the snapshot store
}

// conflict is raised when a save finds that the note changed on disk since
// the editor loaded it (Obsidian, Sync, another editor).
type conflict struct {
	disk    string
	deleted bool
	closing bool     // the save came from leaving the editor
	diff    []string // non-nil while the diff is shown
	off     int
}

// externalDoneMsg arrives when the external editor exits.
type externalDoneMsg struct {
	rel, before string
	err         error
}

// editTarget is the note under the list cursor, if it is one.
func (m *Model) editTarget() (string, bool) {
	e, ok := m.selected()
	if !ok || e.IsDir || !vault.IsNote(e.Name) {
		return "", false
	}
	return e.Rel, true
}

func (m *Model) startEdit() {
	rel, ok := m.editTarget()
	if !ok {
		m.flash = "Select a note to edit"
		return
	}
	m.openEditor(rel)
}

// openEditor opens rel in the built-in editor, at the line the note pane
// was showing at its top.
func (m *Model) openEditor(rel string) {
	text, err := m.vault.Read(rel)
	if err != nil {
		m.flash = "Can't open " + rel + ": " + err.Error()
		return
	}
	row := 0
	if m.notePath == rel && m.noteOff < len(m.lines) {
		row = m.lines[m.noteOff].Src
	}
	m.editor = editor.New(text, m.opts.Vim, m.pal)
	m.edit = editSession{rel: rel, base: text}
	m.focus = paneNote
	m.settle()
	m.editor.GoTo(row)
}

func (m *Model) editorKey(k tea.KeyPressMsg) {
	switch m.editor.HandleKey(k) {
	case editor.Save:
		m.saveEdit(false)
	case editor.Close:
		m.saveEdit(true)
	}
}

func (m *Model) paste(s string) {
	switch {
	case m.conflict != nil:
	case m.editor != nil:
		m.editor.Paste(s)
	case m.prompt != nil:
		m.prompt.in.insert(s)
	case m.picker != nil:
		m.picker.in.insert(s)
		m.picker.filter()
	}
}

// saveEdit writes the editor's text, unless the note changed on disk in the
// meantime, in which case the user decides.
func (m *Model) saveEdit(closing bool) {
	s := &m.edit
	text := m.editor.Text()
	if text == s.base {
		if closing {
			m.closeEditor()
		}
		return
	}
	disk, err := m.vault.Read(s.rel)
	deleted := errors.Is(err, fs.ErrNotExist)
	if err != nil && !deleted {
		m.flash = "Can't check " + s.rel + " before saving: " + err.Error()
		return
	}
	if deleted || disk != s.base {
		m.conflict = &conflict{disk: disk, deleted: deleted, closing: closing}
		return
	}
	if !s.snapshotted {
		if err := m.snaps.Save(s.rel, s.base); err != nil {
			m.flash = "Saved, but u won't be able to undo this edit: " + err.Error()
		}
		s.snapshotted = true
	}
	if err := m.vault.Write(s.rel, text); err != nil {
		m.flash = "Couldn't save: " + err.Error()
		return
	}
	s.base = text
	m.editor.MarkSaved()
	m.flash = "Saved " + s.rel
	if closing {
		m.closeEditor()
	}
}

func (m *Model) conflictKey(k tea.KeyPressMsg) {
	c := m.conflict
	switch k.String() {
	case "m":
		m.keepMine()
	case "t":
		m.takeTheirs()
	case "d":
		if c.diff == nil {
			c.diff = diffLines(c.disk, m.editor.Text())
		} else {
			c.diff = nil
		}
		c.off = 0
	case "j", "down":
		c.off = min(c.off+1, max(len(c.diff)-1, 0))
	case "k", "up":
		c.off = max(c.off-1, 0)
	case "esc":
		if c.diff != nil {
			c.diff = nil
			return
		}
		m.conflict = nil
		m.flash = "Still editing; nothing saved yet"
	}
}

// keepMine overwrites the version on disk, keeping it as a snapshot first
// so u can bring it back.
func (m *Model) keepMine() {
	c, s := m.conflict, &m.edit
	if !s.snapshotted {
		if err := m.snaps.Save(s.rel, s.base); err != nil {
			m.flash = "Couldn't keep a backup, so nothing was saved: " + err.Error()
			return
		}
	}
	if !c.deleted {
		if err := m.snaps.Save(s.rel, c.disk); err != nil {
			m.flash = "Couldn't keep a backup of the version on disk, so nothing was saved: " + err.Error()
			return
		}
	}
	m.conflict = nil
	s.snapshotted = true
	text := m.editor.Text()
	if err := m.vault.Write(s.rel, text); err != nil {
		m.flash = "Couldn't save: " + err.Error()
		return
	}
	s.base = text
	m.editor.MarkSaved()
	m.flash = "Kept your version · u brings back the one from disk"
	if c.closing {
		m.closeEditor()
	}
}

// takeTheirs drops the editor's text for the version on disk, keeping
// yours as a snapshot so u can bring it back.
func (m *Model) takeTheirs() {
	c, s := m.conflict, &m.edit
	if err := m.snaps.Save(s.rel, m.editor.Text()); err != nil {
		m.flash = "Couldn't keep a backup of your version, so nothing changed: " + err.Error()
		return
	}
	m.conflict = nil
	if c.deleted {
		m.closeEditor()
		m.flash = "Left it deleted · your version is kept as a snapshot"
		return
	}
	m.editor.Reset(c.disk)
	// From here on it's a fresh edit of their version.
	s.base, s.snapshotted = c.disk, false
	m.flash = "Took the version on disk · u brings yours back"
	if c.closing {
		m.closeEditor()
	}
}

func (m *Model) closeEditor() {
	row, rel := m.editor.TopRow(), m.edit.rel
	m.editor, m.conflict, m.edit = nil, nil, editSession{}
	if err := m.reload(); err != nil {
		m.flash = err.Error()
	}
	m.selectRel(rel)
	m.jumpSrc = row
}

func diffLines(disk, mine string) []string {
	u := udiff.Unified("on disk", "yours", disk, mine)
	if u == "" {
		return []string{"No differences"}
	}
	return strings.Split(strings.TrimRight(u, "\n"), "\n")
}

func (m *Model) editorPane(w, h int) []string {
	title := displayName(m.edit.rel)
	if m.editor.Dirty() {
		title += " ●"
	}
	var body []string
	if c := m.conflict; c != nil && c.diff != nil {
		title = "on disk → yours"
		for i := c.off; i < min(len(c.diff), c.off+h-2); i++ {
			body = append(body, " "+m.diffLine(c.diff[i]))
		}
	} else {
		for _, l := range m.editor.View() {
			body = append(body, " "+l)
		}
	}
	return m.box(title, body, w, h, true)
}

func (m *Model) diffLine(l string) string {
	switch {
	case strings.HasPrefix(l, "@@"):
		return m.st.diffHunk.Render(l)
	case strings.HasPrefix(l, "+"):
		return m.st.diffAdd.Render(l)
	case strings.HasPrefix(l, "-"):
		return m.st.diffDel.Render(l)
	}
	return m.st.muted.Render(l)
}

func (m *Model) editLine() string {
	mode, hint := " EDIT ", "ctrl+s save · esc done"
	if m.opts.Vim {
		mode, hint = " INSERT ", "esc normal mode"
		if m.editor.Mode() == editor.Normal {
			mode, hint = " NORMAL ", "i insert · esc done"
		}
	}
	left := m.st.pill.Render(mode) + " " + m.st.text.Render(m.edit.rel)
	right := m.st.muted.Render(hint)
	if m.flash != "" {
		right = m.st.flash.Render(m.flash)
	}
	return spread(left, right, m.width)
}

func (m *Model) conflictLine() string {
	c := m.conflict
	what := "changed on disk"
	if c.deleted {
		what = "was deleted on disk"
	}
	left := m.st.dangerPill.Render(" CONFLICT ") + " " + m.st.text.Render(displayName(m.edit.rel)+" "+what+" while you edited.")
	keys := "m keep mine · t take theirs · d diff · esc keep editing"
	if c.diff != nil {
		keys = "j/k scroll · m keep mine · t take theirs · d hide diff"
	}
	return spread(left, m.st.bold.Render(keys), m.width)
}

// startExternal hands the note to an external editor; Skrin pauses until it
// exits.
func (m *Model) startExternal() tea.Cmd {
	rel, ok := m.editTarget()
	if !ok {
		m.flash = "Select a note to edit"
		return nil
	}
	before, err := m.vault.Read(rel)
	if err != nil {
		m.flash = "Can't open " + rel + ": " + err.Error()
		return nil
	}
	args := externalEditor(m.opts.ExternalEditor)
	c := exec.Command(args[0], append(args[1:], m.vault.Abs(rel))...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return externalDoneMsg{rel: rel, before: before, err: err}
	})
}

// externalEditor is the command for E: the configured one, else $VISUAL,
// $EDITOR, nvim, vi.
func externalEditor(configured string) []string {
	for _, s := range []string{configured, os.Getenv("VISUAL"), os.Getenv("EDITOR")} {
		if f := strings.Fields(s); len(f) > 0 {
			return f
		}
	}
	if _, err := exec.LookPath("nvim"); err == nil {
		return []string{"nvim"}
	}
	return []string{"vi"}
}

// externalDone keeps the pre-edit version as a snapshot if the external
// editor changed the note.
func (m *Model) externalDone(msg externalDoneMsg) {
	after, err := m.vault.Read(msg.rel)
	changed := err == nil && after != msg.before
	switch {
	case changed:
		if err := m.snaps.Save(msg.rel, msg.before); err != nil {
			m.flash = "Changes saved, but u can't undo them: " + err.Error()
		} else {
			m.flash = "Saved your changes · u undoes them"
		}
	case msg.err != nil:
		m.flash = "Editor failed: " + msg.err.Error()
	default:
		m.flash = "No changes"
	}
	if err := m.reload(); err != nil {
		m.flash = err.Error()
	}
}

// restoreVersion is u (the note as it was before its last edit) and Ctrl-r
// (take that back).
func (m *Model) restoreVersion(redo bool) {
	rel, ok := m.editTarget()
	if !ok {
		m.flash = "Select a note first"
		return
	}
	cur, err := m.vault.Read(rel)
	if err != nil {
		m.flash = err.Error()
		return
	}
	var snap snapshot.Snapshot
	if redo {
		snap, ok, err = m.snaps.Redo(rel, cur)
	} else {
		snap, ok, err = m.snaps.Undo(rel, cur)
	}
	name := describe([]string{rel})
	switch {
	case err != nil && !ok:
		m.flash = err.Error()
		return
	case !ok && redo:
		m.flash = "Nothing to redo for " + name
		return
	case !ok:
		m.flash = "No earlier version of " + name
		return
	}
	if err := m.vault.Write(rel, snap.Content); err != nil {
		m.flash = "Couldn't restore: " + err.Error()
		return
	}
	if err := m.reload(); err != nil {
		m.flash = err.Error()
		return
	}
	if redo {
		m.flash = "Put back the newer version of " + name + " · u undoes"
	} else {
		m.flash = fmt.Sprintf("Restored %s as it was %s · Ctrl-r redoes", name, when(snap.Time, m.opts.Now()))
	}
}

func when(t, now time.Time) string {
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return "at " + t.Format("15:04")
	}
	return "on " + t.Format("02 Jan 15:04")
}
