// Package markdown renders Obsidian-flavoured markdown into themed terminal
// lines.
//
// Rendering is line-oriented, like Obsidian's live preview: every source
// line becomes one or more display lines, and each display line remembers
// its source line. Heading jumps, "open at search hit" and switching to the
// editor without losing your place all rely on that mapping.
package markdown

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"gopkg.in/yaml.v3"

	"github.com/lurioso/skrin/internal/imgmeta"
	"github.com/lurioso/skrin/internal/theme"
)

// Line is one display line of a rendered note.
type Line struct {
	Text    string // styled; at most Options.Width cells wide
	Src     int    // 0-based source line it came from
	Heading int    // 1–6 on the first display line of a heading, else 0
	Links   []Link // links whose text starts, or continues, on this line
	Image   *Image // set on every display line of a rendered image embed
}

// Image marks a display line as (part of) an image embed rendered on a
// line of its own (`![[photo.png]]` or `![alt](Assets/photo.png)`). Every
// row the embed occupies carries an Image with the same Path, Row counting
// up from 0 and Rows fixed — that is what keeps the 1:n Src mapping true
// for a multi-row image the same way it holds for any other block.
type Image struct {
	Path string // vault-relative
	Row  int    // 0-based row of this display line within the image block
	Rows int    // total rows the image occupies (1 for a placeholder frame)
	// Pixels is true when Text is blank filler and the caller (internal/ui)
	// is expected to draw real pixels over this row; false means Text
	// already carries the fully rendered placeholder frame and nothing
	// further needs drawing.
	Pixels        bool
	Width, Height int // the image's real pixel dimensions, when known
}

// ImageOptions controls how image embeds render. The zero value renders
// every embed as the placeholder frame, in every one of its three states
// (found, missing, unsupported) — the always-working default.
type ImageOptions struct {
	// Meta looks up an embed's target file. Nil treats every target as
	// not found, which still renders safely.
	Meta func(target string) (w, h int, size int64, status ImageStatus)
	// CellW, CellH are the terminal's own cell size in pixels. Either
	// being 0 means the terminal hasn't answered (or sixel is off), so
	// every embed renders as the placeholder frame regardless of Meta.
	CellW, CellH int
	// PaneHeight caps how many rows one image may occupy, keeping aspect;
	// 0 means unconstrained.
	PaneHeight int
}

// ImageStatus is what Meta found for an image embed's target.
type ImageStatus int

const (
	ImageOK          ImageStatus = iota // exists, and Skrin can read its header
	ImageMissing                        // no file at that path
	ImageUnsupported                    // a file is there, but not a format Skrin can decode
)

// Link is a link as drawn on a display line.
type Link struct {
	Col    int    // display column where its text starts on the line
	Target string // wikilink target with any #heading ("Note#Morning"), or a markdown link's URL
	Wiki   bool
}

// Options controls rendering.
type Options struct {
	Width   int
	Palette theme.Palette
	// Resolve reports whether a wikilink target (the part before any #)
	// exists; unresolved links are drawn dimmed. Nil treats all as resolved.
	Resolve func(target string) bool
	// Images controls how image embeds render; see ImageOptions.
	Images ImageOptions
}

// Render renders src into display lines.
func Render(src string, o Options) []Line {
	r := &renderer{width: max(o.Width, 10), pal: o.Palette, resolve: o.Resolve, images: o.Images, st: newStyles(o.Palette)}
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	if n := len(lines); n > 1 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	for i := range lines {
		lines[i] = strings.ReplaceAll(lines[i], "\t", "    ")
	}
	start := 0
	if end := frontmatterEnd(lines); end > 0 {
		r.properties(lines[1:end], end)
		start = end + 1
	}
	fence := "" // opening fence while inside a code block
	for i := start; i < len(lines); i++ {
		l := lines[i]
		switch {
		case fence != "":
			if isFenceClose(l, fence) {
				fence = ""
				r.emit(i, "", "", []span{plain(l, r.st.muted)}, 0, clip)
			} else {
				// Code keeps its layout (ASCII art, indentation), so long
				// lines are clipped rather than wrapped, as in Obsidian.
				r.emit(i, r.st.rule.Render("│ "), "", []span{plain(l, r.st.code)}, 0, clip)
			}
		case fenceMarker(l) != "":
			fence = fenceMarker(l)
			r.callout = nil
			r.emit(i, "", "", []span{plain(l, r.st.muted)}, 0, clip)
		case isTableRow(l):
			j := i + 1
			for j < len(lines) && isTableRow(lines[j]) {
				j++
			}
			r.callout = nil
			r.table(lines[i:j], i)
			i = j - 1
		default:
			r.block(i, l)
		}
	}
	return r.out
}

type wrapMode int

const (
	wrapWords wrapMode = iota // break between words
	clip                      // one line, cut at the edge with "…" (code, rules)
)

type span struct {
	text  string
	style lipgloss.Style
	link  int // index into renderer.links, or -1
}

func plain(text string, style lipgloss.Style) span { return span{text, style, -1} }

type styles struct {
	text, muted, rule, quote, code, bullet, done, doneText, propKey lipgloss.Style
	heading                                                         [6]lipgloss.Style
}

func newStyles(p theme.Palette) styles {
	s := lipgloss.NewStyle
	st := styles{
		text:     s().Foreground(p.Foreground),
		muted:    s().Foreground(p.DarkForeground),
		rule:     s().Foreground(p.Border),
		quote:    s().Foreground(p.LightForeground).Italic(true),
		code:     s().Foreground(p.Code),
		bullet:   s().Foreground(p.Accent),
		done:     s().Foreground(p.Green),
		doneText: s().Foreground(p.DarkForeground).Strikethrough(true),
		propKey:  s().Foreground(p.DarkForeground),
	}
	for i, c := range p.Headings {
		st.heading[i] = s().Foreground(c).Bold(true)
	}
	st.heading[0] = st.heading[0].Underline(true)
	return st
}

type renderer struct {
	width   int
	pal     theme.Palette
	resolve func(string) bool
	images  ImageOptions
	st      styles
	callout *lipgloss.Style // colour of the callout we're inside, if any
	links   []Link          // every link seen, referenced by span.link
	out     []Line
}

var (
	// The closing hashes of "# Title ##" are markup, not text: Obsidian
	// hides them and so does internal/index, so the renderer must too.
	headingRE  = regexp.MustCompile(`^ {0,3}(#{1,6})[ \t]+(.*?)(?:[ \t]+#+)?[ \t]*$`)
	listRE     = regexp.MustCompile(`^(\s*)([-*+]|\d{1,9}[.)])[ \t]+(\[(.)\](?:[ \t]+|$))?(.*)$`)
	calloutRE  = regexp.MustCompile(`^\[!([A-Za-z]+)\][+-]?\s*(.*)$`)
	tableSepRE = regexp.MustCompile(`^\|?[\s:|-]+\|?$`)
	// An image embed alone on its own line: ![[photo.png]] (any alias or
	// size hint after a "|" is ignored) or ![alt](Assets/photo.png).
	// Mixed with other text on the line, an embed stays an inline link,
	// same as any other embed — this only catches the block form, which
	// is how a pasted screenshot actually looks in a note.
	imageEmbedRE = regexp.MustCompile(`^!\[\[([^\[\]|]+)(?:\|[^\[\]]*)?\]\]$`)
	imageLinkRE  = regexp.MustCompile(`^!\[([^\[\]]*)\]\(([^()\s]+)\)$`)
	inlineRE     = regexp.MustCompile(strings.Join([]string{
		"`[^`]+`",                    // code
		`!?\[\[[^\[\]]+\]\]`,         // wikilink or embed
		`\[[^\[\]]+\]\([^()\s]+\)`,   // markdown link
		`\*\*[^*]+\*\*`, `__[^_]+__`, // bold
		`~~[^~]+~~`,                           // strikethrough
		`==[^=]+==`,                           // highlight
		`\*[^*\s](?:[^*]*[^*\s])?\*`,          // italic
		`(?:^|[\s(])_[^_\s](?:[^_]*[^_\s])?_`, // italic
		`(?:^|[\s(])#[\p{L}\p{N}_/-]*[\p{L}_/-][\p{L}\p{N}_/-]*`, // tag
	}, "|"))
)

func (r *renderer) block(i int, l string) {
	t := strings.TrimSpace(l)
	isQuote := strings.HasPrefix(strings.TrimLeft(l, " "), ">")
	if !isQuote {
		r.callout = nil
	}
	switch {
	case t == "":
		r.emit(i, "", "", nil, 0, wrapWords)
	case headingRE.MatchString(l):
		m := headingRE.FindStringSubmatch(l)
		level := len(m[1])
		r.emit(i, "", "", r.inline(strings.TrimSpace(m[2]), r.st.heading[level-1]), level, wrapWords)
	case isRule(t):
		r.emit(i, "", "", []span{plain(strings.Repeat("─", r.width), r.st.rule)}, 0, clip)
	case isQuote:
		r.quote(i, l)
	case listRE.MatchString(l):
		r.listItem(i, listRE.FindStringSubmatch(l))
	default:
		if target, alt, ok := imageEmbed(t); ok {
			r.image(i, target, alt)
			return
		}
		r.emit(i, "", "", r.inline(l, r.st.text), 0, wrapWords)
	}
}

// imageEmbed reports whether t (already trimmed) is exactly one image
// embed and nothing else, and if so its target path and alt text.
func imageEmbed(t string) (target, alt string, ok bool) {
	if m := imageEmbedRE.FindStringSubmatch(t); m != nil && imgmeta.IsImage(m[1]) {
		return m[1], "", true
	}
	if m := imageLinkRE.FindStringSubmatch(t); m != nil && imgmeta.IsImage(m[2]) {
		return m[2], m[1], true
	}
	return "", "", false
}

func (r *renderer) quote(i int, l string) {
	body := strings.TrimPrefix(strings.TrimPrefix(strings.TrimLeft(l, " "), ">"), " ")
	if m := calloutRE.FindStringSubmatch(body); m != nil {
		kind := strings.ToLower(m[1])
		c, icon := r.calloutLook(kind)
		st := lipgloss.NewStyle().Foreground(c)
		r.callout = &st
		title := strings.TrimSpace(m[2])
		if title == "" {
			title = strings.ToUpper(kind[:1]) + kind[1:]
		}
		bar := st.Render("▌ ")
		spans := append([]span{plain(icon+" ", st.Bold(true))}, r.inline(title, st.Bold(true))...)
		r.emit(i, bar, bar, spans, 0, wrapWords)
		return
	}
	bar, base := r.st.rule.Render("▌ "), r.st.quote
	if r.callout != nil {
		bar, base = r.callout.Render("▌ "), r.st.text
	}
	r.emit(i, bar, bar, r.inline(body, base), 0, wrapWords)
}

// calloutLook picks the colour and icon for an Obsidian callout type.
func (r *renderer) calloutLook(kind string) (color.Color, string) {
	p := r.pal
	switch kind {
	case "abstract", "summary", "tldr", "tip", "hint", "important":
		return p.Cyan, "✦"
	case "success", "check", "done":
		return p.Green, "✔"
	case "question", "help", "faq":
		return p.Yellow, "?"
	case "warning", "caution", "attention":
		return p.Orange, "!"
	case "failure", "fail", "missing", "danger", "error", "bug":
		return p.Red, "✘"
	case "example":
		return p.Magenta, "☰"
	case "quote", "cite":
		return p.DarkForeground, "❝"
	}
	return p.Blue, "ℹ" // note, info, todo and unknown types
}

func (r *renderer) listItem(i int, m []string) {
	indent, marker, base := m[1], "", r.st.text
	switch {
	case m[3] != "":
		switch m[4] {
		case "x", "X":
			marker, base = r.st.done.Render("☑"), r.st.doneText
		case " ":
			marker = r.st.bullet.Render("☐")
		default:
			marker = r.st.bullet.Render("[" + m[4] + "]")
		}
	case strings.ContainsAny(m[2], ".)"):
		marker = r.st.bullet.Render(m[2])
	default:
		marker = r.st.bullet.Render("•")
	}
	prefix := indent + marker + " "
	r.emit(i, prefix, strings.Repeat(" ", ansi.StringWidth(prefix)), r.inline(m[5], base), 0, wrapWords)
}

// image renders an image embed on a line of its own: real sixel pixels
// when the terminal has answered that it can show them and the file's
// dimensions are known, a placeholder frame otherwise. Either way the
// embed stays a link (f/Enter still work on it).
func (r *renderer) image(src int, target, alt string) {
	id := r.addLink(target, true)
	var w, h int
	var size int64
	status := ImageMissing
	if r.images.Meta != nil {
		w, h, size, status = r.images.Meta(target)
	}
	name := target
	if slash := strings.LastIndexByte(name, '/'); slash >= 0 {
		name = name[slash+1:]
	}
	canDrawPixels := status == ImageOK && r.images.CellW > 0 && r.images.CellH > 0 && w > 0 && h > 0
	if !canDrawPixels {
		r.emitImagePlaceholder(src, id, target, name, w, h, size, status)
		return
	}
	r.emitImagePixels(src, target, w, h)
}

// emitImagePlaceholder is the always-working default: one line naming the
// file, its real dimensions and size when known, or the reason it can't
// show those. Reuses emit()'s own wrap/link-column logic in clip mode,
// which is guaranteed to produce exactly one display line.
func (r *renderer) emitImagePlaceholder(src, id int, target, name string, w, h int, size int64, status ImageStatus) {
	frame := "▗▖  " + name
	style := r.st.text.Foreground(r.pal.Link).Underline(true)
	switch status {
	case ImageOK:
		frame += fmt.Sprintf(" · %d×%d · %s", w, h, humanSize(size))
	case ImageMissing:
		frame += " — not found"
		style = r.st.muted.Underline(true)
	case ImageUnsupported:
		frame += " — unsupported image format"
		style = r.st.muted.Underline(true)
	}
	before := len(r.out)
	r.emit(src, "", "", []span{{frame, style, id}}, 0, clip)
	for i := before; i < len(r.out); i++ {
		r.out[i].Image = &Image{Path: target, Row: i - before, Rows: len(r.out) - before}
	}
}

// emitImagePixels reserves rows for a real sixel image, scaled to the pane's
// width and, when that would make it too tall, to the pane's height instead
// — aspect kept either way. internal/ui does the actual decode/scale/draw;
// here we only need to agree with it on how many rows the image takes, so
// the 1:n Src mapping holds for every one of them.
func (r *renderer) emitImagePixels(src int, path string, w, h int) {
	rows := imageRows(w, h, r.width, r.images.CellW, r.images.CellH, r.images.PaneHeight)
	for row := range rows {
		r.out = append(r.out, Line{
			Text:  strings.Repeat(" ", r.width),
			Src:   src,
			Links: []Link{{Col: 0, Target: path, Wiki: true}},
			Image: &Image{Path: path, Row: row, Rows: rows, Pixels: true, Width: w, Height: h},
		})
	}
}

// imageRows is how many terminal rows an image of w×h real pixels occupies
// once scaled to fit cellsWide character columns, each cellW×cellH pixels —
// capped to maxRows (keeping aspect) when set.
func imageRows(w, h, cellsWide, cellW, cellH, maxRows int) int {
	if w <= 0 || h <= 0 || cellW <= 0 || cellH <= 0 || cellsWide <= 0 {
		return 1
	}
	paneWidthPx := cellsWide * cellW
	scaledH := float64(h) * float64(paneWidthPx) / float64(w)
	rows := (int(scaledH) + cellH - 1) / cellH // round up
	if rows < 1 {
		rows = 1
	}
	if maxRows > 0 && rows > maxRows {
		rows = maxRows
	}
	return rows
}

// humanSize formats a byte count the way a reader thinks about file sizes:
// "1.2 MB", not "1258291 bytes".
func humanSize(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.0f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	}
}

func isTableRow(l string) bool { return strings.HasPrefix(strings.TrimSpace(l), "|") }

func isTableSep(l string) bool {
	t := strings.TrimSpace(l)
	return tableSepRE.MatchString(t) && strings.Contains(t, "-")
}

// splitCells splits a pipe-table row. Obsidian writes a pipe inside a cell
// as \| (as in [[Note\|alias]]), which stays inside the cell.
func splitCells(l string) []string {
	t := strings.TrimSpace(strings.ReplaceAll(l, `\|`, "\x00"))
	t = strings.TrimSuffix(strings.TrimPrefix(t, "|"), "|")
	cells := strings.Split(t, "|")
	for i, c := range cells {
		cells[i] = strings.TrimSpace(strings.ReplaceAll(c, "\x00", "|"))
	}
	return cells
}

// table renders a run of pipe-table rows as aligned columns. When the table
// is wider than the pane, the widest columns shrink and their cells are
// clipped with "…".
func (r *renderer) table(rows []string, start int) {
	cells := make([][][]span, len(rows)) // nil for separator rows
	var widths []int
	for k, l := range rows {
		if isTableSep(l) {
			continue
		}
		base := r.st.text
		if k+1 < len(rows) && isTableSep(rows[k+1]) {
			base = base.Bold(true) // header row
		}
		for c, cell := range splitCells(l) {
			spans := r.inline(cell, base)
			cells[k] = append(cells[k], spans)
			if c == len(widths) {
				widths = append(widths, 0)
			}
			widths[c] = max(widths[c], spansWidth(spans))
		}
	}
	// Each column takes "│ " + cell + " ", plus one closing "│".
	total := func() int {
		n := 1
		for _, w := range widths {
			n += w + 3
		}
		return n
	}
	for total() > r.width {
		widest := 0
		for c, w := range widths {
			if w > widths[widest] {
				widest = c
			}
		}
		if widths[widest] <= 3 {
			break
		}
		widths[widest]--
	}
	bar := r.st.rule.Render("│")
	for k := range rows {
		var b strings.Builder
		var links []Link
		if cells[k] == nil {
			parts := make([]string, len(widths))
			for c, w := range widths {
				parts[c] = strings.Repeat("─", w+2)
			}
			b.WriteString(r.st.rule.Render("├" + strings.Join(parts, "┼") + "┤"))
		} else {
			b.WriteString(bar)
			col := 1
			for c, w := range widths {
				var spans []span
				if c < len(cells[k]) {
					spans = cells[k][c]
				}
				for _, sp := range spans {
					if sp.link >= 0 && col+1 < r.width {
						l := r.links[sp.link]
						l.Col = col + 1
						links = append(links, l)
						break
					}
				}
				b.WriteString(" " + r.fitSpans(spans, w) + " " + bar)
				col += w + 3
			}
		}
		r.out = append(r.out, Line{Text: ansi.Truncate(b.String(), r.width, ""), Src: start + k, Links: links})
	}
}

// fitSpans renders spans in exactly w cells, clipping with "…".
func (r *renderer) fitSpans(spans []span, w int) string {
	lines := wrap(spans, w, clip)
	tail := ""
	if len(lines) > 1 {
		lines = wrap(spans, max(w-1, 1), clip)
		tail = r.st.muted.Render("…")
	}
	return padRight(renderTokens(lines[0], spans)+tail, w)
}

func spansWidth(spans []span) int {
	n := 0
	for _, sp := range spans {
		n += ansi.StringWidth(sp.text)
	}
	return n
}

// properties renders YAML frontmatter as a compact key/value box. body holds
// the lines between the --- fences, starting at source line 1.
func (r *renderer) properties(body []string, closeLine int) {
	var doc yaml.Node
	err := yaml.Unmarshal([]byte(strings.Join(body, "\n")), &doc)
	if err != nil || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		for j, l := range body {
			r.emit(j+1, "", "", []span{plain(l, r.st.muted)}, 0, wrapWords)
		}
	} else {
		pairs := doc.Content[0].Content
		kw := 0
		for k := 0; k+1 < len(pairs); k += 2 {
			kw = max(kw, ansi.StringWidth(pairs[k].Value))
		}
		kw = min(kw, 14)
		for k := 0; k+1 < len(pairs); k += 2 {
			key, val := pairs[k], pairs[k+1]
			prefix := r.st.propKey.Render(padRight(ansi.Truncate(key.Value, kw, "…"), kw)) + "  "
			r.emit(key.Line, prefix, strings.Repeat(" ", kw+2), r.inline(propValue(key.Value, val), r.st.text), 0, wrapWords)
		}
	}
	r.emit(closeLine, "", "", []span{plain(strings.Repeat("─", r.width), r.st.rule)}, 0, clip)
}

func propValue(key string, n *yaml.Node) string {
	var items []string
	switch n.Kind {
	case yaml.ScalarNode:
		items = []string{n.Value}
	case yaml.SequenceNode:
		for _, c := range n.Content {
			items = append(items, c.Value)
		}
	default:
		return "…"
	}
	if k := strings.ToLower(key); k == "tags" || k == "tag" {
		var tags []string
		for _, it := range items {
			for _, t := range strings.FieldsFunc(it, func(r rune) bool { return r == ',' || unicode.IsSpace(r) }) {
				tags = append(tags, "#"+strings.TrimPrefix(t, "#"))
			}
		}
		return strings.Join(tags, " ")
	}
	return strings.Join(items, ", ")
}

// inline styles the spans of one line of text on top of base.
func (r *renderer) inline(s string, base lipgloss.Style) []span {
	var out []span
	last := 0
	for _, loc := range inlineRE.FindAllStringIndex(s, -1) {
		a, b := loc[0], loc[1]
		// Tags and _italics_ match together with the character before them.
		if strings.ContainsRune(" \t\r\f(", rune(s[a])) {
			a++
		}
		if a > last {
			out = append(out, plain(s[last:a], base))
		}
		out = append(out, r.token(s[a:b], base)...)
		last = b
	}
	if last < len(s) {
		out = append(out, plain(s[last:], base))
	}
	return out
}

func (r *renderer) token(tok string, base lipgloss.Style) []span {
	switch {
	case strings.HasPrefix(tok, "`"):
		return []span{plain(strings.Trim(tok, "`"), r.st.code)}
	case strings.HasPrefix(tok, "![[") || strings.HasPrefix(tok, "[["):
		return []span{r.wikilink(tok, base)}
	case strings.HasPrefix(tok, "["):
		mid := strings.Index(tok, "](")
		id := r.addLink(tok[mid+2:len(tok)-1], false)
		spans := r.inline(tok[1:mid], base.Foreground(r.pal.Link).Underline(true))
		for i := range spans {
			if spans[i].link < 0 {
				spans[i].link = id
			}
		}
		return spans
	case strings.HasPrefix(tok, "**") || strings.HasPrefix(tok, "__"):
		return r.inline(tok[2:len(tok)-2], base.Bold(true))
	case strings.HasPrefix(tok, "~~"):
		return r.inline(tok[2:len(tok)-2], base.Strikethrough(true))
	case strings.HasPrefix(tok, "=="):
		return r.inline(tok[2:len(tok)-2], base.Background(r.pal.Selection))
	case strings.HasPrefix(tok, "*") || strings.HasPrefix(tok, "_"):
		return r.inline(tok[1:len(tok)-1], base.Italic(true))
	case strings.HasPrefix(tok, "#"):
		return []span{plain(tok, base.Foreground(r.pal.Tag))}
	}
	return []span{plain(tok, base)}
}

func (r *renderer) addLink(target string, wiki bool) int {
	r.links = append(r.links, Link{Target: target, Wiki: wiki})
	return len(r.links) - 1
}

// wikilink renders [[target#heading|alias]] and ![[embeds]] by their display
// text, dimmed when the target note doesn't exist.
func (r *renderer) wikilink(tok string, base lipgloss.Style) span {
	embed := strings.HasPrefix(tok, "!")
	inner := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(tok, "!"), "[["), "]]")
	target, alias, hasAlias := strings.Cut(strings.ReplaceAll(inner, `\|`, "|"), "|")
	note, sub, _ := strings.Cut(target, "#")
	display := alias
	if !hasAlias {
		display = note
		if sub != "" {
			if display != "" {
				display += " › "
			}
			display += strings.TrimPrefix(sub, "^")
		}
	}
	if embed {
		display = "⧉ " + display
	}
	st := base.Foreground(r.pal.Link).Underline(true)
	if note != "" && r.resolve != nil && !r.resolve(note) {
		st = base.Foreground(r.pal.DarkForeground).Underline(true)
	}
	return span{display, st, r.addLink(target, true)}
}

// emit wraps spans to the width left after prefix and appends the display
// lines; continuation lines start with cont instead of prefix.
func (r *renderer) emit(src int, prefix, cont string, spans []span, heading int, mode wrapMode) {
	avail := max(r.width-ansi.StringWidth(prefix), 1)
	lines := wrap(spans, avail, mode)
	tail := ""
	if mode == clip && len(lines) > 1 {
		lines = wrap(spans, max(avail-1, 1), mode)[:1]
		tail = r.st.muted.Render("…")
	}
	for j, toks := range lines {
		p, h := prefix, heading
		if j > 0 {
			p, h = cont, 0
		}
		text := ansi.Truncate(p+renderTokens(toks, spans)+tail, r.width, "")
		r.out = append(r.out, Line{Text: text, Src: src, Heading: h, Links: r.linksIn(toks, spans, ansi.StringWidth(p))})
	}
}

// linksIn lists the links on one display line and where each starts.
func (r *renderer) linksIn(toks []token, spans []span, col int) []Link {
	var out []Link
	seen := map[int]bool{}
	for _, t := range toks {
		if id := spans[t.span].link; id >= 0 && !seen[id] && col < r.width {
			seen[id] = true
			l := r.links[id]
			l.Col = col
			out = append(out, l)
		}
		col += t.w
	}
	return out
}

type token struct {
	text  string
	w     int
	span  int
	space bool
}

// wrap breaks spans into lines of at most width cells. It always returns at
// least one (possibly empty) line. In wrapWords mode lines break only at
// whitespace; a word may cross styles (as in "`code`,") and still stays
// whole. In clip mode the text is one unbreakable run, cut wherever it
// reaches the width.
func wrap(spans []span, width int, mode wrapMode) [][]token {
	toks := tokenize(spans, mode)
	var lines [][]token
	var cur []token
	curW := 0
	flush := func() {
		for len(cur) > 0 && cur[len(cur)-1].space {
			curW -= cur[len(cur)-1].w
			cur = cur[:len(cur)-1]
		}
		lines = append(lines, cur)
		cur, curW = nil, 0
	}
	for i := 0; i < len(toks); {
		// A unit is one space run, or every consecutive non-space token.
		j := i + 1
		if !toks[i].space {
			for j < len(toks) && !toks[j].space {
				j++
			}
		}
		unit, uw := toks[i:j], 0
		for _, t := range unit {
			uw += t.w
		}
		i = j
		switch {
		case unit[0].space && curW == 0 && len(lines) > 0:
			// Drop spaces at the start of a continuation line.
		case curW+uw <= width:
			cur = append(cur, unit...)
			curW += uw
		case unit[0].space:
			flush()
		default:
			if curW > 0 && mode == wrapWords {
				flush()
			}
			// The word is wider than a line: cut it across lines.
			for _, t := range unit {
				for curW+t.w > width {
					head, tail := splitAt(t, width-curW)
					if head.w > width-curW && curW > 0 {
						flush()
						continue
					}
					cur = append(cur, head)
					flush()
					t = tail
				}
				if t.text != "" {
					cur = append(cur, t)
					curW += t.w
				}
			}
		}
	}
	if len(cur) > 0 || len(lines) == 0 {
		lines = append(lines, cur)
	}
	return lines
}

// tokenize splits spans into tokens: runs of spaces and non-spaces in
// wrapWords mode, single runes in clip mode.
func tokenize(spans []span, mode wrapMode) []token {
	var toks []token
	for si, sp := range spans {
		runes := []rune(sp.text)
		if mode == clip {
			for _, c := range runes {
				toks = append(toks, token{string(c), ansi.StringWidth(string(c)), si, false})
			}
			continue
		}
		start := 0
		for k := 1; k <= len(runes); k++ {
			if k == len(runes) || unicode.IsSpace(runes[k]) != unicode.IsSpace(runes[start]) {
				s := string(runes[start:k])
				toks = append(toks, token{s, ansi.StringWidth(s), si, unicode.IsSpace(runes[start])})
				start = k
			}
		}
	}
	return toks
}

// splitAt cuts a token so the head is at most width cells (and at least one
// rune, so wrapping always makes progress).
func splitAt(t token, width int) (token, token) {
	runes := []rune(t.text)
	w, k := 0, 0
	for k < len(runes) {
		cw := ansi.StringWidth(string(runes[k]))
		if w+cw > width && k > 0 {
			break
		}
		w += cw
		k++
	}
	head, tail := string(runes[:k]), string(runes[k:])
	return token{head, w, t.span, false}, token{tail, t.w - w, t.span, false}
}

func renderTokens(toks []token, spans []span) string {
	var b strings.Builder
	for j := 0; j < len(toks); {
		k := j
		var s strings.Builder
		for k < len(toks) && toks[k].span == toks[j].span {
			s.WriteString(toks[k].text)
			k++
		}
		b.WriteString(spans[toks[j].span].style.Render(s.String()))
		j = k
	}
	return b.String()
}

func frontmatterEnd(lines []string) int {
	if len(lines) < 2 || strings.TrimRight(lines[0], " ") != "---" {
		return 0
	}
	for i := 1; i < len(lines); i++ {
		if t := strings.TrimRight(lines[i], " "); t == "---" || t == "..." {
			return i
		}
	}
	return 0
}

func fenceMarker(l string) string {
	t := strings.TrimLeft(l, " ")
	if len(l)-len(t) > 3 {
		return ""
	}
	for _, f := range []string{"```", "~~~"} {
		if strings.HasPrefix(t, f) {
			return strings.Repeat(f[:1], len(t)-len(strings.TrimLeft(t, f[:1])))
		}
	}
	return ""
}

func isFenceClose(l, fence string) bool {
	t := strings.TrimSpace(l)
	return strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]) == ""
}

func isRule(t string) bool {
	if len(t) < 3 || !strings.ContainsRune("-*_", rune(t[0])) {
		return false
	}
	n := 0
	for _, c := range t {
		switch {
		case c == rune(t[0]):
			n++
		case c != ' ':
			return false
		}
	}
	return n >= 3
}

func padRight(s string, w int) string {
	if pad := w - ansi.StringWidth(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}
