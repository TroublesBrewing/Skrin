// Package theme turns the active Omarchy theme into Skrin's palette and
// reloads it when the theme changes.
package theme

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/BurntSushi/toml"
	"github.com/fsnotify/fsnotify"
)

// Palette holds the theme colours plus the roles Skrin paints with. The role
// mapping matches Omarchy's own Obsidian theme, so Skrin and Obsidian desktop
// look alike.
type Palette struct {
	Name string
	Dark bool

	Accent, Selection, Muted                                 color.Color
	Background, DarkBackground, DarkerBackground, LighterBackground color.Color
	Foreground, DarkForeground, LightForeground              color.Color
	Red, Yellow, Orange, Green, Cyan, Blue, Magenta          color.Color

	Headings                [6]color.Color
	Link, Tag, Code, Border color.Color
}

// gruvbox is the built-in fallback, used outside Omarchy and for keys a
// theme leaves out.
var gruvbox = map[string]string{
	"mode":               "dark",
	"accent":             "#7daea3",
	"selection":          "#504945",
	"muted":              "#665c54",
	"background":         "#282828",
	"dark_background":    "#1e1e1e",
	"darker_background":  "#161616",
	"lighter_background": "#3c3836",
	"foreground":         "#d4be98",
	"dark_foreground":    "#7c6f64",
	"light_foreground":   "#bdae93",
	"red":                "#ea6962",
	"yellow":             "#d8a657",
	"orange":             "#e1875c",
	"green":              "#a9b665",
	"cyan":               "#89b482",
	"blue":               "#7daea3",
	"magenta":            "#d3869b",
}

// derived fills keys a theme may omit from ones it has, in order.
var derived = [][2]string{
	{"dark_background", "background"},
	{"darker_background", "dark_background"},
	{"lighter_background", "background"},
	{"selection", "lighter_background"},
	{"dark_foreground", "foreground"},
	{"light_foreground", "foreground"},
	{"muted", "dark_foreground"},
	{"accent", "blue"},
	{"orange", "yellow"},
}

// Default returns the built-in palette.
func Default() Palette { return build("gruvbox (built-in)", gruvbox) }

// DefaultDir is where Omarchy keeps the active theme.
func DefaultDir() string {
	if d := os.Getenv("SKRIN_THEME_DIR"); d != "" {
		return d
	}
	state := os.Getenv("XDG_STATE_HOME")
	if state == "" {
		home, _ := os.UserHomeDir()
		state = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(state, "omarchy", "current", "theme")
}

// Load reads colors.toml from an Omarchy theme directory, with an optional
// skrin.toml next to it overriding colours or roles (h1–h6, link, tag, code,
// border). A directory without colors.toml yields the built-in palette.
func Load(dir string) (Palette, error) {
	data, err := os.ReadFile(filepath.Join(dir, "colors.toml"))
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Palette{}, err
	}
	vals := map[string]string{}
	if err := merge(vals, data); err != nil {
		return Palette{}, fmt.Errorf("colors.toml: %w", err)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "skrin.toml")); err == nil {
		if err := merge(vals, data); err != nil {
			return Palette{}, fmt.Errorf("skrin.toml: %w", err)
		}
	}
	for _, d := range derived {
		if _, ok := vals[d[0]]; !ok {
			if v, ok := vals[d[1]]; ok {
				vals[d[0]] = v
			}
		}
	}
	for k, v := range gruvbox {
		if _, ok := vals[k]; !ok {
			vals[k] = v
		}
	}
	name := "omarchy"
	if b, err := os.ReadFile(filepath.Join(filepath.Dir(dir), "theme.name")); err == nil {
		if s := strings.TrimSpace(string(b)); s != "" {
			name = s
		}
	}
	return build(name, vals), nil
}

func merge(vals map[string]string, data []byte) error {
	var m map[string]any
	if _, err := toml.Decode(string(data), &m); err != nil {
		return err
	}
	for k, v := range m {
		if s, ok := v.(string); ok {
			vals[k] = s
		}
	}
	return nil
}

func build(name string, v map[string]string) Palette {
	c := func(k string) color.Color { return lipgloss.Color(v[k]) }
	role := func(k, fallback string) color.Color {
		if s, ok := v[k]; ok {
			return lipgloss.Color(s)
		}
		return c(fallback)
	}
	return Palette{
		Name: name,
		Dark: v["mode"] != "light",

		Accent: c("accent"), Selection: c("selection"), Muted: c("muted"),
		Background: c("background"), DarkBackground: c("dark_background"),
		DarkerBackground: c("darker_background"), LighterBackground: c("lighter_background"),
		Foreground: c("foreground"), DarkForeground: c("dark_foreground"), LightForeground: c("light_foreground"),
		Red: c("red"), Yellow: c("yellow"), Orange: c("orange"), Green: c("green"),
		Cyan: c("cyan"), Blue: c("blue"), Magenta: c("magenta"),

		Headings: [6]color.Color{
			role("h1", "red"), role("h2", "green"), role("h3", "yellow"),
			role("h4", "blue"), role("h5", "magenta"), role("h6", "magenta"),
		},
		Link:   role("link", "accent"),
		Tag:    role("tag", "cyan"),
		Code:   role("code", "cyan"),
		Border: role("border", "muted"),
	}
}

// Watch calls onChange with the reloaded palette whenever Omarchy switches
// theme. omarchy-theme-set replaces the whole theme directory and then
// writes theme.name next to it, so the parent directory is what we watch.
func Watch(ctx context.Context, dir string, onChange func(Palette)) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := w.Add(filepath.Dir(dir)); err != nil {
		w.Close()
		return err
	}
	go func() {
		defer w.Close()
		var timer *time.Timer
		reload := func() {
			if p, err := Load(dir); err == nil {
				onChange(p)
			}
		}
		for {
			select {
			case <-ctx.Done():
				if timer != nil {
					timer.Stop()
				}
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if base := filepath.Base(ev.Name); base != "theme.name" && base != filepath.Base(dir) {
					continue
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(150*time.Millisecond, reload)
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return nil
}
