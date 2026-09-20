// Package search parses Obsidian-style search queries and matches them
// against notes. It also finds and replaces plain text.
package search

import (
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// Doc is a note as a query sees it.
type Doc struct {
	Rel   string
	Lines []string
	Tags  []string            // lower-case, without '#'
	Props map[string][]string // lower-case keys; values as written
}

// Hit is a matching line and where on it the query's words matched.
type Hit struct {
	Line  int
	Spans [][2]int // byte offsets in the line
}

type kind int

const (
	kText kind = iota
	kTag
	kProp
	kPath
	kFile
)

type term struct {
	kind  kind
	neg   bool
	value string // lower-case for everything but text
	key   string // kProp
	re    *regexp.Regexp
	bad   string // why this term couldn't be read, for Query.Problem
}

// Query is an OR of groups whose terms must all hold.
type Query struct {
	groups [][]term
	bad    []string // terms that couldn't be read, e.g. a broken /regex/
}

// Problem is what went wrong in the query, or "" when nothing did. A term
// that can't be read is left out rather than matching nothing, so the rest
// of the query still works while this says what was dropped.
func (q Query) Problem() string {
	if len(q.bad) == 0 {
		return ""
	}
	return q.bad[0]
}

// Empty reports a query with nothing to look for.
func (q Query) Empty() bool { return len(q.groups) == 0 }

// Parse reads a query. Unfinished input (a quote still open while typing)
// is read as far as it goes, so Parse never fails.
//
//	words               all must appear, in the note or its name
//	"a phrase"          the exact phrase
//	-word               must not appear (works with every form below)
//	a OR b              either side
//	#tag, tag:tag       the tag, or a tag nested under it
//	[key], [key:value]  has the property, or one containing value
//	path:text           the note's path contains text
//	file:text           the note's file name contains text
//	/regex/             a regular expression, as in Obsidian
func Parse(s string, matchCase bool) Query {
	var q Query
	var group []term
	for _, t := range tokenize(s) {
		if t.raw == "OR" && !t.quoted && !t.bracket {
			if len(group) > 0 {
				q.groups = append(q.groups, group)
				group = nil
			}
			continue
		}
		tm, ok := parseTerm(t, matchCase)
		switch {
		case tm.bad != "":
			q.bad = append(q.bad, tm.bad)
		case ok:
			group = append(group, tm)
		}
	}
	if len(group) > 0 {
		q.groups = append(q.groups, group)
	}
	return q
}

type token struct {
	raw                         string
	neg, quoted, bracket, regex bool
}

// hasClosingSlash says whether a / at i closes again later on, so a lone
// slash in a query is still ordinary text.
func hasClosingSlash(rs []rune, i int) bool {
	for j := i + 1; j < len(rs); j++ {
		if rs[j] == '/' {
			return j > i+1
		}
	}
	return false
}

func tokenize(s string) []token {
	var out []token
	rs := []rune(s)
	for i := 0; i < len(rs); {
		if unicode.IsSpace(rs[i]) {
			i++
			continue
		}
		var t token
		if rs[i] == '-' && i+1 < len(rs) && !unicode.IsSpace(rs[i+1]) {
			t.neg = true
			i++
		}
		switch {
		case rs[i] == '/' && hasClosingSlash(rs, i):
			// /a regex/ holds together across spaces, the way a "phrase"
			// does: the slashes are the quotes.
			j := i + 1
			for rs[j] != '/' {
				j++
			}
			t.raw, t.regex = string(rs[i+1:j]), true
			i = j + 1
		case rs[i] == '"':
			j := i + 1
			for j < len(rs) && rs[j] != '"' {
				j++
			}
			t.raw, t.quoted = string(rs[i+1:j]), true
			i = j + 1
		case rs[i] == '[':
			j := i + 1
			for j < len(rs) && rs[j] != ']' {
				j++
			}
			t.raw, t.bracket = string(rs[i+1:j]), true
			i = j + 1
		default:
			j := i
			for j < len(rs) && !unicode.IsSpace(rs[j]) {
				if rs[j] == '"' { // path:"My folder"
					j++
					for j < len(rs) && rs[j] != '"' {
						j++
					}
				}
				if j < len(rs) {
					j++
				}
			}
			t.raw = strings.ReplaceAll(string(rs[i:j]), `"`, "")
			i = j
		}
		out = append(out, t)
	}
	return out
}

func parseTerm(t token, matchCase bool) (term, bool) {
	tm := term{neg: t.neg}
	raw := t.raw
	switch {
	case t.regex:
		return regexTerm(tm, raw, matchCase)
	case t.bracket:
		k, v, _ := strings.Cut(raw, ":")
		tm.kind, tm.key, tm.value = kProp, strings.ToLower(strings.TrimSpace(k)), strings.ToLower(strings.TrimSpace(v))
		return tm, tm.key != ""
	case t.quoted:
		return textTerm(tm, raw, matchCase)
	case len(raw) > 1 && raw[0] == '#':
		tm.kind, tm.value = kTag, strings.ToLower(raw[1:])
		return tm, true
	}
	if k, v, ok := strings.Cut(raw, ":"); ok {
		switch strings.ToLower(k) {
		case "tag":
			tm.kind, tm.value = kTag, strings.ToLower(strings.TrimPrefix(v, "#"))
			return tm, true
		case "path":
			tm.kind, tm.value = kPath, strings.ToLower(v)
			return tm, true
		case "file":
			tm.kind, tm.value = kFile, strings.ToLower(v)
			return tm, true
		}
	}
	return textTerm(tm, raw, matchCase)
}

// regexTerm reads /pattern/, as Obsidian's search does. A pattern that
// won't compile says so through Query.Problem instead of quietly matching
// nothing — a search that lies is worse than one that refuses.
func regexTerm(tm term, pat string, matchCase bool) (term, bool) {
	p := pat
	if !matchCase {
		p = "(?i)" + p
	}
	re, err := regexp.Compile(p)
	if err != nil {
		tm.bad = "/" + pat + "/ isn't a regular expression: " + cleanRegexErr(err)
		return tm, false
	}
	tm.kind, tm.value, tm.re = kText, pat, re
	return tm, true
}

// cleanRegexErr drops Go's "error parsing regexp: " prefix and the echo of
// the pattern, leaving what is actually wrong.
func cleanRegexErr(err error) string {
	s := strings.TrimPrefix(err.Error(), "error parsing regexp: ")
	if i := strings.LastIndex(s, ": `"); i > 0 {
		s = s[:i]
	}
	return s
}

func textTerm(tm term, raw string, matchCase bool) (term, bool) {
	if raw == "" {
		return tm, false
	}
	p := regexp.QuoteMeta(raw)
	if !matchCase {
		p = "(?i)" + p
	}
	tm.kind, tm.value, tm.re = kText, raw, regexp.MustCompile(p)
	return tm, true
}

// Match reports whether the note matches, and the lines where the query's
// words were found.
func (q Query) Match(d Doc) (bool, []Hit) {
	matched := false
	byLine := map[int][][2]int{}
	for _, g := range q.groups {
		if !groupHolds(g, d) {
			continue
		}
		matched = true
		for _, t := range g {
			if t.kind != kText || t.neg {
				continue
			}
			for i, l := range d.Lines {
				for _, s := range t.re.FindAllStringIndex(l, -1) {
					byLine[i] = append(byLine[i], [2]int{s[0], s[1]})
				}
			}
		}
	}
	if !matched {
		return false, nil
	}
	hits := make([]Hit, 0, len(byLine))
	for line, spans := range byLine {
		hits = append(hits, Hit{Line: line, Spans: merge(spans)})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Line < hits[j].Line })
	return true, hits
}

func groupHolds(g []term, d Doc) bool {
	for _, t := range g {
		if t.holds(d) == t.neg {
			return false
		}
	}
	return true
}

func (t term) holds(d Doc) bool {
	switch t.kind {
	case kText:
		if t.re.MatchString(name(d.Rel)) {
			return true
		}
		for _, l := range d.Lines {
			if t.re.MatchString(l) {
				return true
			}
		}
	case kTag:
		for _, tag := range d.Tags {
			if tag == t.value || strings.HasPrefix(tag, t.value+"/") {
				return true
			}
		}
	case kProp:
		vals, ok := d.Props[t.key]
		if !ok {
			return false
		}
		if t.value == "" {
			return true
		}
		for _, v := range vals {
			if strings.Contains(strings.ToLower(v), t.value) {
				return true
			}
		}
	case kPath:
		return strings.Contains(strings.ToLower(d.Rel), t.value)
	case kFile:
		return strings.Contains(strings.ToLower(name(d.Rel)), t.value)
	}
	return false
}

// name is a note's file name without .md.
func name(rel string) string {
	b := path.Base(rel)
	if strings.EqualFold(path.Ext(b), ".md") {
		return b[:len(b)-3]
	}
	return b
}

// merge sorts spans and joins overlapping ones.
func merge(spans [][2]int) [][2]int {
	sort.Slice(spans, func(i, j int) bool { return spans[i][0] < spans[j][0] })
	var out [][2]int
	for _, s := range spans {
		if n := len(out); n > 0 && s[0] <= out[n-1][1] {
			out[n-1][1] = max(out[n-1][1], s[1])
			continue
		}
		out = append(out, s)
	}
	return out
}
