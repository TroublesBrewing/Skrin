package vault

import (
	"fmt"
	"os"
)

// renameNoReplace renames from to to and refuses to replace anything at the
// destination, returning ErrExists instead. Looking first and renaming
// after isn't enough: os.Rename replaces silently, so a file that Obsidian
// Sync or another app puts there in between would be overwritten without a
// trace. Where the kernel or the filesystem can't do it, renameChecked is
// the best that can be done.
func renameChecked(from, to string) error {
	if _, err := os.Lstat(to); err == nil {
		return fmt.Errorf("%s: %w", to, ErrExists)
	}
	return os.Rename(from, to)
}
