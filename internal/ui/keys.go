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
	actOrderUp // Files' own order, kept in skrin.json
	actOrderDown
	actOrderReset
	actSkimDown  // Alt+↓ / Alt+j: the cursor moves and the note it lands
	actSkimUp    // on opens in the split beside the one already open
	actFindScope // the search panel
	actFindCase
	actFindWords
	actFindReplace
	actNextField
	actPrevField
	actReplaceAll
	actSkipMatch
	actNewBook
	actFetchBook
	actAddQuote
	actSaveBook
	actFolderJumpUp // Ctrl+↑/↓: cursor to the previous/next folder row
	actFolderJumpDown
	actHabits    // T: the habits overlay
	actHabitTab  // H inside it: today → this week → this month
	actQuickNote // i: the quick-note overlay
	actLineNumbers
	actPalette // Ctrl+P: every command, found by name
)

// Contexts say where a binding works. Each gets its own keymap, built from
// the registry below.
const (
	inMain      = "main" // Files and the note
	inEditor    = "editor"
	inList      = "list" // Go to note, backlinks, the outline, the move picker
	inSearch    = "search"
	inDrawer    = "drawer"
	inProposal  = "proposal"
	inHints     = "hints"
	inAsk       = "ask" // a name to type, or a y/n question
	inConflict  = "conflict"
	inManual    = "manual"    // the ? overlay's Keys tab
	inSettings  = "settings"  // its Settings tab
	inGuide     = "guide"     // its Guide tab
	inComplete  = "complete"  // the [[ popup in the editor
	inBookCard  = "bookcard"  // the B card: bibliographic fields, quotes, notes
	inHabits    = "habits"    // the T overlay: today's list, the week and month grids
	inQuickNote = "quicknote" // the i overlay: capture text, folder row
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
	groupMove      = "Moving around"
	groupFiles     = "Files and folders"
	groupNote      = "The note"
	groupSearch    = "Search"
	groupClaude    = "Claude"
	groupSkrin     = "Skrin"
	groupEditor    = "In the editor"
	groupVim       = "Vim keys in the editor (editor.vim = true)"
	groupList      = "In lists: Go to note, backlinks, the outline, moving"
	groupFind      = "In search (/)"
	groupDrawer    = "In the Claude drawer"
	groupProposal  = "On a change Claude proposes"
	groupHints     = "Following links (f)"
	groupAsk       = "When Skrin asks"
	groupConflict  = "On a save conflict"
	groupManual    = "In the Keys tab (?)"
	groupSettings  = "In Settings"
	groupGuide     = "In the Guide"
	groupComplete  = "In link completion ([[)"
	groupBookCard  = "In the Book Card (B)"
	groupHabits    = "In the habits view (T)"
	groupQuickNote = "In the quick note (i)"
)

var defaultBindings = []binding{
	{actDown, []string{"j", "down"}, "move down / scroll", groupMove, inMain},
	{actUp, []string{"k", "up"}, "move up / scroll", groupMove, inMain},
	{actTop, []string{"home"}, "go to the top (GG too)", groupMove, inMain},
	{actBottom, []string{"G", "end"}, "go to the bottom", groupMove, inMain},
	{actHalfDown, []string{"ctrl+d", "pgdown"}, "half a page down", groupMove, inMain},
	{actHalfUp, []string{"ctrl+u", "pgup"}, "half a page up", groupMove, inMain},
	{actLeft, []string{"h", "left"}, "Files: close the folder, or up one · note: back to Files", groupMove, inMain},
	{actRight, []string{"l", "right"}, "Files: opens the folder; a note reads · note: over to the note", groupMove, inMain},
	{actFolderJumpUp, []string{"ctrl+up"}, "Files: jump to the previous folder row", groupMove, inMain},
	{actFolderJumpDown, []string{"ctrl+down"}, "Files: jump to the next folder row", groupMove, inMain},
	{actOpen, []string{"enter"}, "Files: toggle folder / edit note / open file · note: follow link", groupMove, inMain},
	{actParent, []string{"backspace"}, "Files: up to the parent folder · note: go back", groupMove, inMain},
	{actCollapseAll, []string{"H"}, "close all folders", groupMove, inMain},
	{actNextPane, []string{"tab"}, "switch between Files and the note", groupMove, inMain},
	{actPrevPane, []string{"shift+tab"}, "switch between Files and the note", groupMove, inMain},
	{actPane1, []string{"1"}, "Focus on the Files pane", groupMove, inMain},
	{actPane2, []string{"2"}, "Focus on the note pane", groupMove, inMain},
	{actPaneLeft, []string{"shift+left"}, "note, split: to the pane on the left", groupMove, inMain},
	{actPaneRight, []string{"shift+right"}, "note, split: to the pane on the right · Files: focus the split", groupMove, inMain},
	{actBack, []string{"ctrl+o", "alt+left"}, "Files: close the folder · note: go back", groupMove, inMain},
	{actForward, []string{"ctrl+i", "alt+right"}, "Files: open the folder · note: go forward", groupMove, inMain},
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
	{actHabits, []string{"T"}, "habits: today's list, and the week/month grids", groupFiles, inMain},
	{actNewBook, []string{"B"}, "Book Card: catalogue a book, or edit the open book note", groupFiles, inMain},
	{actQuickNote, []string{"i"}, "quick note: capture into a folder, first line names it", groupFiles, inMain},
	{actOrderUp, []string{"shift+up"}, "move the item under the cursor up its level", groupFiles, inMain},
	{actOrderDown, []string{"shift+down"}, "move it down its level", groupFiles, inMain},
	{actOrderReset, []string{"R"}, "put this level back in the default order", groupFiles, inMain},
	{actSkimDown, []string{"alt+down", "alt+j"}, "move down, opening the note beside the open one", groupMove, inMain},
	{actSkimUp, []string{"alt+up", "alt+k"}, "move up, opening the note beside the open one", groupMove, inMain},
	{actEdit, []string{"e"}, "edit the note in Skrin", groupNote, inMain},
	{actEditExternal, []string{"E"}, "edit the note in $EDITOR", groupNote, inMain},
	{actUndoEdit, []string{"u"}, "undo the note's last edit", groupNote, inMain},
	{actRedoEdit, []string{"ctrl+r"}, "redo it", groupNote, inMain},
	{actHints, []string{"f"}, "follow a link: letters appear on each", groupNote, inMain},
	{actLineNumbers, []string{"L"}, "toggle line numbers in notes", groupNote, inMain},
	{actBacklinks, []string{"b"}, "notes linking here", groupNote, inMain},
	{actOutline, []string{"o"}, "outline: jump to a heading", groupNote, inMain},
	{actNextHeading, []string{"}"}, "next heading", groupNote, inMain},
	{actPrevHeading, []string{"{"}, "previous heading", groupNote, inMain},
	{actSearch, []string{"/"}, "search (Alt-r in there: search & replace)", groupSearch, inMain},
	{actSwitcher, []string{"g"}, "go to a note by name (Alt+←/→ there: split)", groupSearch, inMain},
	{actClaude, []string{"c"}, "open or hide the Claude drawer", groupClaude, inMain},
	{actClaudeInput, []string{"C"}, "type to Claude, with the highlighted text if any", groupClaude, inMain},
	{actZen, []string{"z"}, "zen mode: just the note, centred", groupSkrin, inMain},
	{actPalette, []string{"ctrl+p", ":"}, "commands: find any command by what it does, and run it", groupSkrin, inMain},
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
	{actPalette, []string{"ctrl+p"}, "commands: find any command by what it does, and run it", groupEditor, inEditor},
	{actZen, []string{"alt+z"}, "zen mode, still editing", groupEditor, inEditor},
	{actBacklinks, []string{"alt+b"}, "notes linking here", groupEditor, inEditor},
	{actOutline, []string{"alt+o"}, "outline: jump to a heading", groupEditor, inEditor},
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
	{actSplitLeft, []string{"alt+left"}, "Go to note: open the note in a split, on the left", groupList, inList},
	{actSplitRight, []string{"alt+right"}, "Go to note: open the note in a split, on the right", groupList, inList},
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

	{actUp, []string{"up", "ctrl+p"}, "previous field (line up inside Notes)", groupBookCard, inBookCard},
	{actDown, []string{"down", "ctrl+n"}, "next field (line down inside Notes)", groupBookCard, inBookCard},
	{actNextField, []string{"tab"}, "next field", groupBookCard, inBookCard},
	{actPrevField, []string{"shift+tab"}, "previous field", groupBookCard, inBookCard},
	{actPick, []string{"enter"}, "search bar: fetch · a field: next · Save: write the note", groupBookCard, inBookCard},
	{actFetchBook, []string{"alt+f"}, "fetch metadata & cover from the search bar", groupBookCard, inBookCard},
	{actAddQuote, []string{"alt+q"}, "add another quote", groupBookCard, inBookCard},
	{actSaveBook, []string{"ctrl+s"}, "save the book note and download its cover", groupBookCard, inBookCard},
	{actCancel, []string{"esc"}, "step back: close a result list, then the card", groupBookCard, inBookCard},

	{actNone, []string{"tab", "shift+tab"}, "Keys → Settings → Guide, and back", groupManual, inManual},
	{actNone, []string{"j", "k"}, "move between keys", groupManual, inManual},
	{actNone, []string{"enter"}, "give the key under the cursor a new key", groupManual, inManual},
	{actNone, []string{"r"}, "put that one key back to Skrin's own", groupManual, inManual},
	{actNone, []string{"R"}, "put every key back to Skrin's own", groupManual, inManual},
	{actNone, []string{"/"}, "find a key by what it does, or by the key itself", groupManual, inManual},
	{actNone, []string{"esc", "?", "q"}, "clear the filter, then close", groupManual, inManual},
	{actNone, []string{"j", "k"}, "move between settings", groupSettings, inSettings},
	{actNone, []string{"enter", "space"}, "toggle the setting under the cursor", groupSettings, inSettings},
	{actNone, []string{"tab", "shift+tab"}, "Keys → Settings → Guide, and back", groupSettings, inSettings},
	{actNone, []string{"esc", "q"}, "close", groupSettings, inSettings},
	{actNone, []string{"j", "k", "ctrl+d", "ctrl+u", "space"}, "scroll", groupGuide, inGuide},
	{actNone, []string{"home", "g", "G", "end"}, "top / bottom", groupGuide, inGuide},
	{actNone, []string{"/"}, "filter the guide", groupGuide, inGuide},
	{actNone, []string{"tab", "shift+tab"}, "Keys → Settings → Guide, and back", groupGuide, inGuide},
	{actNone, []string{"esc", "?", "q"}, "clear the filter, then close", groupGuide, inGuide},

	{actUp, []string{"k", "up"}, "up a habit (grid: up a row)", groupHabits, inHabits},
	{actDown, []string{"j", "down"}, "down a habit (grid: down a row)", groupHabits, inHabits},
	{actLeft, []string{"h", "left"}, "grid: left a day · today: nothing to the left", groupHabits, inHabits},
	{actRight, []string{"l", "right"}, "grid: right a day · today: nothing to the right", groupHabits, inHabits},
	{actMark, []string{"space"}, "tick / untick the habit under the cursor", groupHabits, inHabits},
	{actHabitTab, []string{"H"}, "today → this week → this month → today", groupHabits, inHabits},
	{actUndoOp, []string{"U"}, "undo the last tick you made from here", groupHabits, inHabits},
	{actCancel, []string{"esc", "ctrl+c"}, "close the habits view", groupHabits, inHabits},

	{actPick, []string{"enter"}, "text: save · folder row: pick the highlighted match", groupQuickNote, inQuickNote},
	{actNewLine, []string{"shift+enter", "alt+enter"}, "a new line", groupQuickNote, inQuickNote},
	{actNextField, []string{"tab"}, "text ↔ folder", groupQuickNote, inQuickNote},
	{actPrevField, []string{"shift+tab"}, "text ↔ folder", groupQuickNote, inQuickNote},
	{actUp, []string{"up"}, "folder row: through the matches · text: up a line", groupQuickNote, inQuickNote},
	{actDown, []string{"down"}, "folder row: through the matches · text: down a line", groupQuickNote, inQuickNote},
	{actCancel, []string{"esc", "ctrl+c"}, "close, write nothing", groupQuickNote, inQuickNote},
}

// actionName is the name each action goes by in config.toml's [keys]
// tables, in collision messages and in the Keys tab. A user's overrides
// are stored against these names, so they are part of the config format:
// add to this table, never rename in it. TestEveryActionHasAName keeps it
// complete.
var actionName = map[action]string{
	actUp: "up", actDown: "down", actTop: "top", actBottom: "bottom",
	actHalfDown: "half-down", actHalfUp: "half-up",
	actLeft: "left", actRight: "right",
	actNextPane: "next-pane", actPrevPane: "prev-pane",
	actPane1: "pane-files", actPane2: "pane-note",
	actPaneLeft: "pane-left", actPaneRight: "pane-right",
	actOpen: "open", actParent: "parent", actCollapseAll: "collapse-all",
	actNewNote: "new-note", actNewFolder: "new-folder",
	actRename: "rename", actMove: "move", actDelete: "delete",
	actMark: "mark", actVisual: "visual", actMarkAll: "mark-all",
	actEscape: "escape", actUndoOp: "undo-op", actDaily: "daily",
	actEdit: "edit", actEditExternal: "edit-external",
	actUndoEdit: "undo-edit", actRedoEdit: "redo-edit",
	actHints: "hints", actLineNumbers: "line-numbers", actBacklinks: "backlinks", actOutline: "outline",
	actNextHeading: "next-heading", actPrevHeading: "prev-heading",
	actBack: "back", actForward: "forward",
	actSearch: "search", actSwitcher: "switcher",
	actClaude: "claude", actClaudeInput: "claude-input",
	actZen: "zen", actHelp: "help", actQuit: "quit",
	actAskClaude: "ask-claude", actPick: "pick", actCancel: "cancel",
	actSplitLeft: "split-left", actSplitRight: "split-right",
	actSend: "send", actNewLine: "new-line", actLeaveDrawer: "leave-drawer",
	actFlipDrawer: "flip-drawer", actNewChat: "new-chat",
	actScrollBack: "scroll-back", actScrollOn: "scroll-on",
	actApply: "apply", actReject: "reject",
	actOrderUp: "order-up", actOrderDown: "order-down", actOrderReset: "order-reset",
	actSkimDown: "skim-down", actSkimUp: "skim-up",
	actFindScope: "find-scope", actFindCase: "find-case",
	actFindWords: "find-words", actFindReplace: "find-replace",
	actNextField: "next-field", actPrevField: "prev-field",
	actReplaceAll: "replace-all", actSkipMatch: "skip-match",
	actNewBook: "new-book", actFetchBook: "fetch-book",
	actAddQuote: "add-quote", actSaveBook: "save-book",
	actFolderJumpUp: "folder-jump-up", actFolderJumpDown: "folder-jump-down",
	actHabits: "habits", actHabitTab: "habit-tab", actQuickNote: "quick-note",
	actPalette: "palette",
}

// actionByName is actionName the other way round, for reading overrides
// back out of config.toml.
var actionByName = func() map[string]action {
	out := map[string]action{}
	for a, n := range actionName {
		out[n] = a
	}
	return out
}()

// keymap is the keymap in force: the registry's defaults with whatever
// the user changed in config.toml's [keys] laid over them. The Keys tab
// of `?` edits it, and every key Skrin handles is looked up through it.
type keymap struct {
	// over is what the user changed: context → action name → its keys.
	// It is exactly what goes in and out of config.toml, so an override
	// survives Skrin learning new default keys for everything else.
	over map[string]map[string][]string

	lookup map[string]map[string]action   // context → key → action
	keys   map[string]map[action][]string // context → action → its keys
	// taken names what holds each key in a context, rebindable or not,
	// so a clash with a component's own key (the editor's vim keys, a
	// list's typing) can be refused by name rather than silently lost.
	taken map[string]map[string]string
}

// newKeymap builds the keymap in force from the user's overrides, which
// may be nil for the defaults alone.
func newKeymap(over map[string]map[string][]string) *keymap {
	km := &keymap{over: map[string]map[string][]string{}}
	for where, acts := range over {
		for name, keys := range acts {
			if _, ok := actionByName[name]; !ok {
				continue // an action this Skrin no longer has
			}
			// An empty list is kept, not skipped: it means a binding
			// deliberately left with no key, usually because its key was
			// given to something else. Treating it as "no override" would
			// hand the default back and leave two bindings claiming the
			// same key after a restart.
			km.setOver(where, name, keys)
		}
	}
	km.rebuild()
	return km
}

func (km *keymap) setOver(where, name string, keys []string) {
	if km.over[where] == nil {
		km.over[where] = map[string][]string{}
	}
	km.over[where][name] = keys
}

// rebuild works the lookups out again from the defaults and the
// overrides. It runs on every change: the registry is small enough that
// there is nothing to gain from being cleverer, and everything to lose
// from the two halves drifting apart.
func (km *keymap) rebuild() {
	km.lookup = map[string]map[string]action{}
	km.keys = map[string]map[action][]string{}
	km.taken = map[string]map[string]string{}
	for _, b := range defaultBindings {
		keys := b.keys
		if b.act != actNone {
			if over, ok := km.over[b.where][actionName[b.act]]; ok {
				keys = over
			}
		}
		if km.taken[b.where] == nil {
			km.taken[b.where] = map[string]string{}
			km.lookup[b.where] = map[string]action{}
			km.keys[b.where] = map[action][]string{}
		}
		for _, k := range keys {
			km.taken[b.where][k] = b.help
		}
		if b.act == actNone {
			continue
		}
		km.keys[b.where][b.act] = keys
		for _, k := range keys {
			km.lookup[b.where][k] = b.act
		}
	}
}

// act is the action key stands for in context where, or actNone.
func (km *keymap) act(where, key string) action { return km.lookup[where][key] }

// bound is the keys that work for action a in context where.
func (km *keymap) bound(where string, a action) []string { return km.keys[where][a] }

// changed reports whether the user gave this binding its own keys.
func (km *keymap) changed(where string, a action) bool {
	_, ok := km.over[where][actionName[a]]
	return ok
}

// defaultKeys is what the registry ships for this binding, whatever the
// user has since made of it.
func defaultKeys(where string, a action) []string {
	for _, b := range defaultBindings {
		if b.where == where && b.act == a {
			return b.keys
		}
	}
	return nil
}

// holder names what already has key k in context where, and whether that
// can be rebound. An empty name means the key is free.
func (km *keymap) holder(where, key string) (help string, rebindable bool) {
	help, ok := km.taken[where][key]
	if !ok {
		return "", false
	}
	return help, km.lookup[where][key] != actNone
}

// set gives action a the single key k in context where, dropping k from
// whatever else had it there. It returns the action k was taken from, if
// any, so the caller can offer to give that one a new key on the spot.
func (km *keymap) set(where string, a action, k string) (displaced action) {
	if prev := km.lookup[where][k]; prev != actNone && prev != a {
		rest := []string{}
		for _, old := range km.bound(where, prev) {
			if old != k {
				rest = append(rest, old)
			}
		}
		km.setOver(where, actionName[prev], rest)
		displaced = prev
	}
	km.setOver(where, actionName[a], []string{k})
	km.rebuild()
	return displaced
}

// reset puts one binding back to the keys the registry ships.
func (km *keymap) reset(where string, a action) {
	delete(km.over[where], actionName[a])
	if len(km.over[where]) == 0 {
		delete(km.over, where)
	}
	km.rebuild()
}

// resetAll drops every override: the whole keymap as it ships.
func (km *keymap) resetAll() {
	km.over = map[string]map[string][]string{}
	km.rebuild()
}

// overrides is the user's changes, for config.toml. It is nil when
// nothing was changed, so an untouched config keeps no [keys] table at
// all.
func (km *keymap) overrides() map[string]map[string][]string {
	if len(km.over) == 0 {
		return nil
	}
	return km.over
}

// actionIn is the action key stands for in context where, under the
// keymap in force for this Skrin: the defaults, plus whatever the user
// changed in the Keys tab of `?`.
func (m *Model) actionIn(where, key string) action { return m.keys.act(where, key) }
