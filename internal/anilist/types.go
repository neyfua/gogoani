package anilist

import "fmt"


type MediaListStatus string

const (
	StatusWatching  MediaListStatus = "CURRENT"
	StatusCompleted MediaListStatus = "COMPLETED"
	StatusPaused    MediaListStatus = "PAUSED"
	StatusDropped   MediaListStatus = "DROPPED"
	StatusPlanning  MediaListStatus = "PLANNING"
	StatusRepeating MediaListStatus = "REPEATING"
)


func (s MediaListStatus) Label() string {
	switch s {
	case StatusWatching:
		return "watching"
	case StatusCompleted:
		return "completed"
	case StatusPaused:
		return "paused"
	case StatusDropped:
		return "dropped"
	case StatusPlanning:
		return "planning"
	case StatusRepeating:
		return "repeating"
	default:
		return string(s)
	}
}


func ParseStatus(label string) (MediaListStatus, bool) {
	switch label {
	case "watching", "current":
		return StatusWatching, true
	case "completed", "complete":
		return StatusCompleted, true
	case "paused":
		return StatusPaused, true
	case "dropped":
		return StatusDropped, true
	case "planning", "plan to watch":
		return StatusPlanning, true
	case "repeating":
		return StatusRepeating, true
	default:
		return "", false
	}
}


func AllStatuses() []MediaListStatus {
	return []MediaListStatus{
		StatusWatching,
		StatusCompleted,
		StatusPaused,
		StatusDropped,
		StatusPlanning,
		StatusRepeating,
	}
}




type ViewerResponse struct {
	Data struct {
		Viewer struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			Avatar      struct {
				Large  string `json:"large"`
				Medium string `json:"medium"`
			} `json:"avatar"`
		} `json:"Viewer"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}


type MediaSearchResponse struct {
	Data struct {
		Page struct {
			Media []struct {
				ID    int `json:"id"`
				Title struct {
					Romaji  string `json:"romaji"`
					English string `json:"english"`
					Native  string `json:"native"`
				} `json:"title"`
				CoverImage struct {
					Large  string `json:"large"`
					Medium string `json:"medium"`
				} `json:"coverImage"`
				Episodes *int   `json:"episodes"`
				Format   string `json:"format"`
				Status   string `json:"status"`
			} `json:"media"`
		} `json:"Page"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}


type MediaListResponse struct {
	Data struct {
		MediaListCollection struct {
			Lists []struct {
				Name         string `json:"name"`
				IsCustomList bool   `json:"isCustomList"`
				Entries      []struct {
					ID       int     `json:"id"`
					Status   string  `json:"status"`
					Progress int     `json:"progress"`
					Score    float64 `json:"score"`
					Media    struct {
						ID    int `json:"id"`
						Title struct {
							Romaji  string `json:"romaji"`
							English string `json:"english"`
							Native  string `json:"native"`
						} `json:"title"`
						CoverImage struct {
							Medium string `json:"medium"`
						} `json:"coverImage"`
						Episodes *int   `json:"episodes"`
						Format   string `json:"format"`
					} `json:"media"`
				} `json:"entries"`
			} `json:"lists"`
		} `json:"MediaListCollection"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}


type SaveMediaListResponse struct {
	Data struct {
		SaveMediaListEntry struct {
			ID       int    `json:"id"`
			Status   string `json:"status"`
			Progress int    `json:"progress"`
		} `json:"SaveMediaListEntry"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}


type GraphQLError struct {
	Message   string `json:"message"`
	Locations []struct {
		Line   int `json:"line"`
		Column int `json:"column"`
	} `json:"locations,omitempty"`
}


type AnimeEntry struct {
	ListEntryID int             `json:"listEntryId"`
	MediaID     int             `json:"mediaId"`
	Title       string          `json:"title"`
	Status      MediaListStatus `json:"status"`
	Progress    int             `json:"progress"`
	TotalEps    int             `json:"totalEps"`
	Score       float64         `json:"score"`
	Format      string          `json:"format"`
	CoverURL    string          `json:"coverUrl"`
}


func (e AnimeEntry) TitleDisplay() string {
	if e.Title == "" {
		return "Unknown"
	}
	return e.Title
}


func (e AnimeEntry) TotalEpisodesDisplay() string {
	if e.TotalEps == 0 {
		return "?"
	}
	return fmt.Sprintf("%d", e.TotalEps)
}
