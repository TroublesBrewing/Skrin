//go:build unix

package main

import (
	"os"
	"syscall"
)

// relaunch re-runs Skrin pointed at vault, in the same process so the
// terminal and its state stay. It only returns on failure; on success the
// process image is replaced and this never comes back.
func relaunch(vault string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return syscall.Exec(exe, []string{exe, vault}, os.Environ())
}
