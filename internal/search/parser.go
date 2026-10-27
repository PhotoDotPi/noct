package search

import (
	"encoding/json"
	"fmt"
	"time"
)

type searchResponse struct {
	Type    string  `json:"_type"`
	Entries []entry `json:"entries"`
}

type entry struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Channel  string `json:"channel"`
	Uploader string `json:"uploader"`
	Duration int64  `json:"duration"`
}

func parseResults(data []byte) ([]Result, error) {
	if len(data) == 0 {
		return nil, ErrNoResults
	}

	var resp searchResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode yt-dlp output: %w", err)
	}

	results := make([]Result, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		if e.ID == "" || e.Title == "" {
			continue
		}

		channel := e.Channel
		if channel == "" {
			channel = e.Uploader
		}

		results = append(results, Result{
			ID:       e.ID,
			Title:    e.Title,
			Channel:  channel,
			Duration: time.Duration(e.Duration) * time.Second,
			VideoURL: videoURL(e.ID, e.URL),
		})
	}

	if len(results) == 0 {
		return nil, ErrNoResults
	}

	return results, nil
}

func videoURL(id, url string) string {
	if id != "" {
		return "https://www.youtube.com/watch?v=" + id
	}
	return url
}
