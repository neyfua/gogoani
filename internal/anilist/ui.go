package anilist

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ShowStatusList displays entries in a navigable fzf view.
// In interactive mode, selecting an entry opens a status picker
// that updates the anime on AniList and refreshes the local cache.
// watchFn, if non-nil, is called with the selected anime title when "watch" is chosen.
func ShowStatusList(entries []AnimeEntry, statusFilter string, watchFn func(string) error) error {
	filtered, err := FilterEntries(entries, statusFilter)
	if err != nil {
		return err
	}
	if len(filtered) == 0 {
		if statusFilter == "" {
			return fmt.Errorf("anilist: no cached anime found, run 'gogoani anilist --sync'")
		}
		return fmt.Errorf("anilist: no cached anime found with status %q", statusFilter)
	}

	if !isTerminal(os.Stdout.Fd()) {
		return printStatusList(filtered)
	}

	token, err := LoadToken()
	if err != nil {
		return printStatusList(filtered)
	}

	client := NewClient(token)

	for {
		selected, err := fzfSelect(filtered, "AniList > ")
		if err != nil || selected == "" {
			return nil
		}

		selectedEntries := resolveEntries(filtered, selected)
		if len(selectedEntries) == 0 {
			continue
		}

		action, err := pickAction()
		if err != nil || action == "" {
			continue
		}

		switch action {
		case "watch":
			for _, entry := range selectedEntries {
				if watchFn == nil {
					continue
				}
				if err := watchFn(entry.Title); err != nil {
					fmt.Fprintf(os.Stderr, "Error watching %q: %v\n", entry.TitleDisplay(), err)
				}
			}
		case "status":
			newStatus, err := pickStatus()
			if err != nil || newStatus == "" {
				continue
			}
			status, _ := ParseStatus(newStatus)
			for i, entry := range selectedEntries {
				if i > 0 {
					time.Sleep(time.Second)
				}
				if err := updateStatus(client, entry, entry.Progress, status); err != nil {
					fmt.Fprintf(os.Stderr, "Error updating %q: %v\n", entry.TitleDisplay(), err)
				}
			}
		case "progress":
			newProgress, err := pickProgress(selectedEntries)
			if err != nil || newProgress == -1 {
				continue
			}
			for i, entry := range selectedEntries {
				if i > 0 {
					time.Sleep(time.Second)
				}
				p := newProgress
				if p == -2 {
					p = int(entry.TotalEps)
				}
				if err := updateProgress(client, entry, p); err != nil {
					fmt.Fprintf(os.Stderr, "Error updating %q: %v\n", entry.TitleDisplay(), err)
				}
			}
		case "delete":
			ok, err := confirmDelete(selectedEntries)
			if err != nil || !ok {
				continue
			}
			for i, entry := range selectedEntries {
				if i > 0 {
					time.Sleep(time.Second)
				}
				if err := deleteEntry(client, entry); err != nil {
					fmt.Fprintf(os.Stderr, "Error deleting %q: %v\n", entry.TitleDisplay(), err)
				}
			}
		}

		entries, err = LoadList()
		if err != nil {
			return err
		}
		filtered, err = FilterEntries(entries, statusFilter)
		if err != nil {
			return err
		}
	}
}

func updateStatus(client *Client, entry *AnimeEntry, progress int, status MediaListStatus) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if _, err := client.SaveProgress(ctx, entry.MediaID, progress, status); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	if err := UpdateEntryStatus(entry.ListEntryID, entry.MediaID, status); err != nil {
		return fmt.Errorf("update cache: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Updated %q to %s\n", entry.TitleDisplay(), status.Label())
	return nil
}

func updateProgress(client *Client, entry *AnimeEntry, progress int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if _, err := client.SaveProgress(ctx, entry.MediaID, progress, entry.Status); err != nil {
		return fmt.Errorf("update progress: %w", err)
	}

	if err := UpdateEntryProgress(entry.ListEntryID, entry.MediaID, progress); err != nil {
		return fmt.Errorf("update cache: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Updated %q progress to %d\n", entry.TitleDisplay(), progress)
	return nil
}

func deleteEntry(client *Client, entry *AnimeEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := client.DeleteListEntry(ctx, entry.ListEntryID); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if err := DeleteListEntryFromCache(entry.ListEntryID, entry.MediaID); err != nil {
		return fmt.Errorf("update cache: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Deleted %q\n", entry.TitleDisplay())
	return nil
}

//nolint:gosec // G204: fzf subprocess with controlled input (no user data in args)
func fzfSelect(entries []AnimeEntry, prompt string) (string, error) {
	cmd := exec.Command("fzf", "--ansi", "--no-sort", "--multi", "--prompt", prompt, "--header", "esc/ctrl+c back | tab multi-select", "--bind", "shift-up:page-up,shift-down:page-down")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return "", err
	}

	go func() {
		defer stdin.Close()
		for _, entry := range entries {
			fmt.Fprintln(stdin, formatEntry(entry))
		}
	}()

	_ = cmd.Wait()
	return strings.TrimSpace(out.String()), nil
}

func pickAction() (string, error) {
	actions := []string{"watch", "status", "progress", "delete"}
	cmd := exec.Command("fzf", "--ansi", "--no-sort", "--prompt", "Action > ", "--header", "esc/ctrl+c back")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return "", err
	}
	go func() {
		defer stdin.Close()
		for _, a := range actions {
			fmt.Fprintln(stdin, a)
		}
	}()
	_ = cmd.Wait()
	return strings.TrimSpace(out.String()), nil
}

func confirmDelete(entries []*AnimeEntry) (bool, error) {
	scanner := bufio.NewScanner(os.Stdin)
	if len(entries) == 1 {
		fmt.Fprintf(os.Stderr, "Delete %q from AniList? [y/N]: ", entries[0].TitleDisplay())
	} else {
		fmt.Fprintf(os.Stderr, "Delete %d entries from AniList? [y/N]: ", len(entries))
	}
	if !scanner.Scan() {
		return false, fmt.Errorf("no input")
	}
	input := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return input == "y" || input == "yes", nil
}

func pickProgress(entries []*AnimeEntry) (int, error) {
	scanner := bufio.NewScanner(os.Stdin)
	if len(entries) == 1 {
		e := entries[0]
		fmt.Fprintf(os.Stderr, "Enter episodes watched for %q (current: %d, max: %d, or 'max'): ",
			e.TitleDisplay(), e.Progress, e.TotalEps)
	} else {
		fmt.Fprintf(os.Stderr, "Enter episodes watched for %d entries (or 'max'): ", len(entries))
	}
	if !scanner.Scan() {
		return -1, fmt.Errorf("no input")
	}
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return -1, nil
	}
	if strings.ToLower(input) == "max" {
		return -2, nil
	}
	n, err := strconv.Atoi(input)
	if err != nil {
		return -1, fmt.Errorf("invalid number: %s", input)
	}
	if n < 0 {
		return -1, fmt.Errorf("progress cannot be negative")
	}
	return n, nil
}

func resolveEntries(entries []AnimeEntry, selected string) []*AnimeEntry {
	lines := strings.Split(selected, "\n")
	var result []*AnimeEntry
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entry := findEntryByTitle(entries, line)
		if entry != nil {
			result = append(result, entry)
		}
	}
	return result
}

func pickStatus() (string, error) {
	statuses := []string{"watching", "completed", "paused", "dropped", "planning", "repeating"}

	cmd := exec.Command("fzf", "--ansi", "--no-sort", "--prompt", "Change status > ", "--header", "esc/ctrl+c back")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return "", err
	}

	go func() {
		defer stdin.Close()
		for _, s := range statuses {
			fmt.Fprintln(stdin, s)
		}
	}()

	_ = cmd.Wait()
	return strings.TrimSpace(out.String()), nil
}

func findEntryByTitle(entries []AnimeEntry, formatted string) *AnimeEntry {
	for i := range entries {
		if formatEntry(entries[i]) == formatted {
			return &entries[i]
		}
	}
	return nil
}

func isTerminal(fd uintptr) bool {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func FilterEntries(entries []AnimeEntry, statusFilter string) ([]AnimeEntry, error) {
	statusFilter = strings.TrimSpace(strings.ToLower(statusFilter))
	if statusFilter == "" {
		return entries, nil
	}
	status, ok := ParseStatus(statusFilter)
	if !ok {
		return nil, fmt.Errorf("anilist: invalid status %q", statusFilter)
	}

	filtered := make([]AnimeEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Status == status {
			filtered = append(filtered, entry)
		}
	}
	return filtered, nil
}

func formatEntry(entry AnimeEntry) string {
	return fmt.Sprintf("%-12s %3d/%-3s  %s", entry.Status.Label(), entry.Progress, entry.TotalEpisodesDisplay(), entry.TitleDisplay())
}

func printStatusList(entries []AnimeEntry) error {
	for _, entry := range entries {
		fmt.Println(formatEntry(entry))
	}
	return nil
}
