package anilist

import (
	"context"
	"fmt"
)

type DeleteMediaListResponse struct {
	Data struct {
		DeleteMediaListEntry struct {
			Deleted bool `json:"deleted"`
		} `json:"DeleteMediaListEntry"`
	} `json:"data"`
}

func (c *Client) DeleteListEntry(ctx context.Context, listEntryID int) error {
	mutation := `mutation ($id: Int) {
  DeleteMediaListEntry(id: $id) {
    deleted
  }
}`
	vars := map[string]any{"id": listEntryID}
	var resp DeleteMediaListResponse
	if err := c.Mutation(ctx, mutation, vars, &resp); err != nil {
		return err
	}
	if !resp.Data.DeleteMediaListEntry.Deleted {
		return fmt.Errorf("anilist: delete returned false")
	}
	return nil
}

func (c *Client) SaveProgress(ctx context.Context, mediaID int, progress int, status MediaListStatus) (*AnimeEntry, error) {
	mutation := `mutation ($mediaId: Int, $progress: Int, $status: MediaListStatus) {
  SaveMediaListEntry(mediaId: $mediaId, progress: $progress, status: $status) {
    id
    status
    progress
  }
}`

	vars := map[string]any{
		"mediaId":  mediaID,
		"progress": progress,
		"status":   string(status),
	}

	var resp SaveMediaListResponse
	if err := c.Mutation(ctx, mutation, vars, &resp); err != nil {
		return nil, err
	}

	entry := resp.Data.SaveMediaListEntry
	return &AnimeEntry{
		ListEntryID: entry.ID,
		MediaID:     mediaID,
		Progress:    entry.Progress,
		Status:      MediaListStatus(entry.Status),
	}, nil
}
