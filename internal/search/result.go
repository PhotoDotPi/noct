package search

import "time"

type Result struct {
	ID       string
	Title    string
	Channel  string
	Duration time.Duration
	VideoURL string
}
