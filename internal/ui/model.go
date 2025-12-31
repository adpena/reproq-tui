package ui

import (
	"context"
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
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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

	eventsEnabled bool
	eventsBuffer  *events.Buffer
	eventsCh      chan models.Event
	eventsBaseURL string
	eventsURL     string
	eventsCancel  context.CancelFunc
	ctx           context.Context
	cancel        context.CancelFunc

	toast       string
	toastExpiry time.Time
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
		metrics.MetricQueueDepth:       metrics.NewRingBuffer(capacity),
		metrics.MetricTasksTotal:       metrics.NewRingBuffer(capacity),
		metrics.MetricTasksFailed:      metrics.NewRingBuffer(capacity),
		metrics.MetricTasksRunning:     metrics.NewRingBuffer(capacity),
		metrics.MetricWorkerCount:      metrics.NewRingBuffer(capacity),
		metrics.MetricConcurrencyInUse: metrics.NewRingBuffer(capacity),
		metrics.MetricConcurrencyLimit: metrics.NewRingBuffer(capacity),
		metrics.MetricLatencyP95:       metrics.NewRingBuffer(capacity),
		seriesThroughput:               metrics.NewRingBuffer(capacity),
		seriesErrors:                   metrics.NewRingBuffer(capacity),
	}

	filter := textinput.New()
	filter.Placeholder = "filter events (queue, task, worker)"
	filter.CharLimit = 64
	filter.Width = 36

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

	return &Model{
		cfg:               cfg,
		client:            httpClient,
		catalog:           catalog,
		theme:             theme.Resolve(cfg.Theme),
		keymap:            newKeyMap(),
		help:              help.New(),
		filterInput:       filter,
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
	}
}

func (m *Model) Init() tea.Cmd {
	if m.eventsEnabled {
		m.startEvents()
	}
	cmds := []tea.Cmd{
		pollMetricsCmd(m.cfg, m.client, m.catalog),
		pollHealthCmd(m.cfg, m.client),
	}
	if m.statsEnabled {
		cmds = append(cmds, pollStatsCmd(m.cfg, m.client))
	}
	if m.eventsEnabled {
		cmds = append(cmds, listenEventsCmd(m.eventsCh))
	}
	return tea.Batch(cmds...)
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
	if !m.eventsEnabled || m.eventsBaseURL == "" {
		return
	}
	m.restartEvents(m.eventsBaseURL, true)
}

func (m *Model) restartEvents(url string, clear bool) {
	if url == "" {
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

func (m *Model) currentWindow() time.Duration {
	if m.windowIndex < 0 || m.windowIndex >= len(m.windowOptions) {
		return 5 * time.Minute
	}
	return m.windowOptions[m.windowIndex]
}
