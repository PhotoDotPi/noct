package queue

import (
	"testing"

	"github.com/PhotoDotPi/noct/internal/player"
)

func tr(title string) *player.Track {
	return &player.Track{Title: title, Channel: "chan", Source: "https://youtube.com/watch?v=" + title}
}

func TestEmptyQueue(t *testing.T) {
	q := New()
	if q.Current() != nil {
		t.Fatalf("expected nil current, got %v", q.Current())
	}
	if q.HasUpcoming() {
		t.Fatal("expected no upcoming on empty queue")
	}
	if q.Len() != 0 {
		t.Fatalf("expected Len 0, got %d", q.Len())
	}
	if q.HasPrevious() {
		t.Fatal("expected no previous on empty queue")
	}
}

func TestStartCurrentPlaysImmediately(t *testing.T) {
	q := New()
	q.StartCurrent(tr("A"))
	if got := q.Current(); got == nil || got.Title != "A" {
		t.Fatalf("expected current A, got %v", got)
	}
	if q.HasUpcoming() {
		t.Fatal("expected no upcoming after start current")
	}
}

func TestAddUpcomingWhilePlaying(t *testing.T) {
	q := New()
	q.StartCurrent(tr("A"))
	q.AddUpcoming(tr("B"))
	q.AddUpcoming(tr("C"))

	if got := q.Current(); got.Title != "A" {
		t.Fatalf("current should remain A, got %v", got)
	}
	if q.Len() != 2 {
		t.Fatalf("expected 2 upcoming, got %d", q.Len())
	}
	up := q.Upcoming()
	if up[0].Title != "B" || up[1].Title != "C" {
		t.Fatalf("upcoming order wrong: %v %v", up[0].Title, up[1].Title)
	}
}

func TestNextAdvancesAndRequeuesToHistory(t *testing.T) {
	q := New()
	q.StartCurrent(tr("A"))
	q.AddUpcoming(tr("B"))
	q.AddUpcoming(tr("C"))

	next, ok := q.Next()
	if !ok {
		t.Fatal("expected ok")
	}
	if next.Title != "B" {
		t.Fatalf("expected next B, got %v", next)
	}
	if q.Current().Title != "B" {
		t.Fatalf("current should be B, got %v", q.Current())
	}
	if q.Len() != 1 {
		t.Fatalf("after next, expected 1 upcoming, got %d", q.Len())
	}
	if q.Upcoming()[0].Title != "C" {
		t.Fatalf("expected upcoming C, got %v", q.Upcoming()[0])
	}
	if !q.HasPrevious() {
		t.Fatal("expected previous A in history")
	}
}

func TestNextWithEmptyUpcoming(t *testing.T) {
	q := New()
	q.StartCurrent(tr("A"))
	if _, ok := q.Next(); ok {
		t.Fatal("expected Next to report not ok with no upcoming")
	}
	if _, ok := q.Next(); ok {
		t.Fatal("expected repeated Next to be safe and not ok")
	}
}

func TestPreviousFirstPressRestarts(t *testing.T) {
	q := New()
	q.StartCurrent(tr("A"))
	if action := q.PreviousPressed(); action != PrevRestart {
		t.Fatalf("expected PrevRestart, got %d", action)
	}
	if q.Current().Title != "A" {
		t.Fatalf("restart must not change current, got %v", q.Current())
	}
}

func TestPreviousSecondPressNavigates(t *testing.T) {
	q := New()
	q.StartCurrent(tr("A"))
	q.AddUpcoming(tr("B"))
	q.Next()

	if action := q.PreviousPressed(); action != PrevRestart {
		t.Fatalf("expected PrevRestart, got %d", action)
	}
	if action := q.PreviousPressed(); action != PrevNavigate {
		t.Fatalf("expected PrevNavigate, got %d", action)
	}
	if q.Current().Title != "A" {
		t.Fatalf("expected current A after previous navigate, got %v", q.Current())
	}
	if !q.HasUpcoming() || q.Upcoming()[0].Title != "B" {
		t.Fatalf("expected B back at front of upcoming, got %v", q.Upcoming())
	}
	if action := q.PreviousPressed(); action != PrevRestart {
		t.Fatalf("expected PrevRestart after navigation reset, got %d", action)
	}
}

func TestPreviousNoPreviousAfterRestart(t *testing.T) {
	q := New()
	q.StartCurrent(tr("A"))
	if action := q.PreviousPressed(); action != PrevRestart {
		t.Fatalf("expected PrevRestart, got %d", action)
	}
	if action := q.PreviousPressed(); action != PrevNothing {
		t.Fatalf("expected PrevNothing, got %d", action)
	}
	if q.Current().Title != "A" {
		t.Fatalf("current unchanged, got %v", q.Current())
	}
}

func TestPreviousFlagResetsOnNext(t *testing.T) {
	q := New()
	q.StartCurrent(tr("A"))
	q.AddUpcoming(tr("B"))

	if action := q.PreviousPressed(); action != PrevRestart {
		t.Fatalf("expected PrevRestart, got %d", action)
	}
	q.Next()
	if q.Current().Title != "B" {
		t.Fatalf("expected current B, got %v", q.Current())
	}
	if action := q.PreviousPressed(); action != PrevRestart {
		t.Fatalf("expected PrevRestart after Next reset, got %d", action)
	}
}

func TestStartCurrentDiscardsQueue(t *testing.T) {
	q := New()
	q.StartCurrent(tr("A"))
	q.AddUpcoming(tr("B"))
	q.StartCurrent(tr("C"))
	if q.Len() != 0 || q.HasPrevious() {
		t.Fatal("StartCurrent should discard upcoming and history")
	}
	if q.Current().Title != "C" {
		t.Fatalf("expected current C, got %v", q.Current())
	}
}

func TestAddWhenNothingPlayingThenStartCurrent(t *testing.T) {
	q := New()
	q.StartCurrent(tr("A"))
	if q.Current().Title != "A" {
		t.Fatal("expected A current")
	}

	q.AddUpcoming(tr("B"))
	if q.Current().Title != "A" || q.Len() != 1 {
		t.Fatal("B should be upcoming, A still current")
	}
}

func TestNextThenNextTraversal(t *testing.T) {
	q := New()
	q.StartCurrent(tr("A"))
	q.AddUpcoming(tr("B"))
	q.AddUpcoming(tr("C"))

	if cur, _ := q.Next(); cur.Title != "B" {
		t.Fatal("expected B")
	}
	if cur, _ := q.Next(); cur.Title != "C" {
		t.Fatal("expected C")
	}
	if _, ok := q.Next(); ok {
		t.Fatal("expected not ok after draining queue")
	}
	if q.Current().Title != "C" {
		t.Fatalf("expected still C, got %v", q.Current())
	}
}

func TestNextAfterFailureSkipsBrokenTrack(t *testing.T) {
	q := New()
	q.StartCurrent(tr("A"))
	q.AddUpcoming(tr("B"))
	q.AddUpcoming(tr("C"))

	next, ok := q.Next()
	if !ok {
		t.Fatal("expected Next to find B")
	}
	if next.Title != "B" {
		t.Fatalf("expected B after failure skip, got %v", next)
	}
	if q.Len() != 1 || q.Upcoming()[0].Title != "C" {
		t.Fatalf("expected queue to retain C, got %v", q.Upcoming())
	}
	if !q.HasPrevious() {
		t.Fatal("expected A in history after skip")
	}
	if _, ok := q.Next(); !ok || q.Current().Title != "C" {
		t.Fatal("expected to advance to C")
	}
	if _, ok := q.Next(); ok {
		t.Fatalf("expected no further next after draining queue, current=%v", q.Current())
	}
	if q.Current().Title != "C" {
		t.Fatalf("expected current still C at end, got %v", q.Current())
	}
}

func TestEmptyQueueAfterCompletion(t *testing.T) {
	q := New()
	q.StartCurrent(tr("A"))
	q.ClearCurrent()
	if q.Current() != nil {
		t.Fatalf("expected nil current after ClearCurrent, got %v", q.Current())
	}
	q.StartCurrent(tr("B"))
	if q.Current().Title != "B" {
		t.Fatalf("expected B current, got %v", q.Current())
	}
	q.ClearCurrent()
	q.ClearCurrent()
	if q.Current() != nil {
		t.Fatal("expected nil current after second ClearCurrent")
	}
}

func TestInvalidTransitionSafety(t *testing.T) {
	q := New()
	if _, ok := q.Next(); ok {
		t.Fatal("expected Next on empty queue to be not ok")
	}
	if act := q.PreviousPressed(); act != PrevNothing {
		t.Fatalf("expected PrevNothing on empty queue, got %d", act)
	}
	if q.Current() != nil {
		t.Fatal("expected nil current on empty queue")
	}
	if q.HasUpcoming() || q.HasPrevious() || q.Len() != 0 {
		t.Fatal("expected empty queue invariants")
	}
	q.Next()
	q.PreviousPressed()
	q.Next()
	if q.Current() != nil {
		t.Fatal("expected nil current throughout empty-queue operations")
	}
}

func TestAddUpcomingWhenNothingCurrent(t *testing.T) {
	q := New()
	q.AddUpcoming(tr("A"))
	q.AddUpcoming(tr("B"))
	if q.Len() != 2 {
		t.Fatalf("expected 2 upcoming, got %d", q.Len())
	}
	q.StartCurrent(tr("C"))
	if q.Len() != 0 {
		t.Fatalf("expected upcoming discarded after StartCurrent, got %d", q.Len())
	}
	if q.Current().Title != "C" {
		t.Fatalf("expected C current, got %v", q.Current())
	}
}
