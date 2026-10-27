package search

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/PhotoDotPi/noct/internal/dep"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}

	joined := strings.Join(args, " ")
	switch {
	case strings.Contains(joined, "search_query=empty"):
		os.Stdout.WriteString(`{"_type":"playlist","entries":[]}`)
	case strings.Contains(joined, "search_query=garbage"):
		os.Stdout.WriteString(`{ not valid json `)
	default:
		os.Stdout.WriteString(string(sampleOutput()))
	}
}

func newTestProvider(t *testing.T) *ytdlpProvider {
	t.Helper()
	return &ytdlpProvider{
		lookPath: func(string) (string, error) { return "fake-yt-dlp", nil },
		detected: true,
		path:     "fake-yt-dlp",
		command: func(_ context.Context, _ string, args ...string) *exec.Cmd {
			all := append([]string{"-test.run=TestHelperProcess", "--"}, args...)
			cmd := exec.Command(os.Args[0], all...)
			cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
			return cmd
		},
	}
}

func TestSearchInvokesAndParses(t *testing.T) {
	p := newTestProvider(t)
	results, err := p.Search(context.Background(), "anything")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Title != "First Song" {
		t.Errorf("Title: got %q", results[0].Title)
	}
}

func TestSearchMissingDependency(t *testing.T) {
	p := &ytdlpProvider{lookPath: func(string) (string, error) {
		return "", errors.New("executable file not found in $PATH")
	}}

	_, err := p.Search(context.Background(), "anything")
	if !dep.IsMissing(err) {
		t.Fatalf("expected MissingDependencyError, got %v", err)
	}
}

func TestSearchEmptyResults(t *testing.T) {
	p := newTestProvider(t)
	_, err := p.Search(context.Background(), "empty")
	if !errors.Is(err, ErrNoResults) {
		t.Fatalf("expected ErrNoResults, got %v", err)
	}
}

func TestSearchMalformedOutput(t *testing.T) {
	p := newTestProvider(t)
	_, err := p.Search(context.Background(), "garbage")
	if err == nil {
		t.Fatal("expected an error for malformed output")
	}
}
