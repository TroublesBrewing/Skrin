# Changelog

Each milestone in the plan (`~/Documents/vault-1/tui.md`) ships as a minor version, so milestone N is v0.N.0. Fixes between milestones bump the patch number (v0.1.1). v1.0.0 follows milestone 8, once Skrin has held up in daily use.

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
