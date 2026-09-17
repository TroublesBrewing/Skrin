package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
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
	text := manualPlain(m)
	for _, b := range defaultBindings {
		if !strings.Contains(text, b.help) {
			t.Errorf("the manual lacks %q", b.help)
		}
	}
	for _, s := range []string{"Ctrl-d PgDn", "Space", "In the editor", "Search", "Links", "Undo", "rollover_todos"} {
		if !strings.Contains(text, s) {
			t.Errorf("the manual lacks %q", s)
		}
	}
	for _, size := range sizes {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		checkFrame(t, m, fmt.Sprintf("manual %dx%d", size[0], size[1]))
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
	press(m, "tab")
	if m.manual.tab != manualTabKeys {
		t.Fatal("tab again should return to Keys")
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
