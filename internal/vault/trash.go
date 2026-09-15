package vault

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Trashed records where a trashed file or folder went, so it can come back.
type Trashed struct {
	Rel    string // original vault-relative path
	Stored string // absolute path inside the trash
	Info   string // freedesktop .trashinfo file; "" for the vault's .trash
}

// Trash moves rel to the trash named by Obsidian's trashOption: "local" is
// the vault's own .trash folder, anything else the system (freedesktop)
// trash. If the system trash is on another filesystem, .trash is used.
func (v *Vault) Trash(rel, option string) (Trashed, error) {
	if rel == "" {
		return Trashed{}, errors.New("refusing to trash the vault root")
	}
	if option != "local" {
		t, err := v.trashSystem(rel)
		if !errors.Is(err, syscall.EXDEV) {
			return t, err
		}
	}
	return v.trashLocal(rel)
}

// trashSystem follows the freedesktop.org trash spec: reserve a name by
// creating info/<name>.trashinfo, then move the file to files/<name>.
func (v *Vault) trashSystem(rel string) (Trashed, error) {
	dir := systemTrash()
	for _, sub := range []string{"files", "info"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o700); err != nil {
			return Trashed{}, err
		}
	}
	abs := v.Abs(rel)
	for n := 1; ; n++ {
		name := numbered(filepath.Base(abs), n)
		info := filepath.Join(dir, "info", name+".trashinfo")
		f, err := os.OpenFile(info, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return Trashed{}, err
		}
		_, err = fmt.Fprintf(f, "[Trash Info]\nPath=%s\nDeletionDate=%s\n",
			(&url.URL{Path: abs}).EscapedPath(), time.Now().Format("2006-01-02T15:04:05"))
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			os.Remove(info)
			return Trashed{}, err
		}
		stored := filepath.Join(dir, "files", name)
		if _, err := os.Lstat(stored); err == nil {
			os.Remove(info) // a leftover without an info file; pick another name
			continue
		}
		if err := os.Rename(abs, stored); err != nil {
			os.Remove(info)
			return Trashed{}, err
		}
		return Trashed{Rel: rel, Stored: stored, Info: info}, nil
	}
}

// trashLocal moves rel into the vault's .trash folder, as Obsidian does
// with the "Move to Obsidian trash" option.
func (v *Vault) trashLocal(rel string) (Trashed, error) {
	dir := filepath.Join(v.Root, ".trash")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Trashed{}, err
	}
	for n := 1; ; n++ {
		stored := filepath.Join(dir, numbered(filepath.Base(rel), n))
		if _, err := os.Lstat(stored); err == nil {
			continue
		}
		if err := os.Rename(v.Abs(rel), stored); err != nil {
			return Trashed{}, err
		}
		return Trashed{Rel: rel, Stored: stored}, nil
	}
}

// Restore puts a trashed item back where it was.
func (v *Vault) Restore(t Trashed) error {
	if v.Exists(t.Rel) {
		return fmt.Errorf("can't restore %s: %w", t.Rel, ErrExists)
	}
	if _, err := v.mkdirs(parentOf(t.Rel)); err != nil {
		return err
	}
	if err := os.Rename(t.Stored, v.Abs(t.Rel)); err != nil {
		return err
	}
	if t.Info != "" {
		os.Remove(t.Info)
	}
	return nil
}

func systemTrash() string {
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		home, _ := os.UserHomeDir()
		data = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(data, "Trash")
}

// numbered returns name for n == 1, else "name 2.md", "name 3.md", ...
func numbered(name string, n int) string {
	if n == 1 {
		return name
	}
	ext := filepath.Ext(name)
	return fmt.Sprintf("%s %d%s", strings.TrimSuffix(name, ext), n, ext)
}
