package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/lurioso/skrin/internal/config"
)

// A folder→template pairing, the way Settings writes it.
func folderTemplate(folder, tmpl string) config.TemplateRule {
	return config.TemplateRule{Folder: folder, Template: tmpl}
}

func TestNewNoteInAPairedFolderGetsItsTemplate(t *testing.T) {
	m := newTestModelWith(t, Options{
		RolloverTodos:   true,
		FolderTemplates: []config.TemplateRule{folderTemplate("Filosofi", "Templates/Daily template.md")},
	})
	press(m, "1", "j", "j", "l") // into Filosofi, cursor on Antik/
	press(m, "n")
	typeText(m, "Ny tanke")
	press(m, "enter")
	if !m.vault.Exists("Filosofi/Ny tanke.md") {
		t.Fatal("note not created")
	}
	b, err := os.ReadFile(m.vault.Abs("Filosofi/Ny tanke.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "### Todo's") {
		t.Errorf("the note should start from the paired template:\n%s", b)
	}
	if strings.Contains(string(b), "{{") {
		t.Errorf("template variables should be filled in:\n%s", b)
	}
	if m.flash != "Created Filosofi/Ny tanke.md from its folder's template" {
		t.Errorf("flash %q", m.flash)
	}
	// U undoes the whole create in one step, template and all.
	press(m, "esc", "U")
	if m.vault.Exists("Filosofi/Ny tanke.md") {
		t.Error("U should undo the templated create too")
	}
}

func TestNoRuleLeavesANewNoteEmpty(t *testing.T) {
	m := newTestModel(t)
	inFilosofi(m)
	press(m, "j", "n")
	typeText(m, "Tom")
	press(m, "enter")
	b, _ := os.ReadFile(m.vault.Abs("Filosofi/Tom.md"))
	if string(b) != "" {
		t.Errorf("no pairing, so the note is empty: %q", b)
	}
	if m.flash != "Created Filosofi/Tom.md" {
		t.Errorf("flash %q", m.flash)
	}
}

func TestFolderTemplatesMatchExactlyNoInheritance(t *testing.T) {
	m := newTestModelWith(t, Options{
		RolloverTodos:   true,
		FolderTemplates: []config.TemplateRule{folderTemplate("Filosofi", "Templates/Daily template.md")},
	})
	// A note in a subfolder has no rule of its own: the parent's rule
	// must not leak down.
	press(m, "1", "j", "j", "l", "j") // Filosofi/Antik/
	press(m, "n")
	typeText(m, "Zenons efterträdare")
	press(m, "enter")
	b, _ := os.ReadFile(m.vault.Abs("Filosofi/Antik/Zenons efterträdare.md"))
	if string(b) != "" {
		t.Errorf("a subfolder with no rule is empty, the parent's rule doesn't reach down: %q", b)
	}
}

func TestAFolderPairingIsSavedAndCanBeRemoved(t *testing.T) {
	m := newTestModelWith(t, Options{
		RolloverTodos:   true,
		FolderTemplates: []config.TemplateRule{folderTemplate("Filosofi", "Templates/Daily template.md")},
	})
	press(m, "?", "tab") // Settings
	for m.manual.setCur < len(m.settingsItems()) && m.settingsItems()[m.manual.setCur].label != "Folder templates" {
		press(m, "j")
	}
	if frame := ansi.Strip(m.render()); !strings.Contains(frame, "Folder templates: 1 folder") {
		t.Errorf("the row should say one folder is paired:\n%s", frame)
	}
	press(m, "enter") // the list of pairs
	if m.chooser == nil || m.chooser.title != "Folder templates" {
		t.Fatal("enter should open the pairs list")
	}
	labels := paletteLabels(m)
	if len(labels) != 2 || labels[0] != "+ Add a folder…" {
		t.Fatalf("the list should lead with add, then the pair: %v", labels)
	}
	// d on the pair removes it, saving back to config.
	press(m, "down") // off "add", onto the pair
	press(m, "d")
	if len(m.opts.FolderTemplates) != 0 {
		t.Fatalf("d should remove the pair: %v", m.opts.FolderTemplates)
	}
	b, _ := os.ReadFile(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "skrin", "config.toml"))
	if strings.Contains(string(b), "Filosofi") {
		t.Errorf("the removal should be saved to config.toml: %q", b)
	}
}

func TestAddingAFolderTemplateWalksFolderThenTemplate(t *testing.T) {
	m := newTestModelWith(t, Options{RolloverTodos: true})
	withTemplatesPlugin(t, m, "Templates")
	press(m, "?", "tab")
	for m.manual.setCur < len(m.settingsItems()) && m.settingsItems()[m.manual.setCur].label != "Folder templates" {
		press(m, "j")
	}
	press(m, "enter") // pairs list, empty
	press(m, "enter") // "+ Add a folder…"
	if m.chooser == nil || m.chooser.title != "Template for new notes in…" {
		t.Fatal("add should list the folders")
	}
	typeText(m, "Filosofi")
	press(m, "enter") // pick Filosofi → templates list
	if m.chooser == nil || m.chooser.title != "Template for Filosofi/" {
		t.Fatal("picking a folder should list its templates")
	}
	press(m, "enter") // the first template
	if len(m.opts.FolderTemplates) != 1 {
		t.Fatalf("one pairing expected: %v", m.opts.FolderTemplates)
	}
	got := m.opts.FolderTemplates[0]
	if got.Folder != "Filosofi" || got.Template != "Templates/Daily template.md" {
		t.Errorf("pair = %+v", got)
	}
	b, _ := os.ReadFile(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "skrin", "config.toml"))
	if !strings.Contains(string(b), "Filosofi") {
		t.Errorf("the pairing should be saved to config.toml: %q", b)
	}
}
