package ui

import "github.com/charmbracelet/lipgloss"

type noctTheme struct {
	Bg        lipgloss.Color
	Surface   lipgloss.Color
	SurfaceHi lipgloss.Color
	Primary   lipgloss.Color
	Accent    lipgloss.Color
	Cyan      lipgloss.Color
	Muted     lipgloss.Color
	Dim       lipgloss.Color
	Err       lipgloss.Color
	Ok        lipgloss.Color
	Info      lipgloss.Color
	Selection lipgloss.Color
	Progress  lipgloss.Color
}

var defaultTheme = noctTheme{
	Bg:        lipgloss.Color("235"),
	Surface:   lipgloss.Color("237"),
	SurfaceHi: lipgloss.Color("240"),
	Primary:   lipgloss.Color("135"),
	Accent:    lipgloss.Color("205"),
	Cyan:      lipgloss.Color("44"),
	Muted:     lipgloss.Color("243"),
	Dim:       lipgloss.Color("240"),
	Err:       lipgloss.Color("196"),
	Ok:        lipgloss.Color("42"),
	Info:      lipgloss.Color("75"),
	Selection: lipgloss.Color("57"),
	Progress:  lipgloss.Color("135"),
}

var theme = defaultTheme

var (
	sAppTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary).
			MarginRight(1)

	sAppState = lipgloss.NewStyle().
			Foreground(theme.Muted).
			Faint(true)

	sSectionHead = lipgloss.NewStyle().
			Foreground(theme.Muted).
			Bold(true)

	sHintKey = lipgloss.NewStyle().
			Foreground(theme.Cyan).
			Bold(true)

	sHintSep = lipgloss.NewStyle().
			Foreground(theme.Dim)

	sSearchBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Muted).
			Padding(0, 1)

	sSearchBoxFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.Primary).
				Padding(0, 1)

	sSearchPrompt = lipgloss.NewStyle().
			Foreground(theme.Cyan).
			Bold(true)

	sSearchPlaceholder = lipgloss.NewStyle().
				Foreground(theme.Muted).
				Faint(true)

	sResultMark = lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true)

	sResultNum = lipgloss.NewStyle().
			Foreground(theme.Muted).
			Faint(true)

	sResultTitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	sResultChannel = lipgloss.NewStyle().
			Foreground(theme.Muted).
			Faint(true)

	sResultDur = lipgloss.NewStyle().
			Foreground(theme.Cyan)

	sSelectedBg = lipgloss.NewStyle().
			Background(theme.Selection).
			Foreground(lipgloss.Color("255"))

	sQueueItem = lipgloss.NewStyle().
			Foreground(theme.Muted).
			PaddingLeft(1)

	sQueueNow = lipgloss.NewStyle().
			Foreground(theme.Accent).
			Bold(true)

	sPlayerLabel = lipgloss.NewStyle().
			Foreground(theme.Muted).
			Faint(true)

	sPlayerTitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true)

	sPlayerPause = lipgloss.NewStyle().
			Foreground(theme.Accent).
			Bold(true)

	sPlayerPlay = lipgloss.NewStyle().
			Foreground(theme.Ok).
			Bold(true)

	sProgressFill = lipgloss.NewStyle().
			Foreground(theme.Progress)

	sProgressEmpty = lipgloss.NewStyle().
			Foreground(theme.Dim)

	sProgressHead = lipgloss.NewStyle().
			Foreground(theme.Accent).
			Bold(true)

	sTimeLabel = lipgloss.NewStyle().
			Foreground(theme.Muted)

	sVolLabel = lipgloss.NewStyle().
			Foreground(theme.Cyan)

	sError = lipgloss.NewStyle().
		Foreground(theme.Err).
		Bold(true)

	sStatus = lipgloss.NewStyle().
		Foreground(theme.Info)

	sEmpty = lipgloss.NewStyle().
		Foreground(theme.Muted).
		Faint(true)

	sEmptyBig = lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true)

	sRule = lipgloss.NewStyle().
		Foreground(theme.Dim)

	sHelpKey = lipgloss.NewStyle().
			Foreground(theme.Cyan).
			Bold(true)

	sHelpAction = lipgloss.NewStyle().
			Foreground(theme.Muted)

	sNotif = lipgloss.NewStyle().
		Foreground(theme.Ok).
		Bold(true)

	sNotifErr = lipgloss.NewStyle().
			Foreground(theme.Err).
			Bold(true)

	sHelpSection = lipgloss.NewStyle().
			Foreground(theme.Cyan).
			Bold(true).
			Underline(true)

	sHelpKeyBind = lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			Width(12)

	sHelpDesc = lipgloss.NewStyle().
			Foreground(theme.Muted)
)
