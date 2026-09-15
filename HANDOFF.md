# Hand-off — Hermes (PO) ⇄ Claude Code (builder)

We don't share a live channel. This file is the interface between us, alongside
the vault (product decisions, read by the user) and git (the code). The user
decides when each of us runs; this file means neither of them has to repeat
context to the other.

## Rules

- Read this file at the start of a session. Before ending a work session, append a dated entry to your own section and update State.
- Append, never rewrite. Trim your section to the last ~5 entries.
- **Builder:** when work is ready for review, set `Turn: PO` and say what to review (commit range or tag). Ask PO questions as `Q:` lines in your entries.
- **PO:** after a review, write the verdict and set `Turn: Builder` — or `Turn: User` when a decision is needed.
- Product decisions needing the user's sign-off go in the vault (`tui.md` or the proposal note), never here. This file is operational state; the vault is decisions.
- Nothing here marks a release done: the user signs milestones off in `tui.md`, per CLAUDE.md.

## State

- Version: v0.8.0 (released)
- In flight: nothing
- Turn: **User** — sign-off on [[skrin split view]] in the vault
- Next: v0.9.0 = instant-open + split view, once the note is signed off

## From the builder — Claude Code

*(empty — first entry comes at the end of the next build session)*

## From the PO — Hermes

**2026-09-15 — v0.8.0 drawer review: PASS.**
- Snapshot-first on accepted proposals verified (`propose.go` refuses the change if the snapshot fails) — exactly the right paranoia.
- Accepted edits are journal ops, so one `U` undoes them — verified.
- `c` / `C` / `Ctrl-k`, `Alt-p`, `Alt-n`: all in the registry, no conflicts, context-scoped help. Endorsed.

**2026-09-15 — priorities for v0.9.0** (triage of the backlog, in build order):
1. Instant-open (the note follows the Files cursor) + split view, exactly per [[skrin split view]]. Pending the user's sign-off on that note.
2. Fold in the two snapshot-gap stories from the review (journal `U` snapshot, `openDaily` snapshot) — they are the last two gaps in the "u covers every write" promise.
3. Cheap fixes while in there: `AddTodos` heading-line match, CRLF in `editor.Reset`, `f`-flash on a non-note, `vault.Remove("")` guard, `~` in `editor.external`.
4. Defer to v1.0 polish: TOCTOU no-replace renames, fsync before rename, watcher-failure surfacing, same-basename move clash, closing-`##` rendering, index read-error retention.

Q: none.