package ui

import (
	"fmt"
)

func (m Model) renderPlayerBar(l layoutMetrics) string {
	cur := m.queue.Current()
	if cur == nil {
		return m.renderEmptyPlayer(l)
	}

	var left, right, meta string

	icon := "▶"
	if !m.state.Playing {
		icon = "⏸"
	}
	iconSt := sPlayerPlay
	if !m.state.Playing {
		iconSt = sPlayerPause
	}

	title := cur.Title
	if title == "" {
		title = "Unknown track"
	}

	left = iconSt.Render(icon) + " " + sPlayerTitle.Render(truncate(title, l.width/2))

	if l.showProgress {
		pos := formatPosition(m.state.Position)
		dur := formatPosition(m.state.Duration)
		bar := m.renderProgressBar(l.width)
		timeStr := sTimeLabel.Render(fmt.Sprintf(" %s / %s", pos, dur))
		meta = bar + timeStr
	}

	if l.showVolume {
		right = sVolLabel.Render(fmt.Sprintf(" %d%%", m.state.Volume))
	}

	if meta == "" && right == "" {
		return truncateNoTail(left, l.width)
	}

	totalW := l.width
	leftW := ansiWidth(left)
	rightW := ansiWidth(right)
	metaW := 0
	if meta != "" {
		metaW = ansiWidth(meta)
	}

	midW := totalW - leftW - rightW
	if midW < 8 {
		midW = 8
	}

	result := left
	if meta != "" {
		barW := midW - metaW
		if barW > 0 {
			result += repeatStr(" ", barW) + meta
		} else {
			result += " " + meta
		}
	} else {
		result += repeatStr(" ", midW)
	}

	if right != "" {
		pad := totalW - ansiWidth(result) - rightW
		if pad > 0 {
			result += repeatStr(" ", pad)
		}
		result += right
	}

	return truncateNoTail(result, totalW)
}

func (m Model) renderProgressBar(totalW int) string {
	if totalW < 20 {
		return ""
	}

	barW := totalW - 24
	if barW < 8 {
		barW = 8
	}
	if barW > 50 {
		barW = 50
	}

	pos := m.state.Position
	dur := m.state.Duration

	if dur <= 0 || pos <= 0 {
		return sProgressEmpty.Render("─" + repeatStr("─", barW) + "─")
	}

	ratio := pos / dur
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	filled := int(ratio * float64(barW))
	empty := barW - filled

	left := sProgressFill.Render(repeatStr("━", filled))
	head := sProgressHead.Render("●")
	right := sProgressEmpty.Render(repeatStr("─", empty))

	return left + head + right
}

func (m Model) renderEmptyPlayer(l layoutMetrics) string {
	label := sPlayerLabel.Render("♪  ")
	empty := sEmpty.Render("nothing playing — press / to search")
	return truncateNoTail(label+empty, l.width)
}

func formatPosition(seconds float64) string {
	total := int(seconds)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
