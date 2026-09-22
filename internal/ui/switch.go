package ui

import (
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/config"
	"github.com/lurioso/skrin/internal/obsidian"
)

// openSwitchVault lists Obsidian's registered vaults to switch to. It is
// a palette command, not a key: switching vaults is a rare, deliberate
// thing, so it stays out of the way until someone asks for it.
func (m *Model) openSwitchVault() {
	reg, err := obsidian.LoadRegistry(config.ObsidianRegistry())
	if err != nil {
		m.flash = "can't read Obsidian's vault list: " + err.Error()
		return
	}
	type vault struct{ path, name string }
	var vaults []vault
	for _, e := range reg.Vaults {
		if e.Path == "" {
			continue
		}
		vaults = append(vaults, vault{path: e.Path, name: filepath.Base(e.Path)})
	}
	sort.Slice(vaults, func(i, j int) bool {
		return strings.ToLower(vaults[i].name) < strings.ToLower(vaults[j].name)
	})
	if len(vaults) == 0 {
		m.flash = "Obsidian's vault list is empty"
		return
	}

	cur := filepath.Clean(m.vault.Root)
	var items []choice
	for _, vv := range vaults {
		detail := ""
		if filepath.Clean(vv.path) == cur {
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
		title:  "Switch vault",
		prompt: "Vault",
		empty:  "No vault by that name",
		verb:   "open",
		items:  items,
	})
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
