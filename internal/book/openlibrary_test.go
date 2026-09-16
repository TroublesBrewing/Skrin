package book

import (
	"context"
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
	// fail within the fixed timeout rather than hang the card.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond) // the client gives up first
	}))
	defer srv.Close()
	old := openLibraryBase
	openLibraryBase = srv.URL
	defer func() { openLibraryBase = old }()

	oldClient := httpClient
	httpClient = &http.Client{Timeout: 100 * time.Millisecond}
	defer func() { httpClient = oldClient }()

	start := time.Now()
	_, err := Lookup(context.Background(), "Meditations")
	if err == nil {
		t.Fatal("expected an error from an unreachable server")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("Lookup took %s, should fail fast on timeout", elapsed)
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
		w.Write([]byte(`<result><list><record>
			<datafield tag="245"><subfield code="a">Kallocain</subfield></datafield>
			<datafield tag="100"><subfield code="a">Karin Boye</subfield></datafield>
			<datafield tag="264"><subfield code="b">Albert Bonniers</subfield><subfield code="c">1940</subfield></datafield>
		</record></list></result>`))
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

func TestLookupSwedishISBNPrefersLibris(t *testing.T) {
	withLibrisServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<result><list><record>
			<datafield tag="245"><subfield code="a">Kallocain</subfield></datafield>
		</record></list></result>`))
	})
	withOpenLibraryServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Open Library should not be queried when Libris answers a Swedish ISBN")
	})
	results, err := Lookup(context.Background(), "978-91-0-012345-6")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Source != "Libris" {
		t.Errorf("results = %+v", results)
	}
}
