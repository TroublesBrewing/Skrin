package book

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// TestChainStopsAtFirstMatch is the "stops at the first provider with a
// match" rule from the amendment's Lookup order section: once a call
// returns results, later calls in the chain are never invoked.
func TestChainStopsAtFirstMatch(t *testing.T) {
	calledThird := false
	out := chain(context.Background(), []providerCall{
		{"First", func(context.Context) ([]Result, error) { return nil, nil }},
		{"Second", func(context.Context) ([]Result, error) { return []Result{{Title: "Found it"}}, nil }},
		{"Third", func(context.Context) ([]Result, error) { calledThird = true; return []Result{{Title: "Too late"}}, nil }},
	})
	if calledThird {
		t.Error("the chain should stop at the first match, not call every provider")
	}
	if len(out.Results) != 1 || out.Results[0].Title != "Found it" {
		t.Errorf("Results = %+v", out.Results)
	}
	if want := []string{"First", "Second"}; !equalStrings(out.Tried, want) {
		t.Errorf("Tried = %v, want %v", out.Tried, want)
	}
}

// TestChainDistinguishesNoRecordFromDown is the amendment's central
// distinction: a provider that answers cleanly with nothing is "tried",
// not "down" — only a provider that fails to answer at all is down, and
// either way the chain keeps going.
func TestChainDistinguishesNoRecordFromDown(t *testing.T) {
	out := chain(context.Background(), []providerCall{
		{"NoRecord", func(context.Context) ([]Result, error) { return nil, nil }},
		{"Down", func(context.Context) ([]Result, error) { return nil, errors.New("timeout") }},
		{"Match", func(context.Context) ([]Result, error) { return []Result{{Title: "Found it"}}, nil }},
	})
	if want := []string{"NoRecord", "Down", "Match"}; !equalStrings(out.Tried, want) {
		t.Errorf("Tried = %v, want %v", out.Tried, want)
	}
	if want := []string{"Down"}; !equalStrings(out.Down, want) {
		t.Errorf("Down = %v, want %v — a clean no-record must not count as down", out.Down, want)
	}
}

// TestOutcomeMessagesWordForWord checks the honest-outcome flashes the
// amendment specs, including the UX pass's shortened all-miss wording and
// down-provider naming.
func TestOutcomeMessagesWordForWord(t *testing.T) {
	cases := []struct {
		name string
		out  Outcome
		want string
	}{
		{
			name: "clean match, nothing down: silent",
			out:  Outcome{Results: []Result{{Title: "x"}}, Tried: []string{"Open Library"}},
			want: "",
		},
		{
			name: "one down, others answered: names the failure and what's shown",
			out: Outcome{
				Results: []Result{{Source: "Libris"}, {Source: "Libris"}, {Source: "Libris"}},
				Tried:   []string{"Open Library", "Google Books", "Libris"},
				Down:    []string{"Open Library"},
			},
			want: "Open Library didn't answer — showing Libris' 3 matches",
		},
		{
			name: "every provider down: the honest offline message",
			out:  Outcome{Tried: []string{"Open Library", "Google Books", "Libris"}, Down: []string{"Open Library", "Google Books", "Libris"}},
			want: "Book lookup failed (offline) — continue manually",
		},
		{
			name: "no match anywhere, nothing down: the shortened all-miss message",
			out:  Outcome{Tried: []string{"Open Library", "Google Books", "Libris"}},
			want: "No match for 'Kallocain' — none of the three has it · continue manually",
		},
	}
	for _, c := range cases {
		if got := c.out.Message("Kallocain"); got != c.want {
			t.Errorf("%s: Message = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestCapQueryShortensLongQueries is the UX pass's fix for gap 1: a long
// echoed query must not crowd the provider names out of the status line.
func TestCapQueryShortensLongQueries(t *testing.T) {
	long := "Kallocain Karin Boye 1940 first edition Albert Bonniers"
	got := capQuery(long)
	if len(got) >= len(long) {
		t.Errorf("capQuery(%q) = %q, should be shorter", long, got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("capQuery(%q) = %q, should end in an ellipsis", long, got)
	}
	if short := "Meditations"; capQuery(short) != short {
		t.Errorf("capQuery(%q) = %q, a short query shouldn't be touched", short, capQuery(short))
	}
}

// TestLibrisFreeTextFallbackFiresOnlyOnEmptyPredecessors is Amendment 1's
// provider-expansion rule: Libris' free-text search is last-resort, tried
// only once the stronger-ranked providers came up empty.
func TestLibrisFreeTextFallbackFiresOnlyOnEmptyPredecessors(t *testing.T) {
	withOpenLibraryServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"docs":[]}`))
	})
	emptyGoogleBooksServer(t)
	withLibrisServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<xsearch records="1"><collection><record>
			<datafield tag="245"><subfield code="a">Kallocain</subfield></datafield>
		</record></collection></xsearch>`))
	})
	out := lookupText(context.Background(), "kallocain")
	if len(out.Results) != 1 || out.Results[0].Source != "Libris" {
		t.Errorf("Results = %+v, want Libris' free-text fallback to fire", out.Results)
	}
}

func TestLibrisFreeTextFallbackDoesNotFireWhenOpenLibraryAnswers(t *testing.T) {
	withOpenLibraryServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"docs":[{"title":"Meditations"}]}`))
	})
	withLibrisServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Libris shouldn't be queried once an earlier provider already answered")
	})
	out := lookupText(context.Background(), "Meditations")
	if len(out.Results) != 1 || out.Results[0].Source != "Open Library" {
		t.Errorf("Results = %+v", out.Results)
	}
}

// TestBackfillsCoverFromOpenLibraryWhenLibrisWins is Amendment 1's cover-
// quality fix: Libris never carries cover art, so a Swedish-ISBN match it
// wins gets a best-effort cover from Open Library without changing whose
// metadata is used.
func TestBackfillsCoverFromOpenLibraryWhenLibrisWins(t *testing.T) {
	withLibrisServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<xsearch records="1"><collection><record>
			<datafield tag="245"><subfield code="a">Kallocain</subfield></datafield>
		</record></collection></xsearch>`))
	})
	withOpenLibraryServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"title":"Kallocain","covers":[999]}`))
	})
	out := lookupISBN(context.Background(), "978-91-0-012345-6")
	if len(out.Results) != 1 || out.Results[0].Source != "Libris" {
		t.Fatalf("Results = %+v, want the Libris match to still win", out.Results)
	}
	if out.Results[0].CoverURL == "" {
		t.Error("the cover should be backfilled from Open Library")
	}
}

func TestBackfillFallsToGoogleBooksWhenOpenLibraryHasNoCover(t *testing.T) {
	withLibrisServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<xsearch records="1"><collection><record>
			<datafield tag="245"><subfield code="a">Kallocain</subfield></datafield>
		</record></collection></xsearch>`))
	})
	withOpenLibraryServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	withGoogleBooksServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[{"volumeInfo":{"title":"Kallocain","imageLinks":{"thumbnail":"http://books.google.com/c.jpg"}}}]}`))
	})
	out := lookupISBN(context.Background(), "978-91-0-012345-6")
	if len(out.Results) != 1 || out.Results[0].CoverURL == "" {
		t.Errorf("Results = %+v, want a Google Books cover backfilled", out.Results)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
