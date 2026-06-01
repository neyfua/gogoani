package anilist

import (
	"strings"
)

var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true,
	"of": true, "in": true, "to": true, "no": true, "is": true, "it": true,
	"wa": true, "ga": true, "ni": true, "de": true, "wo": true,
	"ha": true, "mo": true, "ka": true, "ya": true, "yo": true,
	"its": true, "but": true, "not": true, "for": true, "on": true,
	"at": true, "by": true, "with": true, "this": true, "that": true,
}

func MatchEntry(entries []AnimeEntry, title string) *AnimeEntry {
	title = strings.ToLower(strings.TrimSpace(title))
	if title == "" {
		return nil
	}

	cleanTitle := clean(title)

	var best *AnimeEntry
	var bestScore int

	for _, e := range entries {
		etitle := strings.ToLower(strings.TrimSpace(e.Title))
		eclean := clean(etitle)
		score := matchScore(cleanTitle, eclean)
		if score > bestScore {
			bestScore = score
			best = &e
		}
	}

	if bestScore < 30 {
		return nil
	}
	return best
}

func clean(s string) string {
	s = strings.NewReplacer(
		"’", "'", "‘", "'",
		"“", "\"", "”", "\"",
		":", " ", "-", " ",
		"—", " ", "–", " ",
		"(", " ", ")", " ",
		"[", " ", "]", " ",
		",", " ", ".", " ",
		"!", " ", "?", " ",
		";", " ", "/", " ",
	).Replace(s)
	s = strings.ToLower(s)
	return strings.Join(strings.Fields(s), " ")
}

func words(s string) []string {
	all := strings.Fields(s)
	filtered := make([]string, 0, len(all))
	for _, w := range all {
		if !stopWords[w] {
			filtered = append(filtered, w)
		}
	}
	return filtered
}

func matchScore(a, b string) int {
	if a == b {
		return 100
	}
	if strings.Contains(a, b) || strings.Contains(b, a) {
		return 80
	}
	aw := words(a)
	bw := words(b)
	if len(aw) < 2 || len(bw) < 2 {
		return 0
	}
	matchCount := 0
	for _, wa := range aw {
		for _, wb := range bw {
			if wa == wb {
				matchCount++
				break
			}
		}
	}
	minLen := len(aw)
	if len(bw) < minLen {
		minLen = len(bw)
	}
	if minLen == 0 {
		return 0
	}
	return (matchCount * 100) / minLen
}
