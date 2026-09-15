// Command skrin is a keyboard-driven terminal UI for an Obsidian vault.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/config"
	"github.com/lurioso/skrin/internal/theme"
	"github.com/lurioso/skrin/internal/ui"
	"github.com/lurioso/skrin/internal/vault"
)

const version = "0.1.0-dev"

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage: skrin [vault]

Skrin is a keyboard-driven terminal UI for an Obsidian vault.
With no vault given, it uses "vault" from ~/.config/skrin/config.toml,
or else the vault Obsidian itself has open.

Flags:
`)
		flag.PrintDefaults()
	}
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("skrin", version)
		return
	}
	if err := run(flag.Arg(0)); err != nil {
		fmt.Fprintln(os.Stderr, "skrin:", err)
		os.Exit(1)
	}
}

func run(vaultArg string) error {
	root, err := resolveVault(vaultArg)
	if err != nil {
		return err
	}
	v, err := vault.Open(root)
	if err != nil {
		return err
	}
	themeDir := theme.DefaultDir()
	pal, themeErr := theme.Load(themeDir)
	if themeErr != nil {
		pal = theme.Default()
	}
	m, err := ui.New(v, pal)
	if err != nil {
		return err
	}
	if themeErr != nil {
		m.Flash("theme unreadable, using built-in colours: " + themeErr.Error())
	}

	p := tea.NewProgram(m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Outside Omarchy there is no theme to watch; the built-in palette stays.
	_ = theme.Watch(ctx, themeDir, func(pal theme.Palette) { p.Send(ui.ThemeMsg{Palette: pal}) })
	if err := v.Watch(ctx, func() { p.Send(ui.VaultChangedMsg{}) }); err != nil {
		m.Flash("live refresh off: " + err.Error())
	}
	_, err = p.Run()
	return err
}

func resolveVault(arg string) (string, error) {
	if arg != "" {
		return arg, nil
	}
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	if cfg.Vault != "" {
		return cfg.Vault, nil
	}
	return config.DiscoverVault()
}
