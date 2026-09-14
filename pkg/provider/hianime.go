package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/neyfua/gogoani/internal/httpclient"
	"github.com/neyfua/gogoani/internal/logger"
	"github.com/neyfua/gogoani/internal/scraper"
)

const (
	HiAnimeBase     = "https://hianime.at"
	HiAnimeSearch   = HiAnimeBase + "/search?keyword=%s"
	HiAnimeEpisodes = HiAnimeBase + "/api/theme/episode/list/%s"
	HiAnimeServers  = HiAnimeBase + "/api/theme/episode/servers?episodeId=%s"
	HiAnimeDetail   = HiAnimeBase + "/%s"
	AniListAPI      = "https://graphql.anilist.co"
)

// xorKey is the deobfuscation key used by the HiAnime embed player.
// Decodes to "otaku-embed-v1".
var xorKey = []byte{111, 116, 97, 107, 117, 45, 101, 109, 98, 101, 100, 45, 118, 49}

var (
	blockRe  = regexp.MustCompile(`<div class="film-detail">[\s\S]*?</div>\s*<div class="clearfix"></div>`)
	slugRe   = regexp.MustCompile(`<a\s+href="[^"]+/([^"/]+)"\s+title="([^"]+)"\s+class="dynamic-name"\s+data-jname="([^"]*)"`)
	formatRe = regexp.MustCompile(`<span class="fdi-item">([^<]+)</span>`)
	yearRe   = regexp.MustCompile(`\((\d{4})`)
	airedRe  = regexp.MustCompile(`Aired:</span>[\s\S]{0,300}?((?:19|20)\d{2})`)
	epRe     = regexp.MustCompile(`data-number="([^"]*)"[\s\S]*?data-id="([0-9]+)"[\s\S]*?/watch/([^"?]+)\?ep=`)
	blobRe   = regexp.MustCompile(`window\.__P="([^"]*)"`)
	m3u8Re   = regexp.MustCompile(`"src":"([^"]*\.m3u8[^"]*)"`)
	srvBlkRe = regexp.MustCompile(`<div class="item server-item"[\s\S]*?</div>`)
	srvAtrRe = regexp.MustCompile(`data-type="([^"]*)"[\s\S]*?data-server-name="([^"]*)"[\s\S]*?data-hash="([^"]*)"`)
)

type yearEntry struct {
	year string
	at   time.Time
}

type HiAnime struct {
	mu        sync.RWMutex
	cache     map[string]any
	yearCache map[string]yearEntry
}

func NewHiAnime() *HiAnime {
	return &HiAnime{cache: make(map[string]any), yearCache: make(map[string]yearEntry)}
}

func decodeHTMLEntities(s string) string {
	s = strings.ReplaceAll(s, "&#039;", "'")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return s
}

func (h *HiAnime) searchRaw(query string) ([]scraper.Anime, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// HiAnime expects spaces as "+" like ani-cli does (tr ' ' '+')
	searchURL := fmt.Sprintf(HiAnimeSearch, strings.ReplaceAll(query, " ", "+"))

	resp, err := httpclient.Request(ctx, "GET", searchURL, nil, nil)
	if err != nil {
		logger.Log.Error("hianime: search request failed", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logger.Log.Error("hianime: search returned non-200 status", "status", resp.StatusCode)
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		logger.Log.Error("hianime: failed to read search response", "error", err)
		return nil, err
	}

	// Strip sidebar and everything after it to avoid matching top-10 / trending widgets
	html := string(body)
	if idx := strings.Index(html, `id="main-sidebar"`); idx != -1 {
		html = html[:idx]
	}

	// Capture EN title + JP name (data-jname) + format from each film block
	// Format is the first fdi-item in fd-infor (TV, MOVIE, OVA, ONA, SPECIAL)
	blocks := blockRe.FindAllString(html, -1)

	type filmInfo struct {
		Slug          string
		Title         string
		TitleJapanese string
		Format        string
	}

	results := make([]scraper.Anime, 0, len(blocks))
	seen := make(map[string]filmInfo)
	for _, block := range blocks {
		slugMatch := slugRe.FindStringSubmatch(block)
		if slugMatch == nil || len(slugMatch) < 4 {
			continue
		}
		slug := slugMatch[1]
		enTitle := decodeHTMLEntities(slugMatch[2])
		jpTitle := decodeHTMLEntities(slugMatch[3])
		key := strings.ToLower(strings.TrimSpace(enTitle))
		if _, exists := seen[key]; exists {
			continue
		}
		format := ""
		if fm := formatRe.FindStringSubmatch(block); fm != nil {
			format = strings.TrimSpace(fm[1])
		}
		seen[key] = filmInfo{Slug: slug, Title: enTitle, TitleJapanese: jpTitle, Format: format}
	}

	// Enrich each result with year: hianime detail page first, AniList as fallback.
	// Format comes from hianime search HTML directly; AniList format is fallback.
	type metaResult struct {
		ID            string
		Title         string
		TitleJapanese string
		Year          string
		Format        string
	}
	metaCh := make(chan metaResult, len(seen))
	for _, res := range seen {
		go func(res filmInfo) {
			year := h.fetchHiAnimeYear(res.Slug)
			var alFormat string
			if year == "" {
				var alYear string
				alYear, alFormat = fetchAniListMeta(res.Title)
				if year == "" {
					year = alYear
				}
			}
			format := res.Format
			if format == "" {
				format = alFormat
			}
			metaCh <- metaResult{ID: res.Slug, Title: res.Title, TitleJapanese: res.TitleJapanese, Year: year, Format: format}
		}(res)
	}

	for range seen {
		mr := <-metaCh
		info := []string{}
		if mr.Year != "" {
			info = append(info, mr.Year)
		}
		if mr.Format == "MOVIE" {
			info = append(info, "Movie")
		}
		display := mr.Title
		if mr.TitleJapanese != "" && !strings.EqualFold(mr.TitleJapanese, mr.Title) {
			display = fmt.Sprintf("%s [%s]", mr.Title, mr.TitleJapanese)
		}
		if len(info) > 0 {
			display = fmt.Sprintf("%s (%s)", display, strings.Join(info, " "))
		}
		results = append(results, scraper.Anime{ID: mr.ID, Title: display, TitleJapanese: mr.TitleJapanese})
	}

	// Sort by year: oldest first; items without year sort by title
	slices.SortFunc(results, func(i, j scraper.Anime) int {
		getY := func(s string) string {
			if m := yearRe.FindStringSubmatch(s); len(m) > 1 {
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

	logger.Log.Debug("hianime: search raw", "query", query, "results", len(results))
	return results, nil
}

func (h *HiAnime) fetchHiAnimeYear(slug string) string {
	h.mu.RLock()
	if e, ok := h.yearCache[slug]; ok && time.Since(e.at) < 24*time.Hour {
		h.mu.RUnlock()
		return e.year
	}
	h.mu.RUnlock()

	year := fetchHiAnimeYearPage(slug)

	h.mu.Lock()
	h.yearCache[slug] = yearEntry{year: year, at: time.Now()}
	h.mu.Unlock()
	return year
}

func fetchHiAnimeYearPage(slug string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	detailURL := fmt.Sprintf(HiAnimeDetail, slug)
	resp, err := httpclient.Request(ctx, "GET", detailURL, nil, nil)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return ""
	}

	// Detail page has: <span class="item-head">Aired:</span> ... Oct 5, 2004 to Mar 27, 2012
	// First 4-digit year is the release year
	if m := airedRe.FindStringSubmatch(string(body)); m != nil {
		return m[1]
	}
	return ""
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

func (h *HiAnime) Search(query string) ([]scraper.Anime, error) {
	h.mu.RLock()
	results, ok := h.cache["search:"+query].([]scraper.Anime)
	h.mu.RUnlock()
	if ok {
		return slices.Clone(results), nil
	}

	results, err := h.searchRaw(query)
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	h.cache["search:"+query] = results
	h.mu.Unlock()
	logger.Log.Debug("hianime: search completed", "query", query, "results", len(results))
	return results, nil
}

type hianimeEpisodeListResponse struct {
	Status     bool   `json:"status"`
	TotalItems int    `json:"totalItems"`
	HTML       string `json:"html"`
}

func fetchEpisodeMaps(ctx context.Context, animeID string) (string, error) {
	// Extract numeric ID from slug (e.g. "one-piece-12345" -> "12345")
	parts := strings.Split(animeID, "-")
	numericID := parts[len(parts)-1]

	episodesURL := fmt.Sprintf(HiAnimeEpisodes, numericID)

	resp, err := httpclient.Request(ctx, "GET", episodesURL, nil, nil)
	if err != nil {
		logger.Log.Error("hianime: episodes request failed", "error", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logger.Log.Error("hianime: episodes returned non-200 status", "status", resp.StatusCode)
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	var epResp hianimeEpisodeListResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10*1024*1024)).Decode(&epResp); err != nil {
		logger.Log.Error("hianime: failed to decode episodes response", "error", err)
		return "", err
	}

	return epResp.HTML, nil
}

type hianimeEpisodeEntry struct {
	Number int
	DataID string
}

func parseEpisodeEntries(html string, animeID string) []hianimeEpisodeEntry {
	// Parse episode items from the HTML field of the API response
	// Each ep-item contains data-number, data-id, and a watch URL with the anime slug
	epMatches := epRe.FindAllStringSubmatch(html, -1)

	entries := make([]hianimeEpisodeEntry, 0, len(epMatches))
	for _, m := range epMatches {
		if len(m) < 4 || m[3] != animeID {
			continue
		}
		num, err := strconv.Atoi(m[1])
		if err != nil || num <= 0 {
			continue
		}
		entries = append(entries, hianimeEpisodeEntry{Number: num, DataID: m[2]})
	}
	return entries
}

func parseEpisodeMaps(html string, animeID string) []scraper.Episode {
	entries := parseEpisodeEntries(html, animeID)
	episodes := make([]scraper.Episode, 0, len(entries))
	for _, e := range entries {
		episodes = append(episodes, scraper.Episode{Number: e.Number, Title: ""})
	}
	return episodes
}

func (h *HiAnime) Episodes(anime scraper.Anime, mode string) ([]scraper.Episode, error) {
	cacheKey := "episodes:" + anime.ID + ":" + mode
	h.mu.RLock()
	results, ok := h.cache[cacheKey].([]scraper.Episode)
	h.mu.RUnlock()
	if ok {
		return slices.Clone(results), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	html, err := fetchEpisodeMaps(ctx, anime.ID)
	if err != nil {
		return nil, err
	}

	episodes := parseEpisodeMaps(html, anime.ID)
	for i := range episodes {
		episodes[i].Mode = mode
	}

	h.mu.Lock()
	h.cache[cacheKey] = episodes
	h.mu.Unlock()
	logger.Log.Debug("hianime: episodes fetched", "anime_id", anime.ID, "count", len(episodes), "mode", mode)
	return episodes, nil
}

// deobfuscateBlob decodes a base64 blob using a rotating XOR key.
func deobfuscateBlob(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}

	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = b ^ xorKey[i%len(xorKey)]
	}

	if !utf8.Valid(out) {
		return "", fmt.Errorf("invalid UTF-8 after deobfuscation")
	}
	return string(out), nil
}

func (h *HiAnime) StreamURL(anime scraper.Anime, episode scraper.Episode) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Step 1: Find the episode data-id by re-fetching episodes
	html, err := fetchEpisodeMaps(ctx, anime.ID)
	if err != nil {
		return "", "", err
	}

	entries := parseEpisodeEntries(html, anime.ID)

	var dataID string
	n := 0
	for _, e := range entries {
		n++
		if n == episode.Number {
			dataID = e.DataID
			break
		}
	}

	if dataID == "" {
		return "", "", fmt.Errorf("episode %d not found", episode.Number)
	}

	// Step 2: Fetch servers for this episode (JSON envelope with html field)
	serversURL := fmt.Sprintf(HiAnimeServers, dataID)
	resp2, err := httpclient.Request(ctx, "GET", serversURL, nil, nil)
	if err != nil {
		logger.Log.Error("hianime: servers request failed", "error", err)
		return "", "", err
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != 200 {
		return "", "", fmt.Errorf("servers status %d", resp2.StatusCode)
	}

	var serversResp struct {
		Status bool   `json:"status"`
		HTML   string `json:"html"`
	}
	if err := json.NewDecoder(io.LimitReader(resp2.Body, 10*1024*1024)).Decode(&serversResp); err != nil {
		logger.Log.Error("hianime: failed to decode servers response", "error", err)
		return "", "", err
	}

	serversStr := serversResp.HTML

	// Step 3: Find ZokoAnime server with matching type (sub/dub).
	// Split into per-server blocks so attributes can't leak across servers.
	var hash string
	for _, block := range srvBlkRe.FindAllString(serversStr, -1) {
		m := srvAtrRe.FindStringSubmatch(block)
		if m != nil && m[1] == episode.Mode && m[2] == "ZokoAnime" {
			hash = m[3]
			break
		}
	}
	if hash == "" {
		for _, block := range srvBlkRe.FindAllString(serversStr, -1) {
			m := srvAtrRe.FindStringSubmatch(block)
			if m != nil && m[2] == "ZokoAnime" {
				hash = m[3]
				break
			}
		}
	}
	if hash == "" {
		return "", "", fmt.Errorf("no ZokoAnime server found")
	}

	// Step 4: Decode plain base64 hash to get embed URL
	// (XOR deobfuscation is only for the window.__P blob, not the data-hash)
	embedBytes, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return "", "", fmt.Errorf("decode embed URL: %w", err)
	}
	embedURL := string(embedBytes)
	if !strings.HasPrefix(embedURL, "https://") {
		return "", "", fmt.Errorf("invalid embed URL scheme")
	}

	// Extract referer from embed URL origin
	refr := embedURL
	if idx := strings.Index(embedURL, "://"); idx != -1 {
		afterProto := embedURL[idx+3:]
		if slashIdx := strings.Index(afterProto, "/"); slashIdx != -1 {
			refr = embedURL[:idx+3+slashIdx+1]
		}
	}

	// Step 5: Fetch embed page and extract obfuscated blob
	resp3, err := httpclient.Request(ctx, "GET", embedURL, nil, nil)
	if err != nil {
		logger.Log.Error("hianime: embed page request failed", "error", err)
		return "", "", err
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != 200 {
		return "", "", fmt.Errorf("embed page status %d", resp3.StatusCode)
	}

	embedBody, err := io.ReadAll(io.LimitReader(resp3.Body, 10*1024*1024))
	if err != nil {
		return "", "", err
	}

	blobMatch := blobRe.FindStringSubmatch(string(embedBody))
	if blobMatch == nil {
		return "", "", fmt.Errorf("no __P blob in embed page")
	}

	// Step 6: Deobfuscate the blob to get JSON
	jsonStr, err := deobfuscateBlob(blobMatch[1])
	if err != nil {
		return "", "", fmt.Errorf("deobfuscate blob: %w", err)
	}

	// Step 7: Extract m3u8 URL from JSON
	m3u8Match := m3u8Re.FindStringSubmatch(jsonStr)
	if m3u8Match == nil {
		return "", "", fmt.Errorf("no m3u8 URL in deobfuscated JSON")
	}

	streamURL := strings.ReplaceAll(m3u8Match[1], `\/`, `/`)
	if !strings.HasPrefix(streamURL, "https://") {
		return "", "", fmt.Errorf("invalid stream URL scheme")
	}

	logger.Log.Debug("hianime: stream resolved", "anime", anime.Title, "episode", episode.Number, "mode", episode.Mode)
	return streamURL, refr, nil
}
