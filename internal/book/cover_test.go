package book

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCoverPathAvoidsDuplicates(t *testing.T) {
	taken := map[string]bool{"Assets/Covers/Meditations-2003.jpg": true}
	exists := func(rel string) bool { return taken[rel] }
	got := CoverPath("Assets/Covers", "Meditations", "2003", ".jpg", exists)
	if got != "Assets/Covers/Meditations-2003-2.jpg" {
		t.Errorf("CoverPath = %q", got)
	}
	// A fresh title/year doesn't collide.
	got = CoverPath("Assets/Covers", "Kallocain", "1940", ".jpg", exists)
	if got != "Assets/Covers/Kallocain-1940.jpg" {
		t.Errorf("CoverPath = %q", got)
	}
}

func TestCoverExtDefaultsToJPG(t *testing.T) {
	cases := map[string]string{
		"https://covers.openlibrary.org/b/id/1-L.jpg":  ".jpg",
		"https://example.com/cover.PNG?x=1":            ".png",
		"https://example.com/cover-with-no-extension":  ".jpg",
	}
	for url, want := range cases {
		if got := CoverExt(url); got != want {
			t.Errorf("CoverExt(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestSaveCoverIsAtomicAndCreatesFolders(t *testing.T) {
	root := t.TempDir()
	abs := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	rel := "Assets/Covers/Test-2020.jpg"
	data := []byte("fake jpeg bytes")
	if err := SaveCover(abs, rel, data); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(abs(rel))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Errorf("saved cover content = %q, want %q", got, data)
	}
	// No leftover temp files in the covers folder.
	entries, _ := os.ReadDir(filepath.Dir(abs(rel)))
	if len(entries) != 1 {
		t.Errorf("covers folder has %d entries, want 1: %v", len(entries), entries)
	}
}
