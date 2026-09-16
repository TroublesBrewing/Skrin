package ui

import "testing"

// The ? manual is generated from the registry, so a key that meant two
// things in one context would make the manual lie about it.
func TestNoKeyMeansTwoThingsInOneContext(t *testing.T) {
	seen := map[string]action{}
	for _, b := range defaultBindings {
		if b.act == actNone {
			continue
		}
		for _, k := range b.keys {
			id := b.where + " " + k
			if a, ok := seen[id]; ok && a != b.act {
				t.Errorf("%q in %s is bound to two actions", k, b.where)
			}
			seen[id] = b.act
		}
	}
}

// Every row needs its context, its manual section and something to say, or
// the manual can't list it where it works.
func TestEveryBindingIsDocumented(t *testing.T) {
	for _, b := range defaultBindings {
		if len(b.keys) == 0 || b.help == "" || b.group == "" || b.where == "" {
			t.Errorf("incomplete binding: %+v", b)
		}
	}
}

// The components that handle their own keys still look them up in the
// registry, so a key only has to be changed in one place.
func TestComponentsDispatchThroughTheRegistry(t *testing.T) {
	for _, c := range []struct{ where, key string }{
		{inSearch, "alt+r"}, {inSearch, "tab"}, {inSearch, "ctrl+s"}, {inSearch, "esc"},
		{inComplete, "enter"}, {inComplete, "esc"}, {inEditor, "ctrl+k"},
		{inList, "enter"}, {inDrawer, "alt+n"},
	} {
		if actionIn(c.where, c.key) == actNone {
			t.Errorf("%q in %s has no action: it can't be dispatched or overridden", c.key, c.where)
		}
	}
}
