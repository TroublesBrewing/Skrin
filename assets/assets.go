// Package assets embeds static files shipped inside the skrin binary.
package assets

import _ "embed"

// ObsidianLogo is the official Obsidian app icon (512×512 PNG), copied from
// the obsidian package's hicolor icon theme.
//
//go:embed obsidian-logo.png
var ObsidianLogo []byte
