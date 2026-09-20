package search

import (
	"reflect"
	"strings"
	"testing"
)

var (
	stoic = Doc{
		Rel:   "Filosofi/Stoic notes.md",
		Lines: []string{"# Stoic", "Calm under pressure.", "Seneca wrote letters"},
		Tags:  []string{"stoa", "filosofi/antik"},
		Props: map[string][]string{"status": {"draft"}, "author": {"Seneca"}, "tags": {"stoa"}},
	}
	daily = Doc{
		Rel:   "Daily/2026-09-15.md",
		Lines: []string{"- [ ] call mum", "calm day"},
		Tags:  []string{"daily"},
	}
)

func TestParseAndMatch(t *testing.T) {
	for q, want := range map[string][2]bool{ // matches stoic, daily
		"calm":             {true, true},
		"calm seneca":      {true, false},
		`"calm under"`:     {true, false},
		"calm -mum":        {true, false},
		"mum OR letters":   {true, true},
		"#stoa":            {true, false},
		"tag:filosofi":     {true, false},
		"#filo":            {false, false}, // not a tag, only the start of one
		"[status]":         {true, false},
		"[status:DRA]":     {true, false},
		"[author:plato]":   {false, false},
		"path:daily":       {false, true},
		`path:"filosofi/"`: {true, false},
		"file:stoic":       {true, false},
		"-#stoa calm":      {false, true},
		"notes":            {true, false}, // the file name counts
		`"unclosed quo`:    {false, false},
		"":                 {false, false},
	} {
		query := Parse(q, false)
		got := [2]bool{}
		got[0], _ = query.Match(stoic)
		got[1], _ = query.Match(daily)
		if got != want {
			t.Errorf("%q matches %v, want %v", q, got, want)
		}
	}
}

func TestHitsAndCase(t *testing.T) {
	_, hits := Parse("calm", false).Match(stoic)
	if !reflect.DeepEqual(hits, []Hit{{Line: 1, Spans: [][2]int{{0, 4}}}}) {
		t.Errorf("hits = %+v", hits)
	}
	if ok, _ := Parse("calm", true).Match(stoic); ok {
		t.Error("match-case search found Calm for calm")
	}
	if ok, _ := Parse("Calm", true).Match(stoic); !ok {
		t.Error("match-case search missed Calm")
	}
	_, hits = Parse("e", false).Match(Doc{Lines: []string{"eee"}})
	if !reflect.DeepEqual(hits[0].Spans, [][2]int{{0, 3}}) {
		t.Errorf("neighbouring spans should merge: %v", hits[0].Spans)
	}
}

func TestFindAndReplace(t *testing.T) {
	s := "Stoic stoicism, STOIC; (stoic) å-stoic_x"
	if got := len(Find(s, "stoic", false, false)); got != 5 {
		t.Errorf("any case: %d matches, want 5", got)
	}
	whole := Find(s, "stoic", false, true)
	if len(whole) != 3 {
		t.Errorf("whole words: %v, want Stoic, STOIC and (stoic)", whole)
	}
	if got := len(Find(s, "stoic", true, false)); got != 3 {
		t.Errorf("match case: %d, want 3", got)
	}
	if got := Replace(s, whole, "Stoa"); got != "Stoa stoicism, Stoa; (Stoa) å-stoic_x" {
		t.Errorf("Replace = %q", got)
	}
	if got := len(Find("Åsa åsa ÅSA", "åsa", false, true)); got != 3 {
		t.Errorf("Unicode case folding: %d matches, want 3", got)
	}
	if Find("abc", "", false, false) != nil {
		t.Error("an empty needle should find nothing")
	}
}

// /regex/ is Obsidian's, and a broken one says what's wrong instead of
// quietly matching nothing.
func TestRegexTerm(t *testing.T) {
	d := Doc{Rel: "Books/Bilbo.md", Lines: []string{"# Bilbo", "The year 1937 and the year 2002."}}
	for _, c := range []struct {
		q    string
		want bool
	}{
		{`/year \d{4}/`, true},
		{`/YEAR \d{4}/`, true},  // case doesn't matter by default
		{`/year \d{5}/`, false}, // a real regex, not a substring
		{`/^The year/`, true},   // anchors work, per line
	} {
		if ok, _ := Parse(c.q, false).Match(d); ok != c.want {
			t.Errorf("%s matched %v, want %v", c.q, ok, c.want)
		}
	}
	if ok, _ := Parse(`/YEAR \d{4}/`, true).Match(d); ok {
		t.Error("with match-case on, the regex should be case-sensitive too")
	}
}

func TestABrokenRegexSaysWhatIsWrong(t *testing.T) {
	q := Parse(`/year (/ stoic`, false)
	if p := q.Problem(); !strings.Contains(p, "isn't a regular expression") {
		t.Errorf("problem = %q", p)
	}
	d := Doc{Rel: "Filosofi/Stoic.md", Lines: []string{"# Stoic", "the stoic year"}}
	if ok, _ := q.Match(d); !ok {
		t.Error("the rest of the query should still work")
	}
}
