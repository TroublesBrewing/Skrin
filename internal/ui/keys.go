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
	actPane3
	actOpen
	actParent
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
	actQuit
)

// binding is one row of the keymap registry. Every key Skrin handles is
// listed here; the ? manual and config.toml overrides build on this table.
type binding struct {
	act  action
	keys []string
	help string
}

var defaultBindings = []binding{
	{actDown, []string{"j", "down"}, "move down / scroll"},
	{actUp, []string{"k", "up"}, "move up / scroll"},
	{actTop, []string{"g", "home"}, "go to top"},
	{actBottom, []string{"G", "end"}, "go to bottom"},
	{actHalfDown, []string{"ctrl+d", "pgdown"}, "half page down"},
	{actHalfUp, []string{"ctrl+u", "pgup"}, "half page up"},
	{actLeft, []string{"h", "left"}, "pane to the left"},
	{actRight, []string{"l", "right"}, "pane to the right"},
	{actNextPane, []string{"tab"}, "next pane"},
	{actPrevPane, []string{"shift+tab"}, "previous pane"},
	{actPane1, []string{"1"}, "folder tree"},
	{actPane2, []string{"2"}, "folder contents"},
	{actPane3, []string{"3"}, "note"},
	{actOpen, []string{"enter"}, "expand folder / open"},
	{actParent, []string{"backspace"}, "parent folder / back"},
	{actNewNote, []string{"n"}, "new note in the current folder"},
	{actNewFolder, []string{"N"}, "new folder in the current folder"},
	{actRename, []string{"r"}, "rename"},
	{actMove, []string{"m"}, "move (the marked items, if any)"},
	{actDelete, []string{"d"}, "delete to the trash (the marked items, if any)"},
	{actMark, []string{"space", " "}, "mark / unmark"},
	{actVisual, []string{"v"}, "mark a range"},
	{actMarkAll, []string{"ctrl+a"}, "mark everything in the folder"},
	{actEscape, []string{"esc"}, "clear marks"},
	{actUndoOp, []string{"U"}, "undo the last file operation"},
	{actDaily, []string{"t"}, "open or create today's daily note"},
	{actQuit, []string{"q", "ctrl+c"}, "quit"},
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
