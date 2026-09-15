# Changelog

Each milestone in the plan (`~/Documents/vault-1/tui.md`) ships as a minor version, so milestone N is v0.N.0. Fixes between milestones bump the patch number (v0.1.1). v1.0.0 follows milestone 8, once Skrin has held up in daily use.

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
