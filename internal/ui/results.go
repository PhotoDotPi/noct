package ui

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/PhotoDotPi/noct/internal/search"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) renderResults(l layoutMetrics) string {
	items := m.results.Items()
	h := l.resultsH()
	w := l.resultsW()

	if len(items) == 0 {
		return m.renderEmptyState(l)
	}

	rowCount := len(items)
	visibleRows := h / 2
	if visibleRows < 1 {
		visibleRows = 1
	}

	start := m.scroll
	if start > rowCount {
		start = 0
	}

	lines := make([]string, 0, visibleRows*2)
	for i := start; i < rowCount && len(lines) < visibleRows*2; i++ {
		it, ok := items[i].(Item)
		if !ok {
			break
		}
		line := m.renderResultRow(it.Result, i, w-1)
		if line != "" {
			lines = append(lines, strings.Split(line, "\n")...)
		}
	}

	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}

	// Apply animated selection highlight.
	selY := int(math.Round(m.selAnim.pos))
	if selY < 0 {
		selY = 0
	}
	if selY >= rowCount {
		selY = rowCount - 1
	}
	selStart := selY - start
	if selStart >= 0 && selStart < visibleRows {
		line1 := selStart * 2
		line2 := line1 + 1
		if line1 < len(lines) {
			lines[line1] = padVisible(sSelectedBg.Render(truncateNoTail(lines[line1], w-1)), w)
		}
		if line2 < len(lines) && lines[line2] != "" {
			lines[line2] = padVisible(sSelectedBg.Render(truncateNoTail(lines[line2], w-1)), w)
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderResultRow(r search.Result, idx int, maxW int) string {
	mark := " "
	if m.isPlayingURL(r.VideoURL) {
		if m.state.Playing {
			mark = "▶"
		} else {
			mark = "⏸"
		}
	}

	num := fmt.Sprintf("%2d", idx+1)
	dur := formatDuration(r.Duration)
	if dur == "0:00" {
		dur = "  -- "
	}

	title := r.Title
	if title == "" {
		title = "Untitled"
	}

	markStr := sResultMark.Render(mark)
	numStr := sResultNum.Render(num + ". ")
	durStr := sResultDur.Render(fmt.Sprintf("%-6s", dur))

	firstLine := " " + markStr + " " + numStr + durStr + " " + sResultTitle.Render(title)
	firstLine = truncateNoTail(firstLine, maxW)

	if r.Channel != "" && maxW > 40 {
		indent := "      "
		chanLine := truncateNoTail(indent+sResultChannel.Render(r.Channel), maxW)
		return firstLine + "\n" + chanLine
	}

	return firstLine
}

type resultDelegate struct{}

func (d resultDelegate) Height() int { return 1 }

func (d resultDelegate) Spacing() int { return 0 }

func (d resultDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d resultDelegate) Render(_ io.Writer, _ list.Model, _ int, _ list.Item) {
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
