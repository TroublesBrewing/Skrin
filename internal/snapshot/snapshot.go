// Package snapshot keeps earlier versions of notes, so an edit can be
// undone (u) and redone (Ctrl-r), even after Skrin restarts. Versions live
// under $XDG_STATE_HOME/skrin/snapshots/<vault>/<note path>/{undo,redo}/,
// one file per version, named by its nanosecond timestamp.
package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// keep is how many undo versions are kept per note.
const keep = 20

// Snapshot is one saved version of a note.
type Snapshot struct {
	Content string
	Time    time.Time
}

// Store holds the snapshots of one vault.
type Store struct {
	dir  string
	root string
}

// Open returns the store for the vault at root. Directories are created
// when the first snapshot is saved.
func Open(root string) *Store {
	state := os.Getenv("XDG_STATE_HOME")
	if state == "" {
		home, _ := os.UserHomeDir()
		state = filepath.Join(home, ".local", "state")
	}
	sum := sha256.Sum256([]byte(root))
	return &Store{dir: filepath.Join(state, "skrin", "snapshots", hex.EncodeToString(sum[:6])), root: root}
}

func (s *Store) stack(rel, name string) string {
	return filepath.Join(s.dir, filepath.FromSlash(rel), name)
}

// Save records content as the version before an edit. It clears the redo
// history, since a new edit makes it meaningless.
func (s *Store) Save(rel, content string) error {
	if err := s.push(rel, "undo", content); err != nil {
		return err
	}
	if err := os.RemoveAll(s.stack(rel, "redo")); err != nil {
		return err
	}
	return s.prune(rel)
}

// Undo takes the newest saved version that differs from current, and
// remembers current so Redo can bring it back. It reports false when there
// is no earlier version.
func (s *Store) Undo(rel, current string) (Snapshot, bool, error) {
	snap, ok, err := s.pop(rel, "undo", current)
	if !ok || err != nil {
		return snap, ok, err
	}
	return snap, true, s.push(rel, "redo", current)
}

// Redo reverses the last Undo.
func (s *Store) Redo(rel, current string) (Snapshot, bool, error) {
	snap, ok, err := s.pop(rel, "redo", current)
	if !ok || err != nil {
		return snap, ok, err
	}
	if err := s.push(rel, "undo", current); err != nil {
		return snap, true, err
	}
	return snap, true, s.prune(rel)
}

// Move makes snapshots follow a note or folder that was renamed or moved.
func (s *Store) Move(from, to string) error {
	if from == "" || to == "" {
		return errors.New("snapshot: can't move the vault root")
	}
	src := filepath.Join(s.dir, filepath.FromSlash(from))
	if _, err := os.Stat(src); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	dst := filepath.Join(s.dir, filepath.FromSlash(to))
	// Anything already at the destination belonged to a note that is gone.
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

func (s *Store) push(rel, name, content string) error {
	dir := s.stack(rel, name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	// A note for humans wondering which vault this directory belongs to.
	_ = os.WriteFile(filepath.Join(s.dir, "vault.txt"), []byte(s.root+"\n"), 0o600)
	t := time.Now()
	for {
		p := filepath.Join(dir, fmt.Sprintf("%020d.md", t.UnixNano()))
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			t = t.Add(time.Nanosecond)
			continue
		}
		if err != nil {
			return err
		}
		// u has to work after a crash too, so the version is on the disk
		// before Save returns.
		_, err = f.WriteString(content)
		if err == nil {
			err = f.Sync()
		}
		return errors.Join(err, f.Close())
	}
}

// pop removes and returns the newest version in a stack, skipping versions
// identical to current.
func (s *Store) pop(rel, name, current string) (Snapshot, bool, error) {
	dir := s.stack(rel, name)
	names := versions(dir)
	for i := len(names) - 1; i >= 0; i-- {
		p := filepath.Join(dir, names[i])
		b, err := os.ReadFile(p)
		if err != nil {
			return Snapshot{}, false, err
		}
		if err := os.Remove(p); err != nil {
			return Snapshot{}, false, err
		}
		if string(b) != current {
			return Snapshot{Content: string(b), Time: stamp(names[i])}, true, nil
		}
	}
	return Snapshot{}, false, nil
}

func (s *Store) prune(rel string) error {
	dir := s.stack(rel, "undo")
	names := versions(dir)
	for _, n := range names[:max(len(names)-keep, 0)] {
		if err := os.Remove(filepath.Join(dir, n)); err != nil {
			return err
		}
	}
	return nil
}

// versions lists a stack's files, oldest first.
func versions(dir string) []string {
	entries, _ := os.ReadDir(dir) // sorted by name, which sorts by time
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			names = append(names, e.Name())
		}
	}
	return names
}

func stamp(name string) time.Time {
	n, _ := strconv.ParseInt(strings.TrimSuffix(name, ".md"), 10, 64)
	return time.Unix(0, n)
}
