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

	"github.com/lurioso/skrin/internal/config"
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
		return nil, fmt.Errorf("can't open the vault at %s: %w", abs, err)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", abs)
	}
	v := &Vault{Root: abs}
	v.migrateOrderFile()
	return v, nil
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

// MarkerDir is Skrin's marker inside a folder it has been opened as a
// vault from. Like Obsidian's .obsidian, it is what makes a folder
// recognisable as a Skrin to discovery; opening a folder writes it, so the
// folder shows up as a known vault next time. It is a dot-directory, so it
// never shows in the Files tree.
const MarkerDir = ".skrin"

// LooksLikeVault reports whether a folder is a vault: it carries Obsidian's
// .obsidian marker or Skrin's .skrin.
func LooksLikeVault(abs string) bool {
	for _, m := range []string{".obsidian", MarkerDir} {
		if fi, err := os.Stat(filepath.Join(abs, m)); err == nil && fi.IsDir() {
			return true
		}
	}
	return false
}

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

// FileInfo is a file's path, modification time and size.
type FileInfo struct {
	Rel     string
	ModTime time.Time
	Size    int64
}

// FileInfos lists every visible file with its time and size, in one walk.
func (v *Vault) FileInfos() ([]FileInfo, error) {
	var out []FileInfo
	err := v.walk(func(rel string, d fs.DirEntry) {
		if d.IsDir() {
			return
		}
		if info, err := d.Info(); err == nil {
			out = append(out, FileInfo{Rel: rel, ModTime: info.ModTime(), Size: info.Size()})
		}
	})
	return out, err
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
		if rel != "" && (Hidden(d.Name()) || rel == OrderFile || rel == config.SettingsFile) {
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
		if Hidden(d.Name()) || (rel == "" && (d.Name() == OrderFile || d.Name() == config.SettingsFile)) {
			continue
		}
		e := Entry{Name: d.Name(), Rel: path.Join(rel, d.Name()), IsDir: d.IsDir()}
		if info, err := d.Info(); err == nil {
			e.ModTime = info.ModTime()
		}
		entries = append(entries, e)
	}
	Sort(entries)
	return entries, nil
}

// Sort orders entries the way Obsidian's file explorer does: folders first,
// then files, each A→Z ignoring case.
func Sort(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
}

// Entries lists every visible folder and file below the root, with their
// modification times, in one walk.
func (v *Vault) Entries() ([]Entry, error) {
	var out []Entry
	err := v.walk(func(rel string, d fs.DirEntry) {
		if rel == "" {
			return
		}
		e := Entry{Name: d.Name(), Rel: rel, IsDir: d.IsDir()}
		if info, err := d.Info(); err == nil {
			e.ModTime = info.ModTime()
		}
		out = append(out, e)
	})
	return out, err
}

// Read returns a file's contents.
func (v *Vault) Read(rel string) (string, error) {
	b, err := os.ReadFile(v.Abs(rel))
	return string(b), err
}

// Watch calls onChange (debounced) whenever something visible in the vault
// changes on disk, whether from Skrin, Obsidian desktop, Sync or an editor.
// onTrouble is called when live updates can't be trusted any more: folders
// that couldn't be watched (the system's watch limit, on a big vault) or an
// error from the watcher itself. Going quiet without saying so is the one
// thing it mustn't do. Either callback may be nil.
func (v *Vault) Watch(ctx context.Context, onChange func(), onTrouble func(string)) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := w.Add(v.Root); err != nil {
		w.Close()
		return err
	}
	tell := func(s string) {
		if onTrouble != nil {
			onTrouble(s)
		}
	}
	if missed := watchTree(w, v.Root); missed > 0 {
		tell(unwatched(missed))
	}
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
						if missed := watchTree(w, ev.Name); missed > 0 {
							tell(unwatched(missed))
						}
					}
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(200*time.Millisecond, onChange)
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				tell("live updates: " + err.Error())
			}
		}
	}()
	return nil
}

// watchTree adds dir and every visible folder below it to the watcher, and
// reports how many it couldn't add. A folder created with subfolders
// already inside (mkdir -p, a move) is picked up whole this way.
func watchTree(w *fsnotify.Watcher, dir string) int {
	missed := 0
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if p != dir && Hidden(d.Name()) {
			return filepath.SkipDir
		}
		if w.Add(p) != nil {
			missed++
		}
		return nil
	})
	return missed
}

// unwatched says that live updates have holes in them, and why.
func unwatched(n int) string {
	what := "1 folder isn't"
	if n > 1 {
		what = fmt.Sprintf("%d folders aren't", n)
	}
	return "live updates: " + what + " being watched (the system's watch limit?) — changes there won't show by themselves"
}
