package scraper

import (
	"testing"
)

func TestAnimeString(t *testing.T) {
	anime := Anime{
		ID:    "123",
		Title: "Attack on Titan",
		URL:   "https://example.com/aot",
	}

	result := anime.String()
	if result != "Attack on Titan" {
		t.Errorf("Anime.String() = %q, want %q", result, "Attack on Titan")
	}
}

func TestEpisodeString(t *testing.T) {
	tests := []struct {
		name     string
		episode  Episode
		expected string
	}{
		{
			name: "with title",
			episode: Episode{
				Number: 1,
				Title:  "The Beginning",
				Mode:   "sub",
			},
			expected: "Episode 1 \u2013 The Beginning",
		},
		{
			name: "without title",
			episode: Episode{
				Number: 5,
				Title:  "",
				Mode:   "dub",
			},
			expected: "Episode 5",
		},
		{
			name: "zero episode",
			episode: Episode{
				Number: 0,
				Title:  "",
			},
			expected: "Episode 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.episode.String()
			if result != tt.expected {
				t.Errorf("Episode.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}
