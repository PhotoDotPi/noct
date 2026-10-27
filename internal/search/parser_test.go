package search

import (
	"errors"
	"testing"
	"time"
)

func sampleOutput() []byte {
	return []byte(`{
	  "id": "lo-fi beats",
	  "title": "lo-fi beats",
	  "_type": "playlist",
	  "entries": [
	    {"_type":"url","ie_key":"Youtube","id":"abc123","url":"https://www.youtube.com/watch?v=abc123","title":"First Song","channel":"Channel A","uploader":"Channel A","duration":3674},
	    {"_type":"url","ie_key":"Youtube","id":"def456","url":"https://www.youtube.com/watch?v=def456","title":"Second Song","channel":"Channel B","uploader":"Someone Else","duration":9000},
	    {"_type":"url","ie_key":"Youtube","id":"ghi789","url":"https://www.youtube.com/watch?v=ghi789","title":"Third Song","uploader":"Only Uploader","duration":0}
	  ]
	}`)
}

func TestParseResults(t *testing.T) {
	results, err := parseResults(sampleOutput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	first := results[0]
	if first.ID != "abc123" {
		t.Errorf("ID: got %q want %q", first.ID, "abc123")
	}
	if first.Title != "First Song" {
		t.Errorf("Title: got %q want %q", first.Title, "First Song")
	}
	if first.Channel != "Channel A" {
		t.Errorf("Channel: got %q want %q", first.Channel, "Channel A")
	}
	if first.Duration != 3674*time.Second {
		t.Errorf("Duration: got %v want %v", first.Duration, 3674*time.Second)
	}
	if first.VideoURL != "https://www.youtube.com/watch?v=abc123" {
		t.Errorf("VideoURL: got %q", first.VideoURL)
	}
}

func TestParseResultsFallsBackToUploader(t *testing.T) {
	results, err := parseResults(sampleOutput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	third := results[2]
	if third.Channel != "Only Uploader" {
		t.Errorf("Channel fallback: got %q want %q", third.Channel, "Only Uploader")
	}
	if third.Duration != 0 {
		t.Errorf("Duration: got %v want 0", third.Duration)
	}
}

func TestParseResultsIgnoresIncompleteEntries(t *testing.T) {
	data := []byte(`{
	  "_type": "playlist",
	  "entries": [
	    {"_type":"url","id":"","title":"No ID","channel":"X","duration":10},
	    {"_type":"url","id":"good1","title":"","channel":"X","duration":10},
	    {"_type":"url","id":"good2","title":"Real One","channel":"X","duration":10}
	  ]
	}`)

	results, err := parseResults(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "good2" {
		t.Errorf("ID: got %q want %q", results[0].ID, "good2")
	}
}

func TestParseResultsEmptyOutput(t *testing.T) {
	_, err := parseResults(nil)
	if !errors.Is(err, ErrNoResults) {
		t.Errorf("expected ErrNoResults, got %v", err)
	}
}

func TestParseResultsEmptyEntries(t *testing.T) {
	data := []byte(`{"_type":"playlist","entries":[]}`)
	_, err := parseResults(data)
	if !errors.Is(err, ErrNoResults) {
		t.Errorf("expected ErrNoResults, got %v", err)
	}
}

func TestParseResultsMalformedJSON(t *testing.T) {
	data := []byte(`{ this is not json `)
	if _, err := parseResults(data); err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
}

func TestParseResultsNonEmptyGarbage(t *testing.T) {
	data := []byte(`just some plain text`)
	if _, err := parseResults(data); err == nil {
		t.Fatal("expected an error for non-JSON output")
	}
}

func TestVideoURLPrefersID(t *testing.T) {
	if got := videoURL("vid123", ""); got != "https://www.youtube.com/watch?v=vid123" {
		t.Errorf("got %q", got)
	}
	if got := videoURL("", "https://example.com/x"); got != "https://example.com/x" {
		t.Errorf("got %q", got)
	}
}
