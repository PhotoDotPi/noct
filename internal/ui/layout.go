package ui

import "github.com/charmbracelet/x/ansi"

type layoutMode int

const (
	layoutCompact layoutMode = iota
	layoutNormal
	layoutWide
	layoutUltra
)

type layoutMetrics struct {
	width        int
	height       int
	mode         layoutMode
	showHints    bool
	showStatus   bool
	showProgress bool
	showVolume   bool
	showQueue    bool
	showNowPlay  bool
	contentTop   int
	contentH     int
	queueW       int
	queueRows    int
}

func computeLayout(w, h int) layoutMetrics {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}

	l := layoutMetrics{width: w, height: h}

	switch {
	case w >= 120:
		l.mode = layoutUltra
	case w >= 90:
		l.mode = layoutWide
	case w >= 60:
		l.mode = layoutNormal
	default:
		l.mode = layoutCompact
	}

	l.showHints = l.mode >= layoutNormal && h >= 14
	l.showStatus = h >= 8
	l.showProgress = w >= 56 && h >= 5
	l.showVolume = w >= 78 && h >= 5
	l.showNowPlay = l.mode >= layoutCompact && h >= 5

	top := 0
	top++ // title
	if l.showHints {
		top++ // hints (single line)
	}
	top++ // rule
	if l.showStatus {
		top++ // status
	}
	top++ // search / content start

	bottom := 2 // rule + player

	contentH := h - top - bottom
	if contentH < 3 && l.showStatus {
		l.showStatus = false
		top--
		contentH = h - top - bottom
	}
	if contentH < 3 && l.showHints {
		l.showHints = false
		top--
		contentH = h - top - bottom
	}
	if contentH < 1 {
		contentH = 1
	}

	l.contentTop = top
	l.contentH = contentH

	l.showQueue = l.mode >= layoutWide && contentH >= 8
	if l.showQueue {
		l.queueW = w / 3
		if l.queueW < 20 {
			l.queueW = 20
		}
		if l.queueW > 36 {
			l.queueW = 36
		}
		l.queueRows = contentH - 2
		if l.queueRows < 2 {
			l.queueRows = 2
		}
	}

	return l
}

func (l layoutMetrics) resultsW() int {
	w := l.width
	if l.showQueue {
		w -= l.queueW + 1
	}
	return w
}

func (l layoutMetrics) resultsH() int {
	h := l.contentH
	if l.showQueue {
		h -= 2
	}
	if h < 1 {
		h = 1
	}
	return h
}

func (l layoutMetrics) progressW() int {
	w := l.width
	if l.showProgress {
		w -= 20
	}
	return w
}

func hrule(width int) string {
	if width < 1 {
		return ""
	}
	return sRule.Render(repeatStr("─", width))
}

func repeatStr(ch string, n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, 0, n*len(ch))
	for i := 0; i < n; i++ {
		b = append(b, ch...)
	}
	return string(b)
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(s, width, "…")
}

func truncateNoTail(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(s, width, "")
}

func ansiWidth(s string) int {
	return ansi.StringWidth(s)
}
