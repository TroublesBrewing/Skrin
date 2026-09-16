package book

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestSearchGoogleBooks(t *testing.T) {
	withGoogleBooksServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[{"volumeInfo":{"title":"Meditations","authors":["Marcus Aurelius"],
			"publisher":"Penguin","publishedDate":"2006","pageCount":304,
			"industryIdentifiers":[{"type":"ISBN_13","identifier":"9780140449334"}],
			"imageLinks":{"thumbnail":"http://books.google.com/cover.jpg"}}}]}`))
	})
	results, err := SearchGoogleBooks(context.Background(), "Meditations")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results", len(results))
	}
	r := results[0]
	if r.Title != "Meditations" || r.Year != "2006" || r.Publisher != "Penguin" || r.Pages != "304" || r.ISBN != "9780140449334" {
		t.Errorf("result = %+v", r)
	}
	if r.Source != "Google Books" {
		t.Errorf("Source = %q", r.Source)
	}
	if !strings.HasPrefix(r.CoverURL, "https://") {
		t.Errorf("CoverURL = %q, should be upgraded to https", r.CoverURL)
	}
}

func TestSearchGoogleBooksNoMatchIsCleanEmptySlice(t *testing.T) {
	withGoogleBooksServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`)) // no "items" key at all: zero matches
	})
	results, err := SearchGoogleBooks(context.Background(), "asdkjfhaksjdhf")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("results = %+v, want none", results)
	}
}

func TestGoogleBooksByISBNNoRecordIsErrNoRecord(t *testing.T) {
	withGoogleBooksServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[]}`))
	})
	_, err := GoogleBooksByISBN(context.Background(), "0000000000")
	if err != ErrNoRecord {
		t.Errorf("err = %v, want ErrNoRecord", err)
	}
}
