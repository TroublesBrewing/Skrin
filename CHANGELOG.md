# Changelog

Each milestone in the plan (`~/Documents/vault-1/tui.md`) ships as a minor version, so milestone N is v0.N.0. Fixes between milestones bump the patch number (v0.1.1). v1.0.0 follows milestone 9, once Skrin has held up in daily use.

## v0.18.0 — 2026-09-17

Milestone: keymap overrides — the "then:" item that has been sitting at the top of the plan's tail since v0.10, brought forward by a quick report (`Quick reports/Better keybinding manual.md`): the manual was one long list of bindings, hard to search, with no way to change a key short of editing Skrin's source.

- **New: `?` is three tabs — Keys · Settings · Guide**, cycled with `Tab` and `Shift+Tab`. The Keys tab is now *only* keys, with a visible cursor on the row you're about to act on; the prose that used to sit below them (Files and the note, Split view, Claude, Search, Links, Undo, Config) moved into the Guide, which is what made the key list hard to scan in the first place.
- **New: change any key from the Keys tab.** `enter` on a row waits for the new key and takes the next one you press. Only what you changed is saved, under `[keys]` in config.toml, stored against the context and the name of the action — so a key you chose survives Skrin changing its own defaults for everything else. `r` puts one binding back to Skrin's own, `R` puts them all back after asking, since neither has an undo.
- **New: collisions are caught and fixed on the spot.** A key another binding already holds in that context asks before taking it, naming what has it and whether that would be left with no key at all. Saying yes moves the cursor to the binding it was taken from, so you can give that one a new key there and then rather than discovering later that it's gone.
- **Two refusals, both by name**: a key the component types or handles itself (the editor's `Ctrl-s`, its vim keys, a list's typing) is refused rather than half-taken, saying what has it; and `?` and quit can't be left without a key, because they are how you would get back to this screen to undo a mistake.
- **Finding a key got easier**: `/` now matches both what a key does and the key itself, so searching `ctrl+k` finds the row that displays `Ctrl-k`. `j`/`k` then walk only the rows that matched, and the status line counts them.
- **Fixed, found while testing this live**: `Save` (added in v0.17.0 for the Settings tab) wrote every setting back, filling config.toml with `vault = ""` and an empty table for each section whether or not you'd set anything. It now writes only what is actually set — while still writing a setting you deliberately turned *off*, since that isn't the same as never having chosen.
- **Fixed, same session**: a binding whose key had been given to something else was saved with no keys, and on the next start quietly took its default key back — leaving two bindings claiming it and breaking the one-key-one-meaning rule the manual depends on. An empty list now means what it says, and the binding stays keyless until you give it one. There's a test that reloads such a keymap and checks no key means two things.
- Tests: the keymap layer (overrides applying, unknown overrides ignored, a keyless binding surviving a reload, displacement, reset one/all, and which keys are the component's own); the tab (rebinding end to end and then firing the action for real through the new key, the collision y/n and the follow-on rebind, both refusals, `r` and `R`, the filter finding by key and by description, reload from config.toml, and that the cursor is visible and moves); config `Save` writing only what's set and round-tripping the keymap. Full suite green, and the whole flow driven by hand in a real terminal, including a restart.

## v0.17.0 — 2026-09-17

Milestone: a settings interface, from the last open Quick report (`Quick reports/We need to talk information security features. Lik.md`). The report asked for two things — a way to turn off the privacy risk of Skrin always reopening the last note on start, and a settings menu to put that and future preferences in — both land together here.

- **New: a Settings tab in `?`.** The manual is now two tabs, switched with `Tab`: Keys (everything it showed before, unchanged) and Settings, a `j`/`k` + `enter`/`space` checklist of the everyday config.toml toggles — remember last open note, carry over yesterday's todos, vim keys in the editor, the Claude drawer, image previews. Each toggle saves to config.toml at once, so the UI and hand-editing never fight over it.
- **Fixed: remembering the last open note is now opt-in, off by default.** Skrin used to always reopen whatever note was open when it last quit; a vault used for anything sensitive could pop that note open the moment it started, in front of whoever's looking. A fresh run now starts at the welcome screen unless "Remember last open note" is turned on in Settings (or `restore_last_note = true` in config.toml) — the Files cursor still comes back to where it was, only the note itself waits for you to ask.
- Config gained `[general] restore_last_note` (default off) and a `Save` path so the Settings tab can write back to config.toml without disturbing anything else in it, including a hand-set `~/` vault path.
- Tests: config round-trip for the new setting and `Save` (`internal/config`); UI coverage for the tab switch, toggling and saving a setting, and both the restore-on and restore-off session paths (`internal/ui`). Full suite green.

## v0.16.4 — 2026-09-17

Quick report from daily use (`Quick reports/Alt key navigation in file tree.md`): Alt+↓/↑ skims the file tree, opening each note beside the one already open, but Alt+←/→ were pinned globally to note history back/forward — so there was no way to expand or collapse the folder under the cursor without letting go of Alt first, breaking the "hold Alt and browse" gesture.

- **Fixed: Alt+←/→ now toggle folders in Files, matching what plain h/l already do there**, instead of jumping through note history. The note pane is unaffected — Alt+←/→ still means back/forward when it's focused, and `ctrl+o`/`ctrl+i` still mean back/forward everywhere. Nothing was removed, and the whole tree can now be skimmed and expanded without ever releasing Alt.
- Tests: a new case opening and closing a folder with alt+→/← while Files has focus, alongside the existing history back/forward regression in the note pane (`internal/ui`). Full suite green.

## v0.16.3 — 2026-09-17

Quick report from daily use (`Quick reports/Testing Quick Note window and finding limitation.md`): once a Quick Note capture grew past the overlay's 8-row cap, the cursor scrolled out of the visible window with no way to bring it back — typing just seemed to vanish.

- **Fixed: the Quick Note capture box now scrolls to keep the cursor visible** once its wrapped text outgrows the 8-row growth cap, instead of freezing on the first 8 lines forever. Same pattern the manual overlay (`internal/ui/help.go`) already used for its own scrolling.
- **Related, not fixed here**: Book Card's "Notes & Reflections" field shares the same text-field type and the same unbounded-render pattern, so a long enough note could show the same symptom. Left for its own ticket — the form around it needs its own look before deciding how it should scroll.
- Tests: a new case typing past the 8-row cap and asserting the cursor's own line stays visible in the rendered box (`internal/ui`). Full suite green.

## v0.16.2 — 2026-09-17

Priority-1 bug from daily use (`Quick reports/img rendering bug.md`): a book cover's sixel pixels could smear across the screen, black out the Files tree, blink on every scroll, and sometimes make Skrin stop responding to keys entirely. Two fix passes at the sixel drawing code (tracking and clearing painted rectangles, sequencing clears before draws, deferring decode/encode off the main goroutine, skipping a redraw when nothing changed) stopped the freeze and the blinking, but the ghosting, the oversized cover and the corrupted Files tree persisted — because the real problem was architectural, not a bug in the drawing logic: sixel pixels are painted with raw escape sequences entirely outside Bubble Tea's own cell diffing, so nothing in `internal/ui` could ever reliably clear or clip them once the terminal had rasterized them somewhere.

- **Replaced sixel with block-art thumbnails.** A found image now renders as a small preview built from `▀` half-block characters — the same technique `internal/logo`'s chest art already used for the theme-coloured splash logo, just sampling real colours from the image instead. Because a thumbnail is plain styled text living in `Line.Text`, Bubble Tea's own renderer handles all the clipping, diffing and redrawing for free: there is no more separate paint pass, no raw escape sequences, and no terminal-capability probing (the DA1/cell-size handshake is gone entirely). This is the class of bug the two previous passes couldn't reach, fixed by removing the architecture that caused it rather than patching around it.
- The preview's box is fixed at 28×12 cells, aspect kept — sized by comparing an actual cover rendered at five candidate sizes side by side, big enough to recognise a book cover without ever ballooning across the screen the way sixel sometimes did.
- Decoding a never-before-seen cover still happens off the main goroutine (file IO and image decode can be slow even if the output is tiny), and only when a note that references it is actually opened or resized — not on every keystroke — so a slow or broken file can't be re-attempted every frame.
- `f`/`Enter` on an image embed already opened it with the desktop's default viewer; that's unchanged and is how you see a cover at full size.
- Retested live and confirmed fixed: no more oversized covers, ghosting or Files-tree corruption.
- Tests: placeholder-before-decoding, switching to the real thumbnail once decoded, deferring the decode into the returned `Cmd` rather than doing it inline, and the cache being reused rather than re-decoding on reopen (`internal/ui`); the aspect-fit box-sizing math at several width/height ratios (`internal/markdown`). Full suite green.

## v0.16.1 — 2026-09-17

Three quick reports from daily use (`Quick reports/`), including a first pass at note transclusion.

- **Fixed: the split view's pane-switch key was undiscoverable.** `shift+←/→` already worked and had manual text, but the status line's right-side hint never mentioned it. It now reads `shift+←/→ switch panes · ? manual · ...` whenever a split is open.
- **Fixed: Quick Notes and the Book Card's Notes & Reflections field ran text off the right edge instead of wrapping.** Both now word-wrap to their box width, hard-breaking a single word longer than the width; the cursor stays correctly positioned on its wrapped row.
- **Fixed: images couldn't be found or linked to.** `[[` completion and Go to note only ever looked at notes. Both now also list images — completing `[[` inserts a proper embed target, and choosing an image from Go to note reveals it in Files (it has no note pane of its own to open).
- **New: `![[Note]]` embeds transclude, a first pass.** A block embed (alone on its own line, same rule as image embeds) now shows the target note's actual content in place, instead of sitting as a plain `⧉ name` link forever. `![[Note#Heading]]` transcludes just that heading's section. A `#^blockid` target, or one that doesn't resolve, still falls back to the plain link. Transclusion is one level deep only — content embedded inside a transcluded note renders as a link rather than transcluding again, which is what keeps it safe from cycles. Applies everywhere notes render: the note pane, splits, zen mode and the Claude drawer. **Known limitation**: a link or image inside transcluded content resolves relative to the note doing the embedding, not the embedded note's own folder — right for absolute or unique targets, wrong for a same-named or relative one written differently. No vault spec note exists yet for this feature; it was scoped live rather than through the usual proposal-and-sign-off process, worth a retroactive ticket.
- Tests: a new status-line hint case; `wrapPlain`/`textArea.wrapped` word-wrap and cursor-mapping tests plus both call sites' regressions (`internal/ui`); `Index.Images()` and its wiring into completion and the switcher (`internal/index`, `internal/ui`); transclusion's resolve/fallback/heading-section/block-id/recursion-safety cases at the renderer level (`internal/markdown`) and end-to-end through the note pane (`internal/ui`). Full suite green.

## v0.16.0 — 2026-09-17

Images in notes ([[skrin images]]): image embeds finally render, instead of sitting as plain links — real sixel pixels where the terminal supports them, a tidy placeholder frame everywhere else.

- **A `![[photo.png]]` or `![alt](Assets/photo.png)` alone on its own line renders**: real sixel pixels, scaled to the pane's width and, if that would run past the pane's height, to the pane's height instead (aspect kept either way), on a terminal that has answered it can draw them; a placeholder frame (`▗▖  photo.png · 1920×1080 · 1.2 MB`) everywhere else, or when the file's missing or its format isn't one Skrin can decode. Either way it stays a link — `f`/`Enter` still open it. An embed mixed with other text on the same line keeps rendering as the existing inline `⧉ name` link, unchanged — scoping the feature to the block form the spec's own placeholder mockup shows.
- **Sixel capability is asked for once, at startup**: a DA1 query (attribute `4` means sixel) and a cell-size query (XTWINOPS `16t`), both harmless no-ops on a terminal — like tmux — that doesn't answer them, so the placeholder path is what always runs there. The editor always shows the raw markdown, embeds included — no pixels are ever drawn over it.
- **`[render] images = true`** in `config.toml`, on by default; turning it off forces placeholders everywhere even once the terminal has claimed sixel — a found embed's name, dimensions and size still show either way. No vault with no images ever needs to know the option exists.
- Zen mode, splits and the drawer all render images through the same renderer, the same rules.
- New `internal/imgmeta` reads an image's header only — dimensions and file size — never a full decode, for `image/gif`, `image/jpeg`, `image/png`, `.webp` and `.bmp` (`golang.org/x/image` for the last two, joining Go's own stdlib decoders).
- Decoded sixel payloads are cached by the file's mtime and size, so a note left open isn't re-encoded on every redraw — only when the image file actually changes.
- Tests: header-dimension parsing (`internal/imgmeta`), the placeholder frame's three states (found, missing, unsupported), both embed syntaxes, the inline-stays-unchanged and non-image-embed regressions, the Src mapping across a multi-row image, pane-height capping, and `imageRows`'s own row-count math (`internal/markdown`); the config-off path, the zero-cell-size guard, wikilink-style resolution for embed targets, and frame-fits-at-every-size with an image note open, all confirmed once sixel capability is known (`internal/ui`). **The sixel pixel draw itself is verifiable only in a real terminal that supports it (`foot`) — tmux never claims sixel, so this milestone's live testing here covers the placeholder path only; the pixel draw needs the user's own eyes at sign-off**, same honesty note as the skim-split chords in v0.13.0.

## v0.15.0 — 2026-09-17

Quick notes ([[skrin quick notes]]): `i` opens a small overlay anywhere in Skrin for capturing a thought without leaving what you're doing, then drops it as a new note and closes without opening it.

- **`i` opens the capture overlay** from Files or an open note (main context only, not from inside another overlay). A multi-line text area takes the note's body; `enter` saves, `shift+enter`/`alt+enter` insert a newline (matching the drawer's own convention).
- **The first line names the note**: trimmed, leading `#`s stripped (so starting a line with a heading marker doesn't leak into the filename), cut at 50 runes, then run through the same `vault.CheckName` every other naming path in Skrin uses.
- **Folder defaults to Obsidian's own "new note" location** — `tab` swaps focus to a folder row seeded with that default, typeable and fuzzy-filtered like the other choosers. Per the signed-off ticket ruling: a declared `NewFileLocation` folder is used as-is; unset, `"current"` (quick notes have no note to be relative to), or a folder that's since been deleted all fall back to the vault root. No new settings — it's read live from Obsidian's own config, same as the dangling-wikilink `offerCreate` flow already did.
- **Three refusals, all inline**: an empty first line, a name that collides with an existing note, and a typed folder that doesn't exist. Nothing is written until all three are clear.
- Saves without ever opening the note — the flash names the path it landed at and that `U` undoes it, and `U` really does undo the create.
- Tests: 16 new cases in `internal/ui/quick_note_test.go` covering opening from both contexts, naming (trim, `#`-strip, 50-cut, both newline chords), all three refusals, every folder-default branch (unset, declared, declared-then-deleted), the tab-filter-pick flow, `U`, that it never opens the note, and frame-fits-at-every-size. Full suite green; live-verified in tmux end to end, including the exact flash wording the spec calls for.

## v0.14.0 — 2026-09-17

Habits in the daily notes ([[skrin habits]]): checkboxes under `### Habits` in each day's note, with a `T` overlay for today's list and the week/month grids. Built by Hermes as stand-in builder over the weekend (commit `d514552`), then audited by Claude to the same standard PO reviews hold builder builds to — two real problems found and fixed before release.

- **Habits, in the daily note.** A `### Habits` section (the daily template supplies the starter list, same as `### Todo's`) holds plain checkboxes; `T` opens an overlay on today's list. `H` cycles today → this week → this month, each a grid of every habit against its days; `space` ticks/unticks under the cursor (today) or the grid's highlighted cell (week/month, writing straight to that day's own note, named in the flash); `U` undoes the last tick from inside the overlay.
- **Fixed: habits could roll over, breaking the feature's own core promise.** The real Rollover Daily Todos plugin scans the *entire* previous note for unchecked checkboxes — not just its `templateHeading` section, which only says where rolled todos land, not what counts as one (confirmed against the plugin's actual behaviour, not just assumed). Left as built, an unfinished habit would duplicate into tomorrow's Todo's list, and — with `deleteOnComplete` on — be deleted from yesterday's note outright. `habit.BlockRange` finds the Habits section's line span (the same boundary rule `Parse` uses) so `openDaily` can exclude it before the rollover scan ever sees it.
- **Fixed: the manual's help text for `l`/right on the today tab was wrong.** It said "over to the note"; the key is a deliberate no-op there. Corrected to match `h`/left's phrasing.
- Tests: `habit.BlockRange`'s own unit tests, `TestHabitsNeverRollOverToTomorrow` (the rollover-interference regression, both directions), plus everything already in `internal/habit` and `internal/ui/habit_view.go`'s test suite. Verified live in tmux: a habit rolling correctly *not* over while a real todo does, tick/untick, `U` inside the overlay, the `H` cycle, and the week/month grids.

## v0.13.1 — 2026-09-17

Amendment 1 ([[skrin library]]): the Book Card's lookup stops lying about why it came up empty, and gains two more providers.

- **Honest lookup outcomes.** "Book lookup failed (offline)" used to cover every miss — a timeout, a 5xx, or a provider's honest "I don't have this." Now a clean no-record from every tried provider is its own message ("No match for '{query}' — none of the three has it · continue manually", the query capped at ~24 runes so a long one can't crowd the words out); a provider that fails to answer is named ("Open Library didn't answer — showing Libris' 3 matches") without ending the lookup; "offline" appears only when every provider genuinely failed to answer. A clean single or multi-result match stays silent — the chooser opening is the success signal — but the transient "Looking up…" flash is now always replaced by something, fixing the backlog's stuck-flash report.
- **Google Books joins the chain** (anonymous, keyless, `books/v1/volumes`) as a third independent source alongside Open Library and Libris.
- **Libris gains a free-text fallback.** Its xsearch API already covered ISBNs; it's now also tried (last, after Open Library and Google Books come up empty) for free-text queries, since it's often the only line with anything at all for a Swedish-language search.
- **Fixed: Libris lookups never actually worked.** The XML the provider expected (`<result><list><record>…`) didn't match Libris' real xsearch response shape (`<xsearch records="N"><collection><record>…`), so every Libris call — the ISBN path Swedish books depend on, and the new free-text fallback — silently failed to parse. This is the root cause behind the backlog's "Swedish ISBN lookups never work at all."
- **Cover-quality backfill.** Libris never returns cover art; when it wins an ISBN match with no cover, a best-effort, silent lookup (Open Library first, then Google Books) backfills one without changing whose metadata won.
- Lookup order: ISBN tries Libris → Open Library → Google Books for a Swedish-prefixed ISBN, Open Library → Google Books → Libris otherwise, all three tried, first match wins. Free text tries Open Library → Google Books → Libris.
- No new keys, no config, no manual changes — the provider set is code, not configuration, per the amendment's scope.
- Tests: `internal/book/lookup_test.go` (chain short-circuiting, no-record vs. down, the honest messages word-for-word, the Libris free-text fallback firing only when earlier providers are empty, the cover backfill), `internal/book/googlebooks_test.go` (result mapping, cover-URL https upgrade, no-record), plus rewritten `internal/book/openlibrary_test.go` cases and a new `internal/ui/book_card_test.go` case for the no-longer-stuck flash.

## v0.13.0 — 2026-09-16

Milestone 11 (polish) continues, the skim-in-the-split amendment lands, and the Library & Book Card feature ships. Four separate builds land together in one release: skim-split, the Files-navigation amendment, the split-focus bugfix, and the Book Card.

- **Skim in the split from Files** ([[skrin split view]] Amendment 1). `Alt+↑`/`Alt+↓` (alias `Alt+j`/`Alt+k`) skim a folder's notes into the split pane without leaving Files — the cursor stays put, the split's note follows. Built over the weekend while Claude was out; reviewed and folded into this release.
- **Library & Book Card.** `B` catalogues a book: metadata fetch and cover art from Open Library, a generated note from a template, quotes and reflections. See `skrin library.md` in the vault for the design. Configurable under `[library]` in `config.toml` (`folder`, `covers_folder`, `default_status`).
- **`l`/`→` opens a folder without stepping onto its first child.** The cursor stays on the folder row; an already-open folder flashes instead of no-op-toggling. `Ctrl+↑`/`Ctrl+↓` jump the cursor to the previous/next folder row, so moving between folders no longer means touching every file row between them.
- **`Space` marks or unmarks in place.** It no longer advances the cursor afterward, so marking a range from the bottom up no longer fights the cursor's own movement.
- **Fixed: opening the split's own note used to discard the reference note.** Pressing `l`/`→` or `Enter` on a note already showing in the split pane now swaps focus to that pane, instead of silently overwriting the main pane's note with it. The same guard now covers `e`, `E`, `u`/`Ctrl-r` and `o` on the split's note, which had the identical bug.
- **Fixed: `Shift+→` from Files was a silent no-op.** The skim amendment promises it focuses the split; it now does — swapping the skimmed note into the main pane and giving it focus, matching the promise instead of doing nothing.
- **The order now lives in `skrin.json`, not `.skrin`** ([[skrin arrange]] Amendment 2). The old name was a dotfile, and Obsidian Sync skips dotfiles, so the manual order stayed on one machine even though it's vault data. `skrin.json` isn't a dotfile, so it syncs like any other file: arrange on one machine, and the order is there on the next. Skrin still hides it from the Files tree by name, and it's invisible in Obsidian's explorer too. A vault with the old `.skrin` migrates silently on next open (no journal step); if both names exist, `skrin.json` wins.
- **Error messages point at what's wrong.** Opening a vault at a path that doesn't exist now says so and names the path, instead of a bare `stat ...: no such file or directory`.
- Tests: `folder_jump_test.go`, `book_card_test.go`, and updates across `arrange_test.go`, `files_test.go`, `links_test.go`, `ops_test.go`, `skim_test.go`, `split_test.go`, `ui_test.go` for the new semantics.

## v0.12.0 — 2026-09-16

Milestone 11 continues with the safety stories deferred from the v0.9.0 review: what happens when something else touches the vault at exactly the wrong moment, or the machine stops at the wrong instant. Nothing here changes a key or a screen.

- **A move never replaces what it finds.** Looking first and renaming after left a window where Obsidian Sync or another app could put a file at the destination, and the rename would overwrite it silently. Renames now refuse in the kernel itself (`renameat2` with `RENAME_NOREPLACE`), so the check and the move are one step. Moves, renames, trashing and restoring all go through it, and each says the destination is taken instead of losing what was there.
- **A note Skrin calls saved survives the power going out.** Writes are flushed to the disk before the rename that puts them in place, and the folder entry is flushed after it. Snapshots are flushed too, so `u` still works after a crash.
- **Live updates say when they have holes in them.** Folders that can't be watched (the system's watch limit, on a big vault) and errors from the watcher used to be dropped on the floor, leaving Skrin quietly stale until a restart. Both now reach the status line.
- **A batch move is all or nothing again.** Moving two marked notes that share a name — `a/Plan.md` and `b/Plan.md` — moved the first and failed the second. The clash is caught before anything moves, and the refusal names both.
- **`# Title ##` reads as "Title".** The closing hashes are markup: Obsidian hides them and Skrin's own index already did, but the renderer showed them.
- **A note that can't be read for a moment keeps its place.** While Sync rewrites a note, an index refresh could drop it, so it vanished from search and links until some later refresh caught it. It now keeps what it had.
- The open note's orange is derived from magenta rather than yellow, so a theme that sets no orange of its own can't give the open note the colour Files uses for marks.

## v0.11.0 — 2026-09-16

Milestone 11 opens: arranging loses its mode (Amendment 1 in `skrin arrange.md`), the keymap registry reaches the components that handle their own keys, and the open note is findable in Files again.

- **Arranging has no mode any more.** In Files, `Shift+↑` and `Shift+↓` move the item under the cursor up and down its level, and `R` puts a level back in the default order. The arrows move the cursor; with Shift they move the thing under it. `A`, `J` and `K` are free again.
  - Every move is its own step and says what it did — *"Moved Anteckningar up · U undoes"* — so `U` takes back the last move rather than a whole session.
  - The first move in a vault still creates `.skrin`, and undoing it takes the file away again.
  - `n` `N` `r` `m` `d` are never paused now: there's no mode left to pause them in.
  - The vault row stays put, and at a level's edge the refusal names the level.
- **Components dispatch through the keymap registry.** The search panel's keys and the `[[` completion popup's keys have real actions in the registry and look them up with `actionIn`, so each key is defined in one place — the groundwork for user keymap overrides.
  - The completion popup's keys are listed in the `?` manual for the first time.
  - Keys that are purely a component's own (typing, the editor's vim keys) stay documentation-only, listed in the manual but handled inside the component.
- **The open note is orange in Files.** It was the accent colour, which gruvbox and many other themes also give to folders, so the note you were reading looked like a folder. Files now keeps one colour per meaning: folders blue, marks yellow, the open note orange.
- Tests: the registry's one-key-one-meaning guard moved to `keys_test.go` where it can be found, alongside new guards that every binding is documented and that the components' keys resolve through the registry; a colour test keeps Files' meanings apart.

## v0.10.0 — 2026-09-15

Milestone 10: arrange mode (`skrin arrange.md` in the vault).

- **`A` puts a Files level in your own order.** `J`/`K` move the item under the cursor up or down its level, and `l`/`h` go into a folder and out. `R` puts the level back in the default order, and `A` or `Esc` leaves.
- Within a level anything can sit anywhere, a file above a folder too. New items appear at the end of an ordered level, and renaming or moving an item keeps its place.
- The status line reads ARRANGE with the level's path, and Files' title says *arranging*. `n` `N` `r` `m` `d` are paused while you arrange.
- The whole session is one step, so one `U` undoes it.
- **From the UX engineer's audit of v0.9.0:**
  - Every note-only key refuses the same way: "Select a note to edit / undo / follow its links / see what links here / see its outline / search in just that one / read in zen mode / split beside".
  - In search, `Esc` steps back one level: the first leaves replace mode, the second closes the panel.
  - Marking a range in Files reads MARK in the status line; VISUAL now means selected text in a note.
- The order lives in `.skrin` at the vault root:
  - Paths never change, and Obsidian's own explorer stays alphabetical.
  - Deleting `.skrin` puts everything back.
  - Obsidian Sync skips dotfiles, so the order stays on each machine unless you copy the file over.

## v0.9.0 — 2026-09-15

Milestone 9: instant-open and split view, from the product owner's and the user's notes (`skrin split view.md` in the vault).

- **Notes open under the cursor again.** Moving onto a note in Files opens it straight away, as before v0.6, so `j`/`k` skims the vault. On a folder, the last note stays open, and `Enter` moves over to the note. This was the user's decision after daily use.
- Following a link, a search hit, Go to note, `t` and going back all move the Files cursor to the note they open. The `autoReveal` setting that v0.6 read isn't needed any more.
- **Split view:**
  - In Go to note, `Shift+→` opens the note beside the one you're reading, on the right, and `Shift+←` on the left. `Enter` still opens it in place.
  - In the note, `Shift+←`/`Shift+→` move between the two panes. `Esc` closes the pane you're in, and `z` closes the other.
  - Two is the most, so a new split replaces the older one.
  - Below 80 columns there's no room: a new split is refused, and one that no longer fits closes with a message.
  - Splits aren't remembered across restarts.
  - The Claude drawer and every note key act on the focused pane.
- **Every key is in the keymap registry, with the context it works in:** Files and the note, the editor, lists, search, the drawer, proposals, prompts, save conflicts and the manual. The manual's key tables are generated from it, one section per context, and the drawer, proposal and list keys work through it.
- **Fixes from Hermes's review of v0.5.4**, which the product owner triaged into v0.9:
  - `U` keeps what's on disk first. Undoing a change to a note snapshots its current text, so `u` can bring back an edit made elsewhere in between. If that snapshot can't be kept, the note is left alone.
  - When `t` tidies yesterday's note (the Rollover plugin's "delete todos from previous day"), it snapshots the note first, so `u` restores it, even after a restart.
  - Rolled-over todos only land under a line that *is* the heading, never in a sentence that mentions it.
  - Notes with Windows line endings keep them through the save-conflict flow.
  - `vault.Remove` refuses the vault root, like the other deletes.
  - `editor.external` accepts a `~/…` path.
  - `f` with no note open says so instead of doing nothing.

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
