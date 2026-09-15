package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRolloverDefaultsOn(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !c.RolloverTodos() {
		t.Error("rollover should default to on")
	}
}

func TestLoadReadsVaultAndRollover(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	dir := filepath.Join(home, "skrin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	conf := "vault = \"/vaults/main\"\n\n[daily]\nrollover_todos = false\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Vault != "/vaults/main" || c.RolloverTodos() {
		t.Errorf("config = %+v, rollover %v", c, c.RolloverTodos())
	}
}

func TestDiscoverVaultUsesObsidianRegistry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "obsidian"), 0o755); err != nil {
		t.Fatal(err)
	}
	reg := `{"vaults":{"a":{"path":"/vaults/one","ts":1,"open":true}}}`
	if err := os.WriteFile(ObsidianRegistry(), []byte(reg), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := DiscoverVault(); err != nil || got != "/vaults/one" {
		t.Errorf("DiscoverVault = %q, %v", got, err)
	}
}
