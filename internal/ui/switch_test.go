package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lurioso/skrin/internal/config"
	"github.com/lurioso/skrin/internal/vault"
)

func TestOpenAVaultListsObsidiansVaults(t *testing.T) {
	m := newTestModel(t)
	dir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "obsidian")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	reg := `{"vaults":{"a":{"path":"/home/u/vault-a","ts":1,"open":true},"b":{"path":"/home/u/vault-b","ts":2,"open":false}}}`
	if err := os.WriteFile(config.ObsidianRegistry(), []byte(reg), 0o644); err != nil {
		t.Fatal(err)
	}

	press(m, "ctrl+p")
	typeText(m, "open a vault")
	press(m, "enter")

	if m.chooser == nil || m.chooser.title != "Open a vault" {
		t.Fatalf("Open a vault should list the vaults (chooser %+v)", m.chooser)
	}
	labels := paletteLabels(m)
	if !has(labels, "vault-a") || !has(labels, "vault-b") {
		t.Errorf("vaults listed: %v", labels)
	}
	if !has(labels, "Open a folder as a Skrin…") {
		t.Errorf("the list should lead with 'Open a folder as a Skrin…': %v", labels)
	}
	// Exactly one entry is the vault open now, marked as such.
	var marked int
	for _, it := range m.chooser.items {
		if it.detail == "open now" {
			marked++
		}
	}
	if marked != 1 {
		t.Errorf("one vault should be marked open now, got %d", marked)
	}
}

func has(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// A vault sitting next to the current one, with its own .obsidian, shows
// up even when Obsidian's registry doesn't know it.
func TestOpenAVaultFindsASiblingVault(t *testing.T) {
	m := newTestModel(t)
	dir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "obsidian")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	root := m.vault.Root
	reg := `{"vaults":{"cur":{"path":` + jsonStr(root) + `,"ts":1,"open":true}}}`
	if err := os.WriteFile(config.ObsidianRegistry(), []byte(reg), 0o644); err != nil {
		t.Fatal(err)
	}
	sibling := filepath.Join(filepath.Dir(root), "Trelian-Minn")
	if err := os.MkdirAll(filepath.Join(sibling, ".obsidian"), 0o755); err != nil {
		t.Fatal(err)
	}

	press(m, "ctrl+p")
	typeText(m, "open a vault")
	press(m, "enter")

	if m.chooser == nil || m.chooser.title != "Open a vault" {
		t.Fatal("Open a vault should open")
	}
	found := false
	for _, l := range paletteLabels(m) {
		if l == "Trelian-Minn" {
			found = true
		}
	}
	if !found {
		t.Errorf("the sibling vault should be listed, got %v", paletteLabels(m))
	}
}

func TestOpenAVaultSavesAndAsksToQuit(t *testing.T) {
	m := newTestModel(t)
	dir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "obsidian")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	reg := `{"vaults":{"a":{"path":"/home/u/vault-a","ts":1,"open":true}}}`
	if err := os.WriteFile(config.ObsidianRegistry(), []byte(reg), 0o644); err != nil {
		t.Fatal(err)
	}

	press(m, "ctrl+p")
	typeText(m, "open a vault")
	press(m, "enter")

	typeText(m, "vault-a")
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

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// "Open a folder as a Skrin…" adopts any folder at all: it writes Skrin's
// .skrin marker, so the folder is recognised as a vault next time, and
// asks to reopen there. A non-directory path is refused.
func TestOpenAFolderAsASkrinWritesTheMarker(t *testing.T) {
	m := newTestModel(t)

	target := t.TempDir()
	// A plain folder, no .obsidian, no marker: the real "open any folder".
	if err := m.openFolderAsSkrin(target); err != nil {
		t.Fatalf("openFolderAsSkrin: %v", err)
	}
	if !vault.LooksLikeVault(target) {
		t.Error("the folder should now carry Skrin's marker")
	}
	if fi, err := os.Stat(filepath.Join(target, vault.MarkerDir)); err != nil || !fi.IsDir() {
		t.Errorf("the .skrin marker directory should exist: %v", err)
	}
	if m.SwitchTo() != target {
		t.Errorf("SwitchTo = %q, want %q", m.SwitchTo(), target)
	}
}

func TestOpenAFolderAsASkrinRefusesANonDirectory(t *testing.T) {
	m := newTestModel(t)
	// A regular file, not a folder.
	f := filepath.Join(t.TempDir(), "note.md")
	if err := os.WriteFile(f, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := m.openFolderAsSkrin(f); err == nil {
		t.Error("a file should be refused as a vault")
	}
	if m.SwitchTo() != "" {
		t.Errorf("SwitchTo should stay empty on a refusal, got %q", m.SwitchTo())
	}
}
