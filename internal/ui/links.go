package ui

import (
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"

	"github.com/lurioso/skrin/internal/index"
	"github.com/lurioso/skrin/internal/markdown"
	"github.com/lurioso/skrin/internal/obsidian"
	"github.com/lurioso/skrin/internal/vault"
)

// place is a spot in the history of followed links.
type place struct {
	rel string
	off int
}

// hint is a letter label on a visible link.
type hint struct {
	label string
	row   int // index into m.lines
	link  markdown.Link
}

type hintState struct {
	hints []hint
	typed string
}

const hintKeys = "asdfghjklqwertyuiopzxcvbnm"

// startHints labels the links in view with letters. With direct (Enter)
// and only one link in view, it follows that link straight away.
func (m *Model) startHints(direct bool) {
	if m.notePath == "" {
		m.flash = "Select a note to follow its links"
		return
	}
	vis := m.layout().bodyH - 2
	var hs []hint
	for i := m.noteOff; i < min(len(m.lines), m.noteOff+vis); i++ {
		for _, l := range m.lines[i].Links {
			hs = append(hs, hint{row: i, link: l})
		}
	}
	switch {
	case len(hs) == 0:
		m.flash = "No links in view"
		return
	case len(hs) == 1 && direct:
		m.follow(hs[0].link)
		return
	}
	for i, l := range hintLabels(len(hs)) {
		hs[i].label = l
	}
	m.hints = &hintState{hints: hs}
	m.focus = paneNote
}

// hintLabels gives n labels: single letters, or pairs when there are more
// links than letters.
func hintLabels(n int) []string {
	keys := []rune(hintKeys)
	var out []string
	if n <= len(keys) {
		for _, k := range keys[:n] {
			out = append(out, string(k))
		}
		return out
	}
	for _, a := range keys {
		for _, b := range keys {
			out = append(out, string(a)+string(b))
			if len(out) == n {
				return out
			}
		}
	}
	return out
}

func (m *Model) hintKey(k tea.KeyPressMsg) {
	h := m.hints
	s := k.String()
	if len([]rune(s)) != 1 {
		m.hints = nil
		return
	}
	h.typed += s
	partial := false
	for _, ht := range h.hints {
		switch {
		case ht.label == h.typed:
			m.hints = nil
			m.follow(ht.link)
			return
		case strings.HasPrefix(ht.label, h.typed):
			partial = true
		}
	}
	if !partial {
		m.hints = nil
		m.flash = "No link labelled " + h.typed
	}
}

// withLabel draws a hint label over a styled line at col.
func (m *Model) withLabel(line string, col int, label string) string {
	return ansi.Cut(line, 0, col) + "\x1b[m" + m.st.hint.Render(label) + ansi.TruncateLeft(line, col+ansi.StringWidth(label), "")
}

// follow opens what a link points to: a note (at its #heading), another
// file, or a web address. A link to a note that doesn't exist offers to
// create it.
func (m *Model) follow(l markdown.Link) {
	target := l.Target
	if !l.Wiki {
		if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
			m.openExternal(target)
			return
		}
		if dec, err := url.PathUnescape(target); err == nil {
			target = dec
		}
	}
	note, sub, _ := strings.Cut(target, "#")
	var rel string
	var ok bool
	if l.Wiki {
		rel, ok = m.idx.Resolve(note, m.notePath)
	} else {
		rel, ok = m.idx.ResolveLink(index.Link{Target: note, Markdown: true}, m.notePath)
	}
	switch {
	case !ok:
		m.offerCreate(note)
	case !vault.IsNote(rel):
		m.openExternal(m.vault.Abs(rel))
	default:
		m.goTo(rel, sub)
	}
}

// goTo opens note rel, at #sub if given.
func (m *Model) goTo(rel, sub string) {
	m.open(rel)
	if sub == "" {
		return
	}
	if line, ok := m.idx.Anchor(rel, sub); ok {
		m.jumpSrc = line
	} else {
		m.flash = fmt.Sprintf("No %q in %s", sub, displayName(rel))
	}
}

// revealInFiles moves the Files cursor to rel and switches focus there.
// It's for files that aren't notes (images) and so have nothing of their
// own to show in the note pane the way goTo's target does.
func (m *Model) revealInFiles(rel string) {
	m.focus = paneFiles
	m.files.reveal(rel)
}

// goToLine opens note rel scrolled to a source line.
func (m *Model) goToLine(rel string, line int) {
	m.open(rel)
	m.jumpSrc = line
}

// open shows note rel on the right and moves over to it, remembering the
// note it replaces so Backspace can go back.
func (m *Model) open(rel string) {
	m.pushHistory()
	m.showNote(rel)
	m.show()
}

// show moves over to the open note and puts the Files cursor on it,
// opening its folders on the way: the cursor and the open note go together.
func (m *Model) show() {
	m.focus = paneNote
	m.files.reveal(m.notePath)
}

func (m *Model) openExternal(target string) {
	if err := m.opts.Open(target); err != nil {
		m.flash = "Couldn't open " + target + ": " + err.Error()
		return
	}
	m.flash = "Opened " + target
}

func (m *Model) pushHistory() {
	if m.notePath == "" {
		return
	}
	m.back = append(m.back, place{m.notePath, m.noteOff})
	if len(m.back) > 100 {
		m.back = m.back[1:]
	}
	m.fwd = nil
}

// goBack steps back (or forward) through followed links. It reports false
// when there is nowhere to go.
func (m *Model) goBack(forward bool) bool {
	from, to := &m.back, &m.fwd
	if forward {
		from, to = &m.fwd, &m.back
	}
	if len(*from) == 0 {
		return false
	}
	p := (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	if m.notePath != "" {
		*to = append(*to, place{m.notePath, m.noteOff})
	}
	if !m.vault.Exists(p.rel) {
		m.flash = p.rel + " is gone"
		return true
	}
	m.showNote(p.rel)
	m.noteOff = p.off
	m.show()
	return true
}

// offerCreate asks to create the note a dangling link points to, where
// Obsidian would put it.
func (m *Model) offerCreate(target string) {
	rel := strings.Trim(target, "/")
	if rel == "" {
		return
	}
	if !vault.IsNote(rel) {
		rel += ".md"
	}
	if !strings.Contains(rel, "/") {
		s := obsidian.LoadSettings(m.vault.Root)
		switch s.NewFileLocation {
		case "current":
			rel = path.Join(parentOf(m.notePath), rel)
		case "folder":
			rel = path.Join(s.NewFileFolderPath, rel)
		}
	}
	for _, part := range strings.Split(rel, "/") {
		if err := vault.CheckName(part); err != nil {
			m.flash = fmt.Sprintf("Can't create %q: %v", target, err)
			return
		}
	}
	m.confirm = &confirm{
		pill: " NEW NOTE ", question: fmt.Sprintf("%q doesn't exist yet. Create %s?", target, rel),
		keys: "y/n", cancel: "Nothing created",
		yes: func() {
			if err := m.createNoteAt(rel); err != nil {
				m.flash = err.Error()
			}
		},
	}
}

func (m *Model) showBacklinks() {
	rel, ok := m.subject()
	if !ok {
		m.flash = "Select a note to see what links here"
		return
	}
	bl := m.idx.Backlinks(rel)
	if len(bl) == 0 {
		m.flash = "No notes link to " + describe([]string{rel})
		return
	}
	c := &chooser{title: fmt.Sprintf("Links to %s (%d)", displayName(rel), len(bl)), prompt: "filter", empty: "No match", verb: "open"}
	for _, b := range bl {
		c.items = append(c.items, choice{
			label: displayName(b.Source), detail: b.Link.Context,
			do: func() { m.goToLine(b.Source, b.Link.Line) },
		})
	}
	m.openChooser(c)
}

func (m *Model) showOutline() {
	rel, ok := m.subjectOpen()
	if !ok {
		m.flash = "Select a note to see its outline"
		return
	}
	hs := m.idx.Headings(rel)
	if len(hs) == 0 {
		m.flash = "No headings in " + describe([]string{rel})
		return
	}
	c := &chooser{title: "Outline of " + displayName(rel), prompt: "filter", empty: "No match", verb: "jump"}
	for _, h := range hs {
		c.items = append(c.items, choice{
			label: strings.Repeat("  ", h.Level-1) + h.Text,
			do:    func() { m.focus = paneNote; m.jumpSrc = h.Line },
		})
	}
	m.openChooser(c)
}

// headingJump scrolls to the next (dir 1) or previous (dir -1) heading.
func (m *Model) headingJump(dir int) {
	if m.notePath == "" {
		return
	}
	for i := m.noteOff + dir; i >= 0 && i < len(m.lines); i += dir {
		if m.lines[i].Heading > 0 {
			m.noteOff = i
			m.focus = paneNote
			return
		}
	}
	if dir > 0 {
		m.flash = "No more headings below"
	} else {
		m.flash = "No headings above"
	}
}

// --- [[ completion in the editor ---------------------------------------

type suggestion struct {
	label, detail, insert string
}

type completion struct {
	items []suggestion
	cur   int
}

const maxSuggestions = 8

// updateCompletion opens, refreshes or closes the [[ popup after an editor
// key.
func (m *Model) updateCompletion() {
	if m.editor == nil {
		m.complete, m.noComplete = nil, false
		return
	}
	q, ok := m.editor.LinkQuery()
	if !ok {
		m.complete, m.noComplete = nil, false
		return
	}
	if m.noComplete {
		return
	}
	m.complete = nil
	if items := m.suggest(q); len(items) > 0 {
		m.complete = &completion{items: items}
	}
}

// suggest lists link completions for what follows "[[": notes and their
// aliases, or after "Note#" that note's headings.
func (m *Model) suggest(q string) []suggestion {
	from := m.edit.rel
	if note, sub, ok := strings.Cut(q, "#"); ok {
		rel, found := m.idx.Resolve(note, from)
		if !found {
			return nil
		}
		var hs []suggestion
		for _, h := range m.idx.Headings(rel) {
			hs = append(hs, suggestion{label: strings.Repeat("#", h.Level) + " " + h.Text, insert: note + "#" + h.Text})
		}
		return filterSuggestions(hs, sub)
	}
	var all []suggestion
	for _, rel := range m.idx.Notes() {
		if rel == from {
			continue
		}
		text := m.idx.LinkText(rel, from, m.edit.linkFormat)
		name := displayName(rel)
		all = append(all, suggestion{label: name, detail: parentOf(rel), insert: text})
		for _, a := range m.idx.Aliases(rel) {
			all = append(all, suggestion{label: a, detail: "alias of " + name, insert: text + "|" + a})
		}
	}
	for _, rel := range m.idx.Images() {
		text := m.idx.LinkText(rel, from, m.edit.linkFormat)
		all = append(all, suggestion{label: displayName(rel), detail: parentOf(rel), insert: text})
	}
	return filterSuggestions(all, q)
}

func filterSuggestions(items []suggestion, q string) []suggestion {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return items[:min(len(items), maxSuggestions)]
	}
	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = strings.ToLower(it.label)
	}
	var out []suggestion
	for _, mt := range fuzzy.Find(q, labels) {
		out = append(out, items[mt.Index])
		if len(out) == maxSuggestions {
			break
		}
	}
	return out
}

// completionKey handles the popup's own keys; the rest go to the editor.
func (m *Model) completionKey(k tea.KeyPressMsg) bool {
	c := m.complete
	switch m.actionIn(inComplete, k.String()) {
	case actUp:
		c.cur = max(c.cur-1, 0)
	case actDown:
		c.cur = min(c.cur+1, len(c.items)-1)
	case actPick:
		m.editor.CompleteLink(c.items[c.cur].insert)
		m.complete = nil
	case actCancel:
		m.complete, m.noComplete = nil, true
	default:
		return false
	}
	return true
}

// completionBox renders the popup and where it goes: under the cursor, or
// above it when there's no room below.
func (m *Model) completionBox() ([]string, int, int) {
	c := m.complete
	w := min(56, m.width-2)
	var body []string
	for i, it := range c.items {
		row := fit(" "+it.label+"  "+m.st.muted.Render(it.detail), w-2)
		if i == c.cur {
			row = m.st.selFocus.Render(fit(" "+it.label+"  "+it.detail, w-2))
		}
		body = append(body, row)
	}
	box := m.box("", body, w, len(body)+2, true)
	l := m.layout()
	r, col := m.editor.CursorPos()
	// Where the editor's text starts on screen.
	ox, oy := l.filesW+2, headerHeight+1
	if m.zen {
		ox, oy = (m.width-l.noteTextW())/2, 1
	}
	x := clamp(ox+col, 0, max(m.width-w, 0))
	y := oy + r + 1
	if y+len(box) > m.height-statusHeight {
		y = max(oy+r-len(box), 0)
	}
	return box, x, y
}

// --- moving with links ---------------------------------------------------

// relink is a link that has to change because its target is moving.
type relink struct {
	source, target string // both as they are before the move
	link           index.Link
}

// referrers lists the links that the moves would break.
func (m *Model) referrers(moves [][2]string) []relink {
	var out []relink
	for _, mv := range moves {
		for _, b := range m.idx.Referrers(mv[0], m.vault.IsDir(mv[0])) {
			r := relink{source: b.Source, target: b.Target, link: b.Link}
			if m.needsRelink(r, moves) {
				out = append(out, r)
			}
		}
	}
	return out
}

// needsRelink: a plain [[Name]] survives a move as long as its name stays
// the same and unique; paths and markdown links always change.
func (m *Model) needsRelink(r relink, moves [][2]string) bool {
	switch {
	case r.link.Markdown || strings.Contains(r.link.Target, "/"):
		return true
	case r.link.Target == "": // [[#Heading]] inside the same note
		return false
	case path.Base(movedPath(r.target, moves)) != path.Base(r.target):
		return true
	}
	return m.idx.NameCount(path.Base(r.target)) > 1
}

func movedPath(p string, moves [][2]string) string {
	for _, mv := range moves {
		if p == mv[0] {
			return mv[1]
		}
		if strings.HasPrefix(p, mv[0]+"/") {
			return mv[1] + p[len(mv[0]):]
		}
	}
	return p
}

// relocate moves or renames files and folders as one undoable operation.
// When links elsewhere point at them it first asks whether to update those
// links, unless Obsidian is set to always update them.
func (m *Model) relocate(moves [][2]string, desc string, done func(moved [][2]string, relinked int, err error)) {
	refs := m.referrers(moves)
	run := func(update bool) { done(m.moveNow(moves, refs, desc, update)) }
	switch {
	case len(refs) == 0:
		run(false)
	case obsidian.LoadSettings(m.vault.Root).AlwaysUpdateLinks:
		run(true)
	default:
		notes := map[string]bool{}
		for _, r := range refs {
			notes[r.source] = true
		}
		m.confirm = &confirm{
			pill:     " LINKS ",
			question: fmt.Sprintf("Update %s in %s so they keep working?", plural(len(refs), "link"), plural(len(notes), "note")),
			keys:     "y update · n leave them · esc cancel",
			cancel:   "Nothing moved",
			yes:      func() { run(true) },
			no:       func() { run(false) },
		}
	}
}

// moveNow does the moves as one journal entry and, with update, rewrites
// the links in refs to follow them. It reports what moved.
func (m *Model) moveNow(moves [][2]string, refs []relink, desc string, update bool) ([][2]string, int, error) {
	var steps []vault.Step
	var moved [][2]string
	var failed error
	for _, mv := range moves {
		dirs, err := m.vault.Move(mv[0], mv[1])
		steps = append(steps, createdSteps(dirs)...)
		if err != nil {
			failed = err
			break
		}
		steps = append(steps, vault.Step{Kind: vault.StepMoved, Rel: mv[1], From: mv[0]})
		moved = append(moved, mv)
		_ = m.snaps.Move(mv[0], mv[1]) // snapshots only help u; losing them loses no note text
	}
	relinked := 0
	if update && failed == nil {
		var s []vault.Step
		s, relinked, failed = m.relinkAfter(refs, moves)
		steps = append(steps, s...)
	}
	if s, ok := m.orderFollows(moved); ok {
		steps = append(steps, s)
	}
	m.journal.Record(vault.Op{Desc: desc, Steps: steps})
	return moved, relinked, failed
}

// relinkAfter rewrites the links in refs once the moves are done. Each
// changed note is snapshotted and becomes a journal step, so U undoes the
// link edits together with the move.
func (m *Model) relinkAfter(refs []relink, moves [][2]string) ([]vault.Step, int, error) {
	if err := m.idx.Update(m.vault); err != nil {
		return nil, 0, err
	}
	format := obsidian.LoadSettings(m.vault.Root).NewLinkFormat
	bySource := map[string][]index.Edit{}
	for _, r := range refs {
		src, tgt := movedPath(r.source, moves), movedPath(r.target, moves)
		if got, ok := m.idx.ResolveLink(r.link, src); ok && got == tgt {
			continue
		}
		text := m.idx.LinkText(tgt, src, format)
		if r.link.Markdown {
			text = markdownPath(r, src, tgt)
		}
		bySource[src] = append(bySource[src], index.Edit{Link: r.link, Target: text})
	}
	sources := make([]string, 0, len(bySource))
	for s := range bySource {
		sources = append(sources, s)
	}
	sort.Strings(sources)
	var steps []vault.Step
	n := 0
	for _, src := range sources {
		old, err := m.vault.Read(src)
		if err != nil {
			return steps, n, err
		}
		updated := index.Apply(old, bySource[src])
		if updated == old {
			continue
		}
		_ = m.snaps.Save(src, old)
		if err := m.vault.Write(src, updated); err != nil {
			return steps, n, err
		}
		steps = append(steps, vault.Step{Kind: vault.StepModified, Rel: src, Content: old})
		n += len(bySource[src])
	}
	return steps, n, nil
}

// markdownPath is a markdown link's new path: relative to the linking note
// if it was relative, otherwise from the vault root.
func markdownPath(r relink, src, tgt string) string {
	t := r.link.Target
	if strings.HasPrefix(t, "/") {
		return "/" + tgt
	}
	if strings.EqualFold(path.Clean(path.Join(path.Dir(r.source), t)), r.target) {
		return index.RelPath(path.Dir(src), tgt)
	}
	return tgt
}

func linksNote(n int) string {
	if n == 0 {
		return ""
	}
	return " · " + plural(n, "link") + " updated"
}
