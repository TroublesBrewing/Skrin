package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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

// flattenCmd runs cmd and every Cmd it fans out to via tea.Batch or
// tea.Sequence, collecting the leaf Msgs. tea.Sequence's own message type
// is unexported, so a slice-of-Cmd is unwrapped by reflection rather than
// a type switch.
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

func rawStrings(msgs []tea.Msg) []string {
	var out []string
	for _, msg := range msgs {
		if r, ok := msg.(tea.RawMsg); ok {
			out = append(out, fmt.Sprint(r.Msg))
		}
	}
	return out
}

// Drawing an image, then losing noteOrigin (an overlay opens over the
// note), must still clear the rectangle the image was painted into —
// imageDraws can't just return nil once there's nothing new to draw, or
// the sixel pixels from the last frame never get erased and linger as a
// smear over whatever's drawn there next.
func TestImageDrawsClearsPaintedRectWhenNoteGoesAway(t *testing.T) {
	m := newImageTestModel(t, Options{Images: true})
	press(m, "1")
	m.files.selectPath("Pic.md")
	press(m, "enter")
	m.Update(uv.PrimaryDeviceAttributesEvent{1, 2, 4})
	m.Update(uv.CellSizeEvent{Width: 8, Height: 16})

	rel := "Assets/photo.png"
	abs := m.vault.Abs(rel)
	// A cached entry must match the real file's mtime/size to count as
	// fresh, so stat it for real rather than faking those fields.
	fi, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	m.imgCache[abs] = sixelImage{mtime: fi.ModTime().UnixNano(), size: fi.Size(), cellsW: m.layout().noteTextW() * m.cellW, data: "SIXELDATA"}

	msgs := flattenCmd(m.imageDraws())
	draws := rawStrings(msgs)
	if len(draws) == 0 {
		t.Fatal("expected at least one raw draw once the image is cached")
	}
	if len(m.painted) == 0 {
		t.Fatal("expected imageDraws to record what it painted")
	}
	painted := m.painted[0]

	// Update() itself calls imageDraws() at the tail of every message, so
	// grab the Cmd from the "?" press directly rather than pressing then
	// calling imageDraws() again — by then m.painted would already be
	// cleared by that automatic call.
	_, drawCmd := m.Update(key("?")) // opens the manual: noteOrigin now refuses to draw
	msgs = flattenCmd(drawCmd)
	clears := rawStrings(msgs)
	if len(clears) == 0 {
		t.Fatal("expected imageDraws to clear the previously painted rect once the note is hidden")
	}
	want := ansi.SetCursorPosition(painted.col, painted.row)
	if !strings.Contains(clears[0], want) {
		t.Errorf("clear command = %q, want it positioned at the painted rect (%q)", clears[0], want)
	}
	if strings.Contains(clears[0], "SIXELDATA") {
		t.Error("clear command should only write blanks, not repaint sixel data")
	}
	if m.painted != nil {
		t.Error("painted should be nil once nothing is drawn")
	}
	press(m, "esc")
}

// A cold cache miss must not decode, scale and encode the image inline in
// imageDraws: that work has to happen inside the Cmd it returns, off the
// main goroutine, or a keystroke that first scrolls a never-before-seen
// cover into view would block the whole program until the encode
// finishes.
func TestImageDrawsDefersEncodingOfACacheMiss(t *testing.T) {
	m := newImageTestModel(t, Options{Images: true})
	press(m, "1")
	m.files.selectPath("Pic.md")
	press(m, "enter")
	m.Update(uv.PrimaryDeviceAttributesEvent{1, 2, 4})
	// The CellSizeEvent's own Update call already runs imageDraws() once
	// at its tail (every Update does), so capture that Cmd directly
	// rather than calling imageDraws() again afterward — a second call
	// would see the file already marked as mid-encode by the first and
	// skip requesting it again.
	_, cmd := m.Update(uv.CellSizeEvent{Width: 8, Height: 16})

	rel := "Assets/photo.png"
	abs := m.vault.Abs(rel)
	if _, ok := m.imgCache[abs]; ok {
		t.Fatal("test setup: image must not be cached yet")
	}

	if cmd == nil {
		t.Fatal("expected a Cmd to encode the cache miss")
	}
	if _, ok := m.imgCache[abs]; ok {
		t.Fatal("imageDraws must not decode or encode synchronously on a cache miss")
	}

	msgs := flattenCmd(cmd)
	var got imagePixelsMsg
	var found bool
	for _, msg := range msgs {
		if p, ok := msg.(imagePixelsMsg); ok {
			got, found = p, true
		}
	}
	if !found {
		t.Fatal("expected an imagePixelsMsg once the deferred encode runs")
	}
	if got.abs != abs || got.data == "" {
		t.Errorf("imagePixelsMsg = %+v, want a populated result for %q", got, abs)
	}

	if _, cmd := m.Update(got); cmd != nil {
		flattenCmd(cmd) // drain: a follow-up draw once the image is cached
	}
	if _, ok := m.imgCache[abs]; !ok {
		t.Error("Update should cache the image once imagePixelsMsg arrives")
	}
}

// Once an image is drawn, a further Update carrying nothing that actually
// changes what should be on screen — an unrelated key, a watcher tick,
// anything — must not clear and redraw it again. Every clear-then-redraw
// is a real, visible flicker on a terminal that has to rasterize sixel
// data, so doing that on every single message even when nothing changed
// is exactly the kind of "blinking on rerenders" the bug report
// described.
func TestImageDrawsSkipsARedrawWhenNothingChanged(t *testing.T) {
	m := newImageTestModel(t, Options{Images: true})
	press(m, "1")
	m.files.selectPath("Pic.md")
	press(m, "enter")
	m.Update(uv.PrimaryDeviceAttributesEvent{1, 2, 4})
	m.Update(uv.CellSizeEvent{Width: 8, Height: 16})

	rel := "Assets/photo.png"
	abs := m.vault.Abs(rel)
	fi, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	m.imgCache[abs] = sixelImage{mtime: fi.ModTime().UnixNano(), size: fi.Size(), cellsW: m.layout().noteTextW() * m.cellW, data: "SIXELDATA"}

	if len(rawStrings(flattenCmd(m.imageDraws()))) == 0 {
		t.Fatal("expected the first draw once the image is cached")
	}
	if len(m.painted) == 0 {
		t.Fatal("expected imageDraws to record what it painted")
	}

	// Nothing about the note, scroll position or cache changed, so a
	// second call must be a pure no-op: no clear, no redraw.
	if cmd := m.imageDraws(); cmd != nil {
		t.Errorf("expected nil (no terminal writes) when nothing changed, got %v", flattenCmd(cmd))
	}
}
