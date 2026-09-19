package ui

import (
	"strings"
	"testing"

	"github.com/lurioso/skrin/internal/session"
)

func sessionWith(recent ...string) session.State { return session.State{Recent: recent} }

// switcherRows opens Go to note and lists its rows as "label — detail".
func switcherRows(m *Model) []string {
	press(m, "g")
	var out []string
	for _, i := range m.chooser.matches {
		it := m.chooser.items[i]
		out = append(out, it.label+" — "+it.detail)
	}
	press(m, "esc")
	return out
}

func TestGoToNoteOffersTheNotesOpenedLately(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "l") // Filosofi/Stoic.md, opened for reading
	press(m, "h", "j") // down to Zeno under Antik: the cursor only passes it
	press(m, "G", "enter")
	press(m, "esc") // Welcome.md, opened straight into editing
	rows := switcherRows(m)
	if len(rows) < 3 {
		t.Fatalf("too few rows: %v", rows)
	}
	if !strings.HasPrefix(rows[0], "Welcome — open now") {
		t.Errorf("the open note stays first: %q", rows[0])
	}
	if !strings.HasPrefix(rows[1], "Stoic — opened lately") {
		t.Errorf("the note opened before it comes next: %v", rows[:3])
	}
	for _, r := range rows[2:] {
		if strings.HasPrefix(r, "Zeno") && strings.Contains(r, "opened lately") {
			t.Error("a note the cursor only passed shouldn't count as opened")
		}
	}
}

func TestNotesOpenedLatelyAreNewestFirstWithoutRepeats(t *testing.T) {
	m := newTestModel(t)
	m.remember("a.md")
	m.remember("b.md")
	m.remember("a.md")
	if got := strings.Join(m.recentNotes, " "); got != "a.md b.md" {
		t.Errorf("newest first, each note once: %q", got)
	}
	for i := range recentNotes + 5 {
		m.remember(string(rune('a'+i%26)) + ".md")
	}
	if len(m.recentNotes) != recentNotes {
		t.Errorf("the list stops at %d: %d", recentNotes, len(m.recentNotes))
	}
}

func TestNotesOpenedLatelySurviveARestart(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l") // Welcome, opened
	if len(m.Session().Recent) == 0 {
		t.Fatal("what was opened should go into the session")
	}
	again := newTestModelWith(t, Options{Session: m.Session()})
	if len(again.recentNotes) == 0 || again.recentNotes[0] != "Welcome.md" {
		t.Errorf("the next run should start with the same list: %v", again.recentNotes)
	}
	gone := newTestModelWith(t, Options{Session: sessionWith("Gone.md")})
	if len(gone.recentNotes) != 0 {
		t.Errorf("a note that's since gone is left out: %v", gone.recentNotes)
	}
}

func TestOpenedLatelyRowReadsWellInTheVaultRoot(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "l") // Welcome.md, in the vault root
	for _, r := range switcherRows(m) {
		if strings.HasPrefix(r, "Welcome") && strings.HasSuffix(strings.TrimSpace(r), "·") {
			t.Errorf("a note in the root shouldn't end with a dangling separator: %q", r)
		}
	}
}
