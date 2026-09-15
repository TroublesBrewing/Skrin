// Package ui is Skrin's Bubble Tea front end: three panes (folder tree,
// folder contents, note) between a header and a status line.
package ui

import (
	"fmt"
	"path"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/logo"
	"github.com/lurioso/skrin/internal/markdown"
	"github.com/lurioso/skrin/internal/theme"
	"github.com/lurioso/skrin/internal/vault"
)

type pane int

const (
	paneTree pane = iota
	paneList
	paneNote
)

const (
	headerHeight = 3
	statusHeight = 1
	// Narrower than this, the tree pane only shows while it has focus.
	treeAutoHideWidth = 100
)

// ThemeMsg carries the reloaded palette after Omarchy switches theme.
type ThemeMsg struct{ Palette theme.Palette }

// VaultChangedMsg reports that files changed on disk.
type VaultChangedMsg struct{}

// Options tune behaviour that depends on the world outside the vault.
type Options struct {
	// RolloverTodos makes `t` carry unfinished todos into a new daily note.
	RolloverTodos bool
	// ObsidianOpen reports whether Obsidian desktop has this vault open. Its
	// Rollover plugin then does the rollover, so Skrin must not.
	ObsidianOpen func() bool
	// Now is the clock; tests pin it.
	Now func() time.Time
}

// Model is the root Bubble Tea model.
type Model struct {
	vault *vault.Vault
	pal   theme.Palette
	st    styles
	keys  map[string]action
	opts  Options
	logo  []string
	logoW int

	width, height int
	focus         pane

	tree    tree
	cwd     string // current folder, where new notes are created
	entries []vault.Entry
	listCur int
	listOff int

	links map[string]bool // lower-cased paths and names a wikilink can resolve to

	notePath  string // selected file ("" when a folder is selected)
	isNote    bool   // notePath is a markdown note
	noteSrc   string
	noteErr   error
	dirInfo   string // summary shown when a folder is selected
	lines     []markdown.Line
	renderedW int // width lines were rendered at; 0 forces a re-render
	noteOff   int

	journal vault.Journal
	marks   map[string]bool // marked items by vault path; may span folders
	visual  *visualRange

	// At most one of these is open; it takes all keys while it is.
	prompt  *prompt
	confirm *confirm
	picker  *picker

	flash string // one-shot status message, cleared by the next key
}

// New builds the model for vault v.
func New(v *vault.Vault, pal theme.Palette, opts Options) (*Model, error) {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.ObsidianOpen == nil {
		opts.ObsidianOpen = func() bool { return false }
	}
	m := &Model{vault: v, keys: keymap(), tree: newTree(), opts: opts, marks: map[string]bool{}}
	m.setPalette(pal)
	if lines, w, err := logo.HalfBlock(headerHeight); err == nil {
		m.logo, m.logoW = lines, w
	}
	if err := m.reload(); err != nil {
		return nil, err
	}
	return m, nil
}

// Flash shows a one-shot message in the status line.
func (m *Model) Flash(s string) { m.flash = s }

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case ThemeMsg:
		m.setPalette(msg.Palette)
		m.renderedW = 0
		m.flash = "theme: " + msg.Palette.Name
	case VaultChangedMsg:
		if err := m.reload(); err != nil {
			m.flash = "refresh failed: " + err.Error()
		}
	case tea.KeyPressMsg:
		m.flash = ""
		switch {
		case m.prompt != nil:
			m.promptKey(msg)
		case m.confirm != nil:
			m.confirmKey(msg)
		case m.picker != nil:
			m.pickerKey(msg)
		default:
			if act, ok := m.keys[msg.String()]; ok {
				if act == actQuit {
					return m, tea.Quit
				}
				m.do(act)
			}
		}
	}
	m.settle()
	return m, nil
}

func (m *Model) setPalette(p theme.Palette) {
	m.pal = p
	m.st = newStyles(p)
}

func (m *Model) do(a action) {
	switch a {
	case actNextPane:
		m.focus = (m.focus + 1) % 3
	case actPrevPane:
		m.focus = (m.focus + 2) % 3
	case actLeft:
		m.focus = max(m.focus-1, paneTree)
	case actRight:
		m.focus = min(m.focus+1, paneNote)
	case actPane1:
		m.focus = paneTree
	case actPane2:
		m.focus = paneList
	case actPane3:
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
		m.escape()
	default:
		switch m.focus {
		case paneTree:
			m.treeAction(a)
		case paneList:
			m.listAction(a)
		case paneNote:
			m.noteAction(a)
		}
	}
}

func (m *Model) treeAction(a action) {
	switch a {
	case actOpen:
		m.tree.toggle()
	case actParent:
		m.goParent()
	default:
		if c := step(m.tree.cur, len(m.tree.rows), a, m.layout().bodyH-2); c != m.tree.cur {
			m.tree.cur = c
			m.setCwd(m.tree.rows[c].path)
		}
	}
}

func (m *Model) listAction(a action) {
	switch a {
	case actOpen:
		if e, ok := m.selected(); ok {
			if e.IsDir {
				m.enterDir(e.Rel, "")
			} else {
				m.focus = paneNote
			}
		}
	case actParent:
		m.goParent()
	case actMark:
		m.toggleMark()
	case actVisual:
		m.toggleVisual()
	case actMarkAll:
		m.markAll()
	default:
		if c := step(m.listCur, len(m.entries), a, m.layout().bodyH-2); c != m.listCur {
			m.listCur = c
			m.preview(false)
			if m.visual != nil {
				m.applyVisual()
			}
		}
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
	case actParent:
		m.focus = paneList
	}
	m.noteOff = clamp(off, 0, maxOff)
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

func (m *Model) goParent() {
	if m.cwd == "" {
		return
	}
	m.enterDir(parentOf(m.cwd), m.cwd)
}

// enterDir makes dir the current folder, selecting entry sel if given.
func (m *Model) enterDir(dir, sel string) {
	m.tree.expandTo(dir)
	m.tree.selectPath(dir)
	m.setCwd(dir)
	m.selectRel(sel)
}

// selectRel puts the list cursor on rel if it is in the current folder.
func (m *Model) selectRel(rel string) {
	for i, e := range m.entries {
		if e.Rel == rel {
			m.listCur = i
			m.preview(false)
			return
		}
	}
}

func (m *Model) setCwd(dir string) {
	if dir == m.cwd {
		return
	}
	m.cwd = dir
	m.visual = nil
	if err := m.loadEntries(false); err != nil {
		m.flash = err.Error()
	}
}

// reload rescans the vault after a change on disk, keeping the user's place.
// If the current folder is gone, its nearest surviving parent takes over.
func (m *Model) reload() error {
	dirs, err := m.vault.Dirs()
	if err != nil {
		return err
	}
	m.tree.setDirs(dirs, m.vault.Name())
	for !m.tree.has(m.cwd) {
		m.cwd = parentOf(m.cwd)
	}
	m.tree.expandTo(m.cwd)
	m.tree.selectPath(m.cwd)
	files, err := m.vault.Files()
	if err != nil {
		return err
	}
	m.links = linkIndex(files)
	for p := range m.marks {
		if !m.vault.Exists(p) {
			delete(m.marks, p)
		}
	}
	return m.loadEntries(true)
}

// loadEntries re-reads the current folder. With keep, the cursor stays on
// the same entry and the note keeps its scroll position.
func (m *Model) loadEntries(keep bool) error {
	var sel string
	if e, ok := m.selected(); ok && keep {
		sel = e.Rel
	}
	entries, err := m.vault.List(m.cwd)
	if err != nil {
		return err
	}
	m.entries, m.listCur = entries, 0
	for i, e := range entries {
		if e.Rel == sel {
			m.listCur = i
		}
	}
	if !keep {
		m.listOff = 0
	}
	m.preview(keep)
	return nil
}

func (m *Model) selected() (vault.Entry, bool) {
	if m.listCur < len(m.entries) {
		return m.entries[m.listCur], true
	}
	return vault.Entry{}, false
}

// preview loads whatever the list cursor is on into the note pane.
func (m *Model) preview(keep bool) {
	prev := m.notePath
	m.notePath, m.isNote, m.noteSrc, m.noteErr, m.dirInfo, m.lines = "", false, "", nil, "", nil
	m.renderedW = 0
	e, ok := m.selected()
	switch {
	case !ok:
	case e.IsDir:
		m.dirInfo = m.describeDir(e.Rel)
	case vault.IsNote(e.Name):
		m.notePath, m.isNote = e.Rel, true
		m.noteSrc, m.noteErr = m.vault.Read(e.Rel)
	default:
		m.notePath = e.Rel
	}
	if !keep || m.notePath != prev {
		m.noteOff = 0
	}
}

func (m *Model) describeDir(rel string) string {
	entries, err := m.vault.List(rel)
	if err != nil {
		return "unreadable: " + err.Error()
	}
	notes, dirs := 0, 0
	for _, e := range entries {
		switch {
		case e.IsDir:
			dirs++
		case vault.IsNote(e.Name):
			notes++
		}
	}
	return plural(notes, "note") + " · " + plural(dirs, "folder")
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
	l := m.layout()
	if m.isNote && m.noteErr == nil && l.noteTextW() != m.renderedW {
		m.lines = markdown.Render(m.noteSrc, markdown.Options{Width: l.noteTextW(), Palette: m.pal, Resolve: m.resolve})
		m.renderedW = l.noteTextW()
	}
	vis := l.bodyH - 2
	m.tree.off = scrollTo(m.tree.cur, m.tree.off, vis, len(m.tree.rows))
	m.listOff = scrollTo(m.listCur, m.listOff, vis, len(m.entries))
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

func (m *Model) resolve(target string) bool {
	t := strings.ToLower(strings.TrimSpace(target))
	return m.links[t] || m.links[t+".md"]
}

// linkIndex maps every name a wikilink may use for a file (its vault path
// or bare file name, lower-cased) to true.
func linkIndex(files []string) map[string]bool {
	idx := make(map[string]bool, 2*len(files))
	for _, f := range files {
		lf := strings.ToLower(f)
		idx[lf] = true
		idx[path.Base(lf)] = true
	}
	return idx
}

type layout struct{ treeW, listW, noteW, bodyH int }

// noteTextW is the width notes render at: the pane minus borders and a
// one-cell margin on each side.
func (l layout) noteTextW() int { return l.noteW - 4 }

func (m *Model) layout() layout {
	l := layout{bodyH: max(m.height-headerHeight-statusHeight, 3)}
	if m.width >= treeAutoHideWidth || m.focus == paneTree {
		l.treeW = clamp(m.width*20/100, 18, 32)
	}
	l.listW = clamp(m.width*26/100, 22, 40)
	l.noteW = m.width - l.treeW - l.listW
	if l.noteW < 14 {
		l.listW = max(m.width-l.treeW-14, 12)
		l.noteW = m.width - l.treeW - l.listW
	}
	return l
}

func parentOf(p string) string {
	if d := path.Dir(p); d != "." {
		return d
	}
	return ""
}

func clamp(v, lo, hi int) int { return max(lo, min(v, hi)) }
