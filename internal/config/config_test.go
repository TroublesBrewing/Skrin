package config

import (
	"os"
	"path/filepath"
	"strings"
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
	if !c.RenderImages() {
		t.Error("RenderImages should default to on")
	}
}

func TestRenderImagesReadsOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	dir := filepath.Join(home, "skrin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[render]\nimages = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.RenderImages() {
		t.Error("RenderImages() = true, want false per config")
	}
}

func TestRenderLineNumbers(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.RenderLineNumbers() {
		t.Error("RenderLineNumbers should default to off")
	}

	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	dir := filepath.Join(home, "skrin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[render]\nline_numbers = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if !c.RenderLineNumbers() {
		t.Error("RenderLineNumbers() = false, want true per config")
	}
}

func TestRenderSpreads(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !c.RenderSpreads() {
		t.Error("RenderSpreads should default to on")
	}

	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	dir := filepath.Join(home, "skrin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[render]\nspreads = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if c, err = Load(); err != nil {
		t.Fatal(err)
	}
	if c.RenderSpreads() {
		t.Error("RenderSpreads() = true, want false per config")
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

func TestRestoreLastNoteDefaultsOff(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.RestoreLastNote() {
		t.Error("RestoreLastNote should default to off: a fresh run starts at the welcome screen")
	}
}

func TestRestoreLastNoteReadsOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	dir := filepath.Join(home, "skrin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	conf := "[general]\nrestore_last_note = true\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !c.RestoreLastNote() {
		t.Error("restore_last_note = true should turn it on")
	}
}

func TestSaveRoundTripsAndKeepsHomeShorthand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	dir := filepath.Join(home, "skrin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	conf := "vault = \"~/notes\"\n\n[editor]\nexternal = \"~/bin/ed --wait\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	b := true
	c.General.RestoreLastNote = &b
	if err := Save(c); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.RestoreLastNote() {
		t.Error("Save then Load should round-trip the new setting")
	}
	if reloaded.Vault != filepath.Join(home, "notes") {
		t.Errorf("Load after Save: vault = %q", reloaded.Vault)
	}
	saved, err := os.ReadFile(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(saved), `vault = "~/notes"`) {
		t.Errorf("Save should keep the ~/ shorthand rather than an absolute path:\n%s", saved)
	}
}

// Save writes the settings screen's changes into a file the user also
// edits by hand, so it must not fill it with an empty key for every
// setting Skrin happens to have.
func TestSaveWritesOnlyWhatIsSet(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	off := false
	c.Render.Images = &off
	if err := Save(c); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	if got := string(saved); strings.Contains(got, `vault = ""`) || strings.Contains(got, `external = ""`) || strings.Contains(got, "[library]") {
		t.Errorf("Save should leave untouched settings out of the file:\n%s", got)
	}
	// But a setting deliberately turned off is not "unset", and has to
	// survive: dropping it would silently turn the thing back on.
	if !strings.Contains(string(saved), "images = false") {
		t.Errorf("an explicit off should be written:\n%s", saved)
	}
	back, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if back.RenderImages() {
		t.Error("images should still be off after a round trip")
	}
	if !back.RolloverTodos() || !back.AssistantEnabled() {
		t.Error("settings that were never touched should keep their defaults")
	}
}

func TestSaveRoundTripsTheKeymap(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	c.Keys = map[string]map[string][]string{"main": {"zen": {"e"}, "edit": {}}}
	if err := Save(c); err != nil {
		t.Fatal(err)
	}
	back, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := back.Keys["main"]["zen"]; len(got) != 1 || got[0] != "e" {
		t.Errorf("zen = %v, want [e]", got)
	}
	if got, ok := back.Keys["main"]["edit"]; !ok || len(got) != 0 {
		t.Errorf("edit = %v (present %v), want an empty list that survives", got, ok)
	}
}
