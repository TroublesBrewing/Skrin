package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestApplyTemplateBodyAtTheCursor(t *testing.T) {
	got, row, col := applyTemplate("before after", 0, 7, "one\ntwo")
	if got != "before one\ntwoafter" || row != 1 || col != 3 {
		t.Errorf("got %q, cursor %d:%d", got, row, col)
	}
}

func TestApplyTemplateGivesANoteWithoutPropertiesTheTemplates(t *testing.T) {
	got, row, col := applyTemplate("# Note\n", 1, 0, "---\ntags: [book]\n---\nBody")
	want := "---\ntags: [book]\n---\n# Note\nBody"
	if got != want || row != 4 || col != 4 {
		t.Errorf("got %q, cursor %d:%d; want %q", got, row, col, want)
	}
}

func TestApplyTemplateMergesPropertiesTheNoteWins(t *testing.T) {
	note := "---\nstatus: reading\ntags:\n  - mine\n---\ntext\n"
	tmpl := "---\nstatus: to-read\nauthor:\nrating: \ntags:\n  - book\n---\n## Notes\n"
	got, row, _ := applyTemplate(note, 5, 4, tmpl)
	want := "---\nstatus: reading\ntags:\n  - mine\nauthor:\nrating: \n---\ntext## Notes\n\n"
	if got != want {
		t.Errorf("got\n%q\nwant\n%q", got, want)
	}
	if row != 8 {
		t.Errorf("the cursor should follow the text down past the two new properties: row %d", row)
	}
}

// withTemplatesPlugin points Obsidian's Templates plugin at folder.
func withTemplatesPlugin(t *testing.T, m *Model, folder string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(m.vault.Root, ".obsidian", "templates.json"), []byte(`{"folder":"`+folder+`","dateFormat":"YYYY-MM-DD"}`), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInsertTemplateFromTheEditor(t *testing.T) {
	m := newTestModel(t)
	withTemplatesPlugin(t, m, "Templates")
	press(m, "G", "enter") // Welcome, editing, at the top
	runCommand(m, "insert template")
	if m.chooser == nil || m.chooser.title != "Insert template" {
		t.Fatal("Insert template should list the templates")
	}
	if got := paletteLabels(m); len(got) != 1 || got[0] != "Daily template" {
		t.Errorf("templates listed: %v", got)
	}
	checkFrame(t, m, "template list over the editor")
	press(m, "enter")
	if !strings.Contains(m.editor.Text(), "### Todo's") || strings.Contains(m.editor.Text(), "{{") {
		t.Errorf("the template should go in, filled in:\n%s", m.editor.Text())
	}
	if !strings.HasPrefix(m.flash, "Inserted Daily template") {
		t.Errorf("flash %q", m.flash)
	}
	press(m, "ctrl+z")
	if strings.Contains(m.editor.Text(), "### Todo's") {
		t.Error("one ctrl+z should take the whole template back out")
	}
}

func TestInsertTemplateFromReadingOpensTheEditor(t *testing.T) {
	m := newTestModel(t)
	withTemplatesPlugin(t, m, "Templates")
	press(m, "G", "l")
	runCommand(m, "insert template")
	if m.editor == nil || m.chooser == nil {
		t.Fatalf("editor %v, list %v", m.editor != nil, m.chooser != nil)
	}
}

func TestInsertTemplateWithoutAFolderSaysWhere(t *testing.T) {
	m := newTestModel(t)
	press(m, "G", "enter")
	runCommand(m, "insert template")
	if m.chooser != nil || !strings.HasPrefix(m.flash, "No templates folder yet: choose one in Settings") {
		t.Errorf("list %v, flash %q", m.chooser != nil, m.flash)
	}
}

func TestChoosingTheTemplatesFolderInSettings(t *testing.T) {
	m := newTestModel(t)
	press(m, "?", "tab")
	for m.manual.setCur < len(settingsItems()) && settingsItems()[m.manual.setCur].label != "Templates folder" {
		press(m, "j")
	}
	if frame := ansi.Strip(m.render()); !strings.Contains(frame, "Templates folder: none yet") {
		t.Errorf("the row should say nothing's chosen:\n%s", frame)
	}
	press(m, "enter")
	if m.chooser == nil || m.chooser.title != "Templates folder" {
		t.Fatal("enter on the row should list the folders")
	}
	typeText(m, "Filosofi/Antik")
	press(m, "enter")
	if m.manual == nil || m.manual.tab != manualTabSettings || settingsItems()[m.manual.setCur].label != "Templates folder" {
		t.Fatal("choosing should come back to the Settings tab, on the same row")
	}
	if m.opts.Config.Templates.Folder != "Filosofi/Antik" || m.flash != "Templates folder: Filosofi/Antik" {
		t.Errorf("folder %q, flash %q", m.opts.Config.Templates.Folder, m.flash)
	}
	b, _ := os.ReadFile(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "skrin", "config.toml"))
	if !strings.Contains(string(b), `folder = "Filosofi/Antik"`) {
		t.Errorf("saved to config.toml: %q", b)
	}
	if frame := ansi.Strip(m.render()); !strings.Contains(frame, "Templates folder: Filosofi/Antik (chosen in Settings)") {
		t.Errorf("the row should show the choice:\n%s", frame)
	}
	press(m, "enter", "esc")
	if m.manual == nil || m.manual.tab != manualTabSettings {
		t.Error("esc from the folder list should go back to Settings")
	}
}

func TestAChosenTemplatesFolderWinsOverObsidians(t *testing.T) {
	m := newTestModel(t)
	withTemplatesPlugin(t, m, "Templates")
	m.opts.Config.Templates.Folder = "Filosofi"
	if f, whose := m.templatesFolder(); f != "Filosofi" || whose != "chosen in Settings" {
		t.Errorf("%q, %q", f, whose)
	}
	m.opts.Config.Templates.Folder = "Gone"
	if f, whose := m.templatesFolder(); f != "Templates" || whose != "Obsidian's" {
		t.Errorf("a chosen folder that's gone falls back to Obsidian's: %q, %q", f, whose)
	}
}
