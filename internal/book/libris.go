package book

import (
	"context"
	"encoding/xml"
	"fmt"
)

// librisBase is Libris's xsearch API host, overridable in tests.
var librisBase = "https://libris.kb.se"

// Libris is the Swedish national library's (Kungliga biblioteket) open
// xsearch API. It has far better coverage of Swedish ISBNs, publishers,
// edition years and translators than Open Library, so it's tried as a
// fallback for 978-91-... ISBNs, or whenever Open Library comes up empty.
type librisResponse struct {
	XMLName xml.Name       `xml:"xsearch"`
	Records []librisRecord `xml:"collection>record"`
}

type librisRecord struct {
	Fields []librisField `xml:"datafield"`
}

type librisField struct {
	Tag       string           `xml:"tag,attr"`
	Subfields []librisSubfield `xml:"subfield"`
}

type librisSubfield struct {
	Code string `xml:"code,attr"`
	Text string `xml:",chardata"`
}

func (f librisField) sub(code string) string {
	for _, s := range f.Subfields {
		if s.Code == code {
			return s.Text
		}
	}
	return ""
}

// resultFrom maps one MARC21-XML record to a Result.
func (rec librisRecord) resultFrom(isbn string) Result {
	r := Result{ISBN: isbn, Source: "Libris"}
	for _, f := range rec.Fields {
		switch f.Tag {
		case "245": // title statement
			r.Title = f.sub("a")
			r.Subtitle = f.sub("b")
		case "100", "700": // main / added author entry
			if name := f.sub("a"); name != "" {
				if role := f.sub("e"); role == "translator" || role == "övers." {
					r.Translators = append(r.Translators, name)
				} else {
					r.Authors = append(r.Authors, name)
				}
			}
		case "264", "260": // publication statement
			if pub := f.sub("b"); pub != "" {
				r.Publisher = pub
			}
			if year := f.sub("c"); year != "" {
				r.Year = yearFrom(year)
			}
		case "300": // physical description
			if pages := f.sub("a"); pages != "" {
				r.Pages = yearFrom(pages) // reuses the digit-run matcher
			}
		}
	}
	return r
}

// LibrisByISBN looks a single ISBN up in Libris's MARC21-XML xsearch
// endpoint. Zero records is ErrNoRecord — Libris answered, it just
// doesn't hold this edition — not a failure.
func LibrisByISBN(ctx context.Context, isbn string) (Result, error) {
	isbn = NormalizeISBN(isbn)
	url := fmt.Sprintf("%s/xsearch?query=isbn:%s&format=marcxml&n=1", librisBase, isbn)
	body, err := getXML(ctx, url)
	if err != nil {
		return Result{}, err
	}
	var resp librisResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return Result{}, err
	}
	if len(resp.Records) == 0 {
		return Result{}, ErrNoRecord
	}
	return resp.Records[0].resultFrom(isbn), nil
}

// LibrisSearch is the free-text fallback Amendment 1 adds: xsearch's
// relevance ranking is weaker than Open Library's, which is why it's
// tried last, but for a Swedish-language query it's often the only line
// that has anything at all (Open Library's Swedish coverage is sparse).
// Zero matches is a clean empty slice, not an error.
func LibrisSearch(ctx context.Context, query string) ([]Result, error) {
	url := fmt.Sprintf("%s/xsearch?query=%s&format=marcxml&n=10", librisBase, urlQueryEscape(query))
	body, err := getXML(ctx, url)
	if err != nil {
		return nil, err
	}
	var resp librisResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	out := make([]Result, 0, len(resp.Records))
	for _, rec := range resp.Records {
		out = append(out, rec.resultFrom(""))
	}
	return out, nil
}
