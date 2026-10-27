package ui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/PhotoDotPi/noct/internal/log"
	"github.com/PhotoDotPi/noct/internal/player"
	"github.com/PhotoDotPi/noct/internal/search"
)

type searchResultsMsg struct {
	results []search.Result
	gen     uint64
}

type searchErrorMsg struct {
	err error
	gen uint64
}

type playFailedMsg struct {
	err error
}

type playerEventMsg struct {
	event player.Event
}

type playerEventsClosedMsg struct{}

func runSearch(ctx context.Context, provider search.Provider, query string, gen uint64) tea.Cmd {
	return func() tea.Msg {
		results, err := provider.Search(ctx, query)
		if err != nil {
			return searchErrorMsg{err: err, gen: gen}
		}
		return searchResultsMsg{results: results, gen: gen}
	}
}

func startSpinner() tea.Cmd {
	return spinner.Tick
}

func playCmd(p player.Player, source string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := p.Play(ctx, source); err != nil {
			return playFailedMsg{err: err}
		}
		return nil
	}
}

func watchPlayerEvents(p player.Player) tea.Cmd {
	events := p.Events()
	return func() tea.Msg {
		ev, ok := <-events
		if !ok {
			return playerEventsClosedMsg{}
		}
		return playerEventMsg{event: ev}
	}
}

func controlCmd(p player.Player, action string, fn func() error) tea.Cmd {
	return func() tea.Msg {
		if err := fn(); err != nil {
			log.Default().Warnf("%s failed: %v", action, err)
		}
		return nil
	}
}
