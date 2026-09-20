// Package session remembers where you were in a vault between runs: the
// open folders in Files, the cursor, and the open note with its scroll
// position. It lives under $XDG_STATE_HOME/skrin/session/, never in the
// vault.
package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

// State is where you were in one vault.
type State struct {
	Expanded []string `json:"expanded"` // open folders, vault-relative
	Cursor   string   `json:"cursor"`   // the row under the cursor in Files
	Open     string   `json:"open"`     // the open note, if any
	Offset   int      `json:"offset"`   // how far down the open note was scrolled
	Claude   string   `json:"claude"`   // the Claude conversation to resume
	Recent   []string `json:"recent"`   // notes opened lately, newest first
	Pinned   []string `json:"pinned"`   // the notes you pinned, in name order
	Drawer   string   `json:"drawer"`   // "bottom" or "right" once flipped with Alt-p; "" for the config's choice
}

func file(root string) string {
	state := os.Getenv("XDG_STATE_HOME")
	if state == "" {
		home, _ := os.UserHomeDir()
		state = filepath.Join(home, ".local", "state")
	}
	sum := sha256.Sum256([]byte(root))
	return filepath.Join(state, "skrin", "session", hex.EncodeToString(sum[:6])+".json")
}

// Load returns the state saved for the vault at root. A missing or broken
// file gives the zero State: a fresh start.
func Load(root string) State {
	var s State
	if data, err := os.ReadFile(file(root)); err == nil {
		_ = json.Unmarshal(data, &s)
	}
	return s
}

// Save stores the state for the vault at root, atomically.
func Save(root string, s State) error {
	p := file(root)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
