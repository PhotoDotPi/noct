package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderHelp(l layoutMetrics) string {
	contentW := 38
	overlayW := contentW + 6
	if l.width-4 < overlayW {
		overlayW = l.width - 4
		if overlayW < 30 {
			overlayW = 30
		}
		contentW = overlayW - 6
	}

	var b strings.Builder

	b.WriteString(sHelpSection.Render("NAVIGATION") + "\n")
	b.WriteString(helpRow("↑ / ↓", "navigate") + "\n")
	b.WriteString(helpRow("Enter", "play / select") + "\n")
	b.WriteString(helpRow("Esc", "back") + "\n")
	b.WriteString("\n")

	b.WriteString(sHelpSection.Render("PLAYBACK") + "\n")
	b.WriteString(helpRow("Space", "pause / resume") + "\n")
	b.WriteString(helpRow("n / p", "next / previous") + "\n")
	b.WriteString(helpRow("← / →", "seek ±5s") + "\n")
	b.WriteString(helpRow("+ / -", "volume") + "\n")
	b.WriteString("\n")

	b.WriteString(sHelpSection.Render("SEARCH") + "\n")
	b.WriteString(helpRow("/", "search") + "\n")
	b.WriteString(helpRow("Enter", "submit") + "\n")
	b.WriteString(helpRow("Esc", "cancel") + "\n")
	b.WriteString("\n")

	b.WriteString(sHelpSection.Render("APP") + "\n")
	b.WriteString(helpRow("?", "help") + "\n")
	b.WriteString(helpRow("q", "quit") + "\n")

	inner := sAppTitle.Render("NOCT HELP") + "\n\n" + b.String()

	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.Primary).
		Padding(1, 2).
		Width(overlayW).
		Render(inner)

	return box
}

func helpRow(key, desc string) string {
	return sHelpKeyBind.Width(12).Render(key) + sHelpDesc.Render(desc)
}
