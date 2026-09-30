package ui

import (
	"strings"
	"testing"

	"github.com/lurioso/skrin/internal/version"
)

func dateVarModel(t *testing.T) *Model {
	t.Helper()
	if !version.Beta {
		t.Skip("a release build leaves every experiment out")
	}
	return newTestModelWith(t, Options{RolloverTodos: true, Beta: true, DateVars: true})
}

// Leaving the editor is what settles it, the same moment due:: dates are
// settled.
func TestDateVariableBecomesTheDateOnLeavingTheEditor(t *testing.T) {
	m := dateVarModel(t)
	inFilosofi(m)
	press(m, "j", "e")
	typeText(m, "written {{date}}")
	press(m, "esc")

	got, err := m.vault.Read("Filosofi/Stoic.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "{{date}}") {
		t.Errorf("the variable is still there: %q", firstLine(got))
	}
	if !strings.Contains(got, today.Format("2006-01-02")) {
		t.Errorf("no date written: %q", firstLine(got))
	}
	if !strings.Contains(m.flash, "{{date}} →") {
		t.Errorf("flash %q should say what changed", m.flash)
	}
}

// A template is the one note that means its variables literally.
func TestATemplateKeepsItsVariables(t *testing.T) {
	m := dateVarModel(t)
	m.opts.Vault.TemplatesFolder = "Templates" // as the Settings row sets it
	if !m.inTemplatesFolder("Templates/Daily template.md") {
		t.Fatal("the fixture's templates folder should be found")
	}
	m.open("Templates/Daily template.md")
	press(m, "e")
	typeText(m, "and {{date}} again")
	press(m, "esc")

	got, err := m.vault.Read("Templates/Daily template.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "and {{date}} again") {
		t.Errorf("a template's variables were filled in: %q", got)
	}
}

// Off, what you typed stays exactly as you typed it.
func TestDateVariablesAreLeftAloneUntilAskedFor(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "e")
	typeText(m, "written {{date}}")
	press(m, "esc")

	got, err := m.vault.Read("Filosofi/Stoic.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "{{date}}") {
		t.Errorf("an experiment that is off may not rewrite what was typed: %q", firstLine(got))
	}
}

// With no templates folder anywhere — not in Settings, not in Obsidian's
// own config — there is nothing to exempt, and nothing pretends there is.
func TestWithNoTemplatesFolderNothingIsExempt(t *testing.T) {
	m := dateVarModel(t)
	if m.inTemplatesFolder("Templates/Daily template.md") {
		t.Error("the fixture declares no templates folder")
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
