package spread

import (
	"fmt"
	"strings"
	"unicode"
)

type tokKind int

const (
	tEOF    tokKind = iota
	tIdent          // a field, a keyword or a function name: rating, file.name, WHERE
	tString         // "text", without the quotes
	tNumber         // 4, 3.5
	tTag            // #books/fiction, without the '#'
	tLink           // [[Note]], just the target: no alias, no #heading
	tOp             // = != < <= > >= ! - & | + * /
	tLParen
	tRParen
	tComma
)

type token struct {
	kind tokKind
	text string
	line int
}

func (t token) String() string {
	switch t.kind {
	case tEOF:
		return "the end of the spread"
	case tString:
		return `"` + t.text + `"`
	case tTag:
		return "#" + t.text
	case tLink:
		return "[[" + t.text + "]]"
	}
	return `"` + t.text + `"`
}

// is reports whether t is keyword kw, in any case.
func (t token) is(kw string) bool { return t.kind == tIdent && strings.EqualFold(t.text, kw) }

func lex(src string) ([]token, error) {
	var toks []token
	r := []rune(src)
	line := 1
	for i := 0; i < len(r); {
		c := r[i]
		start := i
		add := func(k tokKind, text string) { toks = append(toks, token{k, text, line}) }
		switch {
		case c == '\n':
			line++
			i++
		case unicode.IsSpace(c):
			i++
		case c == '"':
			var b strings.Builder
			i++
			for i < len(r) && r[i] != '"' && r[i] != '\n' {
				if r[i] == '\\' && i+1 < len(r) {
					i++
				}
				b.WriteRune(r[i])
				i++
			}
			if i >= len(r) || r[i] != '"' {
				return nil, &Error{line, "a string isn't closed with \""}
			}
			i++
			add(tString, b.String())
		case c == '#':
			i++
			for i < len(r) && isTagRune(r[i]) {
				i++
			}
			if i == start+1 {
				return nil, &Error{line, `"#" needs a tag name after it`}
			}
			add(tTag, strings.ToLower(string(r[start+1:i])))
		case c == '[' && i+1 < len(r) && r[i+1] == '[':
			j := i + 2
			for j+1 < len(r) && r[j] != '\n' && !(r[j] == ']' && r[j+1] == ']') {
				j++
			}
			if j+1 >= len(r) || r[j] != ']' || r[j+1] != ']' {
				return nil, &Error{line, "a [[link]] isn't closed with ]]"}
			}
			inner := string(r[i+2 : j])
			i = j + 2
			target, _, _ := strings.Cut(inner, "|")
			target, _, _ = strings.Cut(target, "#")
			add(tLink, strings.TrimSpace(target))
		case c >= '0' && c <= '9':
			for i < len(r) && (r[i] >= '0' && r[i] <= '9' || r[i] == '.') {
				i++
			}
			add(tNumber, string(r[start:i]))
		case unicode.IsLetter(c) || c == '_':
			for i < len(r) && (unicode.IsLetter(r[i]) || unicode.IsDigit(r[i]) || r[i] == '_' || r[i] == '.' || r[i] == '-') {
				i++
			}
			add(tIdent, string(r[start:i]))
		case c == '(':
			i++
			add(tLParen, "(")
		case c == ')':
			i++
			add(tRParen, ")")
		case c == ',':
			i++
			add(tComma, ",")
		case strings.ContainsRune("!<>", c) && i+1 < len(r) && r[i+1] == '=':
			i += 2
			add(tOp, string(r[start:i]))
		case strings.ContainsRune("=<>!-&|+*/", c):
			i++
			add(tOp, string(c))
		default:
			return nil, &Error{line, fmt.Sprintf("unexpected %q", string(c))}
		}
	}
	return append(toks, token{tEOF, "", line}), nil
}

func isTagRune(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' || c == '-' || c == '/'
}
