package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/PhotoDotPi/noct/internal/log"
	"github.com/PhotoDotPi/noct/internal/player"
	"github.com/PhotoDotPi/noct/internal/search"
	"github.com/PhotoDotPi/noct/internal/ui"
)

func main() {
	if err := log.Setup(log.Info); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not initialize logging: %v\n", err)
	}

	app := ui.New(search.NewProvider(), player.New())

	p := tea.NewProgram(app, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	if m, ok := final.(ui.Model); ok {
		if err := m.Shutdown(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not stop player: %v\n", err)
		}
	}

	if l := log.Default(); l != nil {
		_ = l.Close()
	}

	if err != nil {
		os.Exit(1)
	}
}
