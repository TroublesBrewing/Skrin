package markdown

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/theme"
)

func renderEmbeds(t *testing.T, src string, width int, o EmbedOptions) []Line {
	t.Helper()
	lines := Render(src, Options{Width: width, Palette: theme.Default(), Embeds: o})
	for i, l := range lines {
		if w := ansi.StringWidth(l.Text); w > width {
			t.Errorf("line %d is %d cells wide, max %d: %q", i, w, width, ansi.Strip(l.Text))
		}
	}
	return lines
}

func TestNoteEmbedTranscludesContentWhenResolved(t *testing.T) {
	content := func(target string) (string, bool) {
		if target == "Zeno" {
			return "# Zeno\nTeacher of the Stoics.", true
		}
		return "", false
	}
	lines := renderEmbeds(t, "before\n![[Zeno]]\nafter", 80, EmbedOptions{Content: content})
	var embedded []Line
	for _, l := range lines {
		if l.Src == 1 {
			embedded = append(embedded, l)
		}
	}
	if len(embedded) != 2 {
		t.Fatalf("got %d transcluded lines, want 2 (heading + text): %+v", len(embedded), embedded)
	}
	if !strings.Contains(ansi.Strip(embedded[0].Text), "Zeno") {
		t.Errorf("first transcluded line = %q, want the heading", embedded[0].Text)
	}
	for i, l := range embedded {
		if l.Embed == nil || l.Embed.Target != "Zeno" || l.Embed.Row != i || l.Embed.Rows != len(embedded) {
			t.Errorf("row %d Embed = %+v, want Target=Zeno Row=%d Rows=%d", i, l.Embed, i, len(embedded))
		}
	}
	if plainAt(lines, 0) != "before" || plainAt(lines, 2) != "after" {
		t.Errorf("surrounding lines disturbed: %q / %q", plainAt(lines, 0), plainAt(lines, 2))
	}
}

func TestNoteEmbedFallsBackToLinkWhenUnresolved(t *testing.T) {
	content := func(string) (string, bool) { return "", false }
	lines := renderEmbeds(t, "![[Missing]]", 80, EmbedOptions{Content: content})
	if lines[0].Embed != nil {
		t.Errorf("Embed = %+v, want nil when Content can't resolve the target", lines[0].Embed)
	}
	if text := ansi.Strip(lines[0].Text); !strings.Contains(text, "⧉ Missing") {
		t.Errorf("text = %q, want the fallback embed link", text)
	}
}

func TestNoteEmbedWithNilContentIsUnaffected(t *testing.T) {
	// The zero EmbedOptions (Content == nil) is the always-safe default:
	// every embed renders exactly as it did before transclusion existed.
	lines := renderEmbeds(t, "![[Zeno]]", 80, EmbedOptions{})
	if lines[0].Embed != nil {
		t.Errorf("Embed = %+v, want nil with Content == nil", lines[0].Embed)
	}
}

func TestNoteEmbedMixedWithTextStaysInline(t *testing.T) {
	content := func(string) (string, bool) { return "transcluded", true }
	lines := renderEmbeds(t, "See ![[Zeno]] above", 80, EmbedOptions{Content: content})
	if len(lines) != 1 || lines[0].Embed != nil {
		t.Fatalf("expected an ordinary inline line, got %+v", lines)
	}
}

func TestNoteEmbedBlockIDAlwaysFallsBackToLink(t *testing.T) {
	// Content is never even asked about a block-id target: block-level
	// extraction isn't supported, so it always stays a plain link.
	called := false
	content := func(string) (string, bool) { called = true; return "x", true }
	lines := renderEmbeds(t, "![[Zeno#^abc123]]", 80, EmbedOptions{Content: content})
	if called {
		t.Error("Content should not be called for a #^blockid embed")
	}
	if lines[0].Embed != nil {
		t.Errorf("Embed = %+v, want nil for a block-id embed", lines[0].Embed)
	}
}

func TestNoteEmbedNeverRecursesIntoItsOwnContent(t *testing.T) {
	// A embeds B, B's content also has an embed. Only one level ever
	// transcludes: the inner ![[...]] inside B's content renders as a
	// plain link, not another transclusion, cycle-safe by construction.
	content := func(target string) (string, bool) {
		switch target {
		case "B":
			return "B says ![[C]]", true
		case "C":
			return "C's content, which must not appear", true
		}
		return "", false
	}
	lines := renderEmbeds(t, "![[B]]", 80, EmbedOptions{Content: content})
	var text []string
	for _, l := range lines {
		text = append(text, ansi.Strip(l.Text))
	}
	joined := strings.Join(text, "\n")
	if !strings.Contains(joined, "⧉ C") {
		t.Errorf("inner ![[C]] should fall back to a plain link, got %q", joined)
	}
	if strings.Contains(joined, "must not appear") {
		t.Errorf("transclusion recursed a second level: %q", joined)
	}
}

func TestNoteEmbedResolvingToEmptyContentStillTakesARow(t *testing.T) {
	content := func(string) (string, bool) { return "", true }
	lines := renderEmbeds(t, "before\n![[Empty]]\nafter", 80, EmbedOptions{Content: content})
	var embedded []Line
	for _, l := range lines {
		if l.Src == 1 {
			embedded = append(embedded, l)
		}
	}
	if len(embedded) != 1 || embedded[0].Embed == nil || embedded[0].Embed.Rows != 1 {
		t.Fatalf("empty embed rows = %+v, want exactly one row", embedded)
	}
	if plainAt(lines, 2) != "after" {
		t.Errorf("line after the embed disturbed: %q", plainAt(lines, 2))
	}
}

func TestNoteEmbedWithHeadingPassesTheFullTargetToContent(t *testing.T) {
	var got string
	content := func(target string) (string, bool) { got = target; return "section", true }
	renderEmbeds(t, "![[Zeno#Teachings]]", 80, EmbedOptions{Content: content})
	if got != "Zeno#Teachings" {
		t.Errorf("Content called with %q, want %q", got, "Zeno#Teachings")
	}
}

func TestImageEmbedTakesPriorityOverNoteEmbed(t *testing.T) {
	// An image target must never be treated as a note to transclude, even
	// when EmbedOptions.Content is configured.
	called := false
	content := func(string) (string, bool) { called = true; return "x", true }
	lines := renderEmbeds(t, "![[Assets/photo.png]]", 80, EmbedOptions{Content: content})
	if called {
		t.Error("Content should not be called for an image embed")
	}
	if lines[0].Image == nil {
		t.Error("an image embed must still render as an image, not fall through unrendered")
	}
}
