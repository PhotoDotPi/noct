package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type notifKind int

const (
	notifInfo notifKind = iota
	notifErr
)

const notifDuration = 1500 * time.Millisecond

type notifClearMsg struct{}

func (m *Model) pushNotif(text string) tea.Cmd {
	m.notifText = text
	m.notifKind = notifInfo
	return tea.Tick(notifDuration, func(time.Time) tea.Msg {
		return notifClearMsg{}
	})
}

func (m *Model) pushNotifErr(text string) tea.Cmd {
	m.notifText = text
	m.notifKind = notifErr
	return tea.Tick(notifDuration, func(time.Time) tea.Msg {
		return notifClearMsg{}
	})
}

func (m *Model) clearNotif() {
	m.notifText = ""
}

func (m Model) renderNotification(l layoutMetrics) string {
	n := m.notifText
	if n == "" {
		return ""
	}
	st := sNotif
	if m.notifKind == notifErr {
		st = sNotifErr
	}
	pad := l.width - ansiWidth(n) - 6
	if pad < 1 {
		pad = 1
	}
	return strings.Repeat(" ", pad) + st.Render(n)
}
