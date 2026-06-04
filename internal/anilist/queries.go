package anilist

import (
	"context"
	"slices"
	"strings"
)

type Viewer struct {
	ID   int
	Name string
}

func (c *Client) Viewer(ctx context.Context) (*Viewer, error) {
	query := `query {
  Viewer {
    id
    name
    avatar {
      large
      medium
    }
  }
}`

	var resp ViewerResponse
	if err := c.Query(ctx, query, nil, &resp); err != nil {
		return nil, err
	}
	return &Viewer{
		ID:   resp.Data.Viewer.ID,
		Name: resp.Data.Viewer.Name,
	}, nil
}

func (c *Client) SearchMedia(ctx context.Context, search string) ([]AnimeEntry, error) {
	query := `query ($search: String!) {
  Page(page: 1, perPage: 10) {
    media(search: $search, type: ANIME) {
      id
      title {
        romaji
        english
        native
      }
      coverImage {
        large
        medium
      }
      episodes
      format
      status
    }
  }
}`

	var resp MediaSearchResponse
	if err := c.Query(ctx, query, map[string]any{"search": search}, &resp); err != nil {
		return nil, err
	}

	entries := make([]AnimeEntry, 0, len(resp.Data.Page.Media))
	for _, media := range resp.Data.Page.Media {
		title := media.Title.English
		if title == "" {
			title = media.Title.Romaji
		}
		total := 0
		if media.Episodes != nil {
			total = *media.Episodes
		}
		entries = append(entries, AnimeEntry{
			MediaID:  media.ID,
			Title:    title,
			TotalEps: total,
			Format:   media.Format,
			CoverURL: media.CoverImage.Medium,
		})
	}
	return entries, nil
}

func (c *Client) FetchList(ctx context.Context) ([]AnimeEntry, error) {
	viewer, err := c.Viewer(ctx)
	if err != nil {
		return nil, err
	}
	return c.FetchListByUserID(ctx, viewer.ID)
}

func (c *Client) FetchListByUserID(ctx context.Context, userID int) ([]AnimeEntry, error) {
	query := `query ($userId: Int!) {
  MediaListCollection(type: ANIME, userId: $userId) {
    lists {
      name
      isCustomList
      entries {
        id
        status
        progress
        score
        media {
          id
          title {
            romaji
            english
            native
          }
          coverImage {
            medium
          }
          episodes
          format
        }
      }
    }
  }
}`

	var resp MediaListResponse
	if err := c.Query(ctx, query, map[string]any{"userId": userID}, &resp); err != nil {
		return nil, err
	}

	entries := make([]AnimeEntry, 0)
	for _, list := range resp.Data.MediaListCollection.Lists {
		for _, entry := range list.Entries {
			title := entry.Media.Title.English
			if title == "" {
				title = entry.Media.Title.Romaji
			}
			total := 0
			if entry.Media.Episodes != nil {
				total = *entry.Media.Episodes
			}
			entries = append(entries, AnimeEntry{
				ListEntryID: entry.ID,
				MediaID:     entry.Media.ID,
				Title:       title,
				Status:      MediaListStatus(entry.Status),
				Progress:    entry.Progress,
				TotalEps:    total,
				Score:       entry.Score,
				Format:      entry.Media.Format,
				CoverURL:    entry.Media.CoverImage.Medium,
			})
		}
	}

	slices.SortFunc(entries, func(a, b AnimeEntry) int {
		if a.Status != b.Status {
			return strings.Compare(a.Status.Label(), b.Status.Label())
		}
		return strings.Compare(a.Title, b.Title)
	})
	return entries, nil
}

func (c *Client) SyncList(ctx context.Context) ([]AnimeEntry, error) {
	entries, err := c.FetchList(ctx)
	if err != nil {
		return nil, err
	}
	if err := SaveList(entries); err != nil {
		return nil, err
	}
	return entries, nil
}
