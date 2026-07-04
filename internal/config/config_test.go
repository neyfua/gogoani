package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original env vars
	origPlayer := os.Getenv("GOGOANI_PLAYER")
	origToken := os.Getenv("GOGOANI_ANILIST_TOKEN")
	defer func() {
		os.Setenv("GOGOANI_PLAYER", origPlayer)
		os.Setenv("GOGOANI_ANILIST_TOKEN", origToken)
	}()

	t.Run("default player when no env", func(t *testing.T) {
		os.Unsetenv("GOGOANI_PLAYER")

		cfg := Load()
		if cfg.Player != "mpv" {
			t.Errorf("Load() Player = %q, want %q", cfg.Player, "mpv")
		}
	})

	t.Run("custom player from env", func(t *testing.T) {
		os.Setenv("GOGOANI_PLAYER", "vlc")

		cfg := Load()
		if cfg.Player != "vlc" {
			t.Errorf("Load() Player = %q, want %q", cfg.Player, "vlc")
		}
	})

	t.Run("token enables autosync", func(t *testing.T) {
		os.Unsetenv("GOGOANI_PLAYER")
		os.Setenv("GOGOANI_ANILIST_TOKEN", "test-token-123")

		cfg := Load()
		if !cfg.AutoSync {
			t.Errorf("Load() AutoSync = false, want true when token is set")
		}
		if cfg.AniList.Token != "test-token-123" {
			t.Errorf("Load() AniList.Token = %q, want %q", cfg.AniList.Token, "test-token-123")
		}
	})
}
