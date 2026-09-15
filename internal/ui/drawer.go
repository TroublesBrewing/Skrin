package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/assistant"
	"github.com/lurioso/skrin/internal/daily"
	"github.com/lurioso/skrin/internal/editor"
	"github.com/lurioso/skrin/internal/markdown"
	"github.com/lurioso/skrin/internal/obsidian"
)

// ClaudeSession is a running conversation with Claude.
type ClaudeSession interface {
	Send(text string) error
	Close() error
}

// AssistantOptions set up the Claude drawer.
type AssistantOptions struct {
	Enabled bool
	Right   bool // start on the right instead of along the bottom
	// Claude is how to run Claude: the command, model and MCP config. The
	// working directory, the conversation to resume and the system prompt
	// are filled in here.
	Claude assistant.Options
	// Start runs a conversation; nil means assistant.Start. Tests fake it.
	Start func(assistant.Options, func(assistant.Event)) (ClaudeSession, error)
}

// claudeMsg carries one of Claude's events into the update loop. gen tells
// the current conversation's events from those of one that has ended.
type claudeMsg struct {
	gen int
	ev  assistant.Event
}

// toolMsg is a tool call from Claude waiting for Skrin's answer.
type toolMsg struct {
	req   assistant.Request
	reply chan assistant.Response
}

// drawer is the Claude drawer: the conversation and what you're typing.
type drawer struct {
	open    bool
	right   bool // down the right side instead of along the bottom
	moved   bool // Alt-p flipped it, so the session remembers the side
	input   *editor.Editor
	msgs    []chatMsg
	scroll  int // transcript lines scrolled back from the newest
	busy    bool
	session ClaudeSession
	gen     int
	started bool   // the current process has said hello
	id      string // the conversation, for --resume
	sent    string // the open note as Claude last saw it
	back    pane   // where Esc returns to
}

// chatMsg is one entry in the transcript.
type chatMsg struct {
	who   string // you, claude, tool, info or error
	text  string
	lines []string // rendered at width w
	w     int
}

func (d *drawer) add(who, text string) {
	d.msgs = append(d.msgs, chatMsg{who: who, text: text})
	d.scroll = 0
}

// last is the newest entry if it's by who.
func (d *drawer) last(who string) *chatMsg {
	if n := len(d.msgs); n > 0 && d.msgs[n-1].who == who {
		return &d.msgs[n-1]
	}
	return nil
}

// listen waits for the next event from Claude or its tools.
func (m *Model) listen() tea.Cmd { return func() tea.Msg { return <-m.events } }

// ToolHandler answers Claude's tool calls for assistant.Serve. Each call
// waits for the update loop, and a proposal for the user's y or n.
func (m *Model) ToolHandler() func(assistant.Request) assistant.Response {
	return func(req assistant.Request) assistant.Response {
		reply := make(chan assistant.Response, 1)
		m.events <- toolMsg{req, reply}
		return <-reply
	}
}

// Close ends the conversation with Claude, if one is running.
func (m *Model) Close() {
	if s := m.drawer.session; s != nil {
		s.Close()
	}
}

// drawerRight reports whether the drawer is on the right now. It needs a
// wide terminal; narrower, it goes along the bottom.
func (m *Model) drawerRight() bool { return m.drawer.right && m.width >= 100 }

// drawerSide is the side Alt-p chose, for the session; "" if it wasn't used.
func (m *Model) drawerSide() string {
	switch {
	case !m.drawer.moved:
		return ""
	case m.drawer.right:
		return "right"
	}
	return "bottom"
}

// openDrawer shows the drawer and puts the cursor in its input. Text
// highlighted in the note or the editor goes in first, on a line of its
// own, with a new line below it to type on.
func (m *Model) openDrawer() {
	if !m.opts.Assistant.Enabled {
		m.flash = "The Claude drawer is off: assistant.enabled = false in config.toml"
		return
	}
	d := &m.drawer
	if sel := m.selectionText(); sel != "" {
		if text := d.input.Text(); text != "" && !strings.HasSuffix(text, "\n") {
			sel = "\n" + sel
		}
		d.input.Paste(sel + "\n")
		m.clearSelection()
	}
	if m.focus != paneClaude {
		d.back = m.focus
	}
	d.open, m.zen, m.focus = true, false, paneClaude
}

// toggleDrawer is c: it hides the drawer when it's open but you're
// elsewhere, and otherwise opens it.
func (m *Model) toggleDrawer() {
	if m.drawer.open && m.focus != paneClaude {
		m.drawer.open = false
		return
	}
	m.openDrawer()
}

func (m *Model) leaveDrawer() {
	m.focus = m.drawer.back
	if m.focus == paneClaude {
		m.focus = paneNote
	}
}

func (m *Model) drawerKey(k tea.KeyPressMsg) {
	d := &m.drawer
	switch k.String() {
	case "enter":
		m.sendToClaude()
		return
	case "alt+enter", "shift+enter":
		d.input.HandleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
		return
	case "alt+p":
		d.right, d.moved = !d.right, true
		switch {
		case !d.right:
			m.flash = "Drawer along the bottom · alt+p moves it back"
		case m.width < 100:
			m.flash = "The drawer goes on the right from 100 columns; until then it stays at the bottom"
		default:
			m.flash = "Drawer on the right · alt+p moves it back"
		}
		return
	case "alt+n":
		m.newConversation()
		return
	case "pgup":
		d.scroll += 5
		return
	case "pgdown":
		d.scroll = max(d.scroll-5, 0)
		return
	}
	if d.input.HandleKey(k) == editor.Close {
		m.leaveDrawer()
	}
}

// sendToClaude sends what's typed, led by the open note when Claude hasn't
// seen it as it is now.
func (m *Model) sendToClaude() {
	d := &m.drawer
	text := strings.TrimSpace(d.input.Text())
	switch {
	case text == "":
		return
	case d.busy:
		m.flash = "Claude is still answering; send this when it's done"
		return
	}
	if err := m.ensureClaude(); err != nil {
		d.add("error", "Couldn't start Claude: "+err.Error())
		return
	}
	msg := text
	if ctx := m.noteContext(); ctx != d.sent && !(d.sent == "" && m.notePath == "") {
		msg = ctx + "\n\n" + text
		d.sent = ctx
	}
	if err := d.session.Send(msg); err != nil {
		d.add("error", "Couldn't reach Claude: "+err.Error())
		return
	}
	d.add("you", text)
	d.input.Reset("")
	d.busy = true
}

// noteContext is the open note as Claude gets it: its path and full text,
// unsaved changes in the editor included.
func (m *Model) noteContext() string {
	if m.notePath == "" {
		return "<open-note>none</open-note>"
	}
	src := m.noteSrc
	if m.editor != nil && m.edit.rel == m.notePath {
		src = m.editor.Text()
	}
	return fmt.Sprintf("<open-note path=%q>\n%s\n</open-note>", m.notePath, src)
}

// ensureClaude starts Claude if it isn't running, picking up the
// remembered conversation.
func (m *Model) ensureClaude() error {
	d := &m.drawer
	if d.session != nil {
		return nil
	}
	o := m.opts.Assistant.Claude
	s := obsidian.LoadSettings(m.vault.Root)
	o.Dir, o.Resume = m.vault.Root, d.id
	o.SystemPrompt = assistant.SystemPrompt(assistant.Vault{
		Name: m.vault.Name(), DailyFolder: s.Daily.Folder, DailyFormat: s.Daily.Format, DailyTemplate: daily.TemplatePath(s.Daily),
	})
	start := m.opts.Assistant.Start
	if start == nil {
		start = func(o assistant.Options, emit func(assistant.Event)) (ClaudeSession, error) {
			s, err := assistant.Start(o, emit)
			if err != nil {
				return nil, err
			}
			return s, nil
		}
	}
	d.gen++
	gen, events := d.gen, m.events
	sess, err := start(o, func(ev assistant.Event) { events <- claudeMsg{gen, ev} })
	if err != nil {
		return err
	}
	d.session, d.started, d.sent = sess, false, ""
	return nil
}

func (m *Model) claudeEvent(msg claudeMsg) {
	d := &m.drawer
	if msg.gen != d.gen {
		return // from a conversation that has ended
	}
	switch ev := msg.ev.(type) {
	case assistant.Started:
		d.started, d.id = true, ev.SessionID
	case assistant.Block:
		if c := d.last("claude"); c != nil && c.text != "" {
			c.text, c.lines = c.text+"\n\n", nil
		}
	case assistant.Text:
		c := d.last("claude")
		if c == nil {
			d.add("claude", "")
			c = d.last("claude")
		}
		c.text, c.lines = c.text+ev.Delta, nil
	case assistant.ToolUse:
		if s := m.describeTool(ev); s != "" {
			d.add("tool", s)
		}
	case assistant.Done:
		d.busy = false
		if ev.SessionID != "" {
			d.id = ev.SessionID
		}
		if ev.Err != "" {
			d.add("error", "Claude stopped: "+ev.Err)
		}
		if m.focus != paneClaude {
			m.flash = "Claude has answered · C to reply"
		}
	case assistant.Exited:
		d.session, d.busy = nil, false
		switch {
		case ev.Err == nil:
		case !d.started && d.id != "":
			d.id = ""
			d.add("error", "Couldn't pick up the last conversation, so the next message starts a new one ("+ev.Err.Error()+")")
		default:
			d.add("error", "Claude quit: "+ev.Err.Error())
		}
	}
}

// describeTool is the transcript's line for a tool Claude used.
func (m *Model) describeTool(t assistant.ToolUse) string {
	arg := func(k string) string {
		v, _ := t.Input[k].(string)
		return strings.TrimPrefix(v, m.vault.Root+"/")
	}
	switch t.Name {
	case "Read":
		return "read " + arg("file_path")
	case "Glob":
		return "looked for " + arg("pattern")
	case "Grep":
		return "searched the files for “" + arg("pattern") + "”"
	case "mcp__skrin__vault_search":
		return "searched the vault for “" + arg("query") + "”"
	case "mcp__skrin__current_context":
		return "looked at what's open"
	}
	if strings.HasPrefix(t.Name, "mcp__skrin__propose_") {
		return "" // the proposal speaks for itself
	}
	return "used " + t.Name
}

// newConversation is Alt-n: Claude starts afresh, and what's pending is
// dropped.
func (m *Model) newConversation() {
	d := &m.drawer
	if d.session != nil {
		d.session.Close()
	}
	for _, p := range m.proposals {
		p.reply <- assistant.Response{Text: "The conversation ended before the user decided.", Error: true}
	}
	m.proposals = nil
	d.session, d.id, d.msgs, d.sent, d.busy, d.scroll = nil, "", nil, "", false, 0
	d.gen++
	m.flash = "New conversation with Claude"
}

// drawerPane draws the open drawer: the conversation above, the input
// below.
func (m *Model) drawerPane(w, h int) []string {
	d := &m.drawer
	inner, rows := w-2, h-2
	inputH := clamp(strings.Count(d.input.Text(), "\n")+1, 1, max(min(6, rows/3), 1))
	d.input.SetSize(max(inner-2, 4), inputH)
	transH := max(rows-inputH-1, 0)
	lines := m.transcript(inner - 1)
	d.scroll = clamp(d.scroll, 0, max(len(lines)-transH, 0))
	end := len(lines) - d.scroll
	body := lines[max(end-transH, 0):end]
	for len(body) < transH {
		body = append([]string{""}, body...) // the newest line sits just above the input
	}
	body = append(body, m.st.border.Render(strings.Repeat("─", inner)))
	for i, l := range d.input.View() {
		lead := "  "
		if i == 0 {
			lead = m.st.flash.Render("› ")
		}
		body = append(body, lead+l)
	}
	title := "✦ Claude"
	if d.busy {
		title += " · thinking…"
	}
	return m.box(title, body, w, h, m.focus == paneClaude)
}

// drawerLine is the drawer folded to one line, along the bottom while
// you're elsewhere.
func (m *Model) drawerLine() string {
	d := &m.drawer
	status := "c hides · C to type"
	switch {
	case len(m.proposals) > 0:
		status = "wants to change a note: y or n"
	case d.busy:
		status = "thinking…"
	default:
		if c := d.lastReply(); c != "" {
			status = c
		}
	}
	return fit(" "+m.st.brand.Render("✦ Claude")+m.st.muted.Render(" · "+status), m.width)
}

// lastReply is the last line of Claude's latest reply.
func (d *drawer) lastReply() string {
	for i := len(d.msgs) - 1; i >= 0; i-- {
		if d.msgs[i].who == "claude" {
			lines := strings.Split(strings.TrimSpace(d.msgs[i].text), "\n")
			return strings.TrimSpace(lines[len(lines)-1])
		}
	}
	return ""
}

// transcript is the conversation, rendered at width w.
func (m *Model) transcript(w int) []string {
	d := &m.drawer
	if len(d.msgs) == 0 {
		return []string{
			m.st.muted.Render("Ask Claude about your vault. It sees the open note,"),
			m.st.muted.Render("and any change it wants to make waits for your y."),
			"",
			m.st.muted.Render("enter send · alt+enter new line · esc back"),
			m.st.muted.Render("alt+p move the drawer · alt+n new conversation"),
		}
	}
	var out []string
	for i := range d.msgs {
		c := &d.msgs[i]
		if c.lines == nil || c.w != w {
			c.lines, c.w = m.renderChat(*c, w), w
		}
		if c.who == "you" && i > 0 {
			out = append(out, "")
		}
		out = append(out, c.lines...)
	}
	return out
}

func (m *Model) renderChat(c chatMsg, w int) []string {
	var out []string
	switch c.who {
	case "claude":
		for _, l := range markdown.Render(c.text, markdown.Options{Width: w, Palette: m.pal, Resolve: m.resolve}) {
			out = append(out, l.Text)
		}
		return out
	case "you":
		for i, l := range wrapText(c.text, w-2) {
			lead := "  "
			if i == 0 {
				lead = "› "
			}
			out = append(out, m.st.flash.Render(lead)+m.st.bold.Render(l))
		}
		return out
	}
	st := m.st.flash
	switch c.who {
	case "tool":
		st = m.st.muted
	case "error":
		st = m.st.errText
	}
	for i, l := range wrapText(c.text, w-2) {
		lead := "  "
		if i == 0 && c.who == "tool" {
			lead = "· "
		}
		out = append(out, st.Render(lead+l))
	}
	return out
}

// wrapText wraps each line of s to w cells, keeping the line breaks.
func wrapText(s string, w int) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if ansi.StringWidth(line) <= w {
			out = append(out, line)
			continue
		}
		out = append(out, wrap(line, w)...)
	}
	return out
}
