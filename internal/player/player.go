package player

import "context"

type Track struct {
	Title   string
	Channel string
	Source  string
}

type State struct {
	Playing  bool
	Position float64
	Duration float64
	Volume   int
}

type EventType int

const (
	EventStateChanged EventType = iota
	EventTrackFinished
	EventError
	EventExited
)

type Event struct {
	Type  EventType
	State State
	Err   error
}

type Player interface {
	Play(ctx context.Context, source string) error
	TogglePause() error
	SetPaused(paused bool) error
	Seek(seconds float64) error
	SetVolume(percent int) error
	State() State
	Events() <-chan Event
	Stop() error
	Done() <-chan struct{}
}
