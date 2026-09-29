package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/version"
)

// fakeExe is a stand-in program file the test can replace under the
// running model, the way an install does.
func fakeExe(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "skrin")
	if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func newerModel(t *testing.T, exe string) *Model {
	t.Helper()
	if !version.Beta {
		t.Skip("a release build leaves every experiment out")
	}
	m := newTestModelWith(t, Options{RolloverTodos: true, Beta: true, NewerNotice: true, ExePath: exe})
	m.Init() // takes the stamp this run started from
	return m
}

func TestNewerSkrinIsNoticedAndSaidOnce(t *testing.T) {
	exe := fakeExe(t, "the old build")
	m := newerModel(t, exe)
	if m.exeStamp == "" {
		t.Fatal("no stamp taken at start")
	}
	if m.checkExe() == nil {
		t.Error("an unchanged file should keep the timer running")
	}
	if m.newerNotice() != "" {
		t.Error("nothing has been installed yet")
	}

	// Install a new one under the running Skrin.
	if err := os.WriteFile(exe, []byte("a whole new build, longer"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(exe, time.Now().Add(time.Minute), time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	m.checkExe()
	if !m.newerSkrin {
		t.Fatal("the replacement went unnoticed")
	}
	if !strings.Contains(m.newerNotice(), "restart") {
		t.Errorf("notice %q should say what to do about it", m.newerNotice())
	}
	if line := ansi.Strip(m.statusLine()); !strings.Contains(line, "A newer Skrin is installed") {
		t.Errorf("the status line should carry it: %q", line)
	}

	// It stands until the restart, and stops the timer: the running
	// program is the old one whatever is written next.
	if m.checkExe() != nil {
		t.Error("nothing more to watch once it has been said")
	}
	if m.newerNotice() == "" {
		t.Error("the notice should stand until Skrin is restarted")
	}
}

// A key's own answer still wins the line: the notice is standing there,
// not shouting over what you just did.
func TestAFlashStillWinsOverTheNotice(t *testing.T) {
	m := newerModel(t, fakeExe(t, "old"))
	m.newerSkrin = true
	press(m, "s") // an unbound key answers in the line
	if line := ansi.Strip(m.statusLine()); strings.Contains(line, "A newer Skrin") || !strings.Contains(line, "does nothing here") {
		t.Errorf("status line %q", line)
	}
}

// Off, nothing is said, though the stamp is still taken — switching the
// notice on later must compare against the file Skrin started from.
func TestTheNoticeSaysNothingUntilAskedFor(t *testing.T) {
	exe := fakeExe(t, "old")
	m := newTestModelWith(t, Options{RolloverTodos: true, ExePath: exe})
	m.Init()
	if m.exeStamp == "" {
		t.Error("the stamp should be taken whether or not the notice is on")
	}
	m.newerSkrin = true
	if m.newerNotice() != "" {
		t.Error("an experiment that is off may not speak")
	}
	if line := ansi.Strip(m.statusLine()); strings.Contains(line, "A newer Skrin") {
		t.Errorf("status line %q", line)
	}
}

// A program file that can't be read is not a new version. Skrin says
// nothing rather than crying wolf at a failed stat.
func TestAnUnreadableProgramFileSaysNothing(t *testing.T) {
	exe := fakeExe(t, "old")
	m := newerModel(t, exe)
	if err := os.Remove(exe); err != nil {
		t.Fatal(err)
	}
	m.checkExe()
	if m.newerSkrin {
		t.Error("a file that isn't there is not a newer Skrin")
	}
}
