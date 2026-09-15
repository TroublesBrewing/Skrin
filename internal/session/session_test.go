package session

import (
	"reflect"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if got := Load("/vaults/a"); !reflect.DeepEqual(got, State{}) {
		t.Fatalf("nothing saved yet, got %+v", got)
	}
	want := State{Expanded: []string{"Filosofi", "Filosofi/Antik"}, Cursor: "Filosofi/Stoic.md", Open: "Welcome.md", Offset: 12}
	if err := Save("/vaults/a", want); err != nil {
		t.Fatal(err)
	}
	if got := Load("/vaults/a"); !reflect.DeepEqual(got, want) {
		t.Errorf("Load = %+v, want %+v", got, want)
	}
	if got := Load("/vaults/b"); !reflect.DeepEqual(got, State{}) {
		t.Errorf("another vault shares the state: %+v", got)
	}
}
