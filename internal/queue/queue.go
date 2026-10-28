package queue

import "github.com/PhotoDotPi/noct/internal/player"

type PrevAction int

const (
	PrevRestart PrevAction = iota
	PrevNavigate
	PrevNothing
)

type Queue struct {
	current   *player.Track
	upcoming  []*player.Track
	history   []*player.Track
	restarted bool
}

func New() *Queue {
	return &Queue{}
}

func (q *Queue) Current() *player.Track {
	return q.current
}

func (q *Queue) Upcoming() []*player.Track {
	return q.upcoming
}

func (q *Queue) HasUpcoming() bool {
	return len(q.upcoming) > 0
}

func (q *Queue) Len() int {
	return len(q.upcoming)
}

func (q *Queue) StartCurrent(t *player.Track) {
	q.current = t
	q.upcoming = nil
	q.history = nil
	q.restarted = false
}

func (q *Queue) AddUpcoming(t *player.Track) {
	q.upcoming = append(q.upcoming, t)
}

func (q *Queue) Next() (*player.Track, bool) {
	if len(q.upcoming) == 0 {
		return nil, false
	}
	q.restarted = false
	q.history = append(q.history, q.current)
	q.current = q.upcoming[0]
	q.upcoming = q.upcoming[1:]
	return q.current, true
}

func (q *Queue) PreviousPressed() PrevAction {
	if q.current == nil {
		return PrevNothing
	}
	if q.restarted {
		q.restarted = false
		if len(q.history) == 0 {
			return PrevNothing
		}
		q.upcoming = append([]*player.Track{q.current}, q.upcoming...)
		q.current = q.history[len(q.history)-1]
		q.history = q.history[:len(q.history)-1]
		return PrevNavigate
	}

	q.restarted = true
	return PrevRestart
}

func (q *Queue) HasPrevious() bool {
	return len(q.history) > 0
}

func (q *Queue) RemoveUpcoming(i int) bool {
	if i < 0 || i >= len(q.upcoming) {
		return false
	}
	q.upcoming = append(q.upcoming[:i], q.upcoming[i+1:]...)
	q.restarted = false
	return true
}

func (q *Queue) MoveUpcoming(i, delta int) bool {
	if delta == 0 {
		return false
	}
	j := i + delta
	if i < 0 || i >= len(q.upcoming) || j < 0 || j >= len(q.upcoming) {
		return false
	}
	q.upcoming[i], q.upcoming[j] = q.upcoming[j], q.upcoming[i]
	q.restarted = false
	return true
}

func (q *Queue) ClearUpcoming() {
	q.upcoming = nil
	q.restarted = false
}

func (q *Queue) ClearCurrent() {
	q.current = nil
	q.restarted = false
}
