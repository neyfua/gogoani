package config

import (
	"cmp"
	"os"

	"github.com/neyfua/gogoani/internal/anilist"
)

type Config struct {
	Player   string // media player binary
	Detach   bool   // detach player from the terminal (safe inside tmux)
	AniList  AniListConfig
	AutoSync bool // Enable AniList auto-sync after watching
}

type AniListConfig struct {
	Token string
}

func Load() *Config {
	player := cmp.Or(os.Getenv("GOGOANI_PLAYER"), "mpv")
	token := os.Getenv("GOGOANI_ANILIST_TOKEN")
	if token == "" {
		if cached, err := anilist.LoadToken(); err == nil {
			token = cached
		}
	}
	return &Config{
		Player:   player,
		Detach:   detach(),
		AutoSync: token != "",
		AniList: AniListConfig{
			Token: token,
		},
	}
}

// detach defaults to true (detach player from the terminal) unless
// the user opts out via GOGOANI_NO_DETACH=1.
func detach() bool {
	return os.Getenv("GOGOANI_NO_DETACH") == ""
}
