package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/book"
)

// pump sends key to the model and, if it returned a command (an async
// lookup or save), runs the command and feeds its message straight back
// in — the same round trip Bubble Tea itself does, without a real runtime.
func pump(m *Model, k string) {
	_, cmd := m.Update(key(k))
	if cmd != nil {
		m.Update(cmd())
	}
}

func TestBookCardOpensBlank(t *testing.T) {
	m := newTestModel(t)
	press(m, "B")
	if m.book == nil {
		t.Fatal("B should open the Book Card")
	}
	if m.book.status.value() != "reading" {
		t.Errorf("default status = %q, want reading", m.book.status.value())
	}
	if len(m.book.quotes) != 1 {
		t.Errorf("a fresh card should start with one blank quote row, got %d", len(m.book.quotes))
	}
	if m.book.area != bookAreaSearch {
		t.Errorf("a fresh card should focus the search bar first")
	}
}

func TestBookCardEscCloses(t *testing.T) {
	m := newTestModel(t)
	press(m, "B")
	press(m, "esc")
	if m.book != nil {
		t.Error("esc should close the Book Card")
	}
	if m.flash == "" {
		t.Error("closing should flash something")
	}
}

func TestBookCardTabNavigatesFields(t *testing.T) {
	m := newTestModel(t)
	press(m, "B")
	press(m, "tab") // search -> title
	if m.book.area != bookAreaStatic || m.book.staticIdx != 0 {
		t.Fatalf("area=%v idx=%d, want static/0 (title)", m.book.area, m.book.staticIdx)
	}
	typeText(m, "Meditations")
	press(m, "tab") // title -> subtitle
	if m.book.title.value() != "Meditations" {
		t.Errorf("title = %q", m.book.title.value())
	}
	if m.book.staticIdx != 1 {
		t.Errorf("staticIdx = %d, want 1 (subtitle)", m.book.staticIdx)
	}
	press(m, "shift+tab") // back to title
	if m.book.staticIdx != 0 {
		t.Errorf("shift+tab should step back to title, staticIdx = %d", m.book.staticIdx)
	}
}

func TestBookCardAltQAddsQuoteRow(t *testing.T) {
	m := newTestModel(t)
	press(m, "B")
	press(m, "alt+q")
	if len(m.book.quotes) != 2 {
		t.Fatalf("Alt+q should add a quote row, got %d", len(m.book.quotes))
	}
	if m.book.area != bookAreaQuote || m.book.quoteIdx != 1 || m.book.quoteField != 0 {
		t.Errorf("Alt+q should focus the new row's text field: area=%v idx=%d field=%d", m.book.area, m.book.quoteIdx, m.book.quoteField)
	}
	typeText(m, "carpe diem")
	if m.book.quotes[1].text.value() != "carpe diem" {
		t.Errorf("quote text = %q", m.book.quotes[1].text.value())
	}
}

func TestBookCardSaveRequiresTitle(t *testing.T) {
	m := newTestModel(t)
	press(m, "B")
	pump(m, "ctrl+s")
	if m.book == nil {
		t.Fatal("save without a title should leave the card open")
	}
	if m.book.err == "" {
		t.Error("save without a title should show an error")
	}
}

func TestBookCardSaveCreatesNoteAndClosesCard(t *testing.T) {
	m := newTestModel(t)
	press(m, "B")
	press(m, "tab") // to title
	typeText(m, "Meditations")
	pump(m, "ctrl+s")
	if m.book != nil {
		t.Fatalf("save should close the card, err=%q", m.book.err)
	}
	rel := "Books/Meditations.md"
	src, err := m.vault.Read(rel)
	if err != nil {
		t.Fatalf("Read(%s): %v", rel, err)
	}
	if !strings.Contains(src, "type: book") || !strings.Contains(src, `title: Meditations`) {
		t.Errorf("note content missing expected frontmatter:\n%s", src)
	}
	if m.notePath != rel {
		t.Errorf("notePath = %q, want %q (the new note should open)", m.notePath, rel)
	}
	if m.journal.Len() != 1 {
		t.Errorf("journal.Len() = %d, want 1 (so U undoes the create)", m.journal.Len())
	}
}

func TestBookCardUndoRemovesCreatedNote(t *testing.T) {
	m := newTestModel(t)
	press(m, "B")
	press(m, "tab")
	typeText(m, "Meditations")
	pump(m, "ctrl+s")
	rel := "Books/Meditations.md"
	if !m.vault.Exists(rel) {
		t.Fatal("note wasn't created")
	}
	press(m, "1", "U")
	if m.vault.Exists(rel) {
		t.Error("U should undo the book note's creation")
	}
}

func TestBookCardReopensExistingBookNote(t *testing.T) {
	m := newTestModel(t)
	content := book.Render(book.Book{
		Title: "Kallocain", Authors: []string{"Karin Boye"}, Status: "read", Rating: 5,
	})
	if _, err := m.vault.CreateFile("Books/Kallocain.md", content); err != nil {
		t.Fatal(err)
	}
	m.refresh()
	m.showNote("Books/Kallocain.md")
	press(m, "2", "B")
	if m.book == nil {
		t.Fatal("B on an open book note should open the card")
	}
	if m.book.editingRel != "Books/Kallocain.md" {
		t.Errorf("editingRel = %q", m.book.editingRel)
	}
	if m.book.title.value() != "Kallocain" || m.book.authors.value() != "Karin Boye" {
		t.Errorf("title=%q authors=%q, want prefilled from the note", m.book.title.value(), m.book.authors.value())
	}
	if m.book.rating.value() != "5" {
		t.Errorf("rating = %q, want 5", m.book.rating.value())
	}
}

func TestBookCardEditSavesInPlaceWithSnapshot(t *testing.T) {
	m := newTestModel(t)
	content := book.Render(book.Book{Title: "Kallocain", Authors: []string{"Karin Boye"}, Status: "reading"})
	if _, err := m.vault.CreateFile("Books/Kallocain.md", content); err != nil {
		t.Fatal(err)
	}
	m.refresh()
	m.showNote("Books/Kallocain.md")
	press(m, "2", "B")
	// Move to the Status field and change it.
	for m.book.staticIdx != 13 || m.book.area != bookAreaStatic { // Status is index 13
		press(m, "tab")
	}
	// Clear and retype the status field.
	for range m.book.status.value() {
		press(m, "backspace")
	}
	typeText(m, "read")
	pump(m, "ctrl+s")
	if m.book != nil {
		t.Fatalf("save should close the card, err=%q", m.book.err)
	}
	src, err := m.vault.Read("Books/Kallocain.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(src, "status: read") {
		t.Errorf("status wasn't updated:\n%s", src)
	}
	// u should bring the previous version back (the snapshot store).
	press(m, "u")
	src, err = m.vault.Read("Books/Kallocain.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(src, "status: reading") {
		t.Errorf("u should restore the pre-edit note:\n%s", src)
	}
}

func TestBookCardLookupFillsSingleFieldOnSelection(t *testing.T) {
	m := newTestModel(t)
	m.opts.Library.Lookup = func(ctx context.Context, q string) book.Outcome {
		return book.Outcome{
			Results: []book.Result{
				{Title: "Meditations", Authors: []string{"Marcus Aurelius"}, Year: "2003", Publisher: "Modern Library", CoverURL: "https://covers.example/1.jpg", Source: "Open Library"},
				{Title: "Meditations", Authors: []string{"Marcus Aurelius"}, Year: "2006", Publisher: "Penguin Classics", Source: "Open Library"},
			},
			Tried: []string{"Open Library"},
		}
	}
	press(m, "B")
	typeText(m, "Meditations")
	pump(m, "enter") // fetch from the search bar
	if m.chooser == nil {
		t.Fatal("two results should open the fuzzy chooser")
	}
	press(m, "enter") // pick the first (highlighted) result
	if m.chooser != nil {
		t.Error("picking a result should close the chooser")
	}
	if m.book.title.value() != "Meditations" || m.book.publisher.value() != "Modern Library" {
		t.Errorf("title=%q publisher=%q, want filled from the chosen result", m.book.title.value(), m.book.publisher.value())
	}

	if m.book.pendingCoverURL == "" {
		t.Error("the chosen result's cover should be queued for download")
	}
}

// TestBookCardLookupClearsTheLookingUpFlashOnAMatch is Amendment 1's own
// bug: the "Looking up…" transient flash must never linger once a result
// lands, even a clean match where the chooser opening is the "success"
// signal and there's nothing further to say.
func TestBookCardLookupClearsTheLookingUpFlashOnAMatch(t *testing.T) {
	m := newTestModel(t)
	m.opts.Library.Lookup = func(ctx context.Context, q string) book.Outcome {
		return book.Outcome{
			Results: []book.Result{{Title: "Meditations", Source: "Open Library"}},
			Tried:   []string{"Open Library"},
		}
	}
	press(m, "B")
	typeText(m, "Meditations")
	pump(m, "enter")
	if strings.Contains(m.flash, "Looking up") {
		t.Errorf("flash = %q, the transient flash should be cleared once the result lands", m.flash)
	}
}

func TestBookCardLookupFailureFlashesAndKeepsCardEditable(t *testing.T) {
	m := newTestModel(t)
	m.opts.Library.Lookup = func(ctx context.Context, q string) book.Outcome {
		return book.Outcome{
			Tried: []string{"Open Library", "Google Books", "Libris"},
			Down:  []string{"Open Library", "Google Books", "Libris"},
		}
	}
	press(m, "B")
	typeText(m, "Meditations")
	pump(m, "enter")
	if m.book == nil {
		t.Fatal("a failed lookup must not close the card")
	}
	if !strings.Contains(m.flash, "offline") {
		t.Errorf("flash = %q, want an offline message", m.flash)
	}
}

func TestBookCardSaveDownloadsCover(t *testing.T) {
	m := newTestModel(t)
	m.opts.Library.Lookup = func(ctx context.Context, q string) book.Outcome {
		return book.Outcome{
			Results: []book.Result{{Title: "Meditations", Year: "2003", CoverURL: "https://covers.example/1.jpg"}},
			Tried:   []string{"Open Library"},
		}
	}
	m.opts.Library.FetchCover = func(ctx context.Context, url string) ([]byte, error) {
		return []byte("fake-jpeg-bytes"), nil
	}
	press(m, "B")
	typeText(m, "Meditations")
	pump(m, "enter") // a single result fills the card at once via the chooser
	if m.chooser != nil {
		press(m, "enter")
	}
	pump(m, "ctrl+s")
	if m.book != nil {
		t.Fatalf("save should close the card, err=%q", m.book.err)
	}
	coverPath := filepath.Join(m.vault.Root, "Assets", "Covers", "Meditations-2003.jpg")
	data, err := os.ReadFile(coverPath)
	if err != nil {
		t.Fatalf("cover wasn't saved: %v", err)
	}
	if string(data) != "fake-jpeg-bytes" {
		t.Errorf("cover content = %q", data)
	}
	src, _ := m.vault.Read("Books/Meditations.md")
	if !strings.Contains(src, "cover: Assets/Covers/Meditations-2003.jpg") {
		t.Errorf("frontmatter should reference the saved cover:\n%s", src)
	}
}

func TestBookCardSaveContinuesWithoutCoverOnFetchFailure(t *testing.T) {
	m := newTestModel(t)
	m.opts.Library.Lookup = func(ctx context.Context, q string) book.Outcome {
		return book.Outcome{
			Results: []book.Result{{Title: "Meditations", Year: "2003", CoverURL: "https://covers.example/1.jpg"}},
			Tried:   []string{"Open Library"},
		}
	}
	m.opts.Library.FetchCover = func(ctx context.Context, url string) ([]byte, error) {
		return nil, context.DeadlineExceeded
	}
	press(m, "B")
	typeText(m, "Meditations")
	pump(m, "enter")
	if m.chooser != nil {
		press(m, "enter")
	}
	pump(m, "ctrl+s")
	if m.book != nil {
		t.Fatalf("a cover failure must not block saving the note, err=%q", m.book.err)
	}
	if !strings.Contains(m.flash, "cover") {
		t.Errorf("flash = %q, should mention the cover failure", m.flash)
	}
	src, err := m.vault.Read("Books/Meditations.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(src, "cover: Assets") {
		t.Errorf("note shouldn't reference a cover that failed to download:\n%s", src)
	}
}

func TestBookCardHonoursLibraryConfig(t *testing.T) {
	m := newTestModelWith(t, Options{
		Library: LibraryOptions{Folder: "Library", CoversFolder: "Media/Covers", DefaultStatus: "want"},
	})
	press(m, "B")
	if m.book.status.value() != "want" {
		t.Errorf("default status = %q, want want", m.book.status.value())
	}
	press(m, "tab")
	typeText(m, "Kallocain")
	pump(m, "ctrl+s")
	if !m.vault.Exists("Library/Kallocain.md") {
		t.Error("the note should be created in the configured library folder")
	}
}

func TestBookCardArrowNavigationInNotesAndQuotes(t *testing.T) {
	m := newTestModel(t)
	press(m, "B")

	// Navigate to notes using shift+tab from search (wraps to Save -> Notes).
	press(m, "shift+tab") // to Save
	if m.book.area != bookAreaSave {
		t.Fatalf("area=%v, want save", m.book.area)
	}
	press(m, "up") // to Notes
	if m.book.area != bookAreaNotes {
		t.Fatalf("area=%v, want notes", m.book.area)
	}

	// In single-line notes, pressing "up" should navigate to the previous field (quote speaker).
	press(m, "up")
	if m.book.area != bookAreaQuote || m.book.quoteField != 2 {
		t.Errorf("up from single-line notes should go to quote speaker, got area=%v quoteField=%d", m.book.area, m.book.quoteField)
	}

	// Pressing "down" from quote speaker should go to notes.
	press(m, "down")
	if m.book.area != bookAreaNotes {
		t.Errorf("down from quote speaker should go to notes, got area=%v", m.book.area)
	}

	// Pressing "down" from notes should go to Save.
	press(m, "down")
	if m.book.area != bookAreaSave {
		t.Errorf("down from single-line notes should go to save, got area=%v", m.book.area)
	}
}

func TestWrapPlainBreaksBetweenWords(t *testing.T) {
	rows, rowAt, colAt := wrapPlain([]rune("the quick brown fox"), 10)
	want := []string{"the quick ", "brown fox"}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows=%q, want %q", rows, want)
	}
	// "brown" starts right after the wrap, at row 1 col 0.
	brownAt := len("the quick ")
	if rowAt[brownAt] != 1 || colAt[brownAt] != 0 {
		t.Errorf("rowAt[%d]=%d colAt[%d]=%d, want row=1 col=0", brownAt, rowAt[brownAt], brownAt, colAt[brownAt])
	}
	// The final position (one past the last rune) lands after "fox".
	end := len("the quick brown fox")
	if rowAt[end] != 1 || colAt[end] != len("brown fox") {
		t.Errorf("rowAt[end]=%d colAt[end]=%d, want row=1 col=%d", rowAt[end], colAt[end], len("brown fox"))
	}
}

func TestWrapPlainHardBreaksAnOverlongWord(t *testing.T) {
	rows, _, _ := wrapPlain([]rune("supercalifragilistic"), 6)
	for _, r := range rows {
		if len([]rune(r)) > 6 {
			t.Fatalf("row %q exceeds width 6", r)
		}
	}
	if strings.Join(rows, "") != "supercalifragilistic" {
		t.Fatalf("rows %q lost characters", rows)
	}
}

func TestTextAreaWrappedTracksCursor(t *testing.T) {
	ta := &textArea{}
	ta.set("the quick brown fox jumps")
	// Row 1 ("brown fox ") starts right after "the quick " wraps; "fox"
	// starts 6 cells into that row.
	ta.cur = len([]rune("the quick brown "))
	rows, row, col := ta.wrapped(10)
	if len(rows) < 2 {
		t.Fatalf("expected at least 2 wrapped rows, got %d: %q", len(rows), rows)
	}
	if row != 1 || col != 6 {
		t.Errorf("row=%d col=%d, want row=1 col=6", row, col)
	}
}

func TestQuickNoteBoxWrapsLongLinesInsteadOfRunningOff(t *testing.T) {
	m := newTestModel(t)
	press(m, "n")
	long := strings.Repeat("word ", 40)
	for _, r := range long {
		press(m, string(r))
	}
	body := m.quickNoteBox()
	for _, l := range body {
		if w := lipgloss.Width(l); w > m.width {
			t.Fatalf("quick note line wider (%d) than terminal (%d): %q", w, m.width, l)
		}
	}
}

// Once the text field's wrapped lines outgrow its 8-row cap, the window
// must scroll to keep the cursor visible instead of freezing on the
// first 8 rows forever — the report was that the box "stopped growing"
// with the cursor's own line invisible and no way to scroll to it.
func TestQuickNoteBoxScrollsToKeepCursorVisibleOnceItStopsGrowing(t *testing.T) {
	m := newTestModel(t)
	press(m, "i")
	for i := 0; i < 12; i++ {
		if i > 0 {
			press(m, "shift+enter")
		}
		typeText(m, fmt.Sprintf("line%d", i))
	}
	body := m.quickNoteBox()
	text := ansi.Strip(strings.Join(body, "\n"))
	if !strings.Contains(text, "line11") {
		t.Errorf("cursor's own line (line11) not visible once the field outgrew its cap:\n%s", text)
	}
	if strings.Contains(text, "line0\n") || strings.Contains(text, "line0 ") {
		t.Errorf("expected the window to have scrolled past line0:\n%s", text)
	}
}

func TestBookCardFrameFillsTerminalExactly(t *testing.T) {
	m := newTestModel(t)
	press(m, "B")
	for _, size := range sizes {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		checkFrame(t, m, "Book Card open")
	}
}
