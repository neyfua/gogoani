package anilist

import (
	"testing"
)

func TestMediaListStatusLabel(t *testing.T) {
	tests := []struct {
		status   MediaListStatus
		expected string
	}{
		{StatusWatching, "watching"},
		{StatusCompleted, "completed"},
		{StatusPaused, "paused"},
		{StatusDropped, "dropped"},
		{StatusPlanning, "planning"},
		{StatusRepeating, "repeating"},
		{MediaListStatus("UNKNOWN"), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := tt.status.Label()
			if result != tt.expected {
				t.Errorf("Label() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		label    string
		expected MediaListStatus
		ok       bool
	}{
		{"watching", StatusWatching, true},
		{"current", StatusWatching, true},
		{"completed", StatusCompleted, true},
		{"complete", StatusCompleted, true},
		{"paused", StatusPaused, true},
		{"dropped", StatusDropped, true},
		{"planning", StatusPlanning, true},
		{"plan to watch", StatusPlanning, true},
		{"repeating", StatusRepeating, true},
		{"invalid", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			result, ok := ParseStatus(tt.label)
			if ok != tt.ok {
				t.Errorf("ParseStatus(%q) ok = %v, want %v", tt.label, ok, tt.ok)
			}
			if result != tt.expected {
				t.Errorf("ParseStatus(%q) = %v, want %v", tt.label, result, tt.expected)
			}
		})
	}
}

func TestAllStatuses(t *testing.T) {
	statuses := AllStatuses()
	if len(statuses) != 6 {
		t.Errorf("AllStatuses() returned %d statuses, want 6", len(statuses))
	}

	expected := map[MediaListStatus]bool{
		StatusWatching:  true,
		StatusCompleted: true,
		StatusPaused:    true,
		StatusDropped:   true,
		StatusPlanning:  true,
		StatusRepeating: true,
	}

	for _, s := range statuses {
		if !expected[s] {
			t.Errorf("AllStatuses() contains unexpected status: %v", s)
		}
	}
}

func TestAnimeEntryTitles(t *testing.T) {
	tests := []struct {
		name     string
		entry    AnimeEntry
		expected []string
	}{
		{
			name: "all titles unique",
			entry: AnimeEntry{
				Title:       "English Title",
				TitleRomaji: "Romaji Title",
				TitleNative: "Native Title",
			},
			expected: []string{"English Title", "Romaji Title", "Native Title"},
		},
		{
			name: "some titles empty",
			entry: AnimeEntry{
				Title:       "English Title",
				TitleRomaji: "",
				TitleNative: "Native Title",
			},
			expected: []string{"English Title", "Native Title"},
		},
		{
			name: "duplicate titles",
			entry: AnimeEntry{
				Title:       "Same Title",
				TitleRomaji: "Same Title",
				TitleNative: "Native Title",
			},
			expected: []string{"Same Title", "Native Title"},
		},
		{
			name: "all empty",
			entry: AnimeEntry{
				Title:       "",
				TitleRomaji: "",
				TitleNative: "",
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.Titles()
			if len(result) != len(tt.expected) {
				t.Errorf("Titles() returned %d titles, want %d", len(result), len(tt.expected))
			}
			for i, title := range result {
				if i >= len(tt.expected) || title != tt.expected[i] {
					t.Errorf("Titles()[%d] = %q, want %q", i, title, tt.expected[i])
				}
			}
		})
	}
}

func TestAnimeEntryTitleDisplay(t *testing.T) {
	tests := []struct {
		name     string
		entry    AnimeEntry
		expected string
	}{
		{
			name:     "with title",
			entry:    AnimeEntry{Title: "My Anime"},
			expected: "My Anime",
		},
		{
			name:     "empty title",
			entry:    AnimeEntry{Title: ""},
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.TitleDisplay()
			if result != tt.expected {
				t.Errorf("TitleDisplay() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestAnimeEntryTotalEpisodesDisplay(t *testing.T) {
	tests := []struct {
		name     string
		entry    AnimeEntry
		expected string
	}{
		{
			name:     "with episodes",
			entry:    AnimeEntry{TotalEps: 12},
			expected: "12",
		},
		{
			name:     "zero episodes",
			entry:    AnimeEntry{TotalEps: 0},
			expected: "?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.TotalEpisodesDisplay()
			if result != tt.expected {
				t.Errorf("TotalEpisodesDisplay() = %q, want %q", result, tt.expected)
			}
		})
	}
}
