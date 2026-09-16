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

func TestLoadExpandsHomeInTheEditorCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	dir := filepath.Join(home, "skrin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[editor]\nexternal = \"~/bin/ed --wait\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, "bin/ed") + " --wait"; c.Editor.External != want {
		t.Errorf("external = %q, want %q", c.Editor.External, want)
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

func TestLibraryDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.LibraryFolder() != "Books" {
		t.Errorf("LibraryFolder() = %q, want Books", c.LibraryFolder())
	}
	if c.LibraryCoversFolder() != "Assets/Covers" {
		t.Errorf("LibraryCoversFolder() = %q, want Assets/Covers", c.LibraryCoversFolder())
	}
	if c.LibraryDefaultStatus() != "reading" {
		t.Errorf("LibraryDefaultStatus() = %q, want reading", c.LibraryDefaultStatus())
	}
}

func TestLibraryReadsOverrides(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	dir := filepath.Join(home, "skrin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	conf := "[library]\nfolder = \"Library\"\ncovers_folder = \"Covers\"\ndefault_status = \"want\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.LibraryFolder() != "Library" || c.LibraryCoversFolder() != "Covers" || c.LibraryDefaultStatus() != "want" {
		t.Errorf("library config = %+v", c.Library)
	}
}
