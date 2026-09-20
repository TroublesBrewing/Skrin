package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/config"
	"github.com/lurioso/skrin/internal/editor"
)

// paletteEntry is one command in the palette: an action from the
// registry, under a name that says what it does, plus the other words
// someone new might look for it by. The key is never written here: it
// comes from the keymap in force, so a rebound key shows as rebound.
type paletteEntry struct {
	act  action
	name string
	also string
}

// mainCommands is the palette in Files and the note, the most asked-for
// first — the order it shows in before anything is typed.
var mainCommands = []paletteEntry{
	{actEdit, "Edit the note", "write change modify type"},
	{actNewNote, "New note", "create add file page"},
	{actDaily, "Today's daily note", "journal diary date day"},
	{actSearch, "Search the vault", "find grep text look for replace"},
	{actSwitcher, "Go to a note by name", "open file jump quick switcher"},
	{actPin, "Pin this note", "bookmark star favourite"},
	{actPins, "Pinned notes", "bookmarks starred favourites"},
	{actTags, "Tags in the vault", "browse labels #"},
	{actHints, "Follow a link", "jump open wikilink"},
	{actHintsSplit, "Follow a link into a split", "beside wikilink open"},
	{actBacklinks, "Backlinks: notes linking here", "references mentions incoming"},
	{actOutline, "Outline: jump to a heading", "headings toc table of contents sections"},
	{actQuickNote, "Quick note: capture a thought", "inbox jot capture scratch"},
	{actZen, "Zen mode: just the note, centred", "focus distraction free fullscreen reading"},
	{actNewFolder, "New folder", "create directory"},
	{actRename, "Rename", "name title"},
	{actMove, "Move to another folder", "relocate file"},
	{actDelete, "Delete to the trash", "remove trash"},
	{actUndoEdit, "Undo the note's last edit", "revert restore version history"},
	{actRedoEdit, "Redo the note's last edit", "again"},
	{actUndoOp, "Undo the last file operation", "revert restore"},
	{actEditExternal, "Edit in $EDITOR", "vim nvim external editor"},
	{actSkimDown, "Skim down: open the next note beside this one", "split side by side compare"},
	{actSkimUp, "Skim up: open the note above beside this one", "split side by side compare"},
	{actNextHeading, "Next heading", "section"},
	{actPrevHeading, "Previous heading", "section"},
	{actBack, "Go back", "history previous"},
	{actForward, "Go forward", "history next"},
	{actLineNumbers, "Line numbers on or off", "gutter rows"},
	{actHabits, "Habits: today's list, the week and the month", "tracker streak routine"},
	{actNewBook, "Book Card: catalogue a book", "library reading isbn"},
	{actClaude, "Claude drawer: open or hide", "ai assistant chat"},
	{actClaudeInput, "Ask Claude", "ai assistant chat question"},
	{actMark, "Mark or unmark the item", "select"},
	{actVisual, "Mark a range in Files, or select lines in the note", "select visual copy"},
	{actMarkAll, "Mark everything in the folder", "select all"},
	{actCollapseAll, "Close all folders", "collapse fold tree"},
	{actFolderJumpDown, "Next folder in Files", "jump"},
	{actFolderJumpUp, "Previous folder in Files", "jump"},
	{actOrderUp, "Move the item up its level", "arrange order sort reorder"},
	{actOrderDown, "Move the item down its level", "arrange order sort reorder"},
	{actOrderReset, "Put this level back in the default order", "arrange sort alphabetical reset"},
	{actPane1, "Focus Files", "tree sidebar panel"},
	{actPane2, "Focus the note", "panel"},
	{actFindNote, "Find in the note", "search sök hitta text ctrl+f"},
	{actHelp, "Keys: every key, and change them", "help manual shortcuts keybindings hotkeys cheatsheet"},
	{actQuit, "Quit Skrin", "exit close leave"},
}

// notInPalette are the Files-and-note actions the palette leaves out:
// moving a row or a page, stepping in and out, Esc. Those are what keys
// are for, and nobody looks them up by name. TestPaletteCoversTheKeymap
// holds the two lists to the whole registry.
var notInPalette = map[action]bool{
	actUp: true, actDown: true, actTop: true, actBottom: true,
	actHalfDown: true, actHalfUp: true, actLeft: true, actRight: true,
	actOpen: true, actParent: true, actNextPane: true, actPrevPane: true,
	actPaneLeft: true, actPaneRight: true, actEscape: true, actPalette: true,
}

// editorCommand is one command in the palette while editing. Most of the
// editor's keys belong to the editor itself rather than the registry's
// actions, so a row either names an action or carries its own key and
// what running it does.
type editorCommand struct {
	act  action // actNone for the editor's own keys
	key  string // for those, the key; it can't be rebound
	name string
	also string
	run  func(m *Model) tea.Cmd
}

var editorCommands = []editorCommand{
	{key: "ctrl+s", name: "Save", also: "write keep", run: func(m *Model) tea.Cmd { m.saveEdit(false); return nil }},
	{key: "esc", name: "Leave the editor, saving", also: "done close stop exit finish", run: func(m *Model) tea.Cmd { m.saveEdit(true); return nil }},
	{key: "ctrl+l", name: "To-do: make the line one, or tick it off", also: "task checkbox check done", run: editorKeyRun("ctrl+l")},
	{key: "ctrl+z", name: "Undo typing", also: "revert", run: editorKeyRun("ctrl+z")},
	{key: "ctrl+y", name: "Redo typing", also: "again", run: editorKeyRun("ctrl+y")},
	{key: "ctrl+home", name: "Start of the note", also: "top beginning", run: editorKeyRun("ctrl+home")},
	{key: "ctrl+end", name: "End of the note", also: "bottom", run: editorKeyRun("ctrl+end")},
	{act: actAskClaude, name: "Ask Claude about the selection", also: "ai assistant chat"},
	{act: actZen, name: "Zen mode, still editing", also: "focus distraction free fullscreen"},
	{act: actBacklinks, name: "Backlinks: notes linking here", also: "references mentions incoming"},
	{act: actOutline, name: "Outline: jump to a heading", also: "headings toc sections"},
	{act: actFindNote, name: "Find in the note", also: "search sök hitta text"},
	{name: "Insert table", also: "add new grid columns rows markdown tabell", run: func(m *Model) tea.Cmd { m.startTable(); return nil }},
	{name: "Insert template", also: "add snippet boilerplate mall", run: func(m *Model) tea.Cmd { m.startTemplate(); return nil }},
}

// editorKeyRun runs one of the editor's own keys, as if it were pressed.
func editorKeyRun(k string) func(m *Model) tea.Cmd {
	return func(m *Model) tea.Cmd {
		if m.editor != nil {
			m.editor.HandleKey(keyPress(k))
		}
		return nil
	}
}

// keyPress is the key press for the few editor keys the palette runs.
func keyPress(k string) tea.KeyPressMsg {
	switch k {
	case "ctrl+home":
		return tea.KeyPressMsg{Code: tea.KeyHome, Mod: tea.ModCtrl}
	case "ctrl+end":
		return tea.KeyPressMsg{Code: tea.KeyEnd, Mod: tea.ModCtrl}
	}
	c, _ := strings.CutPrefix(k, "ctrl+")
	return tea.KeyPressMsg{Code: []rune(c)[0], Mod: tea.ModCtrl}
}

// paletteRecentMax is how many commands run lately float to the top.
const paletteRecentMax = 5

// openPalette is Ctrl+P: every command there is where you are, found by
// what it does. It is the one door a newcomer needs, so everything else
// that teaches — the welcome screen, the status line, a key that does
// nothing — points at it.
func (m *Model) openPalette() {
	var items []choice
	if m.editor != nil {
		items = m.editorPaletteItems()
	} else {
		items = m.mainPaletteItems()
	}
	items = append(items, m.settingsPaletteItems()...)
	// Commands run lately come first, most recent at the top: coming back
	// for the same thing is most of what a palette is used for.
	var front, rest []choice
	for _, name := range m.recentCmds {
		for _, it := range items {
			if it.label == name {
				front = append(front, it)
			}
		}
	}
	for _, it := range items {
		if !m.recentCmd(it.label) {
			rest = append(rest, it)
		}
	}
	m.openChooser(&chooser{
		title:   "Commands",
		prompt:  "Command",
		empty:   "No command called that — ? lists every key",
		verb:    "run",
		items:   append(front, rest...),
		byWords: true,
	})
}

func (m *Model) recentCmd(name string) bool {
	for _, n := range m.recentCmds {
		if n == name {
			return true
		}
	}
	return false
}

// ran remembers a command for the top of the next palette.
func (m *Model) ran(name string) {
	out := []string{name}
	for _, n := range m.recentCmds {
		if n != name && len(out) < paletteRecentMax {
			out = append(out, n)
		}
	}
	m.recentCmds = out
}

func (m *Model) mainPaletteItems() []choice {
	var items []choice
	for _, e := range mainCommands {
		if !m.opts.Assistant.Enabled && (e.act == actClaude || e.act == actClaudeInput) {
			continue
		}
		items = append(items, m.actionChoice(inMain, e.act, e.name, e.also))
	}
	return append(items,
		choice{label: "Insert table", also: "add new grid columns rows markdown tabell", run: m.paletteRun("Insert table", func() tea.Cmd {
			m.startTable()
			return nil
		})},
		choice{label: "Insert template", also: "add snippet boilerplate mall", run: m.paletteRun("Insert template", func() tea.Cmd {
			m.startTemplate()
			return nil
		})},
		choice{label: "Settings", also: "preferences options config toggles", run: m.paletteRun("Settings", func() tea.Cmd {
			m.openManual()
			m.manualGoTab(manualTabSettings)
			return nil
		})},
		choice{label: "Guide: how Skrin fits together", also: "help docs documentation tutorial getting started learn", run: m.paletteRun("Guide: how Skrin fits together", func() tea.Cmd {
			m.openManual()
			m.manualGoTab(manualTabGuide)
			return nil
		})},
	)
}

// tableCommands are the palette's commands for the table under the
// cursor, Advanced Tables' set. They only show while the cursor is in one.
var tableCommands = []struct {
	op   editor.TableOp
	name string
	also string
	done string
}{
	{editor.TableFormat, "Table: line it up", "format align tidy", "Table lined up"},
	{editor.TableRowBelow, "Table: add a row below", "insert new", "Row added"},
	{editor.TableRowAbove, "Table: add a row above", "insert new", "Row added"},
	{editor.TableRowDelete, "Table: delete this row", "remove", "Row deleted · ctrl+z brings it back"},
	{editor.TableRowUp, "Table: move this row up", "reorder", "Row moved up"},
	{editor.TableRowDown, "Table: move this row down", "reorder", "Row moved down"},
	{editor.TableColRight, "Table: add a column to the right", "insert new", "Column added"},
	{editor.TableColLeft, "Table: add a column to the left", "insert new", "Column added"},
	{editor.TableColDelete, "Table: delete this column", "remove", "Column deleted · ctrl+z brings it back"},
	{editor.TableColMoveLeft, "Table: move this column left", "reorder", "Column moved left"},
	{editor.TableColMoveRight, "Table: move this column right", "reorder", "Column moved right"},
	{editor.TableAlignLeft, "Table: align this column left", "alignment", "Column aligned left"},
	{editor.TableAlignCenter, "Table: centre this column", "alignment center middle", "Column centred"},
	{editor.TableAlignRight, "Table: align this column right", "alignment numbers", "Column aligned right"},
	{editor.TableSortAsc, "Table: sort by this column, A → Z", "order ascending", "Sorted A → Z"},
	{editor.TableSortDesc, "Table: sort by this column, Z → A", "order descending", "Sorted Z → A"},
}

func (m *Model) editorPaletteItems() []choice {
	var items []choice
	if m.editor.InTable() {
		for _, tc := range tableCommands {
			items = append(items, choice{label: tc.name, also: "table tabell " + tc.also, run: m.paletteRun(tc.name, func() tea.Cmd {
				if m.editor == nil {
					return nil
				}
				if err := m.editor.Table(tc.op); err != nil {
					m.flash = err.Error()
				} else {
					m.flash = tc.done
				}
				return nil
			})})
		}
	}
	for _, c := range editorCommands {
		if c.act == actNone {
			run := c.run
			items = append(items, choice{label: c.name, also: c.also, key: statusKey(c.key), run: m.paletteRun(c.name, func() tea.Cmd { return run(m) })})
			continue
		}
		if c.act == actAskClaude && !m.opts.Assistant.Enabled {
			continue
		}
		items = append(items, m.actionChoice(inEditor, c.act, c.name, c.also))
	}
	return items
}

// actionChoice is a palette row for action a in context where, showing
// the key the keymap gives it now.
func (m *Model) actionChoice(where string, a action, name, also string) choice {
	return choice{
		label: name,
		also:  also + " " + actionName[a],
		key:   m.keyFor(where, a),
		run: m.paletteRun(name, func() tea.Cmd {
			cmd := m.runAction(where, a)
			m.teach(where, a)
			return cmd
		}),
	}
}

// paletteRun wraps a row's command so the palette remembers it was run.
func (m *Model) paletteRun(name string, f func() tea.Cmd) func() tea.Cmd {
	return func() tea.Cmd {
		m.ran(name)
		return f()
	}
}

// runAction does what action a's key would have done in context where.
func (m *Model) runAction(where string, a action) tea.Cmd {
	if a == actQuit {
		return tea.Quit
	}
	if where == inEditor && a == actAskClaude {
		m.openDrawer()
		return nil
	}
	return m.do(a)
}

// teach is the palette's lesson: after a command that has a key, the
// status line names it, so the palette teaches the keys you actually
// use. A command that had something to say for itself keeps the line.
func (m *Model) teach(where string, a action) {
	if m.flash != "" {
		return
	}
	if k := m.keyFor(where, a); k != "" {
		m.flash = "Next time: " + k
	}
}

// settingsPaletteItems are the Settings tab's toggles, runnable from the
// palette: the commands that have no key at all.
func (m *Model) settingsPaletteItems() []choice {
	var items []choice
	for _, it := range settingsItems() {
		name := "Setting: " + it.label
		if it.pick != nil {
			items = append(items, choice{label: name, detail: it.value(m), also: "settings preferences option config folder", run: m.paletteRun(name, func() tea.Cmd {
				it.pick(m)
				return nil
			})})
			continue
		}
		state := "off"
		if it.get(m) {
			state = "on"
		}
		items = append(items, choice{
			label:  name,
			detail: state,
			also:   "settings preferences option toggle config",
			run: m.paletteRun(name, func() tea.Cmd {
				v := !it.get(m)
				it.set(m, v)
				now := "off"
				if v {
					now = "on"
				}
				m.flash = it.label + ": " + now
				if err := config.Save(m.opts.Config); err != nil {
					m.flash = "couldn't save settings: " + err.Error()
				}
				return nil
			}),
		})
	}
	return items
}

// keyFor is the first key action a has in context where, written the way
// the status line writes keys; "" when it has none.
func (m *Model) keyFor(where string, a action) string {
	if keys := m.keys.bound(where, a); len(keys) > 0 {
		return statusKey(keys[0])
	}
	return ""
}

// statusKey writes a key the compact way the status line and the palette
// do: "ctrl+p", "shift+→", "space".
func statusKey(k string) string {
	if k == " " {
		return "space"
	}
	glyph := map[string]string{"left": "←", "right": "→", "up": "↑", "down": "↓"}
	parts := strings.Split(k, "+")
	for i, p := range parts {
		if g, ok := glyph[p]; ok {
			parts[i] = g
		}
	}
	return strings.Join(parts, "+")
}
