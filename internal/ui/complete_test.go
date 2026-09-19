package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// completionVault adds notes that use tags and properties, and opens the
// note at rel in the editor with the cursor at the end of the text.
func completionVault(t *testing.T, rel, text string) *Model {
	t.Helper()
	m := newTestModel(t)
	notes := map[string]string{
		"Notes/A.md": "---\ntype: village\nland: \"[[Aldalor]]\"\ncreated: 2026-09-01\ntags: [Filosofi]\n---\nOm #Filosofi och #stoa",
		"Notes/B.md": "---\ntype: city\nland: \"[[Aldalor]]\"\ncreated: 2026-09-02\n---\n#filosofi",
		"Notes/C.md": "---\ntype: village\n---\n#Filosofi",
		rel:          text,
	}
	for p, body := range notes {
		abs := filepath.Join(m.vault.Root, p)
		os.MkdirAll(filepath.Dir(abs), 0o755)
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.reload(); err != nil {
		t.Fatal(err)
	}
	m.openEditor(rel)
	m.editor.HandleKey(keyPress("ctrl+end")) // key() would read "ctrl+end" as Ctrl+E
	return m
}

func labels(c *completion) []string {
	var out []string
	for _, it := range c.items {
		out = append(out, it.label)
	}
	return out
}

func TestTagCompletionShowsSpellingsSideBySide(t *testing.T) {
	m := completionVault(t, "Notes/New.md", "Idag ")
	typeText(m, "#fil")
	if m.complete == nil || m.complete.kind != completeTag {
		t.Fatal("typing #fil should suggest tags")
	}
	got := labels(m.complete)
	if len(got) != 2 || got[0] != "#Filosofi" || got[1] != "#filosofi" {
		t.Fatalf("both spellings, most used first: %v", got)
	}
	if d := m.complete.items[0].detail; !strings.Contains(d, "2 notes") || !strings.Contains(d, "also written filosofi") {
		t.Errorf("the row should say how often and the other spelling: %q", d)
	}
	checkFrame(t, m, "tag popup")
	press(m, "enter")
	if m.editor.Text() != "Idag #Filosofi" || m.complete != nil {
		t.Errorf("picking should put the tag in: %q", m.editor.Text())
	}
}

func TestValueCompletionInFrontmatter(t *testing.T) {
	m := completionVault(t, "Notes/New.md", "---\ntype: ")
	typeText(m, "vi")
	if m.complete == nil || m.complete.kind != completeValue {
		t.Fatal("typing a value should suggest the ones other notes use")
	}
	if got := labels(m.complete); len(got) != 1 || got[0] != "village" {
		t.Errorf("suggestions %v", got)
	}
	press(m, "enter")
	if m.editor.Text() != "---\ntype: village" {
		t.Errorf("got %q", m.editor.Text())
	}
}

func TestValueCompletionQuotesLinksAndSkipsDates(t *testing.T) {
	m := completionVault(t, "Notes/New.md", "---\nland: ")
	typeText(m, "Al")
	if m.complete == nil || labels(m.complete)[0] != "[[Aldalor]]" {
		t.Fatalf("link values should be suggested: %+v", m.complete)
	}
	press(m, "enter")
	if m.editor.Text() != "---\nland: \"[[Aldalor]]\"" {
		t.Errorf("a link goes in quotes, so the YAML stays valid: %q", m.editor.Text())
	}
	m = completionVault(t, "Notes/New2.md", "---\ncreated: ")
	typeText(m, "20")
	if m.complete != nil {
		t.Errorf("dates aren't suggested: %v", labels(m.complete))
	}
}

func TestTagsPropertySuggestsEveryTag(t *testing.T) {
	m := completionVault(t, "Notes/New.md", "---\ntags:\n  - ")
	typeText(m, "st")
	if m.complete == nil || labels(m.complete)[0] != "#stoa" {
		t.Fatalf("a tags list should suggest #tags from note text too: %+v", m.complete)
	}
	press(m, "enter")
	if m.editor.Text() != "---\ntags:\n  - stoa" {
		t.Errorf("got %q", m.editor.Text())
	}
}

func TestEscClosesTagSuggestionsForThatWord(t *testing.T) {
	m := completionVault(t, "Notes/New.md", "Idag ")
	typeText(m, "#fil")
	press(m, "esc")
	if m.complete != nil || m.editor == nil {
		t.Fatal("esc closes the popup and keeps editing")
	}
	typeText(m, "o")
	if m.complete != nil {
		t.Error("the popup stays closed while the same word is typed")
	}
	typeText(m, " #st")
	if m.complete == nil {
		t.Error("a new tag gets suggestions again")
	}
}
