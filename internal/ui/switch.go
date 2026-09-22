package ui

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/config"
	"github.com/lurioso/skrin/internal/obsidian"
	"github.com/lurioso/skrin/internal/vault"
)

// openSwitchVault lists the vaults to switch to: Obsidian's registered
// vaults, plus any folder next to the current vault that looks like a
// vault (has its own .obsidian or Skrin's .skrin marker). Above them sits
// "Open a folder as a Skrin…", which opens any folder at all. It is a
// palette command, not a key: switching vaults is a rare, deliberate
// thing, so it stays out of the way until someone asks for it.
func (m *Model) openSwitchVault() {
	type candidate struct{ path, name string }

	seen := map[string]bool{}
	var vaults []candidate
	add := func(path string) {
		if path == "" {
			return
		}
		key := filepath.Clean(path)
		if seen[key] {
			return
		}
		seen[key] = true
		vaults = append(vaults, candidate{path: key, name: filepath.Base(key)})
	}

	// The vault open now, always shown, so the list says where you are
	// even when no registry or marker names it.
	add(m.vault.Root)

	// Obsidian's own list, the vaults it knows about.
	if reg, err := obsidian.LoadRegistry(config.ObsidianRegistry()); err == nil {
		for _, e := range reg.Vaults {
			add(e.Path)
		}
	}

	// Siblings of the current vault that look like vaults, so a vault
	// Obsidian hasn't opened yet (a new one sitting next to this one) is
	// still reachable. "Looks like" means a .obsidian or a .skrin marker.
	parent := filepath.Dir(filepath.Clean(m.vault.Root))
	if entries, err := os.ReadDir(parent); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if p := filepath.Join(parent, e.Name()); vault.LooksLikeVault(p) {
				add(p)
			}
		}
	}

	sort.Slice(vaults, func(i, j int) bool {
		return strings.ToLower(vaults[i].name) < strings.ToLower(vaults[j].name)
	})

	cur := filepath.Clean(m.vault.Root)
	var items []choice
	items = append(items, choice{
		label: "Open a folder as a Skrin…",
		do:    m.startOpenFolder,
	})
	for _, vv := range vaults {
		detail := ""
		if vv.path == cur {
			detail = "open now"
		}
		path := vv.path
		items = append(items, choice{
			label:  vv.name,
			detail: detail,
			run: func() tea.Cmd {
				if !m.switchVaultTo(path) {
					return nil
				}
				return tea.Quit
			},
		})
	}
	m.openChooser(&chooser{
		title:  "Open a vault",
		prompt: "Vault",
		empty:  "No vault by that name",
		verb:   "open",
		items:  items,
	})
}

// startOpenFolder asks for a folder path to open as a Skrin. The prompt is
// free text: any folder at all, not just ones already known.
func (m *Model) startOpenFolder() {
	p := &prompt{kind: promptOpenFolder, label: "Open this folder as a Skrin"}
	p.in.set(m.vault.Root + string(filepath.Separator))
	m.prompt = p
}

// openFolderAsSkrin opens any folder as a vault: it writes Skrin's marker
// (a .skrin directory) so the folder is recognised as a vault next time,
// remembers it as the default, and asks to quit so main reopens there. A
// folder that isn't a directory is refused; one that is, is adopted on the
// spot — nothing about its contents is touched beyond the marker.
func (m *Model) openFolderAsSkrin(input string) error {
	root := strings.TrimSpace(input)
	if root == "" {
		return errors.New("no folder given")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return errors.New("that's not a folder")
	}
	// Write the marker if it isn't there, so discovery finds it next time.
	if !vault.LooksLikeVault(abs) {
		if err := os.Mkdir(filepath.Join(abs, vault.MarkerDir), 0o755); err != nil && !os.IsExist(err) {
			return err
		}
	}
	m.switchTo = abs
	m.opts.Config.SetVault(abs)
	if err := config.Save(m.opts.Config); err != nil {
		m.switchTo = ""
		return err
	}
	return nil
}

// switchVaultTo remembers the chosen vault as Skrin's default and asks to
// quit, so main can reopen in the new vault. Saving first means the
// switch survives even if the relaunch is interrupted. It reports whether
// the save held, so a failed save doesn't quit and lose the choice.
func (m *Model) switchVaultTo(path string) bool {
	m.opts.Config.SetVault(path)
	if err := config.Save(m.opts.Config); err != nil {
		m.flash = "couldn't save settings: " + err.Error()
		return false
	}
	m.switchTo = path
	return true
}

// SwitchTo is the vault main should reopen after Skrin quits, or "".
func (m *Model) SwitchTo() string { return m.switchTo }
