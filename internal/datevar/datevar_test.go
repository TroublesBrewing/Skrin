package datevar

import (
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 9, 30, 14, 5, 0, 0, time.Local)

func resolve(t *testing.T, text, base string) (string, []Change) {
	t.Helper()
	return Resolve(text, base, "YYYY-MM-DD", "HH:mm", now)
}

func TestDateAndTimeBecomeWhatTheyMean(t *testing.T) {
	got, changes := resolve(t, "Written {{date}} at {{time}}.", "")
	if got != "Written 2026-09-30 at 14:05." {
		t.Errorf("got %q", got)
	}
	if len(changes) != 2 {
		t.Fatalf("changes %+v", changes)
	}
	if changes[0].Var != "{{date}}" || changes[0].Date != "2026-09-30" {
		t.Errorf("first change %+v", changes[0])
	}
}

func TestAFormatAndAnOffsetAreHonoured(t *testing.T) {
	got, _ := resolve(t, "{{date:dddd}} and {{date+1d:YYYY-MM-DD}}", "")
	if got != "Wednesday and 2026-10-01" {
		t.Errorf("got %q", got)
	}
}

// The heart of it, and the same bargain due:: makes: a variable that was
// already in the note keeps saying what its author wrote.
func TestAnOldVariableIsLeftAlone(t *testing.T) {
	base := "Old line with {{date}} in it.\n"
	got, changes := resolve(t, base+"New line with {{date}}.", base)
	if !strings.HasPrefix(got, "Old line with {{date}} in it.") {
		t.Errorf("the old line was rewritten: %q", got)
	}
	if !strings.HasSuffix(got, "New line with 2026-09-30.") {
		t.Errorf("the new line wasn't: %q", got)
	}
	if len(changes) != 1 {
		t.Errorf("changes %+v", changes)
	}
}

// A note about templates has to be able to show the variable.
func TestCodeIsLeftAlone(t *testing.T) {
	text := "```\n{{date}}\n```\nand `{{time}}` inline"
	got, changes := resolve(t, text, "")
	if got != text {
		t.Errorf("got %q", got)
	}
	if len(changes) != 0 {
		t.Errorf("changes %+v", changes)
	}
}

// A variable the expander doesn't know is not a date, and is left as it
// was written rather than turned into something wrong.
func TestSomethingThatIsNotADateVariableStays(t *testing.T) {
	text := "{{title}} and {{value}} and {{}}"
	got, changes := resolve(t, text, "")
	if got != text {
		t.Errorf("got %q", got)
	}
	if len(changes) != 0 {
		t.Errorf("changes %+v", changes)
	}
}

func TestSeveralOnOneLineAllGetTheirTurn(t *testing.T) {
	got, changes := resolve(t, "{{date}} {{date:dddd}} {{time}}", "")
	if got != "2026-09-30 Wednesday 14:05" {
		t.Errorf("got %q", got)
	}
	if len(changes) != 3 {
		t.Errorf("changes %+v", changes)
	}
}
