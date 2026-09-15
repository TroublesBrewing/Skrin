// Package assistant connects Skrin to Claude Code. Start runs `claude -p`
// headless with stream-json on both ends, one long-lived process per
// conversation. Serve and RunMCP give Claude Skrin's own tools over MCP.
package assistant

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Event is something Claude did. Events arrive in order.
type Event interface{ isEvent() }

// Started reports the conversation's session ID once Claude is up.
type Started struct{ SessionID, Model string }

// Text is a piece of Claude's reply, streamed as it's written.
type Text struct{ Delta string }

// Block marks the start of a new stretch of reply text within a turn.
type Block struct{}

// ToolUse reports a tool Claude called, such as Read or vault_search.
type ToolUse struct {
	Name  string
	Input map[string]any
}

// Done ends a turn: Claude has answered and waits for the next message.
type Done struct {
	SessionID string
	Cost      float64 // in US dollars
	Err       string  // set when the turn failed
}

// Exited reports that the claude process ended.
type Exited struct{ Err error }

func (Started) isEvent() {}
func (Text) isEvent()    {}
func (Block) isEvent()   {}
func (ToolUse) isEvent() {}
func (Done) isEvent()    {}
func (Exited) isEvent()  {}

// Options configure a conversation.
type Options struct {
	Command      string // the claude binary; "" means claude on $PATH
	Dir          string // the working directory: the vault
	Resume       string // a session to continue, if any
	Model        string // "" for Claude Code's default
	SystemPrompt string // added to Claude Code's own
	MCPConfig    string // JSON for --mcp-config; "" for no MCP servers
}

// Args are the command-line arguments for o. Claude may read the vault
// (Read, Glob, Grep) and use Skrin's own tools; anything else that would
// ask for permission is refused.
func Args(o Options) []string {
	args := []string{
		"-p", "--input-format", "stream-json", "--output-format", "stream-json",
		"--verbose", "--include-partial-messages",
		"--tools", "Read,Glob,Grep",
		"--allowedTools", "mcp__skrin",
		"--permission-prompts", "none",
		"--disable-slash-commands",
		"--strict-mcp-config",
	}
	if o.MCPConfig != "" {
		args = append(args, "--mcp-config", o.MCPConfig)
	}
	if o.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", o.SystemPrompt)
	}
	if o.Resume != "" {
		args = append(args, "--resume", o.Resume)
	}
	if o.Model != "" {
		args = append(args, "--model", o.Model)
	}
	return args
}

// Session is a running conversation.
type Session struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	mu    sync.Mutex
}

// Start launches Claude. Its events go to emit, from another goroutine,
// ending with Exited.
func Start(o Options, emit func(Event)) (*Session, error) {
	name := o.Command
	if name == "" {
		name = "claude"
	}
	cmd := exec.Command(name, Args(o)...)
	cmd.Dir = o.Dir
	// Proposals wait for the user to say yes or no, however long that takes.
	cmd.Env = append(os.Environ(), "MCP_TOOL_TIMEOUT=86400000")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go func() {
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 1<<20), 64<<20)
		for sc.Scan() {
			for _, ev := range Parse(sc.Bytes()) {
				emit(ev)
			}
		}
		err := cmd.Wait()
		if msg := lastLine(stderr.String()); err != nil && msg != "" {
			err = fmt.Errorf("%w: %s", err, msg)
		}
		emit(Exited{err})
	}()
	return &Session{cmd: cmd, stdin: stdin}, nil
}

// Send hands Claude the user's next message.
func (s *Session) Send(text string) error {
	b, err := json.Marshal(map[string]any{
		"type":    "user",
		"message": map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": text}}},
	})
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err = s.stdin.Write(append(b, '\n'))
	return err
}

// Close ends the conversation at once, even mid-reply. Claude Code has
// already saved it, so it can be resumed later.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.stdin.Close()
	return s.cmd.Process.Kill()
}

// Parse turns one line of Claude's stream-json output into events. Lines
// it doesn't need, and anything a subagent says, give none.
func Parse(line []byte) []Event {
	var e struct {
		Type      string  `json:"type"`
		Subtype   string  `json:"subtype"`
		SessionID string  `json:"session_id"`
		Model     string  `json:"model"`
		Parent    *string `json:"parent_tool_use_id"`
		IsError   bool    `json:"is_error"`
		Result    string  `json:"result"`
		Cost      float64 `json:"total_cost_usd"`
		Event     struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
			Block struct {
				Type string `json:"type"`
			} `json:"content_block"`
		} `json:"event"`
		Message struct {
			Content []struct {
				Type  string         `json:"type"`
				Name  string         `json:"name"`
				Input map[string]any `json:"input"`
			} `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &e) != nil || e.Parent != nil {
		return nil
	}
	switch e.Type {
	case "system":
		if e.Subtype == "init" {
			return []Event{Started{e.SessionID, e.Model}}
		}
	case "stream_event":
		switch {
		case e.Event.Type == "content_block_delta" && e.Event.Delta.Type == "text_delta":
			return []Event{Text{e.Event.Delta.Text}}
		case e.Event.Type == "content_block_start" && e.Event.Block.Type == "text":
			return []Event{Block{}}
		}
	case "assistant":
		var out []Event
		for _, c := range e.Message.Content {
			if c.Type == "tool_use" {
				out = append(out, ToolUse{c.Name, c.Input})
			}
		}
		return out
	case "result":
		d := Done{SessionID: e.SessionID, Cost: e.Cost}
		if e.IsError {
			d.Err = e.Result
			if d.Err == "" {
				d.Err = e.Subtype
			}
		}
		return []Event{d}
	}
	return nil
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
