package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/neyfua/gogoani/internal/httpclient"
	"github.com/neyfua/gogoani/internal/logger"
	"github.com/neyfua/gogoani/internal/scraper"
)

const (
	AniDBBase     = "https://anidb.app"
	AniDBSearch   = AniDBBase + "/browse?q=%s"
	AniDBEpisodes = AniDBBase + "/api/frontend/anime/%s/episodes"
	AniListAPI    = "https://graphql.anilist.co"
)

type AniDB struct {
	mu       sync.RWMutex
	cache    map[string]any
	initOnce sync.Once
}

func NewAniDB() *AniDB {
	return &AniDB{cache: make(map[string]any)}
}

func (a *AniDB) ensureInit() {
	a.initOnce.Do(func() {})
}

type anidbSearchResponse struct {
	Animes []struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		Year  int    `json:"year"`
		Type  string `json:"type"`
	} `json:"animes"`
}

type anidbEpisodeResponse struct {
	Episodes []struct {
		ID      int    `json:"id"`
		Number  int    `json:"number"`
		Number2 string `json:"number2"`
		Filler  bool   `json:"filler"`
	} `json:"episodes"`
}

func (a *AniDB) Search(query string) ([]scraper.Anime, error) {
	a.mu.RLock()
	results, ok := a.cache["search:"+query].([]scraper.Anime)
	a.mu.RUnlock()
	if ok {
		return slices.Clone(results), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	searchURL := fmt.Sprintf(AniDBSearch, url.QueryEscape(query))

	resp, err := httpclient.Request(ctx, "GET", searchURL, nil, nil)
	if err != nil {
		logger.Log.Error("search request failed", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logger.Log.Error("search returned non-200 status", "status", resp.StatusCode)
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		logger.Log.Error("failed to read search response", "error", err)
		return nil, err
	}

	// AniDB returns HTML with anime cards in format:
	// <a href="https://anidb.app/anime/{slug}-{id}" class="anime-card..." title="{title}">
	re := regexp.MustCompile(`href="https://anidb\.app/anime/[^"]+-(\d+)"[^>]*title="([^"]+)"`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	type anidbResult struct {
		ID    string
		Title string
	}

	results = make([]scraper.Anime, 0, len(matches))
	seen := make(map[string]anidbResult)
	for _, m := range matches {
		if len(m) >= 3 {
			animeID := m[1]
			rawTitle := m[2]

			// Extract English title if available
			titleParts := strings.Split(rawTitle, " / ")
			title := titleParts[0]
			if len(titleParts) > 1 {
				title = titleParts[1]
			}

			if _, exists := seen[animeID]; !exists {
				seen[animeID] = anidbResult{ID: animeID, Title: title}
			}
		}
	}

	type metaResult struct {
		ID     string
		Title  string
		Year   string
		Format string
	}
	metaCh := make(chan metaResult, len(seen))
	for _, res := range seen {
		go func(res anidbResult) {
			year, format := fetchAniListMeta(res.Title)
			metaCh <- metaResult{ID: res.ID, Title: res.Title, Year: year, Format: format}
		}(res)
	}

	for range seen {
		mr := <-metaCh
		title := mr.Title
		info := []string{}
		if mr.Year != "" {
			info = append(info, mr.Year)
		}
		if mr.Format == "MOVIE" {
			info = append(info, "Movie")
		}
		if len(info) > 0 {
			title = fmt.Sprintf("%s (%s)", title, strings.Join(info, " "))
		} else {
			title = fmt.Sprintf("%s (N/A)", title)
		}
		results = append(results, scraper.Anime{ID: mr.ID, Title: title})
	}

	// Sort by year: Oldest at top (2013), newest at bottom (2026), N/A items at the very bottom
	slices.SortFunc(results, func(i, j scraper.Anime) int {
		getY := func(s string) string {
			re := regexp.MustCompile(`\((\d{4})`)
			m := re.FindStringSubmatch(s)
			if len(m) > 1 {
				return m[1]
			}
			return "9999"
		}
		y1 := getY(i.Title)
		y2 := getY(j.Title)
		if y1 != y2 {
			return strings.Compare(y1, y2)
		}
		return strings.Compare(i.Title, j.Title)
	})

	a.mu.Lock()
	a.cache["search:"+query] = results
	a.mu.Unlock()
	logger.Log.Debug("search completed", "query", query, "results", len(results))
	return results, nil
}

func fetchAniListMeta(title string) (string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	payload := map[string]any{
		"query":     `query ($search: String) { Media(search: $search, type: ANIME) { startDate { year } format } }`,
		"variables": map[string]any{"search": title},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		return "", ""
	}

	resp, err := httpclient.Request(ctx, "POST", AniListAPI, map[string]string{"Content-Type": "application/json", "Accept": "application/json"}, &buf)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", ""
	}

	var out struct {
		Data struct {
			Media struct {
				StartDate struct {
					Year int `json:"year"`
				} `json:"startDate"`
				Format string `json:"format"`
			} `json:"Media"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1024*1024)).Decode(&out); err != nil {
		return "", ""
	}
	if out.Data.Media.StartDate.Year == 0 {
		return "", out.Data.Media.Format
	}
	return fmt.Sprintf("%d", out.Data.Media.StartDate.Year), out.Data.Media.Format
}

func (a *AniDB) Episodes(anime scraper.Anime, mode string) ([]scraper.Episode, error) {
	cacheKey := "episodes:" + anime.ID + ":" + mode
	a.mu.RLock()
	results, ok := a.cache[cacheKey].([]scraper.Episode)
	a.mu.RUnlock()
	if ok {
		return slices.Clone(results), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	episodesURL := fmt.Sprintf(AniDBEpisodes, anime.ID)

	resp, err := httpclient.Request(ctx, "GET", episodesURL, nil, nil)
	if err != nil {
		logger.Log.Error("episodes request failed", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logger.Log.Error("episodes returned non-200 status", "status", resp.StatusCode)
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var epResp anidbEpisodeResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10*1024*1024)).Decode(&epResp); err != nil {
		logger.Log.Error("failed to decode episodes response", "error", err)
		return nil, err
	}

	episodes := make([]scraper.Episode, 0, len(epResp.Episodes))
	for i, ep := range epResp.Episodes {
		if ep.Number <= 0 {
			continue
		}
		episodes = append(episodes, scraper.Episode{
			Number: i + 1,
			Title:  "",
			Mode:   mode,
		})
	}

	a.mu.Lock()
	a.cache[cacheKey] = episodes
	a.mu.Unlock()
	logger.Log.Debug("episodes fetched", "anime_id", anime.ID, "count", len(episodes), "mode", mode)
	return episodes, nil
}

type anidbLanguage struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	EmbedURL string `json:"embed_url"`
}

type anidbLanguageResponse struct {
	Languages []anidbLanguage `json:"languages"`
}

func (a *AniDB) StreamURL(anime scraper.Anime, episode scraper.Episode) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Get the episodes to find the episode ID (not the same as episode number)
	episodesURL := fmt.Sprintf(AniDBEpisodes, anime.ID)

	resp, err := httpclient.Request(ctx, "GET", episodesURL, nil, nil)
	if err != nil {
		logger.Log.Error("episodes request failed", "error", err)
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logger.Log.Error("episodes returned non-200 status", "status", resp.StatusCode)
		return "", "", fmt.Errorf("status %d", resp.StatusCode)
	}

	var epResp struct {
		Episodes []struct {
			ID      int    `json:"id"`
			Number  int    `json:"number"`
			Number2 string `json:"number2"`
			Filler  bool   `json:"filler"`
		} `json:"episodes"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10*1024*1024)).Decode(&epResp); err != nil {
		logger.Log.Error("failed to decode episodes response", "error", err)
		return "", "", err
	}

	// Find the episode with matching number
	var epID int
	for _, ep := range epResp.Episodes {
		if ep.Number == episode.Number {
			epID = ep.ID
			break
		}
	}

	if epID == 0 {
		logger.Log.Error("episode not found", "anime_id", anime.ID, "episode", episode.Number)
		return "", "", fmt.Errorf("episode %d not found", episode.Number)
	}

	// Get stream languages
	streamURL := fmt.Sprintf("%s/api/frontend/episode/%d/languages", AniDBBase, epID)
	resp2, err := httpclient.Request(ctx, "GET", streamURL, nil, nil)
	if err != nil {
		logger.Log.Error("stream languages request failed", "error", err)
		return "", "", err
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != 200 {
		logger.Log.Error("stream languages returned non-200 status", "status", resp2.StatusCode)
		return "", "", fmt.Errorf("status %d", resp2.StatusCode)
	}

	var langResp anidbLanguageResponse
	if err := json.NewDecoder(io.LimitReader(resp2.Body, 10*1024*1024)).Decode(&langResp); err != nil {
		logger.Log.Error("failed to decode stream languages", "error", err)
		return "", "", err
	}

	// Get embed URL for requested language
	var embedURL string
	for _, lang := range langResp.Languages {
		if (episode.Mode == "dub" && strings.Contains(lang.Name, "English")) ||
			(episode.Mode == "sub" && strings.Contains(lang.Name, "Japanese")) {
			embedURL = lang.EmbedURL
			break
		}
	}

	if embedURL == "" {
		// Fallback to first available
		if len(langResp.Languages) > 0 {
			embedURL = langResp.Languages[0].EmbedURL
		}
	}

	if embedURL == "" {
		logger.Log.Error("no embed URL found", "anime_id", anime.ID, "episode", episode.Number, "mode", episode.Mode)
		return "", "", fmt.Errorf("no stream URL available")
	}

	// Fetch the embed page to get the m3u8 URL
	resp3, err := httpclient.Request(ctx, "GET", embedURL, nil, nil)
	if err != nil {
		logger.Log.Error("embed page request failed", "error", err)
		return "", "", err
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != 200 {
		logger.Log.Error("embed page returned non-200 status", "status", resp3.StatusCode)
		return "", "", fmt.Errorf("status %d", resp3.StatusCode)
	}

	embedPage, err := io.ReadAll(io.LimitReader(resp3.Body, 10*1024*1024))
	if err != nil {
		logger.Log.Error("failed to read embed page", "error", err)
		return "", "", err
	}

	// Extract m3u8 URL from JavaScript: file: 'https://...m3u8'
	re := regexp.MustCompile(`file:\s*['"]([^'"]*\.m3u8[^'"]*)['"]`)
	matches := re.FindStringSubmatch(string(embedPage))

	var streamURLFinal string
	if len(matches) > 1 {
		streamURLFinal = matches[1]
		// Fix escaped slashes
		streamURLFinal = strings.ReplaceAll(streamURLFinal, "\\/", "/")
	}

	if streamURLFinal == "" {
		logger.Log.Error("no m3u8 URL in embed page", "anime_id", anime.ID, "episode", episode.Number)
		return "", "", fmt.Errorf("no playable stream URL found")
	}

	referer := embedURL
	return streamURLFinal, referer, nil
}
