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
	actPaneLeft
	actPaneRight
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
	actClaude
	actClaudeInput
	actZen
	actHelp
	actQuit
	actAskClaude // editor
	actPick      // lists
	actCancel
	actSplitLeft
	actSplitRight
	actSend // the Claude drawer
	actNewLine
	actLeaveDrawer
	actFlipDrawer
	actNewChat
	actScrollBack
	actScrollOn
	actApply // a change Claude proposes
	actReject
	actOrderUp // Files' own order, kept in .skrin
	actOrderDown
	actOrderReset
	actFindScope // the search panel
	actFindCase
	actFindWords
	actFindReplace
	actNextField
	actPrevField
	actReplaceAll
	actSkipMatch
)

// Contexts say where a binding works. Each gets its own keymap, built from
// the registry below.
const (
	inMain     = "main" // Files and the note
	inEditor   = "editor"
	inList     = "list" // Go to note, backlinks, the outline, the move picker
	inSearch   = "search"
	inDrawer   = "drawer"
	inProposal = "proposal"
	inHints    = "hints"
	inAsk      = "ask" // a name to type, or a y/n question
	inConflict = "conflict"
	inManual   = "manual"
	inComplete = "complete" // the [[ popup in the editor
)

// binding is one row of the keymap registry. Every key Skrin handles is
// listed here, with help text for the context it works in; the ? manual is
// generated from this table. A binding with actNone is handled inside its
// component (the editor, the search panel), and is here to be documented.
type binding struct {
	act   action
	keys  []string
	help  string
	group string // the manual's heading for it
	where string // the context it works in
}

const (
	groupMove     = "Moving around"
	groupFiles    = "Files and folders"
	groupNote     = "The note"
	groupSearch   = "Search"
	groupClaude   = "Claude"
	groupSkrin    = "Skrin"
	groupEditor   = "In the editor"
	groupVim      = "Vim keys in the editor (editor.vim = true)"
	groupList     = "In lists: Go to note, backlinks, the outline, moving"
	groupFind     = "In search (/)"
	groupDrawer   = "In the Claude drawer"
	groupProposal = "On a change Claude proposes"
	groupHints    = "Following links (f)"
	groupAsk      = "When Skrin asks"
	groupConflict = "On a save conflict"
	groupManual   = "In this manual"
	groupComplete = "In link completion ([[)"
)

var defaultBindings = []binding{
	{actDown, []string{"j", "down"}, "move down / scroll", groupMove, inMain},
	{actUp, []string{"k", "up"}, "move up / scroll", groupMove, inMain},
	{actTop, []string{"home"}, "go to the top (GG too)", groupMove, inMain},
	{actBottom, []string{"G", "end"}, "go to the bottom", groupMove, inMain},
	{actHalfDown, []string{"ctrl+d", "pgdown"}, "half a page down", groupMove, inMain},
	{actHalfUp, []string{"ctrl+u", "pgup"}, "half a page up", groupMove, inMain},
	{actLeft, []string{"h", "left"}, "Files: close the folder, or up one · note: back to Files", groupMove, inMain},
	{actRight, []string{"l", "right"}, "Files: into the folder, or over to the note", groupMove, inMain},
	{actOpen, []string{"enter"}, "Files: toggle folder / go to note / open file · note: follow link", groupMove, inMain},
	{actParent, []string{"backspace"}, "Files: up to the parent folder · note: go back", groupMove, inMain},
	{actCollapseAll, []string{"H"}, "close all folders", groupMove, inMain},
	{actNextPane, []string{"tab"}, "switch between Files and the note", groupMove, inMain},
	{actPrevPane, []string{"shift+tab"}, "switch between Files and the note", groupMove, inMain},
	{actPane1, []string{"1"}, "Files", groupMove, inMain},
	{actPane2, []string{"2"}, "the note", groupMove, inMain},
	{actPaneLeft, []string{"shift+left"}, "note, split: to the pane on the left", groupMove, inMain},
	{actPaneRight, []string{"shift+right"}, "note, split: to the pane on the right", groupMove, inMain},
	{actBack, []string{"ctrl+o", "alt+left"}, "go back", groupMove, inMain},
	{actForward, []string{"ctrl+i", "alt+right"}, "go forward", groupMove, inMain},
	{actNewNote, []string{"n"}, "new note in the current folder", groupFiles, inMain},
	{actNewFolder, []string{"N"}, "new folder in the current folder", groupFiles, inMain},
	{actRename, []string{"r"}, "rename", groupFiles, inMain},
	{actMove, []string{"m"}, "move (the marked items, if any)", groupFiles, inMain},
	{actDelete, []string{"d"}, "delete to the trash (the marked items, if any)", groupFiles, inMain},
	{actMark, []string{"space", " "}, "mark / unmark", groupFiles, inMain},
	{actVisual, []string{"v"}, "Files: mark a range · note: select whole lines", groupFiles, inMain},
	{actMarkAll, []string{"ctrl+a"}, "mark everything in the cursor's folder", groupFiles, inMain},
	{actEscape, []string{"esc"}, "clear marks · split: close the pane you're in · zen: leave it", groupFiles, inMain},
	{actUndoOp, []string{"U"}, "undo the last file operation", groupFiles, inMain},
	{actDaily, []string{"t"}, "open or create today's daily note", groupFiles, inMain},
	{actOrderUp, []string{"shift+up"}, "move the item under the cursor up its level", groupFiles, inMain},
	{actOrderDown, []string{"shift+down"}, "move it down its level", groupFiles, inMain},
	{actOrderReset, []string{"R"}, "put this level back in the default order", groupFiles, inMain},
	{actEdit, []string{"e"}, "edit the note in Skrin", groupNote, inMain},
	{actEditExternal, []string{"E"}, "edit the note in $EDITOR", groupNote, inMain},
	{actUndoEdit, []string{"u"}, "undo the note's last edit", groupNote, inMain},
	{actRedoEdit, []string{"ctrl+r"}, "redo it", groupNote, inMain},
	{actHints, []string{"f"}, "follow a link: letters appear on each", groupNote, inMain},
	{actBacklinks, []string{"b"}, "notes linking here", groupNote, inMain},
	{actOutline, []string{"o"}, "outline: jump to a heading", groupNote, inMain},
	{actNextHeading, []string{"}"}, "next heading", groupNote, inMain},
	{actPrevHeading, []string{"{"}, "previous heading", groupNote, inMain},
	{actSearch, []string{"/"}, "search (Alt-r in there: search & replace)", groupSearch, inMain},
	{actSwitcher, []string{"g", "ctrl+p"}, "go to a note by name (Shift+←/→ there: split)", groupSearch, inMain},
	{actClaude, []string{"c"}, "open or hide the Claude drawer", groupClaude, inMain},
	{actClaudeInput, []string{"C"}, "type to Claude, with the highlighted text if any", groupClaude, inMain},
	{actZen, []string{"z"}, "zen mode: just the note, centred", groupSkrin, inMain},
	{actHelp, []string{"?"}, "this manual", groupSkrin, inMain},
	{actQuit, []string{"q", "ctrl+c"}, "quit", groupSkrin, inMain},

	{actNone, []string{"ctrl+s"}, "save", groupEditor, inEditor},
	{actNone, []string{"esc", "ctrl+c"}, "leave the editor, saving first (esc clears a selection first)", groupEditor, inEditor},
	{actNone, []string{"ctrl+z", "ctrl+y"}, "undo / redo typing", groupEditor, inEditor},
	{actNone, []string{"tab", "shift+tab"}, "indent / outdent", groupEditor, inEditor},
	{actNone, []string{"ctrl+l"}, "make the line a to-do, or tick it off", groupEditor, inEditor},
	{actNone, []string{"[["}, "link completion: note names and aliases; add # for headings", groupEditor, inEditor},
	{actNone, []string{"shift+left", "shift+right", "shift+up", "shift+down"}, "select text; typing replaces it", groupEditor, inEditor},
	{actAskClaude, []string{"ctrl+k"}, "ask Claude, with the selection", groupEditor, inEditor},
	{actNone, []string{"ctrl+left", "ctrl+right"}, "a word left / right", groupEditor, inEditor},
	{actNone, []string{"ctrl+home", "ctrl+end"}, "start / end of the note", groupEditor, inEditor},
	{actNone, []string{"i", "a", "I", "A", "o", "O"}, "insert", groupVim, inEditor},
	{actNone, []string{"h", "j", "k", "l", "w", "b", "e"}, "move", groupVim, inEditor},
	{actNone, []string{"0", "^", "$", "gg", "G"}, "line start, first letter, line end, top, bottom", groupVim, inEditor},
	{actNone, []string{"x", "D", "J", "dd"}, "delete a letter, the rest of the line, join, delete the line", groupVim, inEditor},
	{actNone, []string{"yy", "p", "P"}, "copy the line, paste below / above", groupVim, inEditor},
	{actNone, []string{"u", "ctrl+r"}, "undo / redo", groupVim, inEditor},
	{actNone, []string{"v"}, "select; d or x deletes the selection", groupVim, inEditor},
	{actNone, []string{"esc"}, "to normal mode; from normal mode, leave the editor", groupVim, inEditor},

	{actNone, []string{"letters"}, "filter the list", groupList, inList},
	{actUp, []string{"up", "ctrl+p", "shift+tab"}, "up the list", groupList, inList},
	{actDown, []string{"down", "ctrl+n", "tab"}, "down the list", groupList, inList},
	{actPick, []string{"enter"}, "open or choose the row", groupList, inList},
	{actSplitLeft, []string{"shift+left"}, "Go to note: open the note in a split, on the left", groupList, inList},
	{actSplitRight, []string{"shift+right"}, "Go to note: open the note in a split, on the right", groupList, inList},
	{actCancel, []string{"esc", "ctrl+c"}, "close the list", groupList, inList},

	{actNone, []string{"letters"}, "the search, as you type (see Search below)", groupFind, inSearch},
	{actFindScope, []string{"alt+t"}, "this note, or the whole vault", groupFind, inSearch},
	{actFindCase, []string{"alt+c"}, "match case", groupFind, inSearch},
	{actFindReplace, []string{"alt+r"}, "search & replace on or off", groupFind, inSearch},
	{actFindWords, []string{"alt+w"}, "replacing: whole words only", groupFind, inSearch},
	{actNextField, []string{"tab"}, "next field: search, replacement, results", groupFind, inSearch},
	{actPrevField, []string{"shift+tab"}, "the field before it", groupFind, inSearch},
	{actUp, []string{"up", "ctrl+p"}, "up through the results", groupFind, inSearch},
	{actDown, []string{"down", "ctrl+n"}, "down through them", groupFind, inSearch},
	{actNone, []string{"j", "k"}, "in the results: move", groupFind, inSearch},
	{actHalfUp, []string{"pgup"}, "ten results up", groupFind, inSearch},
	{actHalfDown, []string{"pgdown"}, "ten results down", groupFind, inSearch},
	{actPick, []string{"enter"}, "open the note at the result (replacing: to the next field)", groupFind, inSearch},
	{actSkipMatch, []string{"space", " "}, "replacing, in the results: skip a match, or a whole note", groupFind, inSearch},
	{actReplaceAll, []string{"ctrl+s"}, "replacing: replace them all, after a y/n", groupFind, inSearch},
	{actCancel, []string{"esc", "ctrl+c"}, "close; esc leaves replace first, / brings the search back", groupFind, inSearch},

	{actSend, []string{"enter"}, "send", groupDrawer, inDrawer},
	{actNewLine, []string{"alt+enter", "shift+enter"}, "a new line", groupDrawer, inDrawer},
	{actLeaveDrawer, []string{"esc", "ctrl+c"}, "back to where you were (esc clears a selection first)", groupDrawer, inDrawer},
	{actFlipDrawer, []string{"alt+p"}, "move the drawer: along the bottom or on the right", groupDrawer, inDrawer},
	{actNewChat, []string{"alt+n"}, "start a new conversation", groupDrawer, inDrawer},
	{actScrollBack, []string{"pgup"}, "scroll the conversation back", groupDrawer, inDrawer},
	{actScrollOn, []string{"pgdown"}, "scroll it on", groupDrawer, inDrawer},

	{actApply, []string{"y", "Y"}, "apply the change", groupProposal, inProposal},
	{actReject, []string{"n", "N", "esc"}, "reject it; Claude is told", groupProposal, inProposal},
	{actDown, []string{"j", "down"}, "scroll the change", groupProposal, inProposal},
	{actUp, []string{"k", "up"}, "scroll it back", groupProposal, inProposal},

	{actNone, []string{"letters"}, "follow the link with that label", groupHints, inHints},
	{actNone, []string{"esc"}, "stop without following", groupHints, inHints},

	{actNone, []string{"enter"}, "a name: accept it", groupAsk, inAsk},
	{actNone, []string{"y", "n"}, "a question: yes or no", groupAsk, inAsk},
	{actNone, []string{"esc"}, "cancel", groupAsk, inAsk},

	{actNone, []string{"m"}, "keep mine: save over the version on disk (kept as a snapshot)", groupConflict, inConflict},
	{actNone, []string{"t"}, "take theirs (yours is kept as a snapshot)", groupConflict, inConflict},
	{actNone, []string{"d"}, "show or hide the diff", groupConflict, inConflict},
	{actNone, []string{"j", "k"}, "scroll the diff", groupConflict, inConflict},
	{actNone, []string{"esc"}, "keep editing, without saving", groupConflict, inConflict},

	{actUp, []string{"up", "ctrl+p"}, "up the list", groupComplete, inComplete},
	{actDown, []string{"down", "ctrl+n"}, "down the list", groupComplete, inComplete},
	{actPick, []string{"enter", "tab"}, "put the link in", groupComplete, inComplete},
	{actCancel, []string{"esc"}, "close the popup and carry on typing", groupComplete, inComplete},

	{actNone, []string{"j", "k", "ctrl+d", "ctrl+u", "space"}, "scroll", groupManual, inManual},
	{actNone, []string{"home", "g", "G", "end"}, "top / bottom", groupManual, inManual},
	{actNone, []string{"/"}, "filter the manual", groupManual, inManual},
	{actNone, []string{"esc", "?", "q"}, "clear the filter, then close", groupManual, inManual},
}

// keymaps is each context's keymap: key → action.
var keymaps = func() map[string]map[string]action {
	out := map[string]map[string]action{}
	for _, b := range defaultBindings {
		if b.act == actNone {
			continue
		}
		if out[b.where] == nil {
			out[b.where] = map[string]action{}
		}
		for _, k := range b.keys {
			out[b.where][k] = b.act
		}
	}
	return out
}()

// actionIn is the action key stands for in context where, or actNone.
func actionIn(where, key string) action { return keymaps[where][key] }
