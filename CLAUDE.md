# CLAUDE.md

Skrin is a keyboard-driven terminal UI for Obsidian vaults, written in Go with Bubble Tea v2 (`charm.land/*/v2`).

The spec and milestone plan live in the user's vault at `~/Documents/vault-1/tui.md`. The user signs off there and leaves comments at the bottom, so read it before starting a milestone.

## Commands

Go is pinned in `.mise.toml`. If `go` isn't on PATH, prefix commands with `mise exec --`.

- Build: `go build ./...`
- Test: `go test ./...` (all packages have tests; the UI tests assert the frame is exactly terminal-sized)
- Vet: `go vet ./...`
- Run: `go run ./cmd/skrin ~/Work/tries/vault-copy`
- Install: `go build -o ~/.local/bin/skrin ./cmd/skrin`

## Layout

- `cmd/skrin`: flags and vault resolution (argument → `~/.config/skrin/config.toml` → Obsidian's `obsidian.json`); wires the watchers to `tea.Program.Send`.
- `internal/vault`: disk access. Paths are vault-relative with `/`, and `""` is the root. Dot-entries (`.obsidian`, `.trash`) are hidden. Also has the debounced fsnotify watcher.
- `internal/theme`: builds the palette from Omarchy's `~/.local/state/omarchy/current/theme/colors.toml`, with an optional `skrin.toml` override. It live-reloads by watching `theme.name`.
- `internal/markdown`: a line-oriented renderer. Every display `Line` carries its source line (`Src`) and heading level. Heading jumps and "open at line" depend on this, so keep it 1:n.
- `internal/logo`: the embedded official Obsidian icon (`assets/obsidian-logo.png`), drawn with half-block characters.
- `internal/ui`: the root model (`model.go`), rendering (`view.go`), the folder tree, and the keymap registry (`keys.go`). Every key goes through the registry, because the `?` manual will be generated from it.

## Rules

- Test against the copy `~/Work/tries/vault-copy`, never the real vault.
- Never write inside `.obsidian/`.
- The view must fill the terminal exactly. Use the `fit`, `spread` and `box` helpers in `view.go`.
