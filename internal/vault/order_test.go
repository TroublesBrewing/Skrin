package vault

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLoadOrderAndItsFallbacks(t *testing.T) {
	v := makeVault(t, "Welcome.md")
	if o := v.LoadOrder(); len(o) != 0 {
		t.Errorf("no .skrin should mean no order: %v", o)
	}
	if err := os.WriteFile(v.Abs(OrderFile), []byte(`{"Filosofi/": ["b.md", "a"], "/": ["z.md"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	want := Order{"Filosofi": {"b.md", "a"}, "": {"z.md"}}
	if o := v.LoadOrder(); !reflect.DeepEqual(o, want) {
		t.Errorf("LoadOrder = %v, want %v", o, want)
	}
	for _, bad := range []string{"{not json", `{"Filosofi/": "a"}`, `["a"]`} {
		if err := os.WriteFile(v.Abs(OrderFile), []byte(bad), 0o644); err != nil {
			t.Fatal(err)
		}
		if o := v.LoadOrder(); len(o) != 0 {
			t.Errorf("a broken .skrin (%s) should mean no order: %v", bad, o)
		}
	}
}

func TestOrderMarshalRoundTrip(t *testing.T) {
	o := Order{"Filosofi": {"b.md", "a"}, "": {"z.md"}, "a/b": {"c"}}
	text := o.Marshal()
	for _, key := range []string{`"/":`, `"Filosofi/":`, `"a/b/":`} {
		if !strings.Contains(text, key) {
			t.Errorf("%s lacks the key %s", text, key)
		}
	}
	v := makeVault(t)
	if err := v.Write(OrderFile, text); err != nil {
		t.Fatal(err)
	}
	if got := v.LoadOrder(); !reflect.DeepEqual(got, o) {
		t.Errorf("round trip = %v, want %v", got, o)
	}
}

func TestArrangePutsListedFirstAndNewLast(t *testing.T) {
	entries := []Entry{{Name: "a", IsDir: true}, {Name: "b.md"}, {Name: "c.md"}, {Name: "d.md"}}
	Order{"x": {"c.md", "gone.md", "a"}}.Arrange("x", entries)
	var got []string
	for _, e := range entries {
		got = append(got, e.Name)
	}
	if strings.Join(got, ",") != "c.md,a,b.md,d.md" {
		t.Errorf("arranged %v: want the listed ones first (a file above a folder too), then the rest", got)
	}
	Order{}.Arrange("x", entries)
	if entries[0].Name != "c.md" {
		t.Error("a level without an order should be left as it is")
	}
}

func TestOrderFollowsRenamesAndMoves(t *testing.T) {
	o := Order{"": {"b.md", "a"}, "a": {"y.md", "x.md"}, "a/deep": {"z.md"}}
	if !o.Follow([][2]string{{"a", "A"}}) {
		t.Fatal("renaming an ordered folder should change the order")
	}
	want := Order{"": {"b.md", "A"}, "A": {"y.md", "x.md"}, "A/deep": {"z.md"}}
	if !reflect.DeepEqual(o, want) {
		t.Errorf("after the rename: %v, want %v", o, want)
	}
	if !o.Follow([][2]string{{"A/y.md", "b/y.md"}}) || !reflect.DeepEqual(o["A"], []string{"x.md"}) {
		t.Errorf("a note leaving a level should leave its list: %v", o["A"])
	}
	if o.Follow([][2]string{{"other.md", "else.md"}}) {
		t.Error("a move outside every ordered level shouldn't change anything")
	}
}
