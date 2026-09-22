package ui

import (
	"path"
	"sort"
	"strings"

	"github.com/lurioso/skrin/internal/config"
	"github.com/lurioso/skrin/internal/daily"
	"github.com/lurioso/skrin/internal/frontmatter"
	"github.com/lurioso/skrin/internal/obsidian"
	"github.com/lurioso/skrin/internal/vault"
)

// templatesFolder is where Insert template looks, and whose it is: the
// folder chosen in Settings, else Obsidian's own Templates folder, else
// none. A chosen or declared folder that no longer exists counts as none.
func (m *Model) templatesFolder() (folder, whose string) {
	if f := strings.Trim(m.opts.Config.Templates.Folder, "/"); f != "" && m.vault.IsDir(f) {
		return f, "chosen in Settings"
	}
	if f := obsidian.LoadSettings(m.vault.Root).Templates.Folder; f != "" && m.vault.IsDir(f) {
		return f, "Obsidian's"
	}
	return "", ""
}

// startTemplate is Insert template, from the palette: in the editor, a list
// of the templates to put in at the cursor; in the reading view, the note
// opens in the editor first, at the line you were reading.
func (m *Model) startTemplate() {
	folder, _ := m.templatesFolder()
	if folder == "" {
		m.flash = "No templates folder yet: choose one in Settings (? then Tab), or Obsidian's Templates plugin"
		return
	}
	files, err := m.vault.Files()
	if err != nil {
		m.flash = err.Error()
		return
	}
	var tmpls []string
	for _, f := range files {
		if strings.HasPrefix(f, folder+"/") && vault.IsNote(f) {
			tmpls = append(tmpls, f)
		}
	}
	if len(tmpls) == 0 {
		m.flash = "No templates in " + folder + "/ yet: a note there is a template"
		return
	}
	sort.Strings(tmpls)
	if m.editor == nil {
		if _, ok := m.subject(); !ok {
			m.flash = "Select a note to put a template in"
			return
		}
		m.startEdit()
		if m.editor == nil {
			return // startEdit said why
		}
	}
	var items []choice
	for _, rel := range tmpls {
		name := strings.TrimSuffix(strings.TrimPrefix(rel, folder+"/"), path.Ext(rel))
		items = append(items, choice{label: name, do: func() { m.insertTemplate(rel, name) }})
	}
	m.openChooser(&chooser{
		title:  "Insert template",
		prompt: "Template",
		empty:  "No template by that name",
		verb:   "insert",
		items:  items,
	})
}

// insertTemplate puts template rel into the note being edited.
func (m *Model) insertTemplate(rel, name string) {
	if m.editor == nil {
		return
	}
	src, err := m.vault.Read(rel)
	if err != nil {
		m.flash = "Can't read " + rel + ": " + err.Error()
		return
	}
	s := obsidian.LoadSettings(m.vault.Root).Templates
	src = daily.ExpandTemplate(src, displayName(m.edit.rel), s.DateFormat, s.TimeFormat, m.opts.Now())
	row, col := m.editor.Cursor()
	text, row, col := applyTemplate(m.editor.Text(), row, col, src)
	m.editor.Rewrite(text, row, col)
	m.flash = "Inserted " + name + " · ctrl+z takes it out"
}

// applyTemplate puts tmpl into note at row, col, the way Obsidian's
// Templates plugin does: the body at the cursor, and the template's
// properties merged into the note's own — the note's frontmatter gains the
// keys it doesn't have yet (its own values win), or the template's
// frontmatter becomes the note's if it has none. It returns the new text
// and the cursor, at the end of what went in.
func applyTemplate(note string, row, col int, tmpl string) (string, int, int) {
	tmplFM, body := splitFrontmatter(strings.Split(strings.ReplaceAll(tmpl, "\r\n", "\n"), "\n"))
	lines := strings.Split(note, "\n")
	row = clamp(row, 0, len(lines)-1)
	line := []rune(lines[row])
	col = clamp(col, 0, len(line))

	// The body, at the cursor.
	bodyText := strings.Join(body, "\n")
	parts := strings.Split(bodyText, "\n")
	head, tail := string(line[:col]), string(line[col:])
	ins := make([]string, len(parts))
	copy(ins, parts)
	ins[0] = head + ins[0]
	endRow := row + len(parts) - 1
	endCol := len([]rune(ins[len(ins)-1]))
	ins[len(ins)-1] += tail
	lines = append(lines[:row], append(ins, lines[row+1:]...)...)

	if tmplFM == nil {
		return strings.Join(lines, "\n"), endRow, endCol
	}
	// The properties, at the top.
	noteFM, rest := splitFrontmatter(lines)
	var merged []string
	if noteFM == nil {
		merged = append(append([]string{"---"}, tmplFM...), "---")
	} else {
		have := map[string]bool{}
		for _, c := range fmChunks(noteFM) {
			have[c.key] = true
		}
		add := noteFM
		for _, c := range fmChunks(tmplFM) {
			if !have[c.key] {
				add = append(add, c.lines...)
			}
		}
		merged = append(append([]string{"---"}, add...), "---")
	}
	oldTop := len(lines) - len(rest)
	out := append(merged, rest...)
	return strings.Join(out, "\n"), endRow + len(merged) - oldTop, endCol
}

// splitFrontmatter separates a leading --- block's inside lines from the
// rest; nil when there is none.
func splitFrontmatter(lines []string) (fm, rest []string) {
	end := frontmatter.End(lines)
	if end == 0 {
		return nil, lines
	}
	return append([]string{}, lines[1:end]...), lines[end+1:]
}

// fmChunk is one top-level property: its key line and the indented or
// listed lines under it.
type fmChunk struct {
	key   string
	lines []string
}

func fmChunks(fm []string) []fmChunk {
	var out []fmChunk
	for _, l := range fm {
		top := l != "" && l[0] != ' ' && l[0] != '\t' && !strings.HasPrefix(l, "- ") && strings.Contains(l, ":")
		if top || len(out) == 0 {
			key, _, _ := strings.Cut(l, ":")
			out = append(out, fmChunk{key: strings.TrimSpace(key)})
		}
		out[len(out)-1].lines = append(out[len(out)-1].lines, l)
	}
	return out
}

// pickTemplatesFolder is the Settings tab's Templates folder row: a list
// of the vault's folders, with Obsidian's own first when it has one. The
// manual comes back afterwards, on the same row, either way.
func (m *Model) pickTemplatesFolder() {
	dirs, err := m.vault.Dirs()
	if err != nil {
		m.flash = err.Error()
		return
	}
	setCur := 0
	if m.manual != nil {
		setCur = m.manual.setCur
	}
	back := func() {
		m.openManual()
		m.manualGoTab(manualTabSettings)
		m.manual.setCur = setCur
	}
	save := func(folder, said string) {
		m.opts.Config.Templates.Folder = folder
		back()
		if err := config.Save(m.opts.Config); err != nil {
			m.flash = "couldn't save settings: " + err.Error()
			return
		}
		m.flash = said
	}
	var items []choice
	if f := obsidian.LoadSettings(m.vault.Root).Templates.Folder; f != "" {
		items = append(items, choice{label: "Obsidian's: " + f, detail: "follows Obsidian's Templates plugin", do: func() {
			save("", "Templates folder: Obsidian's, "+f)
		}})
	}
	for _, d := range dirs {
		if d == "" {
			continue
		}
		items = append(items, choice{label: d, do: func() { save(d, "Templates folder: "+d) }})
	}
	m.manual = nil
	m.openChooser(&chooser{
		title:  "Templates folder",
		prompt: "Folder",
		empty:  "No folder by that name",
		verb:   "choose",
		items:  items,
		cancel: back,
	})
}

// folderTemplatesRule returns the pair for folder, and its place in the
// list, so the pickers know whether they are adding or replacing.
func (m *Model) folderTemplatesRule(folder string) (config.TemplateRule, int) {
	for i, r := range m.opts.FolderTemplates {
		if r.Folder == folder {
			return r, i
		}
	}
	return config.TemplateRule{}, -1
}

// setFolderTemplates saves the list to config.toml and mirrors it into
// Options, the single source of truth the rest of Skrin reads.
func (m *Model) setFolderTemplates(rules []config.TemplateRule) {
	m.opts.FolderTemplates = rules
	m.opts.Config.Templates.Rules = rules
	if err := config.Save(m.opts.Config); err != nil {
		m.flash = "couldn't save settings: " + err.Error()
	}
}

// pickFolderTemplates opens the list of folder → template pairs, with
// "+ Add a folder…" at the top. Enter on a pair changes its template; d
// removes it; esc steps back to Settings, on the same row.
func (m *Model) pickFolderTemplates() {
	setCur := 0
	if m.manual != nil {
		setCur = m.manual.setCur
	}
	back := func() {
		m.openManual()
		m.manualGoTab(manualTabSettings)
		m.manual.setCur = setCur
	}
	save := func(rules []config.TemplateRule) {
		m.setFolderTemplates(rules)
		back()
	}

	items := []choice{{
		label: "+ Add a folder…",
		do:    func() { m.addFolderTemplate(save) },
	}}
	for _, r := range m.opts.FolderTemplates {
		r := r
		name := displayName(r.Template)
		items = append(items, choice{
			label:  m.folderLabel(r.Folder) + name,
			detail: r.Template,
			do:     func() { m.changeFolderTemplate(r.Folder, save) },
		})
	}
	m.manual = nil
	m.openChooser(&chooser{
		title:  "Folder templates",
		prompt: "Folder",
		empty:  "No folder by that name",
		verb:   "pick its template",
		remove: func(it choice) {
			m.removeFolderTemplate(it, save)
		},
		items:  items,
		cancel: back,
	})
}

// removeFolderTemplate drops the pair the highlighted row stands for. The
// row's label carries the folder (folderLabel(folder) + template name), so
// the pair is found by matching the label against the current rules.
func (m *Model) removeFolderTemplate(it choice, save func([]config.TemplateRule)) {
	var kept []config.TemplateRule
	for _, r := range m.opts.FolderTemplates {
		if it.label != m.folderLabel(r.Folder)+displayName(r.Template) {
			kept = append(kept, r)
		}
	}
	save(kept)
}

// addFolderTemplate walks add-a-folder: pick a folder, then its template,
// then save. A folder that already has a pair is offered first with its
// current template shown, since the new pairing replaces the old one.
func (m *Model) addFolderTemplate(save func([]config.TemplateRule)) {
	dirs, err := m.vault.Dirs()
	if err != nil {
		m.flash = err.Error()
		return
	}
	var items []choice
	for _, d := range dirs {
		if d == "" {
			continue // the root has no name to pair; a root note is created empty
		}
		detail := ""
		if r, _ := m.folderTemplatesRule(d); r.Folder != "" {
			detail = "now " + r.Template
		}
		d := d
		items = append(items, choice{
			label:  m.folderLabel(d),
			detail: detail,
			do:     func() { m.pickFolderTemplateFor(d, save) },
		})
	}
	m.openChooser(&chooser{
		title:  "Template for new notes in…",
		prompt: "Folder",
		empty:  "No folder by that name",
		verb:   "pick its template",
		items:  items,
		cancel: func() { m.pickFolderTemplates() },
	})
}

// changeFolderTemplate is Enter on an existing pair: pick a new template
// for the folder it names.
func (m *Model) changeFolderTemplate(folder string, save func([]config.TemplateRule)) {
	m.pickFolderTemplateFor(folder, save)
}

// pickFolderTemplateFor lists the templates and pairs the chosen one with
// folder, replacing any pair already there. The new pairing is saved at
// once, so nothing the user built is lost on a cancel.
func (m *Model) pickFolderTemplateFor(folder string, save func([]config.TemplateRule)) {
	tfolder, _ := m.templatesFolder()
	if tfolder == "" {
		m.flash = "No templates folder yet: choose one in Settings"
		return
	}
	files, err := m.vault.Files()
	if err != nil {
		m.flash = err.Error()
		return
	}
	var items []choice
	for _, f := range files {
		if !strings.HasPrefix(f, tfolder+"/") || !vault.IsNote(f) {
			continue
		}
		rel := f
		items = append(items, choice{
			label: strings.TrimSuffix(strings.TrimPrefix(rel, tfolder+"/"), path.Ext(rel)),
			do: func() {
				rules := make([]config.TemplateRule, 0, len(m.opts.FolderTemplates)+1)
				for _, r := range m.opts.FolderTemplates {
					if r.Folder != folder {
						rules = append(rules, r)
					}
				}
				rules = append(rules, config.TemplateRule{Folder: folder, Template: rel})
				save(rules)
			},
		})
	}
	if len(items) == 0 {
		m.flash = "No templates in " + tfolder + "/ yet"
		return
	}
	sort.Slice(items, func(i, j int) bool { return items[i].label < items[j].label })
	m.openChooser(&chooser{
		title:  "Template for " + m.folderLabel(folder),
		prompt: "Template",
		empty:  "No template by that name",
		verb:   "pair it",
		items:  items,
		cancel: func() { m.pickFolderTemplates() },
	})
}
