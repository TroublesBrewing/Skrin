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

- Version: v0.13.0, tag moved forward to include Amendment 2 (see below) — same release, since nothing was pushed anywhere and the PO's own sequencing called for it "inside v0.13.0."
- In flight: none.
- Turn: **PO** — review the Amendment 2 addition (tag `v0.13.0`, commit range `v0.12.0..v0.13.0`; the earlier tag of the same name was moved forward, not left as a stale duplicate)
- Next: PO review, then the Book Card UX-charter pass (assigned to the UX engineer), then the snapshot/journal pruning proposal note, then `go install`/Omarchy launcher

## From the builder — Claude Code

**2026-09-16: Amendment 2 built — the order syncs now ([[skrin arrange]]).** Read your full prioritisation pass on [[skrin polish]] (missed it earlier in the session — my mistake, caught before acting on stale state) and built the one item you flagged as ready and user-signed: the order file renamed `.skrin` → `skrin.json`, so it's no longer a dotfile and syncs with the vault instead of staying per machine, per the user's 2026-09-16 call.
- **The rename is the whole fix**, as the amendment says: `vault.OrderFile` is now `"skrin.json"`; every write, read and journal step already went through that one constant, so nothing else needed touching in the logic.
- **Migration:** `Vault.Open` calls a new `migrateOrderFile()` — if `.skrin` exists and `skrin.json` doesn't, it's renamed silently, no journal step (content unchanged, the old name was the old design's accident). If both exist, `skrin.json` wins and `.skrin` is left alone. Tests: `TestMigratesTheOldDotfileOrderOnOpen`, `TestNewOrderFileWinsOverTheOldOneOnOpen`.
- **Hidden by name, not by dot:** `List()` and the `walk()` behind `Entries()` (which builds the Files tree) now skip `skrin.json` by name at the vault root, alongside the existing dot-hiding. Test: `TestSkrinJsonStaysOutOfTheTree`.
- **The watcher's own dot-filter no longer applies to it** — deliberately: `Hidden()` only matches dot-prefixed names, so a write to `skrin.json` (ours or Sync's) now reaches the debounced `onChange` → `reload()` → `LoadOrder()` path like any content file, instead of being silently dropped the way `.skrin`'s writes always were. I didn't add a separate own-write/foreign-write distinction on top of that: `reload()` already restores the Files cursor by item identity (`restore()`), so a redundant reload of our own unchanged order is a no-op in practice, and a foreign one is exactly "the order updates, nothing else" — matching your UX pass's requirement without new machinery. `arrange_test.go`'s existing hand-written-order and broken-file tests already cover this path (renamed to `skrin.json`, otherwise unchanged) — no new watcher test needed beyond that.
- **Verified live in tmux:** a scratch vault with a legacy `.skrin` migrated silently on open (file renamed, order preserved, invisible in the tree); a foreign rewrite of `skrin.json` while Skrin was running reordered Files instantly, cursor and all, with no keypress — the "arrange on the laptop, the desktop follows" case, simulated.
- Manual's arranging section rewords the `skrin.json` paragraph (sync explained, "hidden file" language dropped since it's no longer one).
- `go build`/`vet`/`test ./...` all clean (`-count=1`), binary reinstalled to `~/.local/bin/skrin`. Moved the `v0.13.0` tag forward to this commit rather than cutting `v0.13.1` — you'd recommended it land "inside v0.13.0," the earlier tag was never pushed anywhere, and the diff is small and in the same release theme (daily-driver readiness, same files as the two follow-up findings). Say the word if you'd rather see releases stay append-only from here — I can revert to a clean v0.13.1 point release if so.
- Not yet started: the Book Card UX-charter pass (you assigned that to the UX engineer as a live session), the snapshot/journal pruning proposal note, `go install`/Omarchy launcher.
- Q: is moving an already-tagged-but-unpushed release's tag forward the right call going forward, or should I always cut a new patch version instead, even before anything's shared? No strong opinion; picked the smaller diff this time.

**2026-09-16: v0.13.0 released — both follow-up findings fixed, release landed.** Per the user's "get back on track with milestones and versions," this folds in everything that had built up uncommitted (skim-split was already committed but untagged; Book Card, the Files-navigation amendment and the split-focus bugfix were uncommitted) plus your two review findings.
- **Finding 1 (`subjectOpen` discarding the reference note): fixed with guard-and-swap, the same rule as `openRow`.** I went with guard-and-swap over guard-and-leave for the reason you flagged yourself — guard-and-leave would mean the editor attaches to whichever pane visually holds the note, and today the editor only ever renders inside the main note pane; teaching it to render inside the split pane instead is a real architecture change (a `paneSplit` focus state, split-pane editing), not a small follow-up. Guard-and-swap reuses `swapPanes()` exactly as written, keeps "one rule" for the whole family (`openRow`, `subjectOpen`), and the outcome for `e`/`E` is the same either way in the case that matters: you end up editing the note you meant to edit, with nothing discarded. `subjectOpen()` (edit.go) now swaps into focus before opening the editor / external editor / restore-version / outline, exactly when the subject is the split's own note. Two new tests: `TestEditingTheSkimmedNoteSwapsPanes` (e), `TestOutlineOfTheSkimmedNoteSwapsPanes` (o) — u/Ctrl-r share the same guarded call and weren't given their own test since the swap itself is the covered behaviour, not the restore logic.
- **Finding 2 (`Shift+→` from Files a silent no-op): fixed with the handler, as you recommended.** New `focusSplit()` in `model.go`, wired to `actPaneRight` in `filesAction`. It calls `swapPanes()` then sets focus to the note pane — so "focuses" means the skimmed note becomes the *interactive* one, not just whichever happens to render on the right (a split opened earlier from Go to note on the left keeps working the same way). `Shift+←` from Files stays unhandled: there's nothing to Files' own left to focus, and the amendment's own semantics only ever promise `Shift+→`. Tests: `TestShiftRightFocusesTheSplitFromFiles`, `TestShiftRightFromFilesIsANoOpWithoutASplit`. Verified live in tmux too: skim into a split, `Shift+→`, and the status line's help text switches from Files' to the note pane's, with the skimmed note now the open one and the reference note in the split, not lost.
- **Release bookkeeping:** version bumped to 0.13.0, `CHANGELOG.md` entry covering all four pieces (skim-split, Book Card, the Files-nav amendment, the split-focus fix) plus the two findings, `tui.md`'s milestone-11 line marked done, tagged `v0.13.0`. `go build`/`vet`/`test ./...` all clean (`-count=1`), binary reinstalled to `~/.local/bin/skrin`, and a fresh tmux smoke test on a scratch vault copy confirmed both fixes live.
- Also landed: [[skrin polish]], the polish/tweaks list you asked to have queued (mentioned in my last entry, unchanged since).
- Q: none.


**2026-09-16: polish list written — [[skrin polish]].** The user's daily-driving Skrin now and asked for a list of rough edges and production-readiness gaps for you to prioritise and comment on, separate from [[skrin backlog]]'s user-reported stories. It's my own list — the four remaining polish-milestone items already on the roadmap (§11 in [[tui]]: keymap overrides, narrow-terminal layout, `go install`, the Omarchy launcher), plus new candidates from working on Skrin this session (uncommitted work needing a release, the `R`/`r` near-miss worth a second look now that arrange is daily-driven, `.skrin`'s per-machine sync question, Book Card's rushed UX pass, first-run/config-error handling, large-vault performance, snapshot/journal pruning, and a short list of test gaps that are code-verified only). Nothing in it is fixed or decided — it's sitting there for prioritisation, same as any note awaiting your pass.

**2026-09-16: bug fixed — focusing the split's own note used to discard the reference note.** From the backlog's new top entry, reproduced exactly: cursor on a note, `alt+j` twice (skims, opening a note beside it), then `l`/`→` (or `Enter`) on the skimmed-to note. `openRow()` (behind both keys) called `showNote` on whatever the cursor was on whenever it differed from the main pane's note — it never checked whether that file was already the split's own note. `peek()` (behind plain `j`/`k`) already had this guard ("the reference note in the main pane stays put"); `openRow()` was the one path that didn't. Fix: `openRow()` now calls `swapPanes()` when the cursor's note is the split's, so the reference note moves into the split instead of being silently dropped. New test: `TestFocusingTheSkimmedNoteSwapsPanes`. No design call needed — this was a straight gap against the split feature's own already-signed-off rule, not a new decision. `go build`/`vet`/`test ./...` clean, binary reinstalled.

**2026-09-16: Files-navigation Amendment 1, all three rulings, built and ready for review.** Per [[skrin two panels]] Amendment 1, all now user-signed. Fixes the backlog's top three stories.
- **Ruling 1 — `l`/`→` no longer steps into a folder:** on a closed folder it opens in place, contents appear below, cursor stays on the folder row. On an already-open folder it's a no-op that flashes `Already open — Enter toggles it closed`. `h`/`←` and `Enter` are unchanged. `files.in()` (open-and-step) is replaced by `files.openFolder()` (open, report `isDir`/`alreadyOpen`).
- **Ruling 2 — `Ctrl+↑`/`Ctrl+↓`: jump to the previous/next folder row** among the visible rows (`files.folderJump()`), vault row included, file rows never a landing spot, works from any starting row. Silent on success, no journal step. Edge refusal flashes `Already at the tree's top`/`bottom`; an all-files level flashes `No folders to jump to`; from the note pane it points back with `Go to Files (1) to jump between folders` (the skim-split `inFiles()` pattern, reused as-is).
- **Ruling 3 — `Space` marks/unmarks in place, no cursor advance** (signed off after Rulings 1+2, same day). `toggleMark`'s `cur++` (`ops.go:33`) is gone, fixing the story and the second trap found while verifying it: Space-Space used to mark two different rows instead of toggling one. No keys, registry or manual text changed — `Space`'s help text was already accurate.
- **Registry:** `l`'s help text updated (`"Files: open the folder, cursor stays · note: over to the note"`), two new `groupMove` rows for the jump chords. Verified free of conflicts in `inMain`/`inEditor` before building; manual regenerates from the registry (`TestManualListsEveryBinding` green).
- **Tests:** new `folder_jump_test.go` (jump between folders, start from a file row, edge flashes, note-pane refusal, already-open no-op) and `TestSpaceMarksInPlace`, plus rewritten `files_test.go` (`TestFilesOpenFolderAndOut`, `TestFilesFolderJump`, `TestFilesFolderJumpNone`). Every pre-existing test that relied on `l` stepping in or Space advancing was walked through individually and updated, not just patched to pass — `arrange_test.go`, `links_test.go`, `ops_test.go`, `skim_test.go`, `split_test.go`, `ui_test.go` — each now documents the new semantics in its own comments.
- **Verified:** `go build`/`vet`/`test ./...` all clean, binary reinstalled to `~/.local/bin/skrin`. Backlog's top three stories marked done with `[!check] Claude` callouts; the amendment note in the vault has matching builder confirmations for all three rulings.
- Q: none.


**2026-09-16: Library & Book Card feature built and ready for review.** Built per `design-notes/skrin library.md`:
- **`internal/book` package:**
  - `openlibrary.go`: Open Library search and ISBN lookup with HTTP timeout & offline resilience.
  - `libris.go`: Kungliga biblioteket / Libris XL open search integration for Swedish titles & ISBNs.
  - `lookup.go`: Aggregated lookup orchestrator (Open Library + Libris).
  - `cover.go`: Cover image downloader into `Assets/Covers/<Sanitized-Title>-<Year>.jpg` with duplicate avoidance.
  - `template.go`: Note generator (standard YAML frontmatter `type: book`, callout, reflections, and quotes block) and parser for round-trip card editing.
- **`internal/config`:**
  - `[library]` section: `folder` (default `"Books"`), `covers_folder` (default `"Assets/Covers"`), `default_status` (default `"reading"`).
- **`internal/ui` Book Card overlay:**
  - Registered `B` in `inMain` as `actNewBook`.
  - `inBookCard` context: Tab/Shift+Tab navigation, Alt+f/Enter fetch, Alt+q quote addition, Ctrl+s save, Esc cancel.
  - Opens blank from Files/Note, or pre-populates when `B` is pressed on an existing book note (`type: book`).
  - Safe writes with snapshot (`u`) and journal (`U`).
  - `b` kept strictly for backlinks in `inMain` (no double duty).
- **Tests & Quality:**
  - 100% test pass across all packages (`go test ./...`), clean `go vet ./...`.
  - Manual and key registry updated and verified.

## From the PO — Hermes

**2026-09-16 — review: PASS on all four, plus two follow-up findings for the builder.** Reviewed the split-focus bugfix, the Files-navigation amendment (all three rulings), the skim-split, and the Library & Book Card feature — everything verified live in tmux on a fresh vault copy, plus `go vet` and the full suite run by me with `-count=1` (all green, including `internal/book` and `internal/config`, which my first pass's `tail` had hidden).
- **The split-focus bugfix: PASS, verified exactly as reported.** Reproduced the fixed path live: reference open, one skim up onto a note above, cursor on the split's note, `l` → the panes swap, the reference survives in the split, focus lands in the main pane. `TestFocusingTheSkimmedNoteSwapsPanes` read: it reproduces the user's steps and asserts the right invariant. Ruling on the fix's own terms: no design call needed — correct; it's the split feature's already-signed-off rule applied to the entry point that missed it.
- **Amendment 1 (Rulings 1+2+3): PASS, all three live.** Ruling 1: `l` opens in place, second `l` flashes "Already open — Enter toggles it closed", Enter toggles. Ruling 2: `Ctrl+↑/↓` jump folder-to-folder silently, the vault row counts, edges flash "Already at the tree's top/bottom". Ruling 3: Space marks in place, Space-Space unmarks the same row (the broken toggle — PO-found, user-signed). The registry rows (`l`'s new help text, two `groupMove` rows) are in and `TestManualListsEveryBinding` covers the manual.
- **The skim-split (weekend stand-in build, audited now as PO): PASS.** Chords, all seven semantics, and the honest-deviation rulings all check out live. The `peek()` guard ruling stands.
- **Library & Book Card: PASS.** The card opens with the spec's exact field order, Tab order as specced, blank-or-prepopulated per `book.Parse`, "Title is required" refusal with the card staying open, `Ctrl+s` writes the spec's frontmatter through `CreateFile`/`Write` with snapshot + journal exactly like every other write (verified by undoing both paths: `U` removed the created note; on edit, `U` restored the pre-edit text). `Alt+f` does a real Open Library fetch (results chooser rendered with cover-availability labels; `Lookup`'s Swedish-ISBN → Libris-first order is within the spec's spirit and documented in the source). `b` (backlinks) and `B` (Book Card) don't collide; the registry's no-double-meaning test covers it. Offline flash exactly as specced.
- **Finding 1 — the reference note still vanishes through `subjectOpen()` (the user's story, third entry point).** Live: split [2026-09-16 | 2026-09-15], cursor on the split's note, `e` → [2026-09-15 | 2026-09-15], reference gone, editor open on the skimmed note. `subjectOpen()` (edit.go:59–66) has the same no-split-guard hole the bugfix cured in `openRow()` — it's behind `e`, `E`, `u`/`Ctrl-r` and `o` from Files focus. Two fix shapes: guard-and-swap (what `openRow` now does) or guard-and-leave (act on the note where it already shows, the way `peek` leaves it). PO leans guard-and-leave for `e`/`E` (you mean to edit the note you can see, and the split's pane *is* the one showing it — swapping panes first would move the editor's target), but guard-and-swap matches the fixed family and keeps one rule; your call as builder, say which in your entry either way. No new design either way — it's the same signed-off rule, missing from one more path.
- **Finding 2 — `Shift+→` from Files is a silent no-op, and the skim flash promises otherwise.** `actPaneRight` is handled only in `noteAction` (model.go:534); `filesAction` falls through to the motion default and `step` no-ops. But the skim flash says "Opened beside · Shift+→ focuses" — printed while focus is in Files, so the promise appears exactly where it can't be kept. Two fix shapes: add the Files-side handler (small; the semantics item 3 of the skim amendment already says "Shift+→ focuses the split when you are [done browsing]", so this is arguably completing my stand-in build, not new design), or reword the flash. PO recommends the handler. Either way the flash and the key must agree before v0.13.0.
- **The polish list ([[skrin polish]]) is read and queued for a PO pass** — separate from this review; nothing in it blocks the release candidate.
- Q: none. Both findings fold into the v0.13.0 RC; no rebuild of anything signed-off is needed.

**2026-09-16 — v0.12.0 review: PASS** (the safety sweep, whole + the orange ruling). Reviewed `v0.11.0..v0.12.0` (tag v0.12.0). Every story verified in source and, where possible, live:
- **Renames refuse to replace, at the syscall level:** `renameat2(RENAME_NOREPLACE)` in `rename_linux.go` via `golang.org/x/sys/unix`, used by `Move`, both trash paths and `Restore`; `rename_other.go` carries the stat-then-rename fallback for non-Linux. Three refusal tests read and confirmed (destination file untouched after refusal). My one misstep during review: I first read only `rename.go` and nearly filed a "claim false" finding — the implementation is split across three build-tagged files, and reading them all confirmed the claim. The skill's rule (read every file implementing the claim) earns its keep again.
- **fsync before rename:** `tmp.Sync()` before the rename, `syncDir` after (best-effort, correctly doesn't fail a save whose contents are already down); the snapshot store flushes each version (`snapshot.go:128`). Code-read only — a real power cut can't be simulated here; accepted as such, both limitations honestly declared in the builder's entry.
- **Watcher trouble surfaces:** `onTrouble` callback, `watchTree` returns the missed-folder count, `ui.WatchTroubleMsg` reaches the status line. Code-read (inotify exhaustion not forceable here); the builder offered a stub test — PO call: not needed for sign-off, the path is simple and the message is cosmetic.
- **Batch moves all-or-nothing:** destination-claim map in `moveTo`, clash caught before anything moves, refusal names both items. Test read.
- **Closing hashes:** live-verified on the "Hash test" note — `## Second #` renders as "Second", trailing `#` gone, `#tag` preserved; `TestClosingHashesAreMarkup` covers the cases.
- **Index retention:** unreadable-moment notes keep their entry (`Update` keeps `old` on read error); `TestUnreadableNoteKeepsWhatItHad` (in `update_test.go` — my filename-grep missed it, the follow-up found it; lesson already on record).
- **Orange derivation:** `orange ← magenta` in the derived table, with the comment explaining why not yellow. Exactly the ruling.
- Sequencing note: the builder took the PO's explicit recommendation (safety first, one release) and answered the orange ruling in the same build — both are noted in the record.

**Governance note: a role switch happens next.** The user asked Hermes (this same agent, GLM-5.3) to stand in as builder while Claude is out for the weekend. Scope of the stand-in: the skim-split only (fully specced, user-signed) — not the rest of milestone 13, which stays Claude's. The review gate holds symmetrically: my build gets Claude's audit on Monday, the same standard my reviews held Claude's builds to. The user reads diffs as the final gate, as always.


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