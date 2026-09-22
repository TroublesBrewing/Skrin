//go:build !unix

package main

// relaunch re-runs Skrin pointed at vault. Where exec isn't available the
// switch still took effect — the new vault is saved as default — but the
// process can't hand over its terminal, so it just exits and the user
// opens Skrin again.
func relaunch(vault string) error { return nil }
