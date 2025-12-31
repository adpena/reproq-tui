package ui

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/adpena/reproq-tui/internal/auth"
	"github.com/adpena/reproq-tui/internal/config"
	"github.com/adpena/reproq-tui/internal/events"
	"github.com/adpena/reproq-tui/internal/metrics"
	"github.com/adpena/reproq-tui/internal/theme"
	"github.com/adpena/reproq-tui/pkg/client"
	"github.com/adpena/reproq-tui/pkg/models"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	seriesThroughput = "throughput"
	seriesErrors     = "errors"
)

type focusPane int

const (
	focusLeft focusPane = iota
	focusCenter
	focusRight
)

type setupStage int

const (
	setupNone setupStage = iota
	setupDjango
	setupWorker
)

type Model struct {
	cfg     config.Config
	client  *client.Client
	catalog metrics.Catalog

	width  int
	height int

	theme        theme.Theme
	keymap       keyMap
	help         help.Model
	focus        focusPane
	showHelp     bool
	detailActive bool
	detailIndex  int
	detailViews  []string

	filterInput  textinput.Model
	filterActive bool
	filter       string
	filterLocal  string

	setupActive    bool
	setupStage     setupStage
	setupWorkerURL textinput.Model
	setupDjangoURL textinput.Model
	setupNotice    string
	setupPersist   bool
	setupPersisted bool

	windowOptions []time.Duration
	windowIndex   int

	showEvents bool
	paused     bool

	series       map[string]*metrics.RingBuffer
	lastCounters map[string]models.Sample

	lastSnapshot    models.MetricSnapshot
	lastScrapeErr   error
	lastScrapeAt    time.Time
	lastScrapeDelay time.Duration

	lastHealth    models.HealthStatus
	lastHealthErr error

	statsEnabled   bool
	lastStats      models.DjangoStats
	lastStatsErr   error
	lastStatsAt    time.Time
	lastStatsDelay time.Duration

	authURLInput  textinput.Model
	authURLActive bool
	authURLNotice string

	authEnabled       bool
	authHeaderManaged bool
	authStore         *auth.Store
	authToken         auth.Token
	authFlowActive    bool
	authPair          auth.Pairing
	authNeeded        bool
	authErr           error

	lowMemoryMode bool

	eventsEnabled bool
	eventsBuffer  *events.Buffer
	eventsCh      chan models.Event
	eventsBaseURL string
	eventsURL     string
	eventsCancel  context.CancelFunc
	lastEventAt   time.Time
	ctx           context.Context
	cancel        context.CancelFunc

	toast       string
	toastExpiry time.Time

	spinner spinner.Model

	safeTop int
}

func NewModel(cfg config.Config) *Model {
	httpClient := client.New(client.Options{
		Timeout:            cfg.Timeout,
		Headers:            cfg.Headers,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	})
	catalog := metrics.NewCatalog(cfg.Metrics)
	windowOptions := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute}
	windowIndex := 1
	for idx, option := range windowOptions {
		if cfg.Window == option {
			windowIndex = idx
			break
		}
	}
	maxWindow := windowOptions[len(windowOptions)-1]
	interval := cfg.Interval
	if interval <= 0 {
		interval = time.Second
	}
	capacity := int(maxWindow/interval) + 5
	if capacity < 30 {
		capacity = 30
	}
	series := map[string]*metrics.RingBuffer{
		metrics.MetricQueueDepth:        metrics.NewRingBuffer(capacity),
		metrics.MetricTasksTotal:        metrics.NewRingBuffer(capacity),
		metrics.MetricTasksFailed:       metrics.NewRingBuffer(capacity),
		metrics.MetricTasksRunning:      metrics.NewRingBuffer(capacity),
		metrics.MetricWorkerCount:       metrics.NewRingBuffer(capacity),
		metrics.MetricConcurrencyInUse:  metrics.NewRingBuffer(capacity),
		metrics.MetricConcurrencyLimit:  metrics.NewRingBuffer(capacity),
		metrics.MetricLatencyP95:        metrics.NewRingBuffer(capacity),
		metrics.MetricWorkerMemUsage:    metrics.NewRingBuffer(capacity),
		metrics.MetricDBPoolConnections: metrics.NewRingBuffer(capacity),
		metrics.MetricDBPoolWait:        metrics.NewRingBuffer(capacity),
		seriesThroughput:                metrics.NewRingBuffer(capacity),
		seriesErrors:                    metrics.NewRingBuffer(capacity),
	}

	filter := textinput.New()
	filter.Placeholder = "filter events (queue, task, worker)"
	filter.CharLimit = 64
	filter.Width = 36

	setupWorkerInput := textinput.New()
	setupWorkerInput.Placeholder = "http://localhost:9100 or /metrics"
	setupWorkerInput.CharLimit = 200
	setupWorkerInput.Width = 52

	setupDjangoInput := textinput.New()
	setupDjangoInput.Placeholder = "https://django.example.com"
	setupDjangoInput.CharLimit = 200
	setupDjangoInput.Width = 52

	authURL := textinput.New()
	authURL.Placeholder = "django.example.com"
	authURL.CharLimit = 200
	authURL.Width = 52

	ctx, cancel := context.WithCancel(context.Background())
	authStore := auth.DefaultStore()
	authHeaderManaged := !httpClient.HasHeader("Authorization")
	authToken := auth.Token{}
	if authHeaderManaged {
		if stored, err := authStore.Load(); err == nil {
			if strings.TrimSpace(cfg.DjangoURL) == "" && stored.DjangoURL != "" {
				cfg.DjangoURL = stored.DjangoURL
				if cfg.DjangoStatsURL == "" {
					cfg.DjangoStatsURL = config.DeriveDjangoStatsURL(cfg.DjangoURL)
				}
			}
			if stored.Valid(time.Now()) {
				httpClient.SetHeader("Authorization", "Bearer "+stored.Value)
				authToken = stored
			} else if stored.Value != "" {
				authToken = stored
				if stored.DjangoURL == "" {
					_ = authStore.Clear()
				}
			}
		}
	}
	authEnabled := strings.TrimSpace(cfg.DjangoURL) != ""
	if cfg.WorkerURL != "" {
		setupWorkerInput.SetValue(cfg.WorkerURL)
	}
	if cfg.DjangoURL != "" {
		setupDjangoInput.SetValue(cfg.DjangoURL)
	}
	stage := setupNone
	if strings.TrimSpace(cfg.WorkerMetricsURL) == "" && strings.TrimSpace(cfg.WorkerURL) == "" {
		if strings.TrimSpace(cfg.DjangoURL) == "" {
			stage = setupDjango
		} else {
			stage = setupWorker
		}
	}
	setupActive := stage != setupNone
	if stage == setupDjango {
		setupDjangoInput.Focus()
	} else if stage == setupWorker {
		setupWorkerInput.Focus()
	}

	model := &Model{
		cfg:               cfg,
		client:            httpClient,
		catalog:           catalog,
		theme:             theme.Resolve(cfg.Theme),
		keymap:            newKeyMap(),
		help:              help.New(),
		filterInput:       filter,
		setupActive:       setupActive,
		setupStage:        stage,
		setupWorkerURL:    setupWorkerInput,
		setupDjangoURL:    setupDjangoInput,
		setupPersist:      setupActive,
		windowOptions:     windowOptions,
		windowIndex:       windowIndex,
		showEvents:        true,
		detailViews:       []string{"Queues", "Workers", "Periodic", "Tasks", "Errors"},
		series:            series,
		lastCounters:      map[string]models.Sample{},
		statsEnabled:      cfg.DjangoStatsURL != "",
		authURLInput:      authURL,
		authEnabled:       authEnabled,
		authStore:         authStore,
		authToken:         authToken,
		authHeaderManaged: authHeaderManaged,
		eventsEnabled:     cfg.EventsURL != "",
		eventsBuffer:      events.NewBuffer(200),
		eventsCh:          make(chan models.Event, 50),
		eventsBaseURL:     cfg.EventsURL,
		eventsURL:         cfg.EventsURL,
		ctx:               ctx,
		cancel:            cancel,
		spinner:           spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(theme.Resolve(cfg.Theme).Palette.Accent))),
		safeTop:           safeTopPadding(),
	}
	model.applyInputStyles()
	return model
}

func (m *Model) Init() tea.Cmd {
	if m.setupActive {
		if m.setupStage == setupWorker && strings.TrimSpace(m.cfg.DjangoURL) != "" {
			return fetchTUIConfigCmd(m.cfg, m.client)
		}
		return nil
	}
	if m.eventsEnabled {
		m.startEvents()
	}
	return tea.Batch(m.spinner.Tick, m.startPollingCmds())
}

func (m *Model) applyInputStyles() {
	set := func(input *textinput.Model) {
		input.Prompt = ""
		input.PromptStyle = lipgloss.NewStyle().Foreground(m.theme.Palette.Muted)
		input.TextStyle = lipgloss.NewStyle().Foreground(m.theme.Palette.Text)
		input.PlaceholderStyle = lipgloss.NewStyle().Foreground(m.theme.Palette.Muted)
		input.Cursor.Style = lipgloss.NewStyle().Foreground(m.theme.Palette.Accent)
	}
	set(&m.filterInput)
	set(&m.setupWorkerURL)
	set(&m.setupDjangoURL)
	set(&m.authURLInput)
}

func (m *Model) Close() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.eventsCancel != nil {
		m.eventsCancel()
	}
}

func (m *Model) startEvents() {
	if m.lowMemoryMode || !m.eventsEnabled || m.eventsBaseURL == "" {
		return
	}
	m.restartEvents(m.eventsBaseURL, true)
}

func (m *Model) startPollingCmds() tea.Cmd {
	cmds := []tea.Cmd{}
	if m.cfg.WorkerMetricsURL != "" {
		cmds = append(cmds, pollMetricsCmd(m.cfg, m.client, m.catalog))
	}
	if m.cfg.WorkerHealthURL != "" {
		cmds = append(cmds, pollHealthCmd(m.cfg, m.client))
	}
	if m.statsEnabled {
		cmds = append(cmds, pollStatsCmd(m.cfg, m.client))
	}
	if m.eventsEnabled {
		cmds = append(cmds, listenEventsCmd(m.eventsCh))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m *Model) restartEvents(url string, clear bool) {
	if m.lowMemoryMode || url == "" {
		return
	}
	if m.eventsCancel != nil {
		m.eventsCancel()
	}
	if clear {
		m.eventsBuffer.Clear()
	}
	ctx := m.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	eventsCtx, cancel := context.WithCancel(ctx)
	m.eventsCancel = cancel
	m.eventsURL = url
	go events.Listen(eventsCtx, m.client, url, m.eventsCh)
}

func (m *Model) applyLowMemoryMode(enabled bool) {
	if !enabled || m.lowMemoryMode {
		return
	}
	m.lowMemoryMode = true
	if m.eventsCancel != nil {
		m.eventsCancel()
		m.eventsCancel = nil
	}
	m.eventsEnabled = false
	m.eventsBaseURL = ""
	m.eventsURL = ""
	if m.eventsBuffer != nil {
		m.eventsBuffer.Clear()
	}
}

func (m *Model) currentWindow() time.Duration {
	if m.windowIndex < 0 || m.windowIndex >= len(m.windowOptions) {
		return 5 * time.Minute
	}
	return m.windowOptions[m.windowIndex]
}

func safeTopPadding() int {
	if val := strings.TrimSpace(os.Getenv("REPROQ_TUI_SAFE_TOP")); val != "" {
		return parseSafeTop(val)
	}
	termProgram := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	if termProgram == "iterm.app" && os.Getenv("ITERM_SESSION_ID") != "" {
		return 1
	}
	if termProgram == "ghostty" {
		return 1
	}
	return 0
}

func parseSafeTop(val string) int {
	val = strings.ToLower(strings.TrimSpace(val))
	switch val {
	case "1", "true", "yes", "y", "on":
		return 1
	case "0", "false", "no", "n", "off":
		return 0
	}
	if parsed, err := strconv.Atoi(val); err == nil {
		if parsed < 0 {
			return 0
		}
		return parsed
	}
	return 0
}
