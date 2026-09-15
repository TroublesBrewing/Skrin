package vault

import (
	"errors"
	"os"
	"testing"
)

func TestUndoKeepsWhatsOnDisk(t *testing.T) {
	v := makeVault(t, "a.md")
	var j Journal
	var kept []string
	j.Keep = func(rel, disk string) error {
		kept = append(kept, rel+": "+disk)
		return nil
	}
	j.Record(Op{Desc: "edit", Steps: []Step{{Kind: StepModified, Rel: "a.md", Content: "# a.md"}}})
	if err := os.WriteFile(v.Abs("a.md"), []byte("changed elsewhere"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := j.Undo(v, "system"); err != nil {
		t.Fatal(err)
	}
	if got, _ := v.Read("a.md"); got != "# a.md" {
		t.Errorf("undo left %q", got)
	}
	if len(kept) != 1 || kept[0] != "a.md: changed elsewhere" {
		t.Errorf("kept %q, want the text that was on disk", kept)
	}

	j.Keep = func(string, string) error { return errors.New("disk full") }
	j.Record(Op{Desc: "edit", Steps: []Step{{Kind: StepModified, Rel: "a.md", Content: "older"}}})
	if _, _, err := j.Undo(v, "system"); err == nil {
		t.Error("an undo that can't keep the current text should fail")
	}
	if got, _ := v.Read("a.md"); got != "# a.md" {
		t.Errorf("the note should be left alone: %q", got)
	}
}
