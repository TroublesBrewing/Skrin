package book

import "context"

// Lookup searches for query, trying Open Library first and falling back to
// Libris when Open Library has nothing — which in practice means Swedish
// domestic ISBNs and editions it doesn't carry. A search that looks like an
// ISBN goes straight to each provider's ISBN lookup; anything else is a
// free-text search (Libris has no free-text fallback, since xsearch's
// relevance ranking is far weaker than Open Library's).
func Lookup(ctx context.Context, query string) ([]Result, error) {
	if LooksLikeISBN(query) {
		return lookupISBN(ctx, query)
	}
	results, err := SearchOpenLibrary(ctx, query)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func lookupISBN(ctx context.Context, isbn string) ([]Result, error) {
	if IsSwedishISBN(isbn) {
		if r, err := LibrisByISBN(ctx, isbn); err == nil {
			return []Result{r}, nil
		}
	}
	r, err := OpenLibraryByISBN(ctx, isbn)
	if err == nil {
		return []Result{r}, nil
	}
	if IsSwedishISBN(isbn) {
		return nil, err
	}
	if r, err2 := LibrisByISBN(ctx, isbn); err2 == nil {
		return []Result{r}, nil
	}
	return nil, err
}
