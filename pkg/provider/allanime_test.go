package provider

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/neyfua/gogoani/internal/scraper"
)

// Benchmark cache lookup: sync.Map vs plain map

func BenchmarkSyncMapLoad(b *testing.B) {
	var m sync.Map
	for i := range 1000 {
		m.Store(strconv.Itoa(i), i)
	}
	keys := make([]string, 1000)
	for i := range 1000 {
		keys[i] = strconv.Itoa(i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, k := range keys {
			m.Load(k)
		}
	}
}

func BenchmarkPlainMapLoad(b *testing.B) {
	m := make(map[string]int, 1000)
	for i := range 1000 {
		m[strconv.Itoa(i)] = i
	}
	keys := make([]string, 1000)
	for i := range 1000 {
		keys[i] = strconv.Itoa(i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, k := range keys {
			_ = m[k]
		}
	}
}

func BenchmarkSyncMapStore(b *testing.B) {
	var m sync.Map
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for i := range 1000 {
			m.Store(strconv.Itoa(i), i)
		}
	}
}

func BenchmarkPlainMapStore(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		m := make(map[string]int, 1000)
		for i := range 1000 {
			m[strconv.Itoa(i)] = i
		}
	}
}

// Benchmark episode number parsing: fmt.Sscanf vs strconv.Atoi

func BenchmarkSscanfEpisode(b *testing.B) {
	vals := []string{"1", "5", "12", "100", "999"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, s := range vals {
			var n int
			_, _ = fmt.Sscanf(s, "%d", &n)
		}
	}
}

func BenchmarkAtoiEpisode(b *testing.B) {
	vals := []string{"1", "5", "12", "100", "999"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, s := range vals {
			n, _ := strconv.Atoi(s)
			_ = n
		}
	}
}

// Benchmark string concat for Anime URL

func BenchmarkURLConcat(b *testing.B) {
	ids := make([]string, 100)
	for i := range 100 {
		ids[i] = "abc" + strconv.Itoa(i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, id := range ids {
			_ = AllAnimeAPI + "/anime/" + id
		}
	}
}

// Benchmark full episode parsing (current fmt.Sscanf vs strconv.Atoi)

func BenchmarkEpisodeParsingSscanf(b *testing.B) {
	raw := make([]any, 0, 100)
	for i := 1; i <= 100; i++ {
		raw = append(raw, strconv.Itoa(i))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		episodes := make([]scraper.Episode, 0, len(raw))
		for _, r := range raw {
			s, ok := r.(string)
			if !ok {
				continue
			}
			var num int
			if _, err := fmt.Sscanf(s, "%d", &num); err != nil || num <= 0 {
				continue
			}
			episodes = append(episodes, scraper.Episode{Number: num, Mode: "sub"})
		}
	}
}

func BenchmarkEpisodeParsingAtoi(b *testing.B) {
	raw := make([]any, 0, 100)
	for i := 1; i <= 100; i++ {
		raw = append(raw, strconv.Itoa(i))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		episodes := make([]scraper.Episode, 0, len(raw))
		for _, r := range raw {
			s, ok := r.(string)
			if !ok {
				continue
			}
			num, err := strconv.Atoi(s)
			if err != nil || num <= 0 {
				continue
			}
			episodes = append(episodes, scraper.Episode{Number: num, Mode: "sub"})
		}
	}
}
