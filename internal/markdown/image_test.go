package markdown

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/theme"
)

func renderImages(t *testing.T, src string, width int, o ImageOptions) []Line {
	t.Helper()
	lines := Render(src, Options{Width: width, Palette: theme.Default(), Images: o})
	for i, l := range lines {
		if w := ansi.StringWidth(l.Text); w > width {
			t.Errorf("line %d is %d cells wide, max %d: %q", i, w, width, ansi.Strip(l.Text))
		}
	}
	return lines
}

func TestImageEmbedFoundShowsNameDimensionsAndSize(t *testing.T) {
	meta := func(target string) (int, int, int64, ImageStatus) {
		if target == "Assets/photo.png" {
			return 1920, 1080, 1258291, ImageOK
		}
		return 0, 0, 0, ImageMissing
	}
	lines := renderImages(t, "![[Assets/photo.png]]", 80, ImageOptions{Meta: meta})
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1: %+v", len(lines), lines)
	}
	text := ansi.Strip(lines[0].Text)
	if !strings.Contains(text, "photo.png") || !strings.Contains(text, "1920×1080") || !strings.Contains(text, "1.2 MB") {
		t.Errorf("placeholder text = %q, want name, dims and size", text)
	}
	if lines[0].Image == nil || lines[0].Image.Path != "Assets/photo.png" || lines[0].Image.Rows != 1 {
		t.Errorf("Image = %+v", lines[0].Image)
	}
	if len(lines[0].Links) != 1 || lines[0].Links[0].Target != "Assets/photo.png" || !lines[0].Links[0].Wiki {
		t.Errorf("Links = %+v, want the embed to stay a link", lines[0].Links)
	}
}

func TestImageEmbedMissingSaysSo(t *testing.T) {
	lines := renderImages(t, "![[Assets/gone.png]]", 80, ImageOptions{})
	text := ansi.Strip(lines[0].Text)
	if !strings.Contains(text, "gone.png") || !strings.Contains(text, "not found") {
		t.Errorf("placeholder text = %q, want the not-found reason", text)
	}
}

func TestImageEmbedUnsupportedFormatSaysSo(t *testing.T) {
	meta := func(string) (int, int, int64, ImageStatus) { return 0, 0, 0, ImageUnsupported }
	lines := renderImages(t, "![[Assets/weird.png]]", 80, ImageOptions{Meta: meta})
	text := ansi.Strip(lines[0].Text)
	if !strings.Contains(text, "weird.png") || !strings.Contains(text, "unsupported") {
		t.Errorf("placeholder text = %q, want the unsupported reason", text)
	}
}

func TestImageEmbedMarkdownLinkFormAlsoRenders(t *testing.T) {
	meta := func(string) (int, int, int64, ImageStatus) { return 640, 480, 2048, ImageOK }
	lines := renderImages(t, "![a screenshot](Assets/shot.jpg)", 80, ImageOptions{Meta: meta})
	if len(lines) != 1 || lines[0].Image == nil || lines[0].Image.Path != "Assets/shot.jpg" {
		t.Fatalf("lines = %+v", lines)
	}
}

func TestImageEmbedMixedWithTextStaysInline(t *testing.T) {
	// Only a line that is *exactly* one embed gets the placeholder/pixel
	// treatment; mixed with other text it's the ordinary inline embed.
	lines := renderImages(t, "See ![[Assets/photo.png]] above", 80, ImageOptions{
		Meta: func(string) (int, int, int64, ImageStatus) { return 100, 100, 10, ImageOK },
	})
	if len(lines) != 1 || lines[0].Image != nil {
		t.Fatalf("expected an ordinary inline line, got %+v", lines)
	}
	if text := ansi.Strip(lines[0].Text); !strings.Contains(text, "⧉") {
		t.Errorf("text = %q, want the ordinary embed marker", text)
	}
}

func TestNonImageEmbedIsUnaffected(t *testing.T) {
	// ![[Zeno]] transcludes a note, not an image — must render exactly as
	// it always has, with no Images configuration at all.
	lines := renderImages(t, "![[Zeno]]", 80, ImageOptions{})
	if lines[0].Image != nil {
		t.Errorf("Image = %+v, want nil for a non-image embed", lines[0].Image)
	}
	if text := ansi.Strip(lines[0].Text); !strings.Contains(text, "⧉ Zeno") {
		t.Errorf("text = %q, want the unchanged embed rendering", text)
	}
}

func TestImagesFalseConfigNeverDrawsPixels(t *testing.T) {
	// The zero ImageOptions (config images = false, or the terminal never
	// answered) always renders the placeholder, even when Meta finds a
	// perfectly good image.
	lines := renderImages(t, "![[Assets/photo.png]]", 80, ImageOptions{
		Meta: func(string) (int, int, int64, ImageStatus) { return 1920, 1080, 100, ImageOK },
	})
	if lines[0].Image.Pixels {
		t.Errorf("Image.Pixels = true without a cell size, want the placeholder")
	}
}

func TestImagePixelsSpanTheRightNumberOfRowsAndKeepSrcMapping(t *testing.T) {
	// A 1000x2000 image (portrait) at 80 columns, 8x16px cells: pane is
	// 640px wide, scaled height is 1280px, at 16px/row that's 80 rows.
	lines := renderImages(t, "before\n![[Assets/tall.png]]\nafter", 80, ImageOptions{
		Meta:  func(string) (int, int, int64, ImageStatus) { return 1000, 2000, 100, ImageOK },
		CellW: 8,
		CellH: 16,
	})
	var imgLines []Line
	for _, l := range lines {
		if l.Src == 1 {
			imgLines = append(imgLines, l)
		}
	}
	if len(imgLines) != 80 {
		t.Fatalf("got %d image display lines, want 80", len(imgLines))
	}
	for i, l := range imgLines {
		if !l.Image.Pixels {
			t.Errorf("row %d: Pixels = false, want true", i)
		}
		if l.Image.Row != i {
			t.Errorf("row %d: Image.Row = %d, want %d", i, l.Image.Row, i)
		}
		if l.Image.Rows != 80 {
			t.Errorf("row %d: Image.Rows = %d, want 80", i, l.Image.Rows)
		}
		if l.Src != 1 {
			t.Errorf("row %d: Src = %d, want 1 (every row maps back to its source line)", i, l.Src)
		}
	}
	if plainAt(lines, 0) != "before" || plainAt(lines, 2) != "after" {
		t.Errorf("surrounding lines disturbed: %q / %q", plainAt(lines, 0), plainAt(lines, 2))
	}
}

func TestImagePixelsCapAtPaneHeight(t *testing.T) {
	lines := renderImages(t, "![[Assets/tall.png]]", 80, ImageOptions{
		Meta:       func(string) (int, int, int64, ImageStatus) { return 1000, 2000, 100, ImageOK },
		CellW:      8,
		CellH:      16,
		PaneHeight: 20,
	})
	if len(lines) != 20 {
		t.Fatalf("got %d rows, want the pane-height cap of 20", len(lines))
	}
}

func TestImageRowsMath(t *testing.T) {
	cases := []struct {
		w, h, cellsWide, cellW, cellH, maxRows, want int
	}{
		{1920, 1080, 80, 8, 16, 0, 23},  // 640px pane width -> 360px tall -> ceil(360/16)
		{100, 100, 80, 8, 16, 0, 40},    // square at full pane width
		{0, 0, 80, 8, 16, 0, 1},         // unknown dimensions: never crash, always at least 1
		{1000, 2000, 80, 8, 16, 10, 10}, // capped
	}
	for _, c := range cases {
		got := imageRows(c.w, c.h, c.cellsWide, c.cellW, c.cellH, c.maxRows)
		if got != c.want {
			t.Errorf("imageRows(%d,%d,%d,%d,%d,%d) = %d, want %d", c.w, c.h, c.cellsWide, c.cellW, c.cellH, c.maxRows, got, c.want)
		}
	}
}
