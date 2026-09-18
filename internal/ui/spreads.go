package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/editor"
	"github.com/lurioso/skrin/internal/habit"
	"github.com/lurioso/skrin/internal/index"
	"github.com/lurioso/skrin/internal/markdown"
	sp "github.com/lurioso/skrin/internal/spread"
)

// spreadOptions lets the note at rel show what its spreads find. With
// spreads off, the renderer leaves the blocks as code.
func (m *Model) spreadOptions(rel string) markdown.SpreadOptions {
	if !m.opts.Spreads {
		return markdown.SpreadOptions{}
	}
	return markdown.SpreadOptions{Run: func(query string) markdown.SpreadResult {
		r := sp.Run(query, indexVault{m.idx}, rel)
		return markdown.SpreadResult{Markdown: r.Markdown, Note: r.Note, Err: r.Err}
	}}
}

// indexVault is the index as a spread sees it.
type indexVault struct{ idx *index.Index }

func (v indexVault) Notes() []sp.Note {
	rels := v.idx.Notes()
	out := make([]sp.Note, 0, len(rels))
	for _, rel := range rels {
		doc, _ := v.idx.Doc(rel)
		mod, size, _ := v.idx.Stat(rel)
		out = append(out, sp.Note{Rel: rel, Tags: doc.Tags, Props: doc.Props, Fields: v.idx.Fields(rel),
			Tasks: tasksOutsideHabits(v.idx.Tasks(rel), doc.Lines), Mod: mod, Size: size})
	}
	return out
}

// tasksOutsideHabits leaves out the checkboxes under a note's "### Habits"
// heading: in Skrin a habit isn't a todo — the rollover skips them too — so
// a TASK spread doesn't list every unticked habit of every day.
func tasksOutsideHabits(tasks []index.Task, lines []string) []sp.Task {
	start, end, ok := habit.BlockRange(lines)
	moved := make([]int, len(tasks)) // old index → new, or -1
	var out []sp.Task
	for i, t := range tasks {
		if ok && t.Line >= start && t.Line < end {
			moved[i] = -1
			continue
		}
		parent := -1
		if t.Parent >= 0 {
			parent = moved[t.Parent]
		}
		moved[i] = len(out)
		out = append(out, sp.Task{Line: t.Line, Status: t.Status, Text: t.Text, Parent: parent, Fields: t.Fields})
	}
	return out
}

func (v indexVault) Resolve(target, from string) (string, bool) { return v.idx.Resolve(target, from) }

func (v indexVault) Backlinks(rel string) []string {
	var out []string
	for _, b := range v.idx.Backlinks(rel) {
		out = append(out, b.Source)
	}
	return out
}

func (v indexVault) Outgoing(rel string) []string { return v.idx.Outgoing(rel) }

// foldHint heads a folded spread in the editor: the one sign that there's
// editable text behind the answer.
const foldHint = "spread · move here to edit"

// editorFolds are the spreads in the editor's text, each folded into its
// answer at width w. The block the cursor is in shows as text anyway, so
// it isn't run: typing a query never runs it half-typed.
func (m *Model) editorFolds(w int) []editor.Fold {
	if !m.opts.Spreads {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(m.editor.Text(), "\r\n", "\n"), "\n")
	cur := m.editor.Line()
	var out []editor.Fold
	for _, b := range markdown.SpreadBlocks(lines) {
		f := editor.Fold{Start: b[0], End: b[1], Rows: []string{""}}
		if cur < b[0] || cur > b[1] {
			f.Rows = m.foldRows(strings.Join(lines[b[0]:b[1]+1], "\n"), w)
		}
		out = append(out, f)
	}
	return out
}

func (m *Model) foldRows(block string, w int) []string {
	key := strconv.Itoa(m.vaultGen) + "\x00" + strconv.Itoa(w) + "\x00" + block
	if rows, ok := m.foldCache[key]; ok {
		return rows
	}
	if m.foldCache == nil || len(m.foldCache) > 64 {
		m.foldCache = map[string][]string{}
	}
	rows := []string{m.st.muted.Render(ansi.Truncate(foldHint, w, "…"))}
	opts := markdown.Options{Width: w, Palette: m.pal, Resolve: m.resolveFrom(m.edit.rel), Spreads: m.spreadOptions(m.edit.rel)}
	for _, l := range markdown.Render(block, opts) {
		rows = append(rows, l.Text)
	}
	m.foldCache[key] = rows
	return rows
}
