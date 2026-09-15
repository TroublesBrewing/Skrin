package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/assistant"
	"github.com/lurioso/skrin/internal/session"
)

// fakeClaude stands in for the claude process: it records what it's sent.
type fakeClaude struct {
	opts   []assistant.Options // one per start
	sent   []string
	closed int
}

func (f *fakeClaude) Send(text string) error { f.sent = append(f.sent, text); return nil }
func (f *fakeClaude) Close() error           { f.closed++; return nil }

func newClaudeModel(t *testing.T, opts Options) (*Model, *fakeClaude) {
	t.Helper()
	f := &fakeClaude{}
	opts.Assistant.Enabled = true
	opts.Assistant.Start = func(o assistant.Options, _ func(assistant.Event)) (ClaudeSession, error) {
		f.opts = append(f.opts, o)
		return f, nil
	}
	return newTestModelWith(t, opts), f
}

// claudeSays feeds events from the current conversation.
func claudeSays(m *Model, evs ...assistant.Event) {
	for _, ev := range evs {
		m.Update(claudeMsg{m.drawer.gen, ev})
	}
}

// askTool makes a tool call as Claude would, returning where the answer goes.
func askTool(m *Model, tool string, args map[string]any) chan assistant.Response {
	reply := make(chan assistant.Response, 1)
	m.Update(toolMsg{assistant.Request{Tool: tool, Args: args}, reply})
	return reply
}

func TestDrawerTalksToClaude(t *testing.T) {
	m, f := newClaudeModel(t, Options{})
	press(m, "G", "enter", "C")
	if m.focus != paneClaude || !m.drawer.open {
		t.Fatal("C should open the drawer, ready to type")
	}
	checkFrame(t, m, "drawer, focused")
	typeText(m, "what is this?")
	press(m, "enter")
	if len(f.sent) != 1 || !strings.Contains(f.sent[0], `<open-note path="Welcome.md">`) || !strings.HasSuffix(f.sent[0], "what is this?") {
		t.Fatalf("sent %q", f.sent)
	}
	if o := f.opts[0]; !strings.Contains(o.SystemPrompt, "propose_edit") || o.Dir != m.vault.Root || o.Resume != "" {
		t.Errorf("started with %+v", o)
	}
	claudeSays(m, assistant.Started{SessionID: "s1"}, assistant.Text{Delta: "It's your "}, assistant.Text{Delta: "welcome note."},
		assistant.ToolUse{Name: "Read", Input: map[string]any{"file_path": m.vault.Abs("Welcome.md")}},
		assistant.Done{SessionID: "s1"})
	if m.drawer.busy || m.drawer.id != "s1" {
		t.Errorf("busy %v, conversation %q", m.drawer.busy, m.drawer.id)
	}
	if c := m.drawer.msgs[1]; c.who != "claude" || c.text != "It's your welcome note." {
		t.Errorf("reply %+v", c)
	}
	if c := m.drawer.msgs[2]; c.who != "tool" || c.text != "read Welcome.md" {
		t.Errorf("tool line %+v", c)
	}
	if !strings.Contains(ansi.Strip(m.render()), "It's your welcome note.") {
		t.Error("the reply isn't on screen")
	}
	checkFrame(t, m, "drawer with a reply")
	typeText(m, "and now?")
	press(m, "enter")
	if len(f.sent) != 2 || strings.Contains(f.sent[1], "<open-note") {
		t.Errorf("an unchanged note was sent again: %q", f.sent[1])
	}
	press(m, "esc")
	if m.focus != paneNote || m.layout().drawerH != 1 {
		t.Errorf("esc: focus %v, drawer rows %d; want the note, and the drawer folded", m.focus, m.layout().drawerH)
	}
	checkFrame(t, m, "drawer folded")
	if m.Session().Claude != "s1" {
		t.Error("the conversation should be remembered")
	}
	press(m, "c")
	if m.drawer.open {
		t.Error("c should hide the drawer when you're elsewhere")
	}
}

func TestDrawerTakesHighlightedText(t *testing.T) {
	m, _ := newClaudeModel(t, Options{})
	press(m, "G", "enter", "v", "C")
	if got := m.drawer.input.Text(); got != "# Welcome\n" {
		t.Errorf("from the reading view: %q", got)
	}
	if m.noteSel != nil {
		t.Error("the line selection should be used up")
	}
	press(m, "esc")
	m.drawer.input.Reset("")
	press(m, "e", "shift+right", "shift+right", "shift+right", "ctrl+k")
	if got := m.drawer.input.Text(); got != "# W\n" {
		t.Errorf("from the editor: %q", got)
	}
	press(m, "alt+enter")
	if got := m.drawer.input.Text(); got != "# W\n\n" {
		t.Errorf("alt+enter should add a line: %q", got)
	}
	press(m, "esc")
	if m.editor == nil || m.focus != paneNote {
		t.Error("esc in the drawer should go back to the editor")
	}
}

func TestProposalsWaitForTheUser(t *testing.T) {
	m, _ := newClaudeModel(t, Options{})
	r := askTool(m, "propose_edit", map[string]any{"path": "Welcome.md", "old_text": "# Welcome", "new_text": "# Hello"})
	if len(m.proposals) != 1 || read(m, "Welcome.md") != welcome {
		t.Fatal("the edit should wait for the user")
	}
	if frame := ansi.Strip(m.render()); !strings.Contains(frame, "+# Hello") || !strings.Contains(frame, "y apply") {
		t.Errorf("the diff and the question aren't shown:\n%s", frame)
	}
	checkFrame(t, m, "proposal")
	press(m, "y")
	if got := read(m, "Welcome.md"); !strings.HasPrefix(got, "# Hello\n") {
		t.Fatalf("not applied: %q", got)
	}
	if resp := <-r; resp.Error || !strings.Contains(resp.Text, "approved") {
		t.Errorf("reply %+v", resp)
	}
	press(m, "G", "enter", "u")
	if got := read(m, "Welcome.md"); got != welcome {
		t.Errorf("u should undo Claude's edit: %q", got)
	}

	r = askTool(m, "propose_create", map[string]any{"path": "Ideas/Garden", "content": "# Garden\n"})
	press(m, "y")
	if read(m, "Ideas/Garden.md") != "# Garden\n" || (<-r).Error {
		t.Fatal("create failed")
	}
	press(m, "U")
	if m.vault.Exists("Ideas/Garden.md") {
		t.Error("U should undo Claude's create")
	}

	r = askTool(m, "propose_move", map[string]any{"from": "Filosofi/Stoic.md", "to": "Daily/Stoic"})
	press(m, "y")
	if !m.vault.Exists("Daily/Stoic.md") || (<-r).Error {
		t.Error("move failed")
	}

	r = askTool(m, "propose_delete", map[string]any{"path": "Welcome.md"})
	press(m, "n")
	if !m.vault.Exists("Welcome.md") || !strings.Contains((<-r).Text, "rejected") {
		t.Error("n should reject the delete")
	}
}

func TestProposalsAreChecked(t *testing.T) {
	m, _ := newClaudeModel(t, Options{})
	for _, c := range []struct {
		tool string
		args map[string]any
		want string
	}{
		{"propose_edit", map[string]any{"path": "Welcome.md", "old_text": "nope", "new_text": "x"}, "isn't in"},
		{"propose_edit", map[string]any{"path": "Welcome.md", "old_text": "", "new_text": "x"}, "empty"},
		{"propose_edit", map[string]any{"path": "../etc/passwd", "old_text": "a", "new_text": "b"}, "inside the vault"},
		{"propose_create", map[string]any{"path": ".obsidian/app.json", "content": "{}"}, "hidden"},
		{"propose_create", map[string]any{"path": "Welcome.md", "content": "x"}, "already exists"},
		{"propose_move", map[string]any{"from": "Filosofi", "to": "Filosofi/Antik/F"}, "into itself"},
		{"propose_delete", map[string]any{"path": "Nowhere.md"}, "doesn't exist"},
	} {
		resp := <-askTool(m, c.tool, c.args)
		if !resp.Error || !strings.Contains(resp.Text, c.want) {
			t.Errorf("%s %v: %+v, want an error about %q", c.tool, c.args, resp, c.want)
		}
	}
	if len(m.proposals) != 0 {
		t.Error("refused proposals shouldn't wait for the user")
	}
}

func TestSearchAndContextTools(t *testing.T) {
	m, _ := newClaudeModel(t, Options{})
	if resp := <-askTool(m, "vault_search", map[string]any{"query": "#start"}); !strings.Contains(resp.Text, "Welcome.md") {
		t.Errorf("vault_search: %+v", resp)
	}
	press(m, "G", "enter")
	resp := <-askTool(m, "current_context", nil)
	if !strings.Contains(resp.Text, "Open note: Welcome.md") || !strings.Contains(resp.Text, "<open-note") {
		t.Errorf("current_context: %+v", resp)
	}
}

func TestDrawerOnTheRight(t *testing.T) {
	m, _ := newClaudeModel(t, Options{Assistant: AssistantOptions{Right: true}})
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "C")
	if m.layout().drawerW == 0 {
		t.Fatal("the drawer should be on the right")
	}
	for _, size := range sizes {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		checkFrame(t, m, fmt.Sprintf("drawer on the right, %dx%d", size[0], size[1]))
	}
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	press(m, "alt+p")
	if l := m.layout(); l.drawerW != 0 || l.drawerH == 0 {
		t.Errorf("alt+p should move it to the bottom: %+v", l)
	}
	if m.Session().Drawer != "bottom" {
		t.Error("the side should be remembered")
	}
}

func TestNewConversationAndResume(t *testing.T) {
	m, f := newClaudeModel(t, Options{Session: session.State{Claude: "s9"}})
	press(m, "C")
	typeText(m, "hi")
	press(m, "enter")
	if f.opts[0].Resume != "s9" {
		t.Errorf("should resume the remembered conversation: %+v", f.opts[0])
	}
	press(m, "alt+n")
	if f.closed != 1 || m.drawer.id != "" || len(m.drawer.msgs) != 0 {
		t.Errorf("alt+n: closed %d, id %q, %d messages", f.closed, m.drawer.id, len(m.drawer.msgs))
	}
	m.Update(claudeMsg{m.drawer.gen - 1, assistant.Text{Delta: "late"}})
	if len(m.drawer.msgs) != 0 {
		t.Error("the old conversation's events should be ignored")
	}
	typeText(m, "fresh")
	press(m, "enter")
	if len(f.opts) != 2 || f.opts[1].Resume != "" {
		t.Error("the next message should start a fresh conversation")
	}
}

func TestDrawerCanBeOff(t *testing.T) {
	m := newTestModel(t)
	press(m, "c")
	if m.drawer.open || !strings.Contains(m.flash, "off") {
		t.Errorf("open %v, flash %q", m.drawer.open, m.flash)
	}
}
