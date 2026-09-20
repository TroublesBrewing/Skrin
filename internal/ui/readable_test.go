package ui

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

const longPara = "Stoikerna skilde på det vi kan styra och det vi inte kan styra, och de menade att sinnesro kommer av att lägga sin kraft på det förra och släppa det senare, hur svårt det än är i stunden."

// readableModel opens a note with a long paragraph in a terminal w wide.
func readableModel(t *testing.T, w int, on bool) *Model {
	t.Helper()
	m := newTestModel(t)
	if err := os.WriteFile(m.vault.Abs("Welcome.md"), []byte("# Welcome\n\n"+longPara+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.reload()
	m.opts.ReadableWidth = on
	m.Update(tea.WindowSizeMsg{Width: w, Height: 30})
	press(m, "G", "l")
	return m
}

// paraLines are the frame's rows holding the paragraph, stripped of style,
// cut to the note pane: what's right of Files' border.
func paraLines(m *Model) []string {
	var out []string
	for _, row := range strings.Split(ansi.Strip(m.render()), "\n") {
		if strings.Contains(row, "Stoikerna") || strings.Contains(row, "sinnesro") || strings.Contains(row, "stunden") {
			out = append(out, row)
		}
	}
	return out
}

func TestReadableWidthKeepsTheColumnNarrowAndCentred(t *testing.T) {
	m := readableModel(t, 200, true)
	if got := m.noteTextW(); got != readableWidth {
		t.Fatalf("text width %d, want %d", got, readableWidth)
	}
	if m.noteMargin() <= 0 {
		t.Fatalf("a wide pane should have a margin: %d", m.noteMargin())
	}
	checkFrame(t, m, "readable width, wide")
	for _, l := range m.lines {
		if w := ansi.StringWidth(l.Text); w > readableWidth {
			t.Errorf("a line %d wide: %q", w, ansi.Strip(l.Text))
		}
	}
	rows := paraLines(m)
	if len(rows) < 3 {
		t.Fatalf("the paragraph should wrap onto several lines: %q", rows)
	}
	l := m.layout()
	start := strings.Index(rows[0], "Stoikerna")
	want := l.filesW + 1 + 1 + m.noteMargin()
	if got := ansi.StringWidth(rows[0][:start]); got != want {
		t.Errorf("the text should start after the margin, at column %d: %d", want, got)
	}
}

func TestReadableWidthOffFillsThePane(t *testing.T) {
	m := readableModel(t, 200, false)
	if m.noteTextW() <= readableWidth || m.noteMargin() != 0 {
		t.Errorf("off: width %d, margin %d", m.noteTextW(), m.noteMargin())
	}
}

func TestReadableWidthLeavesNarrowPanesAndZenAlone(t *testing.T) {
	m := readableModel(t, 100, true)
	if m.noteMargin() != 0 || m.noteTextW() != m.fullTextW() {
		t.Errorf("a pane narrower than %d keeps its width: width %d, margin %d", readableWidth, m.noteTextW(), m.noteMargin())
	}
	m = readableModel(t, 200, true)
	press(m, "z")
	if !m.zen || m.noteMargin() != 0 || m.noteTextW() != readableWidth {
		t.Errorf("zen keeps its own width and centring: zen %v, width %d, margin %d", m.zen, m.noteTextW(), m.noteMargin())
	}
	checkFrame(t, m, "zen with readable width on")
}

func TestReadableWidthInTheEditor(t *testing.T) {
	m := readableModel(t, 200, true)
	os.WriteFile(m.vault.Abs("Tagged.md"), []byte("#stoa\n"), 0o644)
	m.reload()
	press(m, "e")
	if m.editor == nil {
		t.Fatal("setup: no editor")
	}
	checkFrame(t, m, "editor, readable width")
	rows := paraLines(m)
	if len(rows) < 3 {
		t.Fatalf("the editor should wrap the paragraph at the readable width too: %q", rows)
	}
	m.editor.HandleKey(keyPress("ctrl+end"))
	press(m, "enter")
	typeText(m, "#st")
	if m.complete == nil {
		t.Fatal("setup: #st should suggest #stoa")
	}
	_, x, _ := m.completionBox()
	if want := m.layout().filesW + 2 + m.noteMargin() + len("#st"); x != want {
		t.Errorf("the popup should open by the cursor, after the margin: x %d, want %d", x, want)
	}
}

func TestReadableWidthInTheSplit(t *testing.T) {
	m := readableModel(t, 260, true)
	// From Welcome in Files: up to Filosofi, open it, and skim down onto
	// Stoic, which opens it in a split beside Welcome.
	press(m, "1", "alt+k", "alt+k", "l", "alt+j", "alt+j")
	if m.split == nil {
		t.Fatal("setup: the skim should have opened a split")
	}
	w, pad := m.splitTextW()
	if w > readableWidth || pad <= 0 {
		t.Errorf("the split's note: width %d, margin %d", w, pad)
	}
	checkFrame(t, m, "split, readable width")
}

func TestReadableWidthSettingsToggle(t *testing.T) {
	m := readableModel(t, 200, true)
	for _, it := range m.settingsItems() {
		if it.label == "Readable line length" {
			it.set(m, false)
		}
	}
	m.settle()
	if m.opts.ReadableWidth || m.opts.Config.RenderReadableWidth() || m.noteTextW() <= readableWidth {
		t.Errorf("turning it off should widen the note at once and in config: width %d", m.noteTextW())
	}
}
