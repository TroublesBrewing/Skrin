package book

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Outcome is what Lookup produces: matches, if any, plus enough about
// which providers were tried and which failed to answer for the honest
// per-provider messages Amendment 1 asks for. A provider that answered
// cleanly with nothing (ErrNoRecord) is "tried" but not "down" — only a
// provider that didn't answer properly (timeout, 5xx, a malformed body)
// counts as down.
type Outcome struct {
	Results []Result
	Tried   []string
	Down    []string
}

// AllDown reports whether every provider tried failed to answer — the
// only case "offline" is honest.
func (o Outcome) AllDown() bool {
	return len(o.Tried) > 0 && len(o.Down) == len(o.Tried)
}

// Message is the flash Amendment 1 asks for: "" when the lookup should
// stay quiet (a clean match, nothing down — the chooser opening speaks
// for itself), and otherwise the honest word for what happened.
func (o Outcome) Message(query string) string {
	switch {
	case len(o.Results) > 0:
		if len(o.Down) == 0 {
			return ""
		}
		return strings.Join(o.Down, " and ") + " didn't answer — showing " + resultsSummary(o.Results)
	case o.AllDown():
		return "Book lookup failed (offline) — continue manually"
	default:
		return "No match for '" + capQuery(query) + "' — none of the three has it · continue manually"
	}
}

// resultsSummary names what's on offer for the "a provider went down, but
// here's what the others found" flash: "Libris' 3 matches" when they all
// came from one source, or a plain count when they're mixed.
func resultsSummary(results []Result) string {
	counts := map[string]int{}
	var order []string
	for _, r := range results {
		if counts[r.Source] == 0 {
			order = append(order, r.Source)
		}
		counts[r.Source]++
	}
	if len(order) == 1 {
		n := counts[order[0]]
		plural := "es"
		if n == 1 {
			plural = ""
		}
		return possessive(order[0]) + fmt.Sprintf(" %d match%s", n, plural)
	}
	return fmt.Sprintf("%d matches", len(results))
}

func possessive(name string) string {
	if strings.HasSuffix(name, "s") {
		return name + "'"
	}
	return name + "'s"
}

// capQuery shortens an echoed query so a long one can't crowd the
// provider names out of the status line — the UX pass's fix for the
// all-miss flash's most important part (what was tried) getting
// truncated away.
func capQuery(query string) string {
	const max = 24
	r := []rune(query)
	if len(r) <= max {
		return query
	}
	return string(r[:max-1]) + "…"
}

// providerCall is one provider's attempt at answering a lookup, named for
// the honest messages.
type providerCall struct {
	name string
	fn   func(context.Context) ([]Result, error)
}

// chain tries each call in order, stopping at the first that returns a
// match. A provider that errors is recorded as down and the chain moves
// on; a provider that cleanly returns nothing is recorded as tried and
// the chain moves on too, without marking it down.
func chain(ctx context.Context, calls []providerCall) Outcome {
	var out Outcome
	for _, c := range calls {
		out.Tried = append(out.Tried, c.name)
		results, err := c.fn(ctx)
		if err != nil {
			out.Down = append(out.Down, c.name)
			continue
		}
		if len(results) > 0 {
			out.Results = results
			return out
		}
	}
	return out
}

// isbnCall wraps a single-Result ISBN lookup as a providerCall: its
// ErrNoRecord becomes a clean empty slice (tried, not down), and every
// other error passes through as down.
func isbnCall(name string, fn func(context.Context, string) (Result, error), isbn string) providerCall {
	return providerCall{name: name, fn: func(ctx context.Context) ([]Result, error) {
		r, err := fn(ctx, isbn)
		if errors.Is(err, ErrNoRecord) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return []Result{r}, nil
	}}
}

// Lookup searches for query across three independent, keyless providers:
// Open Library, Libris (Kungliga biblioteket) and Google Books. A search
// that looks like an ISBN tries each provider's ISBN lookup; anything
// else is a free-text search. See Amendment 1 in skrin library.md for the
// honest-outcome rules Outcome.Message implements and why Libris' weaker
// free-text ranking is tried last, not first.
func Lookup(ctx context.Context, query string) Outcome {
	if LooksLikeISBN(query) {
		return lookupISBN(ctx, query)
	}
	return lookupText(ctx, query)
}

func lookupISBN(ctx context.Context, isbn string) Outcome {
	calls := []providerCall{
		isbnCall("Open Library", OpenLibraryByISBN, isbn),
		isbnCall("Google Books", GoogleBooksByISBN, isbn),
		isbnCall("Libris", LibrisByISBN, isbn),
	}
	if IsSwedishISBN(isbn) {
		// Libris has far better coverage of Swedish ISBNs; ask it first.
		calls[0], calls[2] = calls[2], calls[0]
	}
	out := chain(ctx, calls)
	backfillCover(ctx, &out, isbn)
	return out
}

func lookupText(ctx context.Context, query string) Outcome {
	calls := []providerCall{
		{"Open Library", func(ctx context.Context) ([]Result, error) { return SearchOpenLibrary(ctx, query) }},
		{"Google Books", func(ctx context.Context) ([]Result, error) { return SearchGoogleBooks(ctx, query) }},
		{"Libris", func(ctx context.Context) ([]Result, error) { return LibrisSearch(ctx, query) }},
	}
	return chain(ctx, calls)
}

// backfillCover is Amendment 1's cover-quality fix: Libris never carries
// cover art, so an ISBN match that came from Libris (a Swedish edition
// Open Library or Google Books might still know the cover for) gets a
// best-effort, silent cover lookup from the higher-quality sources —
// Open Library first, Google Books second — without changing which
// provider's metadata won. Failure here is never fatal to the lookup:
// the match already stands on its own.
func backfillCover(ctx context.Context, out *Outcome, isbn string) {
	if len(out.Results) != 1 || out.Results[0].CoverURL != "" || out.Results[0].Source != "Libris" {
		return
	}
	if r, err := OpenLibraryByISBN(ctx, isbn); err == nil && r.CoverURL != "" {
		out.Results[0].CoverURL = r.CoverURL
		return
	}
	if r, err := GoogleBooksByISBN(ctx, isbn); err == nil && r.CoverURL != "" {
		out.Results[0].CoverURL = r.CoverURL
	}
}
