package ui

import (
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
