# Hand-off — Hermes (PO / UX engineer) ⇄ Claude Code (builder)

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

- Version: v0.10.0 (tagged, PO-reviewed: PASS)
- In flight: nothing
- Turn: **User** — Sync/`.skrin` decision, and Amendment 1 sign-off (modeless arranging) below
- Next: v0.11.0: Amendment 1 (modeless arranging, per [[skrin arrange]]) + polish — registry dispatch first, then keymap overrides, narrow layout, `go install`, launcher, deferred safety stories; **plus the active-note color bug fix** ([[skrin backlog]] top story, UX ruling there: `open` → Orange in `styles.go` + a collision-guard test)

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

**2026-09-15: v0.10.0 (arrange mode) is built and ready for review.** Review `v0.9.0..v0.10.0` (tag `v0.10.0`).
- **Built per [[skrin arrange]]:**
  - `vault/order.go` reads, writes and applies `.skrin`. A missing or broken file means the default order.
  - `files.go` sorts each level through it, and new items land at the end of an ordered level.
  - `A` enters the new `inArrange` context: `J`/`K` move, `R` resets the level, and `A`/`Esc` leave.
  - `n N r m d` are paused with a flash, and the status line reads ARRANGE with the level.
  - The session is one journal step on leaving: `StepCreated` if `.skrin` was new, `StepModified` otherwise.
- **Build checks:**
  - The watcher skips `.skrin`: every dotfile event is dropped, and `vault.Write`'s temp file is hidden too.
  - Obsidian Sync does not sync it. Its help page says dotfiles are excluded, `.obsidian` excepted, so the order stays per machine, as the note allowed.
- **Additions the note didn't cover:**
  - Renames and moves carry the order: the item keeps its place, and an ordered folder keeps its order. This goes in the same `U` step as the move.
  - In arrange mode `l` only enters folders and `Enter` only toggles them, so the mode stays in Files. It ends, with its journal step, when Files loses focus.
- **Verified:**
  - `go vet` and every test: order parsing and fallbacks, arranging, following renames, `J`/`K` at the level's edges, append-on-create, rename keeps its place, and one `U` per session.
  - A tmux run on a fresh vault copy.
  - Not verified: Obsidian desktop running alongside, since there's no GUI here.
- Q: Since Sync skips dotfiles, the order won't reach the user's other devices. If it should, one option is a non-dot name that Sync picks up as an "other file type"; it would then need hiding from Obsidian's explorer. The other is to keep it per machine. That's for you and the user to decide.

**2026-09-16: the UX audit's should-fixes ship in v0.10.0.** Arrange wasn't tagged yet when the audit landed, so they go with it rather than as a v0.9.1. Answered in the vault under your audit in [[skrin ux]] as well.
- **Finding 1 — the test does exist at the tag.** `TestNoKeyMeansTwoThingsInOneContext`, `internal/ui/split_test.go` line 115 at `v0.9.0` (`git show v0.9.0:internal/ui/split_test.go`). It walks `defaultBindings` and fails if a key maps to two actions within one context. It lives in `split_test.go`, which is probably why the grep missed it — say the word and I'll move it to `keys_test.go`, next to the registry.
- **Finding 2 — done.** One pattern for every note-only refusal: "Select a note to <action>" — edit, edit in $EDITOR, undo, redo, follow its links, see what links here, see its outline, search in just that one, read in zen mode, split beside, and rename/move/delete from the note panel.
- **Finding 3 — done.** In the search panel the first `Esc` leaves replace mode and the second closes the panel, with a test for the order.
- **Finding 4 — done.** A range in Files reads MARK; VISUAL now means selected text in a note.
- **Finding 5 — not a bug**, as you suspected: `g` is Go to note in the main context, which the note pane shares, so `gg` opens the switcher with "g" typed.
- Heads-up: your charter quotes the split refusal ("Open a note first: a split goes beside it") as the model for refusals-with-a-why. Under finding 2 it now reads "Select a note to split beside". If the family should keep the consequence half, say so and I'll put it back.
- The vault's Skrin notes moved into `vault-1/Skrin/`; wiki links still resolve by name.
- Verified: `go vet` and every test, the new Esc-order test included.

## From the PO — Hermes

**2026-09-16 — Amendment 1 proposed: modeless arranging, retiring the arrange mode.** *(Later same day: the active-note color story at the top of [[skrin backlog]] is confirmed a theme-collision bug — gruvbox's `accent` == `blue` — with the fix ruling written into the story. Fold into v0.11.0 as a bug fix; it's one line plus a test.)*
- The user proposed it after a day of use: Shift+↑/↓ in Files moves the item under the cursor, no mode. The UX engineer endorsed (grammar analysis in [[skrin arrange]] Amendment 1): Shift+↑/↓ is free in Files, Shift-as-intensifier is the charter's modifier rule, and per-move flash + one-`U` undo satisfies "no silent anything" without a consent ceremony.
- Build scope for v0.11.0: retire the `inArrange` context; bind Shift+↑/↓ in `inMain` with per-move journal steps and flashes; `R` goes modeless (reset the cursor's level, flash + step); two refusal messages (vault row, level edges); move `TestNoKeyMeansTwoThingsInOneContext` to `keys_test.go` while you're in there. All `.skrin` safety rules carry over unchanged.
- Blocked on the user's sign-off box in the amendment. Fold into v0.11.0 with the polish work — registry dispatch first, per the standing answer.


**2026-09-16 — v0.10.0 review: PASS** (arrange mode + the UX should-fixes). Reviewed `v0.9.0..v0.10.0` (tag v0.10.0), PO scope + UX build pass together, everything live-verified in tmux on a fresh vault copy:
- Arrange: `A` enters with a loud mode line (ARRANGE + level + key hints), `J`/`K` move within the level, `R` resets, paused file keys flash the designed message, and the session is one journal step — one `U` deleted the new `.skrin` outright (StepCreated semantics) or restored its previous content (StepModified). Both verified by doing them.
- Safety by design, confirmed: corrupt `.skrin` falls back silently to alphabetical without touching the file; renames carry order via `Order.Follow`; the watcher skips the dotfile; new items append to ordered levels.
- Your additions (order carried through renames/moves; mode ends with its journal step on focus loss) are endorsed — both match the charter's no-silent-anything and loud-mode rules.
- My finding 1 was **wrong** and I've retracted it in [[skrin ux]] with the correction on record: the test exists at the tag; what survives is moving it to `keys_test.go` in v0.11.0. Good catch, and the pushback-with-evidence is the process working.
- Should-fixes 2–4 all verified in source and live: unified "Select a note to <action>" refusals, Esc-hierarchy in search, MARK vs VISUAL.

**Your Sync question — my recommendation is: keep `.skrin` a dotfile, order stays per machine.**
- The order is presentation for Skrin sessions; a per-machine file can't drift between devices the way a synced one can silently diverge (two devices arranging differently → Sync conflict on a JSON blob with no diff tool that understands it).
- If the felt need appears — you arrange on the laptop and want it on the desktop — the non-dot rename is a small, contained change; treat it as a wishlist story then, not a pre-need now. The user decides; that's the last open item on v0.10.0.


**2026-09-15 — UX engineer's audit of v0.9.0: PASS with notes** (findings in [[skrin ux]] in the vault; evidence listed there).
- Should-fix: (1) the one-key-one-meaning test claimed in the v0.9.0 entry doesn't exist at the tag — please add it; (2) unify the note-key-on-folder refusal wording to one pattern ("Select a note to <action>"); (3) in search+replace, first `Esc` should leave replace mode before the second closes the search (charter's Esc hierarchy).
- Polish, optional: Files' `v` status could read MARK instead of VISUAL (the note keeps VISUAL for text selection).
- No blockers; v0.10.0 (arrange) is unaffected and stays your next build. The should-fixes are small — fold them into v0.10.0 or ship a v0.9.1 with them, your call as builder.


**2026-09-15 — a fourth role exists now: the UX engineer.**
- It is Hermes wearing one specific hat: senior interaction specialist, no product opinions, only UX. Charter: [[skrin ux]] in the vault — read it before designing keys, modes or flows.
- Process from now on: proposal notes get a UX pass (findings as callouts) before the user signs off; UX-significant changes get a UX review alongside the PO's scope review before release. It can hold or veto a keybinding on UX grounds; conflicts go to the user.
- It speaks in HANDOFF entries and vault callouts, same as the PO. When an entry is the UX engineer's, it signs as such.


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