package vault

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

// renameNoReplace uses renameat2, which the kernel fails rather than
// letting it overwrite: the check and the move are one step, so nothing can
// slip in between them.
func renameNoReplace(from, to string) error {
	err := unix.Renameat2(unix.AT_FDCWD, from, unix.AT_FDCWD, to, unix.RENAME_NOREPLACE)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, unix.EEXIST), errors.Is(err, unix.ENOTEMPTY):
		return fmt.Errorf("%s: %w", to, ErrExists)
	case errors.Is(err, unix.ENOSYS), errors.Is(err, unix.EINVAL), errors.Is(err, unix.EOPNOTSUPP):
		// An old kernel, or a filesystem that doesn't carry the flag (some
		// network and FUSE mounts): fall back to looking first.
		return renameChecked(from, to)
	}
	return err
}
