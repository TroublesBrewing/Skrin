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

func TestInstantOpenDefaultsOn(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !c.InstantOpen() {
		t.Error("InstantOpen should default to on: unset config keeps today's behaviour")
	}
}

func TestInstantOpenReadsOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	dir := filepath.Join(home, "skrin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[general]\ninstant_open = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.InstantOpen() {
		t.Error("instant_open = false should turn it off")
	}
}

func TestObsidiansVaultListPerSystem(t *testing.T) {
	cases := []struct{ goos, xdg, appData, want string }{
		{"linux", "", "", "/home/u/.config/obsidian/obsidian.json"},
		{"linux", "/x/cfg", "", "/x/cfg/obsidian/obsidian.json"},
		{"darwin", "/x/cfg", "", "/home/u/Library/Application Support/obsidian/obsidian.json"},
		{"windows", "", "/w/Roaming", "/w/Roaming/obsidian/obsidian.json"},
		{"windows", "", "", "/home/u/AppData/Roaming/obsidian/obsidian.json"},
	}
	for _, c := range cases {
		if got := filepath.ToSlash(registryPath(c.goos, "/home/u", c.xdg, c.appData)); got != c.want {
			t.Errorf("%s (XDG %q, APPDATA %q): %s, want %s", c.goos, c.xdg, c.appData, got, c.want)
		}
	}
}

func TestReadableWidthDefaultsOn(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !c.RenderReadableWidth() {
		t.Error("a readable line length should be on unless turned off")
	}
	off := false
	c.Render.ReadableWidth = &off
	if c.RenderReadableWidth() {
		t.Error("readable_width = false should turn it off")
	}
}

func TestVaultSettingsRoundTrip(t *testing.T) {
	root := t.TempDir()
	vs := VaultSettings{
		TemplatesFolder: "Templates",
		TemplateRules:   []TemplateRule{{Folder: "Begrepp", Template: "Templates/Begrepp template.md"}},
	}
	if err := SaveVaultSettings(root, vs); err != nil {
		t.Fatal(err)
	}
	got := LoadVaultSettings(root)
	if got.TemplatesFolder != "Templates" || len(got.TemplateRules) != 1 || got.TemplateRules[0].Folder != "Begrepp" {
		t.Errorf("LoadVaultSettings = %+v, want %+v", got, vs)
	}
	// The settings file lives at the vault root, not in config.
	if _, err := os.Stat(filepath.Join(root, SettingsFile)); err != nil {
		t.Errorf("settings file should be at the vault root: %v", err)
	}
}

func TestLoadVaultSettingsMissingOrGarbageIsNone(t *testing.T) {
	root := t.TempDir()
	if got := LoadVaultSettings(root); got.TemplatesFolder != "" || len(got.TemplateRules) != 0 {
		t.Errorf("missing file should mean none: %+v", got)
	}
	if err := os.WriteFile(filepath.Join(root, SettingsFile), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := LoadVaultSettings(root); got.TemplatesFolder != "" || len(got.TemplateRules) != 0 {
		t.Errorf("garbage should mean none: %+v", got)
	}
}

func TestMigrateVaultSettingsMovesOldTemplatesBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	dir := filepath.Join(home, "skrin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	conf := "vault = \"/vaults/main\"\n\n[templates]\n  folder = \"Templates\"\n\n  [[templates.rules]]\n    folder = \"Begrepp\"\n    template = \"Templates/Begrepp template.md\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := MigrateVaultSettings(root, &cfg); err != nil {
		t.Fatal(err)
	}
	vs := LoadVaultSettings(root)
	if vs.TemplatesFolder != "Templates" || len(vs.TemplateRules) != 1 || vs.TemplateRules[0].Folder != "Begrepp" {
		t.Errorf("migration should move the block into the vault settings: %+v", vs)
	}
	// The old block is gone from config.toml so a later Save can't drop
	// or duplicate it.
	saved, _ := os.ReadFile(Path())
	if strings.Contains(string(saved), "[templates]") || strings.Contains(string(saved), "Begrepp") {
		t.Errorf("the old [templates] block should be gone from config.toml:\n%s", saved)
	}
	// A second run is a no-op: the vault settings already hold it.
	if err := MigrateVaultSettings(root, &cfg); err != nil {
		t.Fatal(err)
	}
	if again := LoadVaultSettings(root); len(again.TemplateRules) != 1 {
		t.Errorf("a second migration should not duplicate: %+v", again)
	}
}

func TestMigrateVaultSettingsNeverOverwritesExisting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	dir := filepath.Join(home, "skrin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[templates]\n  folder = \"Old\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	// The vault already chose its own templates folder.
	existing := VaultSettings{TemplatesFolder: "Mine"}
	if err := SaveVaultSettings(root, existing); err != nil {
		t.Fatal(err)
	}
	if err := MigrateVaultSettings(root, &cfg); err != nil {
		t.Fatal(err)
	}
	if got := LoadVaultSettings(root).TemplatesFolder; got != "Mine" {
		t.Errorf("existing vault settings must win: got %q", got)
	}
}
