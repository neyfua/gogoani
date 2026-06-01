package anilist

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)


func AuthWithToken(token string) (*Viewer, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("anilist: empty token")
	}

	client := NewClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	viewer, err := client.Viewer(ctx)
	if err != nil {
		return nil, err
	}
	if err := SaveToken(token); err != nil {
		return nil, err
	}
	return viewer, nil
}


func PromptAuth() (*Viewer, error) {
	fmt.Fprintln(os.Stderr, "Paste AniList API token:")
	fmt.Fprint(os.Stderr, "> ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return nil, fmt.Errorf("anilist: no token provided")
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return AuthWithToken(scanner.Text())
}
