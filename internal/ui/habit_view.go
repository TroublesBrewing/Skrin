package ui

import (
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/daily"
	"github.com/lurioso/skrin/internal/habit"
	"github.com/lurioso/skrin/internal/obsidian"
	"github.com/lurioso/skrin/internal/vault"
)

// habitTab is which view the habits overlay shows.
type habitTab int

const (
	habitsToday habitTab = iota
	habitsWeek
	habitsMonth
)

// habitView is the T overlay: today's list, or the week/month grid. All
// data is the notes themselves — read at open and after every write, never
// stored anywhere.
type habitView struct {
	tab habitTab
	cur int // row: today's list, or the grid's
	col int // grid only: the day column
}

// habitDay is one column of the grid, assembled at open.
type habitDay struct {
	rel  string // "" when the day has no note
	name string // how the column header and flashes spell the day
}

// openHabits opens the habits overlay on today. The empty-template case
// points at the template instead of showing a blank box (catch 4).
func (m *Model) openHabits() {
	s := obsidian.LoadSettings(m.vault.Root)
	rel := daily.Path(s.Daily, m.opts.Now())
	src, err := m.vault.Read(rel)
	haveBlock := false
	if err == nil {
		_, haveBlock = habit.Parse(src)
	}
	if err != nil || !haveBlock {
		tmpl, terr := m.dailyTemplate(s)
		if terr != nil || !hasTemplateHabits(tmpl) {
			m.flash = "No habits in today's note — add them under " + habit.Heading + " in the template"
			return
		}
		// The template has a starter list: offer the one-time insert
		// rather than doing it unasked.
		m.confirm = &confirm{
			pill:     " HABITS ",
			question: "Add " + habit.Heading + " to today's note, from the template?",
			keys:     "y / n",
			yes: func() {
				m.insertHabitsBlock(rel, tmpl)
			},
			cancel: "Left today's note as it was",
		}
		return
	}
	m.habits = &habitView{tab: habitsToday, cur: firstUnticked(src)}
}

// dailyTemplate reads the daily-note template, expanded for today.
func (m *Model) dailyTemplate(s obsidian.Settings) (string, error) {
	tp := daily.TemplatePath(s.Daily)
	if tp == "" {
		return "", nil
	}
	src, err := m.vault.Read(tp)
	if err != nil {
		return "", err
	}
	return daily.Expand(src, s.Daily.Format, m.opts.Now(), m.opts.Now()), nil
}

// hasTemplateHabits reports whether the template carries a starter list.
func hasTemplateHabits(tmpl string) bool {
	b, ok := habit.Parse(tmpl)
	return ok && len(b.Items) > 0
}

// insertHabitsBlock handles the confirm's y: today's note gets its habits
// block, then the overlay opens on it. Two shapes: a note that doesn't
// exist is created from the template — which carries the block, that's why
// the confirm appeared — while a note that exists without one gets the
// template's block spliced in as one journal step.
func (m *Model) insertHabitsBlock(rel, tmpl string) {
	src, err := m.vault.Read(rel)
	if err != nil {
		// Today's note doesn't exist yet: create it from the
		// template, the same way t does. The block comes along.
		s := obsidian.LoadSettings(m.vault.Root)
		now := m.opts.Now()
		content := ""
		if tp := daily.TemplatePath(s.Daily); tp != "" {
			if tsrc, terr := m.vault.Read(tp); terr == nil {
				content = daily.Expand(tsrc, s.Daily.Format, now, now)
			}
		}
		dirs, cerr := m.vault.CreateFile(rel, content)
		if cerr != nil {
			m.flash = "Couldn't create today's note: " + cerr.Error()
			return
		}
		m.journal.Record(vault.Op{Desc: "create " + rel, Steps: append(createdSteps(dirs), vault.Step{Kind: vault.StepCreated, Rel: rel, Content: content})})
		m.refresh()
		m.habits = &habitView{tab: habitsToday, cur: 0}
		m.flash = "Created " + rel + " · U undoes"
		return
	}
	out, ok := habit.Insert(src, tmpl)
	if !ok {
		m.flash = "No habits in the template to add"
		return
	}
	if err := m.vault.Write(rel, out); err != nil {
		m.flash = "Couldn't add the habits block: " + err.Error()
		return
	}
	m.journal.Record(vault.Op{Desc: "add habits to " + rel, Steps: []vault.Step{{Kind: vault.StepModified, Rel: rel, Content: src}}})
	m.refresh()
	m.habits = &habitView{tab: habitsToday, cur: 0}
	m.flash = "Added " + habit.Heading + " to " + rel + " · U undoes"
}

// firstUnticked is where the cursor starts: the first habit not yet done
// today (0 when all are done).
func firstUnticked(src string) int {
	b, _ := habit.Parse(src)
	for i, it := range b.Items {
		if !it.Done {
			return i
		}
	}
	return 0
}

// todayRel is today's daily-note path per Obsidian's settings.
func (m *Model) todayRel() string {
	s := obsidian.LoadSettings(m.vault.Root)
	return daily.Path(s.Daily, m.opts.Now())
}

// toggleHabit flips the habit under the overlay's cursor — today's list,
// or the grid cell — writing the day's note through the same path every
// write takes: snapshot, write, journal. One tick, one U.
func (m *Model) toggleHabit() {
	if m.habits == nil {
		return
	}
	s := obsidian.LoadSettings(m.vault.Root)
	today := daily.Path(s.Daily, m.opts.Now())
	src, err := m.vault.Read(today)
	if err != nil {
		m.flash = "Couldn't read today's note: " + err.Error()
		return
	}
	todayBlock, ok := habit.Parse(src)
	if !ok || m.habits.cur >= len(todayBlock.Items) {
		m.flash = "No habit under the cursor"
		return
	}
	name := todayBlock.Items[m.habits.cur].Text
	if m.habits.tab == habitsToday {
		out, ok := habit.Toggle(src, m.habits.cur)
		if !ok {
			m.flash = "No habit under the cursor"
			return
		}
		m.writeHabitTick(today, src, out, name, "", !todayBlock.Items[m.habits.cur].Done)
		return
	}
	// Grid: the row names today's habit; the column names the day whose
	// note gets the write.
	grid := m.habitGrid(s, m.habits.tab)
	if m.habits.col >= len(grid.Days) {
		return
	}
	day := grid.Days[m.habits.col]
	if day.Path == "" {
		m.flash = "No note for " + day.Name + " — the day is unrecorded"
		return
	}
	daySrc, err := m.vault.Read(day.Path)
	if err != nil {
		m.flash = "Couldn't read " + day.Path + ": " + err.Error()
		return
	}
	dayBlock, ok := habit.Parse(daySrc)
	if !ok {
		m.flash = day.Path + " has no habits block"
		return
	}
	dayIdx := -1
	for i, it := range dayBlock.Items {
		if it.Text == name {
			dayIdx = i
			break
		}
	}
	if dayIdx < 0 {
		m.flash = name + " isn't in " + day.Path
		return
	}
	out, ok := habit.Toggle(daySrc, dayIdx)
	if !ok {
		m.flash = "Couldn't tick " + name + " in " + day.Path
		return
	}
	m.writeHabitTick(day.Path, daySrc, out, name, day.Name, !dayBlock.Items[dayIdx].Done)
}

// writeHabitTick performs the write for one tick or untick: snapshot
// first, then write, then one journal step — the same order every Skrin
// write takes. The flash says which way the box moved.
func (m *Model) writeHabitTick(rel, before, after, name, day string, ticked bool) {
	if err := m.snaps.Save(rel, before); err != nil {
		m.flash = "Left " + name + " as it was: couldn't keep a snapshot first (" + err.Error() + ")"
		return
	}
	if err := m.vault.Write(rel, after); err != nil {
		m.flash = "Couldn't write " + rel + ": " + err.Error()
		return
	}
	m.journal.Record(vault.Op{Desc: "tick " + name + " in " + rel, Steps: []vault.Step{{Kind: vault.StepModified, Rel: rel, Content: before}}})
	m.refresh()
	verb := "Unticked"
	if ticked {
		verb = "Ticked"
	}
	if day != "" {
		m.flash = verb + " " + name + " · " + day + " · U undoes"
	} else {
		m.flash = verb + " " + name + " · U undoes"
	}
}

// habitDays assembles the day columns for a tab: the week runs Monday to
// Sunday around today; the month runs the 1st up to and including today.
func habitDays(m *Model, s obsidian.Settings, tab habitTab) []habitDay {
	now := m.opts.Now()
	n := 7
	start := now
	if tab == habitsWeek {
		start = now.AddDate(0, 0, -int(now.Weekday())+int(time.Monday))
		if now.Weekday() == time.Sunday {
			start = now.AddDate(0, 0, -6)
		}
	}
	if tab == habitsMonth {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		n = now.Day()
	}
	var days []habitDay
	for i := 0; i < n; i++ {
		d := start.AddDate(0, 0, i)
		rel := daily.Path(s.Daily, d)
		name := d.Format("Mon 01-02")
		if !m.vault.Exists(rel) {
			rel = ""
		}
		days = append(days, habitDay{rel: rel, name: name})
	}
	return days
}

// habitGrid builds the whole grid for a tab: today's habit names as the
// rows, the tab's days as the columns, each cell straight from its note.
func (m *Model) habitGrid(s obsidian.Settings, tab habitTab) habit.Grid {
	src, err := m.vault.Read(daily.Path(s.Daily, m.opts.Now()))
	if err != nil {
		return habit.Grid{}
	}
	todayBlock, _ := habit.Parse(src)
	g := habit.Grid{Done: map[string]map[string]bool{}}
	for _, it := range todayBlock.Items {
		g.Names = append(g.Names, it.Text)
	}
	for _, d := range habitDays(m, s, tab) {
		day := habit.Day{Path: d.rel, Name: d.name}
		if d.rel != "" {
			if src, err := m.vault.Read(d.rel); err == nil {
				if b, ok := habit.Parse(src); ok {
					day.Items = b.Items
					g.Done[d.rel] = map[string]bool{}
					for _, it := range b.Items {
						g.Done[d.rel][it.Text] = it.Done
					}
				}
			}
		}
		g.Days = append(g.Days, day)
	}
	return g
}

// habitKey handles a key in the overlay, per the registry's inHabits rows.
func (m *Model) habitKey(k tea.KeyPressMsg) {
	h := m.habits
	if h == nil {
		return
	}
	switch a := m.actionIn(inHabits, k.String()); a {
	case actCancel:
		m.habits = nil
	case actUp:
		if h.cur > 0 {
			h.cur--
		}
	case actDown:
		if h.tab == habitsToday {
			if b, ok := habit.Parse(m.todayNote()); ok && h.cur < len(b.Items)-1 {
				h.cur++
			}
		} else if h.cur < len(m.habitGrid(obsidian.LoadSettings(m.vault.Root), h.tab).Names)-1 {
			h.cur++
		}
	case actLeft:
		if h.tab != habitsToday && h.col > 0 {
			h.col--
		}
	case actRight:
		if h.tab != habitsToday {
			g := m.habitGrid(obsidian.LoadSettings(m.vault.Root), h.tab)
			if h.col < len(g.Days)-1 {
				h.col++
			}
		}
	case actMark:
		m.toggleHabit()
	case actHabitTab:
		h.tab = (h.tab + 1) % 3
		h.col = 0
		// The row cursor stays; the grid's rows are today's names.
		if h.tab == habitsToday {
			h.cur = min(h.cur, max(len(m.habitGrid(obsidian.LoadSettings(m.vault.Root), h.tab).Names)-1, 0))
		}
	case actUndoOp:
		m.undoOp()
	default:
		// actNone and anything unmapped: ignore, the overlay takes
		// all keys while open.
	}
}

// todayNote is today's note text ("" when unread or absent).
func (m *Model) todayNote() string {
	src, err := m.vault.Read(m.todayRel())
	if err != nil {
		return ""
	}
	return src
}

// habitsBox renders the overlay.
func (m *Model) habitsBox() []string {
	h := m.habits
	if h == nil {
		return nil
	}
	s := obsidian.LoadSettings(m.vault.Root)
	w := min(max(m.width-6, 50), 80)
	inner := w - 4
	var body []string
	switch h.tab {
	case habitsToday:
		body = m.habitsTodayBox(s, inner)
	default:
		body = m.habitsGridBox(s, inner, h.tab)
	}
	title := " Habits "
	switch h.tab {
	case habitsWeek:
		title = " Habits — this week "
	case habitsMonth:
		title = " Habits — this month "
	}
	return m.box(title, body, w, len(body)+2, true)
}

func (m *Model) habitsTodayBox(s obsidian.Settings, inner int) []string {
	src := m.todayNote()
	b, ok := habit.Parse(src)
	if !ok {
		return []string{m.st.muted.Render("  No habits in today's note.")}
	}
	var body []string
	for i, it := range b.Items {
		mark := "  "
		if it.Done {
			mark = "  ✓"
		}
		line := mark + " " + it.Text
		if i == m.habits.cur {
			body = append(body, "  "+m.st.selFocus.Render(fit(line, inner-2)))
		} else if it.Done {
			body = append(body, "  "+m.st.muted.Render(fit(line, inner-2)))
		} else {
			body = append(body, "  "+fit(line, inner-2))
		}
	}
	body = append(body, "", " "+m.st.muted.Render("H week/month · space tick · U undo · esc close"))
	return body
}

func (m *Model) habitsGridBox(s obsidian.Settings, inner int, tab habitTab) []string {
	g := m.habitGrid(s, tab)
	if len(g.Names) == 0 {
		return []string{m.st.muted.Render("  No habits in today's note.")}
	}
	// The name column is as wide as the longest name, capped.
	nameW := 0
	for _, n := range g.Names {
		if w := ansi.StringWidth(n); w > nameW {
			nameW = w
		}
	}
	nameW = min(nameW+2, max(inner/2, 8))

	// The month is a wide grid: cells shrink to a tick or a dot, the
	// header is the day number. When even that can't fit, the most
	// recent days stay — the footer says so, nothing is cut silently.
	cellW, label := 5, "Mon 02"
	if tab == habitsMonth {
		cellW, label = 2, "02"
	}
	maxCols := max((inner-nameW-2)/cellW, 1)
	drop := 0
	if len(g.Days) > maxCols {
		drop = len(g.Days) - maxCols
		g.Days = g.Days[drop:]
		// m.habits.col indexes the full day list (toggleHabit's
		// bookkeeping); keep it inside the visible range.
		if m.habits.col < drop {
			m.habits.col = drop
		}
		if m.habits.col >= drop+len(g.Days) {
			m.habits.col = drop + len(g.Days) - 1
		}
	}

	var body []string
	head := "  " + fit("", nameW)
	for i, d := range g.Days {
		cell := fit(dayLabel(d.Name, label), cellW)
		if i == m.habits.col-drop {
			cell = m.st.selFocus.Render(cell)
		} else {
			cell = m.st.muted.Render(cell)
		}
		head += cell
	}
	body = append(body, head)
	// gridCell is the day cell for one row: a tick, a blank (present,
	// unticked), or a dot (no note that day).
	gridCell := func(d habit.Day, name string, cellW int) string {
		switch {
		case d.Path == "":
			if cellW == 2 {
				return "· "
			}
			return "  ·  "
		case g.Done[d.Path][name]:
			if cellW == 2 {
				return "✓ "
			}
			return "  ✓  "
		default:
			if cellW == 2 {
				return "  "
			}
			return "     "
		}
	}
	for r, name := range g.Names {
		line := "  "
		if r == m.habits.cur {
			line += m.st.selFocus.Render(fit(name, nameW))
		} else {
			line += m.st.muted.Render(fit(name, nameW))
		}
		for ci, d := range g.Days {
			cell := gridCell(d, name, cellW)
			if r == m.habits.cur && ci == m.habits.col-drop {
				cell = m.st.selFocus.Render(fit(cell, cellW))
			}
			line += cell
		}
		body = append(body, line)
	}
	foot := "space tick · U undo · H view · esc close"
	if since := habit.Since(g, g.Names[min(m.habits.cur, len(g.Names)-1)]); since != "" {
		foot = "since " + since + " · " + foot
	}
	if drop > 0 {
		foot = "last " + strconv.Itoa(len(g.Days)) + " days shown · " + foot
	}
	body = append(body, "", " "+m.st.muted.Render(foot))
	return body
}

// dayLabel trims a day's full label ("Mon 09-14") to the grid's header
// style: the weekday and date for the week, the day number for the month.
func dayLabel(name, style string) string {
	if style == "02" {
		// "Mon 09-14" → "14"; fall back to whatever is there.
		parts := strings.Split(name, "-")
		if len(parts) == 2 {
			return parts[1]
		}
		return name
	}
	return name
}
