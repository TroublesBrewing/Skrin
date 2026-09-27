package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/clipimg"
)

// fakeClip is a clipboard that hands back whatever the test put in it.
// The real one (internal/clipimg) runs wl-paste or xclip; no test here
// ever does.
type fakeClip struct {
	img  clipimg.Image
	err  error
	asks int
}

func (f *fakeClip) Image() (clipimg.Image, error) {
	f.asks++
	return f.img, f.err
}

// png is a byte string standing in for an image. Nothing in the paste
// path decodes it: the preview is built later, in the background, from
// the file on disk.
const pngBytes = "\x89PNG\r\n\x1a\nfake bytes"

func withClip(t *testing.T, clip clipimg.Reader) *Model {
	t.Helper()
	return newTestModelWith(t, Options{RolloverTodos: true, ClipImage: clip})
}

// Pasting an image is the same Ctrl+V as pasting text: the clipboard is
// asked for an image first, because an image can't come over OSC 52 at
// all, and a clipboard with no image in it falls through to the text
// path untouched.

func TestCtrlVSavesAClipboardImageAndEmbedsIt(t *testing.T) {
	clip := &fakeClip{img: clipimg.Image{Data: []byte(pngBytes), Mime: "image/png", Ext: ".png"}}
	m := withClip(t, clip)
	press(m, "G", "enter") // Welcome.md, straight into the editor
	if m.editor == nil {
		t.Fatal("setup: the editor should be open")
	}
	_, cmd := m.Update(key("ctrl+v"))
	if cmd == nil {
		t.Fatal("ctrl+v should go and ask the clipboard")
	}
	m.Update(cmd())
	if clip.asks != 1 {
		t.Errorf("clipboard asked %d times", clip.asks)
	}

	// The file is in the vault root, Obsidian's own default, named the
	// way Obsidian names a pasted image.
	want := "Pasted image 20260915093000.png"
	body, err := os.ReadFile(filepath.Join(m.vault.Root, want))
	if err != nil {
		t.Fatalf("the image should be saved at the vault root: %v", err)
	}
	if string(body) != pngBytes {
		t.Errorf("saved bytes = %q", body)
	}

	// And the note has Obsidian's embed, alone on its line so that Skrin
	// draws it as a picture.
	if got := m.editor.Text(); !strings.Contains(got, "![["+want+"]]") {
		t.Errorf("the note should embed the image:\n%s", got)
	}
	for _, line := range strings.Split(m.editor.Text(), "\n") {
		if strings.Contains(line, want) && strings.TrimSpace(line) != "![["+want+"]]" {
			t.Errorf("the embed shares its line with text, so it won't render: %q", line)
		}
	}
	if !strings.Contains(m.flash, want) || !strings.Contains(m.flash, "U the file") {
		t.Errorf("flash = %q: it should name the file and the way back", m.flash)
	}
}

func TestUndoTakesAPastedImageBackOutOfTheVault(t *testing.T) {
	clip := &fakeClip{img: clipimg.Image{Data: []byte(pngBytes), Ext: ".png"}}
	m := withClip(t, clip)
	press(m, "G", "enter")
	_, cmd := m.Update(key("ctrl+v"))
	m.Update(cmd())
	rel := "Pasted image 20260915093000.png"
	if !m.vault.Exists(rel) {
		t.Fatal("setup: the image should be in the vault")
	}
	press(m, "esc") // leave the editor; U is a Files key
	press(m, "U")
	if m.vault.Exists(rel) {
		t.Error("U should take the pasted image back out of the vault")
	}
}

func TestASecondPasteInTheSameSecondDoesNotOverwriteTheFirst(t *testing.T) {
	clip := &fakeClip{img: clipimg.Image{Data: []byte(pngBytes), Ext: ".png"}}
	m := withClip(t, clip)
	press(m, "G", "enter")
	for i := 0; i < 2; i++ {
		_, cmd := m.Update(key("ctrl+v"))
		m.Update(cmd())
	}
	// The clock is pinned, so both names would be the same second: the
	// second paste has to find a free name rather than land on the first.
	for _, rel := range []string{"Pasted image 20260915093000.png", "Pasted image 20260915093000-2.png"} {
		if !m.vault.Exists(rel) {
			t.Errorf("%s should exist after two pastes in the same second", rel)
		}
	}
}

func TestAPastedImageFollowsObsidiansAttachmentFolder(t *testing.T) {
	clip := &fakeClip{img: clipimg.Image{Data: []byte(pngBytes), Ext: ".png"}}
	m := withClip(t, clip)
	// Obsidian's own setting, and the folder really is there.
	writeFile(t, m.vault.Root, ".obsidian/app.json", `{"attachmentFolderPath":"Assets"}`)
	if err := os.Mkdir(filepath.Join(m.vault.Root, "Assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	press(m, "G", "enter")
	_, cmd := m.Update(key("ctrl+v"))
	m.Update(cmd())
	rel := "Assets/Pasted image 20260915093000.png"
	if !m.vault.Exists(rel) {
		t.Errorf("the image should land in %s", rel)
	}
	if got := m.editor.Text(); !strings.Contains(got, "![[Pasted image 20260915093000.png]]") {
		t.Errorf("the embed should name the image the short way:\n%s", got)
	}
}

func TestAnAttachmentFolderThatIsNotThereIsNotConjuredUp(t *testing.T) {
	clip := &fakeClip{img: clipimg.Image{Data: []byte(pngBytes), Ext: ".png"}}
	m := withClip(t, clip)
	writeFile(t, m.vault.Root, ".obsidian/app.json", `{"attachmentFolderPath":"Bilder"}`)
	press(m, "G", "enter")
	_, cmd := m.Update(key("ctrl+v"))
	m.Update(cmd())
	// The same ruling as a new note's declared folder: the folder isn't
	// made up, the file goes to the root, and the flash says so.
	if m.vault.Exists("Bilder") {
		t.Error("a folder nobody made shouldn't appear from a paste")
	}
	if !m.vault.Exists("Pasted image 20260915093000.png") {
		t.Error("the image should fall back to the vault root")
	}
	if !strings.Contains(m.flash, "Bilder doesn't exist") {
		t.Errorf("flash = %q: it should say where the image went instead", m.flash)
	}
}

func TestAClipboardWithNoImageStillPastesText(t *testing.T) {
	clip := &fakeClip{err: clipimg.ErrNoImage}
	m := withClip(t, clip)
	press(m, "G", "enter")
	_, cmd := m.Update(key("ctrl+v"))
	m.Update(cmd()) // no image: on to the terminal's clipboard
	m.Update(tea.ClipboardMsg{Content: "from elsewhere", Selection: 'c'})
	if got := m.editor.Text(); !strings.Contains(got, "from elsewhere") {
		t.Errorf("text should still paste as before:\n%s", got)
	}
	if got := len(filesIn(t, m.vault.Root)); got != 0 {
		t.Errorf("a text paste wrote %d files into the vault root", got)
	}
}

func TestWithNoClipboardToolAnEmptyClipboardSaysWhy(t *testing.T) {
	clip := &fakeClip{err: clipimg.ErrNoTool}
	m := withClip(t, clip)
	press(m, "G", "enter")
	_, cmd := m.Update(key("ctrl+v"))
	m.Update(cmd())
	m.Update(tea.ClipboardMsg{Selection: 'c'}) // the terminal has nothing either
	if !strings.Contains(m.flash, "The clipboard is empty") || !strings.Contains(m.flash, "wl-clipboard") {
		t.Errorf("flash = %q: an image it can't reach shouldn't look like an empty clipboard", m.flash)
	}
}

func TestTheBookCardKeepsCtrlVForItsOwnFields(t *testing.T) {
	clip := &fakeClip{img: clipimg.Image{Data: []byte(pngBytes), Ext: ".png"}}
	m := withClip(t, clip)
	press(m, "B")
	if m.book == nil {
		t.Fatal("setup: B should open the Book Card")
	}
	m.Update(key("ctrl+v"))
	if clip.asks != 0 {
		t.Error("a form field that takes text shouldn't be handed an image")
	}
}

// writeFile puts one file in the vault under test, replacing what's
// there, for the settings a test wants Obsidian to have.
func writeFile(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// filesIn lists the plain files directly in dir, for asserting that
// nothing was written where nothing should be.
func filesIn(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && !strings.HasSuffix(e.Name(), ".md") && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, e.Name())
		}
	}
	return out
}
