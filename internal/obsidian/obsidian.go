// Package obsidian reads Obsidian's own settings: its list of vaults and
// the per-vault configuration in .obsidian/. Skrin only ever reads these.
package obsidian

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// VaultEntry is one vault in Obsidian's registry (obsidian.json).
type VaultEntry struct {
	Path string `json:"path"`
	TS   int64  `json:"ts"`
	Open bool   `json:"open"`
}

// Registry is Obsidian's list of known vaults.
type Registry struct {
	Vaults map[string]VaultEntry `json:"vaults"`
}

// LoadRegistry reads obsidian.json.
func LoadRegistry(path string) (Registry, error) {
	var r Registry
	data, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return r, fmt.Errorf("parsing %s: %w", path, err)
	}
	return r, nil
}

// Preferred returns the vault Obsidian has open, or else the most recently
// used one.
func (r Registry) Preferred() (string, bool) {
	best, bestTS, bestOpen := "", int64(-1), false
	for _, v := range r.Vaults {
		if v.Path == "" {
			continue
		}
		if (v.Open && !bestOpen) || (v.Open == bestOpen && v.TS > bestTS) {
			best, bestTS, bestOpen = v.Path, v.TS, v.Open
		}
	}
	return best, best != ""
}

// IsOpen reports whether Obsidian marks the vault at root as open.
func (r Registry) IsOpen(root string) bool {
	for _, v := range r.Vaults {
		if v.Open && filepath.Clean(v.Path) == filepath.Clean(root) {
			return true
		}
	}
	return false
}

// Running reports whether Obsidian desktop is running.
func Running() bool { return running("/proc") }

// running scans a /proc-style directory. Obsidian runs either as its own
// binary (AppImage, .deb, Flatpak) or as system Electron loading
// .../obsidian/app.asar (the Arch package).
func running(proc string) bool {
	entries, err := os.ReadDir(proc)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.Trim(e.Name(), "0123456789") != "" {
			continue
		}
		dir := filepath.Join(proc, e.Name())
		if comm, err := os.ReadFile(filepath.Join(dir, "comm")); err == nil &&
			strings.EqualFold(strings.TrimSpace(string(comm)), "obsidian") {
			return true
		}
		cmdline, err := os.ReadFile(filepath.Join(dir, "cmdline"))
		if err != nil || len(cmdline) == 0 {
			continue
		}
		args := strings.Split(strings.TrimRight(string(cmdline), "\x00"), "\x00")
		if strings.EqualFold(filepath.Base(args[0]), "obsidian") {
			return true
		}
		for _, a := range args[1:] {
			a = strings.ToLower(a)
			if strings.HasSuffix(a, "obsidian/app.asar") || strings.HasSuffix(a, "obsidian/resources/app.asar") {
				return true
			}
		}
	}
	return false
}

// DailyNotes mirrors the core Daily notes plugin settings.
type DailyNotes struct {
	Folder   string // vault-relative, no leading or trailing slash
	Format   string // moment.js format of the file name
	Template string // vault path of the template, as Obsidian stored it
}

// Rollover mirrors the Rollover Daily Todos community plugin's settings.
type Rollover struct {
	Installed         bool // listed in community-plugins.json
	TemplateHeading   string
	DeleteOnComplete  bool
	RemoveEmptyTodos  bool
	RolloverChildren  bool
	DoneStatusMarkers string
}

// Settings are the parts of a vault's .obsidian/ configuration Skrin uses.
type Settings struct {
	TrashOption string // "system", "local" or "none"
	Daily       DailyNotes
	Rollover    Rollover
}

// RolloverPluginID is the Rollover Daily Todos plugin's id.
const RolloverPluginID = "obsidian-rollover-daily-todos"

// LoadSettings reads a vault's settings. Missing or unreadable files leave
// Obsidian's (and the plugin's) defaults in place.
func LoadSettings(root string) Settings {
	s := Settings{
		TrashOption: "system",
		Daily:       DailyNotes{Format: "YYYY-MM-DD"},
		Rollover:    Rollover{TemplateHeading: "none", DoneStatusMarkers: "xX-"},
	}
	dir := filepath.Join(root, ".obsidian")

	var app struct {
		TrashOption string `json:"trashOption"`
	}
	if readJSON(filepath.Join(dir, "app.json"), &app) && app.TrashOption != "" {
		s.TrashOption = app.TrashOption
	}

	var dn struct {
		Folder   string `json:"folder"`
		Format   string `json:"format"`
		Template string `json:"template"`
	}
	if readJSON(filepath.Join(dir, "daily-notes.json"), &dn) {
		s.Daily.Folder = strings.Trim(dn.Folder, "/")
		if dn.Format != "" {
			s.Daily.Format = dn.Format
		}
		s.Daily.Template = dn.Template
	}

	var plugins []string
	if readJSON(filepath.Join(dir, "community-plugins.json"), &plugins) {
		s.Rollover.Installed = slices.Contains(plugins, RolloverPluginID)
	}
	var ro struct {
		TemplateHeading   *string `json:"templateHeading"`
		DeleteOnComplete  *bool   `json:"deleteOnComplete"`
		RemoveEmptyTodos  *bool   `json:"removeEmptyTodos"`
		RolloverChildren  *bool   `json:"rolloverChildren"`
		DoneStatusMarkers *string `json:"doneStatusMarkers"`
	}
	if readJSON(filepath.Join(dir, "plugins", RolloverPluginID, "data.json"), &ro) {
		if ro.TemplateHeading != nil {
			s.Rollover.TemplateHeading = *ro.TemplateHeading
		}
		if ro.DeleteOnComplete != nil {
			s.Rollover.DeleteOnComplete = *ro.DeleteOnComplete
		}
		if ro.RemoveEmptyTodos != nil {
			s.Rollover.RemoveEmptyTodos = *ro.RemoveEmptyTodos
		}
		if ro.RolloverChildren != nil {
			s.Rollover.RolloverChildren = *ro.RolloverChildren
		}
		if ro.DoneStatusMarkers != nil && *ro.DoneStatusMarkers != "" {
			s.Rollover.DoneStatusMarkers = *ro.DoneStatusMarkers
		}
	}
	return s
}

func readJSON(path string, v any) bool {
	data, err := os.ReadFile(path)
	return err == nil && json.Unmarshal(data, v) == nil
}
