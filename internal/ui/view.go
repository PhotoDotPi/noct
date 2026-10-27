package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/PhotoDotPi/noct/internal/dep"
	"github.com/PhotoDotPi/noct/internal/search"
)

func (m Model) View() string {
	l := computeLayout(m.width, m.height)

	var all []string

	if m.helpOpen {
		return m.renderHelpOverlay(l)
	}

	// ---- Header
	var header []string
	header = append(header, m.renderHeader(l))
	if h := m.renderHints(l); h != "" {
		header = append(header, h)
	}
	header = append(header, hrule(l.width))
	if s := m.renderStatus(l); s != "" {
		header = append(header, s)
	}
	all = append(all, strings.Join(header, "\n"))

	// ---- Content
	all = append(all, m.renderContent(l))

	// ---- Footer (rule + player + notification)
	var footer []string
	footer = append(footer, hrule(l.width))
	footer = append(footer, m.renderPlayerBar(l))
	if n := m.renderNotification(l); n != "" {
		footer = append(footer, n)
	}
	all = append(all, strings.Join(footer, "\n"))

	return strings.Join(all, "\n")
}

func (m Model) renderHelpOverlay(l layoutMetrics) string {
	help := m.renderHelp(l)
	lines := strings.Split(help, "\n")
	h := len(lines)
	if h > l.height-2 {
		h = l.height - 2
	}
	if h < 1 {
		h = 1
	}
	box := strings.Join(lines[:h], "\n")

	topPad := (l.height - h) / 2
	if topPad < 0 {
		topPad = 0
	}
	linesOut := make([]string, 0, l.height)
	for i := 0; i < topPad; i++ {
		linesOut = append(linesOut, "")
	}
	for _, ln := range strings.Split(box, "\n") {
		linesOut = append(linesOut, truncateNoTail(ln, l.width))
	}
	for len(linesOut) < l.height {
		linesOut = append(linesOut, "")
	}

	return strings.Join(linesOut, "\n")
}

func (m Model) renderContent(l layoutMetrics) string {
	var body string

	switch {
	case m.helpOpen:
		body = ""
	case m.searching && m.mode != modeSearchInput:
		body = m.renderSearchingView(l)
	case m.mode == modeSearchInput:
		body = m.renderSearchInputView(l)
	case m.mode == modeResults && len(m.resultsList) == 0:
		body = m.renderEmptyView(l)
	case m.mode == modeResults:
		body = m.renderList(l)
	case m.mode == modeIdle && m.queue.Current() == nil && !m.searching:
		body = m.renderEmptyState(l)
	default:
		body = m.renderIdleContent(l)
	}

	if l.showQueue {
		queue := m.renderQueue(l)
		if queue != "" {
			body = joinSideBySide(body, queue, l)
		}
	}

	return truncateHeight(clampBodyWidth(body, l.width), l.contentH)
}

func (m Model) renderIdleContent(l layoutMetrics) string {
	if m.queue.Current() != nil {
		return m.renderIdleNowPlaying(l)
	}
	return ""
}

func (m Model) renderIdleNowPlaying(l layoutMetrics) string {
	var b strings.Builder
	cur := m.queue.Current()
	if cur == nil {
		return ""
	}

	// Hero state indicator.
	stateIcon := sPlayerPlay.Render("▶")
	stateLabel := "PLAYING"
	stateStyle := sPlayerPlay
	if !m.state.Playing {
		stateIcon = sPlayerPause.Render("⏸")
		stateLabel = "PAUSED"
		stateStyle = sPlayerPause
	}

	b.WriteString("\n")
	b.WriteString(sSectionHead.Render("NOW PLAYING") + "\n\n")
	b.WriteString(stateStyle.Render(stateIcon+" "+stateLabel) + "\n\n")
	b.WriteString(sPlayerTitle.Render(cur.Title) + "\n")
	if cur.Channel != "" {
		b.WriteString(sResultChannel.Render(cur.Channel) + "\n")
	}

	if l.showProgress && l.width >= 60 && l.mode != layoutCompact {
		b.WriteString("\n")
		b.WriteString(m.renderProgressBar(l.width-8) + "\n")
		b.WriteString(sTimeLabel.Render(fmt.Sprintf("  %s / %s", formatPosition(m.state.Position), formatPosition(m.state.Duration))) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(sEmpty.Render("Press / to search for more music"))

	return b.String()
}

func joinSideBySide(left, right string, l layoutMetrics) string {
	if l.resultsW() < 20 {
		return left
	}

	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	maxH := l.contentH
	for len(leftLines) < maxH {
		leftLines = append(leftLines, "")
	}
	leftLines = leftLines[:maxH]
	for len(rightLines) < maxH {
		rightLines = append(rightLines, "")
	}
	rightLines = rightLines[:maxH]

	leftW := l.resultsW()
	rightW := l.queueW

	var out []string
	for i := 0; i < maxH; i++ {
		ll := padVisible(leftLines[i], leftW)
		rl := truncateNoTail(rightLines[i], rightW)
		out = append(out, ll+" "+rl)
	}
	return strings.Join(out, "\n")
}

func padVisible(s string, n int) string {
	pad := n - ansiWidth(s)
	if pad <= 0 {
		return truncateNoTail(s, n)
	}
	return s + repeatStr(" ", pad)
}

func clampBodyWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		if ansiWidth(ln) > width {
			lines[i] = truncateNoTail(ln, width)
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderSearchInputView(l layoutMetrics) string {
	return m.searchInput.view(l.width, true)
}

func (m Model) renderSearchingView(l layoutMetrics) string {
	return m.renderSearchingState(l)
}

func (m Model) renderEmptyView(l layoutMetrics) string {
	return m.renderEmptyState(l)
}

func (m Model) renderList(l layoutMetrics) string {
	return m.renderResults(l)
}

func truncateHeight(s string, n int) string {
	if n < 1 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n")
}

func userError(err error) string {
	if dep.IsMissing(err) {
		var depErr *dep.MissingDependencyError
		if errors.As(err, &depErr) && depErr.Name != "" {
			return fmt.Sprintf("%s is required but was not found. Please install it and make sure it is available in your PATH.",
				depErr.Name)
		}
		return "A required dependency is missing. Please install it and make sure it is available in your PATH."
	}
	if errors.Is(err, search.ErrNoResults) {
		return "No results found."
	}
	if errors.Is(err, errPlayerStopped) {
		return "Playback unavailable — MPV stopped. Press / to search for something else."
	}
	if errors.Is(err, errPlaybackProblem) {
		return "Couldn't play this track."
	}
	if err != nil {
		return fmt.Sprintf("Something went wrong. Please try again. (%v)", err)
	}
	return "Something went wrong. Please try again."
}
