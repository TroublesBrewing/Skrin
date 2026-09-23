package ui

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/daily"
	"github.com/lurioso/skrin/internal/habit"
	"github.com/lurioso/skrin/internal/obsidian"
	"github.com/lurioso/skrin/internal/vault"
)

// visualRange is an active v selection: marks on the Files rows from anchor
// to the cursor, on top of the marks that existed before it started.
type visualRange struct {
	anchor int
	base   map[string]bool
}

func (m *Model) toggleMark() {
	e := m.files.selected()
	if e.Rel == "" {
		return
	}
	m.visual = nil
	if m.marks[e.Rel] {
		delete(m.marks, e.Rel)
	} else {
		m.marks[e.Rel] = true
	}
}

// markAll marks the cursor's row and everything next to it in its folder,
// or unmarks them all when they already are.
func (m *Model) markAll() {
	m.visual = nil
	sibs := m.files.siblings()
	all := true
	for _, e := range sibs {
		all = all && m.marks[e.Rel]
	}
	for _, e := range sibs {
		if all {
			delete(m.marks, e.Rel)
		} else {
			m.marks[e.Rel] = true
		}
	}
}

func (m *Model) toggleVisual() {
	if m.visual != nil {
		m.visual = nil
		return
	}
	m.visual = &visualRange{anchor: m.files.cur, base: cloneMarks(m.marks)}
	m.applyVisual()
}

func (m *Model) applyVisual() {
	marks := cloneMarks(m.visual.base)
	lo, hi := min(m.visual.anchor, m.files.cur), max(m.visual.anchor, m.files.cur)
	for i := lo; i <= hi && i < len(m.files.rows); i++ {
		if rel := m.files.rows[i].Rel; rel != "" {
			marks[rel] = true
		}
	}
	m.marks = marks
}

// escape cancels a visual range, or else clears all marks.
func (m *Model) escape() {
	switch {
	case m.visual != nil:
		m.marks, m.visual = m.visual.base, nil
	case len(m.marks) > 0:
		m.marks = map[string]bool{}
		m.flash = "Marks cleared"
	}
}

func cloneMarks(src map[string]bool) map[string]bool {
	dst := make(map[string]bool, len(src))
	for k := range src {
		dst[k] = true
	}
	return dst
}

// targets are what r, m and d act on: the marked items if there are any,
// otherwise the row under the cursor in Files, or the open note in the note
// pane. Marked items inside a marked folder are left out, since they go
// wherever the folder goes.
func (m *Model) targets() []string {
	if len(m.marks) > 0 {
		var out []string
		for p := range m.marks {
			if !insideMarked(p, m.marks) {
				out = append(out, p)
			}
		}
		sort.Strings(out)
		return out
	}
	p := m.files.selected().Rel
	if m.focus == paneNote {
		p = m.notePath
	}
	if p == "" {
		return nil
	}
	return []string{p}
}

func insideMarked(p string, marks map[string]bool) bool {
	for q := parentOf(p); q != ""; q = parentOf(q) {
		if marks[q] {
			return true
		}
	}
	return false
}

func (m *Model) fileAction(a action) {
	m.visual = nil
	ts := m.targets()
	if len(ts) == 0 {
		if m.focus == paneNote {
			m.flash = "Select a note to " + map[action]string{actRename: "rename", actMove: "move", actDelete: "delete"}[a]
		} else {
			m.flash = "The vault itself can't be renamed, moved or deleted"
		}
		return
	}
	switch a {
	case actRename:
		if len(ts) > 1 {
			m.flash = "Rename works on one item at a time. Esc clears the marks."
			return
		}
		m.startRename(ts[0])
	case actMove:
		m.startMove(ts)
	case actDelete:
		m.askDelete(ts)
	}
}

func (m *Model) startCreate(k promptKind) {
	what := "New note in "
	if k == promptNewFolder {
		what = "New folder in "
	}
	cwd := m.cwd()
	// For a new note the folder is shown separately in the prompt line,
	// in the accent colour, because it is the thing Tab can change.
	label := what + m.folderLabel(cwd)
	if k == promptNewNote {
		label = what
	}
	m.prompt = &prompt{kind: k, label: label, folder: cwd}
}

// startCreateSplit is Alt+n: a new note in a split beside the note being
// read, so the reference stays put while the new note is written in the
// other pane. With no note open it falls back to an ordinary new note.
func (m *Model) startCreateSplit() {
	if m.notePath == "" || m.width < splitMinWidth {
		m.startCreate(promptNewNote)
		return
	}
	cwd := m.cwd()
	m.prompt = &prompt{kind: promptNewNote, label: "New note in ", folder: cwd, split: true}
}

func (m *Model) startRename(rel string) {
	name := path.Base(rel)
	if !m.vault.IsDir(rel) && vault.IsNote(name) {
		name = strings.TrimSuffix(name, path.Ext(name))
	}
	p := &prompt{kind: promptRename, label: "Rename " + path.Base(rel) + " to", target: rel}
	p.in.set(name)
	m.prompt = p
}

func (m *Model) submitPrompt() tea.Cmd {
	p := m.prompt
	input := strings.TrimSpace(p.in.value())
	var err error
	switch p.kind {
	case promptNewNote:
		err = m.createNote(input, p.folder, p.split)
	case promptNewFolder:
		err = m.createFolder(input)
	case promptRename:
		err = m.rename(p.target, input)
	case promptExtract:
		err = m.extractTo(input)
	case promptOpenFolder:
		err = m.openFolderAsSkrin(input)
	}
	if err != nil {
		p.err = err.Error()
		return nil
	}
	m.prompt = nil
	// Opening a folder hands the terminal to the new vault: quit, and
	// main reopens there.
	if p.kind == promptOpenFolder {
		return tea.Quit
	}
	return nil
}

// createNote makes a note in the folder the prompt resolved (the one the
// cursor was in when it opened, or one picked with Tab). "sub/name"
// creates the folders on the way. An empty name becomes "Untitled", as in
// Obsidian, and so does "sub/" with no name after the slash, inside sub.
// With split, the note opens in a split beside the one being read.
func (m *Model) createNote(input, folder string, split bool) error {
	cwd := folder
	if i := strings.LastIndex(input, "/"); input == "" || (i >= 0 && strings.TrimSpace(input[i+1:]) == "") {
		dir := cwd
		if sub := strings.TrimSpace(strings.TrimSuffix(input, "/")); sub != "" {
			d, err := joinUserPath(cwd, sub)
			if err != nil {
				return err
			}
			dir = d
		}
		rel := path.Join(dir, m.untitledIn(dir, ".md")+".md")
		if split {
			return m.createNoteAtSplit(rel)
		}
		return m.createNoteAt(rel)
	}
	rel, err := joinUserPath(cwd, input)
	if err != nil {
		return err
	}
	if !vault.IsNote(rel) {
		rel += ".md"
	}
	if split {
		return m.createNoteAtSplit(rel)
	}
	return m.createNoteAt(rel)
}

// writeNewNote puts a new note at rel on disk, filled from its folder's
// paired template if any, and records the undoable create. It reports the
// content it was written with, so the caller can say whether a template
// filled it.
func (m *Model) writeNewNote(rel string) (string, error) {
	if m.vault.Exists(rel) {
		return "", fmt.Errorf("%s already exists", rel)
	}
	content := m.newNoteContent(rel)
	dirs, err := m.vault.CreateFile(rel, content)
	steps := createdSteps(dirs)
	if err == nil {
		steps = append(steps, vault.Step{Kind: vault.StepCreated, Rel: rel})
	}
	m.journal.Record(vault.Op{Desc: "create " + rel, Steps: steps})
	if err != nil {
		return "", err
	}
	return content, nil
}

// createdFlash names a fresh create, saying when a folder template filled it.
func createdFlash(rel, content string) string {
	if content != "" {
		return "Created " + rel + " from its folder's template"
	}
	return "Created " + rel
}

// createNoteAt creates an empty note at rel, puts the Files cursor on it and
// opens it in the editor. A paired folder template fills the note instead of
// leaving it empty.
func (m *Model) createNoteAt(rel string) error {
	content, err := m.writeNewNote(rel)
	if err != nil {
		return err
	}
	m.reveal(rel)
	m.pushHistory()
	m.showNote(rel)
	m.openEditor(rel)
	m.flash = createdFlash(rel, content)
	return nil
}

// createNoteAtSplit is createNoteAt for Alt+n: the note being read becomes
// the split pane, and the new note takes the main pane, open in the
// editor — so the reference stays visible while the new note is written.
func (m *Model) createNoteAtSplit(rel string) error {
	content, err := m.writeNewNote(rel)
	if err != nil {
		return err
	}
	m.reveal(rel)
	m.pushHistory()
	m.split = &noteView{path: m.notePath, src: m.noteSrc, err: m.noteErr, off: m.noteOff}
	m.splitLeft = false
	m.zen = false
	m.showNote(rel)
	m.openEditor(rel)
	m.flash = createdFlash(rel, content) + " · " + displayName(m.split.path) + " is beside it"
	return nil
}

// newNoteContent is what a new note at rel starts with: the template
// paired with its folder, expanded and filled in, or "" when none is
// paired. The match is exact — the folder itself, not any parent — so
// Personer and Personer/Vänner can each have their own template.
func (m *Model) newNoteContent(rel string) string {
	folder := parentOf(rel)
	for _, r := range m.opts.FolderTemplates {
		if r.Folder != folder || r.Template == "" {
			continue
		}
		src, err := m.vault.Read(r.Template)
		if err != nil {
			// A template that's gone reads as no template; the note is
			// still created, just empty, and nothing goes missing.
			return ""
		}
		s := obsidian.LoadSettings(m.vault.Root).Templates
		return daily.ExpandTemplate(src, displayName(rel), s.DateFormat, s.TimeFormat, m.opts.Now())
	}
	return ""
}

func (m *Model) createFolder(input string) error {
	cwd := m.cwd()
	input = strings.TrimSpace(strings.TrimRight(input, "/"))
	if input == "" {
		input = m.untitledIn(cwd, "")
	}
	rel, err := joinUserPath(cwd, input)
	if err != nil {
		return err
	}
	if m.vault.Exists(rel) {
		return fmt.Errorf("%s already exists", rel)
	}
	dirs, err := m.vault.CreateDir(rel)
	m.journal.Record(vault.Op{Desc: "create folder " + rel, Steps: createdSteps(dirs)})
	if err != nil {
		return err
	}
	m.reveal(rel)
	m.flash = "Created folder " + rel + "/"
	return nil
}

func (m *Model) rename(target, input string) error {
	if strings.Contains(input, "/") {
		return errors.New("a name can't contain /; use m to move")
	}
	if err := vault.CheckName(input); err != nil {
		return err
	}
	name := input
	if !m.vault.IsDir(target) && vault.IsNote(path.Base(target)) && !vault.IsNote(name) {
		name += ".md"
	}
	to := path.Join(parentOf(target), name)
	if to == target {
		return nil
	}
	if m.vault.Exists(to) {
		return fmt.Errorf("%s already exists", name)
	}
	m.relocate([][2]string{{target, to}}, "rename "+path.Base(target)+" to "+name, func(moved [][2]string, relinked int, err error) {
		m.marks = map[string]bool{}
		// A rename stays in place, so the cursor goes along with it.
		m.followMoves(moved, true)
		m.refresh()
		if err != nil {
			m.flash = "Rename failed: " + err.Error()
			return
		}
		m.flash = "Renamed to " + name + linksNote(relinked) + " · U undoes"
	})
	return nil
}

func (m *Model) startMove(srcs []string) {
	dirs, err := m.vault.Dirs()
	if err != nil {
		m.flash = err.Error()
		return
	}
	c := &chooser{title: "Move " + describe(srcs), prompt: "to", empty: "No folder matches", verb: "move"}
	for _, d := range dirs {
		if movesIntoItself(d, srcs) {
			continue
		}
		c.items = append(c.items, choice{label: m.folderLabel(d), do: func() { m.moveTo(d, srcs) }})
	}
	m.openChooser(c)
}

func movesIntoItself(dest string, srcs []string) bool {
	for _, s := range srcs {
		if dest == s || strings.HasPrefix(dest, s+"/") {
			return true
		}
	}
	return false
}

// moveTo moves srcs into dest as one undoable operation. Name clashes are
// checked first, so either everything moves or nothing does. The cursor
// stays behind on the next row, ready for the next item to tidy.
func (m *Model) moveTo(dest string, srcs []string) {
	var moves [][2]string
	claimed := map[string]string{} // destination → the item that wants it
	for _, s := range srcs {
		to := path.Join(dest, path.Base(s))
		if to == s {
			continue
		}
		if m.vault.Exists(to) {
			m.flash = fmt.Sprintf("Nothing moved: %s already has a %s", m.folderLabel(dest), path.Base(s))
			return
		}
		// Two marked items can share a name (a/Plan.md and b/Plan.md).
		// Moving the first and failing the second would leave the batch
		// half done, so the clash is caught before anything moves.
		if first, clash := claimed[to]; clash {
			m.flash = fmt.Sprintf("Nothing moved: %s and %s would both become %s", first, s, to)
			return
		}
		claimed[to] = s
		moves = append(moves, [2]string{s, to})
	}
	if len(moves) == 0 {
		m.flash = "Already in " + m.folderLabel(dest)
		return
	}
	m.relocate(moves, "move "+describe(froms(moves))+" to "+m.folderLabel(dest), func(moved [][2]string, relinked int, err error) {
		m.marks = map[string]bool{}
		m.followMoves(moved, false)
		m.refresh()
		if err != nil {
			m.flash = "Move stopped: " + err.Error()
			return
		}
		m.flash = "Moved " + describe(froms(moved)) + " to " + m.folderLabel(dest) + linksNote(relinked) + " · U undoes"
	})
}

func (m *Model) askDelete(ts []string) {
	option := obsidian.LoadSettings(m.vault.Root).TrashOption
	where := "to the trash"
	switch option {
	case "local":
		where = "to the vault's .trash folder"
	case "none":
		where = "permanently"
	}
	folders, notes := 0, 0
	for _, p := range ts {
		if m.vault.IsDir(p) {
			folders++
			notes += m.vault.CountNotes(p)
		}
	}
	var q string
	switch {
	case len(ts) == 1 && folders == 1:
		q = fmt.Sprintf("Delete folder %s and its %s %s?", describe(ts), plural(notes, "note"), where)
	case folders > 0:
		q = fmt.Sprintf("Delete %s %s, including %s with %s?", describe(ts), where, plural(folders, "folder"), plural(notes, "note"))
	default:
		q = fmt.Sprintf("Delete %s %s?", describe(ts), where)
	}
	m.confirm = &confirm{
		pill: " DELETE ", question: q, keys: "y/n", danger: true, cancel: "Nothing deleted",
		yes: func() { m.deletePaths(ts, option) },
	}
}

func (m *Model) deletePaths(ts []string, option string) {
	var steps []vault.Step
	var done []string
	var failed error
	for _, p := range ts {
		if option == "none" {
			if err := m.vault.RemoveAll(p); err != nil {
				failed = err
				continue
			}
			done = append(done, p)
			continue
		}
		t, err := m.vault.Trash(p, option)
		if err != nil {
			failed = err
			continue
		}
		steps = append(steps, vault.Step{Kind: vault.StepTrashed, Rel: p, Trash: t})
		done = append(done, p)
	}
	m.journal.Record(vault.Op{Desc: "delete " + describe(done), Steps: steps})
	m.marks = map[string]bool{}
	m.refresh()
	switch {
	case failed != nil:
		m.flash = "Couldn't delete everything: " + failed.Error()
	case option == "none":
		m.flash = "Deleted " + describe(done) + " permanently"
	default:
		m.flash = "Deleted " + describe(done) + " · U restores"
	}
}

func (m *Model) undoOp() {
	option := obsidian.LoadSettings(m.vault.Root).TrashOption
	op, ok, err := m.journal.Undo(m.vault, option)
	if !ok {
		m.flash = "Nothing to undo"
		return
	}
	var back [][2]string
	for _, s := range op.Steps {
		if s.Kind == vault.StepMoved && m.vault.Exists(s.From) {
			_ = m.snaps.Move(s.Rel, s.From)
			back = append(back, [2]string{s.Rel, s.From})
		}
	}
	m.followMoves(back, true)
	m.refresh()
	// Put the cursor on what came back.
	for _, s := range op.Steps {
		if s.Kind == vault.StepTrashed {
			m.files.reveal(s.Rel)
			break
		}
		if s.Kind == vault.StepMoved {
			m.files.reveal(s.From)
			break
		}
	}
	m.flash = "Undone: " + op.Desc
	if err != nil {
		m.flash = "Undo incomplete: " + err.Error()
	}
}

// openDaily opens today's daily note, creating it first from Obsidian's
// daily-note settings and template. A new note gets the previous daily
// note's unfinished todos, the way the Rollover Daily Todos plugin does it,
// unless Obsidian is open and its plugin will do that itself.
func (m *Model) openDaily() {
	s := obsidian.LoadSettings(m.vault.Root)
	now := m.opts.Now()
	rel := daily.Path(s.Daily, now)
	if m.vault.Exists(rel) {
		m.goTo(rel, "")
		m.flash = "Today's note: " + rel
		return
	}

	var notes []string
	content := ""
	if tp := daily.TemplatePath(s.Daily); tp != "" {
		if src, err := m.vault.Read(tp); err == nil {
			content = daily.Expand(src, s.Daily.Format, now, now)
		} else {
			notes = append(notes, "template "+tp+" not found")
		}
	}

	var prev, prevSrc string
	var rolled, remove []string
	switch {
	case !m.opts.RolloverTodos:
	case m.opts.ObsidianOpen():
		notes = append(notes, "Obsidian is open, so its Rollover plugin carries the todos over")
	default:
		prev = daily.Previous(m.vault.Exists, s.Daily, now)
		if src, err := m.vault.Read(prev); prev != "" && err == nil {
			prevSrc = src
			// The real Rollover plugin scans the whole previous note for
			// any unchecked checkbox, with no regard for section — its
			// templateHeading setting only says where rolled todos land
			// in the new note, not what counts as one. Left alone, that
			// would roll unfinished habits into tomorrow's Todo's list
			// (and delete them from yesterday's note under
			// deleteOnComplete) — the opposite of "habits reset daily."
			// The Habits block is excluded from the scan before it ever
			// reaches the rollover mirror.
			lines := daily.Lines(src)
			if start, end, ok := habit.BlockRange(lines); ok {
				lines = append(append([]string{}, lines[:start]...), lines[end:]...)
			}
			all := daily.UnfinishedTodos(lines, s.Rollover.RolloverChildren, s.Rollover.DoneStatusMarkers)
			rolled = all
			if s.Rollover.RemoveEmptyTodos {
				rolled = daily.DropEmpty(all)
			}
			if s.Rollover.DeleteOnComplete {
				remove = all
			}
		}
	}
	if len(rolled) > 0 {
		content = daily.AddTodos(content, rolled, s.Rollover.TemplateHeading)
	}

	dirs, err := m.vault.CreateFile(rel, content)
	steps := createdSteps(dirs)
	if err != nil {
		m.journal.Record(vault.Op{Desc: "create " + rel, Steps: steps})
		m.flash = "Couldn't create today's note: " + err.Error()
		return
	}
	steps = append(steps, vault.Step{Kind: vault.StepCreated, Rel: rel, Content: content})
	if len(remove) > 0 {
		// Snapshot yesterday's note first, as every write does, so u can
		// bring it back even after a restart.
		if err := m.snaps.Save(prev, prevSrc); err != nil {
			notes = append(notes, "left "+prev+" untidied: couldn't keep a snapshot first ("+err.Error()+")")
		} else if err := m.vault.Write(prev, daily.RemoveLines(prevSrc, remove)); err != nil {
			notes = append(notes, "couldn't tidy "+prev+": "+err.Error())
		} else {
			steps = append(steps, vault.Step{Kind: vault.StepModified, Rel: prev, Content: prevSrc})
		}
	}
	m.journal.Record(vault.Op{Desc: "create " + rel, Steps: steps})
	m.refresh()
	m.goTo(rel, "")

	msg := "Created " + rel
	if len(rolled) > 0 {
		msg += fmt.Sprintf(" · %s rolled over from %s", plural(len(rolled), "todo line"), path.Base(prev))
	}
	if len(notes) > 0 {
		msg += " · " + strings.Join(notes, "; ")
	}
	m.flash = msg
}

// reveal reloads the vault and puts the Files cursor on rel, opening the
// folders above it.
func (m *Model) reveal(rel string) {
	m.refresh()
	m.files.reveal(rel)
}

// refresh reloads the vault after Skrin changed it.
func (m *Model) refresh() {
	if err := m.reload(); err != nil {
		m.flash = err.Error()
	}
}

// followMoves carries the open note, the history, open folders and, with
// cursor, the Files cursor along with items that moved.
func (m *Model) followMoves(moves [][2]string, cursor bool) {
	m.notePath = movedPath(m.notePath, moves)
	if m.split != nil {
		m.split.path = movedPath(m.split.path, moves)
	}
	for _, h := range [][]place{m.back, m.fwd} {
		for i := range h {
			h[i].rel = movedPath(h[i].rel, moves)
		}
	}
	m.files.follow(moves, cursor)
}

func froms(moves [][2]string) []string {
	out := make([]string, len(moves))
	for i, mv := range moves {
		out[i] = mv[0]
	}
	return out
}

func createdSteps(dirs []string) []vault.Step {
	steps := make([]vault.Step, len(dirs))
	for i, d := range dirs {
		steps[i] = vault.Step{Kind: vault.StepCreated, Rel: d}
	}
	return steps
}

// joinUserPath joins a typed "name" or "sub/name" onto dir, checking each
// part.
func joinUserPath(dir, input string) (string, error) {
	parts := strings.Split(input, "/")
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if err := vault.CheckName(p); err != nil {
			return "", err
		}
		parts[i] = p
	}
	return path.Join(append([]string{dir}, parts...)...), nil
}

// untitledIn picks "Untitled", "Untitled 1", ... unused in folder dir.
func (m *Model) untitledIn(dir, ext string) string {
	for n := 0; ; n++ {
		name := "Untitled"
		if n > 0 {
			name = fmt.Sprintf("Untitled %d", n)
		}
		if !m.vault.Exists(path.Join(dir, name+ext)) {
			return name
		}
	}
}

func (m *Model) folderLabel(dir string) string {
	if dir == "" {
		return m.vault.Name() + "/"
	}
	return dir + "/"
}

func describe(paths []string) string {
	if len(paths) == 1 {
		return `"` + displayName(paths[0]) + `"`
	}
	return plural(len(paths), "item")
}

func displayName(rel string) string {
	b := path.Base(rel)
	if vault.IsNote(b) {
		return strings.TrimSuffix(b, path.Ext(b))
	}
	return b
}
