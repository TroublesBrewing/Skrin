// Package config loads Skrin's settings and finds the vault to open.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config mirrors ~/.config/skrin/config.toml.
type Config struct {
	Vault string `toml:"vault"`
}

// Load reads the config file. A missing file is not an error.
func Load() (Config, error) {
	var c Config
	path := filepath.Join(configHome(), "skrin", "config.toml")
	if _, err := toml.DecodeFile(path, &c); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return c, nil
		}
		return c, fmt.Errorf("reading %s: %w", path, err)
	}
	c.Vault = expandHome(c.Vault)
	return c, nil
}

// DiscoverVault asks Obsidian's own vault registry which vault to use: the
// one currently open, or else the most recently used.
func DiscoverVault() (string, error) {
	return discoverFrom(filepath.Join(configHome(), "obsidian", "obsidian.json"))
}

func discoverFrom(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("no vault given and Obsidian's vault list is unreadable: %w", err)
	}
	var reg struct {
		Vaults map[string]struct {
			Path string `json:"path"`
			TS   int64  `json:"ts"`
			Open bool   `json:"open"`
		} `json:"vaults"`
	}
	if err := json.Unmarshal(data, &reg); err != nil {
		return "", fmt.Errorf("parsing %s: %w", path, err)
	}
	best, bestTS, bestOpen := "", int64(-1), false
	for _, v := range reg.Vaults {
		if v.Path == "" {
			continue
		}
		if (v.Open && !bestOpen) || (v.Open == bestOpen && v.TS > bestTS) {
			best, bestTS, bestOpen = v.Path, v.TS, v.Open
		}
	}
	if best == "" {
		return "", fmt.Errorf("no vault given and %s lists none", path)
	}
	return best, nil
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
