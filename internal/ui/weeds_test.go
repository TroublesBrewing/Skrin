package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/config"
	"github.com/lurioso/skrin/internal/weeds"
)

// Weeds is a beta feature, so it takes both switches; every test here
// asks for them, and one test checks that W says so when they are off.
func newWeedsModel(t *testing.T) *Model {
	t.Helper()
	return newTestModelWith(t, Options{RolloverTodos: true, Beta: true, Weeds: true})
}

func weedsText(m *Model) string {
	var b strings.Builder
	for _, l := range m.weedsBox() {
		b.WriteString(ansi.Strip(l))
		b.WriteString("\n")
	}
	return b.String()
}

func TestWeedsIsOffUntilItIsSwitchedOnTwice(t *testing.T) {
	m := newTestModel(t) // no beta, no weeds
	press(m, "W")
	if m.weeds != nil {
		t.Fatal("W should not open the panel while the feature is off")
	}
	if !strings.Contains(m.flash, "beta") || !strings.Contains(m.flash, "Settings") {
		t.Errorf("flash = %q: an off feature must say where to switch it on", m.flash)
	}
	// Beta mode alone isn't enough: the feature has its own row.
	m = newTestModelWith(t, Options{Beta: true})
	press(m, "W")
	if m.weeds != nil {
		t.Error("beta mode alone should not open it")
	}
}

func TestWeedsFindsTheFixtureVaultsDeadLink(t *testing.T) {
	m := newWeedsModel(t)
	press(m, "W")
	if m.weeds == nil {
		t.Fatal("W should open the panel")
	}
	// Welcome.md links to [[Missing]], which isn't there.
	out := weedsText(m)
	if !strings.Contains(out, "[[Missing]]") {
		t.Errorf("the dead link should be listed:\n%s", out)
	}
	if !strings.Contains(out, "Links that lead nowhere") {
		t.Errorf("the group should be titled:\n%s", out)
	}
	if !strings.Contains(out, "Weeds") {
		t.Errorf("the panel should say what it is:\n%s", out)
	}
}

func TestWeedsCursorStartsOnAnItemAndSkipsTheHeadings(t *testing.T) {
	m := newWeedsModel(t)
	press(m, "W")
	if m.weeds.selected() == nil {
		t.Fatal("the cursor should start on an item, never a heading")
	}
	// Walking the whole list must never land on a heading row.
	for i := 0; i < len(m.weeds.rows)+2; i++ {
		press(m, "j")
		if m.weeds.selected() == nil {
			t.Fatalf("the cursor landed on a heading after %d moves down", i+1)
		}
	}
	for i := 0; i < len(m.weeds.rows)+2; i++ {
		press(m, "k")
		if m.weeds.selected() == nil {
			t.Fatalf("the cursor landed on a heading after %d moves up", i+1)
		}
	}
}

func TestEnterOnADeadLinkOpensTheNoteAtItsLine(t *testing.T) {
	m := newWeedsModel(t)
	// A long note with the dead link far down it, so that arriving at the
	// link means the note is scrolled and not merely opened.
	writeFile(t, m.vault.Root, "Long.md", "# Long\n"+strings.Repeat("filler line\n", 90)+"See [[Nowhere]]\n")
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	press(m, "W")
	if !seekWeed(m, func(it *weeds.Item) bool {
		return it.Kind == weeds.DeadLink && it.What == "Nowhere"
	}) {
		t.Fatal("setup: the dead link should be in the list")
	}
	press(m, "enter")
	if m.weeds != nil {
		t.Error("going somewhere should close the panel")
	}
	if m.notePath != "Long.md" {
		t.Fatalf("open note = %q, want the note holding the link", m.notePath)
	}
	// jumpSrc is spent by the render that follows, so what is left to see
	// is where the note ended up scrolled to: the link's line in view.
	if m.noteOff == 0 {
		t.Error("the note should be scrolled to the link, not opened at the top")
	}
	if !strings.Contains(m.flash, "leads nowhere") {
		t.Errorf("flash = %q: it should say why you were sent there", m.flash)
	}
}

func TestNOnADeadLinkMakesTheNoteAndUTakesItBack(t *testing.T) {
	m := newWeedsModel(t)
	press(m, "W")
	if !seekWeed(m, func(it *weeds.Item) bool {
		return it.Kind == weeds.DeadLink && it.What == "Missing"
	}) {
		t.Fatal("setup: the fixture's dead link should be in the list")
	}
	press(m, "n")
	if m.weeds != nil {
		t.Error("making the note should close the panel: the note opens for writing")
	}
	if !m.vault.Exists("Missing.md") {
		t.Fatal("n should make the note the link wanted")
	}
	press(m, "esc") // leave the editor the new note opened in
	press(m, "U")
	if m.vault.Exists("Missing.md") {
		t.Error("U should take the note back out again")
	}
}

func TestNOnARowThatIsNotALinkSaysSoInsteadOfDoingSomethingElse(t *testing.T) {
	m := newWeedsModel(t)
	press(m, "W")
	if !seekWeed(m, func(it *weeds.Item) bool { return it.Kind != weeds.DeadLink }) {
		t.Skip("the fixture vault has only dead links to show")
	}
	press(m, "n")
	if m.weeds == nil {
		t.Fatal("a refusal should leave the panel open")
	}
	if !strings.Contains(m.flash, "isn't a link") {
		t.Errorf("flash = %q: a key that can't act here must say why", m.flash)
	}
}

func TestATidyVaultSaysThereIsNothingToTend(t *testing.T) {
	m := newTestModelWith(t, Options{Beta: true, Weeds: true})
	// Empty the vault of everything the checks could complain about.
	for _, f := range []string{"Welcome.md", "Filosofi/Stoic.md", "Filosofi/Antik/Zeno.md",
		"Daily/2026-09-11.md", "Daily/2026-09-13.md", "Templates/Daily template.md"} {
		if err := os.Remove(filepath.Join(m.vault.Root, filepath.FromSlash(f))); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	press(m, "W")
	if m.weeds == nil {
		t.Fatal("the panel should open even with nothing in it — that is the good news")
	}
	if out := weedsText(m); !strings.Contains(out, "Nothing to tend") {
		t.Errorf("an empty answer should say so:\n%s", out)
	}
}

func TestEscClosesWeeds(t *testing.T) {
	m := newWeedsModel(t)
	press(m, "W", "esc")
	if m.weeds != nil {
		t.Error("esc should close the panel")
	}
	if m.flash != "Weeds closed" {
		t.Errorf("flash = %q", m.flash)
	}
}

func TestWeedsFillsTheFrameExactly(t *testing.T) {
	m := newWeedsModel(t)
	press(m, "W")
	checkFrame(t, m, "the Weeds panel")
}

func TestWeedsIsOutOfTheWayEntirelyWhenOff(t *testing.T) {
	named := func(m *Model) bool {
		for _, c := range m.mainPaletteItems() {
			if strings.Contains(c.label, "Weeds") {
				return true
			}
		}
		return false
	}
	if named(newTestModel(t)) {
		t.Error("an off experiment should not offer a palette row")
	}
	if !named(newWeedsModel(t)) {
		t.Error("switched on, it should be findable by name in the palette")
	}
}

// seekWeed puts the cursor on the first row matching want, and reports
// whether it found one.
func seekWeed(m *Model, want func(*weeds.Item) bool) bool {
	for i, r := range m.weeds.rows {
		if r.item != nil && want(r.item) {
			m.weeds.cur = i
			return true
		}
	}
	return false
}

func TestTemplatesAndDailyNotesAreNotJudgedAsNotes(t *testing.T) {
	m := newTestModelWith(t, Options{Beta: true, Weeds: true,
		Vault: config.VaultSettings{TemplatesFolder: "Templates"}})
	press(m, "W")
	out := weedsText(m)
	// The fixture's template is unlinked, thin and would count as both;
	// a template is all three on purpose.
	if strings.Contains(out, "Daily template") {
		t.Errorf("a template is not a loose end:\n%s", out)
	}
	// The daily notes are standalone by design too.
	for _, day := range []string{"2026-09-11", "2026-09-13"} {
		if strings.Contains(out, day) {
			t.Errorf("a daily note is not a loose end:\n%s", out)
		}
	}
}

func TestADeadEmbedReadsAsAnEmbed(t *testing.T) {
	m := newWeedsModel(t)
	writeFile(t, m.vault.Root, "Shot.md", "# Shot\n\n![[gone.png]]\n")
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	press(m, "W")
	if out := weedsText(m); !strings.Contains(out, "![[gone.png]]") {
		t.Errorf("a dead embed should be written as one:\n%s", out)
	}
}
