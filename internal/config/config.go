// Package config loads Skrin's settings and finds the vault to open.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/lurioso/skrin/internal/obsidian"
)

// Config mirrors ~/.config/skrin/config.toml.
type Config struct {
	Vault string `toml:"vault"`
	// rawVault and rawExternal keep the file's own spelling (e.g. a "~/"
	// shorthand) so Save doesn't turn it into an absolute path just
	// because Load expanded it for use at runtime.
	rawVault    string `toml:"-"`
	rawExternal string `toml:"-"`
	General     struct {
		RestoreLastNote *bool `toml:"restore_last_note"` // unset means off: fresh runs start at the welcome screen
	} `toml:"general"`
	Daily struct {
		RolloverTodos *bool `toml:"rollover_todos"` // unset means on
	} `toml:"daily"`
	Editor struct {
		Vim      bool   `toml:"vim"`      // vim-style keys in the built-in editor
		External string `toml:"external"` // command for E; default $VISUAL, $EDITOR, nvim
	} `toml:"editor"`
	Assistant struct {
		Enabled  *bool  `toml:"enabled"`  // unset means on
		Position string `toml:"position"` // "bottom" (the default) or "right"
		Model    string `toml:"model"`    // "" for Claude Code's default
		Command  string `toml:"command"`  // the claude binary; default claude on $PATH
	} `toml:"assistant"`
	Library struct {
		Folder        string `toml:"folder"`         // default: "Books"
		CoversFolder  string `toml:"covers_folder"`  // default: "Assets/Covers"
		DefaultStatus string `toml:"default_status"` // default: "reading"
	} `toml:"library"`
	Render struct {
		Images *bool `toml:"images"` // unset means on
	} `toml:"render"`
	// Keys is the user's own keymap, written by the Keys tab of `?`:
	// context ("main", "editor", …) → action name → the keys that work
	// for it. Only what was changed is here, so new default keys still
	// arrive with a new Skrin.
	Keys map[string]map[string][]string `toml:"keys"`
}

// AssistantEnabled reports whether the Claude drawer is available. It is on
// unless turned off.
func (c Config) AssistantEnabled() bool {
	return c.Assistant.Enabled == nil || *c.Assistant.Enabled
}

// RolloverTodos reports whether `t` carries unfinished todos into a new
// daily note. It is on unless turned off.
func (c Config) RolloverTodos() bool {
	return c.Daily.RolloverTodos == nil || *c.Daily.RolloverTodos
}

// RestoreLastNote reports whether a fresh run reopens the note that was
// open when Skrin last quit. It is off unless turned on: the default is a
// clean welcome screen, so a vault with private notes never opens one by
// surprise.
func (c Config) RestoreLastNote() bool {
	return c.General.RestoreLastNote != nil && *c.General.RestoreLastNote
}

// LibraryFolder is where new book notes are created: Books, unless set.
func (c Config) LibraryFolder() string {
	if c.Library.Folder != "" {
		return c.Library.Folder
	}
	return "Books"
}

// LibraryCoversFolder is where cover images are saved: Assets/Covers,
// unless set.
func (c Config) LibraryCoversFolder() string {
	if c.Library.CoversFolder != "" {
		return c.Library.CoversFolder
	}
	return "Assets/Covers"
}

// LibraryDefaultStatus is the status a new Book Card starts with: reading,
// unless set.
func (c Config) LibraryDefaultStatus() string {
	if c.Library.DefaultStatus != "" {
		return c.Library.DefaultStatus
	}
	return "reading"
}

// RenderImages reports whether image embeds may show a block-art preview
// of the file. It is on unless turned off; either way, a found embed's
// name, dimensions and size still show in the placeholder — this only
// gates whether a preview is ever attempted.
func (c Config) RenderImages() bool {
	return c.Render.Images == nil || *c.Render.Images
}

// Load reads the config file. A missing file is not an error.
func Load() (Config, error) {
	var c Config
	path := Path()
	if _, err := toml.DecodeFile(path, &c); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return c, nil
		}
		return c, fmt.Errorf("reading %s: %w", path, err)
	}
	c.rawVault, c.rawExternal = c.Vault, c.Editor.External
	c.Vault = expandHome(c.Vault)
	c.Editor.External = expandHome(c.Editor.External)
	return c, nil
}

// Path is where config.toml lives: ~/.config/skrin/config.toml, unless
// XDG_CONFIG_HOME says otherwise.
func Path() string {
	return filepath.Join(configHome(), "skrin", "config.toml")
}

// Save writes c to config.toml, for the settings screen (`?`, Settings
// tab) to persist a toggle. It keeps the file's own spelling of vault and
// editor.external (e.g. a "~/" shorthand) rather than the expanded paths
// Load hands to the rest of Skrin.
func Save(c Config) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out := c
	out.Vault, out.Editor.External = c.rawVault, c.rawExternal
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(out); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ObsidianRegistry is where Obsidian keeps its list of vaults.
func ObsidianRegistry() string {
	return filepath.Join(configHome(), "obsidian", "obsidian.json")
}

// DiscoverVault asks Obsidian's vault list which vault to use: the one
// open now, or else the most recently used.
func DiscoverVault() (string, error) {
	path := ObsidianRegistry()
	reg, err := obsidian.LoadRegistry(path)
	if err != nil {
		return "", fmt.Errorf("no vault given and Obsidian's vault list is unreadable: %w", err)
	}
	if p, ok := reg.Preferred(); ok {
		return p, nil
	}
	return "", fmt.Errorf("no vault given and %s lists none", path)
}

func configHome() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}

func expandHome(p string) string {
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, rest)
	}
	return p
}
