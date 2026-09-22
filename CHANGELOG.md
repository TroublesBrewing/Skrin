# Changelog

Each milestone in the plan (`~/Documents/vault-1/tui.md`) ships as a minor version, so milestone N is v0.N.0. Fixes between milestones bump the patch number (v0.1.1). v1.0.0 follows milestone 9, once Skrin has held up in daily use.

## v0.52.0 — 2026-09-22

**Settings are divided into Global Settings and Vault Settings.** The idea, from the user's own note ("Separate vaults" in the Idélådan, written right after the vault picker landed): *"Vaults should be separate. They should not share vault specific settings. Keybindings and other global skrin settings are fine, necessary even. A hard demand. But vault specific settings like what template goes with what folder is NOT something to share between vaults. Each vault or skrin … is a blank slate. We should divide settings into Global Settings and Vault Settings."*

- **Vault settings now live in the vault**, in `skrin-settings.json` at the vault root — the same way the manual file order already lives in `skrin.json`. Templates folder and folder→template pairs are the vault's own; they follow the vault through sync and never leak from one vault into another. A second vault is a blank slate, exactly as the note demands.
- **Global settings stay in config.toml** — keybindings, theme, editor, rollover, the Claude drawer, and the rest. Those are about how Skrin behaves on this machine, not about any one vault.
- **The Settings tab shows the split**: global rows first, then a `VAULT` heading, then the two vault rows, and a footer that names both files. The `skrin-settings.json` is hidden from the Files tree like `skrin.json` already is.
- **Existing folder templates migrate automatically**: on the first run after this, Skrin moves the old `[templates]` block out of config.toml and into the vault's `skrin-settings.json`, merging without ever overwriting what the vault already has, then drops the old block so a later save can't lose or duplicate it.

Migration verified live in tmux against the real vault: the two folder pairs (Begrepp, Möten) moved intact into `skrin-settings.json`, config.toml was cleaned, and a second vault opened as a blank slate with no leakage. Full suite green (`-count=1`), `go vet` and `gofmt` clean.

## v0.51.0 — 2026-09-22

**"Open a vault" replaces "Switch vault", and any folder can be opened as a Skrin.** The user, having tried the switch for real: "I think you should be able to select any folder freely. If necessary, Skrin can add whatever settings file it needs … Obsidian has the option to 'Open a folder as a vault' and Skrin should be able to do the same." And, on the name: "we will call it 'Open a folder as a skrin'." So the picker became what Obsidian's is.

- **"Open a vault" (Ctrl+P)** lists every vault it can find: Obsidian's registered vaults, folders beside the current one that carry an `.obsidian` or `.skrin` marker, and the vault open now. Above them sits **"Open a folder as a Skrin…"**, which prompts for a path and opens any folder at all — a plain folder that has never been a vault, nothing to prepare.
- **Opening a folder writes Skrin's marker** — a `.skrin` directory — so the folder is recognised as a vault next time, exactly as Obsidian's `.obsidian` marks one. Nothing else about the folder is touched.
- **Discovery recognises both markers** (`.obsidian` and `.skrin`), so a folder adopted through Skrin shows up alongside Obsidian's own vaults on the next run.
- Choosing any vault — from the list or by path — saves it as the default and reopens Skrin there, in the same terminal.

The `.skrin` marker coexists with the old order file of the same name: the order file was renamed to `skrin.json` long ago, and the migration now only moves a regular file, leaving a `.skrin` directory untouched. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: opened a plain `/tmp` folder with no marker, Skrin adopted it (`.skrin` appeared) and showed its notes.

## v0.50.0 — 2026-09-22

**Switch vault, from inside Skrin.** The user: "Oh, just realized we can't select other vaults. That is core functionality." A basic Obsidian feature Skrin was missing — Obsidian has a vault picker at launch and "Open another vault" from inside — so it lands under the locked-period exception, on the user's call.

- **"Switch vault" is a palette command**, not a key: switching vaults is rare and deliberate, so it stays out of the way until someone asks for it, as the charter's "invisible until asked for" says. Ctrl+P → "switch vault" lists Obsidian's registered vaults, in name order, the one open now marked "open now".
- **Choosing one saves it as the default vault and reopens Skrin there**, in the same terminal — the process hands the terminal to itself in the new vault, so nothing is lost in the handoff. The new default survives the relaunch and every launch after, so Skrin stops silently snapping back to the last-opened vault.
- **It reads Obsidian's own registry** (`obsidian.json`), the same list Obsidian's picker shows — nothing to configure, and a vault Skrin knows about is one Obsidian does too.

The switch is a write to config.toml, so it survives even a launch that fails; `U` doesn't apply here (nothing in the vault changed — a switch is not a file operation). Full suite green (`-count=1`), `go vet` and `gofmt` clean, and the switch verified live in tmux: from one real vault to another, back-to-back, and the new vault held across a relaunch.

## v0.49.0 — 2026-09-22

**Folder templates: a new note starts from the template paired with its folder.** The user, asked for explicitly over the locked period: "Det är en feature som jag saknar flera gånger om dagen … det enbart tjänar till att snabba upp anteckningstagandet och hjälper användaren hålla sig organiserad." A new feature, so it steps over the freeze on the user's word and bumps the minor number.

- **The pairing is explicit, not name-matching.** The user chose it over the original idea of matching a template's name to its folder's. In Settings, a new "Folder templates" row opens the list of pairs; "+ Add a folder…" walks folder → template, Enter on a row changes its template, `d` removes it. A folder can pair with any template, and several folders can share one.
- **A new note in a paired folder is created from the template**, filled in (`{{title}}`, `{{date}}`, `{{time}}` and offsets) exactly as Insert template fills one. No pairing, the note is created empty, as before. The template fills a brand-new, empty note — never overwrites anything — and `U` undoes the whole create in one step.
- **A folder matches exactly — no inheritance.** Personer and Personer/Vänner can each have their own template; a note in a subfolder with no rule of its own is created empty, the parent's rule does not leak down.
- **The pairing is stored as the template's full vault path**, under `[[templates.rules]]` in `config.toml`, so it survives a change of templates folder. An empty list means the feature is entirely off — no new key in the app itself, nothing in the way.
- A template that has gone missing is read as no template: the note is still created, just empty, and nothing is lost.

Manual and registry carry the new Settings row and the list's `d` key. Tests cover the templated create, the empty default, exact-match-only (no inheritance), and the add/remove flows saving to `config.toml`. Full suite green (`-count=1`), `go vet` and `gofmt` clean.

## v0.48.0 — 2026-09-22

**Paste reached the Book Card's fields nowhere, and nowhere did a pasted paragraph stay a paragraph.** Found while checking the board; both are the user's own reports in effect. Bugfixes, so both land inside the locked period.

- **The Book Card takes paste now.** The user: "It's not possible to paste information in to the Book card." A paste doesn't arrive as a key press — it comes in as `tea.PasteMsg`, which the model hands to `m.paste` in `internal/ui/edit.go`. That switch names every surface that takes typing (the editor, the search field, the prompt, Quick Note, the Claude drawer, the form) — but never `m.book`. Typed text landed in a card field fine, because card keys are handled in `bookCardKey`; pasted text went nowhere, because the card was simply not a case. The card now has its own `pasteInto`, and every stop that takes typing takes paste: the fetch bar, all sixteen bibliographic fields, each quote's text, page and speaker, and Notes & Reflections. A break becomes a space in a one-line field and a real newline in Notes, exactly as in the same fields elsewhere. `Ctrl+V` on the Save button or an area heading now refuses out loud ("Nothing to paste into here: Tab to a field first") instead of answering "open a note in the editor first" at a card that is open and focused. Verified live in tmux: the same payload pasted into Title, a quote row and Notes, then saved and read back from disk.

- **A terminal paste that lands nowhere no longer goes silent.** `tea.PasteMsg` was passed to `m.paste` and its result discarded, so Ctrl+Shift+V anywhere it couldn't land — the reading view, the card's Save button — did nothing at all with no word about it. It now takes the same refusals as `Ctrl+V`.

- **A pasted paragraph stays a paragraph.** The user's terminal sends CR for a line break in a bracketed paste — the Enter key — not LF. The editor normalised CRLF but not a bare CR, so a two-line paste collapsed onto one line; captured live as `ONE\rTWO` in a saved note. `editor.NormalizeNewlines` now treats CRLF, bare CR and LF each as one break, and the editor, the Book Card's `textArea` and the one-line `lineInput` all use it. This one predates the card fix — the editor had it all along — which is why it is fixed in the shared buffer rather than at the card. Tests paste with CR, as a terminal actually does; an LF-only test passes while the real thing is broken.

- Manual and registry updated: the copy/paste paragraph now names the Book Card, and the card's key group has its own `Ctrl+V` row.
- Keymap test and the full suite green (`-count=1`), `go vet` and `gofmt` clean.

## v0.47.0 — 2026-09-22

**Fixed four weeds, each reproduced live before the fix and verified live after it.** The user's ruling that started it: "Jag bedömer dem som buggar eftersom det är ett helt oönskat beteende ... Det är som ogräs. Allt som växer där man inte vill ha saker är ogräs, oavsett om det är en ros eller maskros." Four behaviours none of which was asked for, in one release — with the change to link-following being the one that shows in the flow, hence the minor bump.

- **Link completion now follows the key that wrote, not the cursor that moved.** The user: "Jag får fortfarande upp en popup vid en länk i edit mode när jag rör markören över länken." `updateCompletion` asked `LinkQuery()` after every editor key, so arrowing through an existing `[[link]]` popped the suggestions up as if something had been typed. The editor now counts every change to the text (`push()` bumps it, cursor movement never does), and the popup only answers a key whose count moved since the popup last looked — so it opens exactly when something is written inside `[[`, refreshes while the writing lasts, and never opens or reopens on movement, Esc stays dismissed, `]]` still closes it. Regression test added: skimming right over both links in a note, arrowing down a wrapped line — no popup; typing `[[Sto` — popup; Esc, arrows again — still nothing.
- **Following a missing link creates the note at once, as Obsidian does.** No question in the way anymore: `f` on a dangling link makes the note where Obsidian would put it, the flash names what was made, and `U` takes it away again. The old yes/no confirm is gone — a click on a missing note in Obsidian never asks, and neither does Skrin now.
- **A declared new-note folder that doesn't exist is treated as unset.** `newFileFolderPath` pointing at a missing folder no longer makes Skrin create the folder — the note lands in the vault root instead, because making folders is a write nobody asked for. (Obsidian in the same situation silently creates the file in the root.)
- **Quick Note's arrows now move through the rows the eye sees.** The user: "Jag kan inte navigera mellan rader i Quick Note eftersom (gissar jag) rutan tolkar min input som 1 lång rad istället för 2 rader vilket är vad jag ser iom wrappingen." `textArea.moveVert` was moving between raw `\n` lines, so Up/Down did nothing on a wrapped line. It now builds the display rows from the same wrap the renderer draws and moves through them, keeping the eye's column — in Quick Note and the book card's Notes field alike. Beyond the text the arrows still leave the field, and with no wrap in force (width never set) they move by raw lines as before.
- **A Quick Note whose first line is a wiki link can be saved.** `[[Note|alias]]` names the note after its alias, `[[Note#Heading]]` after the note — as a link is read — instead of being refused outright for its brackets. A first line of nothing but markup still asks for a name rather than inventing one.
- Key-repeat lag (holding an arrow key) could not be reproduced except as a terminal artifact and is unaffected by these changes; if it still shows in real use, that report is new information.
- Full suite green (`-count=1`), `go vet` and `gofmt` clean.

## v0.46.1 — 2026-09-21

**Fix: closed folders kept reopening after quit.** The user: "Om jag stänger alla mappar och sedan avslutar programmet ... så är vissa mappar alltid öppna igen när jag sedan öppnar programmet." The everyday way to close a folder — `Enter` on it, `toggle()` in `internal/ui/files.go` — set its `expanded` entry to `false` rather than deleting it. `openFolders()`, which feeds the saved session, only checked whether the key existed, not its value, so any folder ever opened and closed during a run came back open next launch.

- `openFolders()` now includes a folder only when its value is `true`. `collapseAll()` and `out()` were already correct — they delete the key outright — which is why the bug only showed on folders closed with `Enter`.
- Test added: open with `toggle()`, close again the same way, assert `openFolders()` is empty. Fails without the fix, passes with it. Full suite green, `go vet` clean.

## v0.46.0 — 2026-09-21

**Removed: the paper grain.** The user, after trying every strength: "Jag ser ingen skillnad i faint, medium och strong. Men vet du vad? Vi struntar i det och slopar den featuren." Out it goes — `internal/ui/paper.go`, its tests, its Settings row, its config keys and its paragraph in the Guide.

- **This is what the beta block is for.** An experiment was built, lived with, judged and deleted, without ever being part of what Skrin promises. Nothing about the removal touches a note, a key or a default.
- **Worth recording, since it wasn't settled:** at *strong* the shades differ by twenty-eight steps of 255, which is not subtle.
- **Correction, checked after the release:** the first guess here was that the grain had never been drawn, from an old running binary. That was wrong. The running process was v0.45.2 — the fixed version, the one whose chooser the user used — in foot directly, no tmux, `COLORTERM=truecolor`, opaque background `#FFFCF0`, which is exactly what Skrin computed from. So the grain was drawn, with shades that should have been visible, and wasn't seen. The one mechanism that would explain it and hasn't been ruled out is the colour profile being taken as 256 rather than 24-bit somewhere inside the drawing, which would snap all three shades onto the same palette entry and show nothing while the rest of the screen looked right. Unproven, and not worth chasing for a feature nobody wants. The idea is in the git history if it is ever worth another look, and the honest verdict stands: a terminal has only cell backgrounds to work with, which is a thin way to make paper.
- `[paper]` in `config.toml` is ignored from here, and can be deleted. Full suite green (`-count=1`), `go vet` and `gofmt` clean.

## v0.45.2 — 2026-09-21

**Fix: the paper grain was invisible.** The user: "Papperstexturen syns inte eller gör väldigt minimal skillnad som inte går att uppfatta på min skärm." It was being painted — it just couldn't be seen, for two reasons, both arithmetic rather than opinion.

- **The default was three steps of 255**, about one per cent. On flexoki-light that gave 252,249,237 against 255,252,240, which no screen shows. The default is now six, and Settings offers **off, faint (4), medium (8) or strong (14)** on one row, so the number that decides everything isn't hidden in a config file.
- **Half the grain was clipping.** The shades were spread either side of the background, but a near-white background has nowhere to lighten to: 255,252,240 plus three is still 255. The grain now goes the way there is room — darker on a light theme, lighter on a dark one — so every shade is its own colour. On flexoki-light at medium that is 255,252,240 → 247,244,232 → 239,236,224, a range of sixteen steps instead of six.
- Tests: the shades being distinct at each strength, and on both a light and a dark palette, so a clipped one can't come back unnoticed. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux that the row and its chooser work; how it looks is still yours to judge in foot.

## v0.45.1 — 2026-09-21

**Fix: Shift+↑ and Shift+↓ select in the reading view too.** The user: "upp- och nerpilar måste kunna markera i view mode också. En del användare är vana vid det och pilarna fungerar på alla andra ställen." Quite right — Shift+arrow already selected in the editor and in every text field, so the reading view was the odd one out.

- **Shift+↑ / Shift+↓ stretch the selection**, and light the cursor themselves when it isn't lit, which is what Shift+arrow does in a document with no selection yet. `J`/`K` do the same, as before.
- **In Files those two keys still order an item** in your own order. One key, two panes, one row in the manual saying both — the same shape `l` and `→` already have.
- **A bug the test found:** the cursor lit on the *last* line of a note shorter than the pane, because "the middle of the view" counted the pane's height rather than the text in it. It now lands in the middle of what is actually on screen, so there is room to stretch either way.
- Tests: Shift+arrow lighting the cursor and growing from nothing, stretching back the other way, and the same keys still ordering in Files. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: two, four, then three lines.

## v0.45.0 — 2026-09-20

**Fix: you can pick lines out of the middle of a note.** The user, on the selection in the reading view: "Buggen med att inte kunna markera ordentligt i läsvyn har irriterat mig. Jag trodde att det var så det skulle vara och att det bara var helt värdelöst." It wasn't meant to be: the reading view has no cursor, only a scroll position, so `v` had nothing to anchor to and took the top line of the view. In a note that fits on the screen there is nothing to scroll, so a selection could only ever start at the first line.

- **`v` now lights a cursor** where your eye already is: the middle of what you are looking at, or the line you last left it on in this note. Reading itself is unchanged — `j` and `k` scroll, and nothing is picked out until you ask.
- **`j`/`k` move the cursor** and take the selection back to that one line, as an arrow key drops a selection anywhere else. **`J`/`K` stretch it** down and up — the bigger version of the same keys, which is what the case grammar says uppercase means.
- **The status line says what to do**: `J/K select · x pull out · ctrl+c copy · esc clear`.
- **Two more silent keys answered:** `shift+up`/`shift+down` (your own order) did nothing at all with the note focused, and now say where that lives, like marking and folders already did.
- **And one small lie fixed:** with a blank line selected the status line showed the whole note's word count, because an empty selection read as no selection. It now says `0 words selected`, which is what is true.
- Tests: the cursor landing in the middle rather than at the top; plain motions moving it while J/K stretch; a block in the middle of a note that fits on screen — the case that was impossible — coming out right; the cursor returning where it was left; and the order keys answering. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux.

## v0.44.0 — 2026-09-20

**New: pull the selected lines into a note of their own.** The idea behind Obsidian's Note Refactor plugin, which the user pointed at — the part of it that earns its place. This is the other half of capture: what was caught quickly, in a daily note, becomes something that can be found.

- **`x` in the note, `Alt+X` in the editor**, with lines selected: they become a new note and a `[[link]]` takes their place.
- **The first line names it**, with its markup taken off — `## Stoicism`, `- [ ] ring the plumber` and `> a quote` all name themselves. Characters a file name can't hold go, as in Obsidian.
- **When the first line gives nothing to go on, Skrin asks.** It never invents a name: a note called "2026-09-20 1423" is one you never find again.
- **It never writes over a note that exists** — it asks for another name, with the reason shown.
- **One U undoes the whole thing**, the new note and the hole it left, because they are one entry in the journal. With the editor open, the file is brought in line with the buffer first, so what the journal records is what was really there.
- **Whole lines only.** A block of text is what becomes a note; half a sentence isn't one.
- **Two fixes that fell out of building it:** a prompt opened while the editor was open never received any keys — the editor swallowed them — so a prompt now takes precedence, as a modal question should. And the lines are held from the moment you ask, so answering a question can't change what moves.
- Tests: the round trip in the reading view with U putting both halves back; the same from the editor; prose naming itself; an empty first line being asked about and the answer naming it; refusing to overwrite; nothing selected saying what to do; and the name-cleaning rules. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux, including U restoring the note exactly.

**Not built, deliberately:** splitting a whole note at its headings (step two, and it needs a preview of what it would create first), the prefix-as-file-name command, and transclusion by default.

## v0.43.0 — 2026-09-20

**New: a built-in light palette, and (beta) paper grain under the note.** Both asked for by the user: "göra nuvarande aktiva tema (flexoki-light) till standardtemat för ljust tema … och jag vill att bakgrunden ska vara väldigt fint texturerat, så att det påminner om papper."

- **flexoki-light is built in.** Skrin has carried one palette of its own, gruvbox, for machines with no Omarchy theme to follow — which is every Mac. It now carries two, and Settings has a row to choose between them: *Colours when there is no system theme*. A system theme still wins over both, and removing one now falls back to the palette you chose rather than always to the dark one.
- **Paper grain (beta):** a faint texture behind the note, one shade either side of the theme's own background. `paper.strength` in `config.toml` (1–10, 3 by default) says how far apart the shades sit.
- **How it's drawn:** a terminal has no texture, only cell backgrounds, so the grain is painted in runs of four cells rather than cell by cell. Each run has to repeat the styling of the text above it; at one run per cell a screenful costs some hundred kilobytes a frame, and in runs it costs a fraction of that. The pattern is a hash of row and column, so it holds still while you scroll instead of shimmering.
- **What it costs, plainly:** it paints the note pane's background, so a transparent terminal turns solid there, and it is more to draw than plain text — not noticeable locally, possibly over ssh. Hence beta, and off by default.
- Tests: the grain leaving every character and every cell of width exactly as it was, including wide runes; holding still between frames and differing between rows; staying off until both switches are on; staying inside the borders; the built-in palettes reading light and dark; an unknown name in the config still giving colours; and the fallback following the chosen built-in. Full suite green (`-count=1`), `go vet` and `gofmt` clean.
- **Not verified by eye.** This sandbox's tmux flattens colours a shade apart, so whether the grain reads as paper or as dirt is a judgement to make in foot. The tests prove it is painted, not that it is beautiful.

## v0.42.0 — 2026-09-20

**New (beta): Habiton, and streaks.** Step one of building the habit tracker out rather than cutting it, named and shaped by the user: "Karaktären heter Habiton och han jobbar inte med skam eller med ord. Han är ordlös men uttrycksfull på sitt sätt. Kroppsspråk är kommunikationsvägen."

- **Habiton stands beside today's list**, pixel art in half-blocks in the theme's own colours, the same technique as the logo. Four states, all body language: **asleep** until something is ticked, **awake** once it is, **arms up** when the day is done, and **sleepy** when little has been ticked over the last few recorded days.
- **No shame, no words.** The worst he does is look short of sleep. A day with no note at all is not counted against you — unrecorded is not the same as undone.
- **He blinks**, three seconds open and 140 ms shut, and nothing else moves. The moods drawn with his eyes already closed don't blink at all.
- **Streaks per habit**, counted back from today through the daily notes, shown in their own column. A single day isn't a streak and doesn't show: every start would otherwise look like a failure.
- **Nothing is stored.** His mood and every streak are read from your checkboxes each time. There is no score file, nothing to sync, nothing to corrupt, and deleting Skrin leaves exactly the markdown you wrote.
- The month behind today is read when the view opens and after each tick, never while drawing.
- Tests: the moods from the habits themselves, including unrecorded days not making him tired and a good day after a bad stretch still reading as bright; that he is drawn and blinks; that the same notes always give the same mood; the streak column in the box, with a single day not showing; and the blink starting with the overlay and stopping with it. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux at four days' running and at none.

**Step two, when there's history to do it on:** what a missed day costs, what a long break does, where any levels sit. Those are numbers that should come from real use rather than be invented now.

## v0.41.0 — 2026-09-20

**New: a BETA block in Settings, and one line that leaves every experiment out of a release.** The user's idea, on deciding the habit tracker should be built out rather than cut: "kan vi typ skapa ett Beta-läge … Och så kan man enkelt utesluta alla beta-saker från 1.0?"

- **Settings ends in a BETA block**, set apart and labelled "experiments, which a release can leave out". It takes two switches: *Beta features* opens the block, and each experiment has its own row inside it. Both are off by default and both are saved to `config.toml` (`[beta] enabled`, and the feature's own key).
- **`version.Beta` is the release switch.** With it false, the beta block disappears from Settings and every experiment reads as off whatever `config.toml` says. v1.0 ships with it false, so leaving the experiments out is one line rather than a judgement call per feature.
- **The habit tracker is the first experiment**, moved into the block. `T` now says it's a beta feature and where to switch it on. Its checkboxes are plain markdown and unaffected, including the rule that keeps them out of tomorrow's todos.
- This gives an impulse a home: something can be built, switched on and lived with for a month without becoming part of what Skrin promises. It is the mechanism the steering document's locked period was reaching for.
- Tests: an experiment hidden until beta mode is on; both switches turning it on and being written to the config; beta off taking every experiment with it whatever its own switch says; and the shape that makes the release switch work. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux, ending with the habits view open again through both switches.

## v0.40.0 — 2026-09-20

**Changed: the habit tracker is off until you switch it on.** The user, looking at whether it belongs in Skrin at all: "vanor känns som något jag ville ha i stunden … Men det måste vara något man slår på i settings." It ships off, as the north star says a feature should — invisible until asked for, like the Claude drawer when it's switched off.

- **Settings has a "Habit tracker" row**, off by default, saved to `config.toml` as `[habits] enabled`.
- **Off, it's out of the way entirely:** no palette row, no tip of the day, and `T` says where to turn it on rather than doing nothing. On, `T` is exactly what it was.
- **Your checkboxes are untouched either way.** `### Habits` in a daily note is plain markdown, and the rule that keeps those boxes out of tomorrow's todos is data behaviour, not part of the view — it holds whether the tracker is on or off, and there's a test for both.
- **Note for the upgrade:** `T` will say the tracker is off the first time. One trip to Settings (`?`, then Tab) brings it back, and it stays on from then on.
- The Claude drawer's Settings text claimed "off hides it entirely, including from the manual". It doesn't hide from the Keys tab, and never did; the text now says what actually happens.
- Tests: off by default with `T`, the palette and every day's tip staying quiet; on, `T` opens as before; the Settings row switching it and writing the config; and habits not rolling over with the switch either way. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux.

Whether it stays at all is the open question on the board card *Vanor som ett riktigt plugin*: built out properly, or taken out. It won't ship half-done.

## v0.39.1 — 2026-09-20

**Fix: the selection count said something untrue, and then got cut in half.** Both found by the user asking whether the editor showed a selection count at all — it did, but not correctly.

- **Shift+Down once, from the start of a line, said "2 lines selected".** The selection ends at the start of the next line, so it covers one line; the count added the empty tail. A selection ending at column 0 no longer counts that line.
- **One measure, never two.** v0.39.0 put a word count beside the line count, and in the editor the pair ran out of room and was truncated. The status line now shows the lines when the selection spans more than one, and the words when it sits inside a single line, where "1 line selected" says nothing anyway. The user's call: "Antingen visas ord ELLER så visas rad … Kapad information riskerar ju att ändå inte vara till någon nytta."
- With nothing selected the whole note's word count is unchanged.
- Tests: the existing selection test was rewritten to the decided rule — its old expectations of "1 line selected" for one line and "3 lines selected" for two Shift+Downs were what the fix changed, so they are gone rather than adjusted around. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: one Shift+Down reads "1 word selected", two read "2 lines selected".

## v0.39.0 — 2026-09-20

**New: five core features Obsidian has and Skrin didn't.** From the comparison the user asked for, built in one pass on their instruction: "Bygg allt ihop, gör kort av dem, se till att allt går rätt till." Each has its own card on the board.

- **Footnotes render.** `[^why]` in the text and `[^why]: …` at the bottom both show as `[1]`, numbered in the order the labels first appear, as Obsidian numbers them. The label is markup and never shown. A reference with no definition, or a definition with no reference, still renders rather than vanishing. Until now they were raw text: a note with footnotes looked broken.
- **Pinned notes — Obsidian's bookmarks.** `p` pins the note you're on or takes it off; `P` lists them. Recents answer "where was I", pins answer "where do I live". Kept per vault, survives a restart, and a pin whose note has gone falls out of the list on the next reload rather than pointing at nothing.
- **A tag panel.** `#` lists every tag in the vault with how many notes use it, and picking one searches for it — the same `#tag` query you'd type, not a second way to search. Each spelling stands on its own row, as in the tag suggestions, so "Filosofi" and "filosofi" show side by side instead of hiding one another.
- **`/regex/` in search**, as in Obsidian. The slashes hold the pattern together across spaces the way quotes hold a phrase, and match-case applies to the regex too. A pattern that won't compile says so in the panel — `/ar (/ isn't a regular expression: missing closing )` — and is left out so the rest of the query still works, rather than quietly matching nothing.
- **A word count** in the status line, reading and writing, counting the selection instead when there is one. Frontmatter doesn't count, and neither do a heading's hashes or a bullet.
- Tests: three for footnote numbering and orphans; five for pinning, the list, the empty case, the restart and a vanished note; three for the tag panel; regex matching, case, and a broken pattern leaving the query working; and word counting with frontmatter, markup and a selection. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: footnotes as `[1]`/`[2]`, `p` pinning, `#` listing both spellings with counts, `/ar \d{4}/` finding the line and `/ar (/` explaining itself.

## v0.38.1 — 2026-09-20

**Fix: three keys in the note did nothing and said nothing**, against the charter's flow rule that no key is silent. Found by a sweep for gaps between the UX charter and the code, after Enter's own gap had sat unbuilt for ten releases.

- **`space`, `Ctrl+A` (marking) and `H` (close all folders)** pressed while the note has focus now say where the thing they act on lives: "Marking is for Files · h goes back there". The key named is whatever `h` has been overridden to.
- **The one-key-one-meaning test now exists.** A hand-off from 2026-09-15 claimed the keymap constitution's rule 4 was covered by a test; the UX audit that September found it wasn't, and it still wasn't. `TestOneKeyMeansOneThingInAContext` checks every binding, with vim mode as the one documented exception, since its rows sit in the editor's context but only apply when `editor.vim` is on. The registry passes as it stands.
- Also checked in the same sweep and found already in order: the "Select a note to <action>" wording across every refusal, and Esc stepping back one level in the search panel with replace on. Full suite green (`-count=1`), `go vet` and `gofmt` clean.

## v0.38.0 — 2026-09-20

**Changed: Enter edits. Links are followed with f, and only f.** The opening model was decided and frozen in the UX charter on 2026-09-19, but the code still had Enter follow a lone link. The user hit it daily: "Enter måste leda till 'edit note' … Jag bygger ju upp en vana (och gör fel vilket skickar mig bort från mina anteckningar redan innan jag börjat skriva på dem)." This makes the code say what the charter says.

- **Enter enters, to write.** On a note being read it opens the editor; on a note under the Files cursor it does what it already did; on a folder it opens or closes it; on any other file it opens the file in its own app. It never follows a link any more, so a habit built on Enter can't take you away from the note you meant to write in.
- **f follows links, and only f.** With one link in view it goes straight there — what Enter used to do. With several it puts a letter on each, as before.
- **Alt+F is f into a split:** one link in view opens beside the note at once; several put letters up, where any letter now opens beside without needing Alt as well. Alt means "into a split" wherever a note can open.
- **`l` in the note isn't silent any more:** "Already here · enter edits this note · f follows a link". No key in the note is silent, per the charter.
- The manual, the Guide, the tip of the day and the plan note now all say the same thing; the plan note points at the charter rather than repeating it.
- Tests: Enter on a note with exactly one link in view opens the editor rather than following it; f follows that lone link; Alt+F takes it into a split, scrolled to the heading it names; `l` in the note answers. The old test for Enter-follows-a-link is gone, replaced by these. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux on the real vault and a scratch one.

## v0.37.0 — 2026-09-20

**New: the editor saves by itself.** Data integrity, on the user's call: "Autosave är en viktig del i data integrity så jag ser det som både en nödvändighet och en av Obsidians basfunktioner." Obsidian has no save command at all, and until now the one way to lose text in Skrin was for the process to die with the editor open — a closed terminal, a dropped ssh session, a crash.

- **A second and a half after you type, it's on disk.** It's a ceiling, not a pause: typing on doesn't put it off. Nothing is said and nothing moves; the ● by the note's name goes out, which is the whole of it.
- **Ctrl+S stays** as "save now", for the habit and for forcing a write before switching away. Leaving the editor still saves, as it always did.
- **A save by itself never decides anything for you.** If the note changed on disk meanwhile (Obsidian, Sync, another editor), writing yours would throw theirs away, so it doesn't write: that choice belongs to Ctrl+S or leaving the editor, where it has always been asked.
- **And it never goes quiet about it.** The status line turns to `HELD  <note>  changed on disk · nothing of yours is written until ctrl+s or esc` and stays that way while you type, because from that moment until you answer, what you write is only in the editor. Answering the conflict ends the hold.
- **What belongs to leaving the note stays there:** `due::` dates and the offer to bring a heading's links along still happen when you leave, not while you write.
- **One snapshot per editing session, as before**, so `u` reaches past every save by itself to the note as it was when you opened it.
- Tests: the clock starting on typing and the tick writing silently; the clock re-arming while there's more to write; a changed file holding the write, showing HELD, and Ctrl+S then raising the conflict and `m` ending the hold; `u` reaching past three saves; dates resolving only on leaving; the heading question only on leaving; a late tick after the editor closed doing nothing. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: typed text was on disk three seconds later without Ctrl+S, an outside write turned the status line to HELD, and Ctrl+S then `m` kept the user's version.

## v0.36.0 — 2026-09-20

**Fix: Ctrl+V pastes.** Copying with Ctrl+C worked, pasting didn't: Skrin only ever took text from a bracketed paste (Ctrl+Shift+V), and a plain Ctrl+V fell through to nothing. Reported by the user: "vi har ctrl+c för att kopiera men ctrl+v fungerar inte för att klistra in."

- **Ctrl+V pastes wherever you're typing:** the editor, the find field, search, the quick note, the table form, the Claude drawer, a list filter.
- **How it works:** a program in a terminal can't reach the system clipboard itself, so Ctrl+V asks the terminal for it over OSC 52, the same channel Ctrl+C copies over.
- **Terminals that won't hand the clipboard back** (reading it would let anything at the far end of an ssh session see what you copied) get a fallback after 300 ms: what you last copied in Skrin goes in, and the status line says that's what happened. With nothing copied here either, it says Ctrl+Shift+V is the way in.
- **An empty clipboard** says so, rather than looking like a failure.
- **Ctrl+V where nothing takes text**, in the reading view, says so instead of doing nothing.
- **The manual says all this now:** `?` lists Ctrl+C as copy and Ctrl+V as paste in the editor, which it never did, and the Guide has a paragraph on copying, pasting and why Ctrl+Shift+V exists.
- Tests: the terminal's answer landing in the editor; the fallback to what Ctrl+C copied, with the flash that explains it; the silent terminal with nothing copied; an empty clipboard; pasting where nothing takes text; and a bracketed paste reaching the search and find fields. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: copy then Ctrl+V pasted the line, an empty clipboard said so, and Ctrl+Shift+V still pastes as before.

## v0.35.0 — 2026-09-20

**New: renaming a heading brings its links along.** The last requirement on the v1.0 bar that wasn't code yet, and a basic Obsidian feature: built during the locked period with the user's yes to this card.

- **Change a heading in the editor** and, when you leave the note, Skrin says which links point at the old name and offers to update them: `"Morning" is now "Sunrise": update 4 links in 3 notes so they keep working?` — `y` updates, `n` or Esc leaves them.
- **Both kinds follow:** `[[Note#Heading]]` and `[[Note#Heading|alias]]` in other notes, `[[#Heading]]` in the note itself, embeds, markdown links, and a heading path like `[[Note#Outer#Inner]]`, where only the part that was renamed changes. A `^block` id is never touched.
- **One U undoes the link edits**, and leaves your heading as you typed it: renaming the heading and updating the links are two things, and you answered them separately.
- **What counts as a rename:** the headings that are still there hold as anchors, and in the gaps a heading gone with one arrived in its place, at the same level, is the rename. Adding or removing a heading is not a rename, nor is changing only spacing or case (those links still resolve), nor is a heading whose name lives on elsewhere in the note.
- Under the hood: `index.ParseHeadings`, `AllLinksTo`, `SameHeading` and `RenameSub`, an `Edit.Sub` that rewrites a link's `#sub`, and the write loop shared with moves (`applyLinkEdits`) so both are snapshotted and undone the same way.
- Tests: eight cases for what is and isn't a rename, the full offer through the keys with links in three notes, `U` putting only the links back, `n` changing nothing, and a heading nobody links to asking nothing. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux on a scratch vault: 3 links updated across two notes, and `U` put them back.

## v0.34.0 — 2026-09-20

**New: Ctrl+F finds text in the note you're reading too.** The same field as in the editor, now also in the reading view. A basic Obsidian feature, built during the locked period with the user's yes to this card.

- **Ctrl+F while reading** opens the find field without going into the editor. Each letter jumps to the next match from where you were, the note scrolls to it, and the match is highlighted where it stands.
- **Enter or ↓ goes to the next match, and Shift+Enter or ↑ to the one before**, round the note, with `2 of 5` in the status line. `no match` leaves the note exactly where it was.
- **Esc closes it where you are**, keeping the scroll position. In the editor nothing changes: the match stays selected, so typing replaces it.
- **The palette has *Find in the note*** in the reading view as well as in the editor.
- Matches are counted on the rendered line, so a match inside a table or a list is highlighted in the right place and wide characters line up.
- Tests: the count and the highlight while reading; a match below the fold scrolls the note; stepping round both ways; Esc closing without leaving anything highlighted; no match leaving the scroll alone; and Ctrl+F with no note open saying so. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: `mall` in *Om mallarna* found 5, scrolled to each, went round and left the view at 12%.

## v0.33.0 — 2026-09-20

**New: Go to note offers the notes you opened lately.** From the wishlist, where it was the cheap thing to try before tabs. A basic Obsidian feature (its quick switcher does the same), built during the locked period with the user's yes to this card.

- **`g` with nothing typed** lists the last ten notes you opened, newest first, marked "opened lately", before the rest of the vault. Typing searches the whole vault as before.
- **Only opening a note counts:** `l`, Enter, a link, a search hit, Go to note, `t`, or opening it in the editor. The cursor passing over a note in Files doesn't, or the list would fill with everything skimmed by.
- **It survives a restart**, kept per vault in the session beside the open folders and the cursor. A note that has since gone is left out.
- Tests: a note opened counts and one only passed doesn't; newest first without repeats and capped at ten; kept across a restart with vanished notes dropped; and a note in the vault root reading without a dangling separator. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: `g` listed the note opened before the current one at the top.

## v0.32.0 — 2026-09-19

**New: find in the note while editing, with Ctrl+F.** Planned from the honest review ("Ingen sökning i editorn … den största enskilda friktionen för någon som skriver på riktigt"). A basic Obsidian feature, built during the locked period with the user's yes to this card.

- **Ctrl+F in the editor** opens a find field in the status line. Each letter jumps to the next match from where the cursor was, and the match shows selected. It works in vim mode too.
- **Enter or ↓ goes to the next match, and Shift+Enter or ↑ to the one before**, round the note. The status line says which match of how many, `2 of 6`, or `no match`, in which case the cursor goes back to where it was.
- **Esc closes it with the match still selected**, as in Obsidian, so typing replaces it. A second Esc clears the selection, as it always has. Case doesn't matter. Pasting goes into the field.
- Also in the palette, "Find in the note", and in the editor's status-line hints (`ctrl+f find`). The Guide explains it under *Search*.
- New: `editor.Find`, `Show` and `MoveTo`.
- Tests: finding ignores case and doesn't overlap, `Show` selects, `MoveTo` clears; in Skrin, the first match after the cursor is selected and the note is untouched while typing, the count, moving forward and back with wrapping, Esc leaving the match selected so typing replaces it, no match going back to the cursor, and a paste searching. The test helper learned Shift+Enter, which it used to read as the letter s. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: "låsning" found `1 of 6`, Enter moved to `2 of 6`, and Esc left it selected.

## v0.31.0 — 2026-09-19

**New: a readable line length.** From the investigation *Läsbarhet*. The user found zen "mycket mer läsbart", works in full screen, and found long, uneven lines hardest: "raderna varierar mycket i längd vilket ger texten ett spretigt utseende". A basic Obsidian feature ("Readable line length"), built during the locked period with the user's yes to this card.

- **Notes stay at most 80 characters wide outside zen**, the same width as zen, so both read alike. The text column sits centred in its pane however wide the window is. The editor follows it, as does the note in a split, each centred in its own pane.
- **On by default**, with a Settings row, *Readable line length*, to turn it off so notes fill their pane again. Saved as `[render] readable_width` in `config.toml`.
- A pane narrower than 80 characters keeps its whole width, with no margin, and zen keeps its own centring. The splash screen isn't shifted.
- The margin sits outside each line, so link hints, selections, line numbers and wrapping work as before. The completion popup moves with it and still opens at the cursor.
- The Guide says so under *The size of things*.
- Tests: the column capped and centred in a 200-wide terminal, with the frame exact and the text starting after the margin; off filling the pane; a narrow pane and zen left alone; the editor wrapping at the readable width, with the completion popup at the cursor after the margin; a split opened by skimming, capped and centred; the Settings toggle widening the note at once and in config; the config default. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: a long note in a 220-wide terminal reads as a centred 80-character column.

## v0.30.0 — 2026-09-19

**New: suggestions for tags and property values the vault already uses.** The user: "förslag när man använder taggar som man redan använt i andra anteckningar. Det minimerar property sprawl ('Filosofi, filosofi, Philosophy, philosophy' och felstavningar)". It was planned for after the locked period, and **the user chose to override the lock for it** ("Kör på"): the first exception to it, and worth having while the vault is still young.

- **`#tags` in the text.** Typing `#fil` suggests the tags the vault already uses, each *spelling* on its own row with how many notes use it: `#Filosofi · 2 notes · also written filosofi`. Choosing one keeps one name instead of making a third. The tag must start a word, so a `#` in a URL or in `[[Note#Heading]]` doesn't count. `# ` is a heading, and code and frontmatter are left alone.
- **Values in frontmatter.** Typing a property's value suggests what other notes gave that property: `type: vi` → `village`. List items under a key (`  - x`) count too. A `tags` property suggests every tag in the vault, including the ones written as `#tag` in note text. Links go in quotes, so the YAML stays valid. Dates, numbers, `title` and `created` aren't suggested, since they are never the same twice.
- **The same popup as `[[` completion.** Enter or Tab puts a suggestion in, and Esc closes it for that word. It opens only at the end of the word being typed, not as the cursor passes an existing one, and not when all it could offer is what's already typed.
- **The index keeps each tag's spelling.** Search and spreads still see one lower-case tag, as Obsidian does. The spelling is kept only to show variants. New: `Index.Tags` and `Index.PropertyValues`.
- The Keys tab's completion group now covers `[[`, `#tags` and properties, and the Guide explains it under Links.
- Tests: the index (spellings counted per note, most used first, search unchanged; values per key with list items and a note counted once); the editor (when a `#` is a tag being typed, in 10 cases; when a frontmatter value is, in 7; a completion as one undo step); the whole flow in Skrin (both spellings side by side with counts, picking; a value picked; a link quoted and dates skipped; a `tags` list fed by every tag; Esc closing for the word and a new tag reopening it). Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: `#fil` showed both spellings, and `type: vi` became `type: village` and was saved.

## v0.29.2 — 2026-09-19

**Fix: Skrin on macOS.** Found when a build for the user's MacBook Air M1 was checked against the code, before Skrin had ever run on a Mac. Three things assumed Linux. On Linux nothing changes.

- **The vault is found on a Mac.** Obsidian keeps its vault list in the system's own settings folder: `~/Library/Application Support/obsidian/` on macOS and `%APPDATA%\obsidian\` on Windows. Skrin looked only in the Linux place, so without `vault = …` in `config.toml` it found no vault.
- **Todos are no longer carried over twice on a Mac.** Skrin tells whether Obsidian is running, so that only one of them runs the Rollover Daily Todos carry-over. It asked `/proc`, which only Linux has, so on a Mac it always thought Obsidian was closed. Elsewhere it now asks `pgrep`, and when it can't tell it counts Obsidian as running, the rule Skrin already had: skipping a carry-over beats doing it twice.
- **Deleted notes go to the Mac's own trash**, `~/.Trash`, where Finder shows them, instead of a Linux trash folder that Finder never looks in. `U` brings them back as before. Finder's own "Put Back" doesn't know about them, since that needs Finder's private record. A vault on another disk uses its `.trash`, as on Linux.
- Tests: the vault-list path on Linux, with and without `XDG_CONFIG_HOME`, and on macOS and Windows; the running check when Obsidian is running, when it isn't and when it can't be told, plus the real `pgrep`'s exit code for a missing process; and the Mac trash (into `~/.Trash`, a name clash, `U` restoring it, and the freedesktop trash untouched), plus the Linux trash unchanged. Everything can be tested on Linux because each piece takes the system as a parameter. Full suite green (`-count=1`), `go vet` clean for Linux and for macOS on ARM, `gofmt` clean. Not verifiable here: a real run on a Mac, which the user does.

## v0.29.1 — 2026-09-19

**Fix: the arrows in Insert table now do what they show.** The user: "Pilarna som indikerar att man kan öka och minska antalet rader och kolumner pekar åt höger och vänster vid varje värde. Men det är med upp/ner-pilarna du faktiskt ändrar värdet." The number fields read `‹ 3 ›`, but ←/→ did nothing on them, and ↑/↓ changed the value. Fixed during the locked period as a bug: the form showed one thing and did another, and the arrows that did nothing gave no answer.

- **←/→ change the number:** ← one fewer, → one more. In the headings field they move the text cursor, as before.
- **↑/↓ move between the fields**, like Tab and Shift+Tab. The footer now reads "↑↓ field · ←→ or digits change".
- Tests: a new test that each arrow does what the form shows, including ← in the headings field, plus the existing form tests moved to the new keys. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux.

## v0.29.0 — 2026-09-18

**New: Insert template**, and three smaller quick reports. The user: "look at the quick notes and see what you can tackle next." Built directly; the reports that need a PO/UX or user decision were left for them (see `HANDOFF.md`).

- **Insert template** (quick report *Insert templates*: "define a template folder in Settings and then use ctrl+p in a note, even in edit mode, to apply templates"). Ctrl+P → "Insert template" lists the notes in the templates folder and puts the chosen one in at the cursor — from the editor, or from the reading view, which opens the note in the editor first. It works the way Obsidian's core Templates plugin does: `{{title}}` is the note's name, `{{date}}` and `{{time}}` use the plugin's formats from `.obsidian/templates.json` (read, never written), and `{{date:FORMAT}}`/`{{time:FORMAT}}` take any moment.js format. A template's properties are merged into the note's own frontmatter rather than inserted as a stray `---` block mid-note: the note gains the properties it lacks and keeps its own values for the rest, or takes the template's frontmatter if it has none. One undo step (`editor.Rewrite`), cursor at the end of what went in.
- **Templates folder in Settings**: a new kind of Settings row that holds a choice rather than on/off. Enter lists the vault's folders (Obsidian's own first, when its Templates plugin has one), saves to `config.toml` as `[templates] folder`, and comes back to the same row. Unset, or set to a folder that's since gone, it follows Obsidian's. Also in the palette as "Setting: Templates folder".
- **The instant-open setting is easier to recognise** (quick report *The setting for opening a new note on cursor or on l/enter should also be exposed in the settings menu*). It already was in the Settings tab, since v0.25.0, as "Open notes as the cursor moves". That named only one side of the choice, and the explanation only shows when the cursor is on the row. It now reads "Open notes on the cursor, not only on l/→ or Enter".
- **↓ on the editor's last row goes to the end of the line** (quick report *In editing mode, if standing on the bottom row …*), and ↑ on the first row to its start, as in most editors. The column you came from is kept, so the other arrow goes straight back. Vim's `j`/`k` are unchanged.
- **The status line counts a selection** (quick report *Counting selected rows*): "3 lines selected", in the reading view's VISUAL mode and in the editor, counted the way "Copied 3 lines" counts.
- Tests: `applyTemplate` (body at the cursor; frontmatter given to a note without; merged into one with, the note's values winning, cursor following), `ExpandTemplate`, reading `templates.json`, Insert template from the editor (listed, filled in, one ctrl+z) and from reading, the no-folder answer, choosing the folder in Settings (listed, saved, back on the row, shown; Esc back to Settings), a chosen folder winning over Obsidian's and falling back when gone; the arrows at both edges with vim's `j` untouched; the selection count in both views; the relabelled setting. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: the folder chosen from Settings, and the vault's own "Anteckning template" inserted from the reading view, its properties becoming the note's frontmatter with `{{date}}` filled in. The scratch note was restored.

## v0.28.0 — 2026-09-18

**New: tables that edit themselves, the way Obsidian's Advanced Tables plugin does.** The user, after v0.27.0's Insert table: "You know the plugin Advanced Tables from Obsidian. Something like that." This reopens the wishlist's "table editor" cut, at the user's word.

- **Tab, Shift+Tab and Enter in a table.** In the editor, Tab and Shift+Tab move from cell to cell, and Enter to the first cell of the row below: a row is filled with Tab, the next begun with Enter. Past the last cell or row a new row appears. Enter on an empty last row takes it away and leaves the table, onto a line of its own with a blank line kept between — a line straight under a table would otherwise be read as one more row of it. Every move lines the table up again, alignment kept, wide characters and escaped `\|` included. Outside a table Tab still indents and Enter still continues lists; Shift+Enter is always a plain new line.
- **Start a table by typing it**: "| Book | Year | Rating" and Tab adds the separator row, lines it up, and moves on.
- **Table commands in the palette**, shown only while the cursor is in a table (Ctrl+P, "table"): add a row below or above, delete or move a row, add a column right or left, delete or move a column, align a column left, centred or right, sort the rows by a column A → Z or Z → A (numbers as numbers, empty cells last), and line the table up. Each is one undo step. The heading row isn't moved, deleted or sorted, and says so if asked; neither is a table's last column deleted.
- In a table, the editor's status line reads "ctrl+p table commands · tab next cell · enter next row". A line starting with `|` inside a code block is code, not a table.
- The logic lives in `internal/editor/table.go` (a parsed table, rendered back with its cell positions for the cursor); the UI adds the commands and the hints. The Guide's "Tables" section describes all of it.
- Tests: in `internal/editor`, starting a table from a heading line, walking the cells both ways, Enter leaving from an empty last row (in the middle and at the end of a note), Tab and Enter untouched outside tables and in code blocks, formatting with alignment, wide characters and escapes, every operation against its exact output with a one-step undo, numeric sorting, and the refusals; in `internal/ui`, table commands only inside a table, one run from the palette with its flash, a refusal's reason, and the status line. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: a three-row table typed from a bare heading line with only Tab and Enter, sorted from the palette, left with Enter, and read back as a real table with the next paragraph apart from it. Two changes came out of that run: Enter first kept the column (so after tabbing across a row, the next row began under its last cell), and leaving the table put the cursor directly under it. Both are fixed and tested. The scratch note was restored afterwards.

## v0.27.0 — 2026-09-18

**New: Insert table.** From the quick report *Det behövs ett enkelt sätt att lägga till tabeller* ("there needs to be an easy way to add tables without spread — an interface"). Built directly at the user's go-ahead.

- **Ctrl+P → "Insert table"** opens a small form: columns and rows (↑/↓, +/−, or type the number — the first digit replaces what's there, like any number field), and optional headings, comma-separated. A live preview shows the markdown it will write. Headings beyond the column count widen the table instead of being dropped; a `|` in a heading is escaped.
- **Enter puts it at the cursor as a block of its own**: in place of a blank line, or after the line you're on, with one blank line kept above and below, since a markdown table needs that to be read as one. It's one undo step (`ctrl+z`), and the cursor lands in the first cell to fill: a heading when none were typed, otherwise the first cell under them. Columns are padded so the table reads as a table in the editor too.
- **From the reading view** it opens the note in the editor first, at the line you were reading; with no note, it says "Select a note to put a table in".
- Deliberately **insert only, not a table editor.** The wishlist lists a table editor under "considered and cut" ("big build, and tables are rarer than todos"), so this is the small version the report asked for: once in the note, a table is plain text. No new key: it lives in the palette (search "table"), where keyless commands belong, until the UX engineer says otherwise. Its form's keys are in the registry and the `?` manual as usual.
- New `editor.InsertBlock`, a small general tool for putting whole lines in as their own block, which a future "Insert template" can use too.
- The Guide has a short "Tables" section.
- Tests: `InsertBlock` in four placements, with a one-step undo and cursor placement; the table markdown (padding, empty headings, widening, escaping); the form from the editor palette (preview, insert, flash, typing landing in the right cell); from the reading view (editor opens, saved to disk); the no-note refusal; number fields typing and staying within 1–12 columns and 1–50 rows; Esc inserting nothing. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: the form from the reading view over the scratch vault, the default table inserted after a heading with blank lines around it, a heading typed, and the reading view drawing it as a real table. Then `u` restored the note.

## v0.26.0 — 2026-09-18

**New: onboarding — one door to every command, and everything points at it.** The user: "a better on-boarding for new users … it's hard to find exactly the right keybinding if you don't already know them, it's not very searchable, and there's a high threshold to start using Skrin", then: "act as a senior UX designer … it's up to you to design and build this experience." Design and reasoning: [[skrin onboarding]]. Closes two quick reports, *On ctrl+p* and *ctrl+c should never close*.

- **The command palette: `Ctrl+P`, or `:`.** Every command where you are, found by what it does in plain words — synonyms too, so "trash" finds delete, "journal" today's note, "toc" the outline. Each row shows its key on the right, from the keymap in force; Enter runs it. Commands with no key are in it as well: every Settings toggle, showing on/off, and the manual's Settings and Guide tabs. In the editor, `Ctrl+P` lists the editor's own commands — save, leave, to-do, undo typing, zen, backlinks, outline. Commands run lately come first.
- **It teaches the keys.** After a command that has a key, the status line says "Next time: z". A command's own answer — a refusal, a result — keeps the line instead.
- **`Ctrl+P` moved from Go to note to the palette** — Obsidian's key for its command palette. Go to note keeps `g`.
- **The status line's hints now follow where you are**: on a folder, on a note in Files, in the note, in a split, with marks, with a selection — built from the keymap in force, so a rebound key shows as rebound and the Claude keys go when the drawer is off (the Settings help already promised that). `ctrl+p commands` is always there; when room runs out, hints drop whole, least useful first.
- **A welcome card**: with no note open, the note pane shows the first eight keys and a tip of the day — a key that makes Skrin quick, a new one each day. It no longer says "move onto a note to open it" when instant-open is off.
- **A key that does nothing says so**: "x does nothing here · ctrl+p finds every command", instead of silence.
- **`Ctrl+C` no longer quits by surprise.** With a selection it copies, as before. With nothing selected the first press says "Nothing selected to copy · ctrl+c again quits, as does q"; a second within 1.5 s quits. `q` still quits at once.
- **The Guide opens with "Getting started"**: the palette, the hints, the welcome card, the manual's tabs.
- The palette is the existing chooser with a right-aligned key, words matched but not shown, and — for the palette only — ranking by whole words (in the name, then the synonyms) before letters-in-order through the name, so a short query doesn't return noise and a typo like "zne" still finds zen.
- Tests: 23 new in `onboard_test.go` and `copy_paste_test.go` — the palette from Files, `:`, and the editor; synonyms and typos; running, teaching, rebound keys, refusals winning, recents, settings saved to config.toml, the manual's tabs, quitting, Claude off; `TestPaletteCoversTheKeymap`, which fails when a new action is neither in the palette nor left out on purpose; hints per situation, rebound, narrow and Claude-off; the stray-key answer; the welcome card both ways and at three sizes; the tip holding for a day; the Ctrl+C double press, starting over after another key, and `q` still quitting at once. Three existing tests updated for the intended changes (`Ctrl+P` → `g` for Go to note, the splash text, and the Ctrl+C one rewritten as the double press). Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux on the scratch vault: welcome card, per-situation hints (folder, note, marks, split, a 70-column window), the palette by synonym and by `:`, the "Next time" lesson, the editor palette, a setting toggled and saved, the stray-key and first-Ctrl+C answers, and a double Ctrl+C quitting.

## v0.25.0 — 2026-09-18

Instant-open, made optional. The user's own request, via /remote-control: "Kan vi göra 'instant open' och 'open on enter/arrow' till ett val för användaren i settings? Obsidian öppnar inte anteckningar direkt vid cursor och många är kanske vana vid det" — Obsidian doesn't open notes right at the cursor, and plenty of people are used to that.

- **New: "Open notes as the cursor moves" in Settings**, on by default (today's behaviour, unchanged). Off is Obsidian's way: the note pane only changes on an explicit open — l/→ to read, Enter to edit — and otherwise keeps showing whatever was open last, including across folder toggles and cursor movement past notes. Turning it back on catches up to the cursor immediately.
- **This partially reverses a call already on record in the UX charter** ("Instant-open beat open-on-Enter in the field — the felt test outranks the argued one"). It doesn't overturn it — instant-open stays the default — but it concedes the argued case has enough of a following to earn a toggle rather than staying an either/or. Flagged for Hermes.
- **How it's built:** `peek()`, the single function that opens the note under the Files cursor, now gates on the new setting; everything downstream (explicit l/→, Enter, folder toggles) was already on its own path and needed no change.
- Tests: on/off behaviour under cursor movement, the previous note staying put with the setting off, catching up to the cursor when turned back on, and the Settings toggle itself. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: toggling in Settings takes effect at once; off stops the note pane following the cursor while l/→ and Enter still open explicitly and folders still toggle; back on, the note pane jumps to the cursor's current note right away.

## v0.24.0 — 2026-09-18

Two more quick reports, built directly, no formal tickets.

- **New: Enter on a note in Files opens it straight into editing.** Previously Enter and l/→ both opened a note for reading; now Enter goes straight to writing, and l/→ keeps opening for reading, unchanged. Enter on a folder still just toggles it, and on a non-note file still opens its own app — only the "Enter on a note" case changed. Standing on the note already open, on the note showing in an open split, or on a fresh note in Files all reach editing the same way; the split case swaps it into focus first, the same courtesy every other edit entry point already gives a reference note.
- **New: Shift+Enter inserts a new line in the editor, exactly like Enter.** It surprised the user that it didn't — most editors treat them the same. Now it's one alias on the same code path, so it undoes exactly the way Enter's own newline does.
- **The blast radius was bigger than either report suggested,** and worth recording as a lesson: Enter opening a note has been the assumption behind nearly every existing interaction with Files since v0.1, including the "instant-open" call already on record in the UX charter. Around 40 existing tests across most of `internal/ui`'s test files assumed Enter meant "open to read" — fixing them was almost entirely mechanical (swap `enter` for `l` where reading was the real intent), but one, `ops_test.go`'s undo-after-external-write test, turned out to have gone silently vacuous under the changed behaviour: it kept "passing" only because the assertion no longer exercised what it claimed to. Fixed at the root, not patched over.
- The Guide and the `?` manual's own key descriptions for Enter and l/→ are updated to match.
- Tests: five new ones for the exact new behaviour (a plain note, a folder, an already-open note, a note in the split, and l/→ staying read-only), two for Shift+Enter (that it inserts a line, and that it undoes the same way Enter's own newline does), plus the ~40 existing tests brought back in line with the new meaning. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: Enter on a note in Files opens the editor directly; l still opens it for reading; Shift+Enter inserts a line and undoes cleanly.
- **Not built, for later discussion (the user's own note):** whether Enter on an *already-open, focused* note (today: follow its one visible link) should also move to "start editing it" — a different code path (`noteAction`, not `filesAction`), and one that would need a new home for the one-link auto-follow it would replace. Recorded in `Quick reports/Enter on a viewed note could mean edit.md`.

## v0.23.0 — 2026-09-18

**New: Alt reaches a view-mode action without leaving what you're doing.** Three quick reports, all the same pattern: "Opening zen mode from edit mode", "Suggestion for key usage", "Follow links to split view". Built directly, no formal tickets.

- **In the editor, Alt-z, Alt-b and Alt-o reach zen, backlinks and the outline** without leaving the editor. Zen keeps editing centred, cursor untouched. The outline moves the *editor's own* cursor to the picked heading, since the reading view isn't even showing. Backlinks leaves the note — the editor didn't have a way to do that safely before, so picking one now saves first (declining stays put if a conflict comes up), the same save-then-go Esc already does.
- **Following a link with `f`: Alt+ on the hint's letter opens it in a split** beside the note, instead of in place — held on any letter of a two-letter label. The note you were reading stays open and focused, the same way skimming does; only the follow-in-place case swaps the main note away.
- **Go to note's split key is now Alt+←/→, not Shift+←/→**, so Alt means "into a split" everywhere a note can open — following a link, or going to one by name. (Shift+←/→ keeps its other, unrelated job: switching focus between the two panes of a split already open.)
- A #heading target on an Alt-opened link scrolls the split to it, the same way it scrolls the main pane — a new `splitJumpSrc`, the split's own version of the mechanism the main note already had.
- **How it's built:** the chooser overlay (backlinks, outline) can now open from inside the editor, so it takes keys ahead of it in the dispatch order — the two never used to coexist, so this changes nothing for any path that isn't the new one.
- Tests: the whole small set from the editor (zen round-trips without losing the cursor, outline moves the editor's cursor, backlinks saves an unsaved change before leaving), Alt-hint into a split (with and without room, and scrolling to a heading), and every existing split/switcher test updated to the new key with no other change needed. Full suite green (`-count=1`), `go vet` and `gofmt` clean. Verified live in tmux: zen and outline round-trip from the editor without losing your place, an unsaved edit is on disk before backlinks navigates away, and an Alt-followed hint opens beside the note you were reading, not instead of it.

## v0.22.0 — 2026-09-18

**New: spreads, phase 3 — live preview in the editor.** Spec [[skrin spreads]]; the user's own request from the original spec ("the dataview is a table while the cursor is not within the code block, changing to the raw query when moving into the code block"). Built at the user's go-ahead ("build it now. I have to see it in action to know if I like it").

- **In the built-in editor, a spread shows its answer** under a dim `spread · move here to edit` line — the user chose the hint over a bare table, so there's always a sign of the editable text behind it. Move onto any of its lines and it opens into the query: from above you land on the opening fence, from below on the closing one. Move off and it folds back, run again if the query changed.
- **Headings and text around it are unaffected** (the user's question): the fold is only the fences and what's between them, so a heading right above stays an ordinary line with the answer under it, and a new line typed under that heading just pushes the fold down.
- **How it's built:** the editor gains `Fold`s. The only code that changed is what *counts* display rows (`displayIndex`, `TopRow`) and what *draws* them (`View`); movement still steps over real lines, so stepping onto a folded line lands inside it and opens it — no movement code needed changing, and the cursor can never sit on a folded row. `internal/ui` recomputes the folds after every update from the text (a new `markdown.SpreadBlocks`, which skips a spread quoted inside a longer code fence, and unclosed ones), rendering each through the same renderer the reading view uses, cached by vault generation, width and block text. The block holding the cursor is never run, so typing a query never runs it half-typed.
- **Answers come from the notes as saved:** a spread over the note being edited sees it as of its last save. The Guide says so.
- The "Show spreads" setting covers the editor too; line numbers give a fold its fence's number and leave the rest of its rows blank.
- Tests: `internal/editor` (what a fold shows, stepping in from both sides, scroll positions counting folded rows, the gutter, stale folds dropped), `internal/markdown` (`SpreadBlocks`), `internal/ui` (a folded spread in the editor, opening it, editing the query and seeing it re-run, spreads off, line numbers, exact frames). Every existing editor test passes unchanged. Live in tmux with a heading directly above the fence: folded under the heading, open on ↓, folded again below, and a new line under the heading moving the fold down.

## v0.21.1 — 2026-09-18

Asked for by the user: "could we add natural language recognition to due dates? … In the action of leaving the edit mode 'tomorrow' in a due key becomes the upcoming date?"

- **New: a due date can be written as a word.** `due:: today`, `tomorrow`, or a weekday — in any of the three field forms (`due:: x`, `[due:: x]`, `(due:: x)`) — becomes the date it means when you leave the editor: `[due:: friday]` → `[due:: 2026-09-25]`. A weekday is always the coming one, so `friday` on a Friday is next week (the user's call). The status line names each one it set, after "Saved": `· due:: friday → 2026-09-25`.
- **Written once, not kept "live":** the note holds a real date from then on, which spreads, sorting and Obsidian all read the same way.
- **Only what you typed this time:** lines already in the note when the editor opened are left alone, so an old `due:: tomorrow` from Obsidian keeps meaning its author's tomorrow. A `Ctrl-s` along the way doesn't make a line "old", and `Ctrl-s` itself never rewrites — that would change the text under the cursor mid-thought.
- **Kept small on purpose** (the user asked whether it was over-engineered): English words only, `due` only, the built-in editor only. No `in 3 days`, no other date keys, no Quick Notes — each easy to add if missed. Left alone: `due:: tomorrow morning`, `overdue::`, one-colon `due:`, code spans and fenced code.
- `u` takes the dates back together with the rest of the edit, like any save.
- New `internal/duedate`; hooked into `saveEdit` on leaving; one Guide paragraph. Tests: the word → date rules (including Friday-on-a-Friday), every form and every look-alike that must stay, only-changed-lines, and in `internal/ui` the flash, `Ctrl-s` then leaving, several dates at once, and undo. Verified live in tmux on a real Friday.

## v0.21.0 — 2026-09-18

**New: spreads, phase 2 — tasks and inline fields.** Ticket `Backlog/Refined/Spreads phase 2.md`, spec [[skrin spreads]] ("Phase 2 as built"). Built at the user's go-ahead ("Include bracketed fields, and build it now") while the PO/UX reviewer was away.

- **`TASK` spreads** list the checkboxes in the notes `FROM` picks, grouped under a link to each note, subtasks nested under their parent. `WHERE` and `SORT` see each task first — `text`, `status`, `completed`, `checked`, `line` and the fields on its own line — then its note's. A matching task brings its subtasks along; a matching subtask whose parent doesn't match shows on its own.
- **Each task ends in a ↗ link that opens its note at that task** — a new `#:n` line anchor in `index.Anchor`, which only ever appears in rendered answers. Spreads stay read-only: `e` and `Ctrl-l` there ticks it.
- **Checkboxes under `### Habits` aren't tasks** and don't show, Skrin's own rule (the rollover already skips them). A deliberate difference from Obsidian with Dataview.
- **Inline fields**, Dataview's convention: `key:: value` on a line of its own, and `[key:: value]` or `(key:: value)` inside a sentence. Keys lower-case with spaces as dashes (`**Due Date**::` → `due-date`), values may hold a `[[link]]`, code is ignored. They join frontmatter of the same name in every spread, but stay out of `/` search, as they do in Obsidian's own search.
- **Found and fixed during the live check:** a field on a task's line first counted for its whole note too, so an undated task borrowed another task's date and `SORT due` put it in the wrong place. A task's fields are now its own. Caught only because the live test note happened to put an undated task above a dated one; there's a regression test for exactly that now, confirmed to fail without the fix.
- `index` parses fields and tasks once per note, incrementally like everything else (`Fields`, `Tasks`, new `fields.go`); `spread` gains `tasks.go`; `ui` hands both through, dropping habits. The Guide tab explains `TASK` and inline fields.
- Tests: index (fields in every form and the ones that mustn't count, tasks with nesting through a plain bullet and a list ended by prose, the line anchor), spread (task selection, nesting, grouping, sorting, links, inline fields joining frontmatter), ui (a task spread on screen, ↗ opening an 80-line note at its task, habits left out, the borrowed-date regression). Full suite green (`-count=1`), `go vet` and `gofmt` clean. Live in tmux on the scratch vault; test notes removed after.

## v0.20.1 — 2026-09-18

User feedback on spreads: "The columns are separated by a divider but the item within each column isn't … imagine a list of 200 items and 16 columns. Following 1 row could be very difficult."

- **Table rows are striped now:** every other body row sits on the theme's lighter background, edge to edge between the table's outer bars, so a row can be followed across a wide table by eye. A row that wraps onto several lines keeps one band across all of them, which also shows where it ends. The header and the first row stay plain, so the first stripe reads as clearly apart from the bold header.
- It's the one table renderer, so this applies to spreads and to every markdown table in a note alike — the same problem, the same fix.
- Chosen over a divider line between rows, which would double every table's height and still leave 16 columns of plain text to track.
- Test: the right rows carry the stripe (and the wrong ones don't — so the test can't pass on a palette with no colour), a wrapped striped row keeps it on every line, and every row stays the same width. Verified live in tmux: the stripe turns on at the first bar and off right after the last, with no bleed into the pane.

## v0.20.0 — 2026-09-18

**New: spreads, phase 1.** From the ticket `Backlog/Refined/Spreads.md` (spec: [[skrin spreads]], formerly "Dataview query blocks"). Built at the user's go-ahead while the PO/UX reviewer was away; that review happens on this build.

- **A ```spread block shows what its query finds** — a table or a list of notes — in the reading view, zen and the split, instead of its text. ```dataview blocks run the same way, so a vault using Obsidian's Dataview plugin shows the same tables in both apps. ```query (Obsidian's embedded search) and ```dataviewjs stay code, and so does a block that's never closed.
- **The query language is a subset of Dataview's DQL:** `TABLE` (with `WITHOUT ID` and `AS "name"`) and `LIST [field]`; `FROM` tags (nested ones included), folders, `[[note]]` (notes linking to it) and `outgoing([[note]])`, combined with `AND`, `OR`, `-`/`!` and parentheses; `WHERE` with `= != < <= > >=`, `AND`/`&`, `OR`/`|`, `!`/`NOT`, `null` and `contains()`/`icontains()`; `SORT` on several fields `ASC`/`DESC`; `LIMIT`. Keywords in any case, clauses in any order.
- **Fields:** frontmatter properties, plus `file.name`, `file.link`, `file.folder`, `file.path`, `file.tags`, `file.mtime`, `file.size`. Properties are text in the index, so each one is read as a number, an ISO date, a single wikilink or text; different kinds are never equal and never in order, and nothing about that is an error.
- **Zero new keys.** Every note in an answer is a real link, so `f` puts a hint on it and `Enter` follows it.
- **It stays current:** any change in the vault re-runs the open note's spreads — verified by editing a rating on disk while Skrin was open and watching the table re-sort itself.
- **Nothing silent:** `No notes match` for an empty answer; a 200-row cap with `+N more · add LIMIT or narrow FROM` when there's no `LIMIT`; mistakes name the problem and the line inside the block (`⚠ Spread: expected FROM, WHERE, SORT or LIMIT, found "WHER" (line 2)`); Dataview syntax that isn't here yet is named rather than called an error (`GROUP BY`, `FLATTEN`, `TASK`, functions other than `contains`, arithmetic, `this`). `file.ctime` says why it isn't there: Linux can't tell when a note was created, and a field quietly answering with the modification time would lie.
- **An off switch:** `[render] spreads = true` in config.toml, "Show spreads" on the Settings tab; off shows the block as code. The Guide tab explains the query language, and its config listing now includes the `[render]` section.
- **How it's built:** a new `internal/spread` (lexer, parser, evaluator) that sees the vault through a small interface, and returns its answer as markdown — a pipe table or a list of wikilinks — which `internal/markdown` renders like any other note text, rows mapped to the fence's line the way transclusion does it. So wrapping table cells (v0.19.1), link hints, line numbers and zen all work on spreads for free. `internal/index` gains `Stat` (a note's mtime and size) and `Outgoing` (the notes it links to).
- **Not in phase 1**, per the spec: `TASK` and inline `key:: value` fields (phase 2), and live preview inside the editor (phase 3) — the editor still shows the block as text.
- Tests: `internal/spread` (the query language end to end over a fake vault, including every error message), `internal/markdown/spread_test.go` (the block is replaced, rows map to the fence, links are followable, which fences stay code, errors and notes, spreads inside an embedded note), `internal/ui/spreads_test.go` (the answer on screen, following a result with `f`, refresh on a vault change, the off switch and the setting), plus `internal/index` and `internal/config`. Verified live in tmux on the scratch vault: tables, the ```dataview list, the typo and empty messages, following a comma-named note, the on-disk edit re-sorting the table, 60 columns with line numbers, and zen.

## v0.19.1 — 2026-09-18

Three quick reports, fixed directly (no formal tickets — see `Quick reports/`):

- **New: Ctrl+C copies a selection to the system clipboard.** Ctrl+C already meant quit/close/cancel everywhere in the keymap, so this follows the same pattern Esc already uses ("clears a selection first, otherwise the bigger action"): with an active selection — the editor's Shift+arrow/vim `v` selection, or the reading view's `v` line selection — Ctrl+C copies it and flashes "Copied N lines" instead of quitting or closing; with no selection, Ctrl+C is unchanged. The copy goes out over OSC 52 (`tea.SetClipboard`), which the terminal itself relays to the system clipboard — no OS-specific clipboard tool, no cgo, and it works the same locally, over SSH, or in tmux with clipboard passthrough on. Not every terminal supports OSC 52 for writing (and even fewer allow reading it back, which is why this doesn't touch `Ctrl+V` — see below); there's no reliable way to detect that in advance, so the flash names what was attempted rather than promising it landed.
- **Fixed: pasting into a Quick Note did nothing.** `Ctrl+V`/bracketed paste already worked in the editor, search, and several other places, but the quick-note overlay (`i`) was missing from the dispatch — pasting into either its text box or its folder field was silently swallowed. Both now accept a paste. (A similar gap exists in the Book Card's fields; not fixed here, out of scope for this pass.)
- **Investigated, not reproduced: arrow keys disabled in a zen-mode Quick Note.** Tried the exact reported scenario — opening `i` over a note in zen mode, typing enough lines to force the capture box's own internal scroll, navigating with ↑/↓ — live in tmux, twice, and both worked correctly. This looks like it was already fixed by v0.16.3's Quick Note scroll fix, which shipped about two and a half hours before this report was filed; flagging rather than closing outright in case it's still reproducible on a real terminal.
- **Fixed: table cells no longer clip long text — they wrap inside the cell.** `internal/markdown`'s table renderer used to shrink columns to fit the pane and then hard-clip whatever still didn't fit, with a trailing "…"; it now word-wraps each cell to its column's width instead, and a table row grows to fit its tallest cell. Every sub-row past a cell's first still carries the row's one source line, the same "blank gutter on a wrapped continuation" rule a normal paragraph or list item gets (and that the new line-numbers gutter already relies on). Verified live in zen mode, the normal view, and at a narrow terminal width; box borders stay aligned throughout.

## v0.19.0 — 2026-09-18

**New: line numbers in notes.** From the ticket [[Line numbers in notes]] (spec: [[skrin line numbers]]), signed off the same day.

- `L` in the main context toggles a dim line-number gutter on and off, live, across the reading pane, zen mode, the split view, and the built-in editor — flashing "Line numbers on"/"Line numbers off". Also toggleable from the Settings tab of `?` ("Line numbers in notes"), persisted to `config.toml` as `[render] line_numbers` (default `false`, off).
- **Reading view, zen and split** each draw the gutter as a prefix on every display line: the real source line number for the first display line of a wrapped paragraph, a blank gutter for its continuation lines, keeping the existing `1:n` Src mapping honest. Digit width grows with the note's line count (minimum 2 columns).
- **The editor** draws its own gutter with the same digit-width rule, and bolds/accents the current line's number so the cursor's line is easy to find at a glance.
- Link hints (`f`) and the `[[` completion popup both account for the gutter's width when it's on, so hint labels and the popup still land in the right column.
- Tests: toggling in the reading view, split view and zen mode, the editor gaining line numbers when opened with the setting on, and the Settings-tab checkbox round-tripping to config. Full suite green (`-count=1`), `go vet` clean, `gofmt` clean on the touched files, and verified live in tmux against the scratch vault — including a real split opened by skimming (the original test only pressed an unbound `s` key and never actually exercised split rendering; fixed to skim a real split open before asserting on it).
- **What isn't fully verified:** the editor's active-line-number color (Accent, in addition to bold) rendered as bold-only with no color in this tmux sandbox's 256-color downgrade path — traced to the terminal's color-profile downgrade, not the code: a direct call to `editor.View()` and a standalone truecolor run both produced the correct bold *and* accent-colored escape sequence (`\x1b[1;38;2;125;174;163m`). This matches the project's existing pattern of sandbox-only color/pixel gaps (e.g. the v0.16.0 sixel draw) — needs a look in `foot` to confirm it renders with color there too.

## v0.18.2 — 2026-09-17

Fix: Skrin was still reopening the last open note on startup even when "Remember last open note" was disabled.

- **Fixed: startup no longer peeks the note under the restored Files cursor.** In v0.17.0, `restore_last_note = false` correctly avoided restoring the active note path in session state, but `settle()` would peek the note under the cursor upon subsequent background lifecycle events (such as theme discovery or opening modals). Peeking is now strictly driven by user cursor movement (`lastPeekCur`/`lastPeekRel`), keeping the welcome splash screen intact on launch until the user explicitly moves the cursor or presses `Enter`/`l`.
- Tests: multi-event lifecycle tests verifying the welcome screen remains visible across background `ThemeMsg`, window resizes, and modal toggles when `restore_last_note` is false, while navigation and explicit `Enter`/`l` actions continue to open notes seamlessly.

## v0.18.1 — 2026-09-17

Asked for as "a zoom indicator for when increasing and decreasing text size with Ctrl+- and Ctrl++".

- **New: the terminal's size shows in the status line for two seconds whenever it changes**, so a zoom visibly lands instead of the screen silently reflowing. Zoom in and the numbers fall, zoom out and they rise.
- **Worth knowing about the keys**: `Ctrl+-` and `Ctrl++` belong to the terminal, not to Skrin — the terminal changes its own font and never passes those keys on, and no program can resize its own text. What Skrin is told is the new size in columns and rows, so that is what it reports. No key was added, and nothing new is in the keymap.
- **It names the one consequence that looks like a fault**: under 80 columns Files only shows while it has focus, so the indicator says "Files hides unless focused" rather than leaving the tree's disappearance unexplained. A split closing still says which note it closed, and that message wins.
- **It takes itself away.** Every other flash waits for the next keypress, because you pressed something and are owed an answer; this one answers a resize nobody pressed a key for. In zen mode that distinction matters — zen shows a status line *only* while there's a flash, so an indicator that waited would have left zen with a permanent bar.
- Tests: the new size showing and reaching the screen; the first size and a repeated size staying quiet; the indicator clearing itself; an old timer refusing to carry off a newer message; the under-80 wording appearing and then not; and zen mode getting its clean screen back. Full suite green, and the whole thing driven live against a real terminal resize, in zen mode too.
- Also in this release: `1` and `2` now say what they do in the manual ("Focus on the Files pane", "Focus on the note pane") instead of just naming the panes, so they read like every other row.

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
