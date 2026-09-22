package editor

import (
	"strings"
	"testing"
)

// A terminal paste sends CR for a line break. Before NormalizeNewlines the
// editor only understood LF and CRLF, so a pasted paragraph from the
// terminal collapsed onto a single line — verified live in tmux, where a
// two-line paste landed as "ONE\rTWO" on one line.
func TestPasteOfCRLinesSplitsThem(t *testing.T) {
	e := open("")
	e.Paste("ONE\rTWO\rTHREE")
	if got, want := e.Text(), "ONE\nTWO\nTHREE"; got != want {
		t.Errorf("Text() = %q, want %q", got, want)
	}
}

func TestPasteOfCRLFLinesSplitsThem(t *testing.T) {
	e := open("")
	e.Paste("ONE\r\nTWO")
	if got, want := e.Text(), "ONE\nTWO"; got != want {
		t.Errorf("Text() = %q, want %q", got, want)
	}
}

func TestPasteOfLFLinesIsUnchanged(t *testing.T) {
	e := open("")
	e.Paste("ONE\nTWO")
	if got, want := e.Text(), "ONE\nTWO"; got != want {
		t.Errorf("Text() = %q, want %q", got, want)
	}
}

func TestNormalizeNewlinesLeavesCleanTextAlone(t *testing.T) {
	for _, s := range []string{"", "plain", "one\ntwo"} {
		if got := NormalizeNewlines(s); got != s {
			t.Errorf("NormalizeNewlines(%q) = %q, want it untouched", s, got)
		}
	}
	if got := NormalizeNewlines("a\r\nb"); got != "a\nb" {
		t.Errorf("CRLF = %q, want one break", got)
	}
	if got := NormalizeNewlines("a\rb"); got != "a\nb" {
		t.Errorf("bare CR = %q, want one break", got)
	}
	if !strings.Contains(NormalizeNewlines("x"), "x") {
		t.Error("plain text must survive")
	}
}
