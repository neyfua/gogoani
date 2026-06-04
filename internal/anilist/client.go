package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/neyfua/gogoani/internal/httpclient"
	"github.com/neyfua/gogoani/internal/logger"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const (
	GraphQLAPI = "https://graphql.anilist.co"
)

type Client struct {
	token string
}

func NewClient(token string) *Client {
	return &Client{token: token}
}

func (c *Client) Token() string {
	return c.token
}

func (c *Client) Query(ctx context.Context, gql string, variables map[string]any, dst any) error {
	return c.request(ctx, gql, variables, dst)
}

func (c *Client) Mutation(ctx context.Context, gql string, variables map[string]any, dst any) error {
	return c.request(ctx, gql, variables, dst)
}

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type gqlResponseEnvelope struct {
	Errors []GraphQLError `json:"errors,omitempty"`
}

func (c *Client) request(ctx context.Context, gql string, variables map[string]any, dst any) error {
	if c.token == "" {
		return fmt.Errorf("anilist: no token configured")
	}

	reqBody := gqlRequest{
		Query:     gql,
		Variables: variables,
	}

	buf := httpclient.GetBuf()
	defer httpclient.PutBuf(buf)

	if err := json.NewEncoder(buf).Encode(reqBody); err != nil {
		return fmt.Errorf("anilist: marshal request: %w", err)
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + c.token,
	}

	reqBytes := append([]byte(nil), buf.Bytes()...)

	var respBody []byte
	var statusCode int
	var err error

	for attempt := range 4 {
		var resp *http.Response
		resp, err = httpclient.Request(ctx, "POST", GraphQLAPI, headers, bytes.NewReader(reqBytes))
		if err != nil {
			return fmt.Errorf("anilist: request failed: %w", err)
		}

		respBody, err = io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
		if closeErr := resp.Body.Close(); closeErr != nil {
			return fmt.Errorf("anilist: close response body: %w", closeErr)
		}
		if err != nil {
			return fmt.Errorf("anilist: read response: %w", err)
		}
		if len(respBody) == 10*1024*1024 {
			return fmt.Errorf("anilist: response too large")
		}

		statusCode = resp.StatusCode

		if len(respBody) == 0 {
			return fmt.Errorf("anilist: empty response body (status=%d)", statusCode)
		}

		logger.Log.Debug("anilist response", "status", statusCode, "body_len", len(respBody), "body_first_200", string(respBody[:min(200, len(respBody))]))

		if statusCode == 429 || isRateLimitError(respBody) {
			if attempt == 3 {
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(5*(attempt+1)) * time.Second):
			}
			continue
		}
		break
	}

	var envelope gqlResponseEnvelope
	if err := json.Unmarshal(respBody, &envelope); err == nil && len(envelope.Errors) > 0 {
		messages := make([]string, 0, len(envelope.Errors))
		for _, e := range envelope.Errors {
			messages = append(messages, e.Message)
		}
		return fmt.Errorf("anilist: graphql error: %v", messages)
	}

	if err := json.Unmarshal(respBody, dst); err != nil {
		return fmt.Errorf("anilist: unmarshal response: %w (status=%d, body: %.500s)", err, statusCode, string(respBody))
	}

	if err := checkGraphQLErrors(respBody, dst); err != nil {
		return err
	}

	return nil
}

func isRateLimitError(body []byte) bool {
	return strings.Contains(strings.ToLower(string(body)), "too many requests")
}

// checkGraphQLErrors inspects the raw response for GraphQL errors
// in typed response structs that embed []GraphQLError.
func checkGraphQLErrors(raw []byte, dst any) error {
	// Re-unmarshal just to check errors field on typed responses.
	// This is safe because the caller's dst already got populated above.
	type errorChecker struct {
		Errors []GraphQLError `json:"errors"`
	}
	var checker errorChecker
	if err := json.Unmarshal(raw, &checker); err != nil {
		return nil
	}
	if len(checker.Errors) > 0 {
		messages := make([]string, 0, len(checker.Errors))
		for _, e := range checker.Errors {
			messages = append(messages, e.Message)
		}
		return fmt.Errorf("anilist: graphql error: %v", messages)
	}
	return nil
}
