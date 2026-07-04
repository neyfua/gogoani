package anilist

import (
	"testing"
)

func TestClean(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple Title", "simple title"},
		{"Title: Subtitle", "title subtitle"},
		{"Title (2023)", "title 2023"},
		{"Title - Season 2", "title season 2"},
		{"Title's Name", "title's name"},
		{"Multiple   Spaces", "multiple spaces"},
		{"Title/Subtitle", "title subtitle"},
		{"Title! Name?", "title name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := clean(tt.input)
			if result != tt.expected {
				t.Errorf("clean(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractNumbers(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
	}{
		{"season 2", []int{2}},
		{"season second", []int{2}},
		{"3rd episode", []int{3}},
		{"10th anniversary", []int{10}},
		{"no numbers here", []int{}},
		{"1 2 3", []int{1, 2, 3}},
		{"first second third", []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractNumbers(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("extractNumbers(%q) returned %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("extractNumbers(%q)[%d] = %d, want %d", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestNumericBonus(t *testing.T) {
	tests := []struct {
		name     string
		aNums    []int
		bNums    []int
		expected int
	}{
		{"matching numbers", []int{2}, []int{2}, 15},
		{"no match", []int{1}, []int{2}, -20},
		{"empty a", []int{}, []int{2}, 0},
		{"empty b", []int{2}, []int{}, 0},
		{"both empty", []int{}, []int{}, 0},
		{"multiple with match", []int{1, 2, 3}, []int{2, 4}, 15},
		{"multiple no match", []int{1, 3}, []int{2, 4}, -20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := numericBonus(tt.aNums, tt.bNums)
			if result != tt.expected {
				t.Errorf("numericBonus(%v, %v) = %d, want %d", tt.aNums, tt.bNums, result, tt.expected)
			}
		})
	}
}

func TestMatchScore(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		minScore int
	}{
		{"exact match", "demon slayer", "demon slayer", 100},
		{"substring match", "demon slayer season 2", "demon slayer", 85},
		{"word overlap", "my hero academia", "hero academia", 66},
		{"no match", "naruto", "bleach", 0},
		{"single word no match", "a", "b", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchScore(tt.a, tt.b)
			if result < tt.minScore {
				t.Errorf("matchScore(%q, %q) = %d, want at least %d", tt.a, tt.b, result, tt.minScore)
			}
		})
	}
}

func TestMatchEntry(t *testing.T) {
	entries := []AnimeEntry{
		{
			MediaID:     1,
			Title:       "Demon Slayer",
			TitleRomaji: "Kimetsu no Yaiba",
			Progress:    5,
		},
		{
			MediaID:     2,
			Title:       "Attack on Titan",
			TitleRomaji: "Shingeki no Kyojin",
			Progress:    10,
		},
		{
			MediaID:     3,
			Title:       "My Hero Academia",
			TitleRomaji: "Boku no Hero Academia",
			Progress:    20,
		},
	}

	tests := []struct {
		name    string
		title   string
		wantID  int
		wantNil bool
	}{
		{"exact match", "Demon Slayer", 1, false},
		{"romaji match", "Kimetsu no Yaiba", 1, false},
		{"partial match", "Attack on", 2, false},
		{"no match", "Nonexistent Anime", 0, true},
		{"empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchEntry(entries, tt.title)
			if tt.wantNil {
				if result != nil {
					t.Errorf("MatchEntry(%q) = %+v, want nil", tt.title, result)
				}
			} else {
				if result == nil {
					t.Errorf("MatchEntry(%q) = nil, want entry with ID %d", tt.title, tt.wantID)
				} else if result.MediaID != tt.wantID {
					t.Errorf("MatchEntry(%q) returned entry ID %d, want %d", tt.title, result.MediaID, tt.wantID)
				}
			}
		})
	}
}

func TestMatchEntryWithProgress(t *testing.T) {
	entries := []AnimeEntry{
		{
			MediaID:     1,
			Title:       "One Piece",
			TitleRomaji: "One Piece",
			Progress:    100,
			TotalEps:    1000,
		},
		{
			MediaID:     2,
			Title:       "One Piece",
			TitleRomaji: "One Piece",
			Progress:    50,
			TotalEps:    1000,
		},
	}

	// When episode number is provided and there are duplicate matches,
	// prefer the entry where progress < episodeNum
	result := MatchEntryWithProgress(entries, "One Piece", 75)
	if result == nil {
		t.Fatal("MatchEntryWithProgress returned nil")
	}

	// Should prefer the entry with Progress=50 since 50 < 75
	if result.MediaID != 2 {
		t.Errorf("MatchEntryWithProgress should prefer entry with lower progress, got ID %d, want 2", result.MediaID)
	}
}

func TestWords(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"demon slayer", []string{"demon", "slayer"}},
		{"the attack on titan", []string{"attack", "titan"}},
		{"my hero academia", []string{"my", "hero", "academia"}},
		{"a an the", []string{}},
		{"", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := words(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("words(%q) returned %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("words(%q)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}
