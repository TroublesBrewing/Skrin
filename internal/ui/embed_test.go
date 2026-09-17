package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/index"
)

func writeEmbedFixtures(t *testing.T, m *Model) {
	t.Helper()
	if err := m.vault.Write("Two Sections.md", "# Two Sections\n## First\nFirst para.\n## Second\nSecond para.\n### Nested\nNested line.\n## Third\nThird para.\n"); err != nil {
		t.Fatal(err)
	}
	if err := m.vault.Write("Embed.md", "# Embed\n![[Zeno]]\n"); err != nil {
		t.Fatal(err)
	}
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
}

func openAndRender(t *testing.T, m *Model, rel string) []string {
	t.Helper()
	m.files.selectPath(rel)
	press(m, "enter")
	var out []string
	for _, l := range m.lines {
		out = append(out, ansi.Strip(l.Text))
	}
	return out
}

func TestNoteEmbedTranscludesWholeNoteInTheNotePane(t *testing.T) {
	m := newTestModel(t)
	writeEmbedFixtures(t, m)
	lines := openAndRender(t, m, "Embed.md")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Zeno") || !strings.Contains(joined, "Teacher of") {
		t.Errorf("Embed.md rendered = %q, want Zeno's transcluded content", joined)
	}
	if strings.Contains(joined, "⧉ Zeno") {
		t.Errorf("Embed.md rendered = %q, should transclude rather than link", joined)
	}
	// Every transcluded row must still map back to the ![[Zeno]] line (1).
	for i, l := range m.lines {
		if l.Embed != nil && l.Src != 1 {
			t.Errorf("row %d: Src=%d for an embedded row, want 1", i, l.Src)
		}
	}
	checkFrame(t, m, "note with a transcluded embed")
}

func TestNoteEmbedWithHeadingTranscludesOnlyThatSection(t *testing.T) {
	m := newTestModel(t)
	writeEmbedFixtures(t, m)
	if err := m.vault.Write("Embed.md", "# Embed\n![[Two Sections#Second]]\n"); err != nil {
		t.Fatal(err)
	}
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	lines := openAndRender(t, m, "Embed.md")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Second") || !strings.Contains(joined, "Second para") || !strings.Contains(joined, "Nested") {
		t.Errorf("rendered = %q, want the Second section (with its Nested subsection)", joined)
	}
	if strings.Contains(joined, "First para") || strings.Contains(joined, "Third para") {
		t.Errorf("rendered = %q, want only the Second section, not First or Third", joined)
	}
}

func TestNoteEmbedOfMissingNoteFallsBackToLink(t *testing.T) {
	m := newTestModel(t)
	writeEmbedFixtures(t, m)
	if err := m.vault.Write("Embed.md", "# Embed\n![[Nope]]\n"); err != nil {
		t.Fatal(err)
	}
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	lines := openAndRender(t, m, "Embed.md")
	if !strings.Contains(strings.Join(lines, "\n"), "⧉ Nope") {
		t.Errorf("rendered = %q, want the fallback link for an unresolved embed", lines)
	}
}

func TestNoteEmbedOfMissingHeadingFallsBackToLink(t *testing.T) {
	m := newTestModel(t)
	writeEmbedFixtures(t, m)
	if err := m.vault.Write("Embed.md", "# Embed\n![[Two Sections#Nope]]\n"); err != nil {
		t.Fatal(err)
	}
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	lines := openAndRender(t, m, "Embed.md")
	if !strings.Contains(strings.Join(lines, "\n"), "⧉ Two Sections") {
		t.Errorf("rendered = %q, want the fallback link for an unknown heading", lines)
	}
}

func TestNoteEmbedOfBlockIDFallsBackToLink(t *testing.T) {
	m := newTestModel(t)
	writeEmbedFixtures(t, m)
	if err := m.vault.Write("Embed.md", "# Embed\n![[Two Sections#^some-block]]\n"); err != nil {
		t.Fatal(err)
	}
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	lines := openAndRender(t, m, "Embed.md")
	if !strings.Contains(strings.Join(lines, "\n"), "⧉ Two Sections") {
		t.Errorf("rendered = %q, want the fallback link for a block-id embed", lines)
	}
}

func TestHeadingSectionEndsAtNextHeadingOfEqualOrHigherLevel(t *testing.T) {
	src := "# T\n## First\nA\n## Second\nB\n### Nested\nC\n## Third\nD\n"
	heads := []index.Heading{
		{Level: 1, Text: "T", Line: 0},
		{Level: 2, Text: "First", Line: 1},
		{Level: 2, Text: "Second", Line: 3},
		{Level: 3, Text: "Nested", Line: 5},
		{Level: 2, Text: "Third", Line: 7},
	}
	got := headingSection(src, heads, 3) // "## Second" starts at line 3
	want := "## Second\nB\n### Nested\nC"
	if got != want {
		t.Errorf("headingSection(Second) = %q, want %q", got, want)
	}
	got = headingSection(src, heads, 7) // "## Third", the last heading: runs to EOF
	want = "## Third\nD"
	if got != want {
		t.Errorf("headingSection(Third, to EOF) = %q, want %q", got, want)
	}
}
