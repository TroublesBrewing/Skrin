package book

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// FetchCover downloads url's bytes, bounded by the same timeout as a
// metadata lookup. It never touches the vault: SaveCover does that.
func FetchCover(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "skrin (https://github.com/lurioso/skrin)")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// CoverExt guesses a file extension from a cover URL, defaulting to .jpg
// (what Open Library's cover service always serves).
func CoverExt(url string) string {
	switch ext := strings.ToLower(path.Ext(strings.SplitN(url, "?", 2)[0])); ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return ext
	default:
		return ".jpg"
	}
}

// CoverPath builds the vault-relative path a cover for title/year would be
// saved at, e.g. "Assets/Covers/Meditations-2003.jpg". If that path is
// already used by different content, a numeric suffix is added
// ("-2.jpg") until a free name is found; abs reports whether a path
// exists so tests can fake the vault.
func CoverPath(coversFolder, title, year, ext string, exists func(rel string) bool) string {
	name := SanitizeFilePart(title)
	name = strings.ReplaceAll(name, " ", "-")
	if name == "" {
		name = "cover"
	}
	if year != "" {
		name += "-" + year
	}
	rel := path.Join(coversFolder, name+ext)
	for i := 2; exists != nil && exists(rel); i++ {
		rel = path.Join(coversFolder, fmt.Sprintf("%s-%d%s", name, i, ext))
	}
	return rel
}

// SaveCover writes data to the vault at rel (an absolute path rooted in
// vaultAbs, computed by the caller with CoverPath), creating the covers
// folder if needed. The write is atomic: a temp file next to the
// destination, synced, then renamed into place, so a crash mid-download
// never leaves a half-written cover.
func SaveCover(vaultAbs func(rel string) string, rel string, data []byte) error {
	abs := vaultAbs(rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(abs), ".skrin-cover-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), abs)
}
