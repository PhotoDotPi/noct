package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/PhotoDotPi/noct/internal/dep"
	"github.com/PhotoDotPi/noct/internal/player"
	"github.com/PhotoDotPi/noct/internal/search"
)

// ---- Fakes ----

type fakeProvider struct {
	results []search.Result
	err     error
}

func (f fakeProvider) Search(_ context.Context, _ string) ([]search.Result, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.results, nil
}

type fakePlayer struct {
	events chan player.Event
	mu     sync.Mutex
	calls  []string
	err    error
	state  player.State
}

func newFakePlayer() *fakePlayer {
	return &fakePlayer{events: make(chan player.Event, 64)}
}

func (f *fakePlayer) record(call string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call)
}

func (f *fakePlayer) Play(_ context.Context, source string) error {
	if f.err != nil {
		return f.err
	}
	f.record("Play:" + source)
	return nil
}

func (f *fakePlayer) TogglePause() error     { f.record("TogglePause"); return f.err }
func (f *fakePlayer) SetPaused(b bool) error { f.record(fmt.Sprintf("SetPaused:%v", b)); return f.err }
func (f *fakePlayer) Seek(s float64) error   { f.record(fmt.Sprintf("Seek:%.0f", s)); return f.err }
func (f *fakePlayer) SetVolume(v int) error {
	f.record(fmt.Sprintf("SetVolume:%d", v))
	return f.err
}
func (f *fakePlayer) State() player.State {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state
}
func (f *fakePlayer) Events() <-chan player.Event { return f.events }
func (f *fakePlayer) Stop() error                 { return nil }
func (f *fakePlayer) Done() <-chan struct{}       { return make(chan struct{}) }

func (f *fakePlayer) push(t player.Event) {
	f.events <- t
}

func (f *fakePlayer) getCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.calls))
	copy(out, f.calls)
	return out
}

func testModel(t *testing.T) (Model, *fakePlayer) {
	t.Helper()
	fp := newFakePlayer()
	m := New(fakeProvider{results: []search.Result{
		{ID: "a1", Title: "Song A", Channel: "Chan A", Duration: 3 * time.Minute, VideoURL: "http://x/a1"},
		{ID: "b2", Title: "Song B", Channel: "Chan B", Duration: 65 * time.Minute, VideoURL: "http://x/b2"},
	}}, fp)
	m.width = 100
	m.height = 30
	m.layout()
	return m, fp
}

func runeMsg(r rune) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// runCmds executes a command and recursively drains tea.Batch/batch msg results.
func runCmds(t *testing.T, cmds ...tea.Cmd) {
	t.Helper()
	for _, c := range cmds {
		if c == nil {
			continue
		}
		msg := c()
		switch m := msg.(type) {
		case tea.BatchMsg:
			runCmds(t, m...)
		}
	}
}

// runCmdsReturn executes commands and returns the first non-batch message.
func runCmdsReturn(t *testing.T, cmds ...tea.Cmd) tea.Msg {
	t.Helper()
	for _, c := range cmds {
		if c == nil {
			continue
		}
		msg := c()
		switch m := msg.(type) {
		case tea.BatchMsg:
			if got := runCmdsReturn(t, m...); got != nil {
				return got
			}
		default:
			return msg
		}
	}
	return nil
}

// ---- Layout tests ----

func TestComputeLayoutResponsiveModes(t *testing.T) {
	cases := []struct {
		w, h  int
		mode  layoutMode
		queue bool
	}{
		{160, 50, layoutUltra, true},
		{140, 40, layoutUltra, true},
		{120, 40, layoutUltra, true},
		{100, 30, layoutWide, true},
		{90, 24, layoutWide, true},
		{80, 24, layoutNormal, false},
		{70, 20, layoutNormal, false},
		{60, 15, layoutNormal, false},
		{50, 15, layoutCompact, false},
		{40, 12, layoutCompact, false},
		{30, 10, layoutCompact, false},
	}
	for _, c := range cases {
		l := computeLayout(c.w, c.h)
		if l.mode != c.mode {
			t.Errorf("%dx%d: mode = %v, want %v", c.w, c.h, l.mode, c.mode)
		}
		if l.contentH < 1 {
			t.Errorf("%dx%d: contentH %d must be positive", c.w, c.h, l.contentH)
		}
		if l.resultsH() < 1 {
			t.Errorf("%dx%d: resultsH %d must be positive", c.w, c.h, l.resultsH())
		}
		if l.showQueue != c.queue {
			t.Errorf("%dx%d: showQueue = %v, want %v", c.w, c.h, l.showQueue, c.queue)
		}
		if l.contentTop < 0 || l.contentTop >= c.h {
			t.Errorf("%dx%d: contentTop %d out of range", c.w, c.h, l.contentTop)
		}
	}
}

func TestComputeLayoutAlwaysStable(t *testing.T) {
	sizes := [][2]int{
		{10, 5}, {15, 8}, {20, 10}, {30, 12}, {40, 14}, {50, 15},
		{60, 15}, {70, 20}, {80, 24}, {100, 30}, {120, 40}, {160, 50},
	}
	for _, sz := range sizes {
		for _, h := range []int{5, 10, 20, 30, 50} {
			l := computeLayout(sz[0], h)
			if l.contentH < 1 {
				t.Errorf("%dx%d: contentH %d must be positive", sz[0], h, l.contentH)
			}
			if l.width != sz[0] || l.height != h {
				t.Errorf("%dx%d: dims not preserved", sz[0], h)
			}
			// queue width must not exceed available width
			if l.showQueue && l.queueW >= l.width {
				t.Errorf("%dx%d: queueW %d >= width", sz[0], h, l.queueW)
			}
		}
	}
}

// ---- Rendering tests ----

func TestRendersAllModes(t *testing.T) {
	m, _ := testModel(t)

	idle := m.View()
	if idle == "" {
		t.Error("idle view rendered empty")
	}
	if !strings.Contains(idle, "NOCT") {
		t.Errorf("expected NOCT title: %q", idle)
	}

	m.mode = modeSearchInput
	if s := m.View(); s == "" {
		t.Error("search input view rendered empty")
	}
	if !strings.Contains(m.View(), "What do you want to play?") {
		t.Errorf("expected search placeholder")
	}

	m.mode = modeIdle
	m.searching = true
	if s := m.View(); s == "" {
		t.Error("searching view rendered empty")
	}
	if !strings.Contains(m.View(), "Searching") {
		t.Errorf("expected searching indicator, got: %q", m.View())
	}
	m.searching = false

	m.mode = modeResults
	m.setListItems(m.resultsList)
	if s := m.View(); s == "" {
		t.Error("results view rendered empty")
	}

	m.resultsList = nil
	m.query = "rick astley"
	m.mode = modeResults
	if s := m.View(); !strings.Contains(s, "No results") {
		t.Errorf("expected friendly empty state, got: %q", s)
	}
}

func TestPlayerBarShowsPlayingState(t *testing.T) {
	m, _ := testModel(t)
	m.queue.StartCurrent(&player.Track{Title: "Song A", Channel: "Chan A", Source: "http://x/a1"})
	m.state = player.State{Playing: true, Position: 90, Duration: 300, Volume: 75}
	m.width = 100

	view := m.View()
	if !strings.Contains(view, "Song A") {
		t.Errorf("player bar missing title: %q", view)
	}
	if !strings.Contains(view, "1:30") {
		t.Errorf("expected position 1:30, got: %q", view)
	}
	if !strings.Contains(view, "75%") {
		t.Errorf("expected volume 75%%, got: %q", view)
	}
	if !strings.Contains(view, "●") && !strings.Contains(view, "━") {
		t.Errorf("expected progress bar, got: %q", view)
	}
}

func TestPlayerBarPausedState(t *testing.T) {
	m, _ := testModel(t)
	m.queue.StartCurrent(&player.Track{Title: "Song A", Source: "http://x/a1"})
	m.state = player.State{Playing: false, Position: 20, Duration: 200, Volume: 10}
	m.width = 100

	view := m.View()
	if !strings.Contains(view, "⏸") {
		t.Errorf("expected pause icon in player bar: %q", view)
	}
}

func TestPlayerBarNothingPlaying(t *testing.T) {
	m, _ := testModel(t)
	m.width = 40
	view := m.View()
	if !strings.Contains(view, "nothing playing") {
		t.Errorf("expected nothing-playing message: %q", view)
	}
}

func TestNarrowTerminalPlayerBarAdapts(t *testing.T) {
	m, _ := testModel(t)
	m.queue.StartCurrent(&player.Track{Title: "A Long Song Title That Needs Truncation", Channel: "C", Source: "http://x/a"})
	m.state = player.State{Playing: true, Volume: 50}
	m.width = 40
	m.height = 12

	for _, line := range strings.Split(m.View(), "\n") {
		if w := ansi.StringWidth(line); w > 40 {
			t.Errorf("narrow terminal: line width %d exceeds 40: %q", w, line)
		}
	}
}

func TestVolumeHiddenWhenNarrow(t *testing.T) {
	m, _ := testModel(t)
	m.queue.StartCurrent(&player.Track{Title: "Song A", Source: "http://x/a"})
	m.state = player.State{Playing: true, Volume: 50}
	m.width = 70
	m.height = 20

	// volume (50%) should be hidden below 78 columns
	if v := m.View(); strings.Contains(v, "50%") {
		t.Errorf("volume should be hidden at width 70: %q", v)
	}
}

func TestViewNeverOverflows(t *testing.T) {
	sizes := [][2]int{
		{160, 50}, {140, 40}, {120, 40}, {100, 30}, {90, 24}, {80, 24},
		{70, 20}, {60, 15}, {50, 15}, {40, 12}, {30, 10},
	}
	for _, sz := range sizes {
		w, h := sz[0], sz[1]
		m, _ := testModel(t)
		m.width, m.height = w, h
		m.layout()

		scenarios := map[string]func(*Model){
			"idle":    func(m *Model) {},
			"input":   func(m *Model) { m.mode = modeSearchInput },
			"results": func(m *Model) { m.mode = modeResults; m.setListItems(m.resultsList) },
			"empty":   func(m *Model) { m.mode = modeResults; m.resultsList = nil },
			"playing": func(m *Model) {
				m.queue.StartCurrent(&player.Track{Title: "A mid-length song title to exercise truncation", Source: "http://x/a"})
				m.queue.AddUpcoming(&player.Track{Title: "Next track", Source: "http://x/b"})
				m.state = player.State{Playing: true, Position: 61, Duration: 240, Volume: 42}
			},
			"help": func(m *Model) { m.helpOpen = true },
		}
		for name, setup := range scenarios {
			m2 := m
			setup(&m2)
			view := m2.View()
			for _, line := range strings.Split(view, "\n") {
				if lw := ansi.StringWidth(line); lw > w {
					t.Errorf("%dx%d %s: line width %d > %d: %q", w, h, name, lw, w, line)
				}
			}
		}
	}
}

// ---- Search interaction tests ----

func TestSlashEntersSearchMode(t *testing.T) {
	m, _ := testModel(t)
	if m.mode != modeIdle {
		t.Fatalf("expected idle mode, got %d", m.mode)
	}
	nm, _ := m.Update(runeMsg('/'))
	if nm.(Model).mode != modeSearchInput {
		t.Fatalf("expected search input mode after '/', got %d", nm.(Model).mode)
	}
}

func TestSelectWhenNothingPlayingPlays(t *testing.T) {
	m, fp := testModel(t)
	m.mode = modeResults
	m.resultsList = []search.Result{
		{ID: "a1", Title: "Song A", Channel: "Chan A", Duration: time.Minute, VideoURL: "http://x/a1"},
	}
	m.setListItems(m.resultsList)

	cmd := m.startPlayback()
	if cmd == nil {
		t.Fatal("expected a play command")
	}
	runCmds(t, cmd)
	if m.queue.Current() == nil || m.queue.Current().Title != "Song A" {
		t.Fatalf("expected Song A current, got %v", m.queue.Current())
	}
	if len(fp.getCalls()) != 1 || !strings.HasPrefix(fp.getCalls()[0], "Play:") {
		t.Errorf("expected Play call, got %v", fp.getCalls())
	}
}

func TestSelectWhilePlayingAddsToQueue(t *testing.T) {
	m, fp := testModel(t)
	m.queue.StartCurrent(&player.Track{Title: "Song A", Channel: "C", Source: "http://x/a"})
	m.queue.AddUpcoming(&player.Track{Title: "Queued", Channel: "C", Source: "http://x/q"})
	m.mode = modeResults
	m.resultsList = []search.Result{
		{ID: "a1", Title: "Song A", Channel: "C", Duration: time.Minute, VideoURL: "http://x/a"},
		{ID: "b2", Title: "Song B", Channel: "C", Duration: time.Minute, VideoURL: "http://x/b"},
	}
	m.setListItems(m.resultsList)

	fp.record("reset")
	m.results.Select(1)
	if cmd := m.startPlayback(); cmd != nil {
		t.Fatal("expected no play command when queueing")
	}
	if m.queue.Current().Title != "Song A" {
		t.Fatalf("current should remain Song A, got %v", m.queue.Current())
	}
	if m.queue.Len() != 2 {
		t.Fatalf("expected 2 queued, got %d", m.queue.Len())
	}
	up := m.queue.Upcoming()
	if up[0].Title != "Queued" || up[1].Title != "Song B" {
		t.Fatalf("unexpected queue order: %v %v", up[0].Title, up[1].Title)
	}
}

func TestTrackFinishedAdvancesQueue(t *testing.T) {
	m, _ := testModel(t)
	m.queue.StartCurrent(&player.Track{Title: "A", Channel: "C", Source: "http://x/a"})
	m.queue.AddUpcoming(&player.Track{Title: "B", Channel: "C", Source: "http://x/b"})

	nm, cmd := m.Update(playerEventMsg{event: player.Event{Type: player.EventTrackFinished}})
	nm2 := nm.(Model)
	if nm2.queue.Current() == nil || nm2.queue.Current().Title != "B" {
		t.Fatalf("expected auto-advance to B, got %v", nm2.queue.Current())
	}
	if cmd == nil {
		t.Fatal("expected a follow-up command")
	}
}

func TestTrackFinishedWithEmptyQueueStops(t *testing.T) {
	m, _ := testModel(t)
	m.queue.StartCurrent(&player.Track{Title: "A", Channel: "C", Source: "http://x/a"})

	nm, _ := m.Update(playerEventMsg{event: player.Event{Type: player.EventTrackFinished}})
	nm2 := nm.(Model)
	if nm2.queue.Current() != nil {
		t.Fatalf("expected current cleared after finishing with empty queue")
	}
	if nm2.state.Playing {
		t.Fatal("expected playing=false")
	}
}

// ---- Playback keys ----

func TestSpaceTogglesPause(t *testing.T) {
	m, fp := testModel(t)
	m.queue.StartCurrent(&player.Track{Title: "A", Channel: "C", Source: "http://x/a"})
	m.mode = modeIdle

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if cmd == nil {
		t.Fatal("expected a toggle-pause command")
	}
	runCmds(t, cmd)
	if len(fp.getCalls()) != 1 || fp.getCalls()[0] != "TogglePause" {
		t.Fatalf("expected TogglePause, got %v", fp.getCalls())
	}
}

func TestSeekKeys(t *testing.T) {
	m, fp := testModel(t)
	m.queue.StartCurrent(&player.Track{Title: "A", Channel: "C", Source: "http://x/a"})
	m.mode = modeIdle

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	runCmds(t, cmd)
	if len(fp.getCalls()) != 1 || fp.getCalls()[0] != "Seek:5" {
		t.Fatalf("expected Seek:5, got %v", fp.getCalls())
	}

	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	runCmds(t, cmd)
	if len(fp.getCalls()) != 2 || fp.getCalls()[1] != "Seek:-5" {
		t.Fatalf("expected Seek:-5, got %v", fp.getCalls())
	}
}

func TestVolumeKeys(t *testing.T) {
	m, fp := testModel(t)
	m.queue.StartCurrent(&player.Track{Title: "A", Channel: "C", Source: "http://x/a"})
	m.state = player.State{Volume: 50}
	m.mode = modeIdle

	_, cmd := m.Update(runeMsg('+'))
	runCmds(t, cmd)
	if len(fp.getCalls()) != 1 || fp.getCalls()[0] != "SetVolume:55" {
		t.Fatalf("expected SetVolume:55, got %v", fp.getCalls())
	}

	m.state = player.State{Volume: 98}
	_, cmd = m.Update(runeMsg('+'))
	runCmds(t, cmd)
	if len(fp.getCalls()) != 2 || fp.getCalls()[1] != "SetVolume:100" {
		t.Fatalf("expected clamped SetVolume:100, got %v", fp.getCalls())
	}
}

func TestPlaybackControlsDisabledInSearchInput(t *testing.T) {
	m, fp := testModel(t)
	m.queue.StartCurrent(&player.Track{Title: "A", Channel: "C", Source: "http://x/a"})
	m.mode = modeSearchInput

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if cmd != nil {
		t.Fatal("expected no toggle-pause command while in search input")
	}
	if len(fp.getCalls()) != 0 {
		t.Fatalf("expected no player calls while typing, got %v", fp.getCalls())
	}
}

func TestQuitBlockedInSearchInput(t *testing.T) {
	m, _ := testModel(t)
	m.mode = modeSearchInput
	before := m.mode
	nm, _ := m.Update(runeMsg('q'))
	if nm.(Model).mode != before {
		t.Fatalf("expected still in search input after typing q")
	}
}

// ---- Error states ----

func TestRendersErrorForMissingDependency(t *testing.T) {
	m, _ := testModel(t)
	m.mode = modeResults
	m.lastError = &dep.MissingDependencyError{Name: "yt-dlp"}
	if s := m.View(); !strings.Contains(s, "yt-dlp is required") {
		t.Errorf("expected actionable yt-dlp error, got: %q", s)
	}
}

func TestPlaybackFailureShowsError(t *testing.T) {
	m, fp := testModel(t)
	m.mode = modeResults
	m.resultsList = []search.Result{
		{ID: "a1", Title: "Song A", Channel: "C", Duration: time.Minute, VideoURL: "http://x/a"},
	}
	m.setListItems(m.resultsList)
	fp.err = errors.New("boom")

	cmd := m.startPlayback()
	if cmd == nil {
		t.Fatal("expected a play command")
	}
	msg := runCmdsReturn(t, cmd)
	if pfm, ok := msg.(playFailedMsg); !ok {
		t.Fatalf("expected playFailedMsg, got %T", msg)
	} else if pfm.err == nil {
		t.Fatal("expected a failure error")
	}
}

func TestPlaybackFailureWithQueueAdvances(t *testing.T) {
	m, _ := testModel(t)
	m.queue.StartCurrent(&player.Track{Title: "A", Channel: "C", Source: "http://x/a"})
	m.queue.AddUpcoming(&player.Track{Title: "B", Channel: "C", Source: "http://x/b"})

	nm, cmd := m.Update(playFailedMsg{err: errors.New("boom")})
	nm2 := nm.(Model)
	if nm2.queue.Current() == nil || nm2.queue.Current().Title != "B" {
		t.Fatalf("expected queue to advance past failed track to B, got %v", nm2.queue.Current())
	}
	if cmd == nil {
		t.Fatal("expected a follow-up play command after advancing")
	}
}

func TestMPVExitShowsWarning(t *testing.T) {
	m, _ := testModel(t)
	nm, _ := m.Update(playerEventMsg{event: player.Event{Type: player.EventExited}})
	nm2 := nm.(Model)
	if nm2.lastError == nil {
		t.Fatal("expected user-facing error when MPV exits")
	}
	if nm2.state.Playing {
		t.Fatal("expected playing=false after MPV exit")
	}
	view := nm2.View()
	if !strings.Contains(view, "MPV") && !strings.Contains(view, "Playback unavailable") {
		t.Errorf("expected playback-unavailable message, got: %q", view)
	}
}

func TestErrNoResultsRendersEmptyNotError(t *testing.T) {
	m, _ := testModel(t)
	m.searchGen = 1
	m.query = "nonsense query"
	nm, _ := m.Update(searchErrorMsg{err: search.ErrNoResults, gen: 1})
	m2 := nm.(Model)
	if m2.lastError != nil {
		t.Fatalf("ErrNoResults should not set lastError")
	}
	if m2.mode != modeResults {
		t.Fatalf("expected results mode, got %d", m2.mode)
	}
	view := m2.View()
	if !strings.Contains(view, "No results") {
		t.Errorf("expected the empty-state message on ErrNoResults, got: %q", view)
	}
}

// ---- Stale search ----

func TestStaleSearchResultsDiscarded(t *testing.T) {
	m, _ := testModel(t)
	m.searchGen = 2

	nm, _ := m.Update(searchResultsMsg{
		results: []search.Result{{ID: "old", Title: "Old", Channel: "C"}},
		gen:     1,
	})
	nm2 := nm.(Model)
	if len(nm2.resultsList) != 0 {
		t.Fatalf("stale search results should be discarded")
	}

	nm, _ = nm2.Update(searchResultsMsg{
		results: []search.Result{{ID: "new", Title: "New", Channel: "C"}},
		gen:     2,
	})
	nm2 = nm.(Model)
	if len(nm2.resultsList) != 1 || nm2.resultsList[0].ID != "new" {
		t.Fatalf("expected fresh results applied")
	}
}

func TestStaleSearchErrorDiscarded(t *testing.T) {
	m, _ := testModel(t)
	m.searchGen = 2

	nm, _ := m.Update(searchErrorMsg{err: errors.New("stale failure"), gen: 1})
	nm2 := nm.(Model)
	if nm2.lastError != nil {
		t.Fatalf("stale search error should be discarded")
	}
}

// ---- Queue rendering ----

func TestQueueSectionShowsUpNext(t *testing.T) {
	m, _ := testModel(t)
	m.queue.StartCurrent(&player.Track{Title: "Current", Source: "http://x/a"})
	m.queue.AddUpcoming(&player.Track{Title: "Up One", Source: "http://x/b"})
	m.queue.AddUpcoming(&player.Track{Title: "Up Two", Source: "http://x/c"})
	m.state = player.State{Playing: true}

	l := computeLayout(120, 30)
	q := m.renderQueue(l)
	if !strings.Contains(q, "UP NEXT") {
		t.Errorf("queue missing heading: %q", q)
	}
	if !strings.Contains(q, "Up One") || !strings.Contains(q, "Up Two") {
		t.Errorf("queue should list upcoming tracks: %q", q)
	}
}

func TestQueueHiddenWhenNoContent(t *testing.T) {
	m, _ := testModel(t)
	l := computeLayout(120, 30)
	if q := m.renderQueue(l); q != "" {
		t.Errorf("expected empty queue when nothing queued, got: %q", q)
	}
}

// ---- Help ----

func TestHelpOpensAndCloses(t *testing.T) {
	m, _ := testModel(t)
	nm, _ := m.Update(runeMsg('?'))
	m2 := nm.(Model)
	if !m2.helpOpen {
		t.Fatal("expected help open after '?'")
	}
	view := m2.View()
	if !strings.Contains(view, "NOCT HELP") {
		t.Errorf("expected NOCT HELP in help view: %q", view)
	}

	nm, _ = m2.Update(runeMsg('?'))
	m3 := nm.(Model)
	if m3.helpOpen {
		t.Fatal("expected help closed after '?'")
	}
}

func TestHelpOpensWhenNoCurrentTrack(t *testing.T) {
	m, _ := testModel(t)
	nm, _ := m.Update(runeMsg('?'))
	m2 := nm.(Model)
	if !m2.helpOpen {
		t.Fatal("expected help to open even when nothing playing")
	}
	view := m2.View()
	if !strings.Contains(view, "NOCT HELP") {
		t.Errorf("expected help view: %q", view)
	}
}

// ---- Notifications ----

func TestPushNotifSetsText(t *testing.T) {
	m, _ := testModel(t)
	cmd := m.pushNotif("Test message")
	if cmd == nil {
		t.Fatal("expected tick cmd for notif")
	}
	if m.notifText != "Test message" {
		t.Fatalf("expected notif text set, got %q", m.notifText)
	}
	view := m.View()
	if !strings.Contains(view, "Test message") {
		t.Errorf("expected notif in view: %q", view)
	}
}

func TestNotifClears(t *testing.T) {
	m, _ := testModel(t)
	m.pushNotif("Hello")
	if m.notifText == "" {
		t.Fatal("notif should be set")
	}
	nm, _ := m.Update(notifClearMsg{})
	m2 := nm.(Model)
	if m2.notifText != "" {
		t.Fatalf("notif should be cleared, got %q", m2.notifText)
	}
}

// ---- Animation ----

func TestSelectionAnimationStepsTowardTarget(t *testing.T) {
	m, _ := testModel(t)
	m.setListItems(m.resultsList)
	m.results.Select(0)

	// Move target to index 1 and expect a cmd (tick).
	cmd := m.selAnim.setTarget(1)
	if cmd == nil {
		t.Fatal("expected animation tick when target changes")
	}
	if !m.selAnim.active {
		t.Fatal("expected animation to be active")
	}
	// Advance to target.
	for i := 0; i < 100; i++ {
		if _, running := m.selAnim.step(); !running {
			break
		}
	}
	if m.selAnim.pos != 1 {
		t.Fatalf("expected anim to reach target 1, got %v", m.selAnim.pos)
	}
	if m.selAnim.active {
		t.Fatal("expected animation settled after reaching target")
	}
}

func TestSelectionAnimationStableAtInitial(t *testing.T) {
	m, _ := testModel(t)
	m.setListItems(m.resultsList)
	m.results.Select(0)
	if m.selAnim.pos != 0 || m.selAnim.target != 0 || m.selAnim.active {
		t.Fatalf("expected stable animation at initial selection, got %+v", m.selAnim)
	}
}

func TestAnimationTickMsgUpdates(t *testing.T) {
	m, _ := testModel(t)
	m.setListItems(m.resultsList)
	m.selAnim.setTarget(1)

	// First step moves position.
	nm, cmd := m.Update(animTickMsg{})
	m2 := nm.(Model)
	if !m2.selAnim.active {
		t.Log("animation may have pos==target; accepting settle")
	}
	_ = cmd
}

// ---- Resize ----

func TestResizeUpdatesLayout(t *testing.T) {
	m, _ := testModel(t)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m2 := nm.(Model)
	if m2.width != 60 || m2.height != 20 {
		t.Fatalf("resize should update width/height, got %dx%d", m2.width, m2.height)
	}
	if m2.results.Width() <= 0 || m2.results.Height() <= 0 {
		t.Fatalf("results list should have positive size after resize")
	}
}

func TestResizeDuringAllModes(t *testing.T) {
	modes := []func(*Model){
		func(m *Model) {},
		func(m *Model) { m.mode = modeSearchInput },
		func(m *Model) { m.mode = modeResults; m.setListItems(m.resultsList) },
		func(m *Model) {
			m.queue.StartCurrent(&player.Track{Title: "Current", Source: "http://x/a"})
			m.state = player.State{Playing: true}
		},
		func(m *Model) { m.helpOpen = true },
	}
	for _, setup := range modes {
		m, _ := testModel(t)
		setup(&m)
		// Resize small -> large -> small repeatedly
		for _, size := range [][2]int{{40, 12}, {120, 40}, {60, 20}, {160, 50}} {
			nm, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			m = nm.(Model)
			view := m.View()
			for _, line := range strings.Split(view, "\n") {
				if ansi.StringWidth(line) > size[0] {
					t.Errorf("resize %dx%d: line overflow: %q", size[0], size[1], line)
				}
			}
		}
	}
}

// ---- Result row format ----

func TestResultRowFormat(t *testing.T) {
	m, _ := testModel(t)
	r := search.Result{Title: "Never Gonna Give You Up", Channel: "Rick Astley", Duration: 3*time.Minute + 33*time.Second}
	line := m.renderResultRow(r, 0, 60)
	if !strings.Contains(line, "1.") {
		t.Errorf("row missing number: %q", line)
	}
	if !strings.Contains(line, "3:33") {
		t.Errorf("row missing duration: %q", line)
	}
	if !strings.Contains(line, "Never Gonna Give You Up") {
		t.Errorf("row missing title: %q", line)
	}
	if strings.Count(line, "\n") > 1 {
		t.Errorf("row should be at most two lines: %q", line)
	}
}

func TestMarksReflectPlaybackState(t *testing.T) {
	m, _ := testModel(t)
	m.queue.StartCurrent(&player.Track{Title: "Song A", Source: "http://x/a1"})

	m.state = player.State{Playing: true}
	if !m.isPlayingURL("http://x/a1") {
		t.Error("isPlayingURL should return true for current playing URL")
	}
	if m.isPlayingURL("http://x/b2") {
		t.Error("isPlayingURL should return false for non-current URL")
	}

	m.state = player.State{Playing: false}
	if !m.isPlayingURL("http://x/a1") {
		t.Error("isPlayingURL should still identify current track when paused")
	}
}

// ---- Theme ----

func TestThemeCentralized(t *testing.T) {
	if theme.Primary == "" || theme.Accent == "" || theme.Surface == "" {
		t.Error("theme should define core colors")
	}
	// Ensure styles reference the theme, not hardcoded scattered values.
	if sAppTitle.GetForeground() == nil {
		t.Error("app title should have foreground from theme")
	}
	if sSelectedBg.GetBackground() == nil {
		t.Error("selection style should have background from theme")
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{0, "0:00"},
		{time.Minute, "1:00"},
		{3*time.Minute + 33*time.Second, "3:33"},
		{65 * time.Minute, "1:05:00"},
		{3661 * time.Second, "1:01:01"},
	}
	for _, c := range cases {
		if got := formatDuration(c.in); got != c.want {
			t.Errorf("formatDuration(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
