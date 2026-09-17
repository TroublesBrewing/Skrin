package ui

import (
	"path"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/sahilm/fuzzy"

	"github.com/lurioso/skrin/internal/obsidian"
	"github.com/lurioso/skrin/internal/vault"
)

// quickNoteArea is which of the overlay's two stops has focus.
type quickNoteArea int

const (
	quickNoteText quickNoteArea = iota
	quickNoteFolder
)

// quickNote is the i overlay: a small text capture that names itself from
// its first line, saved into a folder picked from the vault's own list —
// never typed as a free path, so a nonexistent folder can't be created by
// accident. All of it lives here and nowhere on disk until Enter saves it.
type quickNote struct {
	text   textArea
	folder lineInput

	def     string   // the resolved default folder; stands until folder is touched
	folders []string // every real folder in the vault, root ("") included
	matches []int    // indexes into folders currently matching folder's filter
	cur     int      // the highlighted match, when the folder row is focused

	area quickNoteArea
	err  string
}

// openQuickNote opens a blank capture overlay. The folder row starts on
// the vault's own declared home for new notes (Obsidian's "Default
// location for new notes", already honoured for link-created notes in
// offerCreate) — the vault root when nothing is declared, not a Skrin
// setting.
func (m *Model) openQuickNote() {
	dirs, err := m.vault.Dirs()
	if err != nil {
		m.flash = err.Error()
		return
	}
	c := &quickNote{def: m.quickNoteDefaultFolder(), folders: dirs}
	m.quickNote = c
}

// quickNoteDefaultFolder resolves where a quick note lands absent any
// picking: a declared folder wins only while it still exists (a deleted
// declared folder is as good as unset); unset, "current" (no note context
// applies to a capture from anywhere) or gone all fall back to the vault
// root — a stable, language-neutral default that needs no Skrin setting.
func (m *Model) quickNoteDefaultFolder() string {
	s := obsidian.LoadSettings(m.vault.Root)
	if s.NewFileLocation == "folder" && s.NewFileFolderPath != "" && m.vault.IsDir(s.NewFileFolderPath) {
		return s.NewFileFolderPath
	}
	return ""
}

// folderChosen is the folder the note will save into: the filtered pick if
// one's been made, otherwise the untouched default.
func (c *quickNote) folderChosen() string {
	if strings.TrimSpace(c.folder.value()) == "" {
		return c.def
	}
	return c.folder.value()
}

// filterFolders refreshes matches from the folder field's current text.
func (c *quickNote) filterFolders() {
	c.matches, c.cur = c.matches[:0], 0
	q := strings.TrimSpace(c.folder.value())
	if q == "" {
		for i := range c.folders {
			c.matches = append(c.matches, i)
		}
		return
	}
	labels := make([]string, len(c.folders))
	for i, d := range c.folders {
		labels[i] = d
		if d == "" {
			labels[i] = "/" // the root still needs to be filterable text
		}
	}
	for _, mt := range fuzzy.Find(q, labels) {
		c.matches = append(c.matches, mt.Index)
	}
}

func (c *quickNote) next() {
	if c.area == quickNoteText {
		c.area = quickNoteFolder
		c.filterFolders()
	} else {
		c.area = quickNoteText
	}
}
func (c *quickNote) prev() { c.next() } // only two stops: either direction toggles

// quickNoteName derives a note's file name from the overlay's first typed
// line: trimmed, leading #s stripped (heading syntax isn't content), cut
// at 50 characters hard (a name, not a novella), then the same character
// check every create uses.
func quickNoteName(firstLine string) (string, error) {
	name := strings.TrimSpace(firstLine)
	name = strings.TrimSpace(strings.TrimLeft(name, "#"))
	if r := []rune(name); len(r) > 50 {
		name = strings.TrimSpace(string(r[:50]))
	}
	if name == "" {
		return "", nil
	}
	if err := vault.CheckName(name); err != nil {
		return "", err
	}
	return name, nil
}

func (m *Model) quickNoteKey(k tea.KeyPressMsg) {
	c := m.quickNote
	switch m.actionIn(inQuickNote, k.String()) {
	case actCancel:
		m.quickNote = nil
		m.flash = "Nothing saved"
		return
	case actNewLine:
		if c.area == quickNoteText {
			c.text.insert("\n")
		}
		return
	case actNextField:
		c.err = ""
		c.next()
		return
	case actPrevField:
		c.err = ""
		c.prev()
		return
	case actPick:
		c.err = ""
		if c.area == quickNoteFolder {
			c.pickFolder()
		} else {
			m.saveQuickNote()
		}
		return
	case actUp:
		if c.area == quickNoteFolder {
			c.cur = max(c.cur-1, 0)
		} else {
			c.text.moveVert(-1)
		}
		return
	case actDown:
		if c.area == quickNoteFolder {
			c.cur = min(c.cur+1, max(len(c.matches)-1, 0))
		} else {
			c.text.moveVert(1)
		}
		return
	}
	c.err = ""
	if c.area == quickNoteFolder {
		if c.folder.handle(k) {
			c.filterFolders()
		}
		return
	}
	c.text.handle(k)
}

// pickFolder takes the highlighted match as the folder and returns focus
// to the text. A filter with no match refuses instead, per the spec's
// "typed folder that isn't one" case.
func (c *quickNote) pickFolder() {
	if strings.TrimSpace(c.folder.value()) == "" {
		c.area = quickNoteText // untouched: the default stands
		return
	}
	if len(c.matches) == 0 {
		c.err = "No folder " + c.folder.value() + "/ — pick one from the list"
		return
	}
	c.folder.set(c.folders[c.matches[c.cur]])
	c.area = quickNoteText
}

// saveQuickNote derives the name from the text's first line, checks the
// three refusals and — if all's well — writes the note as one undoable
// create. It never opens the saved note: the flash names it, U undoes it,
// g finds it later.
func (m *Model) saveQuickNote() {
	c := m.quickNote
	lines := strings.SplitN(c.text.value(), "\n", 2)
	firstLine := lines[0]
	body := c.text.value()

	name, err := quickNoteName(firstLine)
	if err != nil {
		c.err = err.Error()
		return
	}
	if name == "" {
		c.err = "The first line becomes the note's name — type one first"
		return
	}
	folder := c.folderChosen()
	if folder != "" && !m.vault.IsDir(folder) {
		c.err = "No folder " + folder + "/ — pick one from the list"
		return
	}
	rel := path.Join(folder, name+".md")
	if m.vault.Exists(rel) {
		c.err = rel + " already exists — change the first line, or Tab and pick another folder"
		return
	}

	dirs, werr := m.vault.CreateFile(rel, body)
	steps := createdSteps(dirs)
	if werr == nil {
		steps = append(steps, vault.Step{Kind: vault.StepCreated, Rel: rel})
	}
	m.journal.Record(vault.Op{Desc: "create " + rel, Steps: steps})
	if werr != nil {
		c.err = werr.Error()
		return
	}
	m.quickNote = nil
	m.flash = "Saved " + rel + " · U undoes"
}

// quickNoteBox renders the overlay: the text field, then the folder row,
// with the filtered matches shown beneath it while it has focus.
func (m *Model) quickNoteBox() []string {
	c := m.quickNote
	if c == nil {
		return nil
	}
	w := min(max(m.width-6, 50), 80)
	inner := w - 4

	var body []string
	lines, curRow, curCol := c.text.wrapped(inner - 2)
	rows := clamp(len(lines), 3, 8)
	// Once the text outgrows rows (the box has stopped growing), scroll
	// the window to keep the cursor's line visible instead of always
	// showing the first rows lines — otherwise typing past the cap
	// leaves the cursor off-screen with no way to see it, which is what
	// the report described as the window "stopping" with no scroll.
	off := 0
	if len(lines) > rows {
		off = clamp(curRow-rows+1, 0, len(lines)-rows)
	}
	for i := 0; i < rows; i++ {
		li := off + i
		if li >= len(lines) {
			body = append(body, "")
			continue
		}
		l := lines[li]
		if c.area == quickNoteText && li == curRow {
			r := []rune(l)
			at := " "
			if curCol < len(r) {
				at = string(r[curCol])
			}
			pre, post := "", ""
			if curCol <= len(r) {
				pre = string(r[:curCol])
			}
			if curCol < len(r) {
				post = string(r[curCol+1:])
			}
			body = append(body, "  "+m.st.text.Render(pre)+m.st.cursor.Render(at)+m.st.text.Render(post))
			continue
		}
		body = append(body, "  "+m.st.text.Render(l))
	}
	body = append(body, strings.Repeat("─", inner))

	folderFocused := c.area == quickNoteFolder
	label := m.st.muted.Render("Folder: ")
	valueView := m.st.muted.Render(m.folderLabel(c.def))
	if strings.TrimSpace(c.folder.value()) != "" || folderFocused {
		cur := m.st.text
		if folderFocused {
			label = m.st.titleFocus.Render("Folder: ")
			cur = m.st.cursor
		}
		valueView = c.folder.view(m.st.text, cur)
	}
	body = append(body, "  "+label+valueView)

	if folderFocused {
		shown := clamp(len(c.matches), 0, 6)
		off := max(0, c.cur-shown+1)
		for i := off; i < min(len(c.matches), off+shown); i++ {
			d := m.folderLabel(c.folders[c.matches[i]])
			if i == c.cur {
				body = append(body, "    "+m.st.selFocus.Render(fit(d, inner-4)))
				continue
			}
			body = append(body, "    "+m.st.text.Render(d))
		}
		if len(c.matches) == 0 {
			body = append(body, "    "+m.st.muted.Render("No match"))
		}
	}

	if c.err != "" {
		body = append(body, m.st.errText.Render("  "+c.err))
	}
	body = append(body, "", " "+m.st.muted.Render("enter save · tab folder · shift+enter newline · esc cancel"))

	h := clamp(len(body)+2, 8, max(m.height-4, 8))
	return m.box(" QUICK NOTE ", body, w, h, true)
}
