// Command skrin is a keyboard-driven terminal UI for an Obsidian vault.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/assistant"
	"github.com/lurioso/skrin/internal/config"
	"github.com/lurioso/skrin/internal/obsidian"
	"github.com/lurioso/skrin/internal/session"
	"github.com/lurioso/skrin/internal/theme"
	"github.com/lurioso/skrin/internal/ui"
	"github.com/lurioso/skrin/internal/vault"
	"github.com/lurioso/skrin/internal/version"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		mcp(os.Args[2:])
		return
	}
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
		fmt.Println("skrin v" + version.Version)
		return
	}
	if err := run(flag.Arg(0)); err != nil {
		fmt.Fprintln(os.Stderr, "skrin:", err)
		os.Exit(1)
	}
}

// mcp is `skrin mcp --socket PATH`, the MCP server Claude starts to reach
// the tools of the running Skrin. It isn't meant to be run by hand.
func mcp(args []string) {
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	sock := fs.String("socket", "", "the socket of the running Skrin")
	fs.Parse(args)
	if err := assistant.RunMCP(os.Stdin, os.Stdout, *sock, version.Version); err != nil {
		fmt.Fprintln(os.Stderr, "skrin mcp:", err)
		os.Exit(1)
	}
}

func run(vaultArg string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	root := vaultArg
	if root == "" {
		root = cfg.Vault
	}
	if root == "" {
		if root, err = config.DiscoverVault(); err != nil {
			return err
		}
	}
	v, err := vault.Open(root)
	if err != nil {
		return err
	}
	// Vault settings (templates) moved from config.toml into the vault's
	// own file in v0.52.0: move any old block over once, so nothing the
	// user paired is lost.
	if err := config.MigrateVaultSettings(v.Root, &cfg); err != nil {
		return err
	}
	vs := config.LoadVaultSettings(v.Root)
	themeDir := theme.DefaultDir()
	builtin := theme.Builtin(cfg.ThemeBuiltin())
	pal, themeErr := theme.LoadOr(themeDir, builtin)
	if themeErr != nil {
		pal = builtin
	}
	exe, err := os.Executable()
	if err != nil {
		exe = "skrin"
	}
	sock := filepath.Join(runtimeDir(), fmt.Sprintf("skrin-%d.sock", os.Getpid()))
	m, err := ui.New(v, pal, ui.Options{
		Session:         session.Load(v.Root),
		RolloverTodos:   cfg.RolloverTodos(),
		RestoreLastNote: cfg.RestoreLastNote(),
		InstantOpen:     cfg.InstantOpen(),
		Vim:             cfg.Editor.Vim,
		ExternalEditor:  cfg.Editor.External,
		Config:          cfg,
		ObsidianOpen: func() bool {
			if !obsidian.Running() {
				return false
			}
			reg, err := obsidian.LoadRegistry(config.ObsidianRegistry())
			// Unsure means open: skipping a rollover beats doing it twice.
			return err != nil || reg.IsOpen(v.Root)
		},
		Assistant: ui.AssistantOptions{
			Enabled: cfg.AssistantEnabled(),
			Right:   cfg.Assistant.Position == "right",
			Claude: assistant.Options{
				Command:   cfg.Assistant.Command,
				Model:     cfg.Assistant.Model,
				MCPConfig: assistant.MCPConfig(exe, sock),
			},
		},
		Library: ui.LibraryOptions{
			Folder:        cfg.LibraryFolder(),
			CoversFolder:  cfg.LibraryCoversFolder(),
			DefaultStatus: cfg.LibraryDefaultStatus(),
		},
		Images:          cfg.RenderImages(),
		Habits:          cfg.HabitsEnabled(),
		Beta:            cfg.BetaEnabled(),
		LineNumbers:     cfg.RenderLineNumbers(),
		ReadableWidth:   cfg.RenderReadableWidth(),
		Spreads:         cfg.RenderSpreads(),
		FolderTemplates: vs.TemplateRules,
		Vault:           vs,
	})
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
	_ = theme.WatchOr(ctx, themeDir, builtin, func(pal theme.Palette) { p.Send(ui.ThemeMsg{Palette: pal}) })
	onTrouble := func(s string) { p.Send(ui.WatchTroubleMsg{Text: s}) }
	if err := v.Watch(ctx, func() { p.Send(ui.VaultChangedMsg{}) }, onTrouble); err != nil {
		m.Flash("live refresh off: " + err.Error())
	}
	if cfg.AssistantEnabled() {
		if err := assistant.Serve(ctx, sock, m.ToolHandler()); err != nil {
			m.Flash("Claude can't reach Skrin's tools: " + err.Error())
		}
		defer os.Remove(sock)
	}
	_, err = p.Run()
	m.Close()
	if serr := session.Save(v.Root, m.Session()); serr != nil {
		fmt.Fprintln(os.Stderr, "skrin: couldn't remember where you were:", serr)
	}
	if to := m.SwitchTo(); to != "" {
		// The user asked to switch vaults. Skrin saved the choice to
		// config and quits here, then hands the terminal to itself in
		// the new vault. Exec replaces this process, so the same
		// terminal stays, and the new run reads the vault from config.
		return relaunch(to)
	}
	return err
}

// runtimeDir is where Skrin's socket goes: $XDG_RUNTIME_DIR, or else the
// temp dir.
func runtimeDir() string {
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return d
	}
	return os.TempDir()
}
