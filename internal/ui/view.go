package ui

import (
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/adpena/reproq-tui/internal/charts"
	"github.com/adpena/reproq-tui/internal/metrics"
	"github.com/adpena/reproq-tui/pkg/models"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading..."
	}
	if m.authURLActive {
		return m.applySafeTop(m.renderAuthURLPrompt())
	}
	if m.authFlowActive {
		return m.applySafeTop(m.renderAuthPrompt())
	}
	if m.setupActive {
		return m.applySafeTop(m.renderSetup())
	}
	if m.detailActive {
		return m.applySafeTop(m.renderDetails())
	}
	if m.showHelp {
		return m.applySafeTop(m.renderHelp())
	}
	statusBar := m.renderStatusBar()
	hero := m.renderHero()
	footer := m.renderFooter()

	height := m.contentHeight()
	mainHeight := height - lipgloss.Height(statusBar) - lipgloss.Height(hero) - lipgloss.Height(footer)
	if mainHeight < 5 {
		mainHeight = 5
	}
	main := m.renderMain(mainHeight)
	content := lipgloss.JoinVertical(lipgloss.Left, statusBar, hero, main, footer)
	return m.applySafeTop(content)
}

func (m *Model) renderHero() string {
	throughput := m.currentThroughput()
	success := m.currentSuccessRatio()
	errors := m.latestValue(seriesErrors)
	latency := m.currentLatencyP95()

	segments := []string{
		m.heroSegment("THROUGHPUT", formatRate(throughput), m.theme.Styles.Accent),
		m.heroSegment("AVAILABILITY", formatPercent(success), m.theme.Styles.AccentAlt),
		m.heroSegment("FAILURE RATE", formatRate(errors), m.theme.Styles.StatusDown),
		m.heroSegment("P95 LATENCY", formatDuration(time.Duration(latency*float64(time.Second))), m.theme.Styles.Muted),
	}

	hero := lipgloss.JoinHorizontal(lipgloss.Top, segments...)
	return lipgloss.NewStyle().
		MarginBottom(1).
		Padding(0, 2).
		Width(m.width).
		Render(hero)
}

func (m *Model) heroSegment(label, value string, style lipgloss.Style) string {
	l := m.theme.Styles.Muted.Render(label)
	v := style.Bold(true).Render(value)
	return lipgloss.JoinVertical(lipgloss.Left, l, v) + strings.Repeat(" ", 10)
}

func (m *Model) renderSetup() string {
	width := maxInt(44, minInt(92, m.width-6))
	title := m.theme.Styles.CardTitle.Render("Connect Reproq TUI")
	lines := []string{title, ""}
	if m.setupStage == setupDjango {
		lines = append(lines,
			m.theme.Styles.Badge.Render("Step 1 of 2"),
			"Paste your Django 6.0 app URL (https optional):",
			m.setupDjangoURL.View(),
		)
	} else {
		lines = append(lines,
			m.theme.Styles.Badge.Render("Step 2 of 2"),
			"Worker URL or full /metrics URL:",
			m.setupWorkerURL.View(),
		)
	}
	if m.setupNotice != "" {
		lines = append(lines, "", m.theme.Styles.StatusWarn.Render(m.setupNotice))
	}
	footer := "enter to continue | tab to switch | esc to quit"
	if m.setupStage == setupDjango {
		footer = "enter to continue | tab to skip | esc to quit"
	}
	lines = append(lines, "", m.theme.Styles.Muted.Render(footer))
	content := strings.Join(lines, "\n")
	card := m.theme.Styles.Card.Width(width).Render(content)
	return m.placeCentered(card)
}

func (m *Model) renderHelp() string {
	content := m.help.FullHelpView(m.keymap.FullHelp())
	width := maxInt(20, minInt(60, m.width-4))
	card := m.theme.Styles.Card.Width(width).Render(content)
	return m.placeCentered(card)
}

func (m *Model) renderAuthURLPrompt() string {
	width := maxInt(40, minInt(90, m.width-6))
	lines := []string{
		m.theme.Styles.CardTitle.Render("Connect Reproq TUI"),
		"",
		"Paste the Django 6.0 app URL (https:// optional):",
		m.authURLInput.View(),
	}
	if m.authURLNotice != "" {
		lines = append(lines, "", m.theme.Styles.StatusWarn.Render(m.authURLNotice))
	}
	lines = append(lines, "", m.theme.Styles.Muted.Render("enter to continue | esc to cancel"))
	content := strings.Join(lines, "\n")
	card := m.theme.Styles.Card.Width(width).Render(content)
	return m.placeCentered(card)
}

func (m *Model) renderAuthPrompt() string {
	width := maxInt(30, minInt(80, m.width-6))
	maxLine := maxInt(10, width-4)
	lines := []string{
		m.theme.Styles.CardTitle.Render("Connect Reproq TUI"),
		"",
		"1) Open this URL:",
	}
	if m.authPair.VerifyURL != "" {
		lines = append(lines, m.theme.Styles.AccentAlt.Render(truncate(m.authPair.VerifyURL, maxLine)))
	} else {
		lines = append(lines, m.theme.Styles.Muted.Render("Starting login..."))
	}
	lines = append(lines, "", "2) Sign in as a superuser and approve.")
	if m.authPair.Code != "" {
		lines = append(lines, fmt.Sprintf("Code: %s", m.theme.Styles.Accent.Render(m.authPair.Code)))
	}
	if !m.authPair.ExpiresAt.IsZero() {
		lines = append(lines, m.theme.Styles.Muted.Render("Expires at "+m.authPair.ExpiresAt.Format("15:04:05")))
	}
	if m.authErr != nil {
		if isAuthError(m.authErr) {
			lines = append(lines, "", m.theme.Styles.Muted.Render("Waiting for approval..."))
		} else {
			lines = append(lines, "", m.theme.Styles.StatusWarn.Render(fmt.Sprintf("Error: %v", m.authErr)))
		}
	}
	lines = append(lines, "", m.theme.Styles.Muted.Render("esc to cancel"))
	content := strings.Join(lines, "\n")
	card := m.theme.Styles.Card.Width(width).Render(content)
	return m.placeCentered(card)
}

func (m *Model) renderDetails() string {
	view := m.detailViews[m.detailIndex%len(m.detailViews)]
	title := fmt.Sprintf("Details: %s", view)
	body := m.detailBody(view)
	footer := m.theme.Styles.Muted.Render("tab to switch | esc to close")
	content := strings.Join([]string{m.theme.Styles.CardTitle.Render(title), body, "", footer}, "\n")
	width := maxInt(30, minInt(70, m.width-6))
	available := m.contentHeight()
	height := maxInt(10, minInt(20, available-6))
	card := m.theme.Styles.Card.Width(width).Height(height).Render(content)
	return m.placeCentered(card)
}

func (m *Model) detailBody(view string) string {
	switch view {
	case "Queues":
		queueDepth := m.queueDepthValue()
		trend := m.queueTrend()
		lines := []string{
			m.labelValue("Total depth", formatNumber(queueDepth)),
			m.labelValue("Trend (1m)", formatNumber(trend)),
		}
		if ready, ok := m.statsTaskCount("READY"); ok {
			lines = append(lines, m.labelValue("Ready", formatCount(ready)))
		}
		if waiting, ok := m.statsWaitingCount(); ok {
			lines = append(lines, m.labelValue("Waiting", formatCount(waiting)))
		}
		queues := m.statsQueueNames()
		lines = append(lines, "")
		if len(queues) > 0 {
			lines = append(lines, "Queues")
			lines = append(lines, truncate(strings.Join(queues, ", "), 60))
		} else if m.statsAvailable() {
			lines = append(lines, m.theme.Styles.Muted.Render("No queues reported."))
		} else {
			lines = append(lines, m.theme.Styles.Muted.Render("No Django stats configured."))
		}
		summaries := m.statsQueueSummaries()
		if len(summaries) > 0 {
			lines = append(lines, "")
			lines = append(lines, "Top queues")
			for i, summary := range summaries {
				if i >= 5 {
					break
				}
				line := fmt.Sprintf(
					"%s %s (R%s W%s Ru%s F%s)",
					summary.Name,
					formatCount(summary.Total),
					formatCount(summary.Ready),
					formatCount(summary.Waiting),
					formatCount(summary.Running),
					formatCount(summary.Failed),
				)
				lines = append(lines, truncate(line, 60))
			}
		}
		paused := m.statsPausedQueues()
		if len(paused) > 0 {
			lines = append(lines, "")
			lines = append(lines, "Paused queues")
			for i, control := range paused {
				if i >= 5 {
					break
				}
				label := control.QueueName
				if control.Database != "" {
					label = fmt.Sprintf("%s@%s", control.QueueName, control.Database)
				}
				line := label
				if reason := strings.TrimSpace(control.Reason); reason != "" {
					line = fmt.Sprintf("%s - %s", label, reason)
				}
				lines = append(lines, truncate(line, 60))
			}
		}
		return strings.Join(lines, "\n")
	case "Workers":
		inUse := m.latestValue(metrics.MetricConcurrencyInUse)
		limit := m.latestValue(metrics.MetricConcurrencyLimit)
		gauge := charts.Gauge(inUse, limit, 20)
		lines := []string{
			m.labelValue("Workers", formatNumber(m.workerCountValue())),
			m.labelValue("Concurrency", fmt.Sprintf("%s/%s", formatNumber(inUse), formatNumber(limit))),
			"",
			fmt.Sprintf("Usage  %s", gauge),
		}
		workers := m.statsWorkersByRecent()
		now := m.referenceTime()
		active, stale := m.splitWorkersByStatus(workers, now)
		lines = append(lines, "")
		if len(workers) == 0 {
			if m.statsAvailable() {
				lines = append(lines, m.theme.Styles.Muted.Render("No workers reported."))
			} else {
				lines = append(lines, m.theme.Styles.Muted.Render("No Django stats configured."))
			}
			return strings.Join(lines, "\n")
		}
		if len(active) > 0 {
			lines = append(lines, fmt.Sprintf("Active workers (%d)", len(active)))
			for i, worker := range active {
				if i >= 4 {
					break
				}
				seen := formatTimestamp(worker.LastSeenAt)
				line := fmt.Sprintf("%s (%s) c=%d %s", worker.WorkerID, worker.Hostname, worker.Concurrency, seen)
				lines = append(lines, truncate(line, 60))
			}
		} else {
			lines = append(lines, m.theme.Styles.Muted.Render("No active workers."))
		}
		if len(stale) > 0 {
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("Stale workers (%d)", len(stale)))
			for i, worker := range stale {
				if i >= 3 {
					break
				}
				seen := formatTimestamp(worker.LastSeenAt)
				line := fmt.Sprintf("%s (%s) c=%d %s", worker.WorkerID, worker.Hostname, worker.Concurrency, seen)
				lines = append(lines, m.theme.Styles.Muted.Render(truncate(line, 60)))
			}
		}
		return strings.Join(lines, "\n")
	case "Periodic":
		periodic := m.statsPeriodicByNextRun()
		if len(periodic) == 0 {
			if m.statsAvailable() {
				return m.theme.Styles.Muted.Render("No periodic tasks reported.")
			}
			return m.theme.Styles.Muted.Render("No Django stats configured.")
		}
		ref := m.referenceTime()
		lines := []string{
			m.labelValue("Total tasks", formatCount(int64(len(periodic)))),
		}
		if sched, ok := m.statsScheduler(); ok {
			lines = append(lines,
				m.labelValue("Scheduler", formatScheduler(sched)),
				m.labelValue("Beat", formatYesNo(sched.BeatEnabled)),
				m.labelValue("pg_cron", formatYesNo(sched.PgCronAvailable)),
			)
			if sched.Warning != "" {
				lines = append(lines, "", m.theme.Styles.StatusWarn.Render(sched.Warning))
			}
		}
		lines = append(lines, "")
		for i, task := range periodic {
			if i >= 10 { // Show more tasks in detail view
				break
			}
			next := formatScheduleDetailed(task.NextRunAt, ref)
			statusIcon := m.theme.Styles.AccentAlt.Render("●")
			if !task.Enabled {
				statusIcon = m.theme.Styles.Muted.Render("○")
			}
			line := fmt.Sprintf("%s %-24s %s", statusIcon, truncate(task.Name, 24), next)
			if !task.Enabled {
				line = m.theme.Styles.Muted.Render(line)
			}
			lines = append(lines, line)
		}
		return strings.Join(lines, "\n")
	case "Databases":
		summaries := m.statsDatabaseSummaries()
		if len(summaries) == 0 {
			if m.statsAvailable() {
				return m.theme.Styles.Muted.Render("No per-database stats reported.")
			}
			return m.theme.Styles.Muted.Render("No Django stats configured.")
		}
		lines := []string{
			m.labelValue("Databases", formatCount(int64(len(summaries)))),
			"",
		}
		for i, summary := range summaries {
			if i >= 6 {
				break
			}
			line := fmt.Sprintf(
				"%s %s (R%s W%s Ru%s F%s) w%d q%d p%d",
				summary.Alias,
				formatCount(summary.Total),
				formatCount(summary.Ready),
				formatCount(summary.Waiting),
				formatCount(summary.Running),
				formatCount(summary.Failed),
				summary.Workers,
				summary.Queues,
				summary.Periodic,
			)
			lines = append(lines, truncate(line, 60))
		}
		return strings.Join(lines, "\n")
	case "Tasks":
		bar := charts.Bar(m.seriesValues(seriesThroughput), 24, 4)
		lines := []string{}
		if ready, ok := m.statsTaskCount("READY"); ok {
			lines = append(lines, m.labelValue("Ready", formatCount(ready)))
		}
		if waiting, ok := m.statsWaitingCount(); ok {
			lines = append(lines, m.labelValue("Waiting", formatCount(waiting)))
		}
		if running, ok := m.statsTaskCount("RUNNING"); ok {
			lines = append(lines, m.labelValue("Running", formatCount(running)))
		}
		if success, ok := m.statsTaskCount("SUCCESSFUL"); ok {
			lines = append(lines, m.labelValue("Success", formatCount(success)))
		}
		if failed, ok := m.statsTaskCount("FAILED"); ok {
			lines = append(lines, m.labelValue("Failed", formatCount(failed)))
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines,
			m.labelValue("Throughput", formatRate(m.currentThroughput())),
			m.labelValue("Success ratio", formatPercent(m.currentSuccessRatio())),
			m.labelValue("Errors", formatRate(m.latestValue(seriesErrors))),
			m.labelValue("P95 latency", formatDuration(time.Duration(m.currentLatencyP95()*float64(time.Second)))),
		)
		if m.statsAvailable() && len(m.lastStats.TopFailing) > 0 {
			lines = append(lines, "", "Top Failing")
			for _, task := range m.lastStats.TopFailing {
				line := fmt.Sprintf("%-32s %s", truncate(task.TaskPath, 32), formatCount(task.Count))
				lines = append(lines, m.theme.Styles.StatusDown.Render(line))
			}
		}
		lines = append(lines,
			"",
			"Throughput trend",
			bar,
		)
		return strings.Join(lines, "\n")
	case "Errors":
		return m.renderErrorList()
	default:
		return m.theme.Styles.Muted.Render("No detail view available.")
	}
}

func (m *Model) renderErrorList() string {
	if !m.eventsEnabled {
		return m.theme.Styles.Muted.Render("No events stream configured.")
	}
	events := m.eventsBuffer.Items()
	lines := []string{}
	for _, event := range events {
		level := strings.ToLower(event.Level)
		if level != "error" && level != "warn" && level != "warning" {
			continue
		}
		timestamp := formatTimestamp(event.Timestamp)
		meta := formatEventMeta(event.Metadata)
		line := fmt.Sprintf("%s %s", timestamp, event.Message)
		if meta != "" {
			line = fmt.Sprintf("%s [%s] %s", timestamp, meta, event.Message)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return m.theme.Styles.Muted.Render("No recent errors.")
	}
	if len(lines) > 8 {
		lines = lines[len(lines)-8:]
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderStatusBar() string {
	state, badge := m.connectionState()
	parts := []string{badge.Render(state)}
	if m.paused {
		parts = append(parts, m.theme.Styles.Badge.Render("PAUSED"))
	}
	if m.lowMemoryMode {
		parts = append(parts, m.theme.Styles.StatusWarn.Render("LOW MEMORY"))
	}
	if host := m.workerHost(); host != "" {
		worker := fmt.Sprintf("%s %s", m.theme.Styles.Muted.Render("worker"), m.theme.Styles.AccentAlt.Render(host))
		parts = append(parts, worker)
	}
	if m.lastHealth.Version != "" {
		parts = append(parts, m.theme.Styles.Accent.Render("v"+m.lastHealth.Version))
	}
	if m.lastHealth.Build != "" {
		parts = append(parts, m.theme.Styles.Muted.Render(m.lastHealth.Build))
	}
	if m.lastScrapeAt.IsZero() {
		parts = append(parts, m.theme.Styles.Muted.Render("scrape pending"))
	} else {
		scrape := fmt.Sprintf("scrape %s (%s)", formatRelative(m.lastScrapeAt), formatDuration(m.lastScrapeDelay))
		parts = append(parts, m.theme.Styles.Muted.Render(scrape))
	}
	if m.lastScrapeErr != nil {
		parts = append(parts, m.theme.Styles.StatusWarn.Render("metrics err"))
	}
	if m.lastHealthErr != nil {
		parts = append(parts, m.theme.Styles.StatusWarn.Render("health err"))
	}
	if m.statsEnabled && m.lastStatsErr != nil {
		parts = append(parts, m.theme.Styles.StatusWarn.Render("stats err"))
	}
	line := strings.Join(parts, "  ")
	return m.theme.Styles.StatusBar.Width(m.width).Render(line)
}

func (m *Model) renderMain(height int) string {
	gap := 4
	leftW, centerW, rightW := m.columnWidths(gap)

	left := m.renderLeftPane(leftW, height)
	center := m.renderCenterPane(centerW, height)
	if !m.showEvents || rightW == 0 {
		return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", gap), center)
	}
	right := m.renderRightPane(rightW, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", gap), center, strings.Repeat(" ", gap), right)
}

func (m *Model) renderLeftPane(width, height int) string {
	loading := m.lastScrapeAt.IsZero()
	ref := m.referenceTime()
	updatedAt := maxTime(m.lastScrapeAt, m.lastStatsAt)

	queueDepth := m.queueDepthValue()
	workerCount := m.workerCountValue()

	queueCount := "-"
	if count, ok := m.statsQueueCount(); ok {
		queueCount = formatCount(count)
	}
	dbCount := "-"
	if summary := m.statsDatabaseSummaries(); len(summary) > 0 {
		dbCount = formatCount(int64(len(summary)))
	}
	periodicCount := "-"
	if count, ok := m.statsPeriodicCount(); ok {
		periodicCount = formatCount(count)
	}
	nextPeriodic := "-"
	if next, ok := m.statsNextPeriodic(); ok {
		nextPeriodic = formatScheduleAt(next.NextRunAt, ref)
	}
	schedulerLabel := "-"
	if sched, ok := m.statsScheduler(); ok {
		schedulerLabel = formatScheduler(sched)
	}
	activeWorkers := "-"
	staleWorkers := "-"
	if active, stale, ok := m.statsWorkerStatusCounts(); ok {
		activeWorkers = formatCount(int64(active))
		staleWorkers = formatCount(int64(stale))
	}
	workerActiveLabel := "Active"
	workerStaleLabel := "Stale"
	if alive, dead, ok := m.statsWorkerHealthCounts(); ok {
		workerActiveLabel = "Alive"
		workerStaleLabel = "Dead"
		activeWorkers = formatCount(alive)
		staleWorkers = formatCount(dead)
	}

	val := func(v string) string {
		if loading {
			return "..."
		}
		return v
	}

	lines := []string{}
	if m.lowMemoryMode {
		lines = append(lines, m.theme.Styles.StatusWarn.Render("Low memory mode enabled — metrics/events unavailable."), "")
	}
	if loading {
		lines = append(lines, m.theme.Styles.Muted.Render("Waiting for first scrape..."))
	}

	// Task Status Summary
	readyCount, _ := m.statsTaskCount("READY")
	waitingCount, _ := m.statsWaitingCount()
	runningCount, _ := m.statsTaskCount("RUNNING")
	failedCount, _ := m.statsTaskCount("FAILED")

	dbWait := m.latestValue(metrics.MetricDBPoolWait)
	dbWaitValue := "-"
	if !math.IsNaN(dbWait) && !math.IsInf(dbWait, 0) {
		dbWaitValue = formatCount(int64(dbWait))
	}

	lines = append(lines,
		m.theme.Styles.PaneHeader.Render("TASKS"),
		m.labelValue("Queue depth", val(formatNumber(queueDepth))),
		m.labelValue("Ready", val(formatCount(readyCount+waitingCount))),
		m.labelValue("Running", val(formatCount(runningCount))),
		m.labelValue("Failed", val(formatCount(failedCount))),
		m.labelValue("Paused", val(formatCount(int64(len(m.statsPausedQueues()))))),
		"",
		m.theme.Styles.PaneHeader.Render("WORKERS"),
		m.labelValue("Total", val(formatNumber(workerCount))),
		m.labelValue(workerActiveLabel, val(activeWorkers)),
		m.labelValue(workerStaleLabel, val(staleWorkers)),
		m.labelValue("Concurrency", val(fmt.Sprintf("%s/%s",
			formatNumber(m.latestValue(metrics.MetricConcurrencyInUse)),
			formatNumber(m.latestValue(metrics.MetricConcurrencyLimit)),
		))),
		"",
		m.theme.Styles.PaneHeader.Render("SYSTEM"),
		m.labelValue("Worker Mem", val(formatBytes(m.latestValue(metrics.MetricWorkerMemUsage)))),
		m.labelValue("DB Pool", val(fmt.Sprintf("%.0f", m.latestValue(metrics.MetricDBPoolConnections)))),
		m.labelValue("DB Wait", val(dbWaitValue)),
		m.labelValue("DBs", val(dbCount)),
		"",
		m.theme.Styles.PaneHeader.Render("PERIODIC"),
		m.labelValue("Count", val(periodicCount)),
		m.labelValue("Scheduler", val(schedulerLabel)),
		m.labelValue("Queues", val(queueCount)),
		m.labelValue("Next up", val(nextPeriodic)),
	)
	body := strings.Join(lines, "\n")
	card := m.card("OBSERVABILITY", body, width, height, m.focus == focusLeft, updatedAt)
	return card
}

func (m *Model) renderCenterPane(width, height int) string {
	gap := 1
	cardHeight := maxInt(6, (height-gap)/2)
	chartWidth := maxInt(10, width-6)
	loading := m.lastScrapeAt.IsZero()

	throughput := m.seriesValues(seriesThroughput)
	queueDepth := m.seriesValues(metrics.MetricQueueDepth)
	errors := m.seriesValues(seriesErrors)
	latency := m.seriesValues(metrics.MetricLatencyP95)

	val := func(v string) string {
		if loading {
			return "..."
		}
		return v
	}

	renderSparkline := func(values []float64, style lipgloss.Style) string {
		if loading {
			return m.loadingOverlay(chartWidth)
		}
		if len(values) == 0 {
			return m.theme.Styles.Muted.Render("No data yet")
		}
		return style.Render(charts.Sparkline(values, chartWidth))
	}

	first := m.chartCard("Throughput", val(formatRate(m.currentThroughput())), renderSparkline(throughput, m.theme.Styles.Accent), width, cardHeight, m.focus == focusCenter, m.lastScrapeAt)
	second := m.chartCard("Queue depth", val(formatNumber(m.latestValue(metrics.MetricQueueDepth))), renderSparkline(queueDepth, m.theme.Styles.AccentAlt), width, cardHeight, false, m.lastScrapeAt)

	remaining := height - (cardHeight*2 + gap)
	if remaining >= cardHeight {
		third := m.chartCard("Errors", val(formatRate(m.latestValue(seriesErrors))), renderSparkline(errors, m.theme.Styles.StatusWarn), width, cardHeight, false, m.lastScrapeAt)
		fourth := m.chartCard("P95 latency", val(formatDuration(time.Duration(m.currentLatencyP95()*float64(time.Second)))), renderSparkline(latency, m.theme.Styles.Muted), width, cardHeight, false, m.lastScrapeAt)
		return lipgloss.JoinVertical(lipgloss.Left, first, strings.Repeat("\n", gap), second, strings.Repeat("\n", gap), third, strings.Repeat("\n", gap), fourth)
	}

	return lipgloss.JoinVertical(lipgloss.Left, first, strings.Repeat("\n", gap), second)
}

func (m *Model) renderRightPane(width, height int) string {
	status := m.eventsStatus()
	header := m.theme.Styles.PaneHeader.Width(width).Render(joinRight("Events", status, width))
	cardStyle := m.theme.Styles.Card
	if m.focus == focusRight {
		cardStyle = cardStyle.BorderForeground(m.theme.Palette.AccentAlt)
	}
	contentWidth := width - cardStyle.GetHorizontalFrameSize()
	if contentWidth < 0 {
		contentWidth = 0
	}
	lines := m.renderEvents(contentWidth, height-1)
	body := strings.Join(lines, "\n")
	card := cardStyle.Width(width).Height(height - 1).Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, header, card)
}

func (m *Model) renderFooter() string {
	if m.filterActive {
		filterLine := fmt.Sprintf("Filter: %s (enter to apply, esc to cancel)", m.filterInput.View())
		return m.theme.Styles.KeyHint.Width(m.width).Render(filterLine)
	}
	help := m.help.ShortHelpView(m.keymap.ShortHelp())
	filter := m.filter
	if filter == "" {
		filter = "-"
	}
	window := m.currentWindow().String()
	parts := []string{
		m.theme.Styles.Muted.Render("Window " + window),
		m.theme.Styles.Muted.Render("Filter " + filter),
		m.authStatusSummary(),
	}
	if m.toast != "" && time.Now().Before(m.toastExpiry) {
		parts = append(parts, m.theme.Styles.AccentAlt.Render(m.toast))
	}
	right := strings.Join(parts, "  ")
	line := joinRight(help, right, m.width)
	return m.theme.Styles.KeyHint.Width(m.width).Render(line)
}

func (m *Model) authStatusSummary() string {
	if !m.authHeaderManaged {
		if m.authNeeded {
			return m.theme.Styles.StatusWarn.Render("Auth failed")
		}
		return m.theme.Styles.Muted.Render("Auth configured")
	}
	if m.authFlowActive {
		return m.theme.Styles.AccentAlt.Render("Auth pending")
	}
	if m.authToken.Value != "" {
		if !m.authToken.ExpiresAt.IsZero() && time.Now().After(m.authToken.ExpiresAt) {
			return m.theme.Styles.StatusWarn.Render("Auth expired")
		}
		return m.theme.Styles.Accent.Render("Auth signed in")
	}
	if m.authNeeded {
		return m.theme.Styles.StatusWarn.Render("Auth required")
	}
	if !m.authEnabled {
		return m.theme.Styles.Muted.Render("Auth -")
	}
	return m.theme.Styles.Muted.Render("Auth signed out")
}

func (m *Model) connectionState() (string, lipgloss.Style) {
	if m.lastScrapeAt.IsZero() {
		return "CONNECTING", m.theme.Styles.StatusBadgeWarn
	}
	if m.lastScrapeErr != nil && m.lastHealthErr != nil {
		return "DOWN", m.theme.Styles.StatusBadgeDown
	}
	if m.lastScrapeErr != nil || m.lastHealthErr != nil || (!m.lastHealth.Healthy && !m.lastHealth.CheckedAt.IsZero()) {
		return "DEGRADED", m.theme.Styles.StatusBadgeWarn
	}
	return "OK", m.theme.Styles.StatusBadgeOK
}

func (m *Model) workerHost() string {
	if m.cfg.WorkerMetricsURL == "" {
		return ""
	}
	parsed, err := url.Parse(m.cfg.WorkerMetricsURL)
	if err != nil {
		return ""
	}
	if parsed.Host != "" {
		return parsed.Host
	}
	return parsed.Path
}

func (m *Model) eventsHost() string {
	if m.eventsBaseURL == "" {
		return ""
	}
	parsed, err := url.Parse(m.eventsBaseURL)
	if err != nil {
		return ""
	}
	if parsed.Host != "" {
		return parsed.Host
	}
	return parsed.Path
}

func (m *Model) renderEvents(width, height int) []string {
	if m.lowMemoryMode {
		return []string{
			m.theme.Styles.StatusWarn.Render("Low memory mode enabled."),
			m.theme.Styles.Muted.Render("Events are disabled."),
		}
	}
	if !m.eventsEnabled {
		return []string{m.theme.Styles.Muted.Render("No events stream configured")}
	}
	events := m.eventsBuffer.Items()
	filtered := make([]string, 0, len(events))
	for _, event := range events {
		if !m.matchFilter(event) {
			continue
		}
		timestamp := formatTimestamp(event.Timestamp)
		meta := formatEventMeta(event.Metadata)
		rawLine := fmt.Sprintf("%s %s", timestamp, event.Message)
		if meta != "" {
			rawLine = fmt.Sprintf("%s [%s] %s", timestamp, meta, event.Message)
		}
		rawLine = truncate(rawLine, width-2)
		line := rawLine
		level := strings.ToLower(event.Level)
		switch level {
		case "error":
			line = m.theme.Styles.StatusDown.Render(rawLine)
		case "warn", "warning":
			line = m.theme.Styles.StatusWarn.Render(rawLine)
		default:
			line = m.theme.Styles.Muted.Render(rawLine)
		}
		filtered = append(filtered, line)
	}
	if len(filtered) == 0 {
		if m.lastScrapeAt.IsZero() {
			filtered = append(filtered, m.theme.Styles.Muted.Render("Waiting for events..."))
		} else {
			filtered = append(filtered, m.theme.Styles.Muted.Render("No events yet"))
		}
		if host := m.eventsHost(); host != "" {
			filtered = append(filtered, m.theme.Styles.Muted.Render("Listening on "+host))
		}
	}
	if len(filtered) > height {
		filtered = filtered[len(filtered)-height:]
	}
	return filtered
}

func (m *Model) eventsStatus() string {
	if m.lowMemoryMode {
		return m.theme.Styles.StatusWarn.Render("low memory")
	}
	if !m.eventsEnabled {
		return m.theme.Styles.Muted.Render("disabled")
	}
	if !m.lastEventAt.IsZero() {
		return m.theme.Styles.Muted.Render("last " + formatTimestamp(m.lastEventAt))
	}
	if m.lastScrapeAt.IsZero() {
		return m.theme.Styles.Muted.Render("connecting")
	}
	return m.theme.Styles.Muted.Render("listening")
}

func (m *Model) matchFilter(event models.Event) bool {
	if strings.TrimSpace(m.filterLocal) == "" {
		return true
	}
	needle := strings.ToLower(m.filterLocal)
	hay := strings.ToLower(strings.Join([]string{
		event.Message,
		event.Type,
		event.Level,
		event.Queue,
		event.TaskID,
		event.WorkerID,
		flattenMeta(event.Metadata),
	}, " "))
	return strings.Contains(hay, needle)
}

func flattenMeta(meta map[string]string) string {
	if len(meta) == 0 {
		return ""
	}
	keys := make([]string, 0, len(meta))
	for key := range meta {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(meta))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:%s", key, meta[key]))
	}
	return strings.Join(parts, " ")
}

func formatEventMeta(meta map[string]string) string {
	if len(meta) == 0 {
		return ""
	}
	role, ok := meta["role"]
	if !ok || role == "" {
		return flattenMeta(meta)
	}
	if len(meta) == 1 {
		return fmt.Sprintf("role:%s", role)
	}
	rest := make(map[string]string, len(meta)-1)
	for key, val := range meta {
		if key == "role" {
			continue
		}
		rest[key] = val
	}
	tail := flattenMeta(rest)
	if tail == "" {
		return fmt.Sprintf("role:%s", role)
	}
	return fmt.Sprintf("role:%s %s", role, tail)
}

func (m *Model) chartCard(title, value, chart string, width, height int, focused bool, updatedAt time.Time) string {
	innerWidth := maxInt(10, width-6)
	header := m.cardHeader(title, value, updatedAt, innerWidth)
	body := strings.Join([]string{header, chart}, "\n")
	cardStyle := m.theme.Styles.Card
	if focused {
		cardStyle = cardStyle.BorderForeground(m.theme.Palette.AccentAlt)
	}
	return cardStyle.Width(width).Height(height).Render(body)
}

func (m *Model) card(title, body string, width, height int, focused bool, updatedAt time.Time) string {
	innerWidth := maxInt(10, width-6)
	header := m.cardHeader(title, "", updatedAt, innerWidth)
	content := strings.Join([]string{header, body}, "\n")
	cardStyle := m.theme.Styles.Card
	if focused {
		cardStyle = cardStyle.BorderForeground(m.theme.Palette.AccentAlt)
	}
	return cardStyle.Width(width).Height(height).Render(content)
}

func (m *Model) labelValue(label, value string) string {
	padded := fmt.Sprintf("%-14s", label)
	labelText := m.theme.Styles.Muted.Render(padded)
	return fmt.Sprintf("%s %s", labelText, m.renderValue(value))
}

func (m *Model) cardHeader(title, value string, updatedAt time.Time, width int) string {
	left := m.theme.Styles.CardTitle.Render(title)
	if value != "" {
		left = fmt.Sprintf("%s  %s", left, m.renderValue(value))
	}
	right := m.updatedText(updatedAt)
	if right == "" {
		return left
	}
	return joinRight(left, right, width)
}

func (m *Model) updatedText(ts time.Time) string {
	if ts.IsZero() {
		return m.theme.Styles.Muted.Render("upd -")
	}
	return m.theme.Styles.Muted.Render("upd " + formatTimestamp(ts))
}

func (m *Model) renderValue(value string) string {
	trimmed := strings.TrimSpace(value)
	switch trimmed {
	case "", "-", "...":
		return m.theme.Styles.Muted.Render(value)
	default:
		return m.theme.Styles.Value.Render(value)
	}
}

func joinRight(left, right string, width int) string {
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	space := width - leftWidth - rightWidth
	if space < 1 {
		space = 1
	}
	return left + strings.Repeat(" ", space) + right
}

func truncate(text string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(text) <= max {
		return text
	}
	runes := []rune(text)
	if len(runes) <= max {
		return string(runes)
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func (m *Model) columnWidths(gap int) (int, int, int) {
	left := clampInt(m.width/4, 22, 34)
	right := 0
	if m.showEvents {
		right = clampInt(m.width/4, 24, 38)
	}
	center := m.width - left - right - gap
	if m.showEvents {
		center -= gap
	}
	if center < 20 {
		right = 0
		center = m.width - left - gap
	}
	if center < 20 {
		left = maxInt(18, m.width/3)
		center = m.width - left - gap
	}
	return left, maxInt(20, center), right
}

func (m *Model) loadingOverlay(width int) string {
	if width <= 0 {
		return ""
	}
	line := strings.Repeat(".", width)
	label := "loading " + m.spinner.View()
	label = truncate(label, width)
	return strings.Join([]string{
		m.theme.Styles.Muted.Render(line),
		m.theme.Styles.Muted.Render(label),
	}, "\n")
}

func maxTime(values ...time.Time) time.Time {
	var latest time.Time
	for _, value := range values {
		if value.After(latest) {
			latest = value
		}
	}
	return latest
}

func (m *Model) contentHeight() int {
	if m.safeTop <= 0 {
		return m.height
	}
	height := m.height - m.safeTop
	if height < 1 {
		return m.height
	}
	return height
}

func (m *Model) applySafeTop(view string) string {
	if m.safeTop <= 0 {
		return view
	}
	return strings.Repeat("\n", m.safeTop) + view
}

func (m *Model) placeCentered(content string) string {
	return lipgloss.Place(m.width, m.contentHeight(), lipgloss.Center, lipgloss.Center, content)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
