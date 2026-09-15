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
- `internal/vault`: disk access. Paths are vault-relative with `/`, and `""` is the root. Dot-entries (`.obsidian`, `.trash`) are hidden. Also has the debounced fsnotify watcher. File operations (`ops.go`) never overwrite. Deletes go to the freedesktop trash or the vault's `.trash` (`trash.go`). `journal.go` is the undo stack behind `U`.
- `internal/obsidian`: read-only access to Obsidian's own config: the vault registry, `.obsidian/*.json`, the Rollover plugin's settings, and whether Obsidian is running.
- `internal/daily`: the daily-note path, moment.js date formatting, template variables and todo rollover, all mirroring Obsidian and the Rollover Daily Todos plugin.
- `internal/theme`: builds the palette from Omarchy's `~/.local/state/omarchy/current/theme/colors.toml`, with an optional `skrin.toml` override. It live-reloads by watching `theme.name`.
- `internal/markdown`: a line-oriented renderer. Every display `Line` carries its source line (`Src`) and heading level. Heading jumps and "open at line" depend on this, so keep it 1:n.
- `internal/editor`: the built-in editor. `editor.go` has the buffer, keys, vim mode and soft-wrap layout. `view.go` has highlighting and rendering. It returns `Save` / `Close` actions and never touches disk itself.
- `internal/snapshot`: per-note version stacks (undo/redo) under `$XDG_STATE_HOME/skrin/snapshots`, keyed by note path so they can follow moves. Every write Skrin makes over a note's previous content snapshots it first. That's what makes `u` safe.
- `internal/index`: the vault-wide index of links, headings, aliases and block ids. It is incremental (it re-parses only notes whose mtime or size changed). It holds Obsidian-style link resolution (`Resolve`), backlinks, `LinkText` (Obsidian's `newLinkFormat`) and `Apply` for rewriting links. `ui.reload` keeps it current.
- `internal/search`: the Obsidian-style query parser and matcher (`search.go`, over `search.Doc` values the index hands out), and plain-text `Find`/`Replace` (`replace.go`).
- `internal/logo`: the embedded official Obsidian icon (`assets/obsidian-logo.png`), drawn with half-block characters.
- `internal/session`: where you were in each vault (open folders, the Files cursor, the open note and its scroll position). It's saved on quit under `$XDG_STATE_HOME/skrin/session` and restored at start.
- `internal/ui`: two panes, Files and the open note. The Files cursor and the open note are independent: moving the cursor never changes the note, and a note opens on Enter.
  - `model.go` has the root model, `view.go` the rendering and `files.go` the Files tree.
  - `ops.go` has file actions and marks. `edit.go` has editing, the save-conflict flow, `E` and `u`.
  - `links.go` has opening notes, following links, history, backlinks, the outline, `[[` completion and moving with link updates.
  - `find.go` has the `/` search panel and Go to note; `replace.go` has the panel's replace mode.
  - `modal.go` has the name prompt, the y/n question and the filterable chooser used for moves, backlinks and the outline.
  - `keys.go` has the keymap registry. Every key goes through it, because the `?` manual will be generated from it.

## Releases

Milestone N of the plan ships as v0.N.0, and fixes in between bump the patch number. To release:

1. Set `Version` in `internal/version/version.go`.
2. Add a section to `CHANGELOG.md`.
3. Commit, then `git tag vX.Y.Z`.
4. Reinstall the binary.
5. Mark the milestone done in `tui.md`, with its version.

## Rules

- Test against the copy `~/Work/tries/vault-copy`, never the real vault.
- Never write inside `.obsidian/`.
- Tests and manual runs must set `XDG_DATA_HOME` and `XDG_STATE_HOME` to temp dirs, so nothing lands in the user's real trash or snapshots.
- The view must fill the terminal exactly. Use the `fit`, `spread` and `box` helpers in `view.go`.
