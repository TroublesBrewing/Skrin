package index

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lurioso/skrin/internal/vault"
)

func build(t *testing.T, files map[string]string) (*Index, *vault.Vault) {
	t.Helper()
	root := t.TempDir()
	for f, body := range files {
		p := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	x := New()
	if err := x.Update(v); err != nil {
		t.Fatal(err)
	}
	return x, v
}

func TestParse(t *testing.T) {
	n := parse("---\naliases: [Stoa, Porch]\nrelated: \"[[Seneca]]\"\n---\n# Stoic ideas ##\nSee [[Epictetus#Discourses|the Discourses]] and ![[diagram.png]].\n" +
		"`[[not a link]]` but [md](Other%20note.md#Top) and [web](https://x.com)\n```\n[[in code]]\n```\n| a | [[Zeno\\|founder]] |\nA key line ^key-1\n")
	if !reflect.DeepEqual(n.aliases, []string{"Stoa", "Porch"}) {
		t.Errorf("aliases = %q", n.aliases)
	}
	if len(n.headings) != 1 || n.headings[0] != (Heading{1, "Stoic ideas", 4}) {
		t.Errorf("headings = %+v", n.headings)
	}
	var got []string
	for _, l := range n.links {
		got = append(got, l.Target+"#"+l.Sub+"|"+l.Alias)
	}
	want := []string{"Seneca#|", "Epictetus#Discourses|the Discourses", "diagram.png#|", "Other note.md#Top|md", "Zeno#|founder"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("links =\n%q\nwant\n%q", got, want)
	}
	if !n.links[2].Embed || n.links[4].Sep != `\|` || !n.links[3].Markdown {
		t.Errorf("link flags wrong: %+v", n.links)
	}
	if n.blocks["key-1"] != 11 {
		t.Errorf("blocks = %v", n.blocks)
	}
}

func TestTagsAndProperties(t *testing.T) {
	n := parse("---\ntags: [Stoa, \"#filosofi/antik\"]\nstatus: draft\ncreated: 2026-09-15\nempty:\naliases: Porch, Colonnade\n---\n" +
		"Body #Idea and #nested/tag, `#notcode` and x.com/#frag\n```\n#incode\n```\n")
	if want := []string{"filosofi/antik", "idea", "nested/tag", "stoa"}; !reflect.DeepEqual(n.tags, want) {
		t.Errorf("tags = %q, want %q", n.tags, want)
	}
	if n.props["status"][0] != "draft" || n.props["created"][0] != "2026-09-15" {
		t.Errorf("props = %v", n.props)
	}
	if _, ok := n.props["empty"]; !ok || len(n.props["empty"]) != 0 {
		t.Errorf("an empty property should exist with no values: %v", n.props)
	}
	if !reflect.DeepEqual(n.aliases, []string{"Porch", "Colonnade"}) {
		t.Errorf("aliases = %q", n.aliases)
	}
}

func TestResolveLikeObsidian(t *testing.T) {
	x, _ := build(t, map[string]string{
		"Welcome.md":             "",
		"Filosofi/Stoic.md":      "",
		"Filosofi/Note.md":       "",
		"Daily/Note.md":          "",
		"Deep/Down/Note.md":      "",
		"img/diagram.png":        "",
		"Filosofi/Antik/Zeno.md": "",
	})
	for _, c := range []struct{ target, from, want string }{
		{"Stoic", "Welcome.md", "Filosofi/Stoic.md"},
		{"stoic", "Welcome.md", "Filosofi/Stoic.md"},
		{"Filosofi/Stoic", "Welcome.md", "Filosofi/Stoic.md"},
		{"Filosofi/Stoic.md", "Welcome.md", "Filosofi/Stoic.md"},
		{"Note", "Daily/x.md", "Daily/Note.md"},          // same folder wins
		{"Note", "Welcome.md", "Daily/Note.md"},          // then the shortest path, then a–z
		{"Down/Note", "Welcome.md", "Deep/Down/Note.md"}, // partial path
		{"Antik/Zeno", "Filosofi/Stoic.md", "Filosofi/Antik/Zeno.md"},
		{"diagram.png", "Welcome.md", "img/diagram.png"},
		{"", "Welcome.md", "Welcome.md"}, // [[#Heading]] in the same note
	} {
		got, ok := x.Resolve(c.target, c.from)
		if !ok || got != c.want {
			t.Errorf("Resolve(%q from %q) = %q %v, want %q", c.target, c.from, got, ok, c.want)
		}
	}
	if _, ok := x.Resolve("Missing", "Welcome.md"); ok {
		t.Error("missing note resolved")
	}
}

func TestBacklinksAndAnchors(t *testing.T) {
	x, _ := build(t, map[string]string{
		"Welcome.md":        "See [[Stoic]] and [[Stoic#Morning]].\n",
		"Daily/2026.md":     "- [ ] read [md link](../Filosofi/Stoic.md)\n",
		"Filosofi/Stoic.md": "# Stoic\n## Morning\nText ^quote\nSelf [[Stoic]]\n",
	})
	bl := x.Backlinks("Filosofi/Stoic.md")
	if len(bl) != 3 {
		t.Fatalf("backlinks = %+v", bl)
	}
	if bl[0].Source != "Daily/2026.md" || bl[0].Link.Context != "- [ ] read [md link](../Filosofi/Stoic.md)" {
		t.Errorf("first backlink = %+v", bl[0])
	}
	if l, ok := x.Anchor("Filosofi/Stoic.md", "morning"); !ok || l != 1 {
		t.Errorf("heading anchor = %d %v", l, ok)
	}
	if l, ok := x.Anchor("Filosofi/Stoic.md", "^quote"); !ok || l != 2 {
		t.Errorf("block anchor = %d %v", l, ok)
	}
	if refs := x.Referrers("Filosofi", true); len(refs) != 4 {
		t.Errorf("folder referrers = %d, want 4 (self link included)", len(refs))
	}
}

func TestUpdateRereadsOnlyChangedNotes(t *testing.T) {
	x, v := build(t, map[string]string{"a.md": "[[b]]", "b.md": ""})
	if len(x.Backlinks("b.md")) != 1 {
		t.Fatal("setup")
	}
	if err := os.WriteFile(v.Abs("a.md"), []byte("no links any more"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := x.Update(v); err != nil {
		t.Fatal(err)
	}
	if len(x.Backlinks("b.md")) != 0 {
		t.Error("changed note not re-read")
	}
}

func TestImagesListsNonNoteFilesButNotNotes(t *testing.T) {
	x, _ := build(t, map[string]string{
		"Note.md":         "",
		"Assets/pic.png":  "not a real png, just bytes",
		"Assets/pic2.JPG": "",
		"Assets/data.txt": "",
	})
	got := x.Images()
	want := []string{"Assets/pic.png", "Assets/pic2.JPG"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Images() = %q, want %q", got, want)
	}
}

func TestLinkTextAndApply(t *testing.T) {
	x, _ := build(t, map[string]string{"A/Note.md": "", "B/Note.md": "", "A/Unique.md": ""})
	if got := x.LinkText("A/Unique.md", "B/x.md", "shortest"); got != "Unique" {
		t.Errorf("shortest unique = %q", got)
	}
	if got := x.LinkText("A/Note.md", "B/x.md", "shortest"); got != "A/Note" {
		t.Errorf("shortest ambiguous = %q", got)
	}
	if got := x.LinkText("A/Note.md", "B/Sub/x.md", "relative"); got != "../../A/Note" {
		t.Errorf("relative = %q", got)
	}

	content := "x [[Old#Top|alias]] y ![[Old]] z [t](Old.md) | [[Old\\|a]] |"
	links := parse(content).links
	var edits []Edit
	for _, l := range links {
		target := "New Name"
		if l.Markdown {
			target = "Sub/New Name.md"
		}
		edits = append(edits, Edit{Link: l, Target: target})
	}
	want := "x [[New Name#Top|alias]] y ![[New Name]] z [t](Sub/New%20Name.md) | [[New Name\\|a]] |"
	if got := Apply(content, edits); got != want {
		t.Errorf("Apply =\n%q\nwant\n%q", got, want)
	}
}

func TestOutgoingListsLinkedNotesOnce(t *testing.T) {
	x, _ := build(t, map[string]string{
		"Log.md":        "[[Dune]] and [[Dune|again]], [[Kallocain]], [[Missing]], ![[cover.png]]",
		"Books/Dune.md": "", "Kallocain.md": "", "cover.png": "",
	})
	if got := x.Outgoing("Log.md"); !reflect.DeepEqual(got, []string{"Books/Dune.md", "Kallocain.md"}) {
		t.Errorf("Outgoing = %q", got)
	}
	if got := x.Outgoing("Nope.md"); got != nil {
		t.Errorf("Outgoing of a missing note = %q", got)
	}
}

func TestStatIsWhatTheIndexRead(t *testing.T) {
	x, v := build(t, map[string]string{"Note.md": "hello"})
	mod, size, ok := x.Stat("Note.md")
	info, err := os.Stat(v.Abs("Note.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok || size != 5 || !mod.Equal(info.ModTime()) {
		t.Errorf("Stat = %v %d %v, want %v 5", mod, size, ok, info.ModTime())
	}
	if _, _, ok := x.Stat("Nope.md"); ok {
		t.Error("Stat of a missing note should report false")
	}
}

func TestInlineFields(t *testing.T) {
	x, _ := build(t, map[string]string{"Book.md": "---\nrating: 4\n---\n" +
		"Rating:: 5\n" +
		"**Due Date**:: 2026-09-30\n" +
		"Read it for [mood:: calm] and (pace:: slow) reasons.\n" +
		"- genre:: sci-fi\n" +
		"> quote:: kept\n" +
		"`code:: skipped` and a url https://x.com::not\n" +
		"```\nfenced:: skipped\n```\n" +
		"Author:: [[Frank Herbert]]\n" +
		"[see:: [[Dune|the book]]]\n" +
		"empty::\n"})
	want := map[string][]string{
		"rating": {"5"}, "due-date": {"2026-09-30"}, "mood": {"calm"}, "pace": {"slow"},
		"genre": {"sci-fi"}, "quote": {"kept"}, "author": {"[[Frank Herbert]]"}, "see": {"[[Dune|the book]]"},
	}
	if got := x.Fields("Book.md"); !reflect.DeepEqual(got, want) {
		t.Errorf("Fields =\n %v\nwant\n %v", got, want)
	}
	if d, _ := x.Doc("Book.md"); !reflect.DeepEqual(d.Props["rating"], []string{"4"}) {
		t.Errorf("inline fields must not leak into properties, which search reads: %v", d.Props)
	}
}

func TestTasks(t *testing.T) {
	x, _ := build(t, map[string]string{"Plan.md": "# Plan\n" +
		"- [ ] call the printer [due:: 2026-09-22] [priority:: high]\n" + // 1
		"    - [x] find the number\n" + // 2
		"    - a plain note\n" + // 3
		"        - [ ] ask about paper\n" + // 4
		"\n" +
		"* [-] cancelled\n" + // 6
		"1. [/] half done\n" + // 7
		"Some prose ends the list.\n" +
		"  - [ ] after prose\n" + // 9
		"```\n- [ ] in code\n```\n" +
		"- [ ]\n" + // 13
		"- [] not a task\n"})
	var got []string
	for _, tk := range x.Tasks("Plan.md") {
		got = append(got, fmt.Sprintf("%d %q %q p%d %v", tk.Line, tk.Status, tk.Text, tk.Parent, tk.Fields))
	}
	want := []string{
		`1 " " "call the printer [due:: 2026-09-22] [priority:: high]" p-1 map[due:[2026-09-22] priority:[high]]`,
		`2 "x" "find the number" p0 map[]`,
		`4 " " "ask about paper" p0 map[]`,
		`6 "-" "cancelled" p-1 map[]`,
		`7 "/" "half done" p-1 map[]`,
		`9 " " "after prose" p-1 map[]`,
		`13 " " "" p-1 map[]`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("tasks =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if f := x.Fields("Plan.md"); f["due"] != nil {
		t.Errorf("a task's fields are the task's, not its note's: %v", f)
	}
}

func TestLineAnchor(t *testing.T) {
	x, _ := build(t, map[string]string{"N.md": "one\ntwo\nthree"})
	for sub, want := range map[string]int{":1": 0, ":3": 2} {
		if got, ok := x.Anchor("N.md", sub); !ok || got != want {
			t.Errorf("Anchor(%q) = %d %v, want %d", sub, got, ok, want)
		}
	}
	for _, sub := range []string{":0", ":4", ":x", ":"} {
		if _, ok := x.Anchor("N.md", sub); ok {
			t.Errorf("Anchor(%q) should fail", sub)
		}
	}
}

func TestTagsKeepEachSpellingAndCountNotes(t *testing.T) {
	x, _ := build(t, map[string]string{
		"a.md": "---\ntags: [Filosofi]\n---\nText #Filosofi #stoa",
		"b.md": "Mer #filosofi och #Filosofi",
		"c.md": "#Filosofi",
	})
	got := map[string]int{}
	for _, u := range x.Tags() {
		got[u.Text] = u.Notes
	}
	want := map[string]int{"Filosofi": 3, "filosofi": 1, "stoa": 1}
	if len(got) != len(want) {
		t.Fatalf("tags %v, want %v", got, want)
	}
	for k, n := range want {
		if got[k] != n {
			t.Errorf("%s in %d notes, want %d (a note counts once)", k, got[k], n)
		}
	}
	if first := x.Tags()[0]; first.Text != "Filosofi" {
		t.Errorf("most used first: %+v", first)
	}
	if doc, _ := x.Doc("b.md"); len(doc.Tags) != 1 || doc.Tags[0] != "filosofi" {
		t.Errorf("search still sees one lower-case tag per note: %v", doc.Tags)
	}
}

func TestPropertyValuesCountEachNoteOnce(t *testing.T) {
	x, _ := build(t, map[string]string{
		"a.md": "---\ntype: village\nland: \"[[Aldalor]]\"\nfolk: [elf, elf, human]\n---\n",
		"b.md": "---\nType: city\nland: \"[[Aldalor]]\"\n---\n",
		"c.md": "---\ntype: village\n---\n",
	})
	check := func(key string, want map[string]int) {
		t.Helper()
		got := map[string]int{}
		for _, u := range x.PropertyValues(key) {
			got[u.Text] = u.Notes
		}
		if len(got) != len(want) {
			t.Fatalf("%s: %v, want %v", key, got, want)
		}
		for k, n := range want {
			if got[k] != n {
				t.Errorf("%s: %s in %d notes, want %d", key, k, got[k], n)
			}
		}
	}
	check("type", map[string]int{"village": 2, "city": 1})
	check("land", map[string]int{"[[Aldalor]]": 2})
	check("folk", map[string]int{"elf": 1, "human": 1})
}
