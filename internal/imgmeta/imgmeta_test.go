package imgmeta

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writePNG(t *testing.T, path string, w, h int) int64 {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return int64(buf.Len())
}

func TestReadReturnsPixelDimensionsAndFileSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "photo.png")
	wantSize := writePNG(t, path, 32, 16)

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Width != 32 || got.Height != 16 {
		t.Errorf("dims = %dx%d, want 32x16", got.Width, got.Height)
	}
	if got.Size != wantSize {
		t.Errorf("size = %d, want %d", got.Size, wantSize)
	}
}

func TestReadOnAMissingFileIsNotExist(t *testing.T) {
	_, err := Read(filepath.Join(t.TempDir(), "gone.png"))
	if !os.IsNotExist(err) {
		t.Fatalf("err = %v, want a not-exist error", err)
	}
}

func TestReadOnAnUnsupportedFormatErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(path, []byte("just text, not an image"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Fatal("want an error for a non-image file, got nil")
	}
}

func TestIsImageKnowsTheExtensionsSkrinRenders(t *testing.T) {
	yes := []string{"photo.png", "Photo.PNG", "shot.jpg", "shot.jpeg", "anim.gif", "pic.webp", "old.bmp"}
	for _, n := range yes {
		if !IsImage(n) {
			t.Errorf("IsImage(%q) = false, want true", n)
		}
	}
	no := []string{"note.md", "cover.svg", "readme", "archive.zip"}
	for _, n := range no {
		if IsImage(n) {
			t.Errorf("IsImage(%q) = true, want false", n)
		}
	}
}
