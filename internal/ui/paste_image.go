package ui

import (
	"errors"
	"fmt"
	"path"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/clipimg"
	"github.com/lurioso/skrin/internal/obsidian"
	"github.com/lurioso/skrin/internal/vault"
)

// clipImageMsg carries what the OS clipboard answered when Ctrl+V asked
// it for an image. err is clipimg.ErrNoImage for the ordinary case of a
// clipboard holding text, and clipimg.ErrNoTool when this machine has no
// program that could tell us either way.
type clipImageMsg struct {
	img clipimg.Image
	err error
}

// startPaste is Ctrl+V. With a note open for editing it asks the OS
// clipboard for an image first, because an image can't reach Skrin any
// other way: the terminal's own clipboard (OSC 52, the text path in
// paste.go) carries text and nothing else. A clipboard with no image in
// it, and every surface that isn't the editor, goes straight on to that
// text path, unchanged.
func (m *Model) startPaste() tea.Cmd {
	if m.opts.ClipImage != nil && m.editorTakesPaste() {
		return m.askImagePaste()
	}
	return m.askPaste()
}

// editorTakesPaste reports whether the editor is the surface a paste
// would land in. It mirrors the order of the switch in paste(): every
// field, form and panel that takes text ahead of the editor takes an
// image paste away from it too, since pasting a screenshot into the
// Book Card's Title is not a thing to guess at.
func (m *Model) editorTakesPaste() bool {
	return m.editor != nil &&
		m.conflict == nil && m.noteFind == nil && m.table == nil &&
		m.focus != paneClaude && m.book == nil && m.manual == nil &&
		m.prompt == nil && m.chooser == nil && m.quickNote == nil && m.search == nil
}

// askImagePaste reads the clipboard in the background: it runs a small
// desktop program (see internal/clipimg), which is file IO and a process
// spawn, and neither belongs on the keypress.
func (m *Model) askImagePaste() tea.Cmd {
	m.pastingImage = true
	clip := m.opts.ClipImage
	return func() tea.Msg {
		img, err := clip.Image()
		return clipImageMsg{img: img, err: err}
	}
}

// clipImage handles the clipboard's answer: an image is saved into the
// vault and embedded, and anything else falls through to the text path
// as though Ctrl+V had gone there directly.
func (m *Model) clipImage(msg clipImageMsg) tea.Cmd {
	if !m.pastingImage {
		return nil
	}
	m.pastingImage = false
	if msg.err != nil {
		// Remembered so that an empty text clipboard can own up to why
		// an image paste didn't happen either, rather than leaving the
		// user to guess that a tool is missing.
		m.noImageTool = errors.Is(msg.err, clipimg.ErrNoTool)
		return m.askPaste()
	}
	m.pasteImage(msg.img)
	return nil
}

// pasteImage saves a clipboard image into the vault and puts its embed in
// the note being edited. The file goes where Obsidian's own
// attachmentFolderPath says, which is the vault root unless the vault
// says otherwise, and the line is the embed Obsidian writes too —
// ![[Pasted image ….png]] — so the note reads the same in both programs
// and Skrin draws the picture in half-blocks from the same code that
// draws a book cover.
func (m *Model) pasteImage(img clipimg.Image) {
	if !m.editorTakesPaste() {
		// The editor closed while the clipboard was being read.
		m.flash = "Select a note to paste an image into"
		return
	}
	dir, note := m.attachmentDir()
	rel := m.freeAttachmentName(dir, img.Ext)
	if _, err := m.vault.CreateFile(rel, string(img.Data)); err != nil {
		m.flash = "Couldn't save the pasted image: " + err.Error()
		return
	}
	m.journal.Record(vault.Op{Desc: "paste " + rel, Steps: []vault.Step{{Kind: vault.StepCreated, Rel: rel}}})
	embed := "![[" + m.idx.LinkText(rel, m.edit.rel, m.edit.linkFormat) + "]]"
	m.editor.InsertBlock([]string{embed}, 0, len(embed))
	m.updateCompletion()
	m.refresh()
	m.flash = "Image saved as " + rel + note + " · ctrl+z takes the line out, U the file"
}

// attachmentDir is the folder a pasted image belongs in, plus what to say
// about it. A folder the vault's settings name but nobody has made isn't
// conjured up — the same ruling as a new note's declared folder (v0.47.0):
// the file lands in the root instead, and the flash says so out loud.
func (m *Model) attachmentDir() (dir, note string) {
	dir = obsidian.LoadSettings(m.vault.Root).AttachmentDir(m.edit.rel)
	if dir != "" && !m.vault.IsDir(dir) {
		return "", " · " + dir + " doesn't exist, so it went to the vault root"
	}
	return dir, ""
}

// freeAttachmentName is Obsidian's own name for a pasted image — "Pasted
// image 20260927231455.png", the clock to the second — with a counter
// added if that second already has an image in it, so a paste can never
// land on top of a file that is already there.
func (m *Model) freeAttachmentName(dir, ext string) string {
	base := "Pasted image " + m.opts.Now().Format("20060102150405")
	rel := path.Join(dir, base+ext)
	for i := 2; m.vault.Exists(rel); i++ {
		rel = path.Join(dir, fmt.Sprintf("%s-%d%s", base, i, ext))
	}
	return rel
}
