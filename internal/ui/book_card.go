package ui

import (
	"context"
	"path"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/book"
	"github.com/lurioso/skrin/internal/vault"
)

// LibraryOptions configures the Book Card: where notes and covers land,
// the status a new card starts with, and the network calls it makes.
// Lookup and FetchCover default to book.Lookup and book.FetchCover; tests
// override them so no test ever reaches the network.
type LibraryOptions struct {
	Folder        string
	CoversFolder  string
	DefaultStatus string
	Lookup        func(ctx context.Context, query string) book.Outcome
	FetchCover    func(ctx context.Context, url string) ([]byte, error)
}

func (o LibraryOptions) folder() string {
	if o.Folder != "" {
		return o.Folder
	}
	return "Books"
}

func (o LibraryOptions) coversFolder() string {
	if o.CoversFolder != "" {
		return o.CoversFolder
	}
	return "Assets/Covers"
}

func (o LibraryOptions) defaultStatus() string {
	if o.DefaultStatus != "" {
		return o.DefaultStatus
	}
	return "reading"
}

func (o LibraryOptions) lookup() func(context.Context, string) book.Outcome {
	if o.Lookup != nil {
		return o.Lookup
	}
	return book.Lookup
}

func (o LibraryOptions) fetchCover() func(context.Context, string) ([]byte, error) {
	if o.FetchCover != nil {
		return o.FetchCover
	}
	return book.FetchCover
}

// textArea is a small multi-line text buffer for Notes & Reflections. Tab
// and Shift+Tab leave it for the next field (they're never passed here);
// Up, Down, Home and End move within its own lines instead of between
// fields, and Enter starts a new line rather than moving on, which is what
// tells it apart from every other field in the card.
type textArea struct {
	runes []rune
	cur   int
}

func (t *textArea) value() string { return string(t.runes) }

func (t *textArea) set(s string) {
	t.runes = []rune(strings.ReplaceAll(s, "\r\n", "\n"))
	t.cur = len(t.runes)
}

func (t *textArea) insert(s string) {
	r := []rune(strings.ReplaceAll(s, "\r\n", "\n"))
	t.runes = append(t.runes[:t.cur], append(r, t.runes[t.cur:]...)...)
	t.cur += len(r)
}

func (t *textArea) lineStart(at int) int {
	for at > 0 && t.runes[at-1] != '\n' {
		at--
	}
	return at
}

func (t *textArea) lineEnd(at int) int {
	for at < len(t.runes) && t.runes[at] != '\n' {
		at++
	}
	return at
}

// moveVert moves the cursor a line up (dir<0) or down (dir>0), keeping its
// column when the line it lands on is at least as long. It reports whether
// the movement was handled within the text area. If dir<0 at the first line
// or dir>0 at the last line, it returns false so the caller can navigate to
// adjacent fields.
func (t *textArea) moveVert(dir int) bool {
	col := t.cur - t.lineStart(t.cur)
	if dir < 0 {
		start := t.lineStart(t.cur)
		if start == 0 {
			return false
		}
		prevStart := t.lineStart(start - 1)
		t.cur = min(prevStart+col, start-1)
		return true
	}
	end := t.lineEnd(t.cur)
	if end == len(t.runes) {
		return false
	}
	nextStart := end + 1
	t.cur = min(nextStart+col, t.lineEnd(nextStart))
	return true
}

// handle applies an editing key and reports whether it used it. Tab and
// Shift+Tab are deliberately not handled: the card's field navigation
// catches those before offering the key here.
func (t *textArea) handle(k tea.KeyPressMsg) bool {
	switch k.String() {
	case "backspace", "ctrl+h":
		if t.cur > 0 {
			t.runes = append(t.runes[:t.cur-1], t.runes[t.cur:]...)
			t.cur--
		}
	case "delete":
		if t.cur < len(t.runes) {
			t.runes = append(t.runes[:t.cur], t.runes[t.cur+1:]...)
		}
	case "left":
		t.cur = max(t.cur-1, 0)
	case "right":
		t.cur = min(t.cur+1, len(t.runes))
	case "up":
		return t.moveVert(-1)
	case "down":
		return t.moveVert(1)
	case "home":
		t.cur = t.lineStart(t.cur)
	case "end":
		t.cur = t.lineEnd(t.cur)
	case "enter":
		t.insert("\n")
	default:
		if k.Text == "" || k.Mod&(tea.ModCtrl|tea.ModAlt) != 0 {
			return false
		}
		t.insert(k.Text)
	}
	return true
}

// lines splits the buffer for rendering, with the cursor's row and column.
func (t *textArea) lines() (lines []string, row, col int) {
	s := string(t.runes)
	lines = strings.Split(s, "\n")
	before := string(t.runes[:t.cur])
	row = strings.Count(before, "\n")
	if i := strings.LastIndexByte(before, '\n'); i >= 0 {
		col = len([]rune(before[i+1:]))
	} else {
		col = len([]rune(before))
	}
	return lines, row, col
}

// wrapped is lines but with every raw line word-wrapped to at most w
// cells, so a long line never runs off the edge of whatever box shows it.
// The cursor position is carried through in wrapped-row terms.
func (t *textArea) wrapped(w int) (rows []string, curRow, curCol int) {
	lines, rawRow, rawCol := t.lines()
	for i, l := range lines {
		wr, rowAt, colAt := wrapPlain([]rune(l), w)
		if i == rawRow {
			curRow, curCol = len(rows)+rowAt[rawCol], colAt[rawCol]
		}
		rows = append(rows, wr...)
	}
	return rows, curRow, curCol
}

// wrapPlain word-wraps the runes of one line to at most w cells a row,
// breaking between words when possible; a single word longer than w is
// hard-broken instead of overflowing. rowAt/colAt map every rune position
// in r (0 through len(r), so the position right after the last rune has
// an entry too) to the display row and column it lands on — how a cursor
// index survives the wrap.
func wrapPlain(r []rune, w int) (rows []string, rowAt, colAt []int) {
	if w < 1 {
		w = 1
	}
	n := len(r)
	rowAt, colAt = make([]int, n+1), make([]int, n+1)
	var cur []rune
	row := 0
	for i := 0; i < n; {
		j := i
		if r[i] == ' ' {
			j = i + 1
		} else {
			for j < n && r[j] != ' ' {
				j++
			}
		}
		wordStart, word := i, r[i:j]
		if len(cur) > 0 && len(cur)+len(word) > w {
			rows = append(rows, string(cur))
			cur, row = nil, row+1
			if word[0] == ' ' { // the wrap eats the one space that forced it
				rowAt[wordStart], colAt[wordStart] = row, 0
				wordStart++
				word = word[1:]
			}
		}
		for k, ch := range word {
			idx := wordStart + k
			if len(cur) == w {
				rows = append(rows, string(cur))
				cur, row = nil, row+1
			}
			rowAt[idx], colAt[idx] = row, len(cur)
			cur = append(cur, ch)
		}
		i = j
	}
	rows = append(rows, string(cur))
	rowAt[n], colAt[n] = row, len(cur)
	return rows, rowAt, colAt
}

// quoteRow is one passage in the card's Quotes section.
type quoteRow struct {
	text, page, speaker lineInput
}

// bookArea says which part of the card a field lives in, for Tab order.
type bookArea int

const (
	bookAreaSearch bookArea = iota
	bookAreaStatic
	bookAreaQuote
	bookAreaNotes
	bookAreaSave
)

// bookCard is the B overlay: the fetch bar, every bibliographic field, the
// quotes and the reflections, plus enough state to save what's typed.
type bookCard struct {
	editingRel string // "" for a new note; otherwise the note being edited
	origSrc    string // editingRel's text when the card opened, for the undo snapshot
	origCover  string // its existing cover path, kept unless a fetch replaces it

	search   lineInput
	fetching bool

	title, subtitle, authors, translators lineInput
	origYear, editionYear, pages          lineInput
	publisher, edition, isbn, format      lineInput
	shelf, rating, status                 lineInput
	started, finished                     lineInput

	quotes []quoteRow
	notes  textArea

	pendingCoverURL string // queued by a fetch result; downloaded on save

	area       bookArea
	staticIdx  int
	quoteIdx   int
	quoteField int // 0 text, 1 page, 2 speaker

	err string
}

// staticFields lists the single-line bibliographic fields in Tab order.
func (c *bookCard) staticFields() []*lineInput {
	return []*lineInput{
		&c.title, &c.subtitle, &c.authors, &c.translators,
		&c.origYear, &c.editionYear, &c.pages,
		&c.publisher, &c.edition, &c.isbn, &c.format,
		&c.shelf, &c.rating, &c.status, &c.started, &c.finished,
	}
}

var staticLabels = []string{
	"Title", "Subtitle", "Author(s)", "Translator(s)",
	"Orig. Year", "Print Year", "Pages",
	"Publisher", "Edition", "ISBN", "Format",
	"Shelf", "Rating (0-5)", "Status", "Started", "Finished",
}

// flatCount is the number of stops the Tab order visits.
func (c *bookCard) flatCount() int {
	return 1 + len(c.staticFields()) + 3*len(c.quotes) + 1 + 1 // search, static, quotes, notes, save
}

func (c *bookCard) flatIndex() int {
	n := len(c.staticFields())
	switch c.area {
	case bookAreaSearch:
		return 0
	case bookAreaStatic:
		return 1 + c.staticIdx
	case bookAreaQuote:
		return 1 + n + c.quoteIdx*3 + c.quoteField
	case bookAreaNotes:
		return 1 + n + 3*len(c.quotes)
	default: // bookAreaSave
		return 1 + n + 3*len(c.quotes) + 1
	}
}

func (c *bookCard) setFlat(i int) {
	total := c.flatCount()
	i = ((i % total) + total) % total
	n := len(c.staticFields())
	switch {
	case i == 0:
		c.area = bookAreaSearch
	case i < 1+n:
		c.area, c.staticIdx = bookAreaStatic, i-1
	case i < 1+n+3*len(c.quotes):
		rem := i - 1 - n
		c.area, c.quoteIdx, c.quoteField = bookAreaQuote, rem/3, rem%3
	case i == 1+n+3*len(c.quotes):
		c.area = bookAreaNotes
	default:
		c.area = bookAreaSave
	}
}

func (c *bookCard) next() { c.setFlat(c.flatIndex() + 1) }
func (c *bookCard) prev() { c.setFlat(c.flatIndex() - 1) }

// focusedInput is the lineInput under focus, or nil in Notes and Save.
func (c *bookCard) focusedInput() *lineInput {
	switch c.area {
	case bookAreaSearch:
		return &c.search
	case bookAreaStatic:
		return c.staticFields()[c.staticIdx]
	case bookAreaQuote:
		q := &c.quotes[c.quoteIdx]
		switch c.quoteField {
		case 0:
			return &q.text
		case 1:
			return &q.page
		default:
			return &q.speaker
		}
	}
	return nil
}

// addQuote appends a blank quote row and focuses its text field.
func (c *bookCard) addQuote() {
	c.quotes = append(c.quotes, quoteRow{})
	c.area, c.quoteIdx, c.quoteField = bookAreaQuote, len(c.quotes)-1, 0
}

// toBook builds the Book the card currently describes.
func (c *bookCard) toBook(defaultTags []string) book.Book {
	b := book.Book{
		Title:        c.title.value(),
		Subtitle:     c.subtitle.value(),
		Authors:      splitList(c.authors.value()),
		Translators:  splitList(c.translators.value()),
		OriginalYear: c.origYear.value(),
		EditionYear:  c.editionYear.value(),
		Publisher:    c.publisher.value(),
		Edition:      c.edition.value(),
		ISBN:         c.isbn.value(),
		Format:       c.format.value(),
		Pages:        c.pages.value(),
		Shelf:        c.shelf.value(),
		Status:       strings.ToLower(strings.TrimSpace(c.status.value())),
		Rating:       clamp(atoiOr(c.rating.value(), 0), 0, 5),
		Started:      c.started.value(),
		Finished:     c.finished.value(),
		Tags:         defaultTags,
		Cover:        c.origCover,
		Notes:        c.notes.value(),
	}
	for _, q := range c.quotes {
		if strings.TrimSpace(q.text.value()) == "" {
			continue
		}
		b.Quotes = append(b.Quotes, book.Quote{Text: q.text.value(), Page: q.page.value(), Speaker: q.speaker.value()})
	}
	return b
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func atoiOr(s string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return fallback
	}
	return n
}

// openBookCard opens a blank Book Card, or — if the open note parses as a
// book note — one pre-filled from it for editing.
func (m *Model) openBookCard() {
	c := &bookCard{}
	if m.notePath != "" {
		if b, ok := book.Parse(m.noteSrc); ok {
			c.editingRel, c.origSrc, c.origCover = m.notePath, m.noteSrc, b.Cover
			fillBookCard(c, b)
			m.book = c
			m.flash = "Editing " + m.notePath
			return
		}
	}
	c.status.set(m.opts.Library.defaultStatus())
	c.quotes = []quoteRow{{}}
	m.book = c
}

func fillBookCard(c *bookCard, b book.Book) {
	c.title.set(b.Title)
	c.subtitle.set(b.Subtitle)
	c.authors.set(strings.Join(b.Authors, ", "))
	c.translators.set(strings.Join(b.Translators, ", "))
	c.origYear.set(b.OriginalYear)
	c.editionYear.set(b.EditionYear)
	c.pages.set(b.Pages)
	c.publisher.set(b.Publisher)
	c.edition.set(b.Edition)
	c.isbn.set(b.ISBN)
	c.format.set(b.Format)
	c.shelf.set(b.Shelf)
	if b.Rating > 0 {
		c.rating.set(strconv.Itoa(b.Rating))
	}
	c.status.set(b.Status)
	c.started.set(b.Started)
	c.finished.set(b.Finished)
	c.notes.set(b.Notes)
	for _, q := range b.Quotes {
		row := quoteRow{}
		row.text.set(q.Text)
		row.page.set(q.Page)
		row.speaker.set(q.Speaker)
		c.quotes = append(c.quotes, row)
	}
	if len(c.quotes) == 0 {
		c.quotes = []quoteRow{{}}
	}
}

// bookLookupMsg carries a metadata search's outcome back to the update
// loop.
type bookLookupMsg struct {
	query   string
	outcome book.Outcome
}

// bookSaveMsg carries a save's cover download (if any) back to the update
// loop, which does the actual vault write on the main goroutine.
type bookSaveMsg struct {
	rel, editingRel, origSrc string
	b                        book.Book
	coverRel                 string
	coverData                []byte
	coverErr                 error
}

func (m *Model) bookCardKey(k tea.KeyPressMsg) tea.Cmd {
	c := m.book
	key := k.String()
	switch m.actionIn(inBookCard, key) {
	case actCancel:
		m.book = nil
		m.flash = "Book Card closed"
		return nil
	case actFetchBook:
		return m.startBookLookup()
	case actAddQuote:
		c.addQuote()
		return nil
	case actSaveBook:
		return m.startBookSave()
	case actUp:
		c.err = ""
		if c.area == bookAreaNotes {
			if !c.notes.moveVert(-1) {
				c.prev()
			}
		} else {
			c.prev()
		}
		return nil
	case actDown:
		c.err = ""
		if c.area == bookAreaNotes {
			if !c.notes.moveVert(1) {
				c.next()
			}
		} else {
			c.next()
		}
		return nil
	case actNextField:
		c.err = ""
		c.next()
		return nil
	case actPrevField:
		c.err = ""
		c.prev()
		return nil
	case actPick:
		switch c.area {
		case bookAreaSearch:
			return m.startBookLookup()
		case bookAreaSave:
			return m.startBookSave()
		default:
			c.next()
		}
		return nil
	}
	// Everything else edits the focused field (text typing, backspace, left/right, etc.).
	if c.area == bookAreaNotes {
		c.notes.handle(k)
		return nil
	}
	if in := c.focusedInput(); in != nil {
		in.handle(k)
	}
	return nil
}

// startBookLookup fetches metadata for the search bar's query. The network
// call runs off the update loop, so the card stays responsive; a 5s
// timeout inside book.Lookup keeps a dead network from hanging it.
func (m *Model) startBookLookup() tea.Cmd {
	c := m.book
	query := strings.TrimSpace(c.search.value())
	if query == "" {
		return nil
	}
	c.fetching = true
	m.flash = "Looking up " + query + "…"
	lookup := m.opts.Library.lookup()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		outcome := lookup(ctx, query)
		return bookLookupMsg{query: query, outcome: outcome}
	}
}

// bookLookupDone applies a lookup's outcome: a single match fills the
// card at once, several open the fuzzy chooser, none flashes and leaves
// the card exactly as it was. Amendment 1's honest-outcome message rides
// alongside either path: silent on a clean match (the chooser opening
// speaks for itself), naming what's wrong otherwise (a down provider, or
// the truthful word for "nothing, anywhere" — "offline" only when every
// provider failed to answer). The flash is always assigned, even to "",
// so the transient "Looking up…" flash never gets stuck once the result
// lands — the exact bug the backlog reported and this amendment specs
// as its own to fix.
func (m *Model) bookLookupDone(msg bookLookupMsg) {
	if m.book == nil {
		return
	}
	m.book.fetching = false
	o := msg.outcome
	m.flash = o.Message(msg.query)
	if len(o.Results) == 0 {
		return
	}
	items := make([]choice, len(o.Results))
	for i, r := range o.Results {
		r := r
		items[i] = choice{label: r.Label(), detail: r.Detail(), do: func() { m.applyBookResult(r) }}
	}
	m.openChooser(&chooser{
		title:  "Book lookup",
		prompt: "filter",
		empty:  "no matches",
		verb:   "use",
		items:  items,
	})
}

// applyBookResult fills the card from a chosen search result and queues
// its cover, if any, for download on save.
func (m *Model) applyBookResult(r book.Result) {
	c := m.book
	if c == nil {
		return
	}
	c.title.set(r.Title)
	c.subtitle.set(r.Subtitle)
	if len(r.Authors) > 0 {
		c.authors.set(strings.Join(r.Authors, ", "))
	}
	if len(r.Translators) > 0 {
		c.translators.set(strings.Join(r.Translators, ", "))
	}
	if r.Year != "" {
		c.editionYear.set(r.Year)
	}
	if r.Publisher != "" {
		c.publisher.set(r.Publisher)
	}
	if r.ISBN != "" {
		c.isbn.set(r.ISBN)
	}
	if r.Pages != "" {
		c.pages.set(r.Pages)
	}
	c.pendingCoverURL = r.CoverURL
	c.origCover = "" // a new fetch replaces whatever cover was there
	m.flash = "Filled in from " + r.Source
}

// startBookSave validates the card and, if it queued a cover, downloads it
// off the update loop before finishBookSave writes anything to the vault.
func (m *Model) startBookSave() tea.Cmd {
	c := m.book
	title := strings.TrimSpace(c.title.value())
	if title == "" {
		c.err = "Title is required"
		c.area, c.staticIdx = bookAreaStatic, 0
		return nil
	}
	b := c.toBook(book.DefaultTags)
	rel := c.editingRel
	if rel == "" {
		rel = path.Join(m.opts.Library.folder(), book.NoteName(title))
		if m.vault.Exists(rel) {
			c.err = rel + " already exists"
			return nil
		}
	}
	coverURL := c.pendingCoverURL
	var coverRel string
	if coverURL != "" {
		coverRel = book.CoverPath(m.opts.Library.coversFolder(), title, b.EditionYear, book.CoverExt(coverURL), m.vault.Exists)
	}
	c.err = ""
	fetchCover := m.opts.Library.fetchCover()
	origSrc, editingRel := c.origSrc, c.editingRel
	return func() tea.Msg {
		var data []byte
		var err error
		if coverURL != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			data, err = fetchCover(ctx, coverURL)
			cancel()
		}
		return bookSaveMsg{rel: rel, editingRel: editingRel, origSrc: origSrc, b: b, coverRel: coverRel, coverData: data, coverErr: err}
	}
}

// finishBookSave writes the note (and its cover, if one downloaded) to the
// vault on the main goroutine, snapshotting and journalling exactly like
// every other write Skrin makes, so u and U both work on a book note.
func (m *Model) finishBookSave(msg bookSaveMsg) {
	b := msg.b
	var coverNote string
	if msg.coverRel != "" {
		if msg.coverErr != nil {
			coverNote = " (cover download failed, offline? — saved without it)"
		} else if err := book.SaveCover(m.vault.Abs, msg.coverRel, msg.coverData); err != nil {
			coverNote = " (couldn't save the cover: " + err.Error() + ")"
		} else {
			b.Cover = msg.coverRel
		}
	}
	content := book.Render(b)

	if msg.editingRel == "" {
		dirs, err := m.vault.CreateFile(msg.rel, content)
		steps := createdSteps(dirs)
		if err == nil {
			steps = append(steps, vault.Step{Kind: vault.StepCreated, Rel: msg.rel})
		}
		m.journal.Record(vault.Op{Desc: "create " + msg.rel, Steps: steps})
		if err != nil {
			if m.book != nil {
				m.book.err = err.Error()
			}
			return
		}
	} else {
		if err := m.snaps.Save(msg.rel, msg.origSrc); err != nil {
			if m.book != nil {
				m.book.err = "couldn't snapshot the note first: " + err.Error()
			}
			return
		}
		if err := m.vault.Write(msg.rel, content); err != nil {
			if m.book != nil {
				m.book.err = err.Error()
			}
			return
		}
		m.journal.Record(vault.Op{Desc: "edit " + msg.rel, Steps: []vault.Step{{Kind: vault.StepModified, Rel: msg.rel, Content: msg.origSrc}}})
	}
	m.book = nil
	m.reveal(msg.rel)
	m.pushHistory()
	m.showNote(msg.rel)
	m.flash = "Saved " + msg.rel + coverNote
}

// bookCardBox renders the card as a floating overlay.
func (m *Model) bookCardBox() []string {
	c := m.book
	w := min(max(m.width-6, 60), 92)
	inner := w - 4
	var body []string
	row := func(focused bool, s string) string {
		if focused {
			return "  " + m.st.selFocus.Render(fit(s, inner-2))
		}
		return "  " + fit(s, inner-2)
	}
	field := func(label string, in *lineInput, focused bool) string {
		text := m.st.muted.Render(label+": ") + in.view(m.st.text, m.st.cursor)
		if focused {
			text = m.st.titleFocus.Render(label+": ") + in.view(m.st.text, m.st.cursor)
		}
		return text
	}

	searchFocused := c.area == bookAreaSearch
	fetchLabel := "Fetch ▸ "
	if c.fetching {
		fetchLabel = "Fetching… "
	}
	body = append(body, row(searchFocused, fetchLabel+c.search.view(m.st.text, m.st.cursor)))
	body = append(body, strings.Repeat("─", inner))

	labels := staticLabels
	fields := c.staticFields()
	// Pairs of fields share a row where the mock-up does (year/pages,
	// publisher/edition, isbn/format, shelf/rating, status/started); the
	// rest sit alone. Kept simple and vertical here for width safety at
	// small terminal sizes; the fields and their order match the spec.
	for i, f := range fields {
		focused := c.area == bookAreaStatic && c.staticIdx == i
		body = append(body, row(focused, field(labels[i], f, focused)))
	}
	if c.err != "" {
		body = append(body, m.st.errText.Render("  "+c.err))
	}
	body = append(body, strings.Repeat("─", inner))

	body = append(body, "  "+m.st.muted.Render("Quotes & Passages:")+"  "+m.st.muted.Render("(Alt+q +row)"))
	for qi, q := range c.quotes {
		tf := c.area == bookAreaQuote && c.quoteIdx == qi && c.quoteField == 0
		pf := c.area == bookAreaQuote && c.quoteIdx == qi && c.quoteField == 1
		sf := c.area == bookAreaQuote && c.quoteIdx == qi && c.quoteField == 2

		tCur := m.st.text
		if tf {
			tCur = m.st.cursor
		}
		body = append(body, row(tf, "\" "+q.text.view(m.st.text, tCur)+" \""))

		pageCur := m.st.text
		pageLabel := m.st.muted.Render("Page: ")
		if pf {
			pageCur = m.st.cursor
			pageLabel = m.st.titleFocus.Render("Page: ")
		}
		pagePart := pageLabel + q.page.view(m.st.text, pageCur)

		speakerCur := m.st.text
		speakerLabel := m.st.muted.Render("Speaker: ")
		if sf {
			speakerCur = m.st.cursor
			speakerLabel = m.st.titleFocus.Render("Speaker: ")
		}
		speakerPart := speakerLabel + q.speaker.view(m.st.text, speakerCur)

		body = append(body, "    "+pagePart)
		body = append(body, "    "+speakerPart)
	}
	body = append(body, strings.Repeat("─", inner))

	body = append(body, "  "+m.st.muted.Render("Notes & Reflections:"))
	lines, curRow, curCol := c.notes.wrapped(inner - 2)
	for i, l := range lines {
		text := l
		if c.area == bookAreaNotes && i == curRow {
			r := []rune(l)
			at := " "
			if curCol < len(r) {
				at = string(r[curCol])
			}
			pre, post := "", ""
			if curCol <= len(r) {
				pre = string(r[:curCol])
			}
			if curCol < len(r) {
				post = string(r[curCol+1:])
			}
			text = m.st.text.Render(pre) + m.st.cursor.Render(at) + m.st.text.Render(post)
			body = append(body, "  "+text)
			continue
		}
		body = append(body, "  "+m.st.text.Render(text))
	}
	body = append(body, "")

	saveLabel := "[ Save ]"
	if c.area == bookAreaSave {
		saveLabel = m.st.selFocus.Render(saveLabel)
	} else {
		saveLabel = m.st.bold.Render(saveLabel)
	}
	body = append(body, "  "+saveLabel+"   "+m.st.muted.Render("Ctrl+s save · Esc cancel · Alt+q +quote"))

	title := "Book Card"
	if c.editingRel != "" {
		title = "Book Card — " + displayName(c.editingRel)
	}
	h := clamp(len(body)+2, 10, max(m.height-4, 10))
	return m.box(title, body, w, h, true)
}
