package ui

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"github.com/lurioso/skrin/internal/imgmeta"
	"github.com/lurioso/skrin/internal/index"
	"github.com/lurioso/skrin/internal/markdown"
)

// previewMaxCols and previewMaxRows bound the block-art thumbnail's box:
// an embed scales down to fit inside it, aspect kept. Picked after
// showing the user actual renders of a real cover at five candidate
// sizes side by side (the vault's "example images" note) — big enough to
// recognise a book cover, small enough that a preview can never balloon
// across the screen the way the sixel path it replaced sometimes did.
const (
	previewMaxCols = 28
	previewMaxRows = 12
)

// imageMeta is markdown.ImageOptions.Meta for note from: it resolves an
// embed's target the same way a wikilink does, then reads just the
// image's header. This runs whether or not thumbnails are on — a
// placeholder still shows a found file's real dimensions and size.
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

// imageOptions is the markdown.ImageOptions for note from: Thumbnail is
// nil (placeholder only) unless the config has images on.
func (m *Model) imageOptions(from string, paneVis int) markdown.ImageOptions {
	opts := markdown.ImageOptions{
		Meta:       m.imageMeta(from),
		PaneHeight: paneVis,
		MaxCols:    previewMaxCols,
		MaxRows:    previewMaxRows,
	}
	if m.opts.Images {
		opts.Thumbnail = m.thumbnail(from)
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

// thumbEntry is one image's block-art preview, cached by the file's
// mtime and size so it isn't rebuilt on every keystroke — only when the
// file actually changes. lines is nil when the file couldn't be decoded;
// that negative result is cached too, so a genuinely broken image isn't
// retried every time its note renders.
type thumbEntry struct {
	mtime, size int64
	lines       []string
}

// thumbRequest is one image the renderer just asked for and didn't have
// cached, queued for Update to turn into a background decode job once
// settle() returns.
type thumbRequest struct {
	abs        string
	cols, rows int
}

// thumbKey identifies one cached (or in-flight) thumbnail: the file plus
// the box it was rendered for, since a split pane can ask for a smaller
// box than the main note pane for the very same file.
func thumbKey(abs string, cols, rows int) string {
	return fmt.Sprintf("%s|%dx%d", abs, cols, rows)
}

// thumbnail is markdown.ImageOptions.Thumbnail for note from: a plain
// cache lookup, cheap enough to run inline on every render. A cache miss
// doesn't decode anything itself — it queues the request in m.wantThumb,
// which Update turns into a background tea.Cmd after settle() returns,
// so a cover never seen before never blocks a keypress on file IO, image
// decode and scaling.
func (m *Model) thumbnail(from string) func(target string, cols, rows int) ([]string, bool) {
	return func(target string, cols, rows int) ([]string, bool) {
		rel, ok := m.idx.Resolve(target, from)
		if !ok {
			return nil, false
		}
		abs := m.vault.Abs(rel)
		fi, err := os.Stat(abs)
		if err != nil {
			return nil, false
		}
		key := thumbKey(abs, cols, rows)
		if e, ok := m.thumbCache[key]; ok && e.mtime == fi.ModTime().UnixNano() && e.size == fi.Size() {
			return e.lines, e.lines != nil
		}
		if !m.pendingThumb[key] {
			if m.pendingThumb == nil {
				m.pendingThumb = map[string]bool{}
			}
			m.pendingThumb[key] = true
			m.wantThumb = append(m.wantThumb, thumbRequest{abs: abs, cols: cols, rows: rows})
		}
		return nil, false
	}
}

// imageThumbMsg carries one image's freshly decoded, scaled and
// half-block-encoded preview back from the background goroutine that
// built it. stated is false when the file couldn't even be stat'd
// (nothing to cache, try again next time); lines is nil when the file
// was found but couldn't be decoded, cached as a permanent miss so a
// genuinely broken file isn't re-attempted on every render.
type imageThumbMsg struct {
	abs         string
	cols, rows  int
	mtime, size int64
	lines       []string
	stated      bool
}

// startThumbnails turns every thumbnail settle() asked for and didn't
// have cached into a background decode job, one tea.Cmd per file+box —
// never inline, so a note full of never-before-seen covers can't block
// a keystroke. Only one job is ever in flight per file+box at a time
// (m.pendingThumb, cleared once the result comes back).
func (m *Model) startThumbnails() tea.Cmd {
	if len(m.wantThumb) == 0 {
		return nil
	}
	reqs := m.wantThumb
	m.wantThumb = nil
	cmds := make([]tea.Cmd, len(reqs))
	for i, req := range reqs {
		req := req
		cmds[i] = func() tea.Msg { return renderThumbnail(req.abs, req.cols, req.rows) }
	}
	return tea.Batch(cmds...)
}

// renderThumbnail does the slow part — decode and scale the image at abs
// to cols×(rows*2) real pixels, then encode it as cols*rows half-block
// cells, foreground the top pixel of each pair and background the
// bottom. It's the same technique internal/logo's chest art uses, just
// with colours sampled from the real image instead of the theme
// palette. It touches no Model state, only the filesystem, so it's safe
// to run inside a tea.Cmd's own goroutine.
func renderThumbnail(abs string, cols, rows int) imageThumbMsg {
	if cols <= 0 || rows <= 0 {
		return imageThumbMsg{abs: abs, cols: cols, rows: rows}
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return imageThumbMsg{abs: abs, cols: cols, rows: rows}
	}
	bad := imageThumbMsg{abs: abs, cols: cols, rows: rows, mtime: fi.ModTime().UnixNano(), size: fi.Size(), stated: true}
	f, err := os.Open(abs)
	if err != nil {
		return bad
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return bad
	}
	scaled := image.NewRGBA(image.Rect(0, 0, cols, rows*2))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), img, img.Bounds(), draw.Over, nil)
	lines := make([]string, rows)
	for row := 0; row < rows; row++ {
		var b strings.Builder
		for x := 0; x < cols; x++ {
			b.WriteString(halfBlockCell(scaled.At(x, 2*row), scaled.At(x, 2*row+1)))
		}
		lines[row] = b.String()
	}
	bad.lines = lines
	return bad
}

// halfBlockCell renders one terminal cell as two stacked image pixels: ▀
// foreground for the top, background for the bottom.
func halfBlockCell(top, bot color.Color) string {
	return lipgloss.NewStyle().Foreground(top).Background(bot).Render("▀")
}
