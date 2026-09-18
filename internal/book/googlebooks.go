package book

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// googleBooksBase is the Google Books volumes API, overridable in tests.
// Anonymous, no API key, a free quota — the third independent line per
// Amendment 1 to skrin library.md.
var googleBooksBase = "https://www.googleapis.com/books/v1/volumes"

type gbSearchResponse struct {
	Items []gbItem `json:"items"`
}

type gbItem struct {
	VolumeInfo gbVolumeInfo `json:"volumeInfo"`
}

type gbVolumeInfo struct {
	Title               string         `json:"title"`
	Subtitle            string         `json:"subtitle"`
	Authors             []string       `json:"authors"`
	Publisher           string         `json:"publisher"`
	PublishedDate       string         `json:"publishedDate"`
	PageCount           int            `json:"pageCount"`
	IndustryIdentifiers []gbIdentifier `json:"industryIdentifiers"`
	ImageLinks          gbImageLinks   `json:"imageLinks"`
}

type gbIdentifier struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

type gbImageLinks struct {
	Thumbnail      string `json:"thumbnail"`
	SmallThumbnail string `json:"smallThumbnail"`
}

func (v gbVolumeInfo) toResult() Result {
	r := Result{
		Title:     v.Title,
		Subtitle:  v.Subtitle,
		Authors:   v.Authors,
		Publisher: v.Publisher,
		Source:    "Google Books",
	}
	if v.PageCount > 0 {
		r.Pages = strconv.Itoa(v.PageCount)
	}
	if v.PublishedDate != "" {
		r.Year = yearFrom(v.PublishedDate)
	}
	for _, id := range v.IndustryIdentifiers {
		if id.Type == "ISBN_13" {
			r.ISBN = id.Identifier
			break
		}
		if id.Type == "ISBN_10" && r.ISBN == "" {
			r.ISBN = id.Identifier
		}
	}
	// Google serves cover thumbnails over http; the note's own embed and
	// every other provider's cover URL here are https, so upgrade it
	// rather than mix schemes.
	if v.ImageLinks.Thumbnail != "" {
		r.CoverURL = strings.Replace(v.ImageLinks.Thumbnail, "http://", "https://", 1)
	} else if v.ImageLinks.SmallThumbnail != "" {
		r.CoverURL = strings.Replace(v.ImageLinks.SmallThumbnail, "http://", "https://", 1)
	}
	return r
}

// SearchGoogleBooks queries Google Books' free-text search, returning up
// to 10 matches. A query with no matches is a clean empty slice, not an
// error — the same "no record is a result" contract as the other
// providers.
func SearchGoogleBooks(ctx context.Context, query string) ([]Result, error) {
	url := fmt.Sprintf("%s?q=%s&maxResults=10", googleBooksBase, urlQueryEscape(query))
	var resp gbSearchResponse
	if err := getJSON(ctx, url, &resp); err != nil {
		return nil, err
	}
	out := make([]Result, 0, len(resp.Items))
	for _, item := range resp.Items {
		out = append(out, item.VolumeInfo.toResult())
	}
	return out, nil
}

// GoogleBooksByISBN looks a single ISBN up. Google Books has no dedicated
// ISBN endpoint; an isbn: search term does the same job. Zero matches is
// ErrNoRecord, not a failure.
func GoogleBooksByISBN(ctx context.Context, isbn string) (Result, error) {
	isbn = NormalizeISBN(isbn)
	url := fmt.Sprintf("%s?q=isbn:%s", googleBooksBase, isbn)
	var resp gbSearchResponse
	if err := getJSON(ctx, url, &resp); err != nil {
		return Result{}, err
	}
	if len(resp.Items) == 0 {
		return Result{}, ErrNoRecord
	}
	r := resp.Items[0].VolumeInfo.toResult()
	if r.ISBN == "" {
		r.ISBN = isbn
	}
	return r, nil
}
