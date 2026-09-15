package assistant

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"sync"
)

// Request is a tool call from Claude, passed on by `skrin mcp` to the
// running Skrin.
type Request struct {
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
}

// Response is Skrin's answer, handed back to Claude as the tool's result.
type Response struct {
	Text  string `json:"text"`
	Error bool   `json:"error,omitempty"`
}

// Tool describes one of Skrin's tools to Claude.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func object(props map[string]any, required ...string) map[string]any {
	o := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		o["required"] = required
	}
	return o
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }

// Tools are Skrin's tools as Claude sees them. Every change goes through
// the user: the propose_ tools wait until they have said yes or no.
var Tools = []Tool{
	{"vault_search", "Search the vault with Obsidian's search syntax: words (all must match), \"exact phrases\", -exclude, OR, #tag or tag:#tag, [property] or [property:value], path:Folder and file:name. Returns the matching notes and lines.",
		object(map[string]any{"query": str("the search, e.g. #idea [status:draft] stoic")}, "query")},
	{"current_context", "What the user has open in Skrin right now: the open note with its full text, any highlighted text, and the row under the cursor in Files.",
		object(map[string]any{})},
	{"propose_create", "Propose a new note. Skrin shows it to the user, who approves or rejects it; the result says which.",
		object(map[string]any{"path": str("vault-relative path, e.g. Ideas/Garden.md"), "content": str("the note's full text")}, "path", "content")},
	{"propose_edit", "Propose an edit to a note: old_text, which must appear exactly once in it, becomes new_text. Skrin shows the user a diff; they approve or reject it.",
		object(map[string]any{"path": str("vault-relative path of the note"), "old_text": str("the exact text to replace; include enough to be unique"), "new_text": str("what replaces it")}, "path", "old_text", "new_text")},
	{"propose_move", "Propose moving or renaming a note or folder. Links to it are updated as Obsidian would. The user approves or rejects it.",
		object(map[string]any{"from": str("vault-relative path now"), "to": str("vault-relative path after the move")}, "from", "to")},
	{"propose_delete", "Propose deleting a note or folder (to the trash, so it can be restored). The user approves or rejects it.",
		object(map[string]any{"path": str("vault-relative path")}, "path")},
}

// Serve answers tool calls on a Unix socket at path until ctx ends. handle
// may block, say until the user has decided about a change.
func Serve(ctx context.Context, path string, handle func(Request) Response) error {
	_ = os.Remove(path)
	l, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	_ = os.Chmod(path, 0o600)
	go func() {
		<-ctx.Done()
		l.Close()
	}()
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				var req Request
				if json.NewDecoder(c).Decode(&req) != nil {
					return
				}
				_ = json.NewEncoder(c).Encode(handle(req))
			}()
		}
	}()
	return nil
}

// Call sends one request to the Skrin listening at path.
func Call(path string, req Request) (Response, error) {
	c, err := net.Dial("unix", path)
	if err != nil {
		return Response{}, err
	}
	defer c.Close()
	if err := json.NewEncoder(c).Encode(req); err != nil {
		return Response{}, err
	}
	var resp Response
	err = json.NewDecoder(c).Decode(&resp)
	return resp, err
}

// MCPConfig is the --mcp-config JSON that makes Claude start `exe mcp`
// talking to the socket.
func MCPConfig(exe, socket string) string {
	b, _ := json.Marshal(map[string]any{"mcpServers": map[string]any{
		"skrin": map[string]any{"type": "stdio", "command": exe, "args": []string{"mcp", "--socket", socket}},
	}})
	return string(b)
}

// RunMCP is `skrin mcp`: an MCP server on in and out (JSON-RPC, one
// message per line) that passes each tool call on to the Skrin listening
// on socket. Calls run side by side, since a proposal can wait a long time
// for the user.
func RunMCP(in io.Reader, out io.Writer, socket, version string) error {
	var mu sync.Mutex
	var wg sync.WaitGroup
	enc := json.NewEncoder(out)
	reply := func(id json.RawMessage, result any, rpcErr map[string]any) {
		msg := map[string]any{"jsonrpc": "2.0", "id": id}
		if rpcErr != nil {
			msg["error"] = rpcErr
		} else {
			msg["result"] = result
		}
		mu.Lock()
		defer mu.Unlock()
		_ = enc.Encode(msg)
	}
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	for sc.Scan() {
		var msg struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(sc.Bytes(), &msg) != nil || len(msg.ID) == 0 {
			continue // not for us, or a notification
		}
		switch msg.Method {
		case "initialize":
			var p struct {
				ProtocolVersion string `json:"protocolVersion"`
			}
			_ = json.Unmarshal(msg.Params, &p)
			if p.ProtocolVersion == "" {
				p.ProtocolVersion = "2025-06-18"
			}
			reply(msg.ID, map[string]any{
				"protocolVersion": p.ProtocolVersion,
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "skrin", "version": version},
			}, nil)
		case "ping":
			reply(msg.ID, map[string]any{}, nil)
		case "tools/list":
			reply(msg.ID, map[string]any{"tools": Tools}, nil)
		case "tools/call":
			var p struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			_ = json.Unmarshal(msg.Params, &p)
			wg.Add(1)
			go func(id json.RawMessage) {
				defer wg.Done()
				resp, err := Call(socket, Request{p.Name, p.Arguments})
				if err != nil {
					resp = Response{Text: "Skrin isn't reachable: " + err.Error(), Error: true}
				}
				reply(id, map[string]any{
					"content": []any{map[string]any{"type": "text", "text": resp.Text}},
					"isError": resp.Error,
				}, nil)
			}(msg.ID)
		default:
			reply(msg.ID, nil, map[string]any{"code": -32601, "message": "method not found: " + msg.Method})
		}
	}
	wg.Wait()
	return sc.Err()
}
