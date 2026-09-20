package habiton

import (
	"strings"
	"testing"

	"github.com/lurioso/skrin/internal/habit"
	"github.com/lurioso/skrin/internal/theme"
)

// grid builds a grid where day i has the ticks given, oldest first. "xx-"
// means two ticked and one not; "" means the day has no note at all.
func grid(names []string, days ...string) (habit.Grid, string) {
	g := habit.Grid{Names: names, Done: map[string]map[string]bool{}}
	last := ""
	for i, d := range days {
		path := ""
		if d != "" {
			path = string(rune('a'+i)) + ".md"
			g.Done[path] = map[string]bool{}
			for j, c := range d {
				if j < len(names) {
					g.Done[path][names[j]] = c == 'x'
				}
			}
		}
		g.Days = append(g.Days, habit.Day{Path: path})
		last = path
	}
	return g, last
}

func TestMoodFollowsTheHabitsThemselves(t *testing.T) {
	names := []string{"a", "b", "c"}
	for _, c := range []struct {
		name string
		days []string
		want Mood
	}{
		{"all of today ticked", []string{"xxx", "xxx"}, Bright},
		{"some of today", []string{"xxx", "x--"}, Awake},
		{"none today, but the days behind are good", []string{"xxx", "xxx", "---"}, Asleep},
		{"little for days", []string{"---", "x--", "---"}, Sleepy},
		{"all of today, after a bad stretch", []string{"---", "---", "xxx"}, Bright},
		{"nothing recorded yet", []string{"", ""}, Asleep},
	} {
		t.Run(c.name, func(t *testing.T) {
			g, today := grid(names, c.days...)
			if got := Read(g, today); got != c.want {
				t.Errorf("mood = %v, want %v", got, c.want)
			}
		})
	}
}

// A day with no note is not a day you failed: it is a day with no note.
func TestDaysWithoutANoteDontMakeHimTired(t *testing.T) {
	g, today := grid([]string{"a", "b"}, "xx", "", "", "x-")
	if got := Read(g, today); got == Sleepy {
		t.Error("two unrecorded days shouldn't read as two bad days")
	}
}

func TestHeIsDrawnAndBlinks(t *testing.T) {
	pal := theme.Default()
	for mood := Asleep; mood <= Sleepy; mood++ {
		open, w := Draw(mood, 0, pal)
		shut, w2 := Draw(mood, 1, pal)
		if w != 12 || w2 != 12 || len(open) != 6 {
			t.Fatalf("mood %v: %d×%d cells", mood, w, len(open))
		}
		if mood == Awake || mood == Bright {
			if strings.Join(open, "\n") == strings.Join(shut, "\n") {
				t.Errorf("mood %v should blink", mood)
			}
		}
	}
}

// Nothing about Habiton is written down: he is read from the notes every
// time, so there is no score to lose, corrupt or sync.
func TestHeKeepsNoStateOfHisOwn(t *testing.T) {
	g, today := grid([]string{"a"}, "x")
	if Read(g, today) != Read(g, today) {
		t.Error("the same notes must always give the same mood")
	}
}
