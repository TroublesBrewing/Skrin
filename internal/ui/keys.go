package ui

type action int

const (
	actNone action = iota
	actUp
	actDown
	actTop
	actBottom
	actHalfDown
	actHalfUp
	actLeft
	actRight
	actNextPane
	actPrevPane
	actPane1
	actPane2
	actOpen
	actParent
	actCollapseAll
	actNewNote
	actNewFolder
	actRename
	actMove
	actDelete
	actMark
	actVisual
	actMarkAll
	actEscape
	actUndoOp
	actDaily
	actEdit
	actEditExternal
	actUndoEdit
	actRedoEdit
	actHints
	actBacklinks
	actOutline
	actNextHeading
	actPrevHeading
	actBack
	actForward
	actSearch
	actSwitcher
	actZen
	actHelp
	actQuit
)

// binding is one row of the keymap registry. Every key Skrin handles is
// listed here; the ? manual and config.toml overrides build on this table.
type binding struct {
	act   action
	keys  []string
	help  string
	group string // the heading it sits under in the ? manual
}

const (
	groupMove   = "Moving around"
	groupFiles  = "Files and folders"
	groupNote   = "The note"
	groupSearch = "Search"
	groupSkrin  = "Skrin"
)

var defaultBindings = []binding{
	{actDown, []string{"j", "down"}, "move down / scroll", groupMove},
	{actUp, []string{"k", "up"}, "move up / scroll", groupMove},
	{actTop, []string{"home"}, "go to the top (GG too)", groupMove},
	{actBottom, []string{"G", "end"}, "go to the bottom", groupMove},
	{actHalfDown, []string{"ctrl+d", "pgdown"}, "half a page down", groupMove},
	{actHalfUp, []string{"ctrl+u", "pgup"}, "half a page up", groupMove},
	{actLeft, []string{"h", "left"}, "Files: close the folder, or up one · note: back to Files", groupMove},
	{actRight, []string{"l", "right"}, "Files: into the folder, or open the file", groupMove},
	{actOpen, []string{"enter"}, "open/close a folder, open a file / follow a link", groupMove},
	{actParent, []string{"backspace"}, "Files: up to the parent folder · note: go back", groupMove},
	{actCollapseAll, []string{"H"}, "close all folders", groupMove},
	{actNextPane, []string{"tab"}, "switch panel", groupMove},
	{actPrevPane, []string{"shift+tab"}, "switch panel", groupMove},
	{actPane1, []string{"1"}, "Files", groupMove},
	{actPane2, []string{"2"}, "the note", groupMove},
	{actBack, []string{"ctrl+o", "alt+left"}, "go back", groupMove},
	{actForward, []string{"ctrl+i", "alt+right"}, "go forward", groupMove},
	{actNewNote, []string{"n"}, "new note in the current folder", groupFiles},
	{actNewFolder, []string{"N"}, "new folder in the current folder", groupFiles},
	{actRename, []string{"r"}, "rename", groupFiles},
	{actMove, []string{"m"}, "move (the marked items, if any)", groupFiles},
	{actDelete, []string{"d"}, "delete to the trash (the marked items, if any)", groupFiles},
	{actMark, []string{"space", " "}, "mark / unmark", groupFiles},
	{actVisual, []string{"v"}, "mark a range", groupFiles},
	{actMarkAll, []string{"ctrl+a"}, "mark everything in the cursor's folder", groupFiles},
	{actEscape, []string{"esc"}, "clear marks; in zen mode, leave it", groupFiles},
	{actUndoOp, []string{"U"}, "undo the last file operation", groupFiles},
	{actDaily, []string{"t"}, "open or create today's daily note", groupFiles},
	{actEdit, []string{"e"}, "edit the note in Skrin", groupNote},
	{actEditExternal, []string{"E"}, "edit the note in $EDITOR", groupNote},
	{actUndoEdit, []string{"u"}, "undo the note's last edit", groupNote},
	{actRedoEdit, []string{"ctrl+r"}, "redo it", groupNote},
	{actHints, []string{"f"}, "follow a link: letters appear on each", groupNote},
	{actBacklinks, []string{"b"}, "notes linking here", groupNote},
	{actOutline, []string{"o"}, "outline: jump to a heading", groupNote},
	{actNextHeading, []string{"}"}, "next heading", groupNote},
	{actPrevHeading, []string{"{"}, "previous heading", groupNote},
	{actSearch, []string{"/"}, "search (Alt-r in there: search & replace)", groupSearch},
	{actSwitcher, []string{"g", "ctrl+p"}, "go to a note by name", groupSearch},
	{actZen, []string{"z"}, "zen mode: just the note, centred", groupSkrin},
	{actHelp, []string{"?"}, "this manual", groupSkrin},
	{actQuit, []string{"q", "ctrl+c"}, "quit", groupSkrin},
}

func keymap() map[string]action {
	m := map[string]action{}
	for _, b := range defaultBindings {
		for _, k := range b.keys {
			m[k] = b.act
		}
	}
	return m
}
