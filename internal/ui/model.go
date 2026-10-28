package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/PhotoDotPi/noct/internal/log"
	"github.com/PhotoDotPi/noct/internal/player"
	"github.com/PhotoDotPi/noct/internal/queue"
	"github.com/PhotoDotPi/noct/internal/search"
)

type mode int

const (
	modeIdle mode = iota
	modeSearchInput
	modeResults
)

const (
	seekStepSeconds   = 5.0
	volumeStepPercent = 5
)

var (
	errPlaybackProblem = errors.New("Playback encountered an error")
	errPlayerStopped   = errors.New("MPV stopped unexpectedly")
)

type Model struct {
	provider search.Provider
	player   player.Player
	queue    *queue.Queue

	searchInput *searchInput
	results     list.Model
	loading     spinner.Model

	mode      mode
	query     string
	searching bool
	lastError error

	resultsList []search.Result

	scroll    int
	searchGen uint64

	selAnim animState

	helpOpen bool

	queueFocused bool
	queueSel     int
	queueScroll  int

	notifText string
	notifKind notifKind

	state player.State

	width  int
	height int
	ready  bool
	ctx    context.Context
	cancel context.CancelFunc
}

func New(provider search.Provider, p player.Player) Model {
	ctx, cancel := context.WithCancel(context.Background())
	s := newAnim()
	return Model{
		provider:    provider,
		player:      p,
		queue:       queue.New(),
		searchInput: newSearchInput(),
		loading:     newSpinner(),
		selAnim:     s,
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (m Model) Shutdown() error {
	return m.player.Stop()
}

func (m Model) Init() tea.Cmd {
	return watchPlayerEvents(m.player)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		l := computeLayout(m.width, m.height)
		if !l.showQueue || len(m.queue.Upcoming()) == 0 {
			m.queueFocused = false
		}
		m.clampQueueSelection()
		return m, m.layout()

	case tea.KeyMsg:
		return m, m.handleKey(msg)

	case spinner.TickMsg:
		if m.searching {
			var cmd tea.Cmd
			m.loading, cmd = m.loading.Update(msg)
			return m, cmd
		}
		return m, nil

	case searchResultsMsg:
		if msg.gen != m.searchGen {
			return m, nil
		}
		m.searching = false
		m.lastError = nil
		m.resultsList = msg.results
		m.mode = modeResults
		m.setListItems(msg.results)
		log.Default().Infof("search returned %d results (gen %d)", len(msg.results), msg.gen)
		return m, nil

	case searchErrorMsg:
		if msg.gen != m.searchGen {
			return m, nil
		}
		m.searching = false
		m.mode = modeResults
		if msg.err != nil && !errors.Is(msg.err, search.ErrNoResults) {
			m.lastError = msg.err
		}
		log.Default().Errorf("search failed (gen %d): %v", msg.gen, msg.err)
		return m, nil

	case playFailedMsg:
		m.state.Playing = false
		m.lastError = msg.err
		m.pushNotifErr("Couldn't play this track")
		log.Default().Errorf("playback failed: %v", msg.err)
		if m.queue.HasUpcoming() {
			if next, ok := m.queue.Next(); ok {
				log.Default().Infof("skipping failed track, advancing to %q", next.Title)
				return m, tea.Batch(m.playCurrent(), m.pushNotif("Skipping to next"), watchPlayerEvents(m.player))
			}
		}
		return m, nil

	case playerEventMsg:
		return m.handlePlayerEvent(msg.event)

	case playerEventsClosedMsg:
		m.state.Playing = false
		m.lastError = errPlayerStopped
		return m, nil

	case notifClearMsg:
		m.clearNotif()
		return m, nil

	case animTickMsg:
		if _, running := m.selAnim.step(); running {
			return m, m.selAnim.restartCmd()
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handlePlayerEvent(ev player.Event) (tea.Model, tea.Cmd) {
	switch ev.Type {
	case player.EventStateChanged:
		wasPlaying := m.state.Playing
		m.state = ev.State
		m.lastError = nil
		var cmds []tea.Cmd
		if wasPlaying != m.state.Playing {
			if m.state.Playing {
				cmds = append(cmds, m.pushNotif("Playing ▶"))
			} else {
				cmds = append(cmds, m.pushNotif("Paused ⏸"))
			}
		}
		cmds = append(cmds, watchPlayerEvents(m.player))
		return m, tea.Batch(cmds...)
	case player.EventTrackFinished:
		log.Default().Infof("track finished")
		m.clampQueueSelection()
		return m, tea.Batch(watchPlayerEvents(m.player), m.onTrackFinished())
	case player.EventError:
		m.lastError = errPlaybackProblem
		m.pushNotifErr("Playback error")
		log.Default().Errorf("player error: %v", ev.Err)
	case player.EventExited:
		m.state.Playing = false
		m.lastError = errPlayerStopped
		log.Default().Warnf("mpv exited")
	}
	return m, watchPlayerEvents(m.player)
}

func (m *Model) onTrackFinished() tea.Cmd {
	next, ok := m.queue.Next()
	if ok {
		m.clampQueueSelection()
		log.Default().Infof("auto-advancing to %q", next.Title)
		return tea.Batch(m.playCurrent(), m.pushNotif("Next: "+next.Title))
	}
	m.queue.ClearCurrent()
	m.queueFocused = false
	m.queueSel = 0
	m.state.Playing = false
	m.pushNotif("Queue finished")
	log.Default().Infof("queue empty; playback stopped")
	return nil
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	if m.helpOpen {
		switch msg.String() {
		case "?", "esc", "q", "ctrl+c":
			m.helpOpen = false
			return nil
		}
		return nil
	}

	switch msg.String() {
	case "ctrl+c":
		return tea.Quit
	case "q":
		if m.mode != modeSearchInput {
			return tea.Quit
		}
	case "?":
		if m.mode != modeSearchInput {
			m.helpOpen = true
			return nil
		}
	}

	if m.queueFocused {
		if cmd := m.handleQueueKey(msg); cmd != nil {
			return cmd
		}
		switch msg.String() {
		case "tab", "esc":
			m.queueFocused = false
			return nil
		}
		return nil
	}

	switch msg.String() {
	case "tab":
		if m.canFocusQueue() {
			m.queueFocused = true
			m.clampQueueSelection()
			if len(m.queue.Upcoming()) > 0 && m.queueSel >= len(m.queue.Upcoming()) {
				m.queueSel = len(m.queue.Upcoming()) - 1
			}
			if len(m.queue.Upcoming()) == 0 {
				m.queueFocused = false
			}
			return nil
		}
	}

	switch m.mode {

	case modeSearchInput:
		return m.updateSearchInput(msg)

	case modeResults:
		if cmd := m.handlePlaybackKey(msg); cmd != nil {
			return cmd
		}
		return m.updateResults(msg)

	case modeIdle:
		if cmd := m.handlePlaybackKey(msg); cmd != nil {
			return cmd
		}
		if msg.String() == "/" {
			m.mode = modeSearchInput
			m.lastError = nil
			m.queueFocused = false
			return m.searchInput.focus()
		}
		m.lastError = nil
	}

	return nil
}

func (m *Model) handlePlaybackKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "n":
		return m.nextTrack()
	case "p":
		return m.prevTrack()
	case "space", " ":
		if m.queue.Current() != nil {
			return controlCmd(m.player, "toggle pause", m.player.TogglePause)
		}
	case "left":
		return m.seek(-seekStepSeconds)
	case "right":
		return m.seek(seekStepSeconds)
	case "+", "=":
		return m.changeVolume(volumeStepPercent)
	case "-", "_":
		return m.changeVolume(-volumeStepPercent)
	}
	return nil
}

func (m *Model) updateSearchInput(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = m.previousMode()
		m.searchInput.reset()
		return nil
	case "enter":
		query := strings.TrimSpace(m.searchInput.value())
		if query == "" {
			return nil
		}
		return m.submitSearch(query)
	}
	return m.searchInput.update(msg)
}

func (m *Model) updateResults(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = modeIdle
		m.lastError = nil
		return nil
	case "/":
		m.mode = modeSearchInput
		return m.searchInput.focus()
	case "enter":
		return m.startPlayback()
	}

	var cmd tea.Cmd
	oldIdx := m.results.Index()
	m.results, cmd = m.results.Update(msg)
	newIdx := m.results.Index()
	if newIdx != oldIdx {
		cmd = tea.Batch(cmd, m.selAnim.setTarget(float64(newIdx)))
	}
	m.adjustScroll()
	return cmd
}

func (m *Model) handleQueueKey(msg tea.KeyMsg) tea.Cmd {
	upcoming := m.queue.Upcoming()
	if len(upcoming) == 0 {
		m.queueFocused = false
		m.queueSel = 0
		return nil
	}
	m.clampQueueSelection()

	switch msg.String() {
	case "j", "down":
		if m.queueSel < len(m.queue.Upcoming())-1 {
			m.queueSel++
		}
		return nil
	case "k", "up":
		if m.queueSel > 0 {
			m.queueSel--
		}
		return nil
	case "d", "x", "delete":
		if m.queue.RemoveUpcoming(m.queueSel) {
			m.pushNotif("Removed from queue")
			m.clampQueueSelection()
			return m.pushNotif("Removed from queue")
		}
		m.clampQueueSelection()
		return nil
	case "J":
		if m.queue.MoveUpcoming(m.queueSel, 1) {
			if m.queueSel < len(m.queue.Upcoming())-1 {
				m.queueSel++
			} else {
				m.queueSel = len(m.queue.Upcoming()) - 1
			}
		}
		m.clampQueueSelection()
		return nil
	case "K":
		if m.queue.MoveUpcoming(m.queueSel, -1) {
			if m.queueSel > 0 {
				m.queueSel--
			}
		}
		m.clampQueueSelection()
		return nil
	case "c":
		m.queue.ClearUpcoming()
		m.queueFocused = false
		m.queueSel = 0
		m.queueScroll = 0
		return m.pushNotif("Cleared queue")
	}
	return nil
}

func (m *Model) canFocusQueue() bool {
	l := computeLayout(m.width, m.height)
	if !l.showQueue {
		return false
	}
	return len(m.queue.Upcoming()) > 0
}

func (m *Model) clampQueueSelection() {
	upcoming := m.queue.Upcoming()
	l := computeLayout(m.width, m.height)
	if !l.showQueue || len(upcoming) == 0 {
		m.queueFocused = false
		m.queueSel = 0
		m.queueScroll = 0
		return
	}
	if m.queueSel < 0 {
		m.queueSel = 0
	}
	if m.queueSel >= len(upcoming) {
		m.queueSel = len(upcoming) - 1
	}
	rows := l.queueRows - 4
	if rows < 1 {
		rows = 1
	}
	if m.queueSel < m.queueScroll {
		m.queueScroll = m.queueSel
	} else if m.queueSel >= m.queueScroll+rows {
		m.queueScroll = m.queueSel - rows + 1
	}
	if m.queueScroll < 0 {
		m.queueScroll = 0
	}
	maxScroll := len(upcoming) - rows
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.queueScroll > maxScroll {
		m.queueScroll = maxScroll
	}
}

func (m *Model) adjustScroll() {
	height := computeLayout(m.width, m.height).resultsH()
	idx := m.results.Index()
	count := len(m.results.Items())
	if count == 0 {
		m.scroll = 0
		return
	}
	if idx < m.scroll {
		m.scroll = idx
	} else if idx >= m.scroll+height {
		m.scroll = idx - height + 1
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
	if m.scroll > count-1 {
		m.scroll = count - 1
	}
}

func (m *Model) startPlayback() tea.Cmd {
	r, ok := m.selectedResult()
	if !ok {
		return nil
	}
	m.lastError = nil
	track := &player.Track{
		Title:   r.Title,
		Channel: r.Channel,
		Source:  r.VideoURL,
	}

	if m.queue.Current() == nil {
		m.queue.StartCurrent(track)
		m.queueFocused = false
		m.queueSel = 0
		return tea.Batch(m.playCurrent(), m.pushNotif("Playing: "+r.Title))
	}

	m.queue.AddUpcoming(track)
	m.clampQueueSelection()
	m.pushNotif("Added to queue")
	log.Default().Infof("track added to queue: %q", track.Title)
	return nil
}

func (m *Model) playCurrent() tea.Cmd {
	cur := m.queue.Current()
	if cur == nil {
		return nil
	}
	log.Default().Infof("starting playback for %q", cur.Title)
	return playCmd(m.player, cur.Source)
}

func (m *Model) nextTrack() tea.Cmd {
	next, ok := m.queue.Next()
	if !ok {
		return nil
	}
	if m.queueFocused && m.queueSel > 0 {
		m.queueSel--
	}
	m.clampQueueSelection()
	log.Default().Infof("next track selected: %q", next.Title)
	return tea.Batch(m.playCurrent(), m.pushNotif("Next: "+next.Title))
}

func (m *Model) prevTrack() tea.Cmd {
	if m.queue.Current() == nil {
		return nil
	}
	switch m.queue.PreviousPressed() {
	case queue.PrevRestart:
		log.Default().Infof("restarting current track: %q", m.queue.Current().Title)
		return tea.Batch(m.playCurrent(), m.pushNotif("Restarting"))
	case queue.PrevNavigate:
		if m.queueFocused {
			upcoming := m.queue.Upcoming()
			if len(upcoming) > 0 {
				m.queueSel = 0
			}
		}
		m.clampQueueSelection()
		log.Default().Infof("previous track selected: %q", m.queue.Current().Title)
		return tea.Batch(m.playCurrent(), m.pushNotif("Previous"))
	default:
		return nil
	}
}

func (m *Model) seek(seconds float64) tea.Cmd {
	if m.queue.Current() == nil {
		return nil
	}
	log.Default().Infof("seek %+.1fs", seconds)
	return tea.Batch(
		controlCmd(m.player, "seek", func() error { return m.player.Seek(seconds) }),
		m.pushNotif(fmt.Sprintf("Seek %+ds", int(seconds))),
	)
}

func (m *Model) changeVolume(delta int) tea.Cmd {
	cur := m.state.Volume
	target := cur + delta
	if target < 0 {
		target = 0
	}
	if target > 100 {
		target = 100
	}
	log.Default().Infof("volume %d -> %d", cur, target)
	return tea.Batch(
		controlCmd(m.player, "volume", func() error { return m.player.SetVolume(target) }),
		m.pushNotif(fmt.Sprintf("Volume %d%%", target)),
	)
}

func (m *Model) submitSearch(query string) tea.Cmd {
	m.query = query
	m.searching = true
	m.lastError = nil
	m.mode = modeIdle
	m.searchGen++
	gen := m.searchGen
	log.Default().Infof("search started for %q (gen %d)", query, gen)
	return tea.Batch(
		runSearch(m.ctx, m.provider, query, gen),
		startSpinner(),
		m.pushNotif("Searching…"),
	)
}

func (m Model) selectedResult() (search.Result, bool) {
	if m.resultsList == nil || m.results.Index() < 0 || m.results.Index() >= len(m.resultsList) {
		return search.Result{}, false
	}
	return m.resultsList[m.results.Index()], true
}

func (m *Model) previousMode() mode {
	if m.resultsList != nil {
		return modeResults
	}
	return modeIdle
}

func (m *Model) setListItems(results []search.Result) {
	items := make([]list.Item, len(results))
	for i, r := range results {
		items[i] = Item{Result: r}
	}
	m.results.SetItems(items)
	m.scroll = 0
	if len(items) > 0 {
		m.results.Select(0)
		m.selAnim.pos = 0
		m.selAnim.vel = 0
		m.selAnim.target = 0
		m.selAnim.active = false
	}
}

func (m *Model) layout() tea.Cmd {
	if !m.ready {
		m.results = list.New(nil, resultDelegate{}, m.width, 0)
		m.results.SetShowTitle(false)
		m.results.SetShowStatusBar(false)
		m.results.SetShowHelp(false)
		m.results.SetShowPagination(false)
		m.ready = true
	}

	l := computeLayout(m.width, m.height)
	m.results.SetSize(l.resultsW(), l.resultsH())
	return nil
}

func (m *Model) isPlayingURL(url string) bool {
	cur := m.queue.Current()
	return cur != nil && cur.Source == url
}
