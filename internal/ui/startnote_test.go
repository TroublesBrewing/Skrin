package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lurioso/skrin/internal/config"
	"github.com/lurioso/skrin/internal/session"
	"github.com/lurioso/skrin/internal/version"
)

func startNoteModel(t *testing.T, rel string, extra func(*Options)) *Model {
	t.Helper()
	if !version.Beta {
		t.Skip("a release build leaves every experiment out")
	}
	opts := Options{RolloverTodos: true, Beta: true, Vault: config.VaultSettings{StartNote: rel}}
	if extra != nil {
		extra(&opts)
	}
	return newTestModelWith(t, opts)
}

func TestSkrinOpensOnTheStartNote(t *testing.T) {
	m := startNoteModel(t, "Filosofi/Stoic.md", nil)
	if m.notePath != "Filosofi/Stoic.md" {
		t.Errorf("opened %q", m.notePath)
	}
}

// The start note wins over the note the last run left open: you named
// this one, and it is the same every morning.
func TestTheStartNoteWinsOverTheLastNote(t *testing.T) {
	m := startNoteModel(t, "Filosofi/Stoic.md", func(o *Options) {
		o.RestoreLastNote = true
		o.Session = session.State{Open: "Welcome.md", Offset: 3}
	})
	if m.notePath != "Filosofi/Stoic.md" {
		t.Errorf("opened %q", m.notePath)
	}
}

// Unset, nothing changes: the last note still comes back when that is on.
func TestWithNoStartNoteTheLastNoteStillComesBack(t *testing.T) {
	m := startNoteModel(t, "", func(o *Options) {
		o.RestoreLastNote = true
		o.Session = session.State{Open: "Welcome.md"}
	})
	if m.notePath != "Welcome.md" {
		t.Errorf("opened %q", m.notePath)
	}
}

// A start note that has been deleted or renamed away is not an error: it
// counts as none, and the setting still says what it points at so it can
// be pointed somewhere else.
func TestAStartNoteThatIsGoneLandsOnTheWelcomeCard(t *testing.T) {
	m := startNoteModel(t, "Filosofi/Gone.md", nil)
	if m.notePath != "" {
		t.Errorf("opened %q", m.notePath)
	}
	if lbl := m.startNoteLabel(); !strings.Contains(lbl, "gone") {
		t.Errorf("label %q should own up to it", lbl)
	}
}

// Off, a start note left in the vault's settings is ignored entirely.
func TestTheStartNoteIsIgnoredUntilBetaIsOn(t *testing.T) {
	m := newTestModelWith(t, Options{RolloverTodos: true, Vault: config.VaultSettings{StartNote: "Filosofi/Stoic.md"}})
	if m.notePath != "" {
		t.Errorf("an experiment that is off may not choose where Skrin lands, opened %q", m.notePath)
	}
}

// Choosing one saves it with the vault, not with the machine, and
// clearing it puts the welcome card back.
func TestChoosingAStartNoteSavesItWithTheVault(t *testing.T) {
	m := startNoteModel(t, "", nil)
	m.setStartNote("Welcome.md")
	saved := config.LoadVaultSettings(m.vault.Root)
	if saved.StartNote != "Welcome.md" {
		t.Fatalf("vault settings hold %q", saved.StartNote)
	}
	if _, err := os.Stat(filepath.Join(m.vault.Root, config.SettingsFile)); err != nil {
		t.Errorf("the vault's own settings file: %v", err)
	}
	m.setStartNote("")
	if config.LoadVaultSettings(m.vault.Root).StartNote != "" {
		t.Error("clearing it should put the welcome card back")
	}
	if !strings.Contains(m.flash, "welcome card") {
		t.Errorf("flash %q", m.flash)
	}
}
