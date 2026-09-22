package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lurioso/skrin/internal/config"
)

func TestSwitchVaultListsObsidiansVaults(t *testing.T) {
	m := newTestModel(t)
	// newTestModelWith points XDG_CONFIG_HOME at a temp dir; drop Obsidian's
	// registry there so the picker reads it.
	dir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "obsidian")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// One registry entry is the vault open now, so it gets the "open now"
	// mark; the other two are just other vaults.
	reg := `{"vaults":{"a":{"path":"/home/u/vault-a","ts":1,"open":true},"b":{"path":"/home/u/vault-b","ts":2,"open":false}}}`
	if err := os.WriteFile(config.ObsidianRegistry(), []byte(reg), 0o644); err != nil {
		t.Fatal(err)
	}

	press(m, "ctrl+p")
	typeText(m, "switch vault")
	press(m, "enter")

	if m.chooser == nil || m.chooser.title != "Switch vault" {
		t.Fatalf("Switch vault should list the vaults (chooser %+v)", m.chooser)
	}
	labels := paletteLabels(m)
	if len(labels) != 2 || labels[0] != "vault-a" || labels[1] != "vault-b" {
		t.Errorf("vaults listed: %v", labels)
	}
}

func TestSwitchVaultMarksTheOpenVault(t *testing.T) {
	m := newTestModel(t)
	dir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "obsidian")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	root := m.vault.Root
	reg := `{"vaults":{"cur":{"path":` + jsonStr(root) + `,"ts":1,"open":true},"other":{"path":"/home/u/vault-b","ts":2,"open":false}}}`
	if err := os.WriteFile(config.ObsidianRegistry(), []byte(reg), 0o644); err != nil {
		t.Fatal(err)
	}

	press(m, "ctrl+p")
	typeText(m, "switch vault")
	press(m, "enter")

	var marked int
	for _, it := range m.chooser.items {
		if it.detail == "open now" {
			marked++
		}
	}
	if marked != 1 {
		t.Errorf("the open vault should be marked open now, got %d", marked)
	}
}

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestSwitchVaultSavesAndAsksToQuit(t *testing.T) {
	m := newTestModel(t)
	// Set up a registry so the picker has a vault to choose.
	dir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "obsidian")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	reg := `{"vaults":{"a":{"path":"/home/u/vault-a","ts":1,"open":true}}}`
	if err := os.WriteFile(config.ObsidianRegistry(), []byte(reg), 0o644); err != nil {
		t.Fatal(err)
	}

	press(m, "ctrl+p")
	typeText(m, "switch vault")
	press(m, "enter")

	// Enter on the vault saves it as default and asks to quit.
	_, cmd := m.Update(key("enter"))
	if cmd == nil {
		t.Fatal("choosing a vault should ask to quit")
	}
	if m.SwitchTo() != "/home/u/vault-a" {
		t.Errorf("SwitchTo = %q, want the chosen vault", m.SwitchTo())
	}
	b, _ := os.ReadFile(config.Path())
	if !strings.Contains(string(b), "/home/u/vault-a") {
		t.Errorf("config.toml should name the new vault: %q", b)
	}
}
