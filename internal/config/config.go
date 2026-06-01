package config

import (
	"cmp"
	"os"

	"github.com/neyfua/gogoani/internal/anilist"
)

type Config struct {
	Player   string // media player binary
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
		AutoSync: token != "",
		AniList: AniListConfig{
			Token: token,
		},
	}
}
