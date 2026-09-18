package obsidian

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSettings(t *testing.T) {
	root := t.TempDir()
	write(t, root, ".obsidian/app.json", `{"trashOption":"local","alwaysUpdateLinks":true,"newLinkFormat":"absolute","newFileLocation":"folder","newFileFolderPath":"/Inbox/"}`)
	write(t, root, ".obsidian/daily-notes.json", `{"folder":"/Journal/Daily/","format":"DD.MM.YYYY","template":"Templates/Day"}`)
	write(t, root, ".obsidian/community-plugins.json", `["obsidian-rollover-daily-todos"]`)
	write(t, root, ".obsidian/plugins/obsidian-rollover-daily-todos/data.json",
		`{"templateHeading":"### Todo's","rolloverChildren":true,"removeEmptyTodos":true}`)

	s := LoadSettings(root)
	if s.TrashOption != "local" || !s.AlwaysUpdateLinks || s.NewLinkFormat != "absolute" ||
		s.NewFileLocation != "folder" || s.NewFileFolderPath != "Inbox" {
		t.Errorf("app settings = %+v", s)
	}
	if s.Daily != (DailyNotes{Folder: "Journal/Daily", Format: "DD.MM.YYYY", Template: "Templates/Day"}) {
		t.Errorf("daily notes = %+v", s.Daily)
	}
	want := Rollover{Installed: true, TemplateHeading: "### Todo's", RemoveEmptyTodos: true, RolloverChildren: true, DoneStatusMarkers: "xX-"}
	if s.Rollover != want {
		t.Errorf("rollover = %+v, want %+v", s.Rollover, want)
	}
}

func TestLoadSettingsDefaults(t *testing.T) {
	s := LoadSettings(t.TempDir())
	if s.TrashOption != "system" || s.Daily.Format != "YYYY-MM-DD" || s.Daily.Folder != "" ||
		s.NewLinkFormat != "shortest" || s.NewFileLocation != "root" || s.AlwaysUpdateLinks {
		t.Errorf("defaults = %+v", s)
	}
	if s.Rollover.Installed || s.Rollover.TemplateHeading != "none" {
		t.Errorf("rollover defaults = %+v", s.Rollover)
	}
}

func TestRegistry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "obsidian.json")
	if err := os.WriteFile(path, []byte(`{"vaults":{
		"a":{"path":"/vaults/recent","ts":300},
		"b":{"path":"/vaults/open","ts":100,"open":true}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := LoadRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if p, _ := r.Preferred(); p != "/vaults/open" {
		t.Errorf("preferred = %q, want the open vault", p)
	}
	if !r.IsOpen("/vaults/open/") || r.IsOpen("/vaults/recent") {
		t.Error("IsOpen wrong")
	}
	delete(r.Vaults, "b")
	if p, _ := r.Preferred(); p != "/vaults/recent" {
		t.Errorf("preferred without an open vault = %q", p)
	}
}

func TestRunning(t *testing.T) {
	proc := t.TempDir()
	write(t, proc, "100/comm", "bash\n")
	write(t, proc, "100/cmdline", "vim\x00/home/me/.config/obsidian/obsidian.json\x00")
	write(t, proc, "self/comm", "obsidian\n") // not a pid: ignored
	if running(proc) {
		t.Fatal("editing obsidian.json is not Obsidian running")
	}
	write(t, proc, "200/comm", "electron37\n")
	write(t, proc, "200/cmdline", "/usr/lib/electron37/electron\x00/usr/lib/obsidian/app.asar\x00")
	if !running(proc) {
		t.Fatal("Electron running obsidian/app.asar not detected")
	}
}

func TestLoadSettingsReadsTheTemplatesPlugin(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".obsidian"), 0o755)
	if s := LoadSettings(root).Templates; s.Folder != "" || s.DateFormat != "YYYY-MM-DD" || s.TimeFormat != "HH:mm" {
		t.Errorf("defaults: %+v", s)
	}
	os.WriteFile(filepath.Join(root, ".obsidian", "templates.json"), []byte(`{"folder":"/Meta/Templates/","dateFormat":"D MMM"}`), 0o644)
	if s := LoadSettings(root).Templates; s.Folder != "Meta/Templates" || s.DateFormat != "D MMM" || s.TimeFormat != "HH:mm" {
		t.Errorf("from templates.json: %+v", s)
	}
}
