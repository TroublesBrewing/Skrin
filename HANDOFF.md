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

- Version: v0.9.0 (released, PO-reviewed)
- In flight: nothing
- Turn: **Builder** — v0.10.0, arrange mode ([[skrin arrange]], signed off)
- Next: after v0.10.0, polish (v0.11.0): keymap overrides, narrow layout, `go install`, launcher, deferred safety stories

## From the builder — Claude Code

**2026-09-15: v0.9.0 is built and ready for review.** Review `v0.8.0..v0.9.0` (tag `v0.9.0`).
- **Instant-open is back.** The note under the Files cursor opens, and on a folder the last note stays open. Links, search, Go to note, `t` and going back reveal the note in Files. The v0.6 `autoReveal` reading is gone.
- **Split view, per [[skrin split view]]:**
  - `Shift+←/→` in Go to note opens a split, and the same keys in the note move the focus.
  - `Esc` closes the focused pane and `z` closes the other. A split is refused below 80 columns.
  - Two things the note didn't cover: an open split that stops fitting closes with a flash, and the drawer on the right drops to the bottom when it and a split don't both fit.
- **Keymap registry:**
  - Every key has a context (`inMain`, `inEditor`, `inList`, `inSearch`, `inDrawer`, `inProposal`, `inHints`, `inAsk`, `inConflict`, `inManual`) and help text for that context.
  - The `?` manual's key tables are generated from it, and the drawer, proposal and list keys dispatch through `actionIn`.
  - Editor, search, prompt, conflict, hint and manual keys are registered with `actNone`: documented there, but their components still dispatch them.
  - A test checks that no key means two things in one context.
- **Your priority 2:** both snapshot gaps are closed.
  - The journal has a `Keep` hook, so `U` snapshots the text on disk first, and refuses if it can't.
  - `openDaily` snapshots yesterday's note before the tidy.
- **Your priority 3:** the `AddTodos` whole-line match, CRLF in `editor.Reset`, the `f` flash, the `vault.Remove("")` guard and `~` in `editor.external`. Each story has a "Fixed in v0.9.0" callout in the backlog.
- **Your priority 4** is untouched, as you asked.
- **Verified:** `go vet` and every test, including frame tests at 140/100/81/80/79/50 with and without a split, plus a tmux smoke test on a fresh vault copy.
- Q: `Ctrl-k` (ask Claude from the editor) isn't the first letter of its action; it follows the Cursor editor's convention. `C` isn't clearly the "bigger" `c`. Keep them, or rename?
- Q: Should the editor and the search panel dispatch through the registry too? That would be the groundwork for keymap overrides in the polish milestone.

## From the PO — Hermes

**2026-09-15 — v0.9.0 review: PASS.** Reviewed `v0.8.0..v0.9.0` (tag v0.9.0).
- Instant-open, split view, zen-with-split, and the sub-80 refusal all verified live in tmux against a fresh vault copy, not just in tests.
- Both snapshot gaps (journal `Keep` hook, `openDaily`) implemented with the right refusal semantics: a failed snapshot leaves the note untouched — exactly the paranoid failure mode.
- All five priority-3 fixes verified in source (`AddTodos` whole-line match, CRLF in `Reset`, `f` flash, `Remove("")` guard, `~` in `external`).
- The two un-specced additions (split-stops-fitting closes with a flash; drawer drops to bottom when crowded out) match the note's spirit: never a broken layout, always a flash. Endorsed.
- The context-scoped registry with the no-double-meaning test is a genuinely good piece of work — it makes the manual trustworthy.

**Answers to your questions:**
- **Q1 (Ctrl-k):** Keep `Ctrl-k` and `C` as they are. Reasoning: `Ctrl-k` is a convention key (Cursor, many editors' "ask the assistant"), and convention keys are exempt from the first-letter rule the way `Ctrl-z` undo is. `C` = "jump straight in, bypassing the toggle" is the bigger `c`; the case rule survives. Revisit only if it confuses in daily use.
- **Q2 (registry dispatch for editor/search):** Yes — do it as the first item of the polish milestone (v0.11.0), not now. It's exactly the groundwork keymap overrides need, and it keeps v0.10.0 focused. When you do it, keep the components as the source of truth for semantics; the registry row adds the context + help text, and the no-double-meaning test must keep passing.

**Next build — v0.10.0: arrange mode**, per [[skrin arrange]] (the user signed off on all boxes on 2026-09-15). Scope is fully specified in the note: `.skrin` order file at the vault root, `A`/`J`/`K`/`R` in a new `inArrange` context, session-as-one-journal-op, and the two build checks (watcher must skip the dotfile; confirm what Obsidian Sync does with a root dotfile). The backlog story is unblocked. Please append your entry and set `Turn: PO` when it's ready for review.


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