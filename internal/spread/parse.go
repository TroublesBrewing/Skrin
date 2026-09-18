package spread

import (
	"fmt"
	"strconv"
	"strings"
)

type queryKind int

const (
	kTable queryKind = iota
	kList
	kTask
)

// Query is a parsed spread.
type Query struct {
	kind   queryKind
	noID   bool    // TABLE WITHOUT ID: no note column
	fields []field // a table's columns, or a list's one value
	from   source  // nil: the whole vault
	where  expr    // nil: every note
	sort   []sortKey
	Limit  int // -1: no LIMIT
}

type field struct {
	e    expr
	name string // the column header: as written, or AS's name
}

type sortKey struct {
	e    expr
	desc bool
}

// The file.* fields a spread knows.
var fileFields = map[string]bool{
	"file.name": true, "file.link": true, "file.folder": true, "file.path": true,
	"file.tags": true, "file.mtime": true, "file.size": true,
}

type parser struct {
	toks []token
	pos  int
}

func (p *parser) peek() token { return p.toks[p.pos] }

func (p *parser) next() token {
	t := p.toks[p.pos]
	if t.kind != tEOF {
		p.pos++
	}
	return t
}

func (p *parser) peekOp(ops ...string) bool {
	t := p.peek()
	if t.kind != tOp {
		return false
	}
	for _, o := range ops {
		if t.text == o {
			return true
		}
	}
	return false
}

func errAt(t token, format string, a ...any) error {
	return &Error{t.line, fmt.Sprintf(format, a...)}
}

func isClause(t token) bool {
	for _, kw := range []string{"FROM", "WHERE", "SORT", "LIMIT", "GROUP", "FLATTEN"} {
		if t.is(kw) {
			return true
		}
	}
	return false
}

func (p *parser) atClauseOrEnd() bool { return p.peek().kind == tEOF || isClause(p.peek()) }

// Parse reads the text inside a spread block.
func Parse(src string) (*Query, error) {
	toks, err := lex(src)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	q := &Query{Limit: -1}
	t := p.next()
	switch {
	case t.is("TABLE"):
		q.kind = kTable
		if p.peek().is("WITHOUT") {
			p.next()
			if id := p.next(); !id.is("ID") {
				return nil, errAt(id, "expected ID after WITHOUT, found %s", id)
			}
			q.noID = true
		}
		if !p.atClauseOrEnd() {
			for {
				f, err := p.field()
				if err != nil {
					return nil, err
				}
				q.fields = append(q.fields, f)
				if p.peek().kind != tComma {
					break
				}
				p.next()
			}
		}
		if q.noID && len(q.fields) == 0 {
			return nil, errAt(t, "TABLE WITHOUT ID needs at least one field")
		}
	case t.is("LIST"):
		q.kind = kList
		if !p.atClauseOrEnd() {
			f, err := p.field()
			if err != nil {
				return nil, err
			}
			q.fields = []field{f}
			if c := p.peek(); c.kind == tComma {
				return nil, errAt(c, "LIST shows one value per note — use TABLE for more")
			}
		}
	case t.is("TASK"):
		q.kind = kTask
		if !p.atClauseOrEnd() {
			return nil, errAt(p.peek(), "TASK takes no fields — put conditions in WHERE")
		}
	case t.is("CALENDAR"):
		return nil, errAt(t, "CALENDAR isn't supported yet")
	case t.kind == tEOF:
		return nil, errAt(t, "empty — start with TABLE, LIST or TASK")
	default:
		return nil, errAt(t, "expected TABLE, LIST or TASK, found %s", t)
	}

	haveFrom := false
	for {
		t := p.next()
		switch {
		case t.kind == tEOF:
			return q, nil
		case t.is("FROM"):
			if haveFrom {
				return nil, errAt(t, "FROM is given twice")
			}
			haveFrom = true
			if q.from, err = p.source(); err != nil {
				return nil, err
			}
		case t.is("WHERE"):
			e, err := p.expr()
			if err != nil {
				return nil, err
			}
			if q.where == nil {
				q.where = e
			} else {
				q.where = andE{q.where, e}
			}
		case t.is("SORT"):
			for {
				e, err := p.expr()
				if err != nil {
					return nil, err
				}
				k := sortKey{e: e}
				switch n := p.peek(); {
				case n.is("ASC"), n.is("ASCENDING"):
					p.next()
				case n.is("DESC"), n.is("DESCENDING"):
					p.next()
					k.desc = true
				}
				q.sort = append(q.sort, k)
				if p.peek().kind != tComma {
					break
				}
				p.next()
			}
		case t.is("LIMIT"):
			n := p.next()
			v, err := strconv.Atoi(n.text)
			if n.kind != tNumber || err != nil {
				return nil, errAt(n, "LIMIT needs a whole number, found %s", n)
			}
			q.Limit = v
		case t.is("GROUP"):
			return nil, errAt(t, "GROUP BY isn't supported yet")
		case t.is("FLATTEN"):
			return nil, errAt(t, "FLATTEN isn't supported yet")
		default:
			return nil, errAt(t, "expected FROM, WHERE, SORT or LIMIT, found %s", t)
		}
	}
}

// field is one column: an expression, named by its own text or by AS.
func (p *parser) field() (field, error) {
	start := p.pos
	e, err := p.expr()
	if err != nil {
		return field{}, err
	}
	f := field{e: e, name: p.text(start, p.pos)}
	if p.peek().is("AS") {
		p.next()
		n := p.next()
		if n.kind != tString && n.kind != tIdent {
			return field{}, errAt(n, "AS needs a name after it, found %s", n)
		}
		f.name = n.text
	}
	return f, nil
}

// text is tokens [start, end) as they'd be written.
func (p *parser) text(start, end int) string {
	var b strings.Builder
	for i := start; i < end; i++ {
		t := p.toks[i]
		if i > start && t.kind != tRParen && t.kind != tComma && p.toks[i-1].kind != tLParen {
			b.WriteByte(' ')
		}
		switch t.kind {
		case tString, tTag, tLink:
			b.WriteString(t.String())
		default:
			b.WriteString(t.text)
		}
	}
	return b.String()
}

// --- WHERE, SORT and field expressions ---------------------------------

func (p *parser) expr() (expr, error) { return p.or() }

func (p *parser) or() (expr, error) {
	a, err := p.and()
	for err == nil && (p.peek().is("OR") || p.peekOp("|")) {
		p.next()
		var b expr
		if b, err = p.and(); err == nil {
			a = orE{a, b}
		}
	}
	return a, err
}

func (p *parser) and() (expr, error) {
	a, err := p.not()
	for err == nil && (p.peek().is("AND") || p.peekOp("&")) {
		p.next()
		var b expr
		if b, err = p.not(); err == nil {
			a = andE{a, b}
		}
	}
	return a, err
}

func (p *parser) not() (expr, error) {
	if p.peek().is("NOT") || p.peekOp("!") {
		p.next()
		e, err := p.not()
		return notE{e}, err
	}
	return p.cmp()
}

func (p *parser) cmp() (expr, error) {
	a, err := p.primary()
	if err != nil {
		return nil, err
	}
	if p.peekOp("=", "!=", "<", "<=", ">", ">=") {
		op := p.next().text
		b, err := p.primary()
		if err != nil {
			return nil, err
		}
		a = cmpE{op, a, b}
	}
	if p.peekOp("+", "-", "*", "/") {
		return nil, errAt(p.peek(), "arithmetic (+ - * /) isn't supported yet")
	}
	return a, nil
}

func (p *parser) primary() (expr, error) {
	t := p.next()
	switch t.kind {
	case tLParen:
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		if c := p.next(); c.kind != tRParen {
			return nil, errAt(c, "expected ), found %s", c)
		}
		return e, nil
	case tString:
		return lit{text(t.text)}, nil
	case tNumber:
		return lit{coerce(t.text)}, nil
	case tLink:
		return linkLit{t.text}, nil
	case tTag:
		return nil, errAt(t, "a tag here needs quotes: \"#%s\"", t.text)
	case tOp:
		if t.text == "-" && p.peek().kind == tNumber {
			return lit{coerce("-" + p.next().text)}, nil
		}
	case tIdent:
		switch lower := strings.ToLower(t.text); {
		case lower == "null":
			return lit{value{}}, nil
		case lower == "true", lower == "false":
			return lit{value{k: vBool, b: lower == "true", raw: lower}}, nil
		case p.peek().kind == tLParen:
			return p.call(t)
		case isClause(t), t.is("AND"), t.is("OR"), t.is("AS"):
			return nil, errAt(t, "expected a value, found %s", t)
		case lower == "file.ctime", lower == "file.cday":
			return nil, errAt(t, "%s isn't available: Linux can't tell when a note was created", lower)
		case strings.HasPrefix(lower, "file."):
			if !fileFields[lower] {
				return nil, errAt(t, "%s isn't supported yet", lower)
			}
		case lower == "this" || strings.HasPrefix(lower, "this."):
			return nil, errAt(t, "this isn't supported yet")
		}
		return fieldRef{strings.ToLower(t.text)}, nil
	}
	return nil, errAt(t, "expected a value, found %s", t)
}

func (p *parser) call(name token) (expr, error) {
	fn := strings.ToLower(name.text)
	if fn != "contains" && fn != "icontains" {
		return nil, errAt(name, "%s() isn't supported yet", fn)
	}
	p.next() // (
	var args []expr
	for p.peek().kind != tRParen {
		a, err := p.expr()
		if err != nil {
			return nil, err
		}
		args = append(args, a)
		if p.peek().kind != tComma {
			break
		}
		p.next()
	}
	if c := p.next(); c.kind != tRParen {
		return nil, errAt(c, "expected ) after %s(…), found %s", fn, c)
	}
	if len(args) != 2 {
		return nil, errAt(name, "%s() takes two values: %s(field, value)", fn, fn)
	}
	return containsE{args[0], args[1], fn == "icontains"}, nil
}

// --- FROM ---------------------------------------------------------------

func (p *parser) source() (source, error) {
	a, err := p.andSource()
	for err == nil && (p.peek().is("OR") || p.peekOp("|")) {
		p.next()
		var b source
		if b, err = p.andSource(); err == nil {
			a = orS{a, b}
		}
	}
	return a, err
}

func (p *parser) andSource() (source, error) {
	a, err := p.unarySource()
	for err == nil && (p.peek().is("AND") || p.peekOp("&")) {
		p.next()
		var b source
		if b, err = p.unarySource(); err == nil {
			a = andS{a, b}
		}
	}
	return a, err
}

func (p *parser) unarySource() (source, error) {
	t := p.next()
	switch {
	case t.kind == tOp && (t.text == "-" || t.text == "!"):
		s, err := p.unarySource()
		return notS{s}, err
	case t.kind == tLParen:
		s, err := p.source()
		if err != nil {
			return nil, err
		}
		if c := p.next(); c.kind != tRParen {
			return nil, errAt(c, "expected ), found %s", c)
		}
		return s, nil
	case t.kind == tTag:
		return tagS{t.text}, nil
	case t.kind == tString:
		return folderS{strings.Trim(t.text, "/")}, nil
	case t.kind == tLink:
		return linksToS{t.text}, nil
	case t.is("outgoing"):
		if c := p.next(); c.kind != tLParen {
			return nil, errAt(c, "expected ( after outgoing, found %s", c)
		}
		l := p.next()
		if l.kind != tLink {
			return nil, errAt(l, "outgoing() takes a [[note]], found %s", l)
		}
		if c := p.next(); c.kind != tRParen {
			return nil, errAt(c, "expected ) after outgoing([[…]], found %s", c)
		}
		return linksFromS{l.text}, nil
	}
	return nil, errAt(t, `expected a #tag, a "folder" or a [[note]], found %s`, t)
}
