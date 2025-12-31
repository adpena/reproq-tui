package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adpena/reproq-tui/internal/auth"
	"github.com/adpena/reproq-tui/internal/config"
	"github.com/adpena/reproq-tui/internal/health"
	"github.com/adpena/reproq-tui/internal/metrics"
	"github.com/adpena/reproq-tui/internal/stats"
	"github.com/adpena/reproq-tui/internal/theme"
	"github.com/adpena/reproq-tui/pkg/client"
	"github.com/adpena/reproq-tui/pkg/models"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type metricsMsg struct {
	snapshot  models.MetricSnapshot
	err       error
	attempted time.Time
	latency   time.Duration
}

type healthMsg struct {
	status models.HealthStatus
	err    error
}

type metricsTickMsg struct{}
type healthTickMsg struct{}

type statsMsg struct {
	stats     models.DjangoStats
	err       error
	attempted time.Time
	latency   time.Duration
}

type statsTickMsg struct{}

type eventMsg struct {
	event models.Event
}

type authPairMsg struct {
	pair auth.Pairing
	err  error
}

type authStatusMsg struct {
	status auth.PairStatus
	err    error
}

type authTickMsg struct{}

type snapshotSavedMsg struct {
	path string
	err  error
}

type toastClearMsg struct{}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.authURLActive {
			return m.handleAuthURLInput(msg)
		}
		if m.filterActive {
			return m.handleFilterInput(msg)
		}
		return m.handleKey(msg)
	case metricsMsg:
		m.lastScrapeAt = msg.attempted
		m.lastScrapeDelay = msg.latency
		m.lastScrapeErr = msg.err
		autoLogin := m.noteAuthError(msg.err)
		if msg.err == nil {
			m.lastSnapshot = msg.snapshot
			m.applySnapshot(msg.snapshot)
			m.authNeeded = false
			m.authErr = nil
		}
		if !m.paused {
			tick := tea.Tick(m.cfg.Interval, func(time.Time) tea.Msg {
				return metricsTickMsg{}
			})
			if autoLogin {
				return m, tea.Batch(tick, startAuthCmd(m.cfg, m.client))
			}
			return m, tick
		}
		if autoLogin {
			return m, startAuthCmd(m.cfg, m.client)
		}
		return m, nil
	case healthMsg:
		m.lastHealth = msg.status
		m.lastHealthErr = msg.err
		autoLogin := m.noteAuthError(msg.err)
		if !m.paused {
			tick := tea.Tick(m.cfg.HealthInterval, func(time.Time) tea.Msg {
				return healthTickMsg{}
			})
			if autoLogin {
				return m, tea.Batch(tick, startAuthCmd(m.cfg, m.client))
			}
			return m, tick
		}
		if autoLogin {
			return m, startAuthCmd(m.cfg, m.client)
		}
		return m, nil
	case statsMsg:
		m.lastStatsAt = msg.attempted
		m.lastStatsDelay = msg.latency
		m.lastStatsErr = msg.err
		autoLogin := m.noteAuthError(msg.err)
		if msg.err == nil {
			m.lastStats = msg.stats
		}
		if !m.paused {
			tick := tea.Tick(m.cfg.StatsInterval, func(time.Time) tea.Msg {
				return statsTickMsg{}
			})
			if autoLogin {
				return m, tea.Batch(tick, startAuthCmd(m.cfg, m.client))
			}
			return m, tick
		}
		if autoLogin {
			return m, startAuthCmd(m.cfg, m.client)
		}
		return m, nil
	case metricsTickMsg:
		if m.paused {
			return m, nil
		}
		return m, pollMetricsCmd(m.cfg, m.client, m.catalog)
	case healthTickMsg:
		if m.paused || m.cfg.WorkerHealthURL == "" {
			return m, nil
		}
		return m, pollHealthCmd(m.cfg, m.client)
	case statsTickMsg:
		if m.paused || !m.statsEnabled {
			return m, nil
		}
		return m, pollStatsCmd(m.cfg, m.client)
	case eventMsg:
		if m.eventsEnabled {
			m.eventsBuffer.Add(msg.event)
			return m, listenEventsCmd(m.eventsCh)
		}
		return m, nil
	case authPairMsg:
		if msg.err != nil {
			m.authErr = msg.err
			m.authFlowActive = false
			m.authPair = auth.Pairing{}
			m.toast = fmt.Sprintf("Auth start failed: %v", msg.err)
			m.toastExpiry = time.Now().Add(3 * time.Second)
			return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return toastClearMsg{}
			})
		}
		m.authPair = msg.pair
		m.authFlowActive = true
		m.authErr = nil
		return m, tea.Tick(time.Second, func(time.Time) tea.Msg {
			return authTickMsg{}
		})
	case authTickMsg:
		if !m.authFlowActive {
			return m, nil
		}
		return m, pollAuthCmd(m.cfg, m.client, m.authPair.Code)
	case authStatusMsg:
		if msg.err != nil {
			if client.IsStatus(msg.err, http.StatusNotFound) {
				m.authFlowActive = false
				m.authPair = auth.Pairing{}
				m.toast = "Auth link expired"
			} else {
				m.authErr = msg.err
				m.authFlowActive = false
				m.authPair = auth.Pairing{}
				m.toast = fmt.Sprintf("Auth failed: %v", msg.err)
			}
			m.toastExpiry = time.Now().Add(3 * time.Second)
			return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return toastClearMsg{}
			})
		}
		switch msg.status.Status {
		case "approved":
			token := auth.Token{Value: msg.status.Token, ExpiresAt: msg.status.ExpiresAt, DjangoURL: m.cfg.DjangoURL}
			if err := m.applyAuthToken(token); err != nil {
				m.toast = fmt.Sprintf("Auth save failed: %v", err)
			} else {
				m.toast = "Signed in"
			}
			m.authFlowActive = false
			m.authPair = auth.Pairing{}
			m.toastExpiry = time.Now().Add(3 * time.Second)
			cmds := []tea.Cmd{
				tea.Tick(3*time.Second, func(time.Time) tea.Msg {
					return toastClearMsg{}
				}),
			}
			if !m.paused {
				cmds = append(cmds, pollMetricsCmd(m.cfg, m.client, m.catalog))
				cmds = append(cmds, pollHealthCmd(m.cfg, m.client))
				if m.statsEnabled {
					cmds = append(cmds, pollStatsCmd(m.cfg, m.client))
				}
			}
			return m, tea.Batch(cmds...)
		case "pending":
			return m, tea.Tick(time.Second, func(time.Time) tea.Msg {
				return authTickMsg{}
			})
		default:
			m.authFlowActive = false
			m.authPair = auth.Pairing{}
			m.toast = "Auth link expired"
			m.toastExpiry = time.Now().Add(3 * time.Second)
			return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return toastClearMsg{}
			})
		}
	case snapshotSavedMsg:
		if msg.err != nil {
			m.toast = fmt.Sprintf("Snapshot failed: %v", msg.err)
		} else {
			m.toast = fmt.Sprintf("Snapshot saved: %s", filepath.Base(msg.path))
		}
		m.toastExpiry = time.Now().Add(3 * time.Second)
		return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return toastClearMsg{}
		})
	case toastClearMsg:
		if time.Now().After(m.toastExpiry) {
			m.toast = ""
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.filterActive = false
		return m, nil
	case tea.KeyEnter:
		raw := strings.TrimSpace(m.filterInput.Value())
		parsed := parseSSEFilter(raw)
		m.filter = raw
		m.filterLocal = parsed.local
		m.applyEventFilter(parsed)
		m.filterActive = false
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	return m, cmd
}

func (m *Model) handleAuthURLInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.authURLActive = false
		m.authURLNotice = ""
		m.authURLInput.Blur()
		return m, nil
	case tea.KeyEnter:
		raw := strings.TrimSpace(m.authURLInput.Value())
		if raw == "" {
			m.authURLActive = false
			m.authURLNotice = ""
			m.authURLInput.Blur()
			return m, nil
		}
		normalized, err := normalizeDjangoURLInput(raw)
		if err != nil {
			m.authURLNotice = "Enter a valid Django URL (https:// optional)"
			return m, nil
		}
		m.cfg.DjangoURL = normalized
		if m.cfg.DjangoStatsURL == "" {
			m.cfg.DjangoStatsURL = config.DeriveDjangoStatsURL(normalized)
		}
		m.authEnabled = strings.TrimSpace(m.cfg.DjangoURL) != ""
		m.statsEnabled = m.cfg.DjangoStatsURL != ""
		m.authURLActive = false
		m.authURLNotice = ""
		m.authURLInput.Blur()
		m.authFlowActive = true
		m.authPair = auth.Pairing{}
		return m, startAuthCmd(m.cfg, m.client)
	}
	var cmd tea.Cmd
	m.authURLInput, cmd = m.authURLInput.Update(msg)
	return m, cmd
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keymap.Quit) {
		m.Close()
		return m, tea.Quit
	}
	if m.authFlowActive {
		if msg.Type == tea.KeyEsc || key.Matches(msg, m.keymap.Auth) {
			m.authFlowActive = false
			m.authPair = auth.Pairing{}
			m.toast = "Auth canceled"
			m.toastExpiry = time.Now().Add(2 * time.Second)
			return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
				return toastClearMsg{}
			})
		}
	}
	if m.detailActive {
		switch {
		case msg.Type == tea.KeyEsc || key.Matches(msg, m.keymap.Drilldown):
			m.detailActive = false
			return m, nil
		case key.Matches(msg, m.keymap.FocusNext):
			m.detailIndex = (m.detailIndex + 1) % len(m.detailViews)
			return m, nil
		}
	}
	switch {
	case key.Matches(msg, m.keymap.Help):
		m.showHelp = !m.showHelp
		if m.showHelp {
			m.detailActive = false
		}
		return m, nil
	case key.Matches(msg, m.keymap.Pause):
		m.paused = !m.paused
		if !m.paused {
			cmds := []tea.Cmd{
				pollMetricsCmd(m.cfg, m.client, m.catalog),
				pollHealthCmd(m.cfg, m.client),
			}
			if m.statsEnabled {
				cmds = append(cmds, pollStatsCmd(m.cfg, m.client))
			}
			return m, tea.Batch(cmds...)
		}
		return m, nil
	case key.Matches(msg, m.keymap.Refresh):
		cmds := []tea.Cmd{
			pollMetricsCmd(m.cfg, m.client, m.catalog),
			pollHealthCmd(m.cfg, m.client),
		}
		if m.statsEnabled {
			cmds = append(cmds, pollStatsCmd(m.cfg, m.client))
		}
		return m, tea.Batch(cmds...)
	case key.Matches(msg, m.keymap.WindowShort):
		m.windowIndex = 0
		return m, nil
	case key.Matches(msg, m.keymap.WindowMid):
		m.windowIndex = 1
		return m, nil
	case key.Matches(msg, m.keymap.WindowLong):
		m.windowIndex = 2
		return m, nil
	case key.Matches(msg, m.keymap.FocusNext):
		m.focus = (m.focus + 1) % 3
		if !m.showEvents && m.focus == focusRight {
			m.focus = focusLeft
		}
		return m, nil
	case key.Matches(msg, m.keymap.Filter):
		m.filterActive = true
		m.filterInput.SetValue(m.filter)
		m.filterInput.CursorEnd()
		return m, nil
	case key.Matches(msg, m.keymap.ToggleEvents):
		m.showEvents = !m.showEvents
		if !m.showEvents && m.focus == focusRight {
			m.focus = focusLeft
		}
		return m, nil
	case key.Matches(msg, m.keymap.ToggleTheme):
		m.cfg.Theme = toggleTheme(m.cfg.Theme)
		m.theme = theme.Resolve(m.cfg.Theme)
		return m, nil
	case key.Matches(msg, m.keymap.Snapshot):
		return m, exportSnapshotCmd(m)
	case key.Matches(msg, m.keymap.Drilldown):
		m.detailActive = !m.detailActive
		if m.detailActive {
			m.showHelp = false
		}
		return m, nil
	case key.Matches(msg, m.keymap.Auth):
		if !m.authHeaderManaged {
			m.toast = "Auth is configured via headers"
			m.toastExpiry = time.Now().Add(3 * time.Second)
			return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return toastClearMsg{}
			})
		}
		if m.authToken.Value != "" {
			if err := m.clearAuthToken(); err != nil {
				m.toast = fmt.Sprintf("Sign out failed: %v", err)
			} else {
				m.toast = "Signed out"
			}
			m.toastExpiry = time.Now().Add(3 * time.Second)
			return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return toastClearMsg{}
			})
		}
		if !m.authEnabled {
			m.authURLActive = true
			m.authURLNotice = ""
			m.authURLInput.SetValue("")
			m.authURLInput.CursorEnd()
			m.authURLInput.Focus()
			return m, nil
		}
		m.authFlowActive = true
		m.authPair = auth.Pairing{}
		return m, startAuthCmd(m.cfg, m.client)
	}
	return m, nil
}

func toggleTheme(current string) string {
	switch strings.ToLower(current) {
	case "dark":
		return "light"
	case "light":
		return "auto"
	default:
		return "dark"
	}
}

func normalizeDjangoURLInput(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("empty url")
	}
	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	trimmed = strings.TrimSpace(trimmed)
	lowerTrimmed := strings.ToLower(trimmed)
	if strings.HasPrefix(lowerTrimmed, "http:/") && !strings.HasPrefix(lowerTrimmed, "http://") {
		trimmed = "http://" + trimmed[len("http:/"):]
		lowerTrimmed = "http://" + lowerTrimmed[len("http:/"):]
	}
	if strings.HasPrefix(lowerTrimmed, "https:/") && !strings.HasPrefix(lowerTrimmed, "https://") {
		trimmed = "https://" + trimmed[len("https:/"):]
		lowerTrimmed = "https://" + lowerTrimmed[len("https:/"):]
	}
	if !strings.Contains(trimmed, "://") {
		scheme := "https://"
		host := strings.TrimSpace(strings.SplitN(trimmed, "/", 2)[0])
		hostOnly := host
		if strings.Contains(hostOnly, ":") {
			if parsedHost, _, err := net.SplitHostPort(hostOnly); err == nil {
				hostOnly = parsedHost
			}
		}
		lower := strings.ToLower(hostOnly)
		if lower == "localhost" || strings.HasPrefix(lower, "127.0.0.1") || strings.HasPrefix(lower, "0.0.0.0") {
			scheme = "http://"
		} else if ip := net.ParseIP(strings.Trim(lower, "[]")); ip != nil {
			if ip.IsLoopback() || ip.IsPrivate() {
				scheme = "http://"
			}
		}
		trimmed = scheme + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("invalid url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid scheme")
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	normalized := config.DeriveDjangoURL(parsed.String())
	if normalized == "" {
		return "", fmt.Errorf("invalid url")
	}
	return normalized, nil
}

func pollMetricsCmd(cfg config.Config, httpClient *client.Client, catalog metrics.Catalog) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()
		snapshot, err := metrics.Scrape(ctx, httpClient, cfg.WorkerMetricsURL, catalog)
		if err != nil {
			return metricsMsg{
				err:       err,
				attempted: time.Now(),
				latency:   time.Since(start),
			}
		}
		return metricsMsg{
			snapshot:  snapshot,
			attempted: snapshot.CollectedAt,
			latency:   snapshot.Latency,
		}
	}
}

func pollHealthCmd(cfg config.Config, httpClient *client.Client) tea.Cmd {
	return func() tea.Msg {
		if cfg.WorkerHealthURL == "" {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()
		status, err := health.Fetch(ctx, httpClient, cfg.WorkerHealthURL)
		return healthMsg{status: status, err: err}
	}
}

func pollStatsCmd(cfg config.Config, httpClient *client.Client) tea.Cmd {
	return func() tea.Msg {
		if cfg.DjangoStatsURL == "" {
			return nil
		}
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()
		statsSnapshot, err := stats.Fetch(ctx, httpClient, cfg.DjangoStatsURL)
		return statsMsg{
			stats:     statsSnapshot,
			err:       err,
			attempted: time.Now(),
			latency:   time.Since(start),
		}
	}
}

func listenEventsCmd(ch <-chan models.Event) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return nil
		}
		return eventMsg{event: event}
	}
}

func startAuthCmd(cfg config.Config, httpClient *client.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()
		pair, err := auth.StartPair(ctx, httpClient, cfg.DjangoURL)
		return authPairMsg{pair: pair, err: err}
	}
}

func pollAuthCmd(cfg config.Config, httpClient *client.Client, code string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()
		status, err := auth.CheckPair(ctx, httpClient, cfg.DjangoURL, code)
		return authStatusMsg{status: status, err: err}
	}
}

func (m *Model) applySnapshot(snapshot models.MetricSnapshot) {
	ts := snapshot.CollectedAt
	for key, value := range snapshot.Values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		if buf, ok := m.series[key]; ok {
			buf.Add(models.Sample{Timestamp: ts, Value: value})
		}
	}
	m.updateCounter(metrics.MetricTasksTotal, ts, seriesThroughput)
	m.updateCounter(metrics.MetricTasksFailed, ts, seriesErrors)
}

func (m *Model) applyAuthToken(token auth.Token) error {
	if m.authHeaderManaged {
		m.client.SetHeader("Authorization", "Bearer "+token.Value)
	}
	if token.DjangoURL == "" {
		token.DjangoURL = m.cfg.DjangoURL
	}
	m.authToken = token
	m.authNeeded = false
	m.authErr = nil
	if m.authStore != nil {
		if err := m.authStore.Save(token); err != nil {
			return err
		}
	}
	return nil
}

func (m *Model) clearAuthToken() error {
	if m.authStore != nil {
		if err := m.authStore.Clear(); err != nil {
			return err
		}
	}
	m.authToken = auth.Token{}
	m.authNeeded = false
	m.authErr = nil
	if m.authHeaderManaged {
		m.client.ClearHeader("Authorization")
	}
	return nil
}

func (m *Model) noteAuthError(err error) bool {
	if err == nil {
		return false
	}
	if !isAuthError(err) {
		return false
	}
	wasNeeded := m.authNeeded
	m.authNeeded = true
	m.authErr = err
	if m.authFlowActive {
		return false
	}
	if m.cfg.AutoLogin && m.authEnabled && m.authHeaderManaged && m.authToken.Value == "" {
		m.authFlowActive = true
		m.authPair = auth.Pairing{}
		return true
	}
	if !wasNeeded && m.authEnabled && m.authHeaderManaged && m.authToken.Value == "" {
		m.toast = "Auth required: press L to sign in"
		m.toastExpiry = time.Now().Add(3 * time.Second)
	}
	return false
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	return client.IsStatus(err, http.StatusUnauthorized, http.StatusForbidden)
}

func (m *Model) applyEventFilter(filter sseFilter) {
	if !m.eventsEnabled || m.eventsBaseURL == "" {
		return
	}
	nextURL := buildEventsURL(m.eventsBaseURL, filter)
	if nextURL == "" || nextURL == m.eventsURL {
		return
	}
	m.restartEvents(nextURL, true)
	m.toast = "Events filter applied"
	m.toastExpiry = time.Now().Add(2 * time.Second)
}

func (m *Model) updateCounter(key string, ts time.Time, derivedKey string) {
	buf, ok := m.series[key]
	if !ok {
		return
	}
	latest, ok := buf.Latest()
	if !ok {
		return
	}
	if prev, ok := m.lastCounters[key]; ok {
		delta := latest.Value - prev.Value
		if delta < 0 {
			delta = 0
		}
		elapsed := latest.Timestamp.Sub(prev.Timestamp).Seconds()
		if elapsed > 0 {
			rate := delta / elapsed
			if derived, ok := m.series[derivedKey]; ok {
				derived.Add(models.Sample{Timestamp: ts, Value: rate})
			}
		}
	}
	m.lastCounters[key] = latest
}

func exportSnapshotCmd(m *Model) tea.Cmd {
	data := buildSnapshot(m)
	return func() tea.Msg {
		payload, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return snapshotSavedMsg{err: err}
		}
		filename := fmt.Sprintf("reproq-tui-snapshot-%s.json", time.Now().Format("20060102-150405"))
		if err := os.WriteFile(filename, payload, 0o644); err != nil {
			return snapshotSavedMsg{err: err}
		}
		return snapshotSavedMsg{path: filename}
	}
}

type snapshotExport struct {
	GeneratedAt time.Time                  `json:"generated_at"`
	Window      string                     `json:"window"`
	Metrics     map[string]float64         `json:"metrics"`
	Health      models.HealthStatus        `json:"health"`
	Stats       *models.DjangoStats        `json:"stats,omitempty"`
	Events      []models.Event             `json:"events"`
	Series      map[string][]models.Sample `json:"series"`
}

func buildSnapshot(m *Model) snapshotExport {
	series := map[string][]models.Sample{}
	for key, buf := range m.series {
		samples := buf.Values()
		if len(samples) > 300 {
			samples = samples[len(samples)-300:]
		}
		series[key] = samples
	}
	events := m.eventsBuffer.Items()
	if len(events) > 100 {
		events = events[len(events)-100:]
	}
	metricsValues := map[string]float64{}
	for key, val := range m.lastSnapshot.Values {
		metricsValues[key] = val
	}
	metricsValues["throughput"] = m.currentThroughput()
	metricsValues["error_ratio"] = m.currentErrorRatio()

	return snapshotExport{
		GeneratedAt: time.Now(),
		Window:      m.currentWindow().String(),
		Metrics:     metricsValues,
		Health:      m.lastHealth,
		Stats:       m.statsSnapshot(),
		Events:      events,
		Series:      series,
	}
}
