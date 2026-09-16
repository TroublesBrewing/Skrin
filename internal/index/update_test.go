package index

import (
	"os"
	"strings"
	"testing"
)

// A note that can't be read for a moment — Sync rewriting it mid-refresh —
// must keep what it had in the index, rather than dropping out of search
// and links until some later refresh happens to catch it.
func TestUnreadableNoteKeepsWhatItHad(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read a file with no permissions")
	}
	x, v := build(t, map[string]string{"Stoic.md": "# Stoic\nvirtue is enough\n"})
	if _, ok := x.Doc("Stoic.md"); !ok {
		t.Fatal("the note should be in the index to begin with")
	}
	// Changed, so it would be re-read — and unreadable at that moment.
	if err := os.WriteFile(v.Abs("Stoic.md"), []byte("# Stoic\nvirtue is enough, and more\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(v.Abs("Stoic.md"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(v.Abs("Stoic.md"), 0o644) })
	if err := x.Update(v); err != nil {
		t.Fatal(err)
	}
	content, ok := x.Content("Stoic.md")
	if !ok || !strings.Contains(content, "virtue is enough") {
		t.Errorf("the note fell out of the index: ok = %v, content = %q", ok, content)
	}
}
