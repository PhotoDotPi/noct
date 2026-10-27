package search

import (
	"context"
	"errors"
	"net/url"
	"os/exec"
	"sync"

	"github.com/PhotoDotPi/noct/internal/dep"
)

func isContextCanceled(ctx context.Context) bool {
	return errors.Is(ctx.Err(), context.Canceled)
}

const ytdlpName = "yt-dlp"

type ytdlpProvider struct {
	mu       sync.Mutex
	path     string
	detected bool

	lookPath func(file string) (string, error)

	command func(ctx context.Context, name string, args ...string) *exec.Cmd
}

func NewProvider() Provider {
	return &ytdlpProvider{
		lookPath: exec.LookPath,
		command:  exec.CommandContext,
	}
}

func (p *ytdlpProvider) Search(ctx context.Context, query string) ([]Result, error) {
	path, err := p.resolve()
	if err != nil {
		return nil, err
	}

	searchURL := "https://www.youtube.com/results?search_query=" + url.QueryEscape(query)

	args := []string{
		"--flat-playlist",
		"--dump-single-json",
		"--playlist-end", "25",
		searchURL,
	}

	cmd := p.command(ctx, path, args...)
	out, err := cmd.Output()
	if err != nil {
		if isContextCanceled(ctx) {
			return nil, context.Canceled
		}
		return nil, &SearchExecutionError{Query: query, Err: err}
	}

	return parseResults(out)
}

func (p *ytdlpProvider) resolve() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.detected {
		return p.path, nil
	}

	path, err := p.lookPath(ytdlpName)
	if err != nil {
		return "", &dep.MissingDependencyError{Name: ytdlpName, Err: err}
	}

	p.path = path
	p.detected = true
	return path, nil
}
