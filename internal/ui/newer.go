package ui

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/lurioso/skrin/internal/version"
)

// exeCheckEvery is how often the program file is looked at. A stat of one
// file is nothing, and a minute is soon enough for a notice whose whole
// point is to be there the next time you glance down.
const exeCheckEvery = time.Minute

// exeCheckMsg asks for the program file to be looked at again.
type exeCheckMsg struct{}

// newerNoticeOn is the feature's real state: a beta feature needs the
// build to allow beta, beta mode to be on, and its own switch on.
func (m *Model) newerNoticeOn() bool {
	return version.Beta && m.opts.Beta && m.opts.NewerNotice
}

// exeStampNow describes the program file as it is on disk: its size and
// the time it was last written. Anything that can't be read comes back
// empty, which reads as "nothing to compare" rather than as a change —
// a Skrin that shouts because a stat failed would be worse than one that
// says nothing.
func (m *Model) exeStampNow() string {
	if m.opts.ExePath == "" {
		return ""
	}
	fi, err := os.Stat(m.opts.ExePath)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d·%d", fi.Size(), fi.ModTime().UnixNano())
}

// watchExe takes the stamp this run started from and starts the timer.
// It is armed even with the notice switched off, so switching it on in
// Settings compares against the file Skrin actually started from rather
// than against whatever happens to be there at that moment.
func (m *Model) watchExe() tea.Cmd {
	if m.opts.ExePath == "" {
		return nil
	}
	m.exeStamp = m.exeStampNow()
	if m.exeStamp == "" {
		return nil
	}
	return tea.Tick(exeCheckEvery, func(time.Time) tea.Msg { return exeCheckMsg{} })
}

// checkExe looks at the program file again. Once it has been replaced the
// notice stands until Skrin is restarted: the running program is the old
// one whatever happens next, and an installer that writes the file twice
// shouldn't make the notice flicker.
//
// The stamp is never updated to the new file. Restarting is what takes
// the notice away, because restarting is what makes it untrue.
func (m *Model) checkExe() tea.Cmd {
	if m.newerSkrin {
		return nil // said once, and true until the restart
	}
	if now := m.exeStampNow(); now != "" && now != m.exeStamp {
		m.newerSkrin = true
		return nil
	}
	return tea.Tick(exeCheckEvery, func(time.Time) tea.Msg { return exeCheckMsg{} })
}

// newerNotice is what the status line shows once a newer Skrin has been
// installed under the running one: the whole reason for the card is that
// a fix can sit installed for days without reaching the person it was
// made for.
func (m *Model) newerNotice() string {
	if !m.newerNoticeOn() || !m.newerSkrin {
		return ""
	}
	return "A newer Skrin is installed · restart to use it"
}
