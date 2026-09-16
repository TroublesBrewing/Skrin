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
	XMLName xml.Name      `xml:"result"`
	Records []librisRecord `xml:"list>record"`
}

type librisRecord struct {
	Fields []librisField `xml:"datafield"`
}

type librisField struct {
	Tag       string             `xml:"tag,attr"`
	Subfields []librisSubfield   `xml:"subfield"`
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

// LibrisByISBN looks a single ISBN up in Libris's MARC21-XML xsearch
// endpoint.
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
		return Result{}, fmt.Errorf("libris: no record for %s", isbn)
	}
	r := Result{ISBN: isbn, Source: "Libris"}
	for _, f := range resp.Records[0].Fields {
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
	return r, nil
}
