package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PhotoDotPi/noct/internal/dep"
	"github.com/PhotoDotPi/noct/internal/player"
	"github.com/PhotoDotPi/noct/internal/search"
)

// TestVisualDump renders the UI at many sizes/states and writes plain-text
// captures to an output directory for manual visual inspection.
// Run with: go test ./internal/ui -run TestVisualDump -v -args ... (NOCT_VISUAL_DIR=D:\noct\.visual)
func TestVisualDump(t *testing.T) {
	outDir := os.Getenv("NOCT_VISUAL_DIR")
	if outDir == "" {
		t.Skip("set NOCT_VISUAL_DIR to enable visual dump")
	}
	os.MkdirAll(outDir, 0o755)

	sizes := [][2]int{
		{40, 12}, {50, 15}, {60, 20}, {80, 24}, {100, 30}, {120, 40}, {160, 50},
	}

	type scenario struct {
		name  string
		apply func(*Model)
	}

	fixtures := []search.Result{
		{ID: "1", Title: "Let It Happen", Channel: "Tame Impala", Duration: 7*time.Minute + 47*time.Second, VideoURL: "http://x/1"},
		{ID: "2", Title: "The Less I Know the Better", Channel: "Tame Impala", Duration: 3*time.Minute + 36*time.Second, VideoURL: "http://x/2"},
		{ID: "3", Title: "Eventually", Channel: "Tame Impala", Duration: 5*time.Minute + 19*time.Second, VideoURL: "http://x/3"},
		{ID: "4", Title: "Borderline", Channel: "Tame Impala", Duration: 3*time.Minute + 59*time.Second, VideoURL: "http://x/4"},
	}

	scenarios := []scenario{
		{"idle", func(m *Model) {}},
		{"input", func(m *Model) { m.mode = modeSearchInput }},
		{"searching", func(m *Model) { m.searching = true; m.query = "jazz" }},
		{"empty", func(m *Model) { m.mode = modeResults; m.resultsList = nil; m.query = "rick astley" }},
		{"results", func(m *Model) {
			m.mode = modeResults
			m.query = "tame impala"
			m.resultsList = fixtures
			m.setListItems(fixtures)
		}},
		{"resultsQueue", func(m *Model) {
			m.queue.StartCurrent(&player.Track{Title: "Let It Happen", Channel: "Tame Impala", Source: "http://x/1"})
			m.queue.AddUpcoming(&player.Track{Title: "Eventually", Channel: "Tame Impala", Source: "http://x/3"})
			m.state = player.State{Playing: true, Position: 100, Duration: 467, Volume: 60}
			m.mode = modeResults
			m.query = "tame impala"
			m.resultsList = fixtures
			m.setListItems(fixtures)
		}},
		{"playing", func(m *Model) {
			m.queue.StartCurrent(&player.Track{Title: "Let It Happen", Channel: "Tame Impala", Source: "http://x/1"})
			m.queue.AddUpcoming(&player.Track{Title: "Eventually", Channel: "Tame Impala", Source: "http://x/3"})
			m.queue.AddUpcoming(&player.Track{Title: "Borderline", Channel: "Tame Impala", Source: "http://x/4"})
			m.state = player.State{Playing: true, Position: 151, Duration: 467, Volume: 75}
		}},
		{"paused", func(m *Model) {
			m.queue.StartCurrent(&player.Track{Title: "Let It Happen", Channel: "Tame Impala", Source: "http://x/1"})
			m.state = player.State{Playing: false, Position: 151, Duration: 467, Volume: 75}
		}},
		{"error", func(m *Model) {
			m.mode = modeResults
			m.lastError = &dep.MissingDependencyError{Name: "yt-dlp"}
		}},
		{"help", func(m *Model) { m.helpOpen = true }},
	}

	for _, sc := range scenarios {
		for _, sz := range sizes {
			w, h := sz[0], sz[1]
			m := New(fakeProvider{}, newFakePlayer())
			m.width = w
			m.height = h
			m.layout()
			sc.apply(&m)
			view := m.View()
			path := filepath.Join(outDir, fmt.Sprintf("%s-%dx%d.txt", sc.name, w, h))
			os.WriteFile(path, []byte(view), 0o644)
			t.Logf("wrote %s", path)
		}
	}
}
