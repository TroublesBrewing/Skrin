package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverPrefersOpenVault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "obsidian.json")
	data := `{"vaults":{
		"a":{"path":"/vaults/recent","ts":300},
		"b":{"path":"/vaults/open","ts":100,"open":true},
		"c":{"path":"/vaults/old","ts":50}}}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := discoverFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/vaults/open" {
		t.Errorf("got %q, want the open vault", got)
	}
}

func TestDiscoverFallsBackToMostRecent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "obsidian.json")
	data := `{"vaults":{"a":{"path":"/vaults/old","ts":1},"b":{"path":"/vaults/new","ts":2}}}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := discoverFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/vaults/new" {
		t.Errorf("got %q, want the most recent vault", got)
	}
}
