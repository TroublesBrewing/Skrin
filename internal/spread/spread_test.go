package spread

import (
	"path"
	"strings"
	"testing"
	"time"
)

// fakeVault is a small library: books with ratings and statuses, a
// reading log that links to two of them, and a daily note.
type fakeVault struct {
	notes []Note
	links map[string][]string // note → notes it links to
}

func (v fakeVault) Notes() []Note { return append([]Note(nil), v.notes...) }

func (v fakeVault) Resolve(target, _ string) (string, bool) {
	for _, n := range v.notes {
		if strings.EqualFold(strings.TrimSuffix(path.Base(n.Rel), ".md"), target) ||
			strings.EqualFold(strings.TrimSuffix(n.Rel, ".md"), target) {
			return n.Rel, true
		}
	}
	return "", false
}

func (v fakeVault) Backlinks(rel string) []string {
	var out []string
	for src, to := range v.links {
		for _, t := range to {
			if t == rel {
				out = append(out, src)
			}
		}
	}
	return out
}

func (v fakeVault) Outgoing(rel string) []string { return v.links[rel] }

func day(s string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02", s, time.Local)
	return t
}

var library = fakeVault{
	notes: []Note{
		{Rel: "Books/Dune.md", Tags: []string{"books/fiction"}, Props: map[string][]string{
			"author": {"Frank Herbert"}, "rating": {"5"}, "status": {"read"}, "finished": {"2026-03-01"}}, Mod: day("2026-09-01"), Size: 120},
		{Rel: "Books/Kallocain.md", Tags: []string{"books/fiction"}, Props: map[string][]string{
			"author": {"Karin Boye"}, "rating": {"4"}, "status": {"reading"}}, Mod: day("2026-09-10"), Size: 80},
		{Rel: "Books/Thinking.md", Tags: []string{"books/nonfiction"}, Props: map[string][]string{
			"author": {"Daniel Kahneman"}, "rating": {"3"}, "status": {"read"}, "finished": {"2026-06-15"}}, Mod: day("2026-08-20"), Size: 300},
		{Rel: "Books/Unrated.md", Tags: []string{"books"}, Props: map[string][]string{
			"author": {"Anon | Co"}}, Mod: day("2026-07-01"), Size: 10},
		{Rel: "Reading log.md", Tags: []string{"log"}, Mod: day("2026-09-17"), Size: 50},
		{Rel: "Daily/2026-09-18.md", Mod: day("2026-09-18"), Size: 20},
	},
	links: map[string][]string{
		"Reading log.md":      {"Books/Dune.md", "Books/Kallocain.md"},
		"Daily/2026-09-18.md": {"Reading log.md"},
	},
}

func run(t *testing.T, q string) Result {
	t.Helper()
	return Run(q, library, "Daily/2026-09-18.md")
}

// names lists the notes a result shows, in order, by the name in each link.
func names(md string) []string {
	var out []string
	for _, l := range strings.Split(md, "\n") {
		if i := strings.Index(l, "[["); i >= 0 {
			rest := l[i+2:]
			end := strings.Index(rest, "]]")
			_, name, _ := strings.Cut(strings.ReplaceAll(rest[:end], `\|`, "|"), "|")
			out = append(out, name)
		}
	}
	return out
}

func wantNames(t *testing.T, q string, want ...string) {
	t.Helper()
	r := run(t, q)
	if r.Err != "" {
		t.Fatalf("%s: %s", q, r.Err)
	}
	if got := names(r.Markdown); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("%s\n got  %v\n want %v", q, got, want)
	}
}

func TestFromTagsIncludesNestedTags(t *testing.T) {
	wantNames(t, "LIST FROM #books", "Dune", "Kallocain", "Thinking", "Unrated")
	wantNames(t, "LIST FROM #books/fiction", "Dune", "Kallocain")
	wantNames(t, "LIST FROM #BOOKS/Fiction", "Dune", "Kallocain")
}

func TestFromFoldersAndLinksAndLogic(t *testing.T) {
	wantNames(t, `LIST FROM "Books"`, "Dune", "Kallocain", "Thinking", "Unrated")
	wantNames(t, `LIST FROM "Books/"`, "Dune", "Kallocain", "Thinking", "Unrated")
	wantNames(t, `LIST FROM ""`, "Dune", "Kallocain", "Thinking", "Unrated", "2026-09-18", "Reading log")
	wantNames(t, `LIST FROM [[Dune]]`, "Reading log")
	wantNames(t, `LIST FROM outgoing([[Reading log]])`, "Dune", "Kallocain")
	wantNames(t, `LIST FROM #books AND -#books/fiction`, "Thinking", "Unrated")
	wantNames(t, `LIST FROM #log OR "Daily"`, "2026-09-18", "Reading log")
	wantNames(t, `LIST FROM #books AND !(#books/fiction OR #books/nonfiction)`, "Unrated")
	wantNames(t, `LIST FROM [[Nowhere]]`)
}

func TestWhereComparesNumbersDatesAndText(t *testing.T) {
	wantNames(t, "LIST FROM #books WHERE rating >= 4", "Dune", "Kallocain")
	wantNames(t, "LIST FROM #books WHERE rating < 4", "Thinking")
	wantNames(t, `LIST FROM #books WHERE status = "read"`, "Dune", "Thinking")
	wantNames(t, `LIST FROM #books WHERE finished > "2026-04-01"`, "Thinking")
	wantNames(t, `LIST FROM #books WHERE file.mtime >= "2026-09-01"`, "Dune", "Kallocain")
	wantNames(t, "LIST FROM #books WHERE file.size > 100", "Dune", "Thinking")
	wantNames(t, `LIST FROM #books WHERE rating = "5"`, "Dune")
}

func TestMismatchedKindsAreFalseNotErrors(t *testing.T) {
	wantNames(t, `LIST FROM #books WHERE rating > "high"`)
	wantNames(t, `LIST FROM #books WHERE status != 4`, "Dune", "Kallocain", "Thinking", "Unrated")
}

func TestNullAndBooleanLogic(t *testing.T) {
	wantNames(t, "LIST FROM #books WHERE rating = null", "Unrated")
	wantNames(t, "LIST FROM #books WHERE rating != null", "Dune", "Kallocain", "Thinking")
	wantNames(t, "LIST FROM #books WHERE finished", "Dune", "Thinking")
	wantNames(t, "LIST FROM #books WHERE !finished", "Kallocain", "Unrated")
	wantNames(t, `LIST FROM #books WHERE rating >= 4 AND status = "reading"`, "Kallocain")
	wantNames(t, `LIST FROM #books WHERE rating = 3 OR status = "reading"`, "Kallocain", "Thinking")
	wantNames(t, `LIST FROM #books WHERE NOT (rating = 3 OR status = "reading")`, "Dune", "Unrated")
	wantNames(t, `LIST FROM #books WHERE rating >= 4 & status = "read"`, "Dune")
}

func TestContains(t *testing.T) {
	wantNames(t, `LIST FROM #books WHERE contains(file.tags, "#books/nonfiction")`, "Thinking")
	wantNames(t, `LIST WHERE contains(file.tags, "#books") AND contains(author, "Boye")`, "Kallocain")
	wantNames(t, `LIST FROM #books WHERE contains(author, "boye")`)
	wantNames(t, `LIST FROM #books WHERE icontains(author, "boye")`, "Kallocain")
	wantNames(t, `LIST FROM #books WHERE contains(file.name, "in")`, "Kallocain", "Thinking")
}

func TestSortAndLimit(t *testing.T) {
	wantNames(t, "LIST FROM #books SORT rating DESC", "Dune", "Kallocain", "Thinking", "Unrated")
	wantNames(t, "LIST FROM #books SORT rating ASC", "Thinking", "Kallocain", "Dune", "Unrated")
	wantNames(t, "LIST FROM #books SORT status, rating DESC", "Dune", "Thinking", "Kallocain", "Unrated")
	wantNames(t, "LIST FROM #books SORT file.name DESC LIMIT 2", "Unrated", "Thinking")
	wantNames(t, "LIST FROM #books LIMIT 0")
}

func TestClausesInAnyOrderAndCase(t *testing.T) {
	wantNames(t, "list from #books sort rating desc where rating >= 4", "Dune", "Kallocain")
	wantNames(t, "LIST FROM #books WHERE rating >= 3 WHERE status = \"read\"", "Dune", "Thinking")
}

func TestTableMarkdown(t *testing.T) {
	r := run(t, `TABLE author, rating AS "Stars" FROM #books WHERE rating >= 4 SORT rating DESC`)
	want := "| Note | author | Stars |\n" +
		"| --- | --- | --- |\n" +
		"| [[Books/Dune\\|Dune]] | Frank Herbert | 5 |\n" +
		"| [[Books/Kallocain\\|Kallocain]] | Karin Boye | 4 |\n"
	if r.Err != "" || r.Markdown != want {
		t.Errorf("got %q (err %q)\nwant %q", r.Markdown, r.Err, want)
	}
}

func TestTableEscapesPipesAndLeavesMissingValuesEmpty(t *testing.T) {
	r := run(t, `TABLE author, rating FROM "Books/Unrated"`)
	if !strings.Contains(r.Markdown, `| Anon \| Co |  |`) {
		t.Errorf("got %q", r.Markdown)
	}
}

func TestTableWithoutID(t *testing.T) {
	r := run(t, `TABLE WITHOUT ID author FROM #books/fiction`)
	want := "| author |\n| --- |\n| Frank Herbert |\n| Karin Boye |\n"
	if r.Markdown != want {
		t.Errorf("got %q", r.Markdown)
	}
}

func TestListWithAValue(t *testing.T) {
	r := run(t, `LIST author FROM #books/fiction`)
	want := "- [[Books/Dune|Dune]]: Frank Herbert\n- [[Books/Kallocain|Kallocain]]: Karin Boye\n"
	if r.Markdown != want {
		t.Errorf("got %q", r.Markdown)
	}
}

func TestFileFields(t *testing.T) {
	r := run(t, `TABLE file.folder, file.path, file.tags, file.mtime, file.size FROM "Books/Dune"`)
	for _, s := range []string{"| Books |", "Books/Dune.md", "#books, #books/fiction", "2026-09-01 00:00", "| 120 |"} {
		if !strings.Contains(r.Markdown, s) {
			t.Errorf("missing %q in %q", s, r.Markdown)
		}
	}
}

func TestEmptyResultSaysSo(t *testing.T) {
	r := run(t, `LIST FROM #nothing`)
	if r.Markdown != "" || r.Note != "No notes match" || r.Err != "" {
		t.Errorf("got %+v", r)
	}
}

func TestUnlimitedResultsAreCapped(t *testing.T) {
	var v fakeVault
	for i := 0; i < maxRows+32; i++ {
		v.notes = append(v.notes, Note{Rel: strings.Repeat("a", 1+i/26) + string(rune('a'+i%26)) + ".md"})
	}
	r := Run("LIST", v, "")
	if got := strings.Count(r.Markdown, "\n"); got != maxRows {
		t.Errorf("%d rows shown, want %d", got, maxRows)
	}
	if r.Note != "+32 more · add LIMIT or narrow FROM" {
		t.Errorf("note = %q", r.Note)
	}
	if r := Run("LIST LIMIT 500", v, ""); r.Note != "" || strings.Count(r.Markdown, "\n") != maxRows+32 {
		t.Errorf("an explicit LIMIT isn't capped: note %q", r.Note)
	}
}

func TestErrorsNameTheProblemAndTheLine(t *testing.T) {
	for q, want := range map[string]string{
		"":                                        "Spread: empty — start with TABLE, LIST or TASK (line 1)",
		"SHOW everything":                         `Spread: expected TABLE, LIST or TASK, found "SHOW" (line 1)`,
		"TABLE author\nFROM #books\nWHER x":       `Spread: expected FROM, WHERE, SORT or LIMIT, found "WHER" (line 3)`,
		"LIST FROM #books\nGROUP BY status":       "Spread: GROUP BY isn't supported yet (line 2)",
		"LIST\nFLATTEN authors":                   "Spread: FLATTEN isn't supported yet (line 2)",
		"CALENDAR file.mtime":                     "Spread: CALENDAR isn't supported yet (line 1)",
		"TASK text FROM #todo":                    "Spread: TASK takes no fields — put conditions in WHERE (line 1)",
		"TABLE file.ctime":                        "Spread: file.ctime isn't available: Linux can't tell when a note was created (line 1)",
		"TABLE file.day":                          "Spread: file.day isn't supported yet (line 1)",
		"LIST WHERE rating * 2 > 4":               "Spread: arithmetic (+ - * /) isn't supported yet (line 1)",
		"LIST WHERE startswith(file.name, \"a\")": "Spread: startswith() isn't supported yet (line 1)",
		"LIST WHERE contains(tags)":               "Spread: contains() takes two values: contains(field, value) (line 1)",
		"LIST FROM":                               `Spread: expected a #tag, a "folder" or a [[note]], found the end of the spread (line 1)`,
		`LIST FROM "Books`:                        `Spread: a string isn't closed with " (line 1)`,
		"LIST FROM [[Dune":                        "Spread: a [[link]] isn't closed with ]] (line 1)",
		"LIST LIMIT many":                         `Spread: LIMIT needs a whole number, found "many" (line 1)`,
		"LIST a, b":                               "Spread: LIST shows one value per note — use TABLE for more (line 1)",
		"TABLE WITHOUT ID":                        "Spread: TABLE WITHOUT ID needs at least one field (line 1)",
		"LIST FROM #a FROM #b":                    "Spread: FROM is given twice (line 1)",
		"LIST WHERE file.tags = #books":           `Spread: a tag here needs quotes: "#books" (line 1)`,
		"LIST WHERE (rating > 3":                  "Spread: expected ), found the end of the spread (line 1)",
	} {
		if got := Run(q, library, "").Err; got != want {
			t.Errorf("%q\n got  %s\n want %s", q, got, want)
		}
	}
}
