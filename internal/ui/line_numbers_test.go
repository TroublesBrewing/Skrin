package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestLineNumbersToggleInMain(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l") // open Welcome.md
	if m.opts.LineNumbers {
		t.Fatal("line numbers should be off by default")
	}

	// Press L to toggle on
	press(m, "L")
	if !m.opts.LineNumbers {
		t.Fatal("L should turn line numbers on")
	}
	checkFrame(t, m, "reading view with line numbers")

	// Verify rendered view contains gutter
	rendered := ansi.Strip(m.render())
	if !strings.Contains(rendered, " 1 │ ") {
		t.Errorf("rendered view should contain line 1 gutter, got:\n%s", rendered)
	}

	// Open editor and verify editor also has line numbers on
	press(m, "e")
	if m.editor == nil || !m.editor.LineNumbers() {
		t.Fatal("editor should have line numbers enabled")
	}
	checkFrame(t, m, "editor view with line numbers")

	// Leave editor
	press(m, "esc")

	// Toggle off
	press(m, "L")
	if m.opts.LineNumbers {
		t.Fatal("L should turn line numbers off")
	}
	checkFrame(t, m, "reading view without line numbers")
}

func TestLineNumbersInSplitView(t *testing.T) {
	m := newTestModel(t)
	press(m, "G")     // Welcome.md opens under the cursor, focus stays in Files
	press(m, "alt+k") // Templates/, a folder: moves only
	press(m, "alt+k") // Filosofi/
	press(m, "alt+k") // Daily/
	press(m, "l")     // opens Daily/, cursor stays
	press(m, "alt+j") // onto Daily/2026-09-11.md: the split opens on it
	if m.split == nil || m.split.path != "Daily/2026-09-11.md" {
		t.Fatalf("expected a split open on Daily/2026-09-11.md, got %+v", m.split)
	}
	press(m, "L") // toggle line numbers

	checkFrame(t, m, "split view with line numbers")

	rendered := ansi.Strip(m.render())
	if !strings.Contains(rendered, " 1 │ ") {
		t.Errorf("split view should contain line numbers, got:\n%s", rendered)
	}
}

func TestLineNumbersInZenMode(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l") // open Welcome.md
	press(m, "L")      // line numbers on
	press(m, "z")      // zen mode
	if !m.zen {
		t.Fatal("z should enter zen mode")
	}

	checkFrame(t, m, "zen view with line numbers")

	rendered := ansi.Strip(m.render())
	if !strings.Contains(rendered, " 1 │ ") {
		t.Errorf("zen view should contain line numbers, got:\n%s", rendered)
	}

	// Open editor in zen mode
	press(m, "e")
	checkFrame(t, m, "zen editor with line numbers")
	press(m, "esc")
	press(m, "z") // leave zen
}

func TestLineNumbersSettingsToggle(t *testing.T) {
	m := newTestModel(t)
	if m.opts.LineNumbers {
		t.Fatal("line numbers should default to off")
	}
	press(m, "?", "tab") // open manual -> Settings tab
	if m.manual == nil || m.manual.tab != manualTabSettings {
		t.Fatal("expected settings tab open")
	}

	items := m.settingsItems()
	var row int
	for i, it := range items {
		if it.label == "Line numbers in notes" {
			row = i
		}
	}
	m.manual.setCur = row
	press(m, "enter")
	if !m.opts.LineNumbers {
		t.Fatal("enter on Line numbers in notes should turn it on")
	}
	if !m.opts.Config.RenderLineNumbers() {
		t.Fatal("config should reflect line numbers enabled")
	}

	press(m, "enter")
	if m.opts.LineNumbers {
		t.Fatal("second enter should turn line numbers off")
	}
}
