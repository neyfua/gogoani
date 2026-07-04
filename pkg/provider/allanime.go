package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/neyfua/gogoani/internal/crypto"
	"github.com/neyfua/gogoani/internal/httpclient"
	"github.com/neyfua/gogoani/internal/logger"
	"github.com/neyfua/gogoani/internal/scraper"
)

const (
	AllAnimeAPI = "https://api.allanime.day/api"
	AllAnimeRef = "https://youtu-chan.com"
)

type AllAnime struct {
	mu    sync.RWMutex
	cache map[string]any // in-memory cache for search and episodes
}

func NewAllAnime() *AllAnime {
	return &AllAnime{cache: make(map[string]any)}
}

type gqlResponse struct {
	Data struct {
		ToBeParsed string `json:"tobeparsed"`
		Shows      struct {
			Edges []struct {
				ID                string `json:"_id"`
				Name              string `json:"name"`
				AvailableEpisodes struct {
					Sub int `json:"sub"`
					Dub int `json:"dub"`
				} `json:"availableEpisodes"`
			} `json:"edges"`
		} `json:"shows"`
		Show struct {
			ID                string `json:"_id"`
			AvailableEpisodes struct {
				Sub int `json:"sub"`
				Dub int `json:"dub"`
			} `json:"availableEpisodes"`
			AvailableEpisodesDetail struct {
				Sub []any `json:"sub"`
				Dub []any `json:"dub"`
			} `json:"availableEpisodesDetail"`
		} `json:"show"`
		Episode struct {
			EpisodeString string `json:"episodeString"`
			SourceURLs    []struct {
				SourceName string  `json:"sourceName"`
				SourceURL  string  `json:"sourceUrl"`
				Priority   float64 `json:"priority"`
			} `json:"sourceUrls"`
		} `json:"episode"`
	} `json:"data"`
}

func processResponse(resp *gqlResponse) error {
	if resp.Data.ToBeParsed != "" {
		decrypted, err := crypto.DecryptAllAnime(resp.Data.ToBeParsed)
		if err != nil {
			logger.Log.Error("failed to decrypt response", "error", err)
			return err
		}
		if err := json.Unmarshal([]byte(decrypted), &resp.Data); err != nil {
			logger.Log.Error("failed to unmarshal decrypted data", "error", err)
			return err
		}
	}
	return nil
}

// Search queries AllAnime for anime matching the query
func (a *AllAnime) Search(query string) ([]scraper.Anime, error) {
	a.mu.RLock()
	results, ok := a.cache["search:"+query].([]scraper.Anime)
	a.mu.RUnlock()
	if ok {
		return slices.Clone(results), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	searchGQL := `query( $search: SearchInput $limit: Int $page: Int $translationType: VaildTranslationTypeEnumType $countryOrigin: VaildCountryOriginEnumType ) { shows( search: $search limit: $limit page: $page translationType: $translationType countryOrigin: $countryOrigin ) { edges { _id name availableEpisodes __typename } }}`

	payload := map[string]any{
		"variables": map[string]any{
			"search": map[string]any{
				"allowAdult":   false,
				"allowUnknown": false,
				"query":        query,
			},
			"limit":           40,
			"page":            1,
			"translationType": "sub",
			"countryOrigin":   "ALL",
		},
		"query": searchGQL,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		logger.Log.Error("failed to marshal search payload", "error", err)
		return nil, err
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"Origin":       AllAnimeRef,
		"Referer":      AllAnimeRef,
	}

	resp, err := httpclient.Request(ctx, "POST", AllAnimeAPI, headers, bytes.NewReader(buf.Bytes()))
	if err != nil {
		logger.Log.Error("search request failed", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logger.Log.Error("search returned non-200 status", "status", resp.StatusCode)
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var gqlResp gqlResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10*1024*1024)).Decode(&gqlResp); err != nil {
		logger.Log.Error("failed to decode search response", "error", err)
		return nil, err
	}
	if err := processResponse(&gqlResp); err != nil {
		return nil, err
	}

	edges := gqlResp.Data.Shows.Edges
	results = make([]scraper.Anime, 0, len(edges))
	for _, edge := range edges {
		results = append(results, scraper.Anime{
			ID:    edge.ID,
			Title: edge.Name,
		})
	}

	a.mu.Lock()
	a.cache["search:"+query] = results
	a.mu.Unlock()
	logger.Log.Debug("search completed", "query", query, "results", len(results))
	return results, nil
}

// Episodes fetches the episode list for an anime
func (a *AllAnime) Episodes(anime scraper.Anime, mode string) ([]scraper.Episode, error) {
	cacheKey := "episodes:" + anime.ID + ":" + mode
	a.mu.RLock()
	results, ok := a.cache[cacheKey].([]scraper.Episode)
	a.mu.RUnlock()
	if ok {
		return slices.Clone(results), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	episodesGQL := `query ($showId: String!) { show( _id: $showId ) { _id availableEpisodes availableEpisodesDetail }}`

	payload := map[string]any{
		"variables": map[string]any{
			"showId": anime.ID,
		},
		"query": episodesGQL,
	}

	buf := httpclient.GetBuf()
	defer httpclient.PutBuf(buf)
	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		logger.Log.Error("failed to marshal episodes payload", "error", err)
		return nil, err
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"Origin":       AllAnimeRef,
		"Referer":      AllAnimeRef,
	}

	resp, err := httpclient.Request(ctx, "POST", AllAnimeAPI, headers, bytes.NewReader(buf.Bytes()))
	if err != nil {
		logger.Log.Error("episodes request failed", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logger.Log.Error("episodes returned non-200 status", "status", resp.StatusCode)
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var gqlResp gqlResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10*1024*1024)).Decode(&gqlResp); err != nil {
		logger.Log.Error("failed to decode episodes response", "error", err)
		return nil, err
	}
	if err := processResponse(&gqlResp); err != nil {
		return nil, err
	}

	var episodeList []any
	if mode == "dub" {
		episodeList = gqlResp.Data.Show.AvailableEpisodesDetail.Dub
	} else {
		episodeList = gqlResp.Data.Show.AvailableEpisodesDetail.Sub
	}

	episodes := make([]scraper.Episode, 0, len(episodeList))
	for _, raw := range episodeList {
		s, ok := raw.(string)
		if !ok {
			logger.Log.Debug("skipping non-string episode entry", "type", fmt.Sprintf("%T", raw))
			continue
		}
		num, err := strconv.Atoi(s)
		if err != nil || num <= 0 {
			logger.Log.Debug("skipping invalid episode entry", "value", s)
			continue
		}
		episodes = append(episodes, scraper.Episode{
			Number: num,
			Title:  "",
			Mode:   mode,
		})
	}

	// Reverse episodes to get ascending order (API returns descending)
	slices.Reverse(episodes)

	a.mu.Lock()
	a.cache[cacheKey] = episodes
	a.mu.Unlock()
	logger.Log.Debug("episodes fetched", "anime_id", anime.ID, "count", len(episodes), "mode", mode)
	return episodes, nil
}

// StreamURL fetches the direct stream URL for an episode
func (a *AllAnime) StreamURL(anime scraper.Anime, episode scraper.Episode) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	vars := map[string]any{
		"showId":          anime.ID,
		"translationType": episode.Mode,
		"episodeString":   strconv.Itoa(episode.Number),
	}
	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return "", "", err
	}

	exts := map[string]any{
		"persistedQuery": map[string]any{
			"version":    1,
			"sha256Hash": "d405d0edd690624b66baba3068e0edc3ac90f1597d898a1ec8db4e5c43c00fec",
		},
	}
	extsJSON, err := json.Marshal(exts)
	if err != nil {
		return "", "", err
	}

	u, err := url.Parse(AllAnimeAPI)
	if err != nil {
		return "", "", err
	}
	q := u.Query()
	q.Set("variables", string(varsJSON))
	q.Set("extensions", string(extsJSON))
	u.RawQuery = q.Encode()

	headers := map[string]string{
		"Origin":  AllAnimeRef,
		"Referer": AllAnimeRef,
	}

	resp, err := httpclient.Request(ctx, "GET", u.String(), headers, nil)
	if err != nil {
		logger.Log.Error("stream request failed", "error", err)
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logger.Log.Error("stream returned non-200 status", "status", resp.StatusCode)
		return "", "", fmt.Errorf("status %d", resp.StatusCode)
	}

	var gqlResp gqlResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10*1024*1024)).Decode(&gqlResp); err != nil {
		logger.Log.Error("failed to decode stream response", "error", err)
		return "", "", err
	}
	if err := processResponse(&gqlResp); err != nil {
		return "", "", err
	}

	if len(gqlResp.Data.Episode.SourceURLs) == 0 {
		raw, _ := json.Marshal(gqlResp.Data)
		logger.Log.Error("no sources found",
			"anime_id", anime.ID,
			"episode", episode.Number,
			"data", string(raw),
		)
		return "", "", fmt.Errorf("no sources available")
	}

	// Sort sources by priority descending (highest priority first)
	slices.SortFunc(gqlResp.Data.Episode.SourceURLs, func(a, b struct {
		SourceName string  `json:"sourceName"`
		SourceURL  string  `json:"sourceUrl"`
		Priority   float64 `json:"priority"`
	}) int {
		return slices.Compare([]float64{b.Priority}, []float64{a.Priority})
	})

	var sourceURL string
	var priority float64
	for _, s := range gqlResp.Data.Episode.SourceURLs {
		u := s.SourceURL
		if strings.HasPrefix(u, "--") {
			u = crypto.DecodeHexMap(u)
		}
		if strings.Contains(u, "tobeparsed") {
			dec, err := crypto.DecryptAllAnime(u)
			if err != nil {
				continue
			}
			u = dec
		}

		// Skip /apivtwo/clock.json URLs as they are currently problematic
		if strings.Contains(u, "clock.json") {
			continue
		}

		sourceURL = u
		priority = s.Priority
		break
	}

	if sourceURL == "" {
		logger.Log.Error("no playable sources found", "anime_id", anime.ID, "episode", episode.Number)
		return "", "", fmt.Errorf("no playable sources available")
	}

	if !strings.HasPrefix(sourceURL, "http") && !strings.HasPrefix(sourceURL, "//") {
		// If it's a relative path, prepend the host
		sourceURL = "https://tools.fast4speed.rsvp" + sourceURL
	}

	referer := AllAnimeRef
	if strings.Contains(sourceURL, "mp4upload.com") {
		referer = "https://www.mp4upload.com"
	}

	logger.Log.Debug("stream url fetched", "anime_id", anime.ID, "episode", episode.Number, "priority", priority)
	return sourceURL, referer, nil
}
