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

func TestImagesOffAlwaysShowsPlaceholder(t *testing.T) {
	// The zero ImageOptions (config images = false, or Thumbnail unset)
	// always renders the placeholder, even when Meta finds a perfectly
	// good image.
	lines := renderImages(t, "![[Assets/photo.png]]", 80, ImageOptions{
		Meta: func(string) (int, int, int64, ImageStatus) { return 1920, 1080, 100, ImageOK },
	})
	if lines[0].Image.Rows != 1 {
		t.Errorf("Image.Rows = %d, want 1 (the placeholder), without a Thumbnail func", lines[0].Image.Rows)
	}
	if text := ansi.Strip(lines[0].Text); !strings.Contains(text, "1920×1080") {
		t.Errorf("placeholder text = %q, want its dimensions", text)
	}
}

func TestImageThumbnailRendersWhenReady(t *testing.T) {
	want := []string{"AAAA", "BBBB", "CCCC"}
	var gotCols, gotRows int
	lines := renderImages(t, "before\n![[Assets/photo.png]]\nafter", 80, ImageOptions{
		Meta: func(string) (int, int, int64, ImageStatus) { return 1000, 2000, 100, ImageOK },
		Thumbnail: func(target string, cols, rows int) ([]string, bool) {
			gotCols, gotRows = cols, rows
			return want, true
		},
	})
	var imgLines []Line
	for _, l := range lines {
		if l.Src == 1 {
			imgLines = append(imgLines, l)
		}
	}
	if len(imgLines) != len(want) {
		t.Fatalf("got %d image display lines, want %d", len(imgLines), len(want))
	}
	for i, l := range imgLines {
		if ansi.Strip(l.Text) != want[i] {
			t.Errorf("row %d: Text = %q, want %q", i, ansi.Strip(l.Text), want[i])
		}
		if l.Image.Row != i {
			t.Errorf("row %d: Image.Row = %d, want %d", i, l.Image.Row, i)
		}
		if l.Image.Rows != len(want) {
			t.Errorf("row %d: Image.Rows = %d, want %d", i, l.Image.Rows, len(want))
		}
		if l.Src != 1 {
			t.Errorf("row %d: Src = %d, want 1 (every row maps back to its source line)", i, l.Src)
		}
		if len(l.Links) != 1 || l.Links[0].Target != "Assets/photo.png" {
			t.Errorf("row %d: Links = %+v, want the embed to stay a link", i, l.Links)
		}
	}
	if plainAt(lines, 0) != "before" || plainAt(lines, 2) != "after" {
		t.Errorf("surrounding lines disturbed: %q / %q", plainAt(lines, 0), plainAt(lines, 2))
	}
	if gotCols == 0 || gotRows == 0 {
		t.Errorf("Thumbnail was asked for a zero-sized box: cols=%d rows=%d", gotCols, gotRows)
	}
}

func TestImageThumbnailCacheMissFallsBackToPlaceholder(t *testing.T) {
	lines := renderImages(t, "![[Assets/photo.png]]", 80, ImageOptions{
		Meta:      func(string) (int, int, int64, ImageStatus) { return 1920, 1080, 100, ImageOK },
		Thumbnail: func(string, int, int) ([]string, bool) { return nil, false },
	})
	if lines[0].Image.Rows != 1 {
		t.Errorf("Image.Rows = %d, want 1 (the placeholder) on a cache miss", lines[0].Image.Rows)
	}
	if text := ansi.Strip(lines[0].Text); !strings.Contains(text, "1920×1080") {
		t.Errorf("placeholder text = %q, want its dimensions", text)
	}
}

func TestThumbSizeMath(t *testing.T) {
	cases := []struct {
		w, h, maxCols, maxRows, wantCols, wantRows int
	}{
		{1920, 1080, 28, 12, 28, 8},  // landscape, well within the row cap
		{100, 100, 28, 12, 24, 12},   // square: rows would be 14, so rows cap to 12 and cols shrink
		{1000, 2000, 28, 12, 12, 12}, // tall portrait: capped by rows, cols shrink to keep aspect
		{0, 0, 28, 12, 28, 1},        // unknown dimensions: never crash, always at least 1 row
		{1920, 1080, 0, 12, 1, 1},    // no columns to work with at all
	}
	for _, c := range cases {
		gotCols, gotRows := thumbSize(c.w, c.h, c.maxCols, c.maxRows)
		if gotCols != c.wantCols || gotRows != c.wantRows {
			t.Errorf("thumbSize(%d,%d,%d,%d) = %d,%d, want %d,%d", c.w, c.h, c.maxCols, c.maxRows, gotCols, gotRows, c.wantCols, c.wantRows)
		}
	}
}
