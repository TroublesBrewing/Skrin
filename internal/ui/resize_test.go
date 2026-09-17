package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Ctrl+- and Ctrl++ never reach Skrin: the terminal keeps them, changes
// its own font, and Skrin is told the new size. So the size is the zoom
// indicator, and showing it is how a zoom visibly lands.
func TestResizeShowsTheNewSize(t *testing.T) {
	m := newTestModel(t) // sized once already, at 120×40
	if m.flash != "" {
		t.Fatalf("the first size is not a change: flash %q", m.flash)
	}
	_, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if !strings.Contains(m.flash, "100 × 30") {
		t.Errorf("a resize should show the new size, got %q", m.flash)
	}
	if cmd == nil {
		t.Error("the indicator should bring its own way of going away")
	}
	if !strings.Contains(ansi.Strip(m.render()), "100 × 30") {
		t.Error("and it should actually be on screen")
	}
	// The terminal re-reporting the size it already had is not a change,
	// so it must not flash at something the user didn't do.
	m.flash = ""
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.flash != "" {
		t.Errorf("the same size again should say nothing, got %q", m.flash)
	}
}

func TestTheSizeIndicatorTakesItselfAway(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	shown := m.flash
	if shown == "" {
		t.Fatal("no indicator to clear")
	}
	m.Update(flashDoneMsg{shown})
	if m.flash != "" {
		t.Errorf("the indicator should go by itself, still showing %q", m.flash)
	}
}

// Every flash shares one status line, so a timer started for one message
// must never carry off the message that replaced it.
func TestAnOldTimerLeavesANewerMessageAlone(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	stale := m.flash
	m.flash = "Saved."
	m.Update(flashDoneMsg{stale})
	if m.flash != "Saved." {
		t.Errorf("an old timer took a newer message: flash is now %q", m.flash)
	}
}

// Zooming in far enough takes Files off the screen. That looks like a
// fault unless the thing that caused it says so.
func TestResizeSaysWhenFilesWillHide(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 70, Height: 30})
	if !strings.Contains(m.flash, "70 × 30") || !strings.Contains(m.flash, "Files hides") {
		t.Errorf("under 80 columns it should say what that costs, got %q", m.flash)
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if !strings.Contains(m.flash, "120 × 40") {
		t.Fatalf("flash = %q", m.flash)
	}
	if strings.Contains(m.flash, "Files hides") {
		t.Errorf("back above 80 columns it shouldn't, got %q", m.flash)
	}
}

// Zen mode shows a status line only while there is a flash. An indicator
// that waited for a keypress would leave zen with a bar it never asked
// for, which is exactly what zen is for not having.
func TestZenGetsItsCleanScreenBackAfterAResize(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "enter", "z")
	if !m.zen {
		t.Fatal("should be in zen mode")
	}
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if !strings.Contains(ansi.Strip(m.render()), "100 × 30") {
		t.Error("the indicator should show in zen mode too: you zoom there as well")
	}
	m.Update(flashDoneMsg{m.flash})
	frame := ansi.Strip(m.render())
	if strings.Contains(frame, "100 × 30") || strings.Contains(frame, "VIEW") {
		t.Errorf("zen should be just the note again:\n%s", frame)
	}
	checkFrame(t, m, "zen after the indicator went")
}
