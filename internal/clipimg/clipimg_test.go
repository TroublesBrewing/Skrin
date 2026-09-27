package clipimg

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// fake is a clipboard made of canned answers: which programs exist on
// this imaginary machine, and what each argv prints. No test in this
// package ever runs a real clipboard program.
type fake struct {
	have map[string]bool
	out  map[string]string
	fail map[string]bool
	runs []string
}

func (f *fake) reader() *OS {
	return &OS{
		Look: func(name string) (string, error) {
			if f.have[name] {
				return "/usr/bin/" + name, nil
			}
			return "", errors.New("not found")
		},
		Run: func(name string, args ...string) ([]byte, error) {
			cmd := strings.Join(append([]string{name}, args...), " ")
			f.runs = append(f.runs, cmd)
			if f.fail[cmd] {
				return nil, fmt.Errorf("%s: failed", cmd)
			}
			return []byte(f.out[cmd]), nil
		},
	}
}

func TestAPNGInTheClipboardComesBackAsAPNG(t *testing.T) {
	f := &fake{
		have: map[string]bool{"wl-paste": true},
		out: map[string]string{
			"wl-paste --list-types":                  "image/png\ntext/html\n",
			"wl-paste --no-newline --type image/png": "\x89PNG...",
		},
	}
	img, err := f.reader().Image()
	if err != nil {
		t.Fatalf("Image() = %v", err)
	}
	if img.Mime != "image/png" || img.Ext != ".png" {
		t.Errorf("mime %q, ext %q", img.Mime, img.Ext)
	}
	if string(img.Data) != "\x89PNG..." {
		t.Errorf("data = %q", img.Data)
	}
}

func TestPNGIsPreferredOverTheOtherFormatsOnOffer(t *testing.T) {
	f := &fake{
		have: map[string]bool{"wl-paste": true},
		out: map[string]string{
			// The order the clipboard lists them in must not decide:
			// PNG is lossless and what a screenshot tool offers.
			"wl-paste --list-types":                  "image/bmp\nimage/jpeg\nimage/png\n",
			"wl-paste --no-newline --type image/png": "PNG",
		},
	}
	img, err := f.reader().Image()
	if err != nil {
		t.Fatalf("Image() = %v", err)
	}
	if img.Mime != "image/png" {
		t.Errorf("mime = %q, want image/png", img.Mime)
	}
}

func TestAJPEGClipboardKeepsItsOwnExtension(t *testing.T) {
	f := &fake{
		have: map[string]bool{"wl-paste": true},
		out: map[string]string{
			"wl-paste --list-types":                   "text/plain\nimage/jpeg\n",
			"wl-paste --no-newline --type image/jpeg": "JPEG",
		},
	}
	img, err := f.reader().Image()
	if err != nil {
		t.Fatalf("Image() = %v", err)
	}
	if img.Ext != ".jpg" {
		t.Errorf("ext = %q, want .jpg", img.Ext)
	}
}

func TestTextInTheClipboardIsNoImage(t *testing.T) {
	f := &fake{
		have: map[string]bool{"wl-paste": true},
		out:  map[string]string{"wl-paste --list-types": "text/plain;charset=utf-8\nSTRING\n"},
	}
	_, err := f.reader().Image()
	if !errors.Is(err, ErrNoImage) {
		t.Errorf("Image() = %v, want ErrNoImage", err)
	}
	// The type listing is the only thing it may run: a clipboard with no
	// image in it must not be read out at all.
	if len(f.runs) != 1 {
		t.Errorf("ran %v, want just the type listing", f.runs)
	}
}

func TestAnEmptyReadIsNoImageRatherThanAnEmptyFile(t *testing.T) {
	f := &fake{
		have: map[string]bool{"wl-paste": true},
		out:  map[string]string{"wl-paste --list-types": "image/png\n"},
	}
	_, err := f.reader().Image()
	if !errors.Is(err, ErrNoImage) {
		t.Errorf("Image() = %v, want ErrNoImage", err)
	}
}

func TestNoProgramAtAllSaysSo(t *testing.T) {
	f := &fake{}
	_, err := f.reader().Image()
	if !errors.Is(err, ErrNoTool) {
		t.Errorf("Image() = %v, want ErrNoTool", err)
	}
	if len(f.runs) != 0 {
		t.Errorf("ran %v with no tool installed", f.runs)
	}
}

func TestXClipReadsTheX11Clipboard(t *testing.T) {
	f := &fake{
		have: map[string]bool{"xclip": true},
		out: map[string]string{
			"xclip -selection clipboard -t TARGETS -o":   "TIMESTAMP\nimage/png\n",
			"xclip -selection clipboard -t image/png -o": "PNG",
		},
	}
	img, err := f.reader().Image()
	if err != nil {
		t.Fatalf("Image() = %v", err)
	}
	if string(img.Data) != "PNG" {
		t.Errorf("data = %q", img.Data)
	}
}

func TestOnlyTheFirstToolFoundIsAsked(t *testing.T) {
	// A Wayland session can have xclip installed too, over XWayland.
	// Asking X11 after Wayland said no would read a different, possibly
	// stale clipboard and paste something the user never copied.
	f := &fake{
		have: map[string]bool{"wl-paste": true, "xclip": true},
		out: map[string]string{
			"wl-paste --list-types":                      "text/plain\n",
			"xclip -selection clipboard -t TARGETS -o":   "image/png\n",
			"xclip -selection clipboard -t image/png -o": "PNG",
		},
	}
	if _, err := f.reader().Image(); !errors.Is(err, ErrNoImage) {
		t.Fatalf("Image() = %v, want ErrNoImage", err)
	}
	for _, run := range f.runs {
		if strings.HasPrefix(run, "xclip") {
			t.Errorf("asked xclip as well: %v", f.runs)
		}
	}
}

func TestPngpasteHasNoTypesToList(t *testing.T) {
	f := &fake{
		have: map[string]bool{"pngpaste": true},
		out:  map[string]string{"pngpaste -": "PNG"},
	}
	img, err := f.reader().Image()
	if err != nil {
		t.Fatalf("Image() = %v", err)
	}
	if img.Mime != "image/png" || img.Ext != ".png" || string(img.Data) != "PNG" {
		t.Errorf("image = %+v", img)
	}
}

func TestPngpasteFailingMeansNoImage(t *testing.T) {
	f := &fake{
		have: map[string]bool{"pngpaste": true},
		fail: map[string]bool{"pngpaste -": true},
	}
	if _, err := f.reader().Image(); !errors.Is(err, ErrNoImage) {
		t.Errorf("Image() = %v, want ErrNoImage", err)
	}
}

func TestAToolThatWontSayWhatItHasIsNoImage(t *testing.T) {
	f := &fake{
		have: map[string]bool{"wl-paste": true},
		fail: map[string]bool{"wl-paste --list-types": true},
	}
	if _, err := f.reader().Image(); !errors.Is(err, ErrNoImage) {
		t.Errorf("Image() = %v, want ErrNoImage", err)
	}
}

func TestTheUnregisteredImageJpgSpellingIsAccepted(t *testing.T) {
	f := &fake{
		have: map[string]bool{"wl-paste": true},
		out: map[string]string{
			"wl-paste --list-types":                  "image/jpg\n",
			"wl-paste --no-newline --type image/jpg": "JPEG",
		},
	}
	img, err := f.reader().Image()
	if err != nil {
		t.Fatalf("Image() = %v", err)
	}
	if img.Ext != ".jpg" {
		t.Errorf("ext = %q, want .jpg", img.Ext)
	}
}
