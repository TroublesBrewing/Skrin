// Package ui is Skrin's Bubble Tea front end: Files (the vault as one tree)
// and the open note side by side, between a header and a status line.
package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/config"
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
	paneClaude // typing in the Claude drawer
)

const (
	headerHeight = 3
	statusHeight = 1
	// Narrower than this, Files only shows while it has focus.
	filesAutoHideWidth = 80
	// zenWidth is the widest a note runs in zen mode, for easy reading.
	zenWidth = 80
	// splitMinWidth is the narrowest terminal a split view fits in.
	splitMinWidth = 80
	// splitFilesW is Files' width while the view is split: its minimum.
	splitFilesW = 24
	// flashLinger is how long a message nobody asked for stays. Flashes
	// that answer a keypress wait for the next key, because you pressed
	// something and are owed an answer; one that answers a resize has to
	// take itself away again, or zen mode keeps a status line it only
	// shows while there is a flash.
	flashLinger = 2 * time.Second
)

// ThemeMsg carries the reloaded palette after Omarchy switches theme.
type ThemeMsg struct{ Palette theme.Palette }

// VaultChangedMsg reports that files changed on disk.
type VaultChangedMsg struct{}

// WatchTroubleMsg reports that live updates have holes in them: the watcher
// hit the system's limit, or errored. It is shown, never swallowed.
type WatchTroubleMsg struct{ Text string }

// flashDoneMsg asks for a flash to go once it has been read. It names the
// message it means, so a timer started for one flash never clears a newer
// one that has taken its place.
type flashDoneMsg struct{ text string }

// flashFor shows s and takes it away again after flashLinger, unless
// something else has replaced it by then.
func (m *Model) flashFor(s string) tea.Cmd {
	m.flash = s
	return tea.Tick(flashLinger, func(time.Time) tea.Msg { return flashDoneMsg{s} })
}

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
	// RestoreLastNote reopens Session.Open on start; off means a fresh
	// welcome screen every run, so a vault with private notes never
	// opens one by surprise. Editable from the Settings tab of `?`.
	RestoreLastNote bool
	// Assistant sets up the Claude drawer.
	Assistant AssistantOptions
	// Library sets up the Book Card (B): folders, default status and the
	// metadata/cover lookups it makes.
	Library LibraryOptions
	// Images turns on block-art previews for image embeds; off means the
	// placeholder frame always, everywhere. Either way a found embed's
	// name, dimensions and size still show.
	Images bool
	// LineNumbers turns on line numbers along the left edge of notes and
	// the built-in editor.
	LineNumbers bool
	// Config is the config.toml Skrin loaded, kept so the Settings tab
	// of `?` can show and persist toggles back to it. Its own bare
	// fields (Vim, RolloverTodos, ...) above are what the rest of Skrin
	// actually reads; a settings toggle updates both.
	Config config.Config
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
	opts  Options
	logo  []string
	logoW int
	// splash is the large logo at scale 1 and 2, for the empty note pane.
	splash [][]string

	width, height int
	focus         pane
	zen           bool // z: only the note, centred at a readable width

	files files       // the Files pane; its cursor sits on the open note
	order vault.Order // Files' manual order, from skrin.json

	keys *keymap // the keymap in force: the registry plus the user's own

	// The focused note pane. With a split, the other pane waits in split.
	notePath  string // the open note, "" when none is
	noteSrc   string
	noteErr   error
	lines     []markdown.Line
	renderedW int // width lines were rendered at; 0 forces a re-render
	noteOff   int
	jumpSrc   int // after the next render, scroll to this source line; -1 for none

	split     *noteView // the other note of a split view
	splitLeft bool      // the other note sits left of the focused one

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
	hints     *hintState
	prompt    *prompt
	confirm   *confirm
	chooser   *chooser
	search    *searchPanel
	manual    *manual
	book      *bookCard
	habits    *habitView
	quickNote *quickNote

	lastSearch *searchPanel // reopened by the next /

	drawer    drawer
	proposals []*proposal  // Claude's changes waiting for y or n, oldest first
	noteSel   *lineSel     // v in the reading view
	events    chan tea.Msg // from Claude and its tools, on other goroutines

	// thumbCache holds block-art previews for image embeds, keyed by
	// file path + box size (thumbKey); wantThumb is what settle() just
	// asked for and didn't have cached, drained into background jobs at
	// the tail of Update; pendingThumb tracks which file+box combos
	// already have a job in flight, so a slow file isn't re-decoded
	// every keystroke.
	thumbCache   map[string]thumbEntry
	wantThumb    []thumbRequest
	pendingThumb map[string]bool

	flash string // one-shot status message, cleared by the next key

	// lastPeekCur and lastPeekRel track the Files row peeked last time.
	// notes only open when the cursor moves to a different row in Files
	// (or when explicitly opened with Enter/l), so background events
	// (window resize, theme reload, fsnotify) or opening modals (? settings)
	// never reopen a note by surprise when RestoreLastNote is off.
	lastPeekCur int
	lastPeekRel string
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
		vault: v, idx: index.New(), snaps: snapshot.Open(v.Root), files: newFiles(),
		opts: opts, marks: map[string]bool{}, jumpSrc: -1, events: make(chan tea.Msg, 256),
		thumbCache: map[string]thumbEntry{}, keys: newKeymap(opts.Config.Keys),
	}
	m.journal.Keep = m.snaps.Save // U keeps what's on disk before it restores
	m.drawer.input = editor.New("", false, pal)
	m.drawer.id, m.drawer.right = opts.Session.Claude, opts.Assistant.Right
	switch opts.Session.Drawer {
	case "right":
		m.drawer.right, m.drawer.moved = true, true
	case "bottom":
		m.drawer.right, m.drawer.moved = false, true
	}
	m.setPalette(pal)
	for _, p := range opts.Session.Expanded {
		m.files.expanded[p] = true
	}
	if err := m.reload(); err != nil {
		return nil, err
	}
	m.files.selectPath(opts.Session.Cursor)
	if rel := opts.Session.Open; opts.RestoreLastNote && vault.IsNote(rel) && m.vault.Exists(rel) {
		m.showNote(rel)
		m.noteOff = opts.Session.Offset // clamped once the note is rendered
	}
	m.lastPeekCur = m.files.cur
	m.lastPeekRel = m.files.selected().Rel
	return m, nil
}

// Session is where you are now, for the next run to pick up. A split isn't
// kept: the focused note is.
func (m *Model) Session() session.State {
	return session.State{
		Expanded: m.files.openFolders(),
		Cursor:   m.files.selected().Rel,
		Open:     m.notePath,
		Offset:   m.noteOff,
		Claude:   m.drawer.id,
		Drawer:   m.drawerSide(),
	}
}

// Flash shows a one-shot message in the status line.
func (m *Model) Flash(s string) { m.flash = s }

// sizeNote is the resize indicator. Ctrl+- and Ctrl++ are the terminal's
// own keys, not Skrin's: the terminal changes its font and only tells
// Skrin the new size in columns and rows, so the size is what Skrin can
// honestly report. Zoom in and the numbers fall, zoom out and they rise.
// It also names the one consequence that otherwise looks like a fault —
// Files going away — since that is what crossing 80 columns does. A split
// closing says so for itself, from settle, and that message wins.
func (m *Model) sizeNote() string {
	s := fmt.Sprintf("%d × %d", m.width, m.height)
	if m.width < filesAutoHideWidth {
		s += " · Files hides unless focused"
	}
	return s
}

func (m *Model) Init() tea.Cmd {
	return m.listen()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		was := [2]int{m.width, m.height}
		m.width, m.height = msg.Width, msg.Height
		// The first size isn't a change, and the terminal re-reporting
		// the size it already had isn't one either.
		if was[0] != 0 && was != [2]int{m.width, m.height} {
			cmd = m.flashFor(m.sizeNote())
		}
	case flashDoneMsg:
		if m.flash == msg.text {
			m.flash = ""
		}
	case ThemeMsg:
		m.setPalette(msg.Palette)
		m.renderedW = 0
		if m.split != nil {
			m.split.renderedW = 0
		}
		m.flash = "theme: " + msg.Palette.Name
	case VaultChangedMsg:
		open := m.notePath
		if err := m.reload(); err != nil {
			m.flash = "refresh failed: " + err.Error()
		} else if open != "" && m.notePath == "" {
			m.flash = displayName(open) + " is gone: deleted or renamed outside Skrin"
		}
	case WatchTroubleMsg:
		m.flash = msg.Text
	case claudeMsg:
		m.claudeEvent(msg)
		cmd = m.listen()
	case toolMsg:
		m.toolCall(msg)
		cmd = m.listen()
	case externalDoneMsg:
		m.externalDone(msg)
	case bookLookupMsg:
		m.bookLookupDone(msg)
	case bookSaveMsg:
		m.finishBookSave(msg)
	case imageThumbMsg:
		if msg.stated {
			key := thumbKey(msg.abs, msg.cols, msg.rows)
			m.thumbCache[key] = thumbEntry{mtime: msg.mtime, size: msg.size, lines: msg.lines}
			m.renderedW = 0
			if m.split != nil {
				m.split.renderedW = 0
			}
		}
		delete(m.pendingThumb, thumbKey(msg.abs, msg.cols, msg.rows))
	case tea.PasteMsg:
		m.paste(msg.Content)
	case tea.KeyPressMsg:
		m.flash = ""
		lastG := m.lastG
		m.lastG = time.Time{}
		switch {
		case m.conflict != nil:
			m.conflictKey(msg)
		case len(m.proposals) > 0:
			m.proposalKey(msg)
		case m.focus == paneClaude:
			m.drawerKey(msg)
		case m.editor != nil:
			if m.complete == nil || !m.completionKey(msg) {
				cmd = m.editorKey(msg)
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
		case m.book != nil:
			cmd = m.bookCardKey(msg)
		case m.habits != nil:
			m.habitKey(msg)
		case m.quickNote != nil:
			m.quickNoteKey(msg)
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
			if act := m.actionIn(inMain, key); act != actNone {
				if act == actQuit {
					if key == "ctrl+c" && m.noteSel != nil {
						cmd = m.copySelection()
						break
					}
					return m, tea.Quit
				}
				cmd = m.do(act)
			}
		}
	}
	m.settle()
	return m, tea.Batch(cmd, m.startThumbnails())
}

// setPalette recolours everything, the logo included.
func (m *Model) setPalette(p theme.Palette) {
	m.pal = p
	m.st = newStyles(p)
	m.logo, m.logoW = logo.Header(p)
	if m.drawer.input != nil {
		m.drawer.input.SetPalette(p)
	}
	for i := range m.drawer.msgs {
		m.drawer.msgs[i].lines = nil // render again in the new colours
	}
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
	case actNewBook:
		m.openBookCard()
	case actHabits:
		m.openHabits()
	case actQuickNote:
		m.openQuickNote()
	case actEscape:
		switch {
		case m.noteSel != nil:
			m.noteSel = nil
		case m.split != nil && m.focus == paneNote:
			m.closePane()
		case m.zen:
			m.zen = false
		default:
			m.escape()
		}
	case actOrderUp, actOrderDown:
		m.shiftItem(a == actOrderDown)
	case actSkimDown, actSkimUp:
		m.skimSplit(a == actSkimDown)
	case actFolderJumpUp, actFolderJumpDown:
		m.folderJump(a == actFolderJumpDown)
	case actOrderReset:
		m.resetLevel()
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
	case actLineNumbers:
		m.toggleLineNumbers()
	case actBacklinks:
		m.showBacklinks()
	case actOutline:
		m.showOutline()
	case actNextHeading:
		m.headingJump(1)
	case actPrevHeading:
		m.headingJump(-1)
	case actBack:
		if m.focus == paneFiles {
			m.filesAction(actLeft)
		} else if !m.goBack(false) {
			m.flash = "Nothing to go back to"
		}
	case actForward:
		if m.focus == paneFiles {
			m.filesAction(actRight)
		} else if !m.goBack(true) {
			m.flash = "Nothing to go forward to"
		}
	case actClaude:
		m.toggleDrawer()
	case actClaudeInput:
		m.openDrawer()
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
		m.visual = nil // the rows may change under the range
		isDir, alreadyOpen := f.openFolder()
		switch {
		case !isDir:
			m.openRow()
		case alreadyOpen:
			m.flash = "Already open — Enter toggles it closed"
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
	case actPaneRight:
		m.focusSplit()
	default:
		if c := step(f.cur, len(f.rows), a, m.layout().bodyH-2); c != f.cur {
			f.cur = c
			if m.visual != nil {
				m.applyVisual()
			}
		}
	}
}

// focusSplit is Shift+→ from Files: the "done browsing, now look at what I
// skimmed beside" step the skim flash promises. It swaps the split's note
// into focus — the same swap openRow does when the cursor lands back on the
// split's own note — so the note you skimmed to becomes the interactive
// one, not just whichever happens to render on the right. With no split,
// it's a no-op; there is nothing to Files' own left to focus, so Shift+←
// from Files stays unhandled.
func (m *Model) focusSplit() {
	if m.split != nil {
		m.swapPanes()
		m.focus = paneNote
	}
}

// folderJump is Ctrl+↑/↓: the cursor moves to the previous/next folder row
// among the visible rows, wherever the cursor starts. Silent on success
// (a cursor move, nothing more); loud on refusal, like the arrange keys.
func (m *Model) folderJump(down bool) {
	if !m.inFiles("jump between folders") {
		return
	}
	switch m.files.folderJump(down) {
	case folderJumpEdge:
		edge := "top"
		if down {
			edge = "bottom"
		}
		m.flash = "Already at the tree's " + edge
	case folderJumpNone:
		m.flash = "No folders to jump to"
	}
}

// openRow is Enter in Files: a folder opens or closes, a note (already open
// under the cursor) gets the focus, and any other file opens in its own app.
// A note that's already showing in the split pane gets the focus there
// instead of being reopened into the main pane — the same courtesy peek
// gives j/k, so the reference note is never quietly discarded.
func (m *Model) openRow() {
	e := m.files.selected()
	switch {
	case e.IsDir:
		m.visual = nil
		m.files.toggle()
	case vault.IsNote(e.Name):
		switch {
		case m.split != nil && e.Rel == m.split.path:
			m.swapPanes()
		case e.Rel != m.notePath:
			m.showNote(e.Rel)
		}
		m.focus = paneNote
	default:
		m.openExternal(m.vault.Abs(e.Rel))
	}
	m.lastPeekCur = m.files.cur
	m.lastPeekRel = m.files.selected().Rel
}

// peek opens the note under the Files cursor: notes follow the cursor. A
// folder, or a file that isn't a note, leaves the open note as it is.
// The split's own note is skipped: the skim put it beside on purpose, and
// the reference note in the main pane stays put while the cursor browses.
func (m *Model) peek() {
	e := m.files.selected()
	if e.IsDir || !vault.IsNote(e.Name) {
		return
	}
	if m.split != nil && e.Rel == m.split.path {
		return
	}
	if e.Rel != m.notePath {
		m.showNote(e.Rel)
	}
}

func (m *Model) noteAction(a action) {
	vis := m.layout().bodyH - 2
	if a == actVisual {
		m.toggleNoteSel()
		return
	}
	if m.noteSel != nil && m.moveNoteSel(a, vis) {
		return
	}
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
	case actPaneLeft, actPaneRight:
		// Move to the other pane when it lies that way.
		if m.split != nil && (a == actPaneLeft) == m.splitLeft {
			m.swapPanes()
		}
		return
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
// A split's other note is closed rather than hidden.
func (m *Model) toggleZen() {
	switch {
	case m.zen:
		m.zen = false
	case m.notePath == "":
		m.flash = "Select a note to read in zen mode"
	default:
		if m.split != nil {
			m.flash = "Zen: closed " + displayName(m.split.path)
			m.split = nil
		}
		m.zen, m.focus = true, paneNote
	}
}

// toggleLineNumbers turns line numbers on or off across notes and the editor.
func (m *Model) toggleLineNumbers() {
	m.opts.LineNumbers = !m.opts.LineNumbers
	m.opts.Config.Render.LineNumbers = boolPtr(m.opts.LineNumbers)
	_ = config.Save(m.opts.Config)
	if m.opts.LineNumbers {
		m.flash = "Line numbers on"
	} else {
		m.flash = "Line numbers off"
	}
	m.renderedW = 0
	if m.split != nil {
		m.split.renderedW = 0
	}
	if m.editor != nil {
		m.editor.SetLineNumbers(m.opts.LineNumbers)
	}
	m.settle()
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
// open folders, the cursor in Files and the open notes with their scroll
// positions. An open note that is gone closes.
func (m *Model) reload() error {
	entries, err := m.vault.Entries()
	if err != nil {
		return err
	}
	m.order = m.vault.LoadOrder()
	m.files.set(entries, m.vault.Name(), m.order)
	if err := m.idx.Update(m.vault); err != nil {
		return err
	}
	for p := range m.marks {
		if _, ok := m.files.entry(p); !ok {
			delete(m.marks, p)
		}
	}
	m.loadNote()
	m.loadSplit()
	m.lastPeekCur = m.files.cur
	m.lastPeekRel = m.files.selected().Rel
	return nil
}

// showNote puts note rel in the note pane, scrolled to the top.
func (m *Model) showNote(rel string) {
	m.notePath, m.noteOff, m.hints, m.noteSel = rel, 0, nil, nil
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

// settle keeps things in step after every update: the note under the Files
// cursor is open, a split closes when it no longer fits, notes are
// re-rendered when their pane width changed, and every cursor stays inside
// its scroll window.
func (m *Model) settle() {
	if m.width == 0 {
		return
	}
	if m.focus == paneFiles && m.editor == nil {
		if m.files.cur != m.lastPeekCur || m.files.selected().Rel != m.lastPeekRel {
			m.lastPeekCur = m.files.cur
			m.lastPeekRel = m.files.selected().Rel
			m.peek()
		}
	}
	if m.split != nil && m.width < splitMinWidth {
		m.flash = "No room for a split: closed " + displayName(m.split.path)
		m.split = nil
	}
	if m.zen && m.notePath == "" && m.editor == nil {
		m.zen = false // the note closed under us
	}
	l := m.layout()
	vis := l.bodyH - 2
	textW := m.noteTextW()
	if m.editor != nil {
		m.editor.SetSize(textW, vis)
	}
	if m.notePath != "" && m.noteErr == nil && textW != m.renderedW {
		m.lines = markdown.Render(m.noteSrc, markdown.Options{Width: textW, Palette: m.pal, Resolve: m.resolve, Images: m.imageOptions(m.notePath, vis), Embeds: m.embedOptions(m.notePath)})
		m.renderedW = textW
	}
	if s := m.split; s != nil {
		w := l.splitW - 4
		if m.opts.LineNumbers && s.src != "" {
			digits := max(len(strconv.Itoa(strings.Count(s.src, "\n")+1)), 2)
			w -= (digits + 3) - 1
		}
		w = max(w, 10)
		if s.err == nil && w != s.renderedW {
			s.lines = markdown.Render(s.src, markdown.Options{Width: w, Palette: m.pal, Resolve: m.resolveFrom(s.path), Images: m.imageOptions(s.path, vis), Embeds: m.embedOptions(s.path)})
			s.renderedW = w
		}
		s.off = clamp(s.off, 0, max(len(s.lines)-vis, 0))
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
	if s := m.noteSel; s != nil {
		if len(m.lines) == 0 {
			m.noteSel = nil
		} else {
			s.anchor, s.cur = clamp(s.anchor, 0, len(m.lines)-1), clamp(s.cur, 0, len(m.lines)-1)
		}
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

// totalSrcLines counts lines in the open note's source text.
func (m *Model) totalSrcLines() int {
	if m.noteSrc == "" {
		return 1
	}
	return strings.Count(m.noteSrc, "\n") + 1
}

// gutterWidth is the width of the line numbers column (e.g. " 1 │ ").
func (m *Model) gutterWidth() int {
	if !m.opts.LineNumbers {
		return 0
	}
	digits := max(len(strconv.Itoa(m.totalSrcLines())), 2)
	return digits + 3
}

// noteTextW is the width notes and editor render at: the pane minus
// borders, line number gutter (if on), and a one-cell margin.
func (m *Model) noteTextW() int {
	l := m.layout()
	w := l.noteW - 4
	if m.opts.LineNumbers {
		w -= m.gutterWidth() - 1
	}
	return max(w, 10)
}

// resolve reports whether a wikilink in the open note leads anywhere.
func (m *Model) resolve(target string) bool {
	_, ok := m.idx.Resolve(target, m.notePath)
	return ok
}

// resolveFrom is resolve for links in note from.
func (m *Model) resolveFrom(from string) func(string) bool {
	return func(target string) bool {
		_, ok := m.idx.Resolve(target, from)
		return ok
	}
}

// layout is how the screen is shared out. splitW is the other note of a
// split view; drawerW is the Claude drawer on the right, and drawerH its
// rows along the bottom (1 when folded).
type layout struct{ filesW, noteW, splitW, bodyH, drawerW, drawerH int }

// noteTextW is the width notes render at: the pane minus borders and a
// one-cell margin on each side.
func (l layout) noteTextW() int { return l.noteW - 4 }

func (m *Model) layout() layout {
	if m.zen {
		// A title row above the note and a spare row below it.
		return layout{noteW: min(zenWidth, max(m.width-4, 10)) + 4, bodyH: max(m.height, 3)}
	}
	l := layout{bodyH: max(m.height-headerHeight-statusHeight, 3)}
	w := m.width
	if m.drawer.open {
		switch {
		case m.drawerRight():
			l.drawerW = drawerWidth(m.width)
			w -= l.drawerW
		case m.focus == paneClaude && l.bodyH >= 7:
			l.drawerH = min(clamp(l.bodyH*45/100, 8, 20), l.bodyH-3)
		default:
			l.drawerH = 1 // folded while you're elsewhere
		}
		l.bodyH -= l.drawerH
	}
	if m.split != nil {
		// Files shrinks to its minimum and the notes share the rest.
		l.filesW = splitFilesW
		l.splitW = (w - l.filesW) / 2
		l.noteW = w - l.filesW - l.splitW
		return l
	}
	if w >= filesAutoHideWidth || m.focus == paneFiles {
		l.filesW = clamp(w*30/100, 24, 40)
	}
	l.noteW = w - l.filesW
	if l.filesW > 0 && l.noteW < 14 {
		l.filesW = max(w-14, 12)
		l.noteW = w - l.filesW
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
