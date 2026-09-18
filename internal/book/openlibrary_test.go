package book

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func withOpenLibraryServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	old := openLibraryBase
	openLibraryBase = srv.URL
	t.Cleanup(func() { openLibraryBase = old })
}

func withLibrisServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	old := librisBase
	librisBase = srv.URL
	t.Cleanup(func() { librisBase = old })
}

func withGoogleBooksServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	old := googleBooksBase
	googleBooksBase = srv.URL
	t.Cleanup(func() { googleBooksBase = old })
}

// emptyGoogleBooksServer and emptyLibrisServer answer with a clean "no
// record" (200, empty body) — for tests that need Google Books or Libris
// out of the way without treating them as down.
func emptyGoogleBooksServer(t *testing.T) {
	t.Helper()
	withGoogleBooksServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[]}`))
	})
}

func TestSearchOpenLibrary(t *testing.T) {
	withOpenLibraryServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/search.json") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"docs":[{"title":"Meditations","author_name":["Marcus Aurelius"],
			"first_publish_year":2003,"publisher":["Modern Library"],"isbn":["0812968255"],
			"number_of_pages_median":256,"cover_i":12345}]}`))
	})
	results, err := SearchOpenLibrary(context.Background(), "Meditations")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results", len(results))
	}
	r := results[0]
	if r.Title != "Meditations" || r.Year != "2003" || r.Publisher != "Modern Library" || r.Pages != "256" {
		t.Errorf("result = %+v", r)
	}
	if r.CoverURL == "" {
		t.Error("expected a cover URL")
	}
	if !strings.Contains(r.Label(), "Meditations") || !strings.Contains(r.Detail(), "cover available") {
		t.Errorf("Label/Detail = %q / %q", r.Label(), r.Detail())
	}
}

func TestOpenLibraryByISBN(t *testing.T) {
	withOpenLibraryServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/isbn/"):
			w.Write([]byte(`{"title":"Meditations","publishers":["Modern Library"],
				"publish_date":"2003","number_of_pages":256,"authors":[{"key":"/authors/OL1A"}]}`))
		case strings.HasPrefix(r.URL.Path, "/authors/OL1A"):
			w.Write([]byte(`{"name":"Marcus Aurelius"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	r, err := OpenLibraryByISBN(context.Background(), "0812968255")
	if err != nil {
		t.Fatal(err)
	}
	if r.Title != "Meditations" || r.Year != "2003" || len(r.Authors) != 1 || r.Authors[0] != "Marcus Aurelius" {
		t.Errorf("result = %+v", r)
	}
}

func TestLookupTimesOutOffline(t *testing.T) {
	// A server that never responds simulates being offline: Lookup must
	// fail within the fixed timeout rather than hang the card. Every
	// provider points at the same hanging server, so this is the genuine
	// "every provider tried failed to answer" case Amendment 1's
	// AllDown/"offline" message is for.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond) // the client gives up first
	}))
	defer srv.Close()
	for _, base := range []*string{&openLibraryBase, &googleBooksBase, &librisBase} {
		old := *base
		*base = srv.URL
		defer func(b *string, v string) { *b = v }(base, old)
	}

	oldClient := httpClient
	httpClient = &http.Client{Timeout: 100 * time.Millisecond}
	defer func() { httpClient = oldClient }()

	start := time.Now()
	out := Lookup(context.Background(), "Meditations")
	if !out.AllDown() {
		t.Fatalf("outcome = %+v, want every provider recorded as down", out)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("Lookup took %s, should fail fast on timeout", elapsed)
	}
	if msg := out.Message("Meditations"); !strings.Contains(msg, "offline") {
		t.Errorf("Message = %q, want the offline message when every provider is down", msg)
	}
}

func TestLookupRoutesISBNAndSwedishFallback(t *testing.T) {
	if !LooksLikeISBN("978-91-1-234567-8") {
		t.Error("LooksLikeISBN should accept a Swedish ISBN-13")
	}
	if LooksLikeISBN("Meditations") {
		t.Error("LooksLikeISBN should reject a plain title")
	}
	if !IsSwedishISBN("978-91-1-234567-8") {
		t.Error("IsSwedishISBN should recognise the 978-91 prefix")
	}
	if IsSwedishISBN("0812968255") {
		t.Error("IsSwedishISBN should reject a non-Swedish ISBN")
	}
}

func TestLibrisByISBN(t *testing.T) {
	withLibrisServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<xsearch records="1"><collection><record>
			<datafield tag="245"><subfield code="a">Kallocain</subfield></datafield>
			<datafield tag="100"><subfield code="a">Karin Boye</subfield></datafield>
			<datafield tag="264"><subfield code="b">Albert Bonniers</subfield><subfield code="c">1940</subfield></datafield>
		</record></collection></xsearch>`))
	})
	r, err := LibrisByISBN(context.Background(), "9789100123456")
	if err != nil {
		t.Fatal(err)
	}
	if r.Title != "Kallocain" || r.Publisher != "Albert Bonniers" || r.Year != "1940" {
		t.Errorf("result = %+v", r)
	}
	if len(r.Authors) != 1 || r.Authors[0] != "Karin Boye" {
		t.Errorf("Authors = %v", r.Authors)
	}
}

func TestLibrisByISBNNoRecordIsNotAnError(t *testing.T) {
	// Amendment 1's central bug: Libris (and every provider) answering
	// "I don't have this" must not read as a network failure.
	withLibrisServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<xsearch records="0"><collection /></xsearch>`))
	})
	_, err := LibrisByISBN(context.Background(), "9789177237440")
	if !errors.Is(err, ErrNoRecord) {
		t.Errorf("err = %v, want ErrNoRecord for a clean zero-record answer", err)
	}
}

func TestLookupSwedishISBNPrefersLibris(t *testing.T) {
	withLibrisServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<xsearch records="1"><collection><record>
			<datafield tag="245"><subfield code="a">Kallocain</subfield></datafield>
		</record></collection></xsearch>`))
	})
	// Libris wins the metadata race and is never queried a second time, but
	// it never carries cover art, so the cover-quality backfill still
	// reaches out to Open Library for a cover alone (Amendment 1).
	withOpenLibraryServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"title":"Kallocain","covers":[999]}`))
	})
	out := Lookup(context.Background(), "978-91-0-012345-6")
	if len(out.Results) != 1 || out.Results[0].Source != "Libris" {
		t.Errorf("results = %+v, want the Libris match to still win", out.Results)
	}
	if out.Results[0].CoverURL == "" {
		t.Error("the cover should be backfilled from Open Library")
	}
	if len(out.Down) != 0 {
		t.Errorf("Down = %v, want nothing down on a first-try match", out.Down)
	}
}
