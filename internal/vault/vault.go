// Package vault reads and watches an Obsidian vault on disk. Paths handed
// out are vault-relative with forward slashes; "" is the vault root.
package vault

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Vault is an Obsidian vault rooted at an absolute path.
type Vault struct {
	Root string
}

// Entry is one file or folder inside a folder listing.
type Entry struct {
	Name    string
	Rel     string
	IsDir   bool
	ModTime time.Time
}

// Open checks that root is a directory and returns the vault there.
func Open(root string) (*Vault, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", abs)
	}
	return &Vault{Root: abs}, nil
}

// Name is the vault's folder name, which is what Obsidian calls it too.
func (v *Vault) Name() string { return filepath.Base(v.Root) }

// Abs turns a vault-relative path into an absolute one.
func (v *Vault) Abs(rel string) string { return filepath.Join(v.Root, filepath.FromSlash(rel)) }

func (v *Vault) rel(abs string) string {
	r, err := filepath.Rel(v.Root, abs)
	if err != nil || r == "." {
		return ""
	}
	return filepath.ToSlash(r)
}

// Hidden reports whether an entry stays out of the UI: dot-entries such as
// .obsidian, .trash and .git.
func Hidden(name string) bool { return strings.HasPrefix(name, ".") }

// IsNote reports whether a file name is a markdown note.
func IsNote(name string) bool { return strings.EqualFold(filepath.Ext(name), ".md") }

// Dirs lists every visible folder, the root ("") included.
func (v *Vault) Dirs() ([]string, error) {
	var dirs []string
	err := v.walk(func(rel string, d fs.DirEntry) {
		if d.IsDir() {
			dirs = append(dirs, rel)
		}
	})
	return dirs, err
}

// Files lists every visible file.
func (v *Vault) Files() ([]string, error) {
	var files []string
	err := v.walk(func(rel string, d fs.DirEntry) {
		if !d.IsDir() {
			files = append(files, rel)
		}
	})
	return files, err
}

func (v *Vault) walk(fn func(rel string, d fs.DirEntry)) error {
	return filepath.WalkDir(v.Root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == v.Root {
				return err
			}
			// Skip what we can't read rather than failing the whole vault.
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel := v.rel(p)
		if rel != "" && Hidden(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		fn(rel, d)
		return nil
	})
}

// List returns a folder's visible entries: folders first, then files, each
// sorted case-insensitively.
func (v *Vault) List(rel string) ([]Entry, error) {
	des, err := os.ReadDir(v.Abs(rel))
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(des))
	for _, d := range des {
		if Hidden(d.Name()) {
			continue
		}
		e := Entry{Name: d.Name(), Rel: path.Join(rel, d.Name()), IsDir: d.IsDir()}
		if info, err := d.Info(); err == nil {
			e.ModTime = info.ModTime()
		}
		entries = append(entries, e)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}

// Read returns a file's contents.
func (v *Vault) Read(rel string) (string, error) {
	b, err := os.ReadFile(v.Abs(rel))
	return string(b), err
}

// Watch calls onChange (debounced) whenever something visible in the vault
// changes on disk, whether from Skrin, Obsidian desktop, Sync or an editor.
func (v *Vault) Watch(ctx context.Context, onChange func()) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := w.Add(v.Root); err != nil {
		w.Close()
		return err
	}
	watchTree(w, v.Root)
	go func() {
		defer w.Close()
		var timer *time.Timer
		for {
			select {
			case <-ctx.Done():
				if timer != nil {
					timer.Stop()
				}
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if Hidden(filepath.Base(ev.Name)) {
					continue
				}
				if ev.Has(fsnotify.Create) {
					if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
						watchTree(w, ev.Name)
					}
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(200*time.Millisecond, onChange)
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return nil
}

// watchTree adds dir and every visible folder below it to the watcher. A
// folder created with subfolders already inside (mkdir -p, a move) is
// picked up whole this way.
func watchTree(w *fsnotify.Watcher, dir string) {
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if p != dir && Hidden(d.Name()) {
			return filepath.SkipDir
		}
		_ = w.Add(p)
		return nil
	})
}
