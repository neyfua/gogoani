package anilist

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	cacheDirName = "gogoani"
	tokenFile    = "anilist_token.json"
	listFile     = "anilist_list.json"
)

func CacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("anilist: get home dir: %w", err)
	}
	dir := filepath.Join(home, ".cache", cacheDirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("anilist: create cache dir: %w", err)
	}
	return dir, nil
}

type TokenData struct {
	Token string `json:"token"`
}

func SaveToken(token string) error {
	dir, err := CacheDir()
	if err != nil {
		return err
	}
	data := TokenData{Token: token}
	return writeJSON(filepath.Join(dir, tokenFile), data)
}

func LoadToken() (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	var data TokenData
	if err := readJSON(filepath.Join(dir, tokenFile), &data); err != nil {
		return "", err
	}
	if data.Token == "" {
		return "", fmt.Errorf("anilist: no token found, run 'gogoani anilist --auth' to authenticate")
	}
	return data.Token, nil
}

func RemoveToken() error {
	dir, err := CacheDir()
	if err != nil {
		return err
	}
	return os.Remove(filepath.Join(dir, tokenFile))
}

func SaveList(entries []AnimeEntry) error {
	dir, err := CacheDir()
	if err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, listFile), entries)
}

func LoadList() ([]AnimeEntry, error) {
	dir, err := CacheDir()
	if err != nil {
		return nil, err
	}
	var entries []AnimeEntry
	if err := readJSON(filepath.Join(dir, listFile), &entries); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return entries, nil
}

func ListPath() (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, listFile), nil
}

func UpdateEntryStatus(listEntryID, mediaID int, status MediaListStatus) error {
	entries, err := LoadList()
	if err != nil {
		return err
	}
	updated := false
	for i := range entries {
		if entries[i].ListEntryID == listEntryID && entries[i].MediaID == mediaID {
			entries[i].Status = status
			updated = true
			break
		}
	}
	if !updated {
		return fmt.Errorf("anilist: entry not found in cache")
	}
	return SaveList(entries)
}

func UpdateEntryProgress(listEntryID, mediaID int, progress int) error {
	entries, err := LoadList()
	if err != nil {
		return err
	}
	updated := false
	for i := range entries {
		if entries[i].ListEntryID == listEntryID && entries[i].MediaID == mediaID {
			entries[i].Progress = progress
			updated = true
			break
		}
	}
	if !updated {
		return fmt.Errorf("anilist: entry not found in cache")
	}
	return SaveList(entries)
}

func DeleteListEntryFromCache(listEntryID, mediaID int) error {
	entries, err := LoadList()
	if err != nil {
		return err
	}
	filtered := make([]AnimeEntry, 0, len(entries))
	for _, e := range entries {
		if e.ListEntryID != listEntryID || e.MediaID != mediaID {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == len(entries) {
		return fmt.Errorf("anilist: entry not found in cache")
	}
	return SaveList(filtered)
}

func HasToken() bool {
	dir, err := CacheDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, tokenFile))
	return err == nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("anilist: marshal json: %w", err)
	}

	dir := filepath.Dir(path)
	if fi, err := os.Lstat(dir); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("anilist: refusing to write through symlink")
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("anilist: write file %s: %w", path, err)
	}
	return os.Rename(tmpPath, path)
}

func readJSON(path string, v any) error {
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("anilist: refusing to read through symlink")
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("anilist: read file %s: %w", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("anilist: unmarshal json %s: %w", path, err)
	}
	return nil
}
