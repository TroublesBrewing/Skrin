package theme

import (
	"context"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTheme(t *testing.T, name, colors string) string {
	t.Helper()
	current := t.TempDir()
	dir := filepath.Join(current, "theme")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "colors.toml"), []byte(colors), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(current, "theme.name"), []byte(name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func sameColor(a, b color.Color) bool {
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}

func TestLoadReadsColorsAndName(t *testing.T) {
	dir := writeTheme(t, "tokyo-night", `
mode = "dark"
accent = "#7aa2f7"
background = "#1a1b26"
foreground = "#a9b1d6"
red = "#f7768e"
`)
	p, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "tokyo-night" {
		t.Errorf("name = %q", p.Name)
	}
	if !sameColor(p.Accent, color.RGBA{0x7a, 0xa2, 0xf7, 0xff}) {
		t.Errorf("accent not read from colors.toml")
	}
	if !sameColor(p.Headings[0], p.Red) {
		t.Errorf("h1 should default to red")
	}
	// Missing keys derive from the theme's own colours, not gruvbox.
	if !sameColor(p.DarkBackground, p.Background) {
		t.Errorf("dark_background should fall back to background")
	}
}

func TestSkrinTomlOverridesRoles(t *testing.T) {
	dir := writeTheme(t, "x", `red = "#ff0000"`)
	if err := os.WriteFile(filepath.Join(dir, "skrin.toml"), []byte(`h1 = "#00ff00"`), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !sameColor(p.Headings[0], color.RGBA{0, 0xff, 0, 0xff}) {
		t.Errorf("skrin.toml h1 override ignored")
	}
}

func TestLoadWithoutOmarchy(t *testing.T) {
	p, err := Load(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != Default().Name {
		t.Errorf("want built-in palette, got %q", p.Name)
	}
}

func TestWatchReloadsOnThemeSwitch(t *testing.T) {
	dir := writeTheme(t, "gruvbox", `accent = "#111111"`)
	got := make(chan Palette, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := Watch(ctx, dir, func(p Palette) { got <- p }); err != nil {
		t.Fatal(err)
	}
	// What omarchy-theme-set does: new theme files, then theme.name.
	if err := os.WriteFile(filepath.Join(dir, "colors.toml"), []byte(`accent = "#222222"`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(dir), "theme.name"), []byte("nord\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case p := <-got:
		if p.Name != "nord" || !sameColor(p.Accent, color.RGBA{0x22, 0x22, 0x22, 0xff}) {
			t.Errorf("reloaded palette = %q / %v", p.Name, p.Accent)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no reload after theme switch")
	}
}
