package player

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PhotoDotPi/noct/internal/dep"
)

type fakeComm struct {
	mu     sync.Mutex
	calls  [][]any
	events chan ipcFrame
	closed bool
}

func newFakeComm() *fakeComm {
	return &fakeComm{events: make(chan ipcFrame, 16)}
}

func (f *fakeComm) Command(_ context.Context, args ...any) (any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, args)
	return nil, nil
}

func (f *fakeComm) Events() <-chan ipcFrame { return f.events }

func (f *fakeComm) Close() error {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
	return nil
}

func (f *fakeComm) callList() [][]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func last(calls [][]any) []any {
	if len(calls) == 0 {
		return nil
	}
	return calls[len(calls)-1]
}

func startedPlayer(t *testing.T) (*mpvPlayer, *fakeComm) {
	t.Helper()
	fc := newFakeComm()
	p := &mpvPlayer{
		done:       make(chan struct{}),
		events:     make(chan Event, 32),
		stopEvents: make(chan struct{}),
		logDir:     "",
		lookPath: func(string) (string, error) {
			return "/usr/bin/mpv", nil
		},
		comm:    fc,
		started: true,
	}
	go p.readEvents()
	return p, fc
}

func TestPlayMissingMPV(t *testing.T) {
	p := &mpvPlayer{
		done:       make(chan struct{}),
		events:     make(chan Event, 32),
		stopEvents: make(chan struct{}),
		logDir:     "",
		lookPath: func(string) (string, error) {
			return "", errors.New("executable file not found in $PATH")
		},
	}
	if err := p.Play(context.Background(), "http://x/a"); !dep.IsMissing(err) {
		t.Fatalf("expected MissingDependencyError, got %v", err)
	}
}

func TestPlayMissingYtDlp(t *testing.T) {
	p := &mpvPlayer{
		done:       make(chan struct{}),
		events:     make(chan Event, 32),
		stopEvents: make(chan struct{}),
		logDir:     "",
		lookPath: func(f string) (string, error) {
			if f == mpvName {
				return "/usr/bin/mpv", nil
			}
			return "", errors.New("not found")
		},
	}
	if err := p.Play(context.Background(), "http://x/a"); !dep.IsMissing(err) {
		t.Fatalf("expected MissingDependencyError, got %v", err)
	}
}

func TestPlayEmptySource(t *testing.T) {
	p, _ := startedPlayer(t)
	if err := p.Play(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty source")
	}
}

func TestPlaySendsLoadfile(t *testing.T) {
	p, fc := startedPlayer(t)
	if err := p.Play(context.Background(), "http://x/a"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := last(fc.callList())
	if len(got) != 3 || got[0] != "loadfile" || got[1] != "http://x/a" || got[2] != "replace" {
		t.Fatalf("unexpected loadfile command: %v", got)
	}
}

func TestTogglePauseCycles(t *testing.T) {
	p, fc := startedPlayer(t)
	if err := p.TogglePause(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := last(fc.callList())
	if len(got) != 2 || got[0] != "cycle" || got[1] != "pause" {
		t.Fatalf("unexpected toggle command: %v", got)
	}
}

func TestSetPaused(t *testing.T) {
	p, fc := startedPlayer(t)
	if err := p.SetPaused(true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := last(fc.callList())
	if got[0] != "set_property" || got[1] != "pause" || got[2] != true {
		t.Fatalf("unexpected set_paused command: %v", got)
	}
}

func TestSeekRelative(t *testing.T) {
	p, fc := startedPlayer(t)
	if err := p.Seek(5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := last(fc.callList())
	if got[0] != "seek" || got[1] != 5.0 || got[2] != "relative" {
		t.Fatalf("unexpected seek command: %v", got)
	}
}

func TestSetVolumeClamps(t *testing.T) {
	p, fc := startedPlayer(t)

	_ = p.SetVolume(150)
	if got := last(fc.callList()); got[1] != "volume" || got[2] != 100 {
		t.Fatalf("expected volume clamped to 100, got %v", got)
	}

	_ = p.SetVolume(-20)
	if got := last(fc.callList()); got[1] != "volume" || got[2] != 0 {
		t.Fatalf("expected volume clamped to 0, got %v", got)
	}

	_ = p.SetVolume(63)
	if got := last(fc.callList()); got[1] != "volume" || got[2] != 63 {
		t.Fatalf("expected volume 63, got %v", got)
	}
}

func TestPropertyEventsUpdateState(t *testing.T) {
	p, fc := startedPlayer(t)
	e := p.Events()

	fc.events <- ipcFrame{Event: "property-change", ID: pausePropID, Data: mustRaw(false)}
	fc.events <- ipcFrame{Event: "property-change", ID: volumePropID, Data: mustRaw(57.0)}
	fc.events <- ipcFrame{Event: "property-change", ID: durationPropID, Data: mustRaw(240.0)}

	for i := 0; i < 3; i++ {
		select {
		case <-e:
		case <-time.After(time.Second):
			t.Fatalf("expected state change %d", i)
		}
	}

	st := p.State()
	if !st.Playing {
		t.Errorf("expected playing=true from pause=false, got %+v", st)
	}
	if st.Volume != 57 {
		t.Errorf("expected volume 57, got %d", st.Volume)
	}
	if st.Duration != 240 {
		t.Errorf("expected duration 240, got %v", st.Duration)
	}
}

func TestEndFileEOFEmitsTrackFinished(t *testing.T) {
	p, fc := startedPlayer(t)
	e := p.Events()
	fc.events <- ipcFrame{Event: "end-file", Reason: "eof"}
	select {
	case ev := <-e:
		if ev.Type != EventTrackFinished {
			t.Fatalf("expected EventTrackFinished, got %+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("expected a track finished event")
	}
}

func TestEndFileStopDoesNotEmitTrackFinished(t *testing.T) {
	p, fc := startedPlayer(t)
	e := p.Events()
	fc.events <- ipcFrame{Event: "end-file", Reason: "stop"}

	select {
	case ev := <-e:
		t.Fatalf("expected no event for stop reason, got %+v", ev)
	case <-time.After(300 * time.Millisecond):
	}

	fc.events <- ipcFrame{Event: "end-file", Reason: "eof"}
	select {
	case ev := <-e:
		if ev.Type != EventTrackFinished {
			t.Fatalf("expected EventTrackFinished, got %+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("expected a track finished event for the eof")
	}
}

func TestMPVLaunchArgs(t *testing.T) {
	p := &mpvPlayer{logDir: "/tmp/logs"}
	args := p.mpvLaunchArgs(`\\.\pipe\noct-x`)
	joined := strings.Join(args, " ")
	for _, want := range []string{"--idle=yes", "--input-ipc-server=" + `\\.\pipe\noct-x`, "--no-video", "--ytdl-format=bestaudio", "--no-terminal"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected arg %q in %v", want, args)
		}
	}
	if !strings.HasPrefix(args[len(args)-1], "--log-file=") || !strings.Contains(joined, "mpv.log") {
		t.Errorf("expected a log-file arg, got %v", args)
	}
}

func mustRaw(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func TestReadEventsStopsWhenIPCCloses(t *testing.T) {
	p, fc := startedPlayer(t)
	done := make(chan struct{})
	go func() {
		p.readEvents()
		close(done)
	}()
	close(fc.events)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("readEvents did not return after events channel closed")
	}
}

func TestStopClosesDone(t *testing.T) {
	fc := newFakeComm()
	p := &mpvPlayer{
		done:       make(chan struct{}),
		events:     make(chan Event, 32),
		stopEvents: make(chan struct{}),
		logDir:     "",
		lookPath: func(string) (string, error) {
			return "/usr/bin/mpv", nil
		},
		comm:    fc,
		started: true,
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("first Stop: %v", err)
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
	if p.started {
		t.Fatal("expected started=false after Stop")
	}
}

func TestWatchProcessEmitsExitedOnUnexpectedExit(t *testing.T) {
	fc := newFakeComm()
	p := &mpvPlayer{
		done:       make(chan struct{}),
		events:     make(chan Event, 32),
		stopEvents: make(chan struct{}),
		logDir:     "",
		comm:       fc,
		started:    true,
	}
	cmd := &exec.Cmd{Process: &os.Process{Pid: 123}}
	cmd.ProcessState = &os.ProcessState{}
	done := make(chan struct{})
	go func() {
		p.watchProcess(cmd)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watchProcess did not return")
	}
	select {
	case <-p.done:
	default:
		t.Fatal("expected done closed after watchProcess")
	}
	ev := <-p.events
	if ev.Type != EventExited {
		t.Fatalf("expected EventExited, got %+v", ev)
	}
}

func TestWatchProcessStopDoesNotEmit(t *testing.T) {
	fc := newFakeComm()
	p := &mpvPlayer{
		done:       make(chan struct{}),
		events:     make(chan Event, 32),
		stopEvents: make(chan struct{}),
		logDir:     "",
		comm:       fc,
		started:    true,
	}
	cmd := &exec.Cmd{Process: &os.Process{Pid: 123}}
	cmd.ProcessState = &os.ProcessState{}
	close(p.stopEvents)
	done := make(chan struct{})
	go func() {
		p.watchProcess(cmd)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watchProcess did not return")
	}
	select {
	case ev := <-p.events:
		t.Fatalf("expected no event after Stop, got %+v", ev)
	default:
	}
}

func TestPollPositionStopsOnDone(t *testing.T) {
	fc := newFakeComm()
	p := &mpvPlayer{
		done:       make(chan struct{}),
		events:     make(chan Event, 32),
		stopEvents: make(chan struct{}),
		logDir:     "",
		comm:       fc,
		started:    true,
	}
	done := make(chan struct{})
	go func() {
		p.pollPosition()
		close(done)
	}()
	close(p.done)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("pollPosition did not return after done closed")
	}
}
