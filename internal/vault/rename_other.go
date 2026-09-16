//go:build !linux

package vault

// renameNoReplace looks before it moves. Only Linux has a rename that
// refuses to replace in one step; everywhere else this window stays open,
// narrow as it is.
func renameNoReplace(from, to string) error { return renameChecked(from, to) }
