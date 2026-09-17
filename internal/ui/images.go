package ui

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/sixel"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"github.com/lurioso/skrin/internal/imgmeta"
	"github.com/lurioso/skrin/internal/index"
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

// embedOptions is the markdown.EmbedOptions for note from: Content
// resolves a ![[Note]] block embed's target the same way a wikilink does
// and hands back its raw source, sliced to just one heading's section
// when the target names one.
func (m *Model) embedOptions(from string) markdown.EmbedOptions {
	return markdown.EmbedOptions{Content: m.embedContent(from)}
}

// embedContent resolves a note embed's target, as written after "![[",
// to the source text it transcludes. A "Note#Heading" target slices out
// just that heading's section; a bare "Note" transcludes the whole note.
// Anything it can't turn into real content — an unresolved note, or a
// heading it can't find — reports false, so the caller falls back to a
// plain link.
func (m *Model) embedContent(from string) func(target string) (string, bool) {
	return func(target string) (string, bool) {
		note, sub, _ := strings.Cut(target, "#")
		rel, ok := m.idx.Resolve(note, from)
		if !ok {
			return "", false
		}
		src, err := m.vault.Read(rel)
		if err != nil {
			return "", false
		}
		if sub == "" {
			return src, true
		}
		line, ok := m.idx.Anchor(rel, sub)
		if !ok {
			return "", false
		}
		return headingSection(src, m.idx.Headings(rel), line), true
	}
}

// headingSection slices src down to one heading's section: from the
// heading at line to just before the next heading of the same level or
// higher, or the end of the note — the same end-of-section rule
// internal/habit.BlockRange uses for its one fixed heading, generalised
// to any heading.
func headingSection(src string, heads []index.Heading, line int) string {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	if n := len(lines); n > 1 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	if line >= len(lines) {
		return ""
	}
	end, level := len(lines), 6
	for i, h := range heads {
		if h.Line != line {
			continue
		}
		level = h.Level
		for _, next := range heads[i+1:] {
			if next.Level <= level {
				end = next.Line
				break
			}
		}
		break
	}
	return strings.Join(lines[line:min(end, len(lines))], "\n")
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

// paintedRect is one screen rectangle imageDraws painted with raw sixel
// pixels — outside Bubble Tea's own cell diffing, so nothing else ever
// erases it once drawn. imageDraws only clears and repaints when what it
// wants painted this frame actually differs from m.painted; abs and data
// are part of that comparison; painting again with identical bytes at the
// identical spot is worse than a no-op — every clear-then-redraw is a
// visible flicker on a terminal that has to rasterize sixel data, so
// doing it every single Update (a keystroke, a watcher tick, an
// unrelated drawer event) even when nothing changed is what produced the
// reported blinking as much as any stale pixel ever did.
type paintedRect struct {
	col, row, w, h int
	abs, data      string
}

// clear is the tea.Raw payload that overwrites r with plain spaces.
func (r paintedRect) clear() string {
	var b strings.Builder
	blank := strings.Repeat(" ", r.w)
	for i := 0; i < r.h; i++ {
		b.WriteString(ansi.SetCursorPosition(r.col, r.row+i))
		b.WriteString(blank)
	}
	return b.String()
}

// draw is the tea.Raw payload that paints r's sixel data at its spot.
func (r paintedRect) draw() string {
	return ansi.SetCursorPosition(r.col, r.row) + r.data
}

// imagePixelsMsg carries one image's freshly decoded, scaled and
// sixel-encoded payload back from the background goroutine that built
// it, so a cold cache miss — a cover never seen before — never blocks a
// keypress waiting on file IO, image decode and sixel encoding. stated
// is false when the file couldn't even be stat'd (nothing to cache, try
// again next time); otherwise data is "" when the image was found but
// couldn't be decoded or encoded, cached as a permanent miss so a
// genuinely broken file isn't re-attempted on every frame.
type imagePixelsMsg struct {
	abs              string
	mtime, size      int64
	targetW, targetH int
	data             string
	stated           bool
}

// imageDraws is the tea.Cmd that repaints the note pane's sixel pixels to
// match what settle() just laid out. It first works out what should be
// painted this frame; if that's identical to what's already on screen
// (m.painted), it does nothing at all — no clear, no redraw, no terminal
// writes. Only when the set of visible images, their positions, sizes or
// bytes actually changed does it clear every rect it painted last frame
// and draw the new set, as one tea.Sequence so the clears land before
// the draws — tea.Batch makes no ordering promise, and racing a draw
// ahead of the clear it depends on produced real ghosting during
// development of this fix. A cold cache miss is handed to a background
// goroutine that decodes and encodes it and reports back with
// imagePixelsMsg; only one encode is ever in flight per file at a time.
func (m *Model) imageDraws() tea.Cmd {
	if !m.opts.Images || !m.sixel || m.cellW <= 0 || m.cellH <= 0 {
		return m.clearPainted()
	}
	row, col, ok := m.noteOrigin()
	if !ok {
		return m.clearPainted()
	}
	vis := m.layout().bodyH - 2
	var want []paintedRect
	var encodes []tea.Cmd
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
		abs := m.vault.Abs(rel)
		targetW, targetH := cellsWide*m.cellW, ln.Image.Rows*m.cellH
		drawRow := row + (i - m.noteOff)
		if data, ok := m.sixelCached(abs, targetW, targetH); ok {
			if data != "" {
				want = append(want, paintedRect{col: col, row: drawRow, w: cellsWide, h: ln.Image.Rows, abs: abs, data: data})
			}
			continue // a known-bad cache entry: don't retry every frame
		}
		if m.pendingEncode == nil {
			m.pendingEncode = map[string]bool{}
		}
		if m.pendingEncode[abs] {
			continue // already decoding this file in another goroutine
		}
		m.pendingEncode[abs] = true
		encodes = append(encodes, func() tea.Msg { return encodeSixel(abs, targetW, targetH) })
	}
	var cmds []tea.Cmd
	if !slices.Equal(m.painted, want) {
		for _, r := range m.painted {
			cmds = append(cmds, tea.Raw(r.clear()))
		}
		for _, r := range want {
			cmds = append(cmds, tea.Raw(r.draw()))
		}
		m.painted = want
	}
	seq := tea.Sequence(cmds...)
	if len(encodes) == 0 {
		return seq
	}
	return tea.Batch(append(encodes, seq)...)
}

// clearPainted erases every rect imageDraws painted last frame and forgets
// it, for whenever there's nothing to draw this frame at all (images off,
// sixel unknown, or noteOrigin refuses — the note closed, an overlay
// opened over it, or there's no room).
func (m *Model) clearPainted() tea.Cmd {
	if len(m.painted) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, len(m.painted))
	for i, r := range m.painted {
		cmds[i] = tea.Raw(r.clear())
	}
	m.painted = nil
	return tea.Sequence(cmds...)
}

// sixelCached is the encoded sixel payload for the image at abs already
// in cache for targetW×targetH pixels — a plain map lookup, cheap enough
// to run inline on every keystroke. It never touches disk; a cold miss is
// handled by encodeSixel, off the main goroutine.
func (m *Model) sixelCached(abs string, targetW, targetH int) (string, bool) {
	fi, err := os.Stat(abs)
	if err != nil {
		return "", false
	}
	c, ok := m.imgCache[abs]
	if !ok || c.mtime != fi.ModTime().UnixNano() || c.size != fi.Size() || c.cellsW != targetW {
		return "", false
	}
	return c.data, true
}

// encodeSixel does the slow part — decode, scale and sixel-encode the
// image at abs to targetW×targetH pixels — and is meant to run inside a
// tea.Cmd's own goroutine, never inline in Update. It touches no Model
// state, only the filesystem, so it's safe to run concurrently with
// anything else. Once the file's been stat'd, every failure still
// reports mtime/size (stated: true) so Update caches it as a permanent
// miss rather than retrying a broken file every frame; a failure before
// that point (a bad target size, or the stat itself failing) reports
// stated: false, so nothing is cached and it's free to try again once
// the file might actually be there.
func encodeSixel(abs string, targetW, targetH int) imagePixelsMsg {
	if targetW <= 0 || targetH <= 0 {
		return imagePixelsMsg{abs: abs}
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return imagePixelsMsg{abs: abs}
	}
	bad := imagePixelsMsg{abs: abs, mtime: fi.ModTime().UnixNano(), size: fi.Size(), targetW: targetW, targetH: targetH, stated: true}
	f, err := os.Open(abs)
	if err != nil {
		return bad
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return bad
	}
	scaled := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), img, img.Bounds(), draw.Over, nil)
	var buf bytes.Buffer
	if err := (&sixel.Encoder{}).Encode(&buf, scaled); err != nil {
		return bad
	}
	return imagePixelsMsg{
		abs: abs, mtime: fi.ModTime().UnixNano(), size: fi.Size(),
		targetW: targetW, targetH: targetH, stated: true,
		data: ansi.SixelGraphics(0, 0, 0, buf.Bytes()),
	}
}
