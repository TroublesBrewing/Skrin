// Package clipimg reads an image out of the operating system's clipboard.
//
// Skrin's own copy and paste go through the terminal over OSC 52
// (internal/ui/paste.go), which works in a local terminal, over ssh and
// inside tmux with no OS tool and no cgo — but OSC 52 can only carry
// text. An image has to come from the OS clipboard directly, through
// whichever small program the desktop has: wl-paste on Wayland, xclip on
// X11, pngpaste on macOS. This package is the only place Skrin asks the
// desktop for the clipboard, and it is why pasting an image is the one
// paste that can't work over ssh.
package clipimg

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

var (
	// ErrNoTool means the desktop has no program that can hand an image
	// over, so Skrin can't even tell whether the clipboard holds one.
	ErrNoTool = errors.New("no clipboard tool for images")
	// ErrNoImage means the clipboard was readable and holds no image.
	ErrNoImage = errors.New("no image in the clipboard")
)

// timeout bounds one clipboard program's run. A clipboard read is
// instant when it works at all; a program that hangs must not hang a
// keypress with it.
const timeout = 2 * time.Second

// Image is one image read out of the clipboard.
type Image struct {
	Data []byte
	Mime string
	Ext  string // the file extension for Mime: ".png", ".jpg", …
}

// Reader reads an image out of the clipboard. Image returns ErrNoImage
// when the clipboard holds something else and ErrNoTool when no program
// on this machine can answer at all.
type Reader interface {
	Image() (Image, error)
}

// exts are the clipboard types Skrin accepts, in the order it prefers
// them: the formats internal/imgmeta can read and internal/ui can draw,
// PNG first because it is lossless and what a screenshot tool offers.
var exts = []struct{ mime, ext string }{
	{"image/png", ".png"},
	{"image/jpeg", ".jpg"},
	{"image/gif", ".gif"},
	{"image/webp", ".webp"},
	{"image/bmp", ".bmp"},
}

// tool is one clipboard program: how to ask what the clipboard offers,
// and how to read one type out of it.
type tool struct {
	name  string
	types []string                   // argv that lists the types on offer
	read  func(mime string) []string // argv that writes that type to stdout
	// assume is the type a program that can't list types always hands
	// over; pngpaste is the case.
	assume string
}

// tools are tried in this order, and the first one present is the one
// used — never a second as a fallback. On a Wayland session with xclip
// also installed, asking X11 after Wayland said no would read a
// different, possibly stale clipboard.
var tools = []tool{
	{
		name:  "wl-paste",
		types: []string{"--list-types"},
		read:  func(mime string) []string { return []string{"--no-newline", "--type", mime} },
	},
	{
		name:  "xclip",
		types: []string{"-selection", "clipboard", "-t", "TARGETS", "-o"},
		read: func(mime string) []string {
			return []string{"-selection", "clipboard", "-t", mime, "-o"}
		},
	},
	{
		name:   "pngpaste",
		assume: "image/png",
		read:   func(string) []string { return []string{"-"} },
	},
}

// OS is the real clipboard. Look and Run are exec.LookPath and one
// command's output, replaceable so that no test ever runs a real
// clipboard program.
type OS struct {
	Look func(name string) (string, error)
	Run  func(name string, args ...string) ([]byte, error)
}

// New returns the real clipboard reader.
func New() *OS { return &OS{} }

func (o *OS) look(name string) error {
	look := o.Look
	if look == nil {
		look = exec.LookPath
	}
	_, err := look(name)
	return err
}

func (o *OS) run(name string, args ...string) ([]byte, error) {
	if o.Run != nil {
		return o.Run(name, args...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}

// Image reads the clipboard's image through the first tool this machine
// has.
func (o *OS) Image() (Image, error) {
	for _, t := range tools {
		if o.look(t.name) != nil {
			continue
		}
		return o.image(t)
	}
	return Image{}, ErrNoTool
}

func (o *OS) image(t tool) (Image, error) {
	mime, ext := t.assume, extFor(t.assume)
	if t.types != nil {
		out, err := o.run(t.name, t.types...)
		if err != nil {
			// A program that won't say what's on the clipboard is, as
			// far as pasting goes, the same as a clipboard with no
			// image in it: fall back to the text path and say nothing.
			return Image{}, ErrNoImage
		}
		if mime, ext = pick(string(out)); mime == "" {
			return Image{}, ErrNoImage
		}
	}
	data, err := o.run(t.name, t.read(mime)...)
	if err != nil || len(data) == 0 {
		return Image{}, ErrNoImage
	}
	return Image{Data: data, Mime: mime, Ext: ext}, nil
}

// pick reads a tool's type listing and returns the image type Skrin
// prefers among those on offer, or "" when none is an image it can
// handle. The listings are one type per line (wl-paste, xclip TARGETS),
// and a type never contains a space, so fields are enough.
func pick(list string) (mime, ext string) {
	have := map[string]bool{}
	for _, f := range strings.Fields(list) {
		have[strings.ToLower(f)] = true
	}
	for _, e := range exts {
		if have[e.mime] {
			return e.mime, e.ext
		}
	}
	// image/jpg is not a registered type, but tools and apps do offer
	// it, and refusing a paste over the spelling would be pedantry.
	if have["image/jpg"] {
		return "image/jpg", ".jpg"
	}
	return "", ""
}

// extFor is the file extension for a clipboard type, ".png" when the
// type is one Skrin doesn't know by name (it only ever reads types it
// asked for).
func extFor(mime string) string {
	for _, e := range exts {
		if e.mime == mime {
			return e.ext
		}
	}
	return ".png"
}
