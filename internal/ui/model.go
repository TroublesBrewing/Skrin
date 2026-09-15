// Package ui is Skrin's Bubble Tea front end: Files (the vault as one tree)
// and the open note side by side, between a header and a status line.
package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/editor"
	"github.com/lurioso/skrin/internal/index"
	"github.com/lurioso/skrin/internal/logo"
	"github.com/lurioso/skrin/internal/markdown"
	"github.com/lurioso/skrin/internal/session"
	"github.com/lurioso/skrin/internal/snapshot"
	"github.com/lurioso/skrin/internal/theme"
	"github.com/lurioso/skrin/internal/vault"
)

type pane int

const (
	paneFiles pane = iota
	paneNote
)

const (
	headerHeight = 3
	statusHeight = 1
	// Narrower than this, Files only shows while it has focus.
	filesAutoHideWidth = 80
	// zenWidth is the widest a note runs in zen mode, for easy reading.
	zenWidth = 80
)

// ThemeMsg carries the reloaded palette after Omarchy switches theme.
type ThemeMsg struct{ Palette theme.Palette }

// VaultChangedMsg reports that files changed on disk.
type VaultChangedMsg struct{}

// ggWait is how soon a second G must follow the first to make it GG: the
// top instead of the bottom.
const ggWait = 400 * time.Millisecond

// Options tune behaviour that depends on the world outside the vault.
type Options struct {
	// RolloverTodos makes `t` carry unfinished todos into a new daily note.
	RolloverTodos bool
	// ObsidianOpen reports whether Obsidian desktop has this vault open. Its
	// Rollover plugin then does the rollover, so Skrin must not.
	ObsidianOpen func() bool
	// Vim gives the built-in editor vim-style keys.
	Vim bool
	// ExternalEditor is the command E runs; empty means $VISUAL, $EDITOR,
	// then nvim.
	ExternalEditor string
	// Open hands a file or web address to the desktop; nil means xdg-open.
	Open func(target string) error
	// Session is where the last run in this vault left off.
	Session session.State
	// Now is the clock; tests pin it.
	Now func() time.Time
}

// Model is the root Bubble Tea model.
type Model struct {
	vault *vault.Vault
	idx   *index.Index
	snaps *snapshot.Store
	pal   theme.Palette
	st    styles
	keys  map[string]action
	opts  Options
	logo  []string
	logoW int
	// splash is the large logo at scale 1 and 2, for the empty note pane.
	splash [][]string

	width, height int
	focus         pane
	zen           bool // z: only the note, centred at a readable width

	files files // the Files pane; its cursor moves independently of the open note

	notePath  string // the open note, "" when none is
	noteSrc   string
	noteErr   error
	lines     []markdown.Line
	renderedW int // width lines were rendered at; 0 forces a re-render
	noteOff   int
	jumpSrc   int // after the next render, scroll to this source line; -1 for none

	back, fwd []place // history of opened notes

	lastG time.Time // when G was pressed, if it was the last key (for GG)

	journal vault.Journal
	marks   map[string]bool // marked items by vault path; may span folders
	visual  *visualRange

	editor     *editor.Editor // the built-in editor, while a note is open in it
	edit       editSession
	conflict   *conflict
	complete   *completion // the [[ popup
	noComplete bool        // the popup was dismissed for this link

	// At most one of these is open; it takes all keys while it is.
	hints   *hintState
	prompt  *prompt
	confirm *confirm
	chooser *chooser
	search  *searchPanel
	manual  *manual

	lastSearch *searchPanel // reopened by the next /

	flash string // one-shot status message, cleared by the next key
}

// New builds the model for vault v, back where opts.Session left off.
func New(v *vault.Vault, pal theme.Palette, opts Options) (*Model, error) {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.ObsidianOpen == nil {
		opts.ObsidianOpen = func() bool { return false }
	}
	if opts.Open == nil {
		opts.Open = xdgOpen
	}
	m := &Model{
		vault: v, idx: index.New(), snaps: snapshot.Open(v.Root), keys: keymap(), files: newFiles(),
		opts: opts, marks: map[string]bool{}, jumpSrc: -1,
	}
	m.setPalette(pal)
	for _, p := range opts.Session.Expanded {
		m.files.expanded[p] = true
	}
	if err := m.reload(); err != nil {
		return nil, err
	}
	m.files.selectPath(opts.Session.Cursor)
	if rel := opts.Session.Open; vault.IsNote(rel) && m.vault.Exists(rel) {
		m.showNote(rel)
		m.noteOff = opts.Session.Offset // clamped once the note is rendered
	}
	return m, nil
}

// Session is where you are now, for the next run to pick up.
func (m *Model) Session() session.State {
	return session.State{
		Expanded: m.files.openFolders(),
		Cursor:   m.files.selected().Rel,
		Open:     m.notePath,
		Offset:   m.noteOff,
	}
}

// Flash shows a one-shot message in the status line.
func (m *Model) Flash(s string) { m.flash = s }

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case ThemeMsg:
		m.setPalette(msg.Palette)
		m.renderedW = 0
		m.flash = "theme: " + msg.Palette.Name
	case VaultChangedMsg:
		open := m.notePath
		if err := m.reload(); err != nil {
			m.flash = "refresh failed: " + err.Error()
		} else if open != "" && m.notePath == "" {
			m.flash = displayName(open) + " is gone: deleted or renamed outside Skrin"
		}
	case externalDoneMsg:
		m.externalDone(msg)
	case tea.PasteMsg:
		m.paste(msg.Content)
	case tea.KeyPressMsg:
		m.flash = ""
		lastG := m.lastG
		m.lastG = time.Time{}
		switch {
		case m.conflict != nil:
			m.conflictKey(msg)
		case m.editor != nil:
			if m.complete == nil || !m.completionKey(msg) {
				m.editorKey(msg)
				m.updateCompletion()
			}
		case m.manual != nil:
			m.manualKey(msg)
		case m.confirm != nil:
			m.confirmKey(msg)
		case m.hints != nil:
			m.hintKey(msg)
		case m.prompt != nil:
			m.promptKey(msg)
		case m.chooser != nil:
			m.chooserKey(msg)
		case m.search != nil:
			m.searchKey(msg)
		default:
			key := msg.String()
			if key == "G" {
				// G goes to the bottom at once; a second G right after
				// makes it GG, which goes to the top.
				if !lastG.IsZero() && time.Since(lastG) < ggWait {
					cmd = m.do(actTop)
					break
				}
				m.lastG = time.Now()
			}
			if act, ok := m.keys[key]; ok {
				if act == actQuit {
					return m, tea.Quit
				}
				cmd = m.do(act)
			}
		}
	}
	m.settle()
	return m, cmd
}

// setPalette recolours everything, the logo included.
func (m *Model) setPalette(p theme.Palette) {
	m.pal = p
	m.st = newStyles(p)
	m.logo, m.logoW = logo.Header(p)
	m.splash = m.splash[:0]
	for scale := 1; scale <= 2; scale++ {
		art, _ := logo.Splash(p, scale)
		m.splash = append(m.splash, art)
	}
	if m.editor != nil {
		m.editor.SetPalette(p)
	}
}

func (m *Model) do(a action) tea.Cmd {
	// Asking for Files leaves zen mode.
	if m.zen && (a == actPane1 || a == actNextPane || a == actPrevPane || (a == actLeft && m.focus == paneNote)) {
		m.zen = false
	}
	switch a {
	case actNextPane, actPrevPane:
		if m.focus == paneFiles {
			m.focus = paneNote
		} else {
			m.focus = paneFiles
		}
	case actPane1:
		m.focus = paneFiles
	case actPane2:
		m.focus = paneNote
	case actNewNote:
		m.startCreate(promptNewNote)
	case actNewFolder:
		m.startCreate(promptNewFolder)
	case actRename, actMove, actDelete:
		m.fileAction(a)
	case actUndoOp:
		m.undoOp()
	case actDaily:
		m.openDaily()
	case actEscape:
		if m.zen {
			m.zen = false
		} else {
			m.escape()
		}
	case actZen:
		m.toggleZen()
	case actHelp:
		m.openManual()
	case actEdit:
		m.startEdit()
	case actEditExternal:
		return m.startExternal()
	case actUndoEdit:
		m.restoreVersion(false)
	case actRedoEdit:
		m.restoreVersion(true)
	case actHints:
		m.startHints(false)
	case actBacklinks:
		m.showBacklinks()
	case actOutline:
		m.showOutline()
	case actNextHeading:
		m.headingJump(1)
	case actPrevHeading:
		m.headingJump(-1)
	case actBack:
		if !m.goBack(false) {
			m.flash = "Nothing to go back to"
		}
	case actForward:
		if !m.goBack(true) {
			m.flash = "Nothing to go forward to"
		}
	case actSearch:
		m.openSearch()
	case actSwitcher:
		m.openSwitcher()
	default:
		switch m.focus {
		case paneFiles:
			m.filesAction(a)
		case paneNote:
			m.noteAction(a)
		}
	}
	return nil
}

func (m *Model) filesAction(a action) {
	f := &m.files
	switch a {
	case actOpen:
		m.openRow()
	case actRight:
		m.visual = nil // the rows are about to change under the range
		if !f.in() {
			m.openRow()
		}
	case actLeft:
		m.visual = nil
		f.out()
	case actParent:
		f.up()
	case actCollapseAll:
		m.visual = nil
		f.collapseAll()
	case actMark:
		m.toggleMark()
	case actVisual:
		m.toggleVisual()
	case actMarkAll:
		m.markAll()
	default:
		if c := step(f.cur, len(f.rows), a, m.layout().bodyH-2); c != f.cur {
			f.cur = c
			if m.visual != nil {
				m.applyVisual()
			}
		}
	}
}

// openRow opens what's under the cursor in Files: a folder opens or closes,
// a note opens on the right, and any other file opens in its own app.
func (m *Model) openRow() {
	e := m.files.selected()
	switch {
	case e.IsDir:
		m.visual = nil
		m.files.toggle()
	case e.Rel == m.notePath:
		m.focus = paneNote
	case vault.IsNote(e.Name):
		m.open(e.Rel)
	default:
		m.openExternal(m.vault.Abs(e.Rel))
	}
}

func (m *Model) noteAction(a action) {
	vis := m.layout().bodyH - 2
	maxOff := max(len(m.lines)-vis, 0)
	off := m.noteOff
	switch a {
	case actDown:
		off++
	case actUp:
		off--
	case actHalfDown:
		off += max(vis/2, 1)
	case actHalfUp:
		off -= max(vis/2, 1)
	case actTop:
		off = 0
	case actBottom:
		off = maxOff
	case actLeft:
		m.focus = paneFiles
		return
	case actOpen:
		m.startHints(true)
		return
	case actParent:
		if !m.goBack(false) {
			m.focus = paneFiles
		}
		return
	}
	m.noteOff = clamp(off, 0, maxOff)
}

// toggleZen shows the open note alone, centred, or brings the panels back.
func (m *Model) toggleZen() {
	switch {
	case m.zen:
		m.zen = false
	case m.notePath == "":
		m.flash = "Open a note first: zen mode shows just the note"
	default:
		m.zen, m.focus = true, paneNote
	}
}

// step moves a cursor over n rows; page is the visible height.
func step(cur, n int, a action, page int) int {
	switch a {
	case actDown:
		cur++
	case actUp:
		cur--
	case actTop:
		cur = 0
	case actBottom:
		cur = n - 1
	case actHalfDown:
		cur += max(page/2, 1)
	case actHalfUp:
		cur -= max(page/2, 1)
	}
	return clamp(cur, 0, max(n-1, 0))
}

// cwd is the current folder, where n and N create things. In Files it's the
// folder under the cursor, or the one holding the file under it; in the
// note pane it's the open note's folder.
func (m *Model) cwd() string {
	if m.focus == paneNote && m.notePath != "" {
		return parentOf(m.notePath)
	}
	return m.files.folder()
}

// reload rescans the vault after a change on disk, keeping the user's place:
// open folders, the cursor in Files and the open note with its scroll
// position. An open note that is gone closes.
func (m *Model) reload() error {
	entries, err := m.vault.Entries()
	if err != nil {
		return err
	}
	m.files.set(entries, m.vault.Name())
	if err := m.idx.Update(m.vault); err != nil {
		return err
	}
	for p := range m.marks {
		if _, ok := m.files.entry(p); !ok {
			delete(m.marks, p)
		}
	}
	m.loadNote()
	return nil
}

// showNote puts note rel in the note pane, scrolled to the top.
func (m *Model) showNote(rel string) {
	m.notePath, m.noteOff, m.hints = rel, 0, nil
	m.loadNote()
}

// loadNote reads the open note again. If it's gone, the note pane empties.
func (m *Model) loadNote() {
	m.noteSrc, m.noteErr, m.lines, m.renderedW = "", nil, nil, 0
	if m.notePath == "" {
		return
	}
	src, err := m.vault.Read(m.notePath)
	if errors.Is(err, fs.ErrNotExist) {
		m.notePath, m.noteOff, m.hints = "", 0, nil
		return
	}
	m.noteSrc, m.noteErr = src, err
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// settle re-renders the note when its pane width changed and keeps every
// cursor inside its scroll window. It runs after every update.
func (m *Model) settle() {
	if m.width == 0 {
		return
	}
	if m.zen && m.notePath == "" && m.editor == nil {
		m.zen = false // the note closed under us
	}
	l := m.layout()
	vis := l.bodyH - 2
	if m.editor != nil {
		m.editor.SetSize(l.noteTextW(), vis)
	}
	if m.notePath != "" && m.noteErr == nil && l.noteTextW() != m.renderedW {
		m.lines = markdown.Render(m.noteSrc, markdown.Options{Width: l.noteTextW(), Palette: m.pal, Resolve: m.resolve})
		m.renderedW = l.noteTextW()
	}
	if m.jumpSrc >= 0 && m.editor == nil {
		for i, ln := range m.lines {
			if ln.Src >= m.jumpSrc {
				m.noteOff = i
				break
			}
		}
		m.jumpSrc = -1
	}
	m.files.off = scrollTo(m.files.cur, m.files.off, vis, len(m.files.rows))
	m.noteOff = clamp(m.noteOff, 0, max(len(m.lines)-vis, 0))
}

func scrollTo(cur, off, vis, n int) int {
	if vis <= 0 {
		return cur
	}
	if cur < off {
		off = cur
	}
	if cur >= off+vis {
		off = cur - vis + 1
	}
	return clamp(off, 0, max(n-vis, 0))
}

// resolve reports whether a wikilink in the shown note leads anywhere.
func (m *Model) resolve(target string) bool {
	_, ok := m.idx.Resolve(target, m.notePath)
	return ok
}

type layout struct{ filesW, noteW, bodyH int }

// noteTextW is the width notes render at: the pane minus borders and a
// one-cell margin on each side.
func (l layout) noteTextW() int { return l.noteW - 4 }

func (m *Model) layout() layout {
	if m.zen {
		// A title row above the note and a spare row below it.
		return layout{noteW: min(zenWidth, max(m.width-4, 10)) + 4, bodyH: max(m.height, 3)}
	}
	l := layout{bodyH: max(m.height-headerHeight-statusHeight, 3)}
	if m.width >= filesAutoHideWidth || m.focus == paneFiles {
		l.filesW = clamp(m.width*30/100, 24, 40)
	}
	l.noteW = m.width - l.filesW
	if l.filesW > 0 && l.noteW < 14 {
		l.filesW = max(m.width-14, 12)
		l.noteW = m.width - l.filesW
	}
	return l
}

// xdgOpen opens target with the desktop's app for it.
func xdgOpen(target string) error {
	c := exec.Command("xdg-open", target)
	if err := c.Start(); err != nil {
		return err
	}
	go c.Wait() // don't leave a zombie behind
	return nil
}

func parentOf(p string) string {
	if d := path.Dir(p); d != "." {
		return d
	}
	return ""
}

func clamp(v, lo, hi int) int { return max(lo, min(v, hi)) }
