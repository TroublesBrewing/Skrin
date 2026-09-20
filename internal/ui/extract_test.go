package ui

import (
	"os"
	"strings"
	"testing"
)

const longNote = "# Dagbok\n\nTankar om dagen.\n\n## Stoicism\n\nDet enda vi rår över är våra egna omdömen.\nAllt annat tillhör världen.\n\n## Annat\n\nsist\n"

func extractModel(t *testing.T) *Model {
	t.Helper()
	m := newTestModel(t)
	if err := os.WriteFile(m.vault.Abs("Welcome.md"), []byte(longNote), 0o644); err != nil {
		t.Fatal(err)
	}
	m.reload()
	press(m, "G", "l") // reading Welcome.md
	return m
}

// The whole point: the lines leave, a link stays, and one U puts it all
// back — the note that was made and the hole it left.
func TestPullingLinesIntoTheirOwnNote(t *testing.T) {
	m := extractModel(t)
	press(m, "v")
	selTo(m, 0)        // the cursor to the title
	press(m, "J", "J") // and grow down over what follows
	press(m, "x")
	if m.prompt != nil {
		t.Fatalf("the first line names it, so nothing should be asked: %+v", m.prompt)
	}
	if !m.vault.Exists("Dagbok.md") {
		t.Fatalf("no note made; flash %q", m.flash)
	}
	made := read(m, "Dagbok.md")
	if !strings.Contains(made, "# Dagbok") || !strings.Contains(made, "Tankar om dagen.") {
		t.Errorf("the note should hold the lines:\n%s", made)
	}
	left := read(m, "Welcome.md")
	if !strings.Contains(left, "[[Dagbok]]") {
		t.Errorf("a link should take their place:\n%s", left)
	}
	if strings.Contains(left, "Tankar om dagen.") {
		t.Errorf("and the lines should be gone from here:\n%s", left)
	}
	if !strings.Contains(left, "## Stoicism") || !strings.Contains(left, "sist") {
		t.Errorf("the rest of the note must stay:\n%s", left)
	}
	if !strings.Contains(m.flash, "U undoes it") {
		t.Errorf("flash = %q", m.flash)
	}

	press(m, "U")
	if m.vault.Exists("Dagbok.md") {
		t.Error("U should take the new note back")
	}
	if got := read(m, "Welcome.md"); got != longNote {
		t.Errorf("and put the lines back, exactly:\n%q", got)
	}
}

// A first line with nothing to go on is asked about rather than turned
// into a name nobody could find again.
func TestProseNamesItself(t *testing.T) {
	m := extractModel(t)
	press(m, "e")
	m.editor.MoveTo(2, 0) // "Tankar om dagen."
	press(m, "shift+down")
	press(m, "alt+x")
	if m.prompt != nil {
		t.Fatalf("prose names itself: %+v", m.prompt)
	}
	if !m.vault.Exists("Tankar om dagen.md") {
		t.Errorf("no note; flash %q", m.flash)
	}
}

func TestAnEmptyFirstLineIsAskedAbout(t *testing.T) {
	m := extractModel(t)
	press(m, "e")
	m.editor.MoveTo(1, 0) // the blank line under the heading
	press(m, "shift+down")
	press(m, "alt+x")
	if m.prompt == nil {
		t.Fatalf("an empty line gives no name, so Skrin should ask; flash %q", m.flash)
	}
	typeText(m, "Egen titel")
	press(m, "enter")
	if !m.vault.Exists("Egen titel.md") {
		t.Errorf("the answer should name it; flash %q", m.flash)
	}
}

func TestPullingFromTheEditorLeavesTheLinkThere(t *testing.T) {
	m := extractModel(t)
	press(m, "e")
	m.editor.MoveTo(4, 0) // ## Stoicism
	press(m, "shift+down", "shift+down", "shift+down")
	press(m, "alt+x")
	if !m.vault.Exists("Stoicism.md") {
		t.Fatalf("no note; flash %q", m.flash)
	}
	if !strings.Contains(m.editor.Text(), "[[Stoicism]]") {
		t.Errorf("the editor should hold the link now:\n%s", m.editor.Text())
	}
	if strings.Contains(m.editor.Text(), "Det enda vi rår över") {
		t.Errorf("and not the lines:\n%s", m.editor.Text())
	}
}

func TestItRefusesToOverwriteANoteThatExists(t *testing.T) {
	m := extractModel(t)
	if err := os.WriteFile(m.vault.Abs("Stoicism.md"), []byte("mitt eget\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.reload()
	press(m, "G", "l", "e")
	m.editor.MoveTo(4, 0) // ## Stoicism
	press(m, "shift+down")
	press(m, "alt+x")
	if got := read(m, "Stoicism.md"); got != "mitt eget\n" {
		t.Errorf("an existing note must not be written over: %q", got)
	}
	if m.prompt == nil || !strings.Contains(m.prompt.err+m.flash, "already exists") {
		t.Errorf("and it should say so: prompt %+v flash %q", m.prompt, m.flash)
	}
}

func TestNothingSelectedSaysWhatToDo(t *testing.T) {
	m := extractModel(t)
	press(m, "x")
	if !strings.Contains(m.flash, "Select the lines") {
		t.Errorf("flash = %q", m.flash)
	}
}

func TestNamesAreCleanedTheWayObsidianCleansThem(t *testing.T) {
	for in, want := range map[string]string{
		"## En rubrik":         "En rubrik",
		"- [ ] en uppgift":     "en uppgift",
		"> ett citat":          "ett citat",
		"med / och : och #":    "med och och",
		"   spaces    mitt i ": "spaces mitt i",
		"###":                  "",
		"":                     "",
	} {
		if got := noteName([]string{in}); got != want {
			t.Errorf("noteName(%q) = %q, want %q", in, got, want)
		}
	}
}

// The half that matters: pulled from the editor, one U must put back both
// the note that was made and the lines it took.
func TestUndoFromTheEditorPutsBothHalvesBack(t *testing.T) {
	m := extractModel(t)
	press(m, "e")
	m.editor.MoveTo(4, 0) // ## Stoicism
	press(m, "shift+down", "shift+down", "shift+down")
	press(m, "alt+x")
	if !m.vault.Exists("Stoicism.md") {
		t.Fatalf("no note; flash %q", m.flash)
	}
	press(m, "esc") // leave the editor, saving
	press(m, "U")
	if m.vault.Exists("Stoicism.md") {
		t.Error("U should take the new note back")
	}
	if got := read(m, "Welcome.md"); got != longNote {
		t.Errorf("and put the lines back exactly:\n%q", got)
	}
}
