package index

import (
	"os"
	"path/filepath"
	"reflect"
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
