package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/logo"
	"github.com/lurioso/skrin/internal/vault"
)

// A keyboard app asks you to recall keys; someone new can only recognise
// things. Everything in this file closes that gap from the screen you're
// already looking at: the status line names the keys that matter where
// you are, the welcome screen hands over the first few, and a key that
// does nothing says so — each of them pointing at the palette, where
// anything can be found by what it does.

// statusHint is one "key what" pair on the status line's right side.
type statusHint struct {
	key, what string
	prio      int // lower is kept longer when the line runs out of room
}

// hintLine is the status line's right side in at most w cells: the keys
// worth knowing where you are, with the palette always among them. Hints
// that don't fit are dropped whole, least useful first, never cut in half.
func (m *Model) hintLine(w int) string {
	hs := m.hintsHere()
	keep := make([]bool, len(hs))
	order := make([]int, len(hs))
	for i := range order {
		order[i] = i
	}
	// A stable pick by priority; the line keeps its reading order.
	for i := 1; i < len(order); i++ {
		for j := i; j > 0 && hs[order[j]].prio < hs[order[j-1]].prio; j-- {
			order[j], order[j-1] = order[j-1], order[j]
		}
	}
	used := 0
	for _, i := range order {
		cost := ansi.StringWidth(hs[i].key + " " + hs[i].what)
		if used > 0 {
			cost += 3 // " · "
		}
		if used+cost > w {
			continue
		}
		keep[i], used = true, used+cost
	}
	var out []string
	for i, h := range hs {
		if keep[i] {
			out = append(out, h.key+" "+h.what)
		}
	}
	return strings.Join(out, " · ")
}

// hintsHere are the keys worth knowing in the situation you're in, from
// the keymap in force: a rebound key shows as rebound, an unbound one
// not at all.
func (m *Model) hintsHere() []statusHint {
	var hs []statusHint
	add := func(k, what string, prio int) {
		if k != "" {
			hs = append(hs, statusHint{k, what, prio})
		}
	}
	act := func(a action, what string, prio int) { add(m.keyFor(inMain, a), what, prio) }
	claude := m.opts.Assistant.Enabled

	switch {
	case m.noteSel != nil:
		add("ctrl+c", "copy", 1)
		if claude {
			act(actClaudeInput, "ask Claude", 2)
		}
		act(actEscape, "clear", 3)
	case len(m.marks) > 0 || m.visual != nil:
		act(actMove, "move", 1)
		act(actDelete, "delete", 2)
		act(actMark, "mark", 4)
		act(actEscape, "clear marks", 3)
	case m.split != nil && m.focus == paneNote:
		add(m.paneKeys(), "switch panes", 1)
		act(actEdit, "edit", 2)
		act(actEscape, "close this pane", 3)
	case m.focus == paneFiles:
		e := m.files.selected()
		switch {
		case e.IsDir:
			act(actRight, "open", 1)
			act(actNewNote, "new note", 3)
		case vault.IsNote(e.Name) && m.opts.InstantOpen:
			act(actOpen, "edit", 1)
			act(actRight, "read", 2)
		case vault.IsNote(e.Name):
			act(actRight, "open", 1)
			act(actOpen, "edit", 2)
		default:
			act(actOpen, "open", 1)
		}
		if m.split != nil {
			add(m.paneKeys(), "switch panes", 2)
		}
		act(actSwitcher, "go to note", 4)
		act(actSearch, "search", 5)
		act(actDaily, "today", 6)
	case m.notePath != "":
		act(actEdit, "edit", 1)
		act(actHints, "follow link", 2)
		act(actBacklinks, "backlinks", 4)
		act(actOutline, "outline", 5)
		act(actSearch, "search", 6)
	default:
		act(actSwitcher, "go to note", 1)
		act(actDaily, "today", 2)
		act(actSearch, "search", 3)
	}
	// The palette is the door to everything else, so it is always there;
	// the manual and quitting come last, and go first when room runs out.
	add(m.keyFor(inMain, actPalette), "commands", 0)
	act(actHelp, "keys", 7)
	act(actQuit, "quit", 8)
	return hs
}

// paneKeys is the pair of keys that move between a split's panes, written
// as one: "shift+←/→".
func (m *Model) paneKeys() string {
	l, r := m.keyFor(inMain, actPaneLeft), m.keyFor(inMain, actPaneRight)
	switch {
	case l == "" || r == "":
		return l + r
	case strings.HasSuffix(l, "←") && strings.HasSuffix(r, "→") && strings.TrimSuffix(l, "←") == strings.TrimSuffix(r, "→"):
		return l + "/→"
	}
	return l + "/" + r
}

// welcomeRow is one line of the welcome card: a key and what it does.
type welcomeRow struct{ key, what string }

// welcomeRows are the first keys a newcomer needs, from the keymap in
// force. They tell the truth about instant-open either way.
func (m *Model) welcomeRows() []welcomeRow {
	var rows []welcomeRow
	add := func(a action, what string) {
		if k := m.keyFor(inMain, a); k != "" {
			rows = append(rows, welcomeRow{k, what})
		}
	}
	if d, u := m.keyFor(inMain, actDown), m.keyFor(inMain, actUp); d != "" && u != "" {
		what := "move through Files"
		if m.opts.InstantOpen {
			what = "move through Files — notes open as you land"
		}
		rows = append(rows, welcomeRow{d + "/" + u, what})
	}
	if m.opts.InstantOpen {
		add(actRight, "step into the note to read it")
	} else {
		add(actRight, "open the note under the cursor")
	}
	add(actOpen, "edit it")
	add(actDaily, "today's daily note")
	add(actSwitcher, "go to a note by name")
	add(actSearch, "search the vault")
	add(actPalette, "every command, by name")
	add(actHelp, "keys, settings and the guide")
	return rows
}

// tip is a tip of the day: a key worth finding, and what it does.
type tip struct {
	act  action
	what string
}

// tips reach past the first few keys to the ones that make Skrin quick,
// one a day on the welcome screen.
var tips = []tip{
	{actHints, "follows a link: straight there if there's one, a letter on each if there are more"},
	{actSkimDown, "moves down and opens that note beside the one you're reading"},
	{actBacklinks, "lists every note that links to this one"},
	{actOutline, "jumps to any heading in the note"},
	{actQuickNote, "captures a quick note without leaving what you're reading"},
	{actZen, "is zen mode: just the note, centred"},
	{actUndoEdit, "undoes a note's last edit — made here, in $EDITOR or by Claude"},
	{actUndoOp, "undoes the last create, rename, move or delete"},
	{actOrderDown, "moves the item under the cursor down, into your own order"},
	{actMark, "marks files; m and d then move or delete them all at once"},
	{actHabits, "opens the habits view: today's list, the week and the month"},
	{actClaudeInput, "asks Claude, with the text you've highlighted"},
	{actSearch, "searches the vault; Alt-r in there makes it search & replace"},
}

// tipOfTheDay is today's tip, keyed to the date so it holds still for the
// day and moves on the next. ok is false if no tip has a key.
func (m *Model) tipOfTheDay() (k, what string, ok bool) {
	var live []tip
	for _, t := range tips {
		if t.act == actClaudeInput && !m.opts.Assistant.Enabled {
			continue
		}
		if m.keyFor(inMain, t.act) != "" {
			live = append(live, t)
		}
	}
	if len(live) == 0 {
		return "", "", false
	}
	t := live[m.opts.Now().YearDay()%len(live)]
	return m.keyFor(inMain, t.act), t.what, true
}

// splashBody fills the note pane when no note is open: the chest, Skrin's
// name, the first keys and a tip, centred. The chest shrinks, then goes,
// when the pane is too small for it.
func (m *Model) splashBody(w, vis int) []string {
	rows := m.welcomeRows()
	kw := 0
	for _, r := range rows {
		kw = max(kw, ansi.StringWidth(r.key))
	}
	var block []string
	bw := 0
	for _, r := range rows {
		l := m.st.flash.Render(r.key+strings.Repeat(" ", kw-ansi.StringWidth(r.key))) + "  " + m.st.text.Render(r.what)
		block = append(block, l)
		bw = max(bw, ansi.StringWidth(l))
	}
	indent := strings.Repeat(" ", max((w-bw)/2, 0))
	text := []string{center(m.st.brand.Render("Skrin"), w), ""}
	for _, l := range block {
		text = append(text, indent+l)
	}
	if k, what, ok := m.tipOfTheDay(); ok {
		text = append(text, "", center(m.st.muted.Render("Tip · ")+m.st.flash.Render(k)+" "+m.st.muted.Render(what), w))
	}
	var lines []string
	for scale := 2; scale >= 1; scale-- {
		if lw, lh := logo.SplashSize(scale); lw <= w && lh+1+len(text) <= vis {
			for _, l := range m.splash[scale-1] {
				lines = append(lines, center(l, w))
			}
			lines = append(lines, "")
			break
		}
	}
	lines = append(lines, text...)
	out := make([]string, max((vis-len(lines))/3, 0), vis)
	return append(out, lines...)
}

func center(s string, w int) string {
	return strings.Repeat(" ", max((w-ansi.StringWidth(s))/2, 0)) + s
}

// strayKey reports whether key is one someone could press meaning
// something by it: a character, or a Ctrl or Alt chord. Those get an
// answer when they do nothing, instead of the silence the charter rules
// out.
func strayKey(key string) bool {
	if strings.HasPrefix(key, "ctrl+") || strings.HasPrefix(key, "alt+") {
		return true
	}
	return utf8.RuneCountInString(key) == 1
}

// strayNote answers a key that does nothing in Files or the note, pointing
// at where the right one can be found.
func (m *Model) strayNote(key string) string {
	if k := m.keyFor(inMain, actPalette); k != "" {
		return statusKey(key) + " does nothing here · " + k + " finds every command"
	}
	if k := m.keyFor(inMain, actHelp); k != "" {
		return statusKey(key) + " does nothing here · " + k + " lists every key"
	}
	return statusKey(key) + " does nothing here"
}

// ctrlCNote answers the first Ctrl+C: nothing is selected to copy, and a
// second one quits.
func (m *Model) ctrlCNote() string {
	for _, k := range m.keys.bound(inMain, actQuit) {
		if k != "ctrl+c" {
			return "Nothing selected to copy · ctrl+c again quits, as does " + statusKey(k)
		}
	}
	return "Nothing selected to copy · ctrl+c again quits"
}
