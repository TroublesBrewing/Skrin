package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/config"
)

// manualPlain is the manual as shown now, without colours.
func manualPlain(m *Model) string {
	var b strings.Builder
	for _, l := range m.manualLines(m.manualWidth()) {
		b.WriteString(l.plain + "\n")
	}
	return b.String()
}

func TestManualListsEveryBinding(t *testing.T) {
	m := newTestModel(t)
	press(m, "?")
	if m.manual == nil {
		t.Fatal("? should open the manual")
	}
	// The Keys tab carries every binding; the Guide carries the prose.
	keys := manualPlain(m)
	for _, b := range defaultBindings {
		if !strings.Contains(keys, b.help) {
			t.Errorf("the Keys tab lacks %q", b.help)
		}
	}
	for _, s := range []string{"Ctrl-d PgDn", "Space", "In the editor"} {
		if !strings.Contains(keys, s) {
			t.Errorf("the Keys tab lacks %q", s)
		}
	}
	for _, size := range sizes {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		checkFrame(t, m, fmt.Sprintf("keys tab %dx%d", size[0], size[1]))
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	press(m, "tab", "tab") // Keys → Settings → Guide
	if m.manual.tab != manualTabGuide {
		t.Fatalf("tab twice should reach the Guide, got %v", m.manual.tab)
	}
	guide := manualPlain(m)
	for _, s := range []string{"Search", "Links", "Undo", "rollover_todos"} {
		if !strings.Contains(guide, s) {
			t.Errorf("the Guide lacks %q", s)
		}
	}
	for _, size := range sizes {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		checkFrame(t, m, fmt.Sprintf("guide %dx%d", size[0], size[1]))
	}
	press(m, "?")
	if m.manual != nil {
		t.Error("? again should close it")
	}
}

func TestManualFilterAndScroll(t *testing.T) {
	m := newTestModel(t)
	press(m, "?", "/")
	typeText(m, "zen")
	text := manualPlain(m)
	if !strings.Contains(text, "zen mode") || strings.Contains(text, "rename") {
		t.Errorf("filtered manual:\n%s", text)
	}
	checkFrame(t, m, "filtered manual")
	press(m, "enter", "esc") // enter keeps the filter, esc clears it
	if m.manual == nil || m.manual.in.value() != "" {
		t.Fatal("esc should clear the filter first")
	}
	press(m, "G")
	if m.manual.off == 0 {
		t.Error("G should scroll to the end")
	}
	press(m, "esc")
	if m.manual != nil {
		t.Error("esc should close the manual")
	}
}

func TestManualSettingsTabTogglesAndSaves(t *testing.T) {
	m := newTestModel(t)
	if m.opts.Images {
		t.Fatal("test setup should start with images off, to check the toggle turns it on")
	}
	press(m, "?")
	if m.manual.tab != manualTabKeys {
		t.Fatal("? should open on the Keys tab")
	}
	press(m, "tab")
	if m.manual.tab != manualTabSettings {
		t.Fatal("tab should switch to Settings")
	}
	text := ansi.Strip(m.manualView())
	for _, s := range []string{"Remember last open note", "Carry over yesterday's todos", "Vim keys", "Claude drawer", "Image previews"} {
		if !strings.Contains(text, s) {
			t.Errorf("the Settings tab lacks %q:\n%s", s, text)
		}
	}
	items := settingsItems()
	var imagesRow int
	for i, it := range items {
		if it.label == "Image previews" {
			imagesRow = i
		}
	}
	m.manual.setCur = imagesRow
	press(m, "enter")
	if !m.opts.Images {
		t.Fatal("enter on Image previews should turn it on")
	}
	if m.opts.Config.Render.Images == nil || !*m.opts.Config.Render.Images {
		t.Fatal("toggling should also update the config, so it's saved")
	}
	saved, err := os.ReadFile(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "skrin", "config.toml"))
	if err != nil {
		t.Fatalf("settings should save to config.toml: %v", err)
	}
	if !strings.Contains(string(saved), "images = true") {
		t.Errorf("config.toml should record the toggle:\n%s", saved)
	}
	press(m, "shift+tab")
	if m.manual.tab != manualTabKeys {
		t.Fatal("shift+tab should step back to Keys")
	}
	press(m, "?")
	if m.manual != nil {
		t.Error("? should close the manual from the Keys tab")
	}
}

func TestZenShowsJustTheNote(t *testing.T) {
	m := newTestModel(t)
	press(m, "z")
	if m.zen || !strings.Contains(m.flash, "Select a note to read in zen mode") {
		t.Fatalf("zen %v with no note open, flash %q", m.zen, m.flash)
	}
	press(m, "G", "enter", "z")
	if !m.zen || m.layout().noteTextW() != zenWidth {
		t.Fatalf("zen %v, text width %d", m.zen, m.layout().noteTextW())
	}
	if frame := ansi.Strip(m.render()); strings.Contains(frame, "Files") || strings.Contains(frame, "VIEW") {
		t.Errorf("zen should hide Files and the status line:\n%s", frame)
	}
	for _, size := range sizes {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		checkFrame(t, m, fmt.Sprintf("zen %dx%d", size[0], size[1]))
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	press(m, "e")
	if m.editor == nil || !m.zen {
		t.Fatal("e in zen mode should open the editor and stay in zen")
	}
	typeText(m, " [[Sto")
	checkFrame(t, m, "zen editor with completion")
	press(m, "esc", "esc") // close the popup, then leave the editor
	if m.editor != nil || !m.zen {
		t.Fatalf("editor %v, zen %v: leaving the editor should stay in zen", m.editor != nil, m.zen)
	}
	press(m, "esc")
	if m.zen {
		t.Error("esc should leave zen mode")
	}
	press(m, "z", "tab")
	if m.zen || m.focus != paneFiles {
		t.Error("tab in zen mode should go to Files")
	}
}

// cursorTo puts the Keys tab's cursor on one binding, the way j/k would.
func cursorTo(t *testing.T, m *Model, where string, a action) {
	t.Helper()
	for _, b := range m.bindingRows() {
		if b.where == where && b.act == a {
			m.manual.cur = b
			m.showCursor()
			return
		}
	}
	t.Fatalf("%s is not a row of the Keys tab", actionName[a])
}

func TestKeysTabRebindsAKeyAndSavesIt(t *testing.T) {
	m := newTestModel(t)
	press(m, "?")
	cursorTo(t, m, inMain, actZen)
	press(m, "enter")
	if m.manual.mode != keysCapture {
		t.Fatal("enter should wait for the new key")
	}
	press(m, "ctrl+g")
	if m.manual.mode != keysBrowse {
		t.Fatalf("the key pressed should end the wait, mode = %v", m.manual.mode)
	}
	if m.keys.act(inMain, "ctrl+g") != actZen {
		t.Error("the new key should work")
	}
	if m.keys.act(inMain, "z") != actNone {
		t.Error("the old key should have stopped working")
	}
	saved, err := os.ReadFile(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "skrin", "config.toml"))
	if err != nil {
		t.Fatalf("a changed key should be saved: %v", err)
	}
	if !strings.Contains(string(saved), "ctrl+g") || !strings.Contains(string(saved), "zen") {
		t.Errorf("config.toml should record the new key:\n%s", saved)
	}
	// And it works for real, not just in the keymap.
	press(m, "esc")
	press(m, "G", "enter") // open a note, so zen has something to show
	press(m, "ctrl+g")
	if !m.zen {
		t.Error("the rebound key should actually do the thing")
	}
	press(m, "esc")
	press(m, "z")
	if m.zen {
		t.Error("the old key should do nothing now")
	}
}

func TestKeysTabAsksBeforeTakingAKeyFromAnother(t *testing.T) {
	m := newTestModel(t)
	press(m, "?")
	cursorTo(t, m, inMain, actZen)
	press(m, "enter")
	press(m, "e") // e is "edit the note in Skrin"
	if m.manual.mode != keysAsking {
		t.Fatalf("a key another binding has should ask first, mode = %v", m.manual.mode)
	}
	if !strings.Contains(m.manual.ask.text, "edit the note in Skrin") {
		t.Errorf("the question should name what has the key: %q", m.manual.ask.text)
	}
	press(m, "n")
	if m.keys.act(inMain, "e") != actEdit || m.keys.act(inMain, "z") != actZen {
		t.Fatal("n should leave both bindings as they were")
	}
	press(m, "enter", "e", "y")
	if m.keys.act(inMain, "e") != actZen {
		t.Error("y should hand the key over")
	}
	if len(m.keys.bound(inMain, actEdit)) != 0 {
		t.Error("the binding it was taken from should be left with no key")
	}
	// The cursor moves to the binding left without a key, so it can be
	// given one there and then.
	if m.manual.cur != (bindRef{inMain, actEdit}) {
		t.Errorf("cursor = %+v, want the binding left with no key", m.manual.cur)
	}
	press(m, "enter", "ctrl+e")
	if m.keys.act(inMain, "ctrl+e") != actEdit {
		t.Error("the displaced binding should take its new key right there")
	}
}

// A key the component types or handles itself can't be handed out, and
// saying which one has it beats a silent refusal.
func TestKeysTabRefusesAKeyTheComponentOwns(t *testing.T) {
	m := newTestModel(t)
	press(m, "?")
	cursorTo(t, m, inEditor, actAskClaude)
	press(m, "enter")
	press(m, "ctrl+s") // the editor's own save
	if m.manual.mode != keysCapture {
		t.Error("a refused key should leave you free to press another")
	}
	if !strings.Contains(m.manual.note, "save") {
		t.Errorf("the refusal should name what has the key, got %q", m.manual.note)
	}
	if m.keys.changed(inEditor, actAskClaude) {
		t.Error("nothing should have changed")
	}
	press(m, "alt+c")
	if m.keys.act(inEditor, "alt+c") != actAskClaude {
		t.Error("a free key should still be accepted after a refusal")
	}
}

// ? and quit are how you get back to this screen to undo a mistake, so
// they can't be left without a key from inside it.
func TestKeysTabWontLeaveTheManualUnreachable(t *testing.T) {
	m := newTestModel(t)
	press(m, "?")
	cursorTo(t, m, inMain, actZen)
	press(m, "enter", "?")
	if m.keys.act(inMain, "?") != actHelp {
		t.Error("? should still open the manual")
	}
	if !strings.Contains(m.manual.note, "way back") {
		t.Errorf("the refusal should say why, got %q", m.manual.note)
	}
}

func TestKeysTabPutsKeysBack(t *testing.T) {
	m := newTestModel(t)
	press(m, "?")
	cursorTo(t, m, inMain, actZen)
	press(m, "enter", "ctrl+g")
	press(m, "r")
	if m.keys.act(inMain, "z") != actZen || m.keys.act(inMain, "ctrl+g") != actNone {
		t.Fatal("r should put the one binding back")
	}
	// R asks first, then clears the lot.
	press(m, "enter", "ctrl+g")
	cursorTo(t, m, inMain, actDaily)
	press(m, "enter", "ctrl+t")
	press(m, "R")
	if m.manual.mode != keysAsking {
		t.Fatal("R should ask before forgetting changed keys")
	}
	press(m, "y")
	if m.keys.act(inMain, "z") != actZen || m.keys.act(inMain, "t") != actDaily {
		t.Error("y should put every key back")
	}
	if m.opts.Config.Keys != nil {
		t.Errorf("with nothing changed there should be no [keys] left to save, got %v", m.opts.Config.Keys)
	}
}

// The whole point of the report: finding one key among a hundred.
func TestKeysTabFindsByWhatItDoesAndByTheKey(t *testing.T) {
	m := newTestModel(t)
	press(m, "?", "/")
	typeText(m, "ctrl+k")
	text := manualPlain(m)
	if !strings.Contains(text, "ask Claude, with the selection") {
		t.Errorf("searching for the key itself should find its row:\n%s", text)
	}
	if strings.Contains(text, "rename") {
		t.Errorf("and only its row:\n%s", text)
	}
	press(m, "esc") // clear the filter
	press(m, "/")
	typeText(m, "trash")
	if text := manualPlain(m); !strings.Contains(text, "delete to the trash") {
		t.Errorf("searching for what it does should find it:\n%s", text)
	}
}

// A key changed in the Keys tab is still yours next time Skrin starts.
func TestChangedKeysComeBackFromTheConfig(t *testing.T) {
	m := newTestModelWith(t, Options{Config: config.Config{
		Keys: map[string]map[string][]string{inMain: {"zen": {"ctrl+g"}}},
	}})
	if m.keys.act(inMain, "ctrl+g") != actZen {
		t.Error("a saved key should be in force from the start")
	}
	press(m, "?")
	cursorTo(t, m, inMain, actZen)
	if text := manualPlain(m); !strings.Contains(text, "Ctrl-g") {
		t.Errorf("and the Keys tab should show it:\n%s", text)
	}
}

// A cursor you can't see is the trap the charter warns about, so the row
// the Keys tab is about to act on has to look different from the rest.
func TestKeysTabShowsWhichRowTheCursorIsOn(t *testing.T) {
	m := newTestModel(t)
	press(m, "?")
	styledFor := func(a action) string {
		t.Helper()
		for _, l := range m.manualLines(m.manualWidth()) {
			if l.bind != nil && l.bind.act == a && l.bind.where == inMain {
				return l.styled
			}
		}
		t.Fatalf("%s is not showing", actionName[a])
		return ""
	}
	first, second := styledFor(actDown), styledFor(actUp)
	if first == second {
		t.Fatal("two rows rendered identically: the cursor can't be seen")
	}
	if !strings.Contains(first, "48;2;") {
		t.Errorf("the cursor's row should be a highlighted bar, got %q", first)
	}
	press(m, "j")
	if styledFor(actDown) == first {
		t.Error("j should move the highlight off the row it was on")
	}
	if !strings.Contains(styledFor(actUp), "48;2;") {
		t.Error("j should move the highlight onto the next row")
	}
}
