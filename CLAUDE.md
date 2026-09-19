# CLAUDE.md

Skrin is a keyboard-driven terminal UI for Obsidian vaults, written in Go with Bubble Tea v2 (`charm.land/*/v2`).

## Read first: the steering document and "Skrin nu"

At the start of every Skrin session, read two notes in the user's vault, both in Swedish:

- `~/Documents/vault-1/Skrin/skrin styrdokument.md` holds the direction, the course to v1.0 and, in its section *Hur vi arbetar*, every rule for how we work: roles, the idea box and board, the pace, and what you may build now. It outranks everything except the user. Follow its rules; they are kept only there, on purpose, so they are not repeated here.
- `~/Documents/vault-1/Skrin/Skrin nu.md` shows where things stand: the phase, what's being built now, bugs, the board and the log.

The user's first rule for project management is **no information in two places when it can be avoided**. Give each thing one home and point to it rather than copying it: this file, commit messages and the log point to the vault and CHANGELOG, not the other way round.

The old process (HANDOFF.md, the backlog, quick reports, a milestone per version) was retired on 2026-09-19. It is archived, untouched, under `~/Documents/vault-1/Skrin/Archive/Skrin/Process till 2026-09-19/`.

`~/Documents/vault-1/Skrin/tui.md` is the original spec and stays the reference for how things work. The UX charter `~/Documents/vault-1/Skrin/skrin ux.md` is the rulebook for keys and flows: read it before designing either.

## Commands

Go is pinned in `.mise.toml`. If `go` isn't on PATH, prefix commands with `mise exec --`.

- Build: `go build ./...`
- Test: `go test ./...` (all packages have tests; the UI tests assert the frame is exactly terminal-sized)
- Vet: `go vet ./...`
- Run: `go run ./cmd/skrin ~/Work/tries/vault-copy`
- Install: `go build -o ~/.local/bin/skrin ./cmd/skrin`

## Layout

- `cmd/skrin`: flags and vault resolution (argument → `~/.config/skrin/config.toml` → Obsidian's `obsidian.json`); wires the watchers to `tea.Program.Send`.
- `internal/vault`: disk access. Paths are vault-relative with `/`, and `""` is the root. Dot-entries (`.obsidian`, `.trash`) are hidden. File operations (`ops.go`) never overwrite: every move goes through `renameNoReplace` (`rename.go`, `renameat2` with `RENAME_NOREPLACE` on Linux), so a file appearing at the destination is refused with `ErrExists` rather than replaced. `Write` syncs before the rename, so a saved note survives the power going out. Deletes go to the freedesktop trash or the vault's `.trash` (`trash.go`). `journal.go` is the undo stack behind `U`. `order.go` reads and writes `skrin.json` at the vault root, the manual order of Files levels (an old `.skrin` is renamed on open); a missing or broken file means the default order. The debounced fsnotify watcher is here too: `Watch` takes an `onTrouble` callback and uses it when folders can't be watched or the watcher errors, because live updates must never stop quietly.
- `internal/config`: `~/.config/skrin/config.toml`, loaded and saved. Settings are pointers where unset means a documented default, read through typed accessors (`InstantOpen()`, `RenderSpreads()` …), and the vault is found here when none is given.
- `internal/obsidian`: read-only access to Obsidian's own config: the vault registry, `.obsidian/*.json` (app, daily notes, templates), the Rollover plugin's settings, and whether Obsidian is running.
- `internal/daily`: the daily-note path, moment.js date formatting, template variables (the Daily notes and Templates plugins fill them differently: `Expand` and `ExpandTemplate`) and todo rollover, all mirroring Obsidian and the Rollover Daily Todos plugin.
- `internal/habit`: the `### Habits` block in a daily note. The block is the only state there is, so Obsidian shows and edits the same lines.
- `internal/duedate`: turns a word in a `due::` field (`tomorrow`, `friday`) into the date it means, once, as the note is saved.
- `internal/theme`: builds the palette from Omarchy's `~/.local/state/omarchy/current/theme/colors.toml`, with an optional `skrin.toml` override. It live-reloads by watching `theme.name`.
- `internal/markdown`: a line-oriented renderer. Every display `Line` carries its source line (`Src`) and heading level. Heading jumps and "open at line" depend on this, so keep it 1:n.
- `internal/frontmatter`: where a note's frontmatter ends, Obsidian's way, shared by the renderer, the index and Insert template. The editor's highlighting keeps its own reading, since it colours a block still being typed.
- `internal/spread`: runs spreads, the ```` ```spread ```` / ```` ```dataview ```` query blocks (a subset of Dataview's DQL). A spread only reads, and answers in markdown for the renderer to draw.
- `internal/imgmeta`: an image's pixel size and file size from its header alone, for image embeds.
- `internal/editor`: the built-in editor. `editor.go` has the buffer, keys, vim mode and soft-wrap layout. `view.go` has highlighting and rendering. It returns `Save` / `Close` actions and never touches disk itself. It also keeps the selection (Shift+arrows, vim `v`). `table.go` is Advanced Tables-style editing: Tab/Enter move through a table and line it up, and `Table(op)` runs the row, column, alignment and sort commands.
- `internal/snapshot`: per-note version stacks (undo/redo) under `$XDG_STATE_HOME/skrin/snapshots`, keyed by note path so they can follow moves. Every write Skrin makes over a note's previous content snapshots it first. That's what makes `u` safe.
- `internal/index`: the vault-wide index of links, headings, aliases and block ids. It is incremental (it re-parses only notes whose mtime or size changed), and a note that can't be read at that moment keeps the entry it had rather than dropping out. It holds Obsidian-style link resolution (`Resolve`), backlinks, `LinkText` (Obsidian's `newLinkFormat`) and `Apply` for rewriting links. `ui.reload` keeps it current.
- `internal/search`: the Obsidian-style query parser and matcher (`search.go`, over `search.Doc` values the index hands out), and plain-text `Find`/`Replace` (`replace.go`).
- `internal/book`: the Book Card's lookups (Open Library, Google Books and Libris), cover downloads into the vault, and the book-note template. No UI here.
- `internal/logo`: Skrin's own logo, a small chest drawn as pixel art in half-block characters and painted in the theme's colours. It comes in two sizes: one for the header, and one for the splash in the empty note pane and at the top of the manual.
- `internal/version`: the release number shown in the header and by `skrin --version`.
- `internal/session`: where you were in each vault (open folders, the Files cursor, the open note and its scroll position). It's saved on quit under `$XDG_STATE_HOME/skrin/session` and restored at start.
- `internal/assistant`: Claude Code for the drawer.
  - `claude.go` runs `claude -p` with stream-json both ways and turns its output into events.
  - `mcp.go` is `skrin mcp`, a small hand-written MCP server on stdio, plus the Unix-socket bridge it uses to reach the running Skrin.
  - `prompt.go` is the system prompt.
  - Tests use a fake `claude` script, never the real one.
- `internal/ui`: Files and the open note, or two notes in a split.
  - The note under the Files cursor is the open note: moving the cursor opens notes (`peek`, called from `settle`, unless instant-open is off in Settings), and opening a note any other way reveals it in Files.
  - `model.go` has the root model, `view.go` the rendering (zen mode included) and `files.go` the Files tree.
  - `ops.go` has file actions and marks. `edit.go` has editing, the save-conflict flow, `E` and `u`.
  - `links.go` has opening notes, following links, history, backlinks, the outline, `[[` completion and moving with link updates.
  - `find.go` has the `/` search panel and Go to note; `replace.go` has the panel's replace mode.
  - `modal.go` has the name prompt, the y/n question and the filterable chooser used for moves, backlinks and the outline.
  - `drawer.go` has the Claude drawer, `propose.go` Claude's tools and the y/n proposals, and `select.go` the line selection in the reading view.
  - `arrange.go` keeps Files in your own order, with no mode: `Shift+↑`/`Shift+↓` move the item under the cursor within its level through `skrin.json`, `R` puts a level back in the default order, and every move is its own journal step. Moves and renames carry the order along (`orderFollows`, called from `moveNow`).
  - `palette.go` has the command palette (Ctrl+P): every command by name, with its key. A new main-context action goes in `mainCommands` or, if nobody would look it up by name, `notInPalette`; a test holds the two to the registry.
  - `onboard.go` has what teaches a newcomer from the screen itself: the status line's hints for where you are (generated from the keymap, never hand-written), the welcome card and its tip of the day, and the answers to a key that does nothing and to a first Ctrl+C.
  - `help.go` has the `?` manual (Keys, Settings and Guide tabs, with the Guide's prose), `keybindings.go` the Keys tab where keys are changed, and `settings.go` the Settings tab's rows.
  - `table.go` has Insert table's form and `templates.go` Insert template, both reached from the palette.
  - `book_card.go` has the Book Card (`B`), `habit_view.go` the habits overlay (`T`), `quick_note.go` the quick note (`i`), `images.go` image embeds and their previews, and `spreads.go` what connects spreads to the index and the editor.
  - `styles.go` turns the theme palette into the Lip Gloss styles everything draws with.
  - `split.go` has the split view. The other pane waits in `m.split`; `swapPanes` trades it with the focused one, so all the note code keeps working on `m.notePath`.
  - `keys.go` has the keymap registry. It holds every key, with the context it works in (`inMain`, `inEditor`, `inList`, `inDrawer`, …) and help text for that context.
    - `actionIn` looks a key up in its context, and `help.go`'s `?` manual is generated from the same table.
    - Components that handle their own keys (the search panel, the `[[` popup, the lists, the drawer) still look the action up here with `actionIn`, so a key is changed in one place and can be overridden later. Keys that are purely the component's own (typing, the editor's vim keys) are registered with `actNone`, so the manual still lists them. A new key must go in the registry.

## Releases

Release when there is something worth releasing, not after every fix: new features bump the minor number, fixes the patch number. v1.0 comes when the steering document's v1.0 bar is met. To release:

1. Run the tests and try the change live (see Rules).
2. Set `Version` in `internal/version/version.go`.
3. Add a section to `CHANGELOG.md`.
4. Commit, then `git tag vX.Y.Z`.
5. Reinstall the binary.
6. Add a line to the log in "Skrin nu" that points to the release.

## Rules

- Test against the copy `~/Work/tries/vault-copy`, never the real vault.
- Never write inside `.obsidian/`.
- Tests and manual runs must set `XDG_DATA_HOME` and `XDG_STATE_HOME` to temp dirs, so nothing lands in the user's real trash or snapshots.
- The view must fill the terminal exactly. Use the `fit`, `spread` and `box` helpers in `view.go`.
