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

	"github.com/lurioso/skrin/internal/duedate"
	"github.com/lurioso/skrin/internal/editor"
	"github.com/lurioso/skrin/internal/obsidian"
	"github.com/lurioso/skrin/internal/snapshot"
	"github.com/lurioso/skrin/internal/vault"
)

// editSession is the note open in the built-in editor.
type editSession struct {
	rel         string
	base        string // the note as it was on disk when last loaded or saved
	opened      string // the note as it was when the editor opened, whatever saves came since
	snapshotted bool   // the version from before this session is in the snapshot store
	linkFormat  string // Obsidian's newLinkFormat, for [[ completion
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

// subject is the note that e, E, u, Ctrl-r, b and o act on: the note under
// the cursor while Files has focus, otherwise the open note.
func (m *Model) subject() (string, bool) {
	if m.focus == paneFiles {
		if e := m.files.selected(); !e.IsDir && vault.IsNote(e.Name) {
			return e.Rel, true
		}
		return "", false
	}
	return m.notePath, m.notePath != ""
}

// subjectOpen is subject, opened on the right first if it isn't already,
// for the keys that change or scroll what's on screen. A subject that's
// already showing in the split pane is swapped into focus rather than
// reopened over it — the same courtesy openRow gives Enter/l/→, so e, E,
// u, Ctrl-r and o never quietly discard the reference note either.
func (m *Model) subjectOpen() (string, bool) {
	rel, ok := m.subject()
	switch {
	case !ok:
	case m.split != nil && rel == m.split.path:
		m.swapPanes()
	case rel != m.notePath:
		m.pushHistory()
		m.showNote(rel)
	}
	return rel, ok
}

func (m *Model) startEdit() {
	rel, ok := m.subjectOpen()
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
	m.editor.SetLineNumbers(m.opts.LineNumbers)
	m.edit = editSession{rel: rel, base: text, opened: text, linkFormat: obsidian.LoadSettings(m.vault.Root).NewLinkFormat}
	m.focus = paneNote
	m.settle()
	m.editor.GoTo(row)
}

func (m *Model) editorKey(k tea.KeyPressMsg) tea.Cmd {
	// Alt+<key> reaches a view-mode action without leaving the editor: the
	// same overlay or mode, on the note being edited. Plain letters keep
	// typing text, so these only ever fire with Alt held.
	switch a := m.actionIn(inEditor, k.String()); a {
	case actAskClaude:
		m.openDrawer()
		return nil
	case actZen, actBacklinks, actOutline:
		return m.do(a)
	}
	switch m.editor.HandleKey(k) {
	case editor.Save:
		m.saveEdit(false)
	case editor.Close:
		m.saveEdit(true)
	case editor.Copy:
		return m.copySelection()
	}
	return nil
}

// copySelection sends the current selection to the system clipboard over
// OSC 52 — the terminal itself relays it, so this works the same in a
// local terminal, over SSH, or inside tmux with clipboard passthrough on,
// with no OS-specific clipboard tool and no cgo. It's a no-op, silently,
// when the terminal doesn't support OSC 52 at all: there's no reliable way
// to tell in advance, so the flash names what was attempted rather than
// promising it landed.
func (m *Model) copySelection() tea.Cmd {
	s := m.selectionText()
	if s == "" {
		return nil
	}
	n := strings.Count(s, "\n") + 1
	if n == 1 {
		m.flash = "Copied 1 line"
	} else {
		m.flash = fmt.Sprintf("Copied %d lines", n)
	}
	return tea.SetClipboard(s)
}

func (m *Model) paste(s string) {
	switch {
	case m.conflict != nil:
	case m.focus == paneClaude:
		m.drawer.input.Paste(s)
	case m.editor != nil:
		m.editor.Paste(s)
		m.updateCompletion()
	case m.manual != nil:
		if m.manual.filtering {
			m.manual.in.insert(s)
		}
	case m.prompt != nil:
		m.prompt.in.insert(s)
	case m.chooser != nil:
		m.chooser.in.insert(s)
		m.chooser.filter()
	case m.quickNote != nil:
		c := m.quickNote
		if c.area == quickNoteText {
			c.text.insert(s)
		} else {
			c.folder.insert(s)
			c.filterFolders()
		}
	case m.search != nil:
		m.search.paste(s)
		m.runSearch()
	}
}

// saveEdit writes the editor's text, unless the note changed on disk in the
// meantime, in which case the user decides.
func (m *Model) saveEdit(closing bool) {
	s := &m.edit
	var dates string
	if closing {
		dates = m.resolveDueDates()
	}
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
	m.flash = "Saved " + s.rel + dates
	if closing {
		m.closeEditor()
	}
}

// resolveDueDates turns "due:: tomorrow" and the like into dates in the
// editor's text, on the lines typed or changed since the editor opened —
// a Ctrl-s along the way doesn't make them old — and says what it did for
// the save flash.
func (m *Model) resolveDueDates() string {
	text, changes := duedate.Resolve(m.editor.Text(), m.edit.opened, m.opts.Now())
	if len(changes) == 0 {
		return ""
	}
	m.editor.Reset(text)
	var parts []string
	for _, c := range changes {
		parts = append(parts, c.Word+" → "+c.Date)
	}
	if len(changes) == 1 {
		return " · due:: " + parts[0]
	}
	return fmt.Sprintf(" · %d due dates: %s", len(changes), strings.Join(parts, ", "))
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
	row := m.editor.TopRow()
	m.editor, m.conflict, m.edit = nil, nil, editSession{}
	m.complete, m.noComplete = nil, false
	m.refresh()
	m.jumpSrc = row
}

func diffLines(disk, mine string) []string {
	u := udiff.Unified("on disk", "yours", disk, mine)
	if u == "" {
		return []string{"No differences"}
	}
	return strings.Split(strings.TrimRight(u, "\n"), "\n")
}

// editorBody is the editor's title and lines, or the conflict diff.
func (m *Model) editorBody(vis int) (string, []string) {
	title := displayName(m.edit.rel)
	if m.editor.Dirty() {
		title += " ●"
	}
	var body []string
	if c := m.conflict; c != nil && c.diff != nil {
		title = "on disk → yours"
		for i := c.off; i < min(len(c.diff), c.off+vis); i++ {
			body = append(body, " "+m.diffLine(c.diff[i]))
		}
	} else {
		for _, l := range m.editor.View() {
			if m.editor.LineNumbers() {
				body = append(body, l)
			} else {
				body = append(body, " "+l)
			}
		}
	}
	return title, body
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
	mode, hint := " EDIT ", "ctrl+s save · ctrl+l to-do · esc done"
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
	rel, ok := m.subjectOpen()
	if !ok {
		m.flash = "Select a note to edit in $EDITOR"
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
	rel, ok := m.subjectOpen()
	if !ok {
		verb := "undo"
		if redo {
			verb = "redo"
		}
		m.flash = "Select a note to " + verb
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
