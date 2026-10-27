package player

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/PhotoDotPi/noct/internal/dep"
	"github.com/PhotoDotPi/noct/internal/log"
)

const (
	mpvName   = "mpv"
	ytDlpName = "yt-dlp"
)

const (
	pausePropID    = 1
	volumePropID   = 2
	durationPropID = 3

	pollInterval = 300 * time.Millisecond
)

type ipcComm interface {
	Command(ctx context.Context, args ...any) (any, error)
	Events() <-chan ipcFrame
	Close() error
}

type mpvPlayer struct {
	mu sync.Mutex

	proc     *exec.Cmd
	conn     net.Conn
	comm     ipcComm
	started  bool
	done     chan struct{}
	doneOnce sync.Once

	state State

	events     chan Event
	stopEvents chan struct{}
	stopOnce   sync.Once

	lookPath func(file string) (string, error)
	command  func(name string, args ...string) *exec.Cmd
	logDir   string
}

func New() Player {
	return &mpvPlayer{
		done:       make(chan struct{}),
		events:     make(chan Event, 32),
		stopEvents: make(chan struct{}),
		logDir:     defaultLogDir(),
		lookPath:   exec.LookPath,
		command:    exec.Command,
	}
}

func (p *mpvPlayer) Play(ctx context.Context, source string) error {
	if source == "" {
		return errors.New("invalid playback source: source is empty")
	}
	if err := p.ensureStarted(ctx); err != nil {
		return err
	}

	if _, err := p.comm.Command(ctx, "loadfile", source, "replace"); err != nil {
		return &ControlError{Action: "load", Err: err}
	}

	log.Default().Infof("loadfile: %q", source)
	return nil
}

func (p *mpvPlayer) TogglePause() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := p.comm.Command(ctx, "cycle", "pause"); err != nil {
		return &ControlError{Action: "toggle pause", Err: err}
	}
	log.Default().Infof("toggle pause")
	return nil
}

func (p *mpvPlayer) SetPaused(paused bool) error {
	return p.setProp("pause", paused)
}

func (p *mpvPlayer) Seek(seconds float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := p.comm.Command(ctx, "seek", seconds, "relative"); err != nil {
		return &ControlError{Action: "seek", Err: err}
	}
	log.Default().Infof("seek %+.1fs", seconds)
	return nil
}

func (p *mpvPlayer) SetVolume(percent int) error {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return p.setProp("volume", percent)
}

func (p *mpvPlayer) setProp(name string, value any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := p.comm.Command(ctx, "set_property", name, value); err != nil {
		return &ControlError{Action: "set " + name, Err: err}
	}
	return nil
}

func (p *mpvPlayer) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (p *mpvPlayer) Events() <-chan Event {
	return p.events
}

func (p *mpvPlayer) Done() <-chan struct{} {
	return p.done
}

func (p *mpvPlayer) ensureStarted(ctx context.Context) error {
	p.mu.Lock()
	already := p.started
	p.mu.Unlock()
	if already {
		return nil
	}

	mpvPath, err := p.resolveMPV()
	if err != nil {
		return err
	}
	if _, err := p.lookPath(ytDlpName); err != nil {
		return &dep.MissingDependencyError{Name: ytDlpName, Err: err}
	}
	if err := p.ensureLogDir(); err != nil {
		return err
	}

	endpoint := ipcEndpoint()
	args := p.mpvLaunchArgs(endpoint)
	cmd := p.command(mpvPath, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		log.Default().Errorf("mpv failed to start: %v", err)
		return &StartError{Err: err}
	}

	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := dialIPC(dialCtx, endpoint)
	if err != nil {
		_ = cmd.Process.Kill()
		log.Default().Errorf("mpv ipc connect failed: %v", err)
		return fmt.Errorf("connect to mpv ipc: %w", err)
	}

	comm := newIPC(conn)
	p.observeProperties(comm)

	p.mu.Lock()
	p.proc = cmd
	p.conn = conn
	p.comm = comm
	p.started = true
	p.mu.Unlock()

	log.Default().Infof("mpv started (pid %d), ipc connected", cmd.Process.Pid)

	go p.watchProcess(cmd)
	go p.readEvents()
	go p.pollPosition()

	return nil
}

func (p *mpvPlayer) observeProperties(comm ipcComm) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := comm.Command(ctx, "observe_property", pausePropID, "pause"); err != nil {
		log.Default().Warnf("observe pause failed: %v", err)
	}
	if _, err := comm.Command(ctx, "observe_property", volumePropID, "volume"); err != nil {
		log.Default().Warnf("observe volume failed: %v", err)
	}
	if _, err := comm.Command(ctx, "observe_property", durationPropID, "duration"); err != nil {
		log.Default().Warnf("observe duration failed: %v", err)
	}
}

func (p *mpvPlayer) watchProcess(cmd *exec.Cmd) {
	_ = cmd.Wait()
	p.mu.Lock()
	started := p.started
	p.mu.Unlock()

	select {
	case <-p.stopEvents:
		return
	default:
	}

	if started {
		p.emit(Event{Type: EventExited})
	}
	p.doneOnce.Do(func() { close(p.done) })
}

func (p *mpvPlayer) readEvents() {
	comm := p.comm
	for fr := range comm.Events() {
		switch {
		case fr.Event == "property-change":
			p.handleProperty(fr)
		case fr.Event == "end-file" && fr.Reason == "eof":
			log.Default().Infof("track finished (eof)")
			p.emit(Event{Type: EventTrackFinished, State: p.State()})
		}
	}
}

func (p *mpvPlayer) handleProperty(fr ipcFrame) {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch fr.ID {
	case pausePropID:
		if b, ok := fr.DataBool(); ok {
			p.state.Playing = !b
		}
	case volumePropID:
		if f, ok := fr.DataFloat(); ok {
			p.state.Volume = int(f + 0.5)
		}
	case durationPropID:
		if f, ok := fr.DataFloat(); ok {
			p.state.Duration = f
		}
	}
	p.emitLocked(Event{Type: EventStateChanged, State: p.state})
}

func (p *mpvPlayer) pollPosition() {
	comm := p.comm
	t := time.NewTicker(pollInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			val, err := comm.Command(ctx, "get_property", "time-pos")
			cancel()
			if err != nil {
				select {
				case <-p.done:
					return
				default:
				}
				continue
			}
			f, ok := val.(float64)
			if !ok {
				continue
			}
			p.mu.Lock()
			if f != p.state.Position {
				p.state.Position = f
				p.emitLocked(Event{Type: EventStateChanged, State: p.state})
			}
			p.mu.Unlock()
		case <-p.stopEvents:
			return
		case <-p.done:
			return
		}
	}
}

func (p *mpvPlayer) emit(e Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.emitLocked(e)
}

func (p *mpvPlayer) emitLocked(e Event) {
	select {
	case p.events <- e:
	case <-p.stopEvents:
	default:
	}
}

func (p *mpvPlayer) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stopOnce.Do(func() { close(p.stopEvents) })

	if p.comm != nil {
		_ = p.comm.Close()
		p.comm = nil
	}
	if p.proc != nil && p.proc.Process != nil {
		log.Default().Infof("stopping mpv (pid %d)", p.proc.Process.Pid)
		_ = p.proc.Process.Kill()
		_, _ = p.proc.Process.Wait()
		p.proc = nil
	}
	p.started = false
	return nil
}

func (p *mpvPlayer) resolveMPV() (string, error) {
	return resolve(p, mpvName)
}

func (p *mpvPlayer) ensureLogDir() error {
	if p.logDir == "" {
		return nil
	}
	return os.MkdirAll(p.logDir, 0o755)
}

func (p *mpvPlayer) mpvLaunchArgs(endpoint string) []string {
	args := []string{
		"--no-video",
		"--ytdl-format=bestaudio",
		"--force-window=no",
		"--no-terminal",
		"--idle=yes",
		"--input-ipc-server=" + endpoint,
	}
	if p.logDir != "" {
		args = append(args, "--log-file="+filepath.Join(p.logDir, "mpv.log"))
	}
	return args
}

func defaultLogDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "noct")
}

func resolve(p *mpvPlayer, name string) (string, error) {
	path, err := p.lookPath(name)
	if err != nil {
		return "", &dep.MissingDependencyError{Name: name, Err: err}
	}
	return path, nil
}

type StartError struct {
	Err error
}

func (e *StartError) Error() string {
	return fmt.Sprintf("could not start mpv: %v", e.Err)
}

func (e *StartError) Unwrap() error { return e.Err }

type ControlError struct {
	Action string
	Err    error
}

func (e *ControlError) Error() string {
	return fmt.Sprintf("%s failed: %v", e.Action, e.Err)
}

func (e *ControlError) Unwrap() error { return e.Err }
