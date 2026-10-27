//go:build integration

package player

import (
	"context"
	"testing"
	"time"
)

func TestRealMPVControls(t *testing.T) {
	p := New().(*mpvPlayer)
	defer p.Stop()

	source := "https://www.youtube.com/watch?v=jNQXAC9IVRw"
	if err := p.Play(context.Background(), source); err != nil {
		t.Fatalf("Play: %v", err)
	}

	ev := p.Events()

	if !waitForOK(20*time.Second, func() bool {
		st := p.State()
		return st.Playing && st.Duration > 0
	}) {
		t.Fatalf("never got playing+duration, state=%+v", p.State())
	}

	if err := p.SetVolume(40); err != nil {
		t.Fatalf("SetVolume: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool { return p.State().Volume == 40 })

	pos1 := p.State().Position
	time.Sleep(1500 * time.Millisecond)
	if pos2 := p.State().Position; pos2 <= pos1 {
		t.Fatalf("position did not advance: pos1=%v pos2=%v", pos1, pos2)
	}

	if err := p.TogglePause(); err != nil {
		t.Fatalf("TogglePause: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool { return !p.State().Playing })

	if err := p.TogglePause(); err != nil {
		t.Fatalf("TogglePause: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool { return p.State().Playing })

	if err := p.Seek(5); err != nil {
		t.Fatalf("Seek: %v", err)
	}
	_ = ev
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	if !waitForOK(timeout, cond) {
		t.Fatal("condition not met within timeout")
	}
}

func waitForOK(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(150 * time.Millisecond)
	}
	return false
}
