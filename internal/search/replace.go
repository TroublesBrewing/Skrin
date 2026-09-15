package search

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Find returns every occurrence of needle in s as byte spans. Without
// matchCase, letters match in any case (Unicode-aware). With wholeWord, a
// match must not have a letter, digit or underscore right before or after
// it.
func Find(s, needle string, matchCase, wholeWord bool) [][2]int {
	if needle == "" {
		return nil
	}
	p := regexp.QuoteMeta(needle)
	if !matchCase {
		p = "(?i)" + p
	}
	var out [][2]int
	for _, m := range regexp.MustCompile(p).FindAllStringIndex(s, -1) {
		if wholeWord && (wordChar(lastRune(s[:m[0]])) || wordChar(firstRune(s[m[1]:]))) {
			continue
		}
		out = append(out, [2]int{m[0], m[1]})
	}
	return out
}

// Replace puts with in place of each span. Spans must be sorted and must
// not overlap, as Find returns them.
func Replace(s string, spans [][2]int, with string) string {
	var b strings.Builder
	pos := 0
	for _, sp := range spans {
		b.WriteString(s[pos:sp[0]])
		b.WriteString(with)
		pos = sp[1]
	}
	b.WriteString(s[pos:])
	return b.String()
}

func wordChar(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }

func lastRune(s string) rune {
	r, _ := utf8.DecodeLastRuneInString(s)
	return r
}

func firstRune(s string) rune {
	r, _ := utf8.DecodeRuneInString(s)
	return r
}
