package ui

import (
	"errors"
	"fmt"
	"path"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/aymanbagabas/go-udiff"

	"github.com/lurioso/skrin/internal/assistant"
	"github.com/lurioso/skrin/internal/obsidian"
	"github.com/lurioso/skrin/internal/search"
	"github.com/lurioso/skrin/internal/vault"
)

// proposal is a change Claude wants to make, shown in the note pane until
// the user says y or n.
type proposal struct {
	title  string // "Claude wants to edit Filosofi/Stoic.md"
	diff   []string
	off    int
	danger bool                   // a delete
	undo   string                 // how to take it back, for the flash
	apply  func() (string, error) // makes the change; says what was done
	reply  chan assistant.Response
}

// toolCall answers one of Claude's calls to Skrin's tools. Proposals queue
// up for the user's y or n; the rest are answered at once.
func (m *Model) toolCall(msg toolMsg) {
	arg := func(k string) string {
		v, _ := msg.req.Args[k].(string)
		return v
	}
	var resp assistant.Response
	switch msg.req.Tool {
	case "vault_search":
		resp = m.vaultSearch(arg("query"))
	case "current_context":
		resp = assistant.Response{Text: m.currentContext()}
	case "propose_create", "propose_edit", "propose_move", "propose_delete":
		p, err := m.propose(msg.req.Tool, arg)
		if err == nil {
			p.reply = msg.reply
			m.proposals = append(m.proposals, p)
			m.drawer.add("info", "✦ "+p.title+": y applies it, n rejects it")
			return
		}
		resp = assistant.Response{Text: err.Error(), Error: true}
	default:
		resp = assistant.Response{Text: "Skrin has no tool called " + msg.req.Tool, Error: true}
	}
	msg.reply <- resp
}

// propose checks a change Claude wants and prepares it for the user.
func (m *Model) propose(tool string, arg func(string) string) (*proposal, error) {
	switch tool {
	case "propose_create":
		rel, err := m.vaultPath(arg("path"))
		if err != nil {
			return nil, err
		}
		switch {
		case path.Ext(rel) == "":
			rel += ".md"
		case !vault.IsNote(rel):
			return nil, fmt.Errorf("%s isn't a markdown note; Skrin only creates notes", rel)
		}
		if m.vault.Exists(rel) {
			return nil, fmt.Errorf("%s already exists; propose an edit instead", rel)
		}
		for _, part := range strings.Split(rel, "/") {
			if err := vault.CheckName(part); err != nil {
				return nil, err
			}
		}
		content := arg("content")
		return &proposal{
			title: "Claude wants to create " + rel, diff: diffOf("", content), undo: "U",
			apply: func() (string, error) {
				dirs, err := m.vault.CreateFile(rel, content)
				steps := createdSteps(dirs)
				if err == nil {
					steps = append(steps, vault.Step{Kind: vault.StepCreated, Rel: rel, Content: content})
				}
				m.journal.Record(vault.Op{Desc: "create " + rel + " (Claude)", Steps: steps})
				m.refresh()
				if err != nil {
					return "", err
				}
				return "Created " + rel + ".", nil
			},
		}, nil

	case "propose_edit":
		rel, err := m.vaultPath(arg("path"))
		if err != nil {
			return nil, err
		}
		if !vault.IsNote(rel) || !m.vault.Exists(rel) {
			return nil, fmt.Errorf("%s isn't a note in the vault", rel)
		}
		before, err := m.vault.Read(rel)
		if err != nil {
			return nil, err
		}
		old, repl := arg("old_text"), arg("new_text")
		switch n := strings.Count(before, old); {
		case old == "":
			return nil, errors.New("old_text is empty; give the exact text to replace")
		case n == 0:
			return nil, fmt.Errorf("old_text isn't in %s; read the note again and copy the text exactly", rel)
		case n > 1:
			return nil, fmt.Errorf("old_text appears %d times in %s; include more around it so it's unique", n, rel)
		}
		after := strings.Replace(before, old, repl, 1)
		return &proposal{
			title: "Claude wants to edit " + rel, diff: diffOf(before, after), undo: "u",
			apply: func() (string, error) {
				now, err := m.vault.Read(rel)
				switch {
				case err != nil:
					return "", err
				case now != before:
					return "", fmt.Errorf("%s changed since this was proposed; read it again", rel)
				}
				if err := m.snaps.Save(rel, before); err != nil {
					return "", fmt.Errorf("couldn't keep a snapshot, so nothing changed: %w", err)
				}
				if err := m.vault.Write(rel, after); err != nil {
					return "", err
				}
				m.journal.Record(vault.Op{Desc: "Claude's edit of " + rel, Steps: []vault.Step{{Kind: vault.StepModified, Rel: rel, Content: before}}})
				m.refresh()
				return "Edited " + rel + ".", nil
			},
		}, nil

	case "propose_move":
		from, err := m.vaultPath(arg("from"))
		if err != nil {
			return nil, err
		}
		to, err := m.vaultPath(arg("to"))
		if err != nil {
			return nil, err
		}
		if !m.vault.Exists(from) {
			return nil, fmt.Errorf("%s doesn't exist", from)
		}
		if vault.IsNote(from) && path.Ext(to) == "" {
			to += ".md"
		}
		switch {
		case m.vault.Exists(to):
			return nil, fmt.Errorf("%s already exists", to)
		case movesIntoItself(to, []string{from}):
			return nil, fmt.Errorf("can't move %s into itself", from)
		}
		for _, part := range strings.Split(to, "/") {
			if err := vault.CheckName(part); err != nil {
				return nil, err
			}
		}
		moves := [][2]string{{from, to}}
		diff := []string{"-" + from, "+" + to}
		if n := len(m.referrers(moves)); n > 0 {
			diff = append(diff, "@@ "+plural(n, "link")+" to it will be updated so they keep working")
		}
		return &proposal{
			title: "Claude wants to move " + from + " to " + to, diff: diff, undo: "U",
			apply: func() (string, error) {
				moved, relinked, err := m.moveNow(moves, m.referrers(moves), "move "+from+" to "+to+" (Claude)", true)
				m.followMoves(moved, false)
				m.refresh()
				if err != nil {
					return "", err
				}
				return "Moved " + from + " to " + to + linksNote(relinked) + ".", nil
			},
		}, nil

	case "propose_delete":
		rel, err := m.vaultPath(arg("path"))
		if err != nil {
			return nil, err
		}
		if !m.vault.Exists(rel) {
			return nil, fmt.Errorf("%s doesn't exist", rel)
		}
		option := obsidian.LoadSettings(m.vault.Root).TrashOption
		where := "to the trash"
		switch option {
		case "local":
			where = "to the vault's .trash folder"
		case "none":
			where = "permanently"
		}
		var diff []string
		if m.vault.IsDir(rel) {
			diff = []string{"-" + rel + "/", "@@ a folder with " + plural(m.vault.CountNotes(rel), "note")}
		} else if src, err := m.vault.Read(rel); err == nil {
			diff = diffOf(src, "")
		}
		return &proposal{
			title: "Claude wants to delete " + rel + " " + where, diff: diff, danger: true, undo: "U",
			apply: func() (string, error) {
				m.deletePaths([]string{rel}, option)
				if m.vault.Exists(rel) {
					return "", errors.New(m.flash)
				}
				return "Deleted " + rel + " " + where + ".", nil
			},
		}, nil
	}
	return nil, fmt.Errorf("unknown proposal %s", tool)
}

// proposalKey is y or n on the proposal in view, or j/k to scroll its diff.
func (m *Model) proposalKey(k tea.KeyPressMsg) {
	p := m.proposals[0]
	switch m.actionIn(inProposal, k.String()) {
	case actApply:
		m.proposals = m.proposals[1:]
		done, err := p.apply()
		if err != nil {
			p.reply <- assistant.Response{Text: "The user approved it, but it failed: " + err.Error(), Error: true}
			m.drawer.add("error", "Couldn't apply it: "+err.Error())
			m.flash = "Couldn't apply Claude's change: " + err.Error()
			return
		}
		p.reply <- assistant.Response{Text: "The user approved it, and it's done: " + done}
		m.drawer.add("info", "✓ "+done)
		m.flash = done + " · " + p.undo + " undoes it"
	case actReject:
		m.proposals = m.proposals[1:]
		p.reply <- assistant.Response{Text: "The user rejected this change."}
		m.drawer.add("info", "✗ rejected: "+strings.TrimPrefix(p.title, "Claude wants to "))
		m.flash = "Rejected Claude's change"
	case actDown:
		p.off = min(p.off+1, max(len(p.diff)-1, 0))
	case actUp:
		p.off = max(p.off-1, 0)
	}
}

func (m *Model) proposalLine() string {
	p := m.proposals[0]
	pill := m.st.pill
	if p.danger {
		pill = m.st.dangerPill
	}
	q := p.title + "?"
	if n := len(m.proposals); n > 1 {
		q += fmt.Sprintf(" (1 of %d)", n)
	}
	return spread(pill.Render(" CLAUDE ")+" "+m.st.text.Render(q), m.st.bold.Render("y apply · n reject · j/k scroll"), m.width)
}

// vaultPath turns a path from Claude into a vault-relative one, refusing
// anything outside the vault or hidden, like .obsidian.
func (m *Model) vaultPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if rest, ok := strings.CutPrefix(p, m.vault.Root+"/"); ok {
		p = rest
	}
	clean := path.Clean(p)
	if p == "" || strings.HasPrefix(p, "/") || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("%q isn't a path inside the vault", p)
	}
	for _, part := range strings.Split(clean, "/") {
		if strings.HasPrefix(part, ".") {
			return "", fmt.Errorf("%s is hidden; Skrin doesn't touch hidden files like .obsidian", clean)
		}
	}
	return clean, nil
}

// diffOf is a unified diff from a to b, without the file header.
func diffOf(a, b string) []string {
	var out []string
	for _, l := range strings.Split(strings.TrimRight(udiff.Unified("now", "after", a, b), "\n"), "\n") {
		if !strings.HasPrefix(l, "--- ") && !strings.HasPrefix(l, "+++ ") && !strings.HasPrefix(l, `\ `) && l != "" {
			out = append(out, l)
		}
	}
	return out
}

// vaultSearch is the vault_search tool: Skrin's own search, as text.
func (m *Model) vaultSearch(query string) assistant.Response {
	q := search.Parse(query, false)
	if q.Empty() {
		return assistant.Response{Text: "The query is empty.", Error: true}
	}
	var b strings.Builder
	notes := 0
	for _, rel := range m.idx.Notes() {
		d, _ := m.idx.Doc(rel)
		ok, hits := q.Match(d)
		if !ok {
			continue
		}
		if notes++; notes > 30 {
			b.WriteString("…and more notes; narrow the search.\n")
			break
		}
		b.WriteString(rel + "\n")
		for i, h := range hits {
			if i == 5 {
				fmt.Fprintf(&b, "  …%d more lines\n", len(hits)-5)
				break
			}
			if h.Line < len(d.Lines) {
				fmt.Fprintf(&b, "  %d: %s\n", h.Line+1, strings.TrimSpace(d.Lines[h.Line]))
			}
		}
	}
	if notes == 0 {
		return assistant.Response{Text: "No notes match " + query + "."}
	}
	return assistant.Response{Text: b.String()}
}

// currentContext is the current_context tool: what's open and highlighted.
func (m *Model) currentContext() string {
	var b strings.Builder
	if m.notePath == "" {
		b.WriteString("No note is open.\n")
	} else {
		fmt.Fprintf(&b, "Open note: %s\n", m.notePath)
	}
	if sel := m.selectionText(); sel != "" {
		fmt.Fprintf(&b, "Highlighted:\n%s\n", sel)
	}
	if r := m.files.selected().Rel; r != "" {
		fmt.Fprintf(&b, "Row under the cursor in Files: %s\n", r)
	}
	if m.notePath != "" {
		b.WriteString("\n" + m.noteContext() + "\n")
	}
	return b.String()
}
