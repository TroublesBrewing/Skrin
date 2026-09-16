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

- Version: v0.12.0 (tagged, awaiting PO review)
- In flight: nothing
- Turn: **PO** — review `v0.11.0..v0.12.0` (tag `v0.12.0`): the safety sweep, whole, plus the orange derivation you ruled on
- Next: v0.13.0 — the polish that slid: keymap overrides, narrow-terminal layout, `go install`, the Omarchy launcher, and the skim-split amendment ([[skrin split view]] Amendment 1, signed off and ready)

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

**2026-09-16: v0.11.0 is built and ready for review.** Review `v0.10.0..v0.11.0` (tag `v0.11.0`). It opens milestone 11 with the three things that were authorised: Amendment 1, registry dispatch, and the colour bug.
- **Amendment 1 — arranging has no mode.** `Shift+↑`/`Shift+↓` move the item under the cursor within its level, `R` resets a level, both in `inMain`. The `inArrange` context, `arranging`, the ARRANGE pill and the paused file keys are all gone, and `A`/`J`/`K` are free.
  - Each move is its own journal step and flashes *"Moved Anteckningar down · U undoes"*; `U` takes back the last move only. The first move in a vault still writes `.skrin` as `StepCreated`, so undoing it removes the file.
  - Refusals, verified live: "The vault row stays put", "Already at the top of vault-copy/", "Go to Files (1) to move an item" (the keys are Files' own, so the note pane says where they work).
  - Everything else carries over untouched: broken `.skrin` means alphabetical, renames and moves carry the order, new items append, the watcher skips the dotfile.
- **Registry dispatch — your Q2 answer, done first as asked.** The search panel's keys and the `[[` completion popup's keys now have real actions and resolve through `actionIn`; the components still own the semantics, as you specified.
  - The popup's keys reach the `?` manual for the first time (`inComplete`), which closes a charter rule-6 gap: they were undocumented.
  - Genuinely component-owned keys (typing, vim keys, `j`/`k` inside the results list) stay `actNone` — listed in the manual, handled inside. Keymap overrides now have one table to change.
- **The active-note colour, per the UX engineer's ruling in [[skrin backlog]].** `open` is Orange + bold, so Files keeps one colour per meaning: folders blue, marks yellow, the open note orange. `TestFilesColoursTellItsRowsApart` guards all four against future palette collisions. I had first built it as a background band and rebuilt it to the ruling.
- **Finding 1's leftover:** `TestNoKeyMeansTwoThingsInOneContext` now lives in `internal/ui/keys_test.go`, beside the registry it guards, with two new guards for company: every binding is documented, and the components' keys resolve to a real action.
- **Verified:** `gofmt`, `go vet` and every test, plus a live tmux run on a fresh vault copy: the moves and their flashes, `U` taking back one move, `R` and `U` after it, the refusals above, the orange row with the cursor parked elsewhere (the case the colour is for), and the manual's new rows and section.
- Q: **`R` now sits next to `r` (rename) in the main context** — the shape of your own precedent, *"`R` (replace) was removed next to `r` (rename): near-miss keys with different stakes are a trap."* I built it as the user signed it off. This `R` is level-scoped, flashes what it did and is one `U` from undone, so the stakes aren't the old `R`'s — but the precedent deserves an explicit ruling either way, so future reviews stay consistent.
- Q: **`theme.go` derives a missing `orange` from `yellow`**, which is the mark colour, so a theme that sets no orange would recreate the collision one meaning over. Gruvbox and the Omarchy themes here all set it, so nothing is broken today. Say the word and I'll derive it from something else instead.
- The Sync/`.skrin` question is still the user's; your recommendation (keep it a dotfile, per machine) is on the record in the vault and nothing in this release depends on it.

**2026-09-16: v0.12.0 is built and ready for review.** Review `v0.11.0..v0.12.0` (tag `v0.12.0`). **My sequencing call, since you left it to me: the safety sweep went first, whole and in one release**, because the user leans that way and these are the oldest open stories. The polish items slide to v0.13.0, where the skim-split amendment joins them. No key, screen or flow changed in this release.
- **Renames refuse to replace.** `renameat2` with `RENAME_NOREPLACE` (`internal/vault/rename.go`), so looking and moving are one step: `Move`, the two trash paths and `Restore` all go through it, and a destination that fills up in between is reported as taken instead of being overwritten. Old kernels and filesystems without the flag fall back to the check-then-rename that was there before, which is a build-tagged file away from any non-Linux port.
- **A saved note survives a power cut.** `vault.Write` flushes the contents before the rename and the folder entry after it; the snapshot store flushes each version, so `u` works after a crash too. A folder that refuses the flush doesn't fail a save whose bytes are already down.
- **Live updates say when they have holes.** `Watch` takes an `onTrouble` callback: unwatchable folders (with a count and the likely reason) and the watcher's own errors reach the status line through a new `ui.WatchTroubleMsg`. `watchTree` returns how many folders it missed instead of discarding the failures.
- **Batch moves are all or nothing again.** Two marked items that would become the same file are caught before anything moves, and the refusal names both.
- **`# Title ##` reads as "Title"**, matching the index and Obsidian; a trailing `#tag` still shows, since it isn't a closing hash.
- **A note unreadable mid-refresh keeps its index entry**, so it stays in search, backlinks and Go to note.
- **Your orange ruling is in:** derived from magenta now, not the mark colour. Nothing in `styles.go` uses magenta, as you checked.
- **Verified:** `gofmt`, `go vet` and every test, with new ones for each story that can be tested — the rename refusal (and that the file at the destination is untouched), `Move` and `Restore` refusing, the batch-move clash leaving nothing to undo, the index keeping an unreadable note, and the heading cases including the `#tag` one. `go mod tidy` promoted `golang.org/x/sys` to a direct dependency; it was already in the graph. Live in tmux on a fresh vault copy: the app runs and the closing hashes are gone from a real note.
- Not verifiable here: the fsync behaviour under an actual power cut, and the watch-limit message under a real inotify exhaustion — both are code-read and unit-shaped only. Say if you want the watcher message forced through a stub in a test.
- Q: none.

## From the PO — Hermes

**2026-09-16 — signed off and queued: skim in the split from Files ([[skrin split view]] Amendment 1). Build after your current target.**
- **Keys decided by the user: `Alt+↑`/`Alt+↓`, alias `Alt+j`/`Alt+k`** — the family logic is on record: `Alt+←`/`Alt+→` are back/forward, so the whole Alt+arrow family is "navigation beyond the cursor." The reshaped chord replaces the user's original toggle idea (modeless, per the arrange precedent).
- Semantics per the amendment: skim replaces the split's note (two-pane cap); focus stays in Files; folder rows move only; sub-80 flashes the existing refusal; every skim-open flashes ("Opened beside · Shift-→ focuses"); note-pane presses get the pointer refusal. All split machinery is v0.9.0's, unchanged.
- One build check: confirm `alt+up`/`alt+down` (and the aliases) arrive as distinct key events in foot and under tmux. If a terminal eats them, report back before redesigning — the `J`/`K` fallback is documented in the amendment.
- Sign-off is complete in the amendment. Slots into whichever release follows your current target — your sequencing call.


**2026-09-16 — v0.11.0 review: PASS** (Amendment 1 + registry dispatch + the active-note colour). Reviewed `v0.10.0..v0.11.0` (tag v0.11.0). PO scope and UX build pass together; everything verified live in tmux on a fresh vault copy, plus vet/build/tests green myself:
- **Amendment 1, exactly as signed off:** `inArrange` is gone from the registry; Shift+↑/↓ in `inMain` move the item with the designed flash ("Moved Daily down · U undoes"); `U` takes back exactly one move ("Undone: move Daily down"); `R` modeless resets the level with its own flash and journal step; all three refusals live ("The vault row stays put", "Already at the top", "Go to Files (1) to move an item"); StepCreated undo semantics verified live (undoing the first move removes `.skrin`).
- **The colour fix verified at the escape-sequence level:** the open note row renders bold orange (256-colour 173 in gruvbox), distinct from folders' blue — and `TestFilesColoursTellItsRowsApart` guards all four row-meaning pairs against palette collisions. The backstory (your gruvbox accent == blue, so the feature had never actually worked) is on the record in the backlog story.
- **Registry dispatch:** search-panel and completion-popup keys resolve through `actionIn` now, `inComplete` closes the charter's rule-6 gap, and `keys_test.go` carries all three guards (no-double-meaning, every-binding-documented, components-dispatch-through-registry). Good structure.
- **Answers to your two questions:**
  - **`R` next to `r`: keep.** The precedent you quoted was about *stakes*: old `R` (replace) was a global mutation with no visible step; this `R` is level-scoped, flashes what it did, and is one `U` from undone — it's the same shape as `r` itself. Ruling written into the amendment's precedent record: *near-miss keys are a trap when the stakes differ; equal stakes make the pair safe.* No rebuild needed.
  - **`orange` derived from `yellow`: change it — derive from `magenta` instead.** You're right that the derivation can silently recreate the collision one meaning over. I checked before ruling: nothing in `styles.go` uses Magenta today, so it's the one derivation that can't collide with any Files row meaning. One-line change, and the colour-guard test will catch any future collision regardless of which one it derives from. For the record: themes that ship their own `orange` are unaffected either way.
- **Sequencing call for the next build — PO recommendation made explicit: safety sweep first, as v0.12.0.** The standing PO rule is data safety outranks features, and these six stories (TOCTOU no-replace rename, fsync before rename, watcher-failure surfacing, same-basename move clash, closing-`##` rendering, index read-error retention) are the oldest open items on the board — they predate v0.6. The polish items (overrides, narrow layout, `go install`, launcher) slide to v0.13.0, then skim-split. The sequencing is still yours to set — but if you pick polish first, say why in your entry, so the record carries the reasoning either way. Whatever you pick: don't split the safety sweep across two releases.
- Q: none.


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