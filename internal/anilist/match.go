package anilist

import (
	"slices"
	"strconv"
	"strings"
)

var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true,
	"of": true, "in": true, "to": true, "no": true, "is": true, "it": true,
	"wa": true, "ga": true, "ni": true, "de": true, "wo": true,
	"ha": true, "mo": true, "ka": true, "ya": true, "yo": true,
	"its": true, "but": true, "not": true, "for": true, "on": true,
	"at": true, "by": true, "with": true, "this": true, "that": true,
	"da": true, "desu": true, "masu": true, "koto": true,
}

var ordinalMap = map[string]int{
	"first": 1, "second": 2, "third": 3, "fourth": 4, "fifth": 5,
	"sixth": 6, "seventh": 7, "eighth": 8, "ninth": 9, "tenth": 10,
	"1st": 1, "2nd": 2, "3rd": 3, "4th": 4, "5th": 5,
	"6th": 6, "7th": 7, "8th": 8, "9th": 9, "10th": 10,
}

func MatchEntry(entries []AnimeEntry, title string) *AnimeEntry {
	return MatchEntryWithProgress(entries, title, 0)
}

func MatchEntryWithProgress(entries []AnimeEntry, title string, episodeNum int) *AnimeEntry {
	title = strings.ToLower(strings.TrimSpace(title))
	if title == "" {
		return nil
	}

	cleanTitle := clean(title)
	titleNums := extractNumbers(cleanTitle)

	type scored struct {
		entry *AnimeEntry
		score int
	}
	var candidates []scored
	var bestScore int

	for i := range entries {
		e := &entries[i]
		s := scoreForEntry(e, cleanTitle, titleNums)
		if s < 30 {
			continue
		}
		if s > bestScore {
			bestScore = s
		}
		candidates = append(candidates, scored{e, s})
	}

	if bestScore < 30 {
		return nil
	}

	// Filter to best-score entries
	best := candidates[:0]
	for _, c := range candidates {
		if c.score == bestScore {
			best = append(best, c)
		}
	}

	if episodeNum > 0 && len(best) > 1 {
		slices.SortStableFunc(best, func(a, b scored) int {
			aFuture := a.entry.Progress < episodeNum
			bFuture := b.entry.Progress < episodeNum
			if aFuture != bFuture {
				if aFuture {
					return -1
				}
				return 1
			}
			return 0
		})
	}

	return best[0].entry
}

func scoreForEntry(e *AnimeEntry, cleanTitle string, titleNums []int) int {
	best := 0
	for _, t := range e.Titles() {
		etitle := strings.ToLower(strings.TrimSpace(t))
		eclean := clean(etitle)
		score := matchScore(cleanTitle, eclean)
		score += numericBonus(titleNums, extractNumbers(eclean))
		if score > best {
			best = score
		}
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

func extractNumbers(s string) []int {
	var nums []int
	for _, f := range strings.Fields(s) {
		if n, err := strconv.Atoi(f); err == nil {
			nums = append(nums, n)
			continue
		}
		if n, ok := ordinalMap[f]; ok {
			nums = append(nums, n)
		}
	}
	return nums
}

func numericBonus(aNums, bNums []int) int {
	if len(aNums) == 0 || len(bNums) == 0 {
		return 0
	}
	for _, an := range aNums {
		for _, bn := range bNums {
			if an == bn {
				return 15
			}
		}
	}
	return -20
}

func matchScore(a, b string) int {
	if a == b {
		return 100
	}
	aw := words(a)
	bw := words(b)
	if len(aw) >= 2 && len(bw) >= 2 && (strings.Contains(a, b) || strings.Contains(b, a)) {
		return 85
	}
	if len(aw) < 2 || len(bw) < 2 {
		if strings.Contains(a, b) || strings.Contains(b, a) {
			if len(a) >= 2 && len(b) >= 2 {
				return 60
			}
		}
		return 0
	}
	matchCount := 0
	for _, wa := range aw {
		if slices.Contains(bw, wa) {
			matchCount++
		}
	}
	minLen := min(len(bw), len(aw))
	if minLen == 0 {
		return 0
	}
	return (matchCount * 100) / minLen
}
