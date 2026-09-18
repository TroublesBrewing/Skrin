package book

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Result is one metadata match from a provider's search or ISBN lookup.
type Result struct {
	Title       string
	Subtitle    string
	Authors     []string
	Translators []string
	Year        string // the edition's publication year
	Publisher   string
	ISBN        string
	Pages       string
	CoverURL    string // "" when no cover art is known
	Source      string // "Open Library" or "Libris", for the chooser row
}

// Label is the chooser row for a result: "Meditations · Marcus Aurelius
// (2003, Modern Library) [Cover available]".
func (r Result) Label() string {
	s := r.Title
	if len(r.Authors) > 0 {
		s += " · " + strings.Join(r.Authors, ", ")
	}
	detail := ""
	switch {
	case r.Year != "" && r.Publisher != "":
		detail = fmt.Sprintf("(%s, %s)", r.Year, r.Publisher)
	case r.Year != "":
		detail = fmt.Sprintf("(%s)", r.Year)
	case r.Publisher != "":
		detail = "(" + r.Publisher + ")"
	}
	if detail != "" {
		s += " " + detail
	}
	return s
}

// Detail is the chooser row's trailing text.
func (r Result) Detail() string {
	d := r.Source
	if r.CoverURL != "" {
		d += " · cover available"
	}
	return d
}

// timeout bounds every metadata request, so a flaky network never blocks
// the card: 3 seconds, per the offline-fallback spec.
const timeout = 3 * time.Second

// httpClient is overridable in tests, so no test ever reaches the network.
var httpClient = &http.Client{Timeout: timeout}

// openLibraryBase is Open Library's API host, overridable in tests so they
// never touch the real network.
var openLibraryBase = "https://openlibrary.org"

// NormalizeISBN strips dashes and spaces from an ISBN-10 or ISBN-13.
func NormalizeISBN(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '-' || r == ' ' {
			continue
		}
		b.WriteRune(r)
	}
	return strings.ToUpper(b.String())
}

// IsSwedishISBN reports whether isbn is in Sweden's ISBN-13 registrant
// range (978-91-...), where Libris has far better coverage than Open
// Library.
func IsSwedishISBN(isbn string) bool {
	n := NormalizeISBN(isbn)
	return strings.HasPrefix(n, "97891") || strings.HasPrefix(n, "978-91")
}

var isbnPattern = regexp.MustCompile(`^(97[89])?\d{9}[\dXx]$`)

// LooksLikeISBN reports whether s is shaped like an ISBN-10 or ISBN-13,
// so a Search box can route it to ByISBN instead of a text query.
func LooksLikeISBN(s string) bool {
	return isbnPattern.MatchString(NormalizeISBN(s))
}

// httpStatusError carries the HTTP status a provider answered with, so
// callers can tell "the server said no" (404: an honest no-record) from
// "the server didn't answer properly" (anything else non-2xx: down).
type httpStatusError struct {
	status int
	url    string
}

func (e *httpStatusError) Error() string { return fmt.Sprintf("%s: %d", e.url, e.status) }

// getJSON fetches url and decodes it as JSON into out, failing fast on a
// timeout or a non-2xx status rather than hanging the card.
func getJSON(ctx context.Context, url string, out any) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "skrin (https://github.com/lurioso/skrin)")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return &httpStatusError{status: resp.StatusCode, url: url}
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// --- Open Library ---

type olSearchResponse struct {
	Docs []olDoc `json:"docs"`
}

type olDoc struct {
	Title            string   `json:"title"`
	Subtitle         string   `json:"subtitle"`
	AuthorName       []string `json:"author_name"`
	FirstPublishYear int      `json:"first_publish_year"`
	PublishYear      []int    `json:"publish_year"`
	Publisher        []string `json:"publisher"`
	ISBN             []string `json:"isbn"`
	NumberOfPagesMed int      `json:"number_of_pages_median"`
	CoverI           int      `json:"cover_i"`
	Key              string   `json:"key"`
}

type olISBNResponse struct {
	Title         string   `json:"title"`
	Subtitle      string   `json:"subtitle"`
	Authors       []olRef  `json:"authors"`
	Publishers    []string `json:"publishers"`
	PublishDate   string   `json:"publish_date"`
	NumberOfPages int      `json:"number_of_pages"`
	Covers        []int    `json:"covers"`
	ISBN10        []string `json:"isbn_10"`
	ISBN13        []string `json:"isbn_13"`
}

type olRef struct {
	Key string `json:"key"`
}

// SearchOpenLibrary queries Open Library's search endpoint by free text
// (title, author, or a mix), returning up to 10 matches.
func SearchOpenLibrary(ctx context.Context, query string) ([]Result, error) {
	url := openLibraryBase + "/search.json?limit=10&q=" + urlQueryEscape(query)
	var resp olSearchResponse
	if err := getJSON(ctx, url, &resp); err != nil {
		return nil, err
	}
	var out []Result
	for _, d := range resp.Docs {
		r := Result{
			Title:    d.Title,
			Subtitle: d.Subtitle,
			Authors:  d.AuthorName,
			Source:   "Open Library",
		}
		if d.FirstPublishYear > 0 {
			r.Year = strconv.Itoa(d.FirstPublishYear)
		}
		if len(d.Publisher) > 0 {
			r.Publisher = d.Publisher[0]
		}
		if len(d.ISBN) > 0 {
			r.ISBN = d.ISBN[0]
		}
		if d.NumberOfPagesMed > 0 {
			r.Pages = strconv.Itoa(d.NumberOfPagesMed)
		}
		if d.CoverI > 0 {
			r.CoverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", d.CoverI)
		}
		out = append(out, r)
	}
	return out, nil
}

// OpenLibraryByISBN looks a single ISBN up directly. A 404 means Open
// Library simply doesn't hold this edition — ErrNoRecord, not a failure.
func OpenLibraryByISBN(ctx context.Context, isbn string) (Result, error) {
	isbn = NormalizeISBN(isbn)
	url := openLibraryBase + "/isbn/" + isbn + ".json"
	var resp olISBNResponse
	if err := getJSON(ctx, url, &resp); err != nil {
		var hse *httpStatusError
		if errors.As(err, &hse) && hse.status == http.StatusNotFound {
			return Result{}, ErrNoRecord
		}
		return Result{}, err
	}
	r := Result{
		Title:    resp.Title,
		Subtitle: resp.Subtitle,
		ISBN:     isbn,
		Source:   "Open Library",
	}
	if len(resp.Publishers) > 0 {
		r.Publisher = resp.Publishers[0]
	}
	if resp.NumberOfPages > 0 {
		r.Pages = strconv.Itoa(resp.NumberOfPages)
	}
	if resp.PublishDate != "" {
		r.Year = yearFrom(resp.PublishDate)
	}
	if len(resp.Covers) > 0 && resp.Covers[0] > 0 {
		r.CoverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", resp.Covers[0])
	}
	for _, a := range resp.Authors {
		if name, err := openLibraryAuthorName(ctx, a.Key); err == nil && name != "" {
			r.Authors = append(r.Authors, name)
		}
	}
	return r, nil
}

func openLibraryAuthorName(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", nil
	}
	var resp struct {
		Name string `json:"name"`
	}
	if err := getJSON(ctx, openLibraryBase+key+".json", &resp); err != nil {
		return "", err
	}
	return resp.Name, nil
}

var yearPattern = regexp.MustCompile(`\d{4}`)

func yearFrom(s string) string {
	return yearPattern.FindString(s)
}

// getXML fetches url and returns its raw body, for the one endpoint
// (Libris) that answers in XML rather than JSON.
func getXML(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "skrin (https://github.com/lurioso/skrin)")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, &httpStatusError{status: resp.StatusCode, url: url}
	}
	return io.ReadAll(resp.Body)
}

func urlQueryEscape(s string) string {
	r := strings.NewReplacer(" ", "+", "&", "%26", "#", "%23")
	return r.Replace(s)
}
