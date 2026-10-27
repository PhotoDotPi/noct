package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"

	"github.com/PhotoDotPi/noct/internal/search"
)

type Item struct {
	Result search.Result
}

func (i Item) FilterValue() string { return i.Result.Title }

func newSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = sPlayerPlay
	return s
}

func formatDuration(d time.Duration) string {
	total := int(d.Seconds())
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
