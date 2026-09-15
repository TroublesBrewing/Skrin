package vault

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// StepKind says what a journal step did.
type StepKind int

const (
	StepCreated  StepKind = iota // Rel was created (file or folder)
	StepMoved                    // From was moved (or renamed) to Rel
	StepTrashed                  // Rel went to the trash; Trash says where
	StepModified                 // Rel's contents were replaced; Content has the old text
)

// Step is one reversible change.
type Step struct {
	Kind    StepKind
	Rel     string
	From    string
	Trash   Trashed
	Content string // StepCreated: a new file's initial text; StepModified: the old text
}

// Op is one user action, possibly many steps (a bulk move, a note created
// inside new folders). U undoes a whole Op.
type Op struct {
	Desc  string
	Steps []Step
}

// Journal is the session's stack of undoable operations.
type Journal struct {
	ops []Op
	// Keep, when set, saves a note's current text before an undo writes
	// over it, so a change made on disk in between can still be brought
	// back; Skrin hands it the snapshot store. If Keep fails, the note is
	// left as it is.
	Keep func(rel, disk string) error
}

const maxOps = 200

// Record pushes op; operations without steps are ignored.
func (j *Journal) Record(op Op) {
	if len(op.Steps) == 0 {
		return
	}
	j.ops = append(j.ops, op)
	if len(j.ops) > maxOps {
		j.ops = j.ops[len(j.ops)-maxOps:]
	}
}

// Len is the number of operations that can be undone.
func (j *Journal) Len() int { return len(j.ops) }

// Undo reverts the most recent operation, its steps in reverse order. It
// reports false when there is nothing to undo. A step that fails doesn't
// stop the others; all failures are returned together.
func (j *Journal) Undo(v *Vault, trashOption string) (Op, bool, error) {
	if len(j.ops) == 0 {
		return Op{}, false, nil
	}
	op := j.ops[len(j.ops)-1]
	j.ops = j.ops[:len(j.ops)-1]
	var errs []error
	for i := len(op.Steps) - 1; i >= 0; i-- {
		if err := undoStep(v, op.Steps[i], trashOption, j.Keep); err != nil {
			errs = append(errs, err)
		}
	}
	return op, true, errors.Join(errs...)
}

func undoStep(v *Vault, s Step, trashOption string, keep func(rel, disk string) error) error {
	switch s.Kind {
	case StepCreated:
		fi, err := os.Stat(v.Abs(s.Rel))
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		// Anything that gained content since it was created goes to the
		// trash instead of being deleted, so nothing written is ever lost.
		if fi.IsDir() {
			if v.Remove(s.Rel) == nil {
				return nil
			}
		} else if cur, err := v.Read(s.Rel); err == nil && cur == s.Content {
			return v.Remove(s.Rel)
		}
		_, err = v.Trash(s.Rel, trashOption)
		return err
	case StepMoved:
		_, err := v.Move(s.Rel, s.From)
		return err
	case StepTrashed:
		return v.Restore(s.Trash)
	case StepModified:
		if cur, err := v.Read(s.Rel); keep != nil && err == nil && cur != s.Content {
			if err := keep(s.Rel, cur); err != nil {
				return fmt.Errorf("left %s as it is: couldn't keep its current text first: %w", s.Rel, err)
			}
		}
		return v.Write(s.Rel, s.Content)
	}
	return nil
}
