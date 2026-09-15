# Changelog

Each milestone in the plan (`~/Documents/vault-1/tui.md`) ships as a minor version, so milestone N is v0.N.0. Fixes between milestones bump the patch number (v0.1.1). v1.0.0 follows milestone 9, once Skrin has held up in daily use.

## v0.8.0 — 2026-09-15

Milestone 8: the Claude drawer.

- **Opening it:** `c` opens the Claude drawer, `C` jumps straight to typing, and `Ctrl-k` does the same from the editor. Replies stream in as Claude writes them, with a line for each file it reads or search it runs.
- **Where it sits:** along the bottom by default, folded to one line while you're elsewhere, or on the right with `assistant.position = "right"`. `Alt-p` flips it, and Skrin remembers the side per vault.
- **What Claude sees:** the open note, sent with your message whenever it has changed. Highlighted text goes into your message, with a new line below it to type on.
- **New ways to highlight:**
  - Shift and the arrows in the editor. Typing replaces the selection.
  - `v` in vim mode.
  - `v` with `j`/`k` for whole lines in the reading view.
- **Claude can read the whole vault but changes nothing by itself.** A change it wants shows as a diff in the note panel. `y` applies it and `n` rejects it. `u` undoes an applied edit, and `U` a create, move or delete.
- **Typing:** `Enter` sends, and `Alt-Enter` adds a line.
- **Memory:** one conversation per vault, picked up again next time. `Alt-n` starts a fresh one.
- **How it runs:**
  - Skrin runs your Claude Code (`claude -p`, streaming JSON) with only Read, Glob and Grep.
  - It adds Skrin's own tools, served by `skrin mcp` over a Unix socket: `vault_search`, `current_context` and the four `propose_` tools.
- `assistant.enabled = false` turns the drawer off, and `assistant.model` picks a model.

## v0.7.0 — 2026-09-15

Milestone 7: zen mode, the manual and the rebrand.

- **Zen mode:** `z` shows only the open note, centred at 80 columns, without Files, the header or the status line.
  - It works while editing too. The status line only comes back to ask something or show a message.
  - `z` or `Esc` leaves zen mode. `Tab`, `1` or `h` leave it too and go to Files.
- **The manual:** `?` opens a full-screen manual.
  - Keys, generated from the keymap so the list stays true, plus the editor's keys and the vim keys.
  - Search syntax, links, undo and the config file.
  - `/` filters it, `j`/`k` scroll, and `Esc` closes it.
- **Rebrand:** Skrin has its own logo, a small chest drawn in the theme's colours. It recolours when the Omarchy theme changes.
  - The Obsidian icon and "unofficial TUI for Obsidian vaults" are gone. The header now reads "a terminal home for your vault".
  - With no note open, the note panel shows the chest large, the name and a few hints. That's the splash; there's nothing to wait for or dismiss.
  - The chest also opens the manual.
  - The chest is drawn with half-block characters everywhere, not sixel. As pixel art it's already sharp that way, and no image can linger on screen.
- The status line hints now include `?` and `z`.

## v0.6.0 — 2026-09-15

Milestone 6: two panels, like Obsidian (`skrin two panels.md` in the vault).

- **Files** replaces the folder tree and the folder contents. It's one tree holding folders and files, folders first, A→Z. Notes show without `.md`, and other files are dimmed. The top row is the vault itself.
- A note opens on `Enter` (or `l`), as in Obsidian, and stays open while you move around Files. The open note is highlighted in Files.
- Keys in Files:
  - `l` goes into a folder, opening it. `h` closes the folder, or goes up when it's closed. `Backspace` goes up.
  - `Enter` opens or closes a folder, opens a note, or opens any other file in its own app.
  - `H` closes all folders.
  - `1` is Files, `2` the note, and `Tab` switches between them. `3` is gone.
- Which item a key acts on:
  - In Files, it's the row under the cursor. `e`, `E`, `u`, `Ctrl-r` and `o` open that note first.
  - In the note, it's the open note.
  - Marks still come first for `m` and `d`.
- The Files cursor stays put when a link, a search hit, Go to note, `t` or going back opens a note. The exception is when Obsidian's "Automatically reveal current file" is on.
- Renaming or moving the open note keeps it open under its new name. Deleting it empties the note panel.
- `v` ranges can cross folders. `Ctrl-a` marks everything next to the cursor in its folder.
- Skrin remembers the open folders, the cursor and the open note for each vault, under `~/.local/state/skrin/session/`.
- Files and the note fit side by side down to 80 columns; it was 100.
- The status line shows the open note's modified date.

## v0.5.4 — 2026-09-15

- Go to note: the first row is the note already open, or an empty row when none is. So `Enter` on an empty field just closes the list, with no need to reach for `Esc`. From the backlog.

## v0.5.3 — 2026-09-15

- `g` opens Go to note instantly; there's no more wait for a second `g`.
- `G` / `End` go to the bottom, and `GG` (a second `G` within 0.4 s) or `Home` go to the top. Neither key waits: `G` jumps right away, and a quick second `G` takes it to the top.

## v0.5.2 — 2026-09-15

- `Ctrl-l` in the editor, the same key as Obsidian's "Toggle checkbox", from the backlog:
  - An empty or plain line becomes `- [ ] …`, with the cursor ready to type.
  - A list item becomes a to-do.
  - A to-do is ticked off, or back on.

## v0.5.1 — 2026-09-15

Fixes from the backlog note (`skrin backlog.md` in the vault).

- `n` with `Folder/` creates the folder with an `Untitled` note inside, as an empty name does. `N` accepts a trailing `/`.
- Search & replace moved into search:
  - `R` is gone. In `/`, `Alt-r` switches replace on and off.
  - All the panel's toggles are Alt keys: `Alt-t` this note / whole vault (now for plain search too), `Alt-c` case, `Alt-w` whole words.
- Search results are easier to tell apart: every note starts with a full-width title bar (name, folder, match count), notes are separated by a gap, and matching lines sit under a `│` line-number gutter.
- `g` opens Go to note after a 0.3 s wait for a second `g`.
  - `gg` goes to the top, and so does `Home`.
  - `g` followed by a letter opens Go to note with the letter typed.
  - `Ctrl-p` still works.

## v0.5.0 — 2026-09-15

Milestone 5: search & replace.

- `/` searches the vault as you type. Results are grouped by note, with each matching line shown and the match highlighted. `Enter` opens the note at that line, and the next `/` brings the query back.
  - `word word` must all appear, in the note or its name.
  - `"a phrase"`, `-exclude` and `a OR b`.
  - `#tag` or `tag:tag`, where nested tags count.
  - `[property]` and `[property:value]`.
  - `path:` and `file:`.
  - `Alt-c` matches case.
- `Ctrl-p` jumps to a note by name or alias. `Enter` on a name that doesn't exist offers to create it.
- `R` is search & replace, in this note or the whole vault (`Ctrl-t`).
  - Toggles: case-sensitive (`Alt-c`) and whole words (`Alt-w`). Plain text only.
  - A live preview shows every match as old→new and marks ⚠ on matches inside `[[links]]`. `Space` skips a match or a whole note.
  - `Ctrl-s` asks, then replaces.
  - A note changed on disk since the preview is skipped, not overwritten.
  - Every changed note gets a snapshot (`u`), and one `U` undoes the whole replace.
- The index now keeps tags (from the body and the `tags` property), properties as written, and each note's text.

## v0.4.0 — 2026-09-15

Milestone 4: links and headings.

- A vault-wide index of links, headings, aliases and block ids. Links resolve the way Obsidian resolves them:
  1. the vault path
  2. a path relative to the note
  3. the file name, preferring the note's own folder, then the shortest path
- Following links:
  - `f` puts letters on the links in view; type one to follow it. `Enter` in the note pane does the same, and follows directly when only one link is in view.
  - Links to `#Heading` and `#^block` land on that line.
  - Web links and other files (images, PDFs) open with `xdg-open`.
- `Backspace` / `Ctrl-o` / `Alt-←` go back through followed links; `Ctrl-i` / `Alt-→` go forward.
- A link to a note that doesn't exist offers to create it, in the place Obsidian's "Default location for new notes" setting says.
- `b` lists the notes linking to this one, with the linking line; `Enter` opens it there.
- `o` shows the outline, filterable. `{` and `}` jump to the previous and next heading.
- Typing `[[` in the editor suggests notes and their aliases. `[[Note#` suggests that note's headings. `Enter` or `Tab` accepts, `Esc` dismisses.
- Rename and move update links:
  - Skrin asks "Update N links in M notes?" when links would break. Plain `[[Name]]` links that still work are left alone.
  - It follows Obsidian's "Always update internal links" and link-format settings.
  - One `U` undoes the move together with the link edits.

## v0.3.0 — 2026-09-15

Milestone 3: editing.

- `e` edits the note right in the note pane.
  - The editor opens at the line you were reading, and closing it brings you back there.
  - It soft-wraps and highlights markdown lightly.
  - Enter continues lists and checklists; Enter on an empty item ends the list.
  - Tab / Shift-Tab indent list items.
  - `Ctrl-z` / `Ctrl-y` undo and redo keystrokes.
  - `Ctrl-s` saves, and `Esc` saves and leaves.
- `[editor] vim = true` gives the editor vim-style keys: `hjkl`, `w b e`, `0 ^ $`, `gg G`, `i a I A o O`, `x dd yy p P D J`, `u`, `Ctrl-r`.
- `E` opens the note in your own editor (`[editor] external`, else `$VISUAL`, `$EDITOR`, nvim) and Skrin waits until you're done.
- `u` restores a note to how it was before its last edit, and `Ctrl-r` takes that back.
  - This works for edits in Skrin and in `E`, and survives restarts.
  - Earlier versions are kept in `~/.local/state/skrin/snapshots/`, 20 per note, and follow renames and moves.
- Saving checks whether the note changed on disk while you were editing (Obsidian, Sync, another editor). If it did, you choose:
  - `m` keeps yours, `t` takes theirs, `d` shows the difference.
  - Either way the other version is kept, so `u` brings it back.
- A new note (`n`) opens straight in the editor.

## v0.2.0 — 2026-09-15

Milestone 2: file and folder operations, and daily notes.

- `n` and `N` create a note or folder in the current folder. An empty name becomes "Untitled" (then "Untitled 1", …), and `sub/name` creates the folders on the way.
- `r` renames. `m` moves, using a fuzzy folder picker. `d` deletes after a y/n question.
- Deleted items go to the trash Obsidian is set to use: the system trash, or the vault's `.trash`.
- Marks:
  - `Space` marks an item, `v` marks a range, `Ctrl-a` marks the whole folder, and `Esc` clears marks.
  - With marks set, `m` and `d` act on every marked item at once, even across folders.
- `U` undoes the last file operation. A bulk move or delete counts as one operation. The undo history lasts for the session.
- `t` opens today's daily note, or creates it from Obsidian's daily-note settings and template.
  - A new daily note gets the unfinished todos from the previous one, following the Rollover Daily Todos plugin's settings.
  - While Obsidian has the vault open, its plugin does the rollover instead, so todos never appear twice.
  - `[daily] rollover_todos = false` turns this off.
- Not yet: moving or renaming doesn't update links pointing at the item. That comes in v0.4.

## v0.1.0 — 2026-09-15

Milestone 1: a read-only vault browser.

- Three panes: a folder tree, the contents of the current folder, and the selected note rendered in the terminal.
- The vault is chosen from the command-line argument, then `~/.config/skrin/config.toml`, then the vault Obsidian itself has open.
- The note renderer supports:
  - frontmatter shown as a properties box
  - coloured headings, tags, task checkboxes and callouts
  - wikilinks, dimmed when the target note doesn't exist
  - aligned tables, and code blocks clipped rather than wrapped
- Colours come from the active Omarchy theme and update live when the theme changes.
- Panes update when files change on disk.
- The header shows the official Obsidian logo, the version, the vault and the theme.
- Keys: `j`/`k`, `g`/`G`, `Ctrl-d`/`Ctrl-u`, `h`/`l`/`Tab`/`1 2 3`, `Enter`, `Backspace`, `q`.
