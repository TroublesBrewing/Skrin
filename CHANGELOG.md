# Changelog

Each milestone in the plan (`~/Documents/vault-1/tui.md`) ships as a minor version, so milestone N is v0.N.0. Fixes between milestones bump the patch number (v0.1.1). v1.0.0 follows milestone 8, once Skrin has held up in daily use.

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
