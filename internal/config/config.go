package config

import (
	"cmp"
	"os"
)

type Config struct {
	Player string // media player binary
}

func Load() *Config {
	player := cmp.Or(os.Getenv("GOGOANI_PLAYER"), "mpv")
	return &Config{
		Player: player,
	}
}
