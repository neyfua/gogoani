package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/neyfua/gogoani/internal/httpclient"
	"golang.org/x/term"
)

const (
	OAuthAuthorizationURL = "https://anilist.co/api/v2/oauth/authorize"
	OAuthTokenURL         = "https://anilist.co/api/v2/oauth/token"
)

func PromptOAuthClientID() (string, error) {
	fmt.Fprint(os.Stderr, "Paste AniList Client ID: ")
	byteID, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("anilist: read client ID: %w", err)
	}
	return strings.TrimSpace(string(byteID)), nil
}

func PromptOAuthClientSecret() (string, error) {
	fmt.Fprint(os.Stderr, "Paste AniList Client Secret: ")
	byteSecret, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("anilist: read client secret: %w", err)
	}
	return strings.TrimSpace(string(byteSecret)), nil
}

func GenerateAuthorizationURL(clientID string) string {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", "https://anilist.co/api/v2/oauth/pin")
	return OAuthAuthorizationURL + "?" + params.Encode()
}

func PromptForAuthCode() (string, error) {
	fmt.Fprint(os.Stderr, "Enter the authorization code: ")
	byteCode, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("anilist: read auth code: %w", err)
	}
	return strings.TrimSpace(string(byteCode)), nil
}

func ExchangePINForToken(ctx context.Context, clientID, clientSecret, code string) (string, error) {
	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("client_id", clientID)
	params.Set("client_secret", clientSecret)
	params.Set("code", code)
	params.Set("redirect_uri", "https://anilist.co/api/v2/oauth/pin")

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	resp, err := httpclient.Request(ctx, "POST", OAuthTokenURL, headers, bytes.NewReader([]byte(params.Encode())))
	if err != nil {
		return "", fmt.Errorf("anilist: token request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("anilist: read response body: %w", err)
	}

	type errorResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	var errResp errorResp
	if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error != "" {
		return "", fmt.Errorf("anilist: token exchange failed: %s (status=%d)", errResp.Message, resp.StatusCode)
	}

	var tokenResp OAuthTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("anilist: unmarshal token response: %w (status=%d, body: %.500s)", err, resp.StatusCode, string(respBody))
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("anilist: no access token in response: %s", string(respBody))
	}

	return tokenResp.AccessToken, nil
}

func AuthWithPIN(clientID, clientSecret string) (*Viewer, error) {
	fmt.Fprintf(os.Stderr, "Visit this URL to authorize:\n%s\n\n", GenerateAuthorizationURL(clientID))
	fmt.Fprintln(os.Stderr, "After authorizing, the page will show the authorization code")
	fmt.Fprintln(os.Stderr, "Copy the code from the page and paste it below")

	authCode, err := PromptForAuthCode()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	token, err := ExchangePINForToken(ctx, clientID, clientSecret, authCode)
	if err != nil {
		return nil, err
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("anilist: empty token")
	}

	client := NewClient(token)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()

	viewer, err := client.Viewer(ctx2)
	if err != nil {
		return nil, err
	}
	if err := SaveToken(token); err != nil {
		return nil, err
	}
	return viewer, nil
}

func AuthInteractive() (*Viewer, error) {
	clientID, err := PromptOAuthClientID()
	if err != nil {
		return nil, err
	}

	clientSecret, err := PromptOAuthClientSecret()
	if err != nil {
		return nil, err
	}

	return AuthWithPIN(clientID, clientSecret)
}
