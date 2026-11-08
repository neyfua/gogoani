package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/neyfua/gogoani/internal/anilist"
	"gopkg.in/yaml.v3"
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
	token := os.Getenv("GOGOANI_ANILIST_TOKEN")
	if token == "" {
		if cached, err := anilist.LoadToken(); err == nil {
			token = cached
		}
	}
	return &Config{
		Player:   loadPlayer(),
		Detach:   detach(),
		AutoSync: token != "",
		AniList: AniListConfig{
			Token: token,
		},
	}
}

// loadPlayer reads the optional, user-authored config without creating it.
func loadPlayer() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gogoani: locating config: %v\n", err)
		return "mpv"
	}
	path := filepath.Join(dir, "gogoani", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "gogoani: ignoring %s: %v\n", path, err)
		}
		return "mpv"
	}
	var cfg struct {
		Player string `yaml:"player"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "gogoani: ignoring %s: %v\n", path, err)
		return "mpv"
	}
	if p := strings.TrimSpace(cfg.Player); p != "" {
		return p
	}
	return "mpv"
}

// detach defaults to true (detach player from the terminal) unless
// the user opts out via GOGOANI_NO_DETACH=1.
func detach() bool {
	return os.Getenv("GOGOANI_NO_DETACH") == ""
}
