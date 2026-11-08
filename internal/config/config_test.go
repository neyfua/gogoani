package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPlayer(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("GOGOANI_PLAYER", "vlc") // Legacy environment variable is ignored.
	path := filepath.Join(dir, "gogoani", "config.yaml")

	if got := Load().Player; got != "mpv" {
		t.Fatalf("default player = %q, want mpv", got)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("Load created config directory or stat failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, yaml, want string
	}{
		{"vlc", "player: vlc # or mpv\n", "vlc"},
		{"mpv", "player: mpv\n", "mpv"},
		{"quoted", "player: 'vlc'\n", "vlc"},
		{"empty file", "", "mpv"},
		{"empty player", "player: # vlc or mpv\n", "mpv"},
		{"blank player", "player: '  '\n", "mpv"},
		{"malformed", "player: [\n", "mpv"},
		{"wrong type", "player: [vlc]\n", "mpv"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(tt.yaml), 0o600); err != nil {
				t.Fatal(err)
			}
			if got := Load().Player; got != tt.want {
				t.Fatalf("configured player = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadAniListToken(t *testing.T) {
	t.Setenv("GOGOANI_ANILIST_TOKEN", "test-token-123")
	cfg := Load()
	if !cfg.AutoSync {
		t.Fatal("AutoSync = false, want true when token is set")
	}
	if cfg.AniList.Token != "test-token-123" {
		t.Fatalf("AniList.Token = %q, want test-token-123", cfg.AniList.Token)
	}
}
