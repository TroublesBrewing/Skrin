// Package config loads Skrin's settings and finds the vault to open.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/lurioso/skrin/internal/obsidian"
)

// Config mirrors ~/.config/skrin/config.toml. Everything is omitempty, so
// Save writes back only what is actually set rather than filling the
// user's config with empty keys for every setting Skrin has. A *bool that
// was deliberately set to false still writes: only an unset one is left
// out, which is what keeps "off" apart from "not chosen".
type Config struct {
	Vault string `toml:"vault,omitempty"`
	// rawVault and rawExternal keep the file's own spelling (e.g. a "~/"
	// shorthand) so Save doesn't turn it into an absolute path just
	// because Load expanded it for use at runtime.
	rawVault    string `toml:"-"`
	rawExternal string `toml:"-"`
	General     struct {
		RestoreLastNote *bool `toml:"restore_last_note,omitempty"` // unset means off: fresh runs start at the welcome screen
		InstantOpen     *bool `toml:"instant_open,omitempty"`      // unset means on: moving the Files cursor opens the note under it
	} `toml:"general,omitempty"`
	Daily struct {
		RolloverTodos *bool `toml:"rollover_todos,omitempty"` // unset means on
	} `toml:"daily,omitempty"`
	Editor struct {
		Vim      bool   `toml:"vim,omitempty"`      // vim-style keys in the built-in editor
		External string `toml:"external,omitempty"` // command for E; default $VISUAL, $EDITOR, nvim
	} `toml:"editor,omitempty"`
	Assistant struct {
		Enabled  *bool  `toml:"enabled,omitempty"`  // unset means on
		Position string `toml:"position,omitempty"` // "bottom" (the default) or "right"
		Model    string `toml:"model,omitempty"`    // "" for Claude Code's default
		Command  string `toml:"command,omitempty"`  // the claude binary; default claude on $PATH
	} `toml:"assistant,omitempty"`
	Templates struct {
		Folder string `toml:"folder,omitempty"` // unset: Obsidian's own Templates folder
	} `toml:"templates,omitempty"`
	Theme struct {
		Builtin string `toml:"builtin,omitempty"` // "gruvbox" (the default) or "flexoki-light", used when there is no system theme
	} `toml:"theme,omitempty"`
	Beta struct {
		Enabled *bool `toml:"enabled,omitempty"` // unset means off: experiments are asked for, never assumed
	} `toml:"beta,omitempty"`
	Paper struct {
		Enabled  *bool `toml:"enabled,omitempty"`  // unset means off
		Strength int   `toml:"strength,omitempty"` // how far the grain moves from the background, 1-20; unset means 6
	} `toml:"paper,omitempty"`
	Habits struct {
		Enabled *bool `toml:"enabled,omitempty"` // unset means off: the habits view is asked for, never assumed
	} `toml:"habits,omitempty"`
	Library struct {
		Folder        string `toml:"folder,omitempty"`         // default: "Books"
		CoversFolder  string `toml:"covers_folder,omitempty"`  // default: "Assets/Covers"
		DefaultStatus string `toml:"default_status,omitempty"` // default: "reading"
	} `toml:"library,omitempty"`
	Render struct {
		Images        *bool `toml:"images,omitempty"`         // unset means on
		LineNumbers   *bool `toml:"line_numbers,omitempty"`   // unset means off: notes and editor start without line numbers
		ReadableWidth *bool `toml:"readable_width,omitempty"` // unset means on: notes stay at most 80 characters wide
		Spreads       *bool `toml:"spreads,omitempty"`        // unset means on
	} `toml:"render,omitempty"`
	// Keys is the user's own keymap, written by the Keys tab of `?`:
	// context ("main", "editor", …) → action name → the keys that work
	// for it. Only what was changed is here, so new default keys still
	// arrive with a new Skrin.
	Keys map[string]map[string][]string `toml:"keys,omitempty"`
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

// InstantOpen reports whether moving the Files cursor opens the note under
// it, Skrin's own way since v0.1. It is on unless turned off; off is
// Obsidian's way — the note pane only changes when a note is opened
// explicitly (l/→ to read, Enter to edit), and stays on whatever was open
// last otherwise.
func (c Config) InstantOpen() bool {
	return c.General.InstantOpen == nil || *c.General.InstantOpen
}

// ThemeBuiltin is the palette Skrin uses when there is no system theme to
// follow: gruvbox unless another built-in is named.
func (c Config) ThemeBuiltin() string {
	if c.Theme.Builtin == "" {
		return "gruvbox"
	}
	return c.Theme.Builtin
}

// BetaEnabled reports whether Settings shows its beta block, where the
// experiments live. Off unless asked for, and ignored altogether in a
// build with version.Beta false.
func (c Config) BetaEnabled() bool {
	return c.Beta.Enabled != nil && *c.Beta.Enabled
}

// PaperEnabled reports whether the note gets its paper grain.
func (c Config) PaperEnabled() bool {
	return c.Paper.Enabled != nil && *c.Paper.Enabled
}

// PaperStrength is how far the grain's shades sit from the background, in
// steps of 0-255. Below about four nothing is visible on a light theme:
// three steps out of 255 is roughly one per cent, which a screen doesn't
// show. Out-of-range values are clamped rather than refused, since a
// number in a config file should never stop Skrin starting.
func (c Config) PaperStrength() int {
	if c.Paper.Strength == 0 {
		return 6
	}
	return min(max(c.Paper.Strength, 1), 20)
}

// HabitsEnabled reports whether the habits view is switched on. It is off unless
// asked for: habit tracking isn't what a vault is for, so it stays out of
// the way — no key, no palette row, no tip — until someone wants it. The
// checkboxes under "### Habits" in a daily note are plain markdown and go
// on working either way, including the rule that keeps them out of
// tomorrow's todos.
func (c Config) HabitsEnabled() bool {
	return c.Habits.Enabled != nil && *c.Habits.Enabled
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

// RenderReadableWidth reports whether notes keep a readable line length,
// at most 80 characters wide and centred in their pane, as in zen and in
// Obsidian's "Readable line length". It is on unless turned off.
func (c Config) RenderReadableWidth() bool {
	return c.Render.ReadableWidth == nil || *c.Render.ReadableWidth
}

// RenderLineNumbers reports whether line numbers show along the left edge
// of notes and the editor. It is off unless turned on.
func (c Config) RenderLineNumbers() bool {
	return c.Render.LineNumbers != nil && *c.Render.LineNumbers
}

// RenderSpreads reports whether ```spread and ```dataview blocks show their
// answer rather than their query. It is on unless turned off.
func (c Config) RenderSpreads() bool {
	return c.Render.Spreads == nil || *c.Render.Spreads
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

// ObsidianRegistry is where Obsidian keeps its list of vaults: in the
// system's own settings folder, which differs from one system to another.
func ObsidianRegistry() string {
	home, _ := os.UserHomeDir()
	return registryPath(runtime.GOOS, home, os.Getenv("XDG_CONFIG_HOME"), os.Getenv("APPDATA"))
}

func registryPath(goos, home, xdgConfig, appData string) string {
	switch goos {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "obsidian", "obsidian.json")
	case "windows":
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "obsidian", "obsidian.json")
	}
	if xdgConfig == "" {
		xdgConfig = filepath.Join(home, ".config")
	}
	return filepath.Join(xdgConfig, "obsidian", "obsidian.json")
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
