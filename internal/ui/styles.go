package ui

import (
	"charm.land/lipgloss/v2"

	"github.com/lurioso/skrin/internal/theme"
)

type styles struct {
	text, bold, muted, dir, brand, flash, errText lipgloss.Style
	border, borderFocus, title, titleFocus        lipgloss.Style
	selFocus, selBlur, marked, cursor             lipgloss.Style
	pill, dangerPill                              lipgloss.Style
	diffAdd, diffDel, diffHunk                    lipgloss.Style
}

func newStyles(p theme.Palette) styles {
	s := lipgloss.NewStyle
	return styles{
		text:        s().Foreground(p.Foreground),
		bold:        s().Foreground(p.Foreground).Bold(true),
		muted:       s().Foreground(p.DarkForeground),
		dir:         s().Foreground(p.Blue).Bold(true),
		brand:       s().Foreground(p.Accent).Bold(true),
		flash:       s().Foreground(p.Accent),
		errText:     s().Foreground(p.Red),
		border:      s().Foreground(p.Border),
		borderFocus: s().Foreground(p.Accent),
		title:       s().Foreground(p.DarkForeground),
		titleFocus:  s().Foreground(p.Accent).Bold(true),
		selFocus:    s().Background(p.Selection).Foreground(p.LightForeground).Bold(true),
		selBlur:     s().Foreground(p.Accent),
		marked:      s().Foreground(p.Yellow),
		cursor:      s().Background(p.Foreground).Foreground(p.Background),
		pill:        s().Background(p.Accent).Foreground(p.Background).Bold(true),
		dangerPill:  s().Background(p.Red).Foreground(p.Background).Bold(true),
		diffAdd:     s().Foreground(p.Green),
		diffDel:     s().Foreground(p.Red),
		diffHunk:    s().Foreground(p.Cyan),
	}
}
