package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// writePNG drops a tiny real PNG at rel inside root, so imgmeta can read
// its real header.
func writePNG(t *testing.T, root, rel string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newImageTestModel(t *testing.T, opts Options) *Model {
	t.Helper()
	m := newTestModelWith(t, opts)
	writePNG(t, m.vault.Root, "Assets/photo.png", 1920, 1080)
	if err := m.vault.Write("Pic.md", "# Pic\n![[Assets/photo.png]]\nSee ![[Assets/missing.png]] too.\n"); err != nil {
		t.Fatal(err)
	}
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	return m
}

// Without a real terminal answering DA1, sixel never turns on: every image
// embed renders as the placeholder frame, and the frame still fills the
// terminal exactly.
func TestImageEmbedRendersAsPlaceholderWithoutSixel(t *testing.T) {
	m := newImageTestModel(t, Options{Images: true})
	press(m, "1")
	m.files.selectPath("Pic.md")
	press(m, "enter")
	checkFrame(t, m, "note with an image embed, no sixel")

	var found bool
	for _, l := range m.lines {
		if l.Image == nil {
			continue
		}
		found = true
		if l.Image.Pixels {
			t.Errorf("line has Image.Pixels set without sixel capability: %+v", l.Image)
		}
		text := ansi.Strip(l.Text)
		if l.Image.Path == "Assets/photo.png" && !strings.Contains(text, "1920×1080") {
			t.Errorf("found image placeholder = %q, want its dimensions", text)
		}
		if l.Image.Path == "Assets/missing.png" && !strings.Contains(text, "not found") {
			t.Errorf("missing image placeholder = %q, want \"not found\"", text)
		}
	}
	if !found {
		t.Fatal("no Image line rendered for the note's embeds")
	}
}

// Turning the config option off must force placeholders even once the
// terminal has claimed sixel and answered a cell size.
func TestImageEmbedConfigOffForcesPlaceholder(t *testing.T) {
	m := newImageTestModel(t, Options{Images: false})
	m.Update(uv.PrimaryDeviceAttributesEvent{1, 2, 4})
	m.Update(uv.CellSizeEvent{Width: 8, Height: 16})
	press(m, "1")
	m.files.selectPath("Pic.md")
	press(m, "enter")
	checkFrame(t, m, "note with an image embed, images off in config")

	for _, l := range m.lines {
		if l.Image != nil && l.Image.Pixels {
			t.Errorf("Image.Pixels set with images off in config: %+v", l.Image)
		}
	}
}

// Once the terminal claims sixel and reports a cell size, a found image
// switches to pixel rows instead of the placeholder frame — and the frame
// still fills the terminal exactly, since the filler rows are blank text.
func TestImageEmbedDrawsPixelsOnceSixelIsKnown(t *testing.T) {
	m := newImageTestModel(t, Options{Images: true})
	press(m, "1")
	m.files.selectPath("Pic.md")
	press(m, "enter")

	m.Update(uv.PrimaryDeviceAttributesEvent{1, 2, 4}) // 4: claims sixel
	m.Update(uv.CellSizeEvent{Width: 8, Height: 16})
	checkFrame(t, m, "note with an image embed, sixel known")

	var pixelRows int
	for _, l := range m.lines {
		if l.Image != nil && l.Image.Path == "Assets/photo.png" {
			if !l.Image.Pixels {
				t.Errorf("found image line didn't switch to pixels once sixel is known: %+v", l.Image)
			}
			pixelRows++
		}
	}
	if pixelRows == 0 {
		t.Error("no pixel rows reserved for the found image")
	}
}

// A cell-size event of 0 (a terminal that answered nonsense) must not turn
// pixels on.
func TestImageEmbedIgnoresZeroCellSize(t *testing.T) {
	m := newImageTestModel(t, Options{Images: true})
	m.Update(uv.PrimaryDeviceAttributesEvent{1, 2, 4})
	m.Update(uv.CellSizeEvent{Width: 0, Height: 0})
	press(m, "1")
	m.files.selectPath("Pic.md")
	press(m, "enter")
	checkFrame(t, m, "note with an image embed, zero cell size")

	for _, l := range m.lines {
		if l.Image != nil && l.Image.Pixels {
			t.Error("Image.Pixels set from a zero cell size")
		}
	}
}

func TestImageMetaResolvesLikeAWikilink(t *testing.T) {
	m := newImageTestModel(t, Options{Images: true})
	w, h, size, status := m.imageMeta("Pic.md")("Assets/photo.png")
	if status != 0 /* markdown.ImageOK */ || w != 1920 || h != 1080 || size <= 0 {
		t.Errorf("imageMeta(found) = %d,%d,%d,%v", w, h, size, status)
	}
	_, _, _, status = m.imageMeta("Pic.md")("Assets/missing.png")
	if status != 1 /* markdown.ImageMissing */ {
		t.Errorf("imageMeta(missing) status = %v, want ImageMissing", status)
	}
}

func TestNoteOriginIsNoneWhenAModalIsUp(t *testing.T) {
	m := newImageTestModel(t, Options{Images: true})
	press(m, "1")
	m.files.selectPath("Pic.md")
	press(m, "enter")
	m.Update(uv.PrimaryDeviceAttributesEvent{1, 2, 4})
	m.Update(uv.CellSizeEvent{Width: 8, Height: 16})
	if _, _, ok := m.noteOrigin(); !ok {
		t.Fatal("noteOrigin should be usable with a note open and nothing over it")
	}
	press(m, "?") // opens the manual
	if _, _, ok := m.noteOrigin(); ok {
		t.Error("noteOrigin should refuse to draw under the manual")
	}
	press(m, "esc")
}
