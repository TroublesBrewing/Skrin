package vault

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ErrExists is returned when a create or move would overwrite something.
var ErrExists = errors.New("already exists")

// badChars are refused by Obsidian in file names, or break wikilinks.
const badChars = `\/:*?"<>|#^[]`

// CheckName validates one file or folder name typed by the user.
func CheckName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return errors.New("the name is empty")
	case name == "." || name == "..":
		return fmt.Errorf("%q isn't a usable name", name)
	case strings.HasPrefix(name, "."):
		return errors.New("names starting with . would be hidden")
	case strings.ContainsAny(name, badChars):
		return fmt.Errorf("names can't contain any of %s", badChars)
	}
	return nil
}

// Exists reports whether anything is at rel.
func (v *Vault) Exists(rel string) bool {
	_, err := os.Lstat(v.Abs(rel))
	return err == nil
}

// IsDir reports whether rel is a folder.
func (v *Vault) IsDir(rel string) bool {
	fi, err := os.Stat(v.Abs(rel))
	return err == nil && fi.IsDir()
}

// CreateFile writes a new file, failing with ErrExists if something is
// already there. Missing parent folders are created and returned outermost
// first, so an undo can remove them too.
func (v *Vault) CreateFile(rel, content string) ([]string, error) {
	dirs, err := v.mkdirs(parentOf(rel))
	if err != nil {
		return dirs, err
	}
	f, err := os.OpenFile(v.Abs(rel), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return dirs, fmt.Errorf("%s: %w", rel, ErrExists)
	}
	if err != nil {
		return dirs, err
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return dirs, err
	}
	return dirs, f.Close()
}

// CreateDir makes folder rel and any missing parents, returning the folders
// it created, outermost first.
func (v *Vault) CreateDir(rel string) ([]string, error) {
	if v.Exists(rel) {
		return nil, fmt.Errorf("%s: %w", rel, ErrExists)
	}
	return v.mkdirs(rel)
}

func (v *Vault) mkdirs(rel string) ([]string, error) {
	if rel == "" {
		return nil, nil
	}
	var created []string
	parts := strings.Split(rel, "/")
	for i := range parts {
		p := strings.Join(parts[:i+1], "/")
		err := os.Mkdir(v.Abs(p), 0o755)
		switch {
		case err == nil:
			created = append(created, p)
		case errors.Is(err, fs.ErrExist):
			if !v.IsDir(p) {
				return created, fmt.Errorf("%s is a file, not a folder", p)
			}
		default:
			return created, err
		}
	}
	return created, nil
}

// Move renames from to to, creating missing parent folders (returned as
// for CreateFile). It never overwrites, and won't put a folder inside
// itself.
func (v *Vault) Move(from, to string) ([]string, error) {
	if to == from {
		return nil, nil
	}
	if strings.HasPrefix(to, from+"/") {
		return nil, fmt.Errorf("can't move %s into itself", from)
	}
	if v.Exists(to) {
		return nil, fmt.Errorf("%s: %w", to, ErrExists)
	}
	dirs, err := v.mkdirs(parentOf(to))
	if err != nil {
		return dirs, err
	}
	return dirs, os.Rename(v.Abs(from), v.Abs(to))
}

// Write replaces a file's contents atomically (temp file, then rename),
// keeping its permissions.
func (v *Vault) Write(rel, content string) error {
	abs := v.Abs(rel)
	mode := fs.FileMode(0o644)
	if fi, err := os.Stat(abs); err == nil {
		mode = fi.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(abs), ".skrin-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), abs)
}

// Remove deletes a file or an empty folder for good. Like every delete, it
// refuses the vault root.
func (v *Vault) Remove(rel string) error {
	if rel == "" {
		return errors.New("refusing to delete the vault root")
	}
	return os.Remove(v.Abs(rel))
}

// RemoveAll deletes rel and everything under it for good. It is only used
// when Obsidian is set to delete without a trash.
func (v *Vault) RemoveAll(rel string) error {
	if rel == "" {
		return errors.New("refusing to delete the vault root")
	}
	return os.RemoveAll(v.Abs(rel))
}

// CountNotes counts the notes at or below rel.
func (v *Vault) CountNotes(rel string) int {
	root := v.Abs(rel)
	n := 0
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if p != root && Hidden(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() && IsNote(d.Name()) {
			n++
		}
		return nil
	})
	return n
}

func parentOf(rel string) string {
	if d := path.Dir(rel); d != "." {
		return d
	}
	return ""
}
