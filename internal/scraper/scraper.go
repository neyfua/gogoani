package scraper

import "fmt"

// Anime holds basic anime metadata.
type Anime struct {
	ID    string
	Title string
	URL   string
}

func (a Anime) String() string { return a.Title }

// Episode holds episode metadata and stream sources.
type Episode struct {
	Number  int
	Title   string
	Mode    string   // "sub" or "dub"
	Sources []string // direct stream URLs
}

func (e Episode) String() string {
	if e.Title == "" {
		return fmt.Sprintf("Episode %d", e.Number)
	}
	return fmt.Sprintf("Episode %d – %s", e.Number, e.Title)
}

// Provider is the interface every anime source must implement.
type Provider interface {
	Search(query string) ([]Anime, error)
	Episodes(anime Anime, mode string) ([]Episode, error)
	StreamURL(anime Anime, episode Episode) (url string, referer string, err error)
}
