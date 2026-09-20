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
	km := newKeymap(nil)
	for _, c := range []struct{ where, key string }{
		{inSearch, "alt+r"}, {inSearch, "tab"}, {inSearch, "ctrl+s"}, {inSearch, "esc"},
		{inComplete, "enter"}, {inComplete, "esc"}, {inEditor, "ctrl+k"},
		{inList, "enter"}, {inDrawer, "alt+n"},
	} {
		if km.act(c.where, c.key) == actNone {
			t.Errorf("%q in %s has no action: it can't be dispatched or overridden", c.key, c.where)
		}
	}
}

// Overrides are stored against the action's name, so every action the
// registry dispatches needs one, and no two may share it.
func TestEveryActionHasAName(t *testing.T) {
	seen := map[string]action{}
	for _, b := range defaultBindings {
		if b.act == actNone {
			continue
		}
		n, ok := actionName[b.act]
		if !ok || n == "" {
			t.Errorf("the action of %q in %s has no name: it can't be overridden or saved", b.help, b.where)
			continue
		}
		if a, ok := seen[n]; ok && a != b.act {
			t.Errorf("two actions are both called %q", n)
		}
		seen[n] = b.act
		for _, r := range n {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
				t.Errorf("action name %q can't be a bare key in config.toml", n)
				break
			}
		}
	}
}

// An override names a context and an action, so one action must not be
// listed twice in one context: there would be no saying which row an
// override belongs to.
func TestEachActionAppearsOnceInAContext(t *testing.T) {
	seen := map[string]string{}
	for _, b := range defaultBindings {
		if b.act == actNone {
			continue
		}
		id := b.where + " " + actionName[b.act]
		if first, ok := seen[id]; ok {
			t.Errorf("%s is in %s twice: %q and %q", actionName[b.act], b.where, first, b.help)
		}
		seen[id] = b.help
	}
}

func TestOverrideReplacesTheDefaultKey(t *testing.T) {
	km := newKeymap(map[string]map[string][]string{
		inMain: {"edit": {"ctrl+e"}},
	})
	if km.act(inMain, "ctrl+e") != actEdit {
		t.Error("the override's key should work")
	}
	if km.act(inMain, "e") != actNone {
		t.Error("the default key should stop working once it's been replaced")
	}
	if !km.changed(inMain, actEdit) {
		t.Error("the binding should count as changed")
	}
	if km.act(inMain, "n") != actNewNote {
		t.Error("everything else should keep its default key")
	}
}

// An override for an action this Skrin no longer has must not stop the
// rest of the keymap loading: an old config outliving a rename is exactly
// when the keys matter most.
func TestUnknownOverridesAreIgnored(t *testing.T) {
	km := newKeymap(map[string]map[string][]string{
		inMain: {"edit": {"ctrl+e"}, "teleport": {"ctrl+t"}},
	})
	if km.act(inMain, "ctrl+e") != actEdit {
		t.Error("the good override should still apply")
	}
	if km.act(inMain, "r") != actRename {
		t.Error("an action with no override should keep its default")
	}
}

// A binding whose key was given away is saved with no keys at all. That
// has to survive a restart: handing the default back would leave two
// bindings claiming the same key.
func TestABindingLeftWithNoKeyStaysThatWay(t *testing.T) {
	km := newKeymap(map[string]map[string][]string{
		inMain: {"zen": {"e"}, "edit": {}},
	})
	if km.act(inMain, "e") != actZen {
		t.Error("the key that was given away should belong to its new action")
	}
	if len(km.bound(inMain, actEdit)) != 0 {
		t.Errorf("the binding it came from should still have no key, got %v", km.bound(inMain, actEdit))
	}
	// The invariant the whole registry rests on, across a restart.
	seen := map[string]action{}
	for _, b := range defaultBindings {
		if b.act == actNone || b.where != inMain {
			continue
		}
		for _, k := range km.bound(inMain, b.act) {
			if a, ok := seen[k]; ok && a != b.act {
				t.Errorf("%q means two things after reloading the keymap", k)
			}
			seen[k] = b.act
		}
	}
}

func TestSetTakesTheKeyFromWhoeverHadIt(t *testing.T) {
	km := newKeymap(nil)
	displaced := km.set(inMain, actNewNote, "e")
	if displaced != actEdit {
		t.Fatalf("displaced = %q, want the action that had e", actionName[displaced])
	}
	if km.act(inMain, "e") != actNewNote {
		t.Error("e should now be the new action's")
	}
	if len(km.bound(inMain, actEdit)) != 0 {
		t.Errorf("the displaced action should be left with no keys, got %v", km.bound(inMain, actEdit))
	}
	km.reset(inMain, actNewNote)
	km.reset(inMain, actEdit)
	if km.act(inMain, "e") != actEdit || km.act(inMain, "n") != actNewNote {
		t.Error("resetting both should put the defaults back")
	}
	if km.overrides() != nil {
		t.Errorf("with nothing changed there should be no [keys] to save, got %v", km.overrides())
	}
}

// A key a component handles itself can't be given away, so the Keys tab
// has to tell that apart from a free key and from one it can offer to
// take.
func TestHolderTellsRebindableKeysFromTheComponentsOwn(t *testing.T) {
	km := newKeymap(nil)
	if help, rebindable := km.holder(inMain, "e"); help == "" || !rebindable {
		t.Errorf("e in main: help %q, rebindable %v; want a rebindable holder", help, rebindable)
	}
	if help, rebindable := km.holder(inEditor, "ctrl+s"); help == "" || rebindable {
		t.Errorf("ctrl+s in the editor: help %q, rebindable %v; want the editor's own", help, rebindable)
	}
	if help, _ := km.holder(inMain, "ctrl+alt+shift+f9"); help != "" {
		t.Errorf("an unused key should be free, got %q", help)
	}
}

func TestResetAllDropsEveryOverride(t *testing.T) {
	km := newKeymap(map[string]map[string][]string{
		inMain:   {"edit": {"ctrl+e"}},
		inHabits: {"habit-tab": {"w"}},
	})
	km.resetAll()
	if km.act(inMain, "e") != actEdit || km.act(inHabits, "H") != actHabitTab {
		t.Error("every default should be back")
	}
	if km.overrides() != nil {
		t.Error("nothing should be left to save")
	}
}

// The keymap constitution, rule 4: one key, one meaning per context. An
// override names a context and a key, so two rows claiming the same key in
// one context would make the manual — and the override — ambiguous.
//
// The one exception is vim mode: its rows sit in the editor's context but
// only apply when editor.vim is on, so a key may mean one thing there and
// another in the ordinary editor.
func TestOneKeyMeansOneThingInAContext(t *testing.T) {
	type row struct{ group, help string }
	seen := map[string]row{}
	for _, b := range defaultBindings {
		for _, k := range b.keys {
			id := b.where + " " + k
			first, ok := seen[id]
			switch {
			case !ok:
				seen[id] = row{b.group, b.help}
			case (first.group == groupVim) != (b.group == groupVim):
				// vim mode and the ordinary editor: different modes.
			default:
				t.Errorf("%q means two things in %s: %q and %q", k, b.where, first.help, b.help)
			}
		}
	}
}
