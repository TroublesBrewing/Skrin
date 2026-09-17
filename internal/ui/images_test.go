package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// writePNG drops a tiny real PNG at rel inside root, so imgmeta can read
// its real header and renderThumbnail can actually decode it.
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

// flattenCmd runs cmd and every Cmd it fans out to via tea.Batch,
// collecting the leaf Msgs.
func flattenCmd(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	return flattenMsg(cmd())
}

func flattenMsg(msg tea.Msg) []tea.Msg {
	if msg == nil {
		return nil
	}
	if bm, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range bm {
			out = append(out, flattenCmd(c)...)
		}
		return out
	}
	if v := reflect.ValueOf(msg); v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeOf((*tea.Cmd)(nil)).Elem() {
		var out []tea.Msg
		for i := 0; i < v.Len(); i++ {
			if c, ok := v.Index(i).Interface().(tea.Cmd); ok {
				out = append(out, flattenCmd(c)...)
			}
		}
		return out
	}
	return []tea.Msg{msg}
}

func imageThumbMsgs(msgs []tea.Msg) []imageThumbMsg {
	var out []imageThumbMsg
	for _, msg := range msgs {
		if t, ok := msg.(imageThumbMsg); ok {
			out = append(out, t)
		}
	}
	return out
}

// Before its background decode has run, a found image still renders as
// the placeholder frame — never blank, never garbled — and the frame
// still fills the terminal exactly.
func TestImageEmbedRendersAsPlaceholderBeforeDecoding(t *testing.T) {
	m := newImageTestModel(t, Options{Images: true})
	press(m, "1")
	m.files.selectPath("Pic.md")
	press(m, "enter")
	checkFrame(t, m, "note with an image embed, not yet decoded")

	var found bool
	for _, l := range m.lines {
		if l.Image == nil {
			continue
		}
		found = true
		if l.Image.Rows != 1 {
			t.Errorf("Image.Rows = %d, want 1 (the placeholder) before decoding finishes", l.Image.Rows)
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

// Turning the config option off must force placeholders forever, never
// queuing a decode at all.
func TestImageEmbedConfigOffForcesPlaceholder(t *testing.T) {
	m := newImageTestModel(t, Options{Images: false})
	press(m, "1")
	m.files.selectPath("Pic.md")
	_, cmd := m.Update(key("enter"))
	checkFrame(t, m, "note with an image embed, images off in config")

	for _, l := range m.lines {
		if l.Image != nil && l.Image.Rows != 1 {
			t.Errorf("Image.Rows = %d, want 1 (the placeholder) with images off in config", l.Image.Rows)
		}
	}
	if len(imageThumbMsgs(flattenCmd(cmd))) != 0 {
		t.Error("images off in config should never queue a thumbnail decode")
	}
}

// Opening a note with a found, never-before-seen image queues a
// background decode; once its imageThumbMsg comes back, the note
// re-renders with the real preview in place of the placeholder.
func TestImageEmbedSwitchesToThumbnailOnceDecoded(t *testing.T) {
	m := newImageTestModel(t, Options{Images: true})
	press(m, "1")
	m.files.selectPath("Pic.md")
	_, cmd := m.Update(key("enter"))

	got := imageThumbMsgs(flattenCmd(cmd))
	if len(got) != 1 {
		t.Fatalf("got %d imageThumbMsg, want 1 for the never-before-seen image", len(got))
	}
	msg := got[0]
	if msg.abs != m.vault.Abs("Assets/photo.png") || msg.lines == nil {
		t.Fatalf("imageThumbMsg = %+v, want a populated decode of the real PNG", msg)
	}

	m.Update(msg)
	checkFrame(t, m, "note with an image embed, thumbnail decoded")

	var pixelRows int
	for _, l := range m.lines {
		if l.Image != nil && l.Image.Path == "Assets/photo.png" {
			pixelRows++
		}
	}
	if pixelRows != len(msg.lines) {
		t.Errorf("got %d image display lines, want %d (one per decoded row)", pixelRows, len(msg.lines))
	}
	if pixelRows <= 1 {
		t.Error("expected more than the one-row placeholder once decoded")
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

func TestImagesAreOfferedByWikilinkCompletion(t *testing.T) {
	m := newImageTestModel(t, Options{Images: true})
	m.files.selectPath("Welcome.md")
	press(m, "enter", "e")
	typeText(m, "[[photo")
	if m.complete == nil {
		t.Fatal("[[photo should suggest the image")
	}
	var found bool
	for _, it := range m.complete.items {
		if it.label == "photo.png" {
			found = true
			if it.insert != "Assets/photo.png" && it.insert != "photo.png" {
				t.Errorf("insert = %q", it.insert)
			}
		}
	}
	if !found {
		t.Error("photo.png missing from [[ completion")
	}
}

func TestGoToNoteListsImagesAndRevealsThemInFiles(t *testing.T) {
	m := newImageTestModel(t, Options{Images: true})
	press(m, "g")
	typeText(m, "photo")
	var found bool
	for _, i := range m.chooser.matches {
		if m.chooser.items[i].label == "photo.png" {
			found = true
		}
	}
	if !found {
		t.Fatal("Go to note should list photo.png")
	}
	press(m, "enter")
	if m.chooser != nil {
		t.Error("choosing an image should close the chooser")
	}
	if m.focus != paneFiles || m.files.selected().Rel != "Assets/photo.png" {
		t.Errorf("focus=%v selected=%q, want Files focused on Assets/photo.png", m.focus, m.files.selected().Rel)
	}
}

// A cache miss must not decode, scale and encode the image inline while
// rendering: that work has to happen inside the Cmd Update returns, off
// the main goroutine, or a keystroke that first scrolls a
// never-before-seen cover into view would block the whole program until
// the decode finishes.
func TestImageThumbnailDefersDecodingOfACacheMiss(t *testing.T) {
	m := newImageTestModel(t, Options{Images: true})
	press(m, "1")
	m.files.selectPath("Pic.md")
	_, cmd := m.Update(key("enter"))

	abs := m.vault.Abs("Assets/photo.png")
	for key := range m.thumbCache {
		t.Fatalf("test setup: %q must not be cached yet", key)
	}
	got := imageThumbMsgs(flattenCmd(cmd))
	if len(got) != 1 {
		t.Fatalf("got %d imageThumbMsg, want 1", len(got))
	}
	if got[0].abs != abs {
		t.Errorf("imageThumbMsg.abs = %q, want %q", got[0].abs, abs)
	}

	m.Update(got[0])
	key := thumbKey(got[0].abs, got[0].cols, got[0].rows)
	if e, ok := m.thumbCache[key]; !ok || e.lines == nil {
		t.Error("Update should cache the thumbnail once imageThumbMsg arrives")
	}
	if m.pendingThumb[key] {
		t.Error("pendingThumb should be cleared once the result comes back")
	}
}

// Once a note's image is cached, opening it again must not queue another
// decode — the whole point of the cache is that a file already seen
// doesn't get re-decoded on every visit.
func TestImageThumbnailCacheIsReusedOnReopen(t *testing.T) {
	m := newImageTestModel(t, Options{Images: true})
	press(m, "1")
	m.files.selectPath("Pic.md")
	_, cmd := m.Update(key("enter"))
	got := imageThumbMsgs(flattenCmd(cmd))
	if len(got) != 1 {
		t.Fatalf("got %d imageThumbMsg, want 1", len(got))
	}
	m.Update(got[0])

	press(m, "esc") // back to Files, closing the note
	m.files.selectPath("Pic.md")
	_, cmd = m.Update(key("enter"))
	if more := imageThumbMsgs(flattenCmd(cmd)); len(more) != 0 {
		t.Errorf("reopening a cached image queued another decode: %+v", more)
	}
}
