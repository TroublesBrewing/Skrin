package tagname

import "testing"

func TestATagInTheTextIsRenamed(t *testing.T) {
	got, n := Rename("A thought #filosofi and another #filosofi.\n", "filosofi", "filosofi-2")
	want := "A thought #filosofi-2 and another #filosofi-2.\n"
	if got != want || n != 2 {
		t.Errorf("Rename = %q (%d), want %q (2)", got, n, want)
	}
}

func TestAWordThatMerelyStartsTheSameIsLeftAlone(t *testing.T) {
	got, n := Rename("#ux and #uxdesign and #ux/research\n", "ux", "design")
	want := "#design and #uxdesign and #design/research\n"
	if got != want || n != 2 {
		t.Errorf("Rename = %q (%d), want %q (2)", got, n, want)
	}
}

func TestNestedTagsComeAlong(t *testing.T) {
	got, n := Rename("#work/urgent #work #work/later/maybe\n", "work", "job")
	want := "#job/urgent #job #job/later/maybe\n"
	if got != want || n != 3 {
		t.Errorf("Rename = %q (%d), want %q (3)", got, n, want)
	}
}

func TestEverySpellingIsGatheredIntoOne(t *testing.T) {
	// The usual reason for renaming a tag at all: one note wrote it with a
	// capital and now there are two tags where one was meant.
	got, n := Rename("#Filosofi here, #filosofi there, #FILOSOFI everywhere\n", "Filosofi", "filosofi")
	want := "#filosofi here, #filosofi there, #filosofi everywhere\n"
	if got != want || n != 3 {
		t.Errorf("Rename = %q (%d), want %q (3)", got, n, want)
	}
}

func TestCodeAndLinksAreNotTags(t *testing.T) {
	src := "```\n#old in a fence\n```\nInline `#old` too.\nAnd [[#old]] is a heading link.\nBut #old is a tag.\n"
	got, n := Rename(src, "old", "new")
	if n != 1 {
		t.Fatalf("changed %d, want only the real tag:\n%s", n, got)
	}
	if want := "But #new is a tag.\n"; got[len(got)-len(want):] != want {
		t.Errorf("the tag in prose should change:\n%s", got)
	}
	if got[:len("```\n#old in a fence\n```")] != "```\n#old in a fence\n```" {
		t.Errorf("the fence should be untouched:\n%s", got)
	}
}

func TestATagsPropertyIsRenamedInEveryShape(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"---\ntags: [filosofi, stoa]\n---\n", "---\ntags: [filosofi-2, stoa]\n---\n"},
		{"---\ntags:\n  - filosofi\n  - stoa\n---\n", "---\ntags:\n  - filosofi-2\n  - stoa\n---\n"},
		{"---\ntag: filosofi\n---\n", "---\ntag: filosofi-2\n---\n"},
		{"---\ntags: [\"#filosofi\"]\n---\n", "---\ntags: [\"#filosofi-2\"]\n---\n"},
	} {
		got, n := Rename(c.in, "filosofi", "filosofi-2")
		if got != c.want || n != 1 {
			t.Errorf("Rename(%q) = %q (%d), want %q (1)", c.in, got, n, c.want)
		}
	}
}

func TestAnotherPropertyHoldingTheSameWordIsLeftAlone(t *testing.T) {
	src := "---\nauthor: filosofi\ntags: [filosofi]\ntitle: filosofi\n---\n#filosofi\n"
	got, n := Rename(src, "filosofi", "x")
	want := "---\nauthor: filosofi\ntags: [x]\ntitle: filosofi\n---\n#x\n"
	if got != want || n != 2 {
		t.Errorf("Rename = %q (%d), want %q (2)", got, n, want)
	}
}

func TestANoteWithoutTheTagIsUnchanged(t *testing.T) {
	src := "---\ntags: [other]\n---\nNothing here.\n"
	got, n := Rename(src, "filosofi", "x")
	if got != src || n != 0 {
		t.Errorf("Rename = %q (%d), want it untouched", got, n)
	}
}

func TestRenamingToTheSameNameDoesNothing(t *testing.T) {
	src := "#x\n"
	if got, n := Rename(src, "x", "x"); got != src || n != 0 {
		t.Errorf("Rename = %q (%d)", got, n)
	}
	if got, n := Rename(src, "x", "#x"); got != src || n != 0 {
		t.Errorf("a leading # is not a different name: %q (%d)", got, n)
	}
}

func TestCleanTakesOffTheHashAndTheEdges(t *testing.T) {
	for in, want := range map[string]string{
		"#tag": "tag", " tag ": "tag", "#tag/": "tag", "tag/sub": "tag/sub", "": "",
	} {
		if got := Clean(in); got != want {
			t.Errorf("Clean(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidSaysWhyANameWontDo(t *testing.T) {
	for _, bad := range []string{"", "  ", "two words", "a//b", "no!", "2026"} {
		if err := Valid(bad); err == nil {
			t.Errorf("Valid(%q) should refuse", bad)
		}
	}
	for _, good := range []string{"tag", "#tag", "parent/child", "with-hyphen", "with_underscore", "idé", "v2"} {
		if err := Valid(good); err != nil {
			t.Errorf("Valid(%q) = %v, want ok", good, err)
		}
	}
}

func TestANonASCIITagIsRenamedWhole(t *testing.T) {
	got, n := Rename("#idé and #idélåda\n", "idé", "tanke")
	want := "#tanke and #idélåda\n"
	if got != want || n != 1 {
		t.Errorf("Rename = %q (%d), want %q (1)", got, n, want)
	}
}
