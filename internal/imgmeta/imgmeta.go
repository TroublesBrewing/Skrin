// Package imgmeta reads just enough of an image file to describe it: its
// pixel dimensions and its size on disk. It never fully decodes the image
// for this — image.DecodeConfig reads only the header — so a note full of
// embeds costs nothing more than a stat and a few header bytes per image.
package imgmeta

import (
	"image"
	_ "image/gif"  // register GIF with image.DecodeConfig
	_ "image/jpeg" // register JPEG with image.DecodeConfig
	_ "image/png"  // register PNG with image.DecodeConfig
	"os"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/bmp"  // register BMP with image.DecodeConfig
	_ "golang.org/x/image/webp" // register WebP with image.DecodeConfig
)

// Info is what's known about an embedded image file.
type Info struct {
	Width, Height int
	Size          int64 // bytes on disk
}

// Read stats abs (an absolute filesystem path) and reads its image header.
// A missing file returns an fs.ErrNotExist-wrapping error; an existing file
// in a format none of Skrin's registered decoders understand returns
// image.ErrFormat.
func Read(abs string) (Info, error) {
	fi, err := os.Stat(abs)
	if err != nil {
		return Info{}, err
	}
	f, err := os.Open(abs)
	if err != nil {
		return Info{}, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return Info{}, err
	}
	return Info{Width: cfg.Width, Height: cfg.Height, Size: fi.Size()}, nil
}

// IsImage reports whether name's extension is one Skrin renders a preview
// or placeholder for.
func IsImage(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
		return true
	}
	return false
}
