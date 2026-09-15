package assistant

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	cases := []struct {
		line string
		want []Event
	}{
		{`{"type":"system","subtype":"init","session_id":"s1","model":"claude-opus-5"}`, []Event{Started{"s1", "claude-opus-5"}}},
		{`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hej"}},"parent_tool_use_id":null}`, []Event{Text{"hej"}}},
		{`{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}}`, []Event{Block{}}},
		{`{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"thinking"}}}`, nil},
		{`{"type":"assistant","message":{"content":[{"type":"text","text":"hi"},{"type":"tool_use","name":"Read","input":{"file_path":"/v/a.md"}}]}}`, []Event{ToolUse{"Read", map[string]any{"file_path": "/v/a.md"}}}},
		{`{"type":"result","subtype":"success","session_id":"s1","total_cost_usd":0.5}`, []Event{Done{SessionID: "s1", Cost: 0.5}}},
		{`{"type":"result","subtype":"error_during_execution","is_error":true,"session_id":"s1"}`, []Event{Done{SessionID: "s1", Err: "error_during_execution"}}},
		{`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"sub"}},"parent_tool_use_id":"toolu_1"}`, nil},
		{`not json`, nil},
	}
	for _, c := range cases {
		if got := Parse([]byte(c.line)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Parse(%s)\n got %#v\nwant %#v", c.line, got, c.want)
		}
	}
}

func TestArgsLockDownTools(t *testing.T) {
	args := Args(Options{Resume: "s1", MCPConfig: "{}", SystemPrompt: "be kind"})
	joined := strings.Join(args, " ")
	for _, want := range []string{"--tools Read,Glob,Grep", "--allowedTools mcp__skrin", "--permission-prompts none", "--strict-mcp-config", "--resume s1", "--input-format stream-json"} {
		if !strings.Contains(joined, want) {
			t.Errorf("args lack %q: %s", want, joined)
		}
	}
	if slices.Contains(args, "--model") {
		t.Error("no model was asked for")
	}
}

// fakeClaude answers every message with "hej" and ends the turn.
const fakeClaude = `#!/bin/sh
echo '{"type":"system","subtype":"init","session_id":"s1","model":"fake"}'
while read line; do
  echo '{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"hej"}}}'
  echo '{"type":"result","subtype":"success","session_id":"s1","total_cost_usd":0.01}'
done
`

func TestSessionTalksStreamJSON(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(bin, []byte(fakeClaude), 0o755); err != nil {
		t.Fatal(err)
	}
	events := make(chan Event, 16)
	s, err := Start(Options{Command: bin, Dir: t.TempDir()}, func(e Event) { events <- e })
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Send("hi"); err != nil {
		t.Fatal(err)
	}
	var got []Event
	for len(got) < 3 {
		select {
		case e := <-events:
			got = append(got, e)
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out; got %#v", got)
		}
	}
	want := []Event{Started{"s1", "fake"}, Text{"hej"}, Done{SessionID: "s1", Cost: 0.01}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("events %#v, want %#v", got, want)
	}
	s.Close()
	select {
	case e := <-events:
		if _, ok := e.(Exited); !ok {
			t.Errorf("after Close: %#v, want Exited", e)
		}
	case <-time.After(5 * time.Second):
		t.Error("no Exited after Close")
	}
}

func TestMCPPassesToolCallsToSkrin(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "s.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := Serve(ctx, sock, func(r Request) Response {
		return Response{Text: r.Tool + ": " + r.Args["query"].(string)}
	})
	if err != nil {
		t.Fatal(err)
	}
	in, feed := io.Pipe()
	out, read := io.Pipe()
	go func() { RunMCP(in, read, sock, "0.8.0"); read.Close() }()
	replies := bufio.NewScanner(out)
	call := func(msg string) map[string]any {
		t.Helper()
		if _, err := io.WriteString(feed, msg+"\n"); err != nil {
			t.Fatal(err)
		}
		if !replies.Scan() {
			t.Fatal("no reply")
		}
		var r map[string]any
		if err := json.Unmarshal(replies.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	init := call(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	if init["result"].(map[string]any)["protocolVersion"] != "2025-06-18" {
		t.Errorf("initialize: %v", init)
	}
	io.WriteString(feed, `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n")
	list := call(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if n := len(list["result"].(map[string]any)["tools"].([]any)); n != len(Tools) {
		t.Errorf("tools/list gave %d tools, want %d", n, len(Tools))
	}
	res := call(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"vault_search","arguments":{"query":"#idea"}}}`)
	content := res["result"].(map[string]any)["content"].([]any)[0].(map[string]any)
	if content["text"] != "vault_search: #idea" {
		t.Errorf("tools/call: %v", res)
	}
	feed.Close()
}

func TestSystemPromptNamesTheVault(t *testing.T) {
	p := SystemPrompt(Vault{Name: "vault-1", DailyFolder: "Daily", DailyFormat: "YYYY-MM-DD", DailyTemplate: "Templates/Daily template.md"})
	for _, want := range []string{`"vault-1"`, "Daily", "YYYY-MM-DD", "propose_edit", "<open-note>"} {
		if !strings.Contains(p, want) {
			t.Errorf("system prompt lacks %q", want)
		}
	}
}
