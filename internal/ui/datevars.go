package ui

import (
	"fmt"
	"strings"

	"github.com/lurioso/skrin/internal/datevar"
	"github.com/lurioso/skrin/internal/obsidian"
	"github.com/lurioso/skrin/internal/version"
)

// dateVarsOn is the feature's real state: a beta feature needs the build
// to allow beta, beta mode to be on, and its own switch on.
func (m *Model) dateVarsOn() bool {
	return version.Beta && m.opts.Beta && m.opts.DateVars
}

// inTemplatesFolder reports whether rel is a template. A template holds
// {{date}} on purpose — that is the whole point of it — so filling the
// variable in while the template is being written would destroy the thing
// being written. The one exception that makes the feature safe.
func (m *Model) inTemplatesFolder(rel string) bool {
	folder, _ := m.templatesFolder()
	return folder != "" && strings.HasPrefix(rel, folder+"/")
}

// resolveDateVars turns {{date}} and its family into dates as the editor
// is left, and says what it did. It runs beside resolveDueDates and on
// the same terms: only lines written or changed during this edit, never
// inside code, and one step of undo takes it back.
func (m *Model) resolveDateVars() string {
	if !m.dateVarsOn() || m.inTemplatesFolder(m.edit.rel) {
		return ""
	}
	s := obsidian.LoadSettings(m.vault.Root).Templates
	text, changes := datevar.Resolve(m.editor.Text(), m.edit.opened, s.DateFormat, s.TimeFormat, m.opts.Now())
	if len(changes) == 0 {
		return ""
	}
	m.editor.Reset(text)
	if len(changes) == 1 {
		return " · " + changes[0].Var + " → " + changes[0].Date
	}
	var parts []string
	for _, c := range changes {
		parts = append(parts, c.Var+" → "+c.Date)
	}
	return fmt.Sprintf(" · %d date variables: %s", len(changes), strings.Join(parts, ", "))
}
