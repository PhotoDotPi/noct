package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderQueue(l layoutMetrics) string {
	if !l.showQueue {
		return ""
	}

	lines := make([]string, 0)

	cur := m.queue.Current()
	upcoming := m.queue.Upcoming()
	if cur == nil && len(upcoming) == 0 {
		return ""
	}

	lines = append(lines, sSectionHead.Render("UP NEXT"))
	lines = append(lines, "")

	if cur != nil {
		lines = append(lines, sQueueNow.Render("▸ Now"))
		lines = append(lines, sQueueNow.Render("  "+truncate(cur.Title, l.queueW-4)))
		lines = append(lines, "")
	}

	if len(upcoming) == 0 {
		lines = append(lines, sEmpty.Render("  Queue is empty"))
	} else {
		lines = append(lines, sSectionHead.Render(fmt.Sprintf("  %d queued", len(upcoming))))
		lines = append(lines, "")
		for i, t := range upcoming {
			if i >= l.queueRows-4 {
				lines = append(lines, sEmpty.Render(fmt.Sprintf("  …%d more", len(upcoming)-i)))
				break
			}
			num := fmt.Sprintf(" %d.", i+1)
			title := truncate(t.Title, l.queueW-8)
			lines = append(lines, sQueueItem.Render(num)+" "+sResultTitle.Render(title))
		}
	}

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		Width(l.queueW).
		Height(l.contentH).
		Render(content)

	return box
}
