package anilist

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListAndTokenDirectories(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	entries := []AnimeEntry{{Title: "Example"}}
	if err := SaveList(entries); err != nil {
		t.Fatal(err)
	}
	path, err := ListPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".local", "state", "gogoani", "anilist_list.json")
	if path != want {
		t.Fatalf("ListPath() = %q, want %q", path, want)
	}
	loaded, err := LoadList()
	if err != nil || len(loaded) != 1 || loaded[0].Title != "Example" {
		t.Fatalf("LoadList() = %v, %v", loaded, err)
	}

	if err := SaveToken("secret"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".cache", "gogoani", "anilist_token.json")); err != nil {
		t.Fatal(err)
	}
}
