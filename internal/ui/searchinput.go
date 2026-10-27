package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type searchInput struct {
	input textinput.Model
}

func newSearchInput() *searchInput {
	t := textinput.New()
	t.Prompt = ""
	t.Placeholder = "What do you want to play?"
	t.CharLimit = 200
	t.Width = 40
	t.TextStyle = sSearchPrompt
	t.PromptStyle = sSearchPrompt
	t.CursorStyle = sResultMark
	t.PlaceholderStyle = sSearchPlaceholder
	return &searchInput{input: t}
}

func (s *searchInput) focus() tea.Cmd {
	s.input.Focus()
	return textinput.Blink
}

func (s *searchInput) blur() {
	s.input.Blur()
}

func (s *searchInput) value() string {
	return s.input.Value()
}

func (s *searchInput) reset() {
	s.input.SetValue("")
	s.input.Blur()
}

func (s *searchInput) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return cmd
}

func (s *searchInput) view(width int, focused bool) string {
	// Reserve space for: prompt glyph (2) + box padding (4) + box borders (2).
	s.input.Width = width - 9
	if s.input.Width < 10 {
		s.input.Width = 10
	}

	inner := s.input.View()

	box := sSearchBox
	if focused {
		box = sSearchBoxFocused
	}

	prompt := sSearchPrompt.Render("⌕ ")
	return box.Width(width).Render(prompt + inner)
}
