package ui

import (
	"fmt"
	"strings"
)

func (m Model) renderHeader(l layoutMetrics) string {
	var parts []string

	title := sAppTitle.Render("♪ NOCT")
	parts = append(parts, title)

	if m.state.Playing && l.showNowPlay {
		cur := m.queue.Current()
		if cur != nil {
			icon := sPlayerPlay.Render("▶")
			st := sPlayerTitle.Render(truncate(cur.Title, l.width-12))
			parts = append(parts, " ", icon, " ", st)
		}
	} else if m.mode == modeIdle && !m.searching && m.queue.Current() == nil {
		state := sAppState.Render("idle")
		parts = append(parts, " ", state)
	}

	left := strings.Join(parts, "")
	return left
}

func (m Model) renderHints(l layoutMetrics) string {
	if !l.showHints {
		return ""
	}

	var segs []string
	add := func(key, label string) {
		segs = append(segs, sHelpKey.Render(key)+" "+sHelpAction.Render(label))
	}

	switch {
	case m.mode == modeSearchInput:
		add("Esc", "cancel")
	case m.helpOpen:
		add("Esc", "close")
	case m.mode == modeResults:
		add("↑↓", "navigate")
		add("Enter", "play")
		if l.mode >= layoutWide {
			add("/", "search")
			add("Space", "pause")
		}
		add("q", "quit")
	default:
		add("/", "search")
		if l.mode >= layoutNormal {
			add("Space", "pause")
			add("←→", "seek")
			add("n/p", "next/prev")
			add("?", "help")
		}
		add("q", "quit")
	}

	joined := strings.Join(segs, "  "+sHintSep.Render("·")+"  ")
	return truncateNoTail(joined, l.width)
}

func (m Model) renderStatus(l layoutMetrics) string {
	if !l.showStatus {
		return ""
	}
	msg := m.statusText()
	if msg == "" {
		return ""
	}
	if m.lastError != nil {
		return sError.Render("▸ " + truncate(msg, l.width-3))
	}
	return sStatus.Render("▸ " + truncate(msg, l.width-3))
}

func (m Model) statusText() string {
	switch {
	case m.searching:
		return fmt.Sprintf("Searching for %q …", m.query)
	case m.lastError != nil:
		return userError(m.lastError)
	case m.mode == modeResults && len(m.resultsList) == 0 && m.query != "":
		return fmt.Sprintf("No results for %q", m.query)
	case m.mode == modeResults && len(m.resultsList) > 0:
		return fmt.Sprintf("%d results", len(m.resultsList))
	case m.mode == modeIdle && m.queue.Current() != nil:
		return ""
	case m.mode == modeIdle:
		return "Press / to search"
	}
	return ""
}

func (m Model) renderEmptyState(l layoutMetrics) string {
	var b strings.Builder

	b.WriteString("\n")

	switch {
	case m.lastError != nil:
		b.WriteString(sError.Render("⚠") + "  " + sError.Render(truncate(userError(m.lastError), l.width-6)) + "\n\n")
		b.WriteString(sHintKey.Render("/") + " " + sHelpAction.Render("search for something else"))
	case len(m.resultsList) == 0 && m.query != "":
		b.WriteString(sEmptyBig.Render("♪ No results") + "\n\n")
		b.WriteString(sEmpty.Render("Nothing found for") + " " + sEmptyBig.Render(truncate(m.query, l.width-20)) + "\n")
		b.WriteString(sEmpty.Render("Try a different search."))
	default:
		b.WriteString(sEmptyBig.Render("♪ Noct") + "\n\n")
		b.WriteString(sEmpty.Render("Search for something to play.") + "\n\n")
		b.WriteString(sHintKey.Render("/") + " " + sHelpAction.Render("search"))
	}

	return b.String()
}

func (m Model) renderSearchingState(l layoutMetrics) string {
	var b strings.Builder
	b.WriteString("\n\n")

	spinner := sPlayerPlay.Render(m.loading.View())
	b.WriteString(spinner + "  " + sStatus.Render("Searching…") + "\n")

	if m.query != "" {
		b.WriteString(sEmpty.Render("  "+truncate(m.query, l.width-4)) + "\n")
	}

	return b.String()
}
