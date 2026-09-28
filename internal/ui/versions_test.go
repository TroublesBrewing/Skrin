package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// The time machine is a beta feature, so these ask for both switches.
func newVersionsModel(t *testing.T) *Model {
	t.Helper()
	return newTestModelWith(t, Options{RolloverTodos: true, Beta: true, Versions: true})
}

func versionsText(m *Model) string {
	var b strings.Builder
	for _, l := range m.versionsBox() {
		b.WriteString(ansi.Strip(l))
		b.WriteString("\n")
	}
	return b.String()
}

// history writes a note three times through Skrin's own editor, which is
// what leaves the snapshots the time machine reads.
func versionsFixture(t *testing.T) *Model {
	t.Helper()
	m := newVersionsModel(t)
	writeFile(t, m.vault.Root, "Note.md", "# Note\n\nfirst\n")
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	for _, word := range []string{"second", "third"} {
		m.open("Note.md")
		press(m, "enter") // into the editor
		if m.editor == nil {
			t.Fatal("setup: the editor should open")
		}
		m.editor.Reset("# Note\n\n" + word + "\n")
		press(m, "ctrl+s", "esc")
	}
	return m
}

func TestTheTimeMachineIsOffUntilSwitchedOnTwice(t *testing.T) {
	m := newTestModel(t) // no beta, no versions
	press(m, "G", "l", "V")
	if m.versions != nil {
		t.Fatal("V should not open the panel while the feature is off")
	}
	if !strings.Contains(m.flash, "beta") || !strings.Contains(m.flash, "Settings") {
		t.Errorf("flash = %q: an off feature must say where to switch it on", m.flash)
	}
	m = newTestModelWith(t, Options{Beta: true}) // beta mode alone
	press(m, "G", "l", "V")
	if m.versions != nil {
		t.Error("beta mode alone should not open it")
	}
}

func TestANoteWithNoHistorySaysSoRatherThanOpeningEmpty(t *testing.T) {
	m := newVersionsModel(t)
	press(m, "G", "l") // Welcome.md, untouched by Skrin
	press(m, "V")
	if m.versions != nil {
		t.Error("there is nothing to show yet")
	}
	if !strings.Contains(m.flash, "No earlier versions") {
		t.Errorf("flash = %q: it should say why, and that versions are kept from now on", m.flash)
	}
}

func TestTheLineRunsFromOldestToNowAndStartsOneStepBack(t *testing.T) {
	m := versionsFixture(t)
	m.open("Note.md")
	press(m, "V")
	if m.versions == nil {
		t.Fatal("V should open the panel once there are versions")
	}
	v := m.versions
	if len(v.saved) != 2 {
		t.Fatalf("saved = %d versions, want the two Skrin wrote over", len(v.saved))
	}
	if v.saved[0].Content != "# Note\n\nfirst\n" {
		t.Errorf("oldest first: version 1 = %q", v.saved[0].Content)
	}
	if v.saved[1].Content != "# Note\n\nsecond\n" {
		t.Errorf("version 2 = %q", v.saved[1].Content)
	}
	if v.now != "# Note\n\nthird\n" {
		t.Errorf("now = %q", v.now)
	}
	// It opens where u would take you: the newest saved version.
	if v.cur != 1 {
		t.Errorf("cur = %d, want the newest saved version", v.cur)
	}
	if out := versionsText(m); !strings.Contains(out, "second") {
		t.Errorf("the panel should show that version's text:\n%s", out)
	}
}

func TestMovingInTimeShowsEachVersionAndStopsAtBothEnds(t *testing.T) {
	m := versionsFixture(t)
	m.open("Note.md")
	press(m, "V", "left")
	if out := versionsText(m); !strings.Contains(out, "first") {
		t.Errorf("← should go further back:\n%s", out)
	}
	press(m, "left", "left") // already at the oldest
	if m.versions.cur != 0 {
		t.Errorf("cur = %d, want to stop at the oldest", m.versions.cur)
	}
	press(m, "right", "right") // to now
	if m.versions.cur != len(m.versions.saved) {
		t.Errorf("cur = %d, want the right-hand end: the note as it is", m.versions.cur)
	}
	out := versionsText(m)
	if !strings.Contains(out, "third") || !strings.Contains(out, "as it is now") {
		t.Errorf("the end of the line is the note as it is:\n%s", out)
	}
	press(m, "right")
	if m.versions.cur != len(m.versions.saved) {
		t.Error("there is nothing past now")
	}
}

func TestDShowsWhatChangedAgainstNow(t *testing.T) {
	m := versionsFixture(t)
	m.open("Note.md")
	press(m, "V", "d")
	out := versionsText(m)
	if !strings.Contains(out, "-second") || !strings.Contains(out, "+third") {
		t.Errorf("d should show the diff against now:\n%s", out)
	}
	press(m, "d")
	if out := versionsText(m); strings.Contains(out, "+third") {
		t.Errorf("d again should go back to the text itself:\n%s", out)
	}
}

func TestTheStripSaysWhereYouAreAndHowFarFromNow(t *testing.T) {
	m := versionsFixture(t)
	m.open("Note.md")
	press(m, "V")
	out := versionsText(m)
	if !strings.Contains(out, "2 of 3") {
		t.Errorf("the strip should count the versions and now:\n%s", out)
	}
	if !strings.Contains(out, "+1 −1 against now") {
		t.Errorf("the strip should say how far this version is from now:\n%s", out)
	}
}

func TestEnterPutsAVersionBackAndUndoTakesItOffAgain(t *testing.T) {
	m := versionsFixture(t)
	m.open("Note.md")
	press(m, "V", "left") // the oldest version
	press(m, "enter")
	if m.versions != nil {
		t.Error("restoring should close the panel")
	}
	src, err := m.vault.Read("Note.md")
	if err != nil {
		t.Fatal(err)
	}
	if src != "# Note\n\nfirst\n" {
		t.Errorf("the note should read as that version:\n%s", src)
	}
	if !strings.Contains(m.flash, "Restored") || !strings.Contains(m.flash, "undoes") {
		t.Errorf("flash = %q", m.flash)
	}
	press(m, "U")
	back, _ := m.vault.Read("Note.md")
	if back != "# Note\n\nthird\n" {
		t.Errorf("U should put the note back as it was before the restore:\n%s", back)
	}
}

func TestPuttingBackKeepsTheTextItWroteOverSoUAlsoWorks(t *testing.T) {
	m := versionsFixture(t)
	m.open("Note.md")
	press(m, "V", "left", "enter")
	// u is the other way back: it walks the snapshots, and the text that
	// was there at the restore must be the newest of them.
	press(m, "u")
	src, _ := m.vault.Read("Note.md")
	if src != "# Note\n\nthird\n" {
		t.Errorf("u should step back to the text the restore wrote over:\n%s", src)
	}
}

func TestEnterOnNowChangesNothingAndSaysSo(t *testing.T) {
	m := versionsFixture(t)
	m.open("Note.md")
	press(m, "V", "right") // now
	press(m, "enter")
	if m.versions == nil {
		t.Fatal("a refusal should leave the panel open")
	}
	if !strings.Contains(m.flash, "as it is") {
		t.Errorf("flash = %q", m.flash)
	}
	src, _ := m.vault.Read("Note.md")
	if src != "# Note\n\nthird\n" {
		t.Errorf("nothing should have been written:\n%s", src)
	}
}

func TestItNeverOpensOverANoteThatIsBeingEdited(t *testing.T) {
	m := versionsFixture(t)
	m.open("Note.md")
	press(m, "enter") // into the editor
	if m.editor == nil {
		t.Fatal("setup: the editor should be open")
	}
	// From the keyboard V can't even get here — with the editor open it is
	// a letter being typed, as it must be. The guard is for every other way
	// in, and this is the one thing the time machine must never do: write
	// over text that is still being typed.
	m.openVersions()
	if m.versions != nil {
		t.Fatal("it must not open over text that is still being typed")
	}
	if !strings.Contains(m.flash, "ctrl+s") || !strings.Contains(m.flash, "esc") {
		t.Errorf("flash = %q: the refusal should name the way out", m.flash)
	}
}

func TestVIsATypedLetterWhileTheEditorIsOpen(t *testing.T) {
	m := versionsFixture(t)
	m.open("Note.md")
	press(m, "enter")
	before := m.editor.Text()
	press(m, "V")
	if m.editor.Text() == before {
		t.Error("with the editor open, V is a letter: it should be typed")
	}
	if m.versions != nil {
		t.Error("and it should certainly not open a panel")
	}
}

func TestEscClosesTheTimeMachineWithoutTouchingAnything(t *testing.T) {
	m := versionsFixture(t)
	m.open("Note.md")
	press(m, "V", "left", "esc")
	if m.versions != nil {
		t.Error("esc should close it")
	}
	if !strings.Contains(m.flash, "nothing changed") {
		t.Errorf("flash = %q", m.flash)
	}
	src, _ := m.vault.Read("Note.md")
	if src != "# Note\n\nthird\n" {
		t.Errorf("the note should be untouched:\n%s", src)
	}
}

func TestTheTimeMachineFillsTheFrameExactly(t *testing.T) {
	m := versionsFixture(t)
	m.open("Note.md")
	press(m, "V")
	checkFrame(t, m, "the time machine")
	press(m, "d")
	checkFrame(t, m, "the time machine's diff")
}

func TestItIsOutOfTheWayEntirelyWhenOff(t *testing.T) {
	named := func(m *Model) bool {
		for _, c := range m.mainPaletteItems() {
			if strings.Contains(c.label, "Time machine") {
				return true
			}
		}
		return false
	}
	if named(newTestModel(t)) {
		t.Error("an off experiment should not offer a palette row")
	}
	if !named(newVersionsModel(t)) {
		t.Error("switched on, it should be findable by name in the palette")
	}
}
