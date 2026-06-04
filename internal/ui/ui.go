package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/neyfua/gogoani/internal/anilist"
	"github.com/neyfua/gogoani/internal/config"
	"github.com/neyfua/gogoani/internal/logger"
	"github.com/neyfua/gogoani/internal/player"
	"github.com/neyfua/gogoani/internal/scraper"
	"github.com/neyfua/gogoani/pkg/provider"
)

// PlayAnimeByTitle searches for an anime by title and starts the episode playback flow,
// bypassing the anime selection step if there's an exact title match or only one result.
func PlayAnimeByTitle(cfg *config.Config, title string, mode string) error {
	aa := provider.NewAllAnime()
	pl := player.New(cfg.Player)

	animes, err := aa.Search(title)
	if err != nil {
		return err
	}
	if len(animes) == 0 {
		return fmt.Errorf("no results found for %q", title)
	}

	var anime scraper.Anime
	var found bool
	if len(animes) == 1 {
		anime = animes[0]
		found = true
	} else {
		// Try exact match first
		for _, a := range animes {
			if strings.EqualFold(a.Title, title) {
				anime = a
				found = true
				break
			}
		}
		// Try substring match (e.g. AllAnime "86" matches AniList "86 EIGHTY-SIX")
		if !found {
			titleLower := strings.ToLower(title)
			for _, a := range animes {
				aLower := strings.ToLower(a.Title)
				if strings.Contains(titleLower, aLower) || strings.Contains(aLower, titleLower) {
					anime = a
					found = true
					break
				}
			}
		}
	}
	if !found {
		selected, err := selectAnime(animes, "Select anime: ")
		if err != nil {
			return err
		}
		anime = selected
	}

	return playEpisodes(cfg, aa, pl, anime, mode)
}

func Run(cfg *config.Config, query string, mode string) error {
	aa := provider.NewAllAnime()
	pl := player.New(cfg.Player)

	for {
		if query == "" {
			var err error
			query, err = promptInput("Search anime: ")
			if err != nil {
				return err
			}
		}

		animes, err := aa.Search(query)
		if err != nil {
			return err
		}

		if len(animes) == 0 {
			fmt.Println("No results found.")
			return nil
		}

		anime, err := selectAnime(animes, "Select anime: ")
		if err != nil {
			return err
		}

		if err := playEpisodes(cfg, aa, pl, anime, mode); err != nil {
			return err
		}

		query = ""
	}
}

func playEpisodes(cfg *config.Config, aa *provider.AllAnime, pl *player.Player, anime scraper.Anime, mode string) error {
	logger.Log.Debug("fetching episodes", "anime", anime.Title, "mode", mode)
	episodes, err := aa.Episodes(anime, mode)
	if err != nil {
		return err
	}

	if len(episodes) == 0 {
		fmt.Println("No episodes found.")
		return nil
	}

	// Initial episode selection
	episodeIdx, err := selectEpisode(episodes, "Select episode: ")
	if err != nil {
		return err
	}

	// Playback loop
	for {
		if episodeIdx < 0 || episodeIdx >= len(episodes) {
			return fmt.Errorf("episode index %d out of bounds (0-%d)", episodeIdx, len(episodes)-1)
		}
		episode := episodes[episodeIdx]

		logger.Log.Debug("fetching stream url", "anime", anime.Title, "episode", episode.Number)
		url, referer, err := aa.StreamURL(anime, episode)
		if err != nil {
			return err
		}

		if url == "" {
			return fmt.Errorf("no stream URL available for %s episode %d", anime.Title, episode.Number)
		}

		logger.Log.Debug("playing", "url", url, "referer", referer)
		if referer != "" {
			if err := pl.Start(url, "--referrer="+referer); err != nil {
				return err
			}
		} else {
			if err := pl.Start(url); err != nil {
				return err
			}
		}

		// Show menu immediately while mpv plays
		action, err := showEpisodeMenu(episodes, episodeIdx)
		if stopErr := pl.Stop(); stopErr != nil {
			logger.Log.Warn("player stop", "error", stopErr)
		}

		watchedEp := episode.Number

		// Sync the episode that was just watched (skip only if we replayed)
		if cfg.AutoSync && action != "replay" {
			syncAniList(cfg, anime.Title, watchedEp)
		}

		if err != nil {
			if strings.Contains(err.Error(), "selection cancelled") {
				return nil // silent exit on user cancellation
			}
			return err
		}

		switch action {
		case "prev":
			episodeIdx--
		case "next":
			episodeIdx++
		case "replay":
			// same index, continue
		case "select":
			newIdx, err := selectEpisode(episodes, "Select episode: ")
			if err != nil {
				return err
			}
			episodeIdx = newIdx
		case "watch_another":
			return nil
		case "quit":
			return nil
		}
	}
}

func selectAnime(animes []scraper.Anime, prompt string) (scraper.Anime, error) {
	anime, err := fzfSelect(animes, prompt)
	if err != nil {
		return scraper.Anime{}, err
	}
	return anime, nil
}

func selectEpisode(episodes []scraper.Episode, prompt string) (int, error) {
	episode, err := fzfSelect(episodes, prompt)
	if err != nil {
		return 0, err
	}
	for i, ep := range episodes {
		if ep.Number == episode.Number {
			return i, nil
		}
	}
	return 0, nil
}

type menuOption struct {
	label   string
	action  string
	enabled bool
}

func (m menuOption) String() string {
	return m.label
}

func showEpisodeMenu(episodes []scraper.Episode, currentIdx int) (string, error) {
	if currentIdx < 0 || currentIdx >= len(episodes) {
		return "", fmt.Errorf("episode index %d out of bounds", currentIdx)
	}
	opts := buildMenuOptions(episodes, currentIdx)
	currentEpisode := episodes[currentIdx]
	selected, err := fzfSelectMenu(opts, currentEpisode, len(episodes), "What next? ")
	if err != nil {
		return "", err
	}
	if !selected.enabled {
		return "", fmt.Errorf("option not available")
	}
	return selected.action, nil
}

func buildMenuOptions(episodes []scraper.Episode, currentIdx int) []menuOption {
	return []menuOption{
		{label: "Next Episode", action: "next", enabled: currentIdx < len(episodes)-1},
		{label: "Previous Episode", action: "prev", enabled: currentIdx > 0},
		{label: "Select Different Episode", action: "select", enabled: true},
		{label: "Replay Current Episode", action: "replay", enabled: true},
		{label: "Watch Another Anime", action: "watch_another", enabled: true},
		{label: "Quit", action: "quit", enabled: true},
	}
}

//nolint:gosec // G204: fzf subprocess with controlled input (no user data in args)
func fzfSelectMenu(items []menuOption, currentEpisode scraper.Episode, totalEpisodes int, prompt string) (menuOption, error) {
	cmd := exec.Command("fzf", "--ansi", "--prompt", prompt,
		"--preview", fmt.Sprintf("echo '\033[1;36mNow Playing: Episode %d/%d\033[0m'", currentEpisode.Number, totalEpisodes),
		"--preview-window", "down:1:nohidden,noborder,noinfo",
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return numericSelectMenu(items, currentEpisode, prompt)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return numericSelectMenu(items, currentEpisode, prompt)
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		logger.Log.Debug("fzf start failed, falling back to numeric selection", "error", err)
		return numericSelectMenu(items, currentEpisode, prompt)
	}

	go func() {
		defer stdin.Close()
		for _, item := range items {
			fmt.Fprintln(stdin, item.String())
		}
	}()

	output, err := io.ReadAll(stdout)
	if err != nil {
		return numericSelectMenu(items, currentEpisode, prompt)
	}

	if err := cmd.Wait(); err != nil {
		var zero menuOption
		return zero, fmt.Errorf("selection cancelled")
	}

	selected := strings.TrimSpace(string(output))
	if selected == "" {
		var zero menuOption
		return zero, fmt.Errorf("selection cancelled")
	}

	for _, item := range items {
		if item.String() == selected {
			return item, nil
		}
	}

	return numericSelectMenu(items, currentEpisode, prompt)
}

func numericSelectMenu(items []menuOption, currentEpisode scraper.Episode, prompt string) (menuOption, error) {
	for i, item := range items {
		fmt.Printf("[%d] %s\n", i+1, item.String())
	}
	fmt.Println()
	fmt.Printf("Now Playing: Episode %d\n", currentEpisode.Number)
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		var zero menuOption
		return zero, fmt.Errorf("selection cancelled")
	}
	if err := scanner.Err(); err != nil {
		var zero menuOption
		return zero, fmt.Errorf("selection failed: %w", err)
	}
	var idx int
	_, err := fmt.Sscanf(scanner.Text(), "%d", &idx)
	if err != nil || idx < 1 || idx > len(items) {
		var zero menuOption
		return zero, fmt.Errorf("invalid selection")
	}
	return items[idx-1], nil
}

func promptInput(prompt string) (string, error) {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("cancelled")
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return strings.TrimSpace(scanner.Text()), nil
}

//nolint:gosec // G204: fzf subprocess with controlled input (no user data in args)
func fzfSelect[T fmt.Stringer](items []T, prompt string) (T, error) {
	cmd := exec.Command("fzf", "--prompt", prompt, "--bind", "shift-up:page-up,shift-down:page-down")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return numericSelect(items, prompt)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return numericSelect(items, prompt)
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		logger.Log.Debug("fzf start failed, falling back to numeric selection", "error", err)
		return numericSelect(items, prompt)
	}

	go func() {
		defer stdin.Close()
		for _, item := range items {
			fmt.Fprintln(stdin, item.String())
		}
	}()

	output, err := io.ReadAll(stdout)
	if err != nil {
		return numericSelect(items, prompt)
	}

	if err := cmd.Wait(); err != nil {
		return items[0], fmt.Errorf("selection cancelled")
	}

	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return items[0], fmt.Errorf("selection cancelled")
	}

	for _, item := range items {
		if item.String() == selected {
			return item, nil
		}
	}

	return numericSelect(items, prompt)
}

func numericSelect[T fmt.Stringer](items []T, prompt string) (T, error) {
	for i, item := range items {
		fmt.Printf("[%d] %s\n", i+1, item.String())
	}
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return items[0], fmt.Errorf("selection cancelled")
	}
	if err := scanner.Err(); err != nil {
		return items[0], fmt.Errorf("selection failed: %w", err)
	}
	var idx int
	_, err := fmt.Sscanf(scanner.Text(), "%d", &idx)
	if err != nil || idx < 1 || idx > len(items) {
		return items[0], fmt.Errorf("invalid selection")
	}
	return items[idx-1], nil
}

func syncAniList(cfg *config.Config, title string, episodeNum int) {
	entries, err := anilist.LoadList()
	if err != nil {
		logger.Log.Debug("anilist: no cached list, skipping sync", "error", err)
		return
	}
	entry := anilist.MatchEntry(entries, title)
	if entry == nil {
		logger.Log.Debug("anilist: no matching entry found", "title", title)
		return
	}

	if episodeNum == entry.Progress {
		logger.Log.Debug("anilist: episode already synced, skipping", "title", entry.Title, "episode", episodeNum)
		return
	}

	client := anilist.NewClient(cfg.AniList.Token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := client.SaveProgress(ctx, entry.MediaID, episodeNum, entry.Status); err != nil {
		logger.Log.Warn("anilist: sync failed", "title", entry.Title, "error", err)
		return
	}
	if err := anilist.UpdateEntryProgress(entry.ListEntryID, entry.MediaID, episodeNum); err != nil {
		logger.Log.Warn("anilist: cache update failed", "error", err)
	}
	logger.Log.Info("anilist: synced", "title", entry.Title, "episode", episodeNum)
	fmt.Fprintf(os.Stderr, "✓ Synced %q episode %d to AniList\n", entry.Title, episodeNum)
}
