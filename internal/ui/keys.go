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
