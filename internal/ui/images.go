package ui

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/sixel"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"github.com/lurioso/skrin/internal/imgmeta"
	"github.com/lurioso/skrin/internal/markdown"
)

// sixelImage is one image's encoded sixel payload, cached until the file
// changes so a note isn't re-decoded on every keystroke.
type sixelImage struct {
	mtime, size int64
	cellsW      int // the pixel width it was encoded for
	data        string
}

// imageMeta is markdown.ImageOptions.Meta for note from: it resolves an
// embed's target the same way a wikilink does, then reads just the image's
// header. This runs whether or not sixel drawing is on — a placeholder
// still shows a found file's real dimensions and size.
func (m *Model) imageMeta(from string) func(target string) (int, int, int64, markdown.ImageStatus) {
	return func(target string) (int, int, int64, markdown.ImageStatus) {
		rel, ok := m.idx.Resolve(target, from)
		if !ok {
			return 0, 0, 0, markdown.ImageMissing
		}
		info, err := imgmeta.Read(m.vault.Abs(rel))
		if err != nil {
			if os.IsNotExist(err) {
				return 0, 0, 0, markdown.ImageMissing
			}
			return 0, 0, 0, markdown.ImageUnsupported
		}
		return info.Width, info.Height, info.Size, markdown.ImageOK
	}
}

// imageOptions is the markdown.ImageOptions for note from: CellW/CellH stay
// 0 (placeholder only) unless both the terminal has answered that it draws
// sixel and the config has images on.
func (m *Model) imageOptions(from string, paneVis int) markdown.ImageOptions {
	opts := markdown.ImageOptions{Meta: m.imageMeta(from), PaneHeight: paneVis}
	if m.opts.Images && m.sixel && m.cellW > 0 && m.cellH > 0 {
		opts.CellW, opts.CellH = m.cellW, m.cellH
	}
	return opts
}

// noteOrigin is the 1-indexed terminal row and column of the note pane's
// text body — where noteBody's row 0 lands — for positioning a sixel draw.
// ok is false when nothing should draw there right now (a modal's up, or
// there's no room).
func (m *Model) noteOrigin() (row, col int, ok bool) {
	if m.width == 0 || m.height == 0 || m.notePath == "" || m.editor != nil {
		return 0, 0, false
	}
	if m.manual != nil || m.book != nil || m.habits != nil || m.quickNote != nil ||
		m.chooser != nil || m.search != nil || len(m.proposals) > 0 {
		return 0, 0, false
	}
	l := m.layout()
	if m.zen {
		margin := max((m.width-l.noteTextW())/2-1, 0)
		return 2, margin + 2, true
	}
	col = 2 // the pane's left border, then noteBody's own leading space
	if l.filesW > 0 {
		col += l.filesW
	}
	if m.split != nil && m.splitLeft {
		col += l.splitW
	}
	return headerHeight + 2, col, true
}

// imageDraws is the tea.Raw commands that paint every image currently
// visible in the note pane with real sixel pixels, positioned by cursor
// move. Called after settle() reflows the note, so it always matches what
// is about to be shown. Returns nil the moment sixel isn't on, or there's
// nothing to draw.
func (m *Model) imageDraws() tea.Cmd {
	if !m.opts.Images || !m.sixel || m.cellW <= 0 || m.cellH <= 0 {
		return nil
	}
	row, col, ok := m.noteOrigin()
	if !ok {
		return nil
	}
	vis := m.layout().bodyH - 2
	var cmds []tea.Cmd
	for i := m.noteOff; i < min(len(m.lines), m.noteOff+vis); i++ {
		ln := m.lines[i]
		if ln.Image == nil || !ln.Image.Pixels || ln.Image.Row != 0 {
			continue
		}
		rel, ok := m.idx.Resolve(ln.Image.Path, m.notePath)
		if !ok {
			continue
		}
		cellsWide := m.layout().noteTextW()
		if m.zen {
			cellsWide = m.layout().noteTextW()
		}
		data, ok := m.sixelFor(m.vault.Abs(rel), cellsWide*m.cellW, ln.Image.Rows*m.cellH)
		if !ok {
			continue
		}
		drawRow := row + (i - m.noteOff)
		cmds = append(cmds, tea.Raw(ansi.SetCursorPosition(col, drawRow)+data))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// sixelFor is the encoded sixel payload for the image at abs, scaled to
// targetW×targetH pixels — cached by the file's mtime and size, so a note
// left open isn't re-decoded on every redraw, only when the file changes.
func (m *Model) sixelFor(abs string, targetW, targetH int) (string, bool) {
	if targetW <= 0 || targetH <= 0 {
		return "", false
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return "", false
	}
	mtime := fi.ModTime().UnixNano()
	if c, ok := m.imgCache[abs]; ok && c.mtime == mtime && c.size == fi.Size() && c.cellsW == targetW {
		return c.data, true
	}
	f, err := os.Open(abs)
	if err != nil {
		return "", false
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return "", false
	}
	scaled := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), img, img.Bounds(), draw.Over, nil)
	var buf bytes.Buffer
	if err := (&sixel.Encoder{}).Encode(&buf, scaled); err != nil {
		return "", false
	}
	data := ansi.SixelGraphics(0, 0, 0, buf.Bytes())
	m.imgCache[abs] = sixelImage{mtime: mtime, size: fi.Size(), cellsW: targetW, data: data}
	return data, true
}
