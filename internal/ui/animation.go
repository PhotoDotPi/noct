package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/harmonica"
)

type animState struct {
	spring   harmonica.Spring
	pos      float64
	vel      float64
	target   float64
	active   bool
	lastTick time.Time
}

const (
	animFPS       = 60
	animStiffness = 5.0
	animDamping   = 1.0
)

func newAnim() animState {
	return animState{
		spring: harmonica.NewSpring(harmonica.FPS(animFPS), animStiffness, animDamping),
	}
}

type animTickMsg struct{}

func (a *animState) animate() tea.Cmd {
	if a.target == a.pos && !a.active {
		return nil
	}
	a.active = true
	return tea.Tick(time.Second/animFPS, func(time.Time) tea.Msg {
		return animTickMsg{}
	})
}

func (a *animState) setTarget(t float64) tea.Cmd {
	if t == a.target {
		return nil
	}
	a.target = t
	return a.animate()
}

func (a *animState) step() (float64, bool) {
	if !a.active || a.pos == a.target {
		a.active = false
		return a.pos, false
	}
	a.pos, a.vel = a.spring.Update(a.pos, a.vel, a.target)
	if abs(a.pos-a.target) < 0.05 {
		a.pos = a.target
		a.vel = 0
		a.active = false
		return a.pos, true
	}
	return a.pos, true
}

func (a *animState) restartCmd() tea.Cmd {
	return tea.Tick(time.Second/animFPS, func(time.Time) tea.Msg {
		return animTickMsg{}
	})
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
