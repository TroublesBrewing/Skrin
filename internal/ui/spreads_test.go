package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// withSpread puts a spread into Welcome.md, then opens it. The fixture's
// Daily/ holds 2026-09-11.md and 2026-09-13.md.
func withSpread(t *testing.T, opts Options, query string) *Model {
	t.Helper()
	m := newTestModelWith(t, opts)
	if err := os.WriteFile(m.vault.Abs("Welcome.md"), []byte("# Welcome\n```spread\n"+query+"\n```\nafter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	onWelcome(m)
	return m
}

func screen(m *Model) string { return ansi.Strip(m.render()) }

func TestSpreadShowsWhatItFinds(t *testing.T) {
	m := withSpread(t, Options{Spreads: true}, `LIST FROM "Daily"`)
	s := screen(m)
	for _, want := range []string{"2026-09-11", "2026-09-13", "after"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "```") || strings.Contains(s, `FROM "Daily"`) {
		t.Errorf("the query is still showing:\n%s", s)
	}
	checkFrame(t, m, "a spread in the note")
}

func TestSpreadResultsAreLinksToFollow(t *testing.T) {
	m := withSpread(t, Options{Spreads: true}, `LIST FROM "Daily"`)
	press(m, "f")
	if m.hints == nil || len(m.hints.hints) != 2 {
		t.Fatalf("hints = %+v", m.hints)
	}
	press(m, "a")
	if m.notePath != "Daily/2026-09-11.md" {
		t.Errorf("followed to %q", m.notePath)
	}
}

func TestSpreadStaysCurrentWhenTheVaultChanges(t *testing.T) {
	m := withSpread(t, Options{Spreads: true}, `LIST FROM "Daily"`)
	if err := os.WriteFile(m.vault.Abs("Daily/2026-09-14.md"), []byte("new day"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(VaultChangedMsg{})
	if s := screen(m); !strings.Contains(s, "2026-09-14") {
		t.Errorf("a new note didn't show up in the spread:\n%s", s)
	}
}

func TestSpreadTableAndErrors(t *testing.T) {
	m := withSpread(t, Options{Spreads: true}, "TABLE file.folder\nFROM \"Daily\"\nSORT file.name DESC")
	s := screen(m)
	if i, j := strings.Index(s, "2026-09-13"), strings.Index(s, "2026-09-11"); i < 0 || j < 0 || i > j {
		t.Errorf("want 2026-09-13 before 2026-09-11:\n%s", s)
	}
	if !strings.Contains(s, "file.folder") || !strings.Contains(s, "Daily") {
		t.Errorf("the table's header and folder column are missing:\n%s", s)
	}
	m = withSpread(t, Options{Spreads: true}, "LIST FROM \"Daily\"\nGROUP BY file.folder")
	if s := screen(m); !strings.Contains(s, "⚠ Spread: GROUP BY isn't supported yet (line 2)") {
		t.Errorf("no message for unsupported syntax:\n%s", s)
	}
}

func TestSpreadsOffShowTheQuery(t *testing.T) {
	m := withSpread(t, Options{}, `LIST FROM "Daily"`)
	if s := screen(m); !strings.Contains(s, `LIST FROM "Daily"`) || strings.Contains(s, "2026-09-11") {
		t.Errorf("with spreads off, the query should show as code:\n%s", s)
	}
}

func TestShowSpreadsSetting(t *testing.T) {
	m := withSpread(t, Options{Spreads: true}, `LIST FROM "Daily"`)
	for _, it := range settingsItems() {
		if it.label == "Show spreads" {
			it.set(m, false)
		}
	}
	m.settle()
	if s := screen(m); !strings.Contains(s, `LIST FROM "Daily"`) {
		t.Errorf("turning spreads off should show the query again:\n%s", s)
	}
	if m.opts.Config.RenderSpreads() {
		t.Error("the setting should be kept in config")
	}
}

// Daily/2026-09-13.md in the fixture: "- [ ] call mum", "- [x] done thing",
// "- [ ] " (empty), "- [-] cancelled", "* [ ] star task".
func TestTaskSpreadListsTasks(t *testing.T) {
	m := withSpread(t, Options{Spreads: true}, `TASK FROM "Daily" WHERE !completed`)
	s := screen(m)
	for _, want := range []string{"2026-09-13", "call mum", "cancelled", "star task"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "done thing") {
		t.Errorf("a completed task is showing:\n%s", s)
	}
	checkFrame(t, m, "a task spread")
}

func TestTaskLinkOpensTheNoteAtTheTask(t *testing.T) {
	m := newTestModelWith(t, Options{Spreads: true})
	body := "# Long\n" + strings.Repeat("filler\n", 80) + "- [ ] the one far down\n"
	for rel, text := range map[string]string{
		"Long.md":    body,
		"Welcome.md": "# Welcome\n```spread\nTASK FROM \"Long\"\n```\n",
	} {
		if err := os.WriteFile(m.vault.Abs(rel), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m.Update(VaultChangedMsg{})
	onWelcome(m)
	press(m, "f")
	if m.hints == nil || len(m.hints.hints) != 2 { // the note, and the task's ↗
		t.Fatalf("hints = %+v", m.hints)
	}
	press(m, "s") // the second hint: the task's ↗
	if m.notePath != "Long.md" {
		t.Fatalf("opened %q", m.notePath)
	}
	if m.flash != "" {
		t.Errorf("flash = %q", m.flash)
	}
	if !strings.Contains(screen(m), "the one far down") || m.noteOff == 0 {
		t.Errorf("the note didn't open at the task (offset %d):\n%s", m.noteOff, screen(m))
	}
}

func TestTaskSpreadLeavesHabitsOut(t *testing.T) {
	m := newTestModelWith(t, Options{Spreads: true})
	for rel, text := range map[string]string{
		"Daily/2026-09-14.md": "### Habits\n- [ ] Stretch\n- [ ] Read\n\n### Todo's\n- [ ] buy milk\n",
		"Welcome.md":          "# Welcome\n```spread\nTASK FROM \"Daily/2026-09-14\"\n```\n",
	} {
		if err := os.WriteFile(m.vault.Abs(rel), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m.Update(VaultChangedMsg{})
	onWelcome(m)
	s := screen(m)
	if !strings.Contains(s, "buy milk") || strings.Contains(s, "Stretch") || strings.Contains(s, "Read") {
		t.Errorf("habits should stay out of a task spread:\n%s", s)
	}
}

// An undated task above a dated one: SORT due must put the dated one
// first, not let the undated one borrow its neighbour's date.
func TestTaskWithoutAFieldDoesntBorrowItsNeighbours(t *testing.T) {
	m := newTestModelWith(t, Options{Spreads: true})
	for rel, text := range map[string]string{
		"Venue.md":   "# Venue\n- [-] rent the barn\n- [ ] book the room [due:: 2026-09-20]\n",
		"Welcome.md": "# Welcome\n```spread\nTASK FROM \"Venue\" SORT due\n```\n",
	} {
		if err := os.WriteFile(m.vault.Abs(rel), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m.Update(VaultChangedMsg{})
	onWelcome(m)
	s := screen(m)
	if i, j := strings.Index(s, "book the room"), strings.Index(s, "rent the barn"); i < 0 || j < 0 || i > j {
		t.Errorf("the dated task should sort first:\n%s", s)
	}
}

// editSpread opens Welcome.md, holding a spread, in the built-in editor,
// with the cursor on its first line — above the spread.
func editSpread(t *testing.T, opts Options) *Model {
	t.Helper()
	m := withSpread(t, opts, `LIST FROM "Daily"`)
	press(m, "e")
	if m.editor == nil || m.editor.Line() != 0 {
		t.Fatalf("editor not open at the top")
	}
	return m
}

func TestTheEditorShowsASpreadFolded(t *testing.T) {
	m := editSpread(t, Options{Spreads: true})
	s := screen(m)
	for _, want := range []string{foldHint, "2026-09-11", "2026-09-13", "after"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "```spread") || strings.Contains(s, `LIST FROM "Daily"`) {
		t.Errorf("the query should be folded away:\n%s", s)
	}
	checkFrame(t, m, "the editor with a folded spread")
}

func TestMovingIntoASpreadOpensItAndOutFoldsIt(t *testing.T) {
	m := editSpread(t, Options{Spreads: true})
	press(m, "down")
	if s := screen(m); !strings.Contains(s, "```spread") || !strings.Contains(s, `LIST FROM "Daily"`) || strings.Contains(s, foldHint) {
		t.Fatalf("on the fence, the query should show:\n%s", s)
	}
	checkFrame(t, m, "an opened spread")
	press(m, "down", "end")
	typeText(m, " LIMIT 1")
	press(m, "down", "down") // onto "after"
	s := screen(m)
	if !strings.Contains(s, foldHint) || !strings.Contains(s, "2026-09-11") || strings.Contains(s, "2026-09-13") {
		t.Errorf("out again, the edited query should run:\n%s", s)
	}
	press(m, "up") // up into it lands on the closing fence
	if !strings.Contains(screen(m), "LIMIT 1") {
		t.Errorf("up from below should open it:\n%s", screen(m))
	}
}

func TestSpreadsOffShowTheQueryInTheEditor(t *testing.T) {
	m := editSpread(t, Options{})
	if s := screen(m); strings.Contains(s, foldHint) || !strings.Contains(s, `LIST FROM "Daily"`) {
		t.Errorf("with spreads off, the editor shows plain text:\n%s", s)
	}
}

func TestAFoldedSpreadNumbersOnce(t *testing.T) {
	m := editSpread(t, Options{Spreads: true, LineNumbers: true})
	s := screen(m)
	if !strings.Contains(s, " 2 │ "+foldHint) || !strings.Contains(s, " 5 │ after") {
		t.Errorf("the fold should sit on line 2, and after on 5:\n%s", s)
	}
	checkFrame(t, m, "a folded spread with line numbers")
}
