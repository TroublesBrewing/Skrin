package ui

import (
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
		out = append(out, sp.Note{Rel: rel, Tags: doc.Tags, Props: doc.Props, Mod: mod, Size: size})
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
