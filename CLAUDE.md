# CLAUDE.md

Skrin is a keyboard-driven terminal UI for Obsidian vaults, written in Go with Bubble Tea v2 (`charm.land/*/v2`).

The spec and milestone plan live in the user's vault at `~/Documents/vault-1/tui.md`. The user signs off there and leaves comments at the bottom, so read it before starting a milestone.

## Roles

There are four of us. You (Claude Code) are the **builder**. Hermes Agent is the **product owner**; its reviews and stories come with PO callouts in the vault notes and it holds the UX/keybinding-ergonomics bar. The **UX engineer** is a dedicated role Hermes also runs: a senior interaction specialist whose only loyalty is UX — it holds Skrin to the charter in `~/Documents/vault-1/skrin ux.md` (read it before designing keys or flows), runs UX passes on proposal notes before the user signs off and on UX-significant changes before release, and can hold or veto a keybinding on UX grounds. The user has the final word and signs off in `tui.md`. Practical points:

- Before starting a milestone, read `tui.md`, `skrin backlog.md`, and any proposal note with an open sign-off box (e.g. [[skrin split view]]). Proposal notes define scope before a story is built.
- The 13 stories tagged *(LLM-added …)* in the backlog were found by Hermes's code review of v0.5.4; they are triaged in the notes around them but not yet human verified. The stories tagged *Product owner, 2026-09-15* are from user feedback after daily use and reflect the user's current intent.
- When a review story conflicts with a milestone in flight, fix the story into the milestone's scope rather than deferring it.
- Hermes reviews builds against the spec, the acceptance criteria, and the keymap rules (lowercase = local, uppercase = global/bigger; first letter of the action; every key in the registry with context-aware help text; no prefix keys unless the spec demands one).
- The hand-off between us lives in `HANDOFF.md`. Read it at the start of every session and follow its rules: append dated entries to your section, update the State line when your work is done, set `Turn:` to say whose move is next, and list what needs review (commit range or tag). It is operational state between us; product decisions stay in the vault notes.
- Release versions are the PO's and user's call: propose, don't decide.

## Commands

Go is pinned in `.mise.toml`. If `go` isn't on PATH, prefix commands with `mise exec --`.

- Build: `go build ./...`
- Test: `go test ./...` (all packages have tests; the UI tests assert the frame is exactly terminal-sized)
- Vet: `go vet ./...`
- Run: `go run ./cmd/skrin ~/Work/tries/vault-copy`
- Install: `go build -o ~/.local/bin/skrin ./cmd/skrin`

## Layout

- `cmd/skrin`: flags and vault resolution (argument → `~/.config/skrin/config.toml` → Obsidian's `obsidian.json`); wires the watchers to `tea.Program.Send`.
- `internal/vault`: disk access. Paths are vault-relative with `/`, and `""` is the root. Dot-entries (`.obsidian`, `.trash`) are hidden. Also has the debounced fsnotify watcher. File operations (`ops.go`) never overwrite. Deletes go to the freedesktop trash or the vault's `.trash` (`trash.go`). `journal.go` is the undo stack behind `U`. `order.go` reads and writes `.skrin`, the manual order of Files levels; a missing or broken file means the default order.
- `internal/obsidian`: read-only access to Obsidian's own config: the vault registry, `.obsidian/*.json`, the Rollover plugin's settings, and whether Obsidian is running.
- `internal/daily`: the daily-note path, moment.js date formatting, template variables and todo rollover, all mirroring Obsidian and the Rollover Daily Todos plugin.
- `internal/theme`: builds the palette from Omarchy's `~/.local/state/omarchy/current/theme/colors.toml`, with an optional `skrin.toml` override. It live-reloads by watching `theme.name`.
- `internal/markdown`: a line-oriented renderer. Every display `Line` carries its source line (`Src`) and heading level. Heading jumps and "open at line" depend on this, so keep it 1:n.
- `internal/editor`: the built-in editor. `editor.go` has the buffer, keys, vim mode and soft-wrap layout. `view.go` has highlighting and rendering. It returns `Save` / `Close` actions and never touches disk itself. It also keeps the selection (Shift+arrows, vim `v`).
- `internal/snapshot`: per-note version stacks (undo/redo) under `$XDG_STATE_HOME/skrin/snapshots`, keyed by note path so they can follow moves. Every write Skrin makes over a note's previous content snapshots it first. That's what makes `u` safe.
- `internal/index`: the vault-wide index of links, headings, aliases and block ids. It is incremental (it re-parses only notes whose mtime or size changed). It holds Obsidian-style link resolution (`Resolve`), backlinks, `LinkText` (Obsidian's `newLinkFormat`) and `Apply` for rewriting links. `ui.reload` keeps it current.
- `internal/search`: the Obsidian-style query parser and matcher (`search.go`, over `search.Doc` values the index hands out), and plain-text `Find`/`Replace` (`replace.go`).
- `internal/logo`: Skrin's own logo, a small chest drawn as pixel art in half-block characters and painted in the theme's colours. It comes in two sizes: one for the header, and one for the splash in the empty note pane and at the top of the manual.
- `internal/session`: where you were in each vault (open folders, the Files cursor, the open note and its scroll position). It's saved on quit under `$XDG_STATE_HOME/skrin/session` and restored at start.
- `internal/assistant`: Claude Code for the drawer.
  - `claude.go` runs `claude -p` with stream-json both ways and turns its output into events.
  - `mcp.go` is `skrin mcp`, a small hand-written MCP server on stdio, plus the Unix-socket bridge it uses to reach the running Skrin.
  - `prompt.go` is the system prompt.
  - Tests use a fake `claude` script, never the real one.
- `internal/ui`: Files and the open note, or two notes in a split.
  - The note under the Files cursor is the open note: moving the cursor opens notes (`peek`, called from `settle`), and opening a note any other way reveals it in Files.
  - `model.go` has the root model, `view.go` the rendering (zen mode included) and `files.go` the Files tree.
  - `ops.go` has file actions and marks. `edit.go` has editing, the save-conflict flow, `E` and `u`.
  - `links.go` has opening notes, following links, history, backlinks, the outline, `[[` completion and moving with link updates.
  - `find.go` has the `/` search panel and Go to note; `replace.go` has the panel's replace mode.
  - `modal.go` has the name prompt, the y/n question and the filterable chooser used for moves, backlinks and the outline.
  - `drawer.go` has the Claude drawer, `propose.go` Claude's tools and the y/n proposals, and `select.go` the line selection in the reading view.
  - `arrange.go` keeps Files in your own order, with no mode: `Shift+↑`/`Shift+↓` move the item under the cursor within its level through `.skrin`, `R` puts a level back in the default order, and every move is its own journal step. Moves and renames carry the order along (`orderFollows`, called from `moveNow`).
  - `split.go` has the split view. The other pane waits in `m.split`; `swapPanes` trades it with the focused one, so all the note code keeps working on `m.notePath`.
  - `keys.go` has the keymap registry. It holds every key, with the context it works in (`inMain`, `inEditor`, `inList`, `inDrawer`, …) and help text for that context.
    - `actionIn` looks a key up in its context, and `help.go`'s `?` manual is generated from the same table.
    - Components that handle their own keys (the search panel, the `[[` popup, the lists, the drawer) still look the action up here with `actionIn`, so a key is changed in one place and can be overridden later. Keys that are purely the component's own (typing, the editor's vim keys) are registered with `actNone`, so the manual still lists them. A new key must go in the registry.

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
