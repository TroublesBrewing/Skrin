package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/version"
)

// manualTab is which of the ? overlay's tabs is showing.
type manualTab int

const (
	manualTabKeys manualTab = iota
	manualTabSettings
	manualTabGuide
	manualTabs // how many there are
)

var manualTabNames = [manualTabs]string{"Keys", "Settings", "Guide"}

// manual is the ? overlay: every key Skrin knows and what it does, the
// settings config.toml also holds, and the guide to how it all fits
// together. The Keys tab is built from the keymap in force, so it can't
// drift from what the keys actually do — and it is where they are
// changed.
type manual struct {
	tab       manualTab
	in        lineInput // the / filter, on the Keys and Guide tabs
	filtering bool      // typing into the filter
	off       int
	setCur    int // the cursor row in the Settings tab

	cur  bindRef  // the binding under the cursor in the Keys tab
	mode keysMode // browsing it, waiting for a key, or asking
	ask  *keysAsk // the question waiting for y or n
	note string   // a line under the list: what just happened, or why not
}

// manualLine is one line of the manual. Level 1 is a section heading and 2
// a sub-heading; a filter keeps the headings above every line it matches.
type manualLine struct {
	plain, styled string
	level         int
	// search is what the filter matches on: the line as shown, plus the
	// keys as they are written in the config, so looking for "ctrl+k"
	// finds the row that shows Ctrl-k.
	search string
	// bind is set on a row for a key that can be changed, naming which.
	bind *bindRef
}

// manualKeyW is the width of the key column in the key tables.
const manualKeyW = 18

func (m *Model) openManual() {
	m.manual = &manual{}
	m.firstBinding()
}

// manualWidth is the manual's text width: a readable column.
func (m *Model) manualWidth() int { return max(min(88, m.width-6), 20) }

// manualVis is how many lines of the body show at once: the box's height
// less its border, the tab strip and the status line.
func (m *Model) manualVis() int { return max(m.height-6, 1) }

func (m *Model) manualKey(k tea.KeyPressMsg) {
	h := m.manual
	// Tab moves between tabs from anywhere that isn't mid-question or
	// mid-capture — those two own every key while they last, because the
	// whole point of capture is that any key can be the answer.
	if h.mode == keysBrowse {
		switch k.String() {
		case "tab":
			m.manualGoTab((h.tab + 1) % manualTabs)
			return
		case "shift+tab":
			m.manualGoTab((h.tab + manualTabs - 1) % manualTabs)
			return
		case "?":
			m.manual = nil
			return
		}
	}
	switch h.tab {
	case manualTabSettings:
		m.settingsKey(k)
		return
	case manualTabKeys:
		m.keysTabKey(k)
		return
	}
	m.guideKey(k)
}

// manualGoTab shows tab t, from the top, with nothing left over from the
// tab before it.
func (m *Model) manualGoTab(t manualTab) {
	h := m.manual
	h.tab, h.off, h.filtering, h.note = t, 0, false, ""
	h.in.set("")
	if t == manualTabKeys {
		m.firstBinding()
	}
}

// guideKey scrolls the Guide tab: the prose half of the manual.
func (m *Model) guideKey(k tea.KeyPressMsg) {
	h := m.manual
	if h.filtering {
		switch k.String() {
		case "esc":
			h.filtering = false
			h.in.set("")
		case "enter":
			h.filtering = false
		default:
			h.in.handle(k)
		}
		h.off = 0
		return
	}
	vis := m.manualVis()
	maxOff := max(len(m.manualLines(m.manualWidth()))-vis, 0)
	switch k.String() {
	case "esc":
		if h.in.value() != "" {
			h.in.set("")
			h.off = 0
			return
		}
		m.manual = nil
		return
	case "q":
		m.manual = nil
		return
	case "/":
		h.filtering = true
	case "j", "down":
		h.off++
	case "k", "up":
		h.off--
	case "ctrl+d", "pgdown", "space":
		h.off += max(vis/2, 1)
	case "ctrl+u", "pgup":
		h.off -= max(vis/2, 1)
	case "home", "g":
		h.off = 0
	case "G", "end":
		h.off = maxOff
	}
	h.off = clamp(h.off, 0, maxOff)
}

func (m *Model) manualView() string {
	h := m.manual
	w := m.manualWidth()
	margin := strings.Repeat(" ", max((m.width-2-w)/2, 1))
	var strip []string
	for t, name := range manualTabNames {
		if manualTab(t) == h.tab {
			strip = append(strip, m.st.pill.Render(" "+name+" "))
			continue
		}
		strip = append(strip, m.st.muted.Render(" "+name+" "))
	}
	body := []string{margin + strings.Join(strip, " "), ""}
	switch h.tab {
	case manualTabSettings:
		for _, l := range m.settingsView(w) {
			body = append(body, margin+l)
		}
	default:
		lines := m.manualLines(w)
		vis := m.manualVis()
		for i := h.off; i < min(len(lines), h.off+vis); i++ {
			body = append(body, margin+lines[i].styled)
		}
		if len(lines) == 0 {
			body = append(body, "", margin+m.st.muted.Render("Nothing here matches “"+h.in.value()+"”."))
		}
	}
	return strings.Join(append(m.box("Manual", body, m.width, m.height-1, true), m.manualStatus()), "\n")
}

// manualStatus is the overlay's status line: what you can do from here,
// and on the Keys tab whatever it is waiting for.
func (m *Model) manualStatus() string {
	h := m.manual
	switch {
	case h.tab == manualTabKeys:
		return m.keysStatus()
	case h.filtering:
		return spread(m.st.pill.Render(" FILTER ")+" "+h.in.view(m.st.text, m.st.cursor), m.st.muted.Render("enter keep · esc clear"), m.width)
	case h.tab == manualTabSettings:
		return spread(m.st.pill.Render(" SETTINGS ")+" "+m.st.text.Render("saved to config.toml as you go"), m.st.muted.Render("j/k move · enter toggle · tab keys · esc close"), m.width)
	default:
		left := m.st.pill.Render(" GUIDE ")
		if q := h.in.value(); q != "" {
			left += " " + m.st.text.Render("matching “"+q+"” · esc shows everything")
		}
		return spread(left, m.st.muted.Render("j/k scroll · / filter · tab keys · esc close"), m.width)
	}
}

// manualLines is the tab's text as shown at width w: all of it, or with a
// filter only the matching lines, under their headings.
func (m *Model) manualLines(w int) []manualLine {
	all := m.guideText(w)
	if m.manual.tab == manualTabKeys {
		all = m.keyLines(w)
	}
	q := strings.ToLower(strings.TrimSpace(m.manual.in.value()))
	if q == "" {
		return all
	}
	var out []manualLine
	var sec, sub *manualLine
	for i := range all {
		l := &all[i]
		switch l.level {
		case 1:
			sec, sub = l, nil
			continue
		case 2:
			sub = l
			continue
		}
		hay := l.search
		if hay == "" {
			hay = l.plain
		}
		if l.plain == "" || !strings.Contains(strings.ToLower(hay), q) {
			continue
		}
		if sec != nil {
			if len(out) > 0 {
				out = append(out, manualLine{})
			}
			out = append(out, *sec)
			sec = nil
		}
		if sub != nil {
			out = append(out, *sub)
			sub = nil
		}
		out = append(out, *l)
	}
	return out
}

// guideText is the Guide tab at text width w: how Skrin fits together,
// in prose. The keys themselves live in the Keys tab, where they can be
// changed as well as read.
func (m *Model) guideText(w int) []manualLine {
	var out []manualLine
	add := func(level int, plain, styled string) {
		out = append(out, manualLine{plain: plain, styled: styled, level: level})
	}
	blank := func() { add(0, "", "") }
	head := func(s string) {
		if len(out) > 0 {
			blank()
		}
		add(1, s, m.st.titleFocus.Render(s))
	}
	para := func(s string) {
		for _, l := range wrap(s, w) {
			add(0, l, m.st.text.Render(l))
		}
	}
	code := func(s string) { add(0, "  "+s, "  "+m.st.muted.Render(s)) }
	key := func(k, desc string) {
		for i, d := range wrap(desc, max(w-manualKeyW-3, 10)) {
			if i > 0 {
				k = ""
			}
			k = k + strings.Repeat(" ", max(manualKeyW-ansi.StringWidth(k), 0))
			add(0, "  "+k+" "+d, "  "+m.st.flash.Render(k)+" "+m.st.text.Render(d))
		}
	}

	// The chest and the name open the manual.
	pad := func(s string) string { return strings.Repeat(" ", max((w-ansi.StringWidth(s))/2, 0)) + s }
	for _, l := range m.splash[0] {
		add(0, "", pad(l))
	}
	blank()
	add(0, "", pad(m.st.brand.Render("Skrin")+m.st.muted.Render(" v"+version.Version)))
	add(0, "", pad(m.st.muted.Render("a terminal home for your vault")))

	head("Files and the note")
	para("Files holds the vault's folders and files. The note under the cursor opens at once, so j and k skim through your notes; on a folder, the last note stays open. Enter moves over to the note.")
	para("Following a link, a search hit, Go to note, t and going back all move the Files cursor to the note they open.")
	para("In Files, keys act on the row under the cursor; in the note, on the open note. When anything is marked, m and d act on the marks.")
	para("n and N create things in the current folder: the folder under the cursor, or the folder of the file under it. In the note it's the open note's folder.")
	para("t opens today's daily note, made from Obsidian's daily-notes settings, with the unfinished todos of the last one carried over.")

	head("The size of things")
	para("Making the text bigger or smaller is the terminal's own job, not Skrin's — in most terminals Ctrl+- and Ctrl++, which never reach Skrin at all. What Skrin is told is the new size in columns and rows, and it shows that in the status line for a moment whenever it changes, so a zoom you just made visibly lands. Zoom in and the numbers fall; zoom out and they rise.")
	para("Two widths are worth knowing. Below 80 columns Files only shows while it has the focus, so the note keeps the screen — the indicator says so, because Files going quiet otherwise looks like a fault. Below 80 there's also no room for a split, and an open one closes, saying which note it closed.")

	head("Split view")
	para("In Go to note, Shift+→ opens the note beside the one you're reading, on the right, and Shift+← on the left. Enter still opens it in place.")
	para("In the note, Shift+← and Shift+→ move between the two panes; keys act on the one with the bright border. Esc closes the pane you're in, and z (zen) closes the other.")
	para("Two is the most, so a new split replaces the older one. Below 80 columns there's no room for one, and splits aren't remembered when you quit.")

	head("Your own order")
	para("In Files, Shift+↑ and Shift+↓ move the item under the cursor up and down its level. There's no mode to enter: the arrows move the cursor, and with Shift they move the thing under it. R puts the level back in the default order.")
	para("Within a level anything can sit anywhere, a file above a folder too. New items appear at the end of an ordered level, and renaming or moving keeps an item's place.")
	para("The order lives in skrin.json at the vault root, invisible in the tree but not a dotfile, so it syncs with the vault: arrange on one machine and it follows to the next. Obsidian keeps its own alphabetical order and doesn't show the file either. Deleting skrin.json puts everything back. Each move is one step: U takes back the last one.")

	head("Claude")
	para("c opens the Claude drawer and C puts you straight into typing; Ctrl-k does it from the editor. The drawer sits along the bottom, or on the right with Alt-p or assistant.position = \"right\".")
	para("Claude sees the open note (the focused one, in a split), sent along with your message whenever it has changed. Highlight text first (v and j/k in the note, Shift and the arrows in the editor) and it goes into your message.")
	para("Claude can read the whole vault but can't change anything by itself. A change it wants shows as a diff in the note panel: y applies it, n rejects it. u undoes an applied edit, and U a create, move or delete.")
	para("The conversation carries on after you quit, one per vault; Alt-n starts a fresh one. assistant.enabled = false turns the drawer off.")

	head("Search")
	para("/ searches as you type, over the whole vault or just this note. The syntax is a subset of Obsidian's:")
	key("word word", "all words must appear")
	key(`"exact phrase"`, "the words in this order")
	key("-word", "leave out notes with it")
	key("a OR b", "either")
	key("#tag  tag:#tag", "a tag; #a finds #a/b too")
	key("[key] [key:value]", "a frontmatter property")
	key("path:Daily", "notes in that folder")
	key("file:stoic", "notes by name")
	para("With Alt-r it becomes search & replace: every change is listed first, ⚠ marks matches inside [[links]], a note that changed on disk meanwhile is skipped, and one U undoes the whole replace.")

	head("Links")
	para("[[Note]], [[Note|alias]], [[Note#Heading]], [[Note#^block]], ![[embeds]] and markdown [text](path) links all work, and resolve the way Obsidian resolves them.")
	para("f puts a letter on each link in view; type it to follow. Enter follows the link when there's only one in view.")
	para("A link to a note that doesn't exist yet is dimmed; following it offers to create the note.")
	para("Backspace or Ctrl-o goes back, Ctrl-i goes forward, and b lists the notes linking here.")
	para("Renaming or moving offers to update the links to what moved, as Obsidian does.")

	head("Spreads")
	para("A spread is a small query in a ```spread block. In the note it shows what it finds — a table or a list of notes — and stays current as notes change. The query language is a subset of Dataview's, so a ```dataview block works the same, and shows the same in Obsidian with Dataview installed.")
	key(`TABLE a, b AS "B"`, "a table: the note, then those fields")
	key("LIST [field]", "a list of notes, with one value each")
	key("TASK", "the checkboxes in those notes, under each note")
	key(`FROM #tag  "Folder"`, "where to look; #a finds #a/b too")
	key("FROM [[note]]", "notes linking to it; outgoing([[note]]) for its links")
	key("FROM a AND -b", "combine with AND, OR, - and parentheses")
	key("WHERE rating >= 4", "= != < <= > >=, AND, OR, !, null")
	key(`contains(f, "x")`, "a list holding x, or text containing it")
	key("SORT f DESC", "order by one field or more")
	key("LIMIT 10", "at most that many")
	para("Fields are the note's frontmatter properties, plus file.name, file.link, file.folder, file.path, file.tags, file.mtime and file.size. Numbers and ISO dates compare as numbers and dates.")
	para("A field can also live in the note's text, Dataview's way: \"rating:: 4\" on a line of its own, or \"[due:: 2026-09-22]\" or \"(due:: 2026-09-22)\" in a sentence. A name with spaces is queried with dashes: \"Due Date::\" is due-date.")
	para("In a TASK spread, WHERE and SORT look at each task: text, status (what's between the brackets), completed, checked, line, and the fields on the task's own line — so \"- [ ] call the printer [due:: 2026-09-22]\" answers to WHERE !completed AND due <= \"2026-09-30\". A task shows with the tasks nested under it. Checkboxes under ### Habits aren't tasks, and don't show.")
	para("A due date can be written as a word: today, tomorrow, or a weekday. Leaving the editor turns it into the date it means — \"due:: friday\" becomes the coming Friday, the next one if it's Friday already — on the lines you typed or changed, and the status line says what it set. The note then holds a real date, which spreads and Obsidian both read.")
	para("Every note in the answer is a link: f and Enter follow it, and a task's ↗ opens its note at that task, where Ctrl-l ticks it. A spread only reads, never writes. Without LIMIT it stops at 200 rows and says how many more there are; a mistake, or Dataview syntax Skrin doesn't do yet, shows as a ⚠ line naming it.")

	head("Undo")
	para("u puts the note back as it was before its last edit, whether you made it here, in $EDITOR, with a replace, or through Claude. Ctrl-r takes that back.")
	para("U undoes the last file operation: a create, rename, move, delete (back from the trash) or replace.")
	para("In the editor, Ctrl-z undoes typing.")
	para("Earlier versions of each note are kept in ~/.local/state/skrin/snapshots, the last 20 per note.")

	head("Config")
	para("The Settings tab of this manual (Tab) toggles the everyday ones without an editor; everything else, plus anything hand-set, lives in ~/.config/skrin/config.toml:")
	blank()
	code(`vault = "~/notes"        # the vault to open; else Obsidian's own list`)
	code(`[general]`)
	code(`restore_last_note = false  # reopen the last note on start`)
	code(`[daily]`)
	code(`rollover_todos = true   # t carries unfinished todos over`)
	code(`[editor]`)
	code(`vim = false             # vim keys in the built-in editor`)
	code(`external = "nvim"       # what E runs; else $VISUAL, $EDITOR, nvim`)
	code(`[assistant]`)
	code(`enabled = true          # the Claude drawer`)
	code(`position = "bottom"     # or "right"`)
	code(`model = ""              # else Claude Code's default, e.g. "sonnet"`)
	code(`[render]`)
	code(`images = true           # block-art previews of image embeds`)
	code(`line_numbers = false    # line numbers in notes and the editor`)
	code(`spreads = true          # spread blocks show what they find`)
	blank()
	para("Keys you change in the Keys tab are saved under [keys], one table per context, against the name of the thing the key does:")
	blank()
	code(`[keys.main]`)
	code(`zen = ["ctrl+g"]        # z becomes Ctrl-g`)
	blank()
	para("Only what you changed is kept there, so everything else still follows Skrin's own keys as they change. Deleting the table, or R in the Keys tab, puts the lot back.")
	blank()
	para("Colours come from the Omarchy theme and follow it live. A skrin.toml next to the theme's colors.toml can override them.")
	para("Where you were (open folders, the cursor, the open note, the Claude conversation) is kept per vault in ~/.local/state/skrin/session, but the note itself only reopens if restore_last_note is on: off by default, so a vault with private notes never opens one by surprise.")
	return out
}

// keyLabel names keys the way the manual shows them: "ctrl+d" as Ctrl-d,
// "left" as ←, and each key once.
func keyLabel(keys []string) string {
	names := map[string]string{
		"left": "←", "right": "→", "up": "↑", "down": "↓", "enter": "Enter", "backspace": "Backspace",
		"space": "Space", " ": "Space", "tab": "Tab", "esc": "Esc", "home": "Home", "end": "End",
		"pgup": "PgUp", "pgdown": "PgDn", "ctrl": "Ctrl", "alt": "Alt", "shift": "Shift",
	}
	var out []string
	seen := map[string]bool{}
	for _, k := range keys {
		parts := strings.Split(k, "+")
		if k == " " {
			parts = []string{" "}
		}
		for i, p := range parts {
			if n, ok := names[p]; ok {
				parts[i] = n
			}
		}
		if s := strings.Join(parts, "-"); !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return strings.Join(out, " ")
}

// wrap breaks s into lines of at most w cells, at spaces.
func wrap(s string, w int) []string {
	var out []string
	line := ""
	for _, word := range strings.Fields(s) {
		switch {
		case line == "":
			line = word
		case ansi.StringWidth(line)+1+ansi.StringWidth(word) <= w:
			line += " " + word
		default:
			out = append(out, line)
			line = word
		}
	}
	return append(out, line)
}
