package ui

import (
	"fmt"
	"net/url"
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
		return m.renderAuthURLPrompt()
	}
	if m.authFlowActive {
		return m.renderAuthPrompt()
	}
	if m.setupActive {
		return m.renderSetup()
	}
	if m.detailActive {
		return m.renderDetails()
	}
	if m.showHelp {
		return m.renderHelp()
	}
	top := m.renderStatusBar()
	bottom := m.renderFooter()
	mainHeight := m.height - lipgloss.Height(top) - lipgloss.Height(bottom)
	if mainHeight < 5 {
		mainHeight = 5
	}
	main := m.renderMain(mainHeight)
	return lipgloss.JoinVertical(lipgloss.Left, top, main, bottom)
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
			"Worker URL or /metrics endpoint:",
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
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, card)
}

func (m *Model) renderHelp() string {
	content := m.help.FullHelpView(m.keymap.FullHelp())
	width := maxInt(20, minInt(60, m.width-4))
	card := m.theme.Styles.Card.Width(width).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, card)
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
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, card)
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
		lines = append(lines, "", m.theme.Styles.StatusWarn.Render(fmt.Sprintf("Error: %v", m.authErr)))
	}
	lines = append(lines, "", m.theme.Styles.Muted.Render("esc to cancel"))
	content := strings.Join(lines, "\n")
	card := m.theme.Styles.Card.Width(width).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, card)
}

func (m *Model) renderDetails() string {
	view := m.detailViews[m.detailIndex%len(m.detailViews)]
	title := fmt.Sprintf("Details: %s", view)
	body := m.detailBody(view)
	footer := m.theme.Styles.Muted.Render("tab to switch | esc to close")
	content := strings.Join([]string{m.theme.Styles.CardTitle.Render(title), body, "", footer}, "\n")
	width := maxInt(30, minInt(70, m.width-6))
	height := maxInt(10, minInt(20, m.height-6))
	card := m.theme.Styles.Card.Width(width).Height(height).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, card)
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
		lines = append(lines, "")
		if len(workers) == 0 {
			if m.statsAvailable() {
				lines = append(lines, m.theme.Styles.Muted.Render("No workers reported."))
			} else {
				lines = append(lines, m.theme.Styles.Muted.Render("No Django stats configured."))
			}
			return strings.Join(lines, "\n")
		}
		lines = append(lines, "Recent workers")
		for i, worker := range workers {
			if i >= 5 {
				break
			}
			seen := formatTimestamp(worker.LastSeenAt)
			line := fmt.Sprintf("%s (%s) c=%d %s", worker.WorkerID, worker.Hostname, worker.Concurrency, seen)
			lines = append(lines, truncate(line, 60))
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
		lines := []string{
			m.labelValue("Total", formatCount(int64(len(periodic)))),
			"",
		}
		for i, task := range periodic {
			if i >= 6 {
				break
			}
			next := formatTimestamp(task.NextRunAt)
			line := fmt.Sprintf("%s  %s", task.Name, next)
			if !task.Enabled {
				line = m.theme.Styles.Muted.Render(line + " (disabled)")
			}
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
		line := fmt.Sprintf("%s %s", formatTimestamp(event.Timestamp), event.Message)
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
	scrape := fmt.Sprintf("scrape %s (%s)", formatRelative(m.lastScrapeAt), formatDuration(m.lastScrapeDelay))
	parts = append(parts, m.theme.Styles.Muted.Render(scrape))
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
	gap := 1
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
	spinnerView := m.spinner.View()

	queueDepth := m.queueDepthValue()
	running := m.runningCountValue()
	workerCount := m.workerCountValue()
	queueCount := "-"
	if count, ok := m.statsQueueCount(); ok {
		queueCount = formatCount(count)
	}
	periodicCount := "-"
	if count, ok := m.statsPeriodicCount(); ok {
		periodicCount = formatCount(count)
	}

	val := func(v string) string {
		if loading {
			return spinnerView
		}
		return v
	}

	lines := []string{
		m.labelValue("Queue depth", val(formatNumber(queueDepth))),
		m.labelValue("Running", val(formatNumber(running))),
		m.labelValue("Throughput", val(formatRate(m.currentThroughput()))),
		m.labelValue("Errors", val(formatRate(m.latestValue(seriesErrors)))),
		m.labelValue("Success", val(formatPercent(m.currentSuccessRatio()))),
		m.labelValue("Workers", val(formatNumber(workerCount))),
		m.labelValue("Queues", val(queueCount)),
		m.labelValue("Periodic", val(periodicCount)),
		m.labelValue("Concurrency", val(fmt.Sprintf("%s/%s",
			formatNumber(m.latestValue(metrics.MetricConcurrencyInUse)),
			formatNumber(m.latestValue(metrics.MetricConcurrencyLimit)),
		))),
		m.labelValue("P95 latency", val(formatDuration(time.Duration(m.currentLatencyP95()*float64(time.Second))))),
	}
	body := strings.Join(lines, "\n")
	card := m.card("Now", body, width, height, m.focus == focusLeft)
	return card
}

func (m *Model) renderCenterPane(width, height int) string {
	gap := 1
	cardHeight := maxInt(6, (height-gap)/2)
	chartWidth := maxInt(10, width-6)
	loading := m.lastScrapeAt.IsZero()
	spinnerView := m.spinner.View()

	throughput := m.seriesValues(seriesThroughput)
	queueDepth := m.seriesValues(metrics.MetricQueueDepth)
	errors := m.seriesValues(seriesErrors)
	latency := m.seriesValues(metrics.MetricLatencyP95)

	val := func(v string) string {
		if loading {
			return spinnerView
		}
		return v
	}

	chart := func(c string) string {
		if loading {
			return "\n\n  " + m.theme.Styles.Muted.Render("Waiting for metrics...")
		}
		return c
	}

	first := m.chartCard("Throughput", val(formatRate(m.currentThroughput())), chart(charts.Sparkline(throughput, chartWidth)), width, cardHeight, m.focus == focusCenter)
	second := m.chartCard("Queue depth", val(formatNumber(m.latestValue(metrics.MetricQueueDepth))), chart(charts.Sparkline(queueDepth, chartWidth)), width, cardHeight, false)

	remaining := height - (cardHeight*2 + gap)
	if remaining >= cardHeight {
		third := m.chartCard("Errors", val(formatRate(m.latestValue(seriesErrors))), chart(charts.Sparkline(errors, chartWidth)), width, cardHeight, false)
		fourth := m.chartCard("P95 latency", val(formatDuration(time.Duration(m.currentLatencyP95()*float64(time.Second)))), chart(charts.Sparkline(latency, chartWidth)), width, cardHeight, false)
		return lipgloss.JoinVertical(lipgloss.Left, first, strings.Repeat("\n", gap), second, strings.Repeat("\n", gap), third, strings.Repeat("\n", gap), fourth)
	}

	return lipgloss.JoinVertical(lipgloss.Left, first, strings.Repeat("\n", gap), second)
}

func (m *Model) renderRightPane(width, height int) string {
	header := m.theme.Styles.PaneHeader.Width(width).Render("Events")
	lines := m.renderEvents(width, height-1)
	body := strings.Join(lines, "\n")
	cardStyle := m.theme.Styles.Card
	if m.focus == focusRight {
		cardStyle = cardStyle.BorderForeground(m.theme.Palette.AccentAlt)
	}
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

func (m *Model) renderEvents(width, height int) []string {
	if !m.eventsEnabled {
		return []string{m.theme.Styles.Muted.Render("No events stream configured")}
	}
	events := m.eventsBuffer.Items()
	filtered := make([]string, 0, len(events))
	for _, event := range events {
		if !m.matchFilter(event) {
			continue
		}
		line := fmt.Sprintf("%s %s", formatTimestamp(event.Timestamp), event.Message)
		level := strings.ToLower(event.Level)
		switch level {
		case "error":
			line = m.theme.Styles.StatusDown.Render(line)
		case "warn", "warning":
			line = m.theme.Styles.StatusWarn.Render(line)
		default:
			line = m.theme.Styles.Muted.Render(line)
		}
		filtered = append(filtered, truncate(line, width-2))
	}
	if len(filtered) == 0 {
		filtered = append(filtered, m.theme.Styles.Muted.Render("No events"))
	}
	if len(filtered) > height {
		filtered = filtered[len(filtered)-height:]
	}
	return filtered
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
	parts := make([]string, 0, len(meta))
	for key, val := range meta {
		parts = append(parts, fmt.Sprintf("%s:%s", key, val))
	}
	return strings.Join(parts, " ")
}

func (m *Model) chartCard(title, value, chart string, width, height int, focused bool) string {
	header := fmt.Sprintf("%s  %s", m.theme.Styles.CardTitle.Render(title), m.theme.Styles.Muted.Render(value))
	body := strings.Join([]string{header, chart}, "\n")
	cardStyle := m.theme.Styles.Card
	if focused {
		cardStyle = cardStyle.BorderForeground(m.theme.Palette.AccentAlt)
	}
	return cardStyle.Width(width).Height(height).Render(body)
}

func (m *Model) card(title, body string, width, height int, focused bool) string {
	header := m.theme.Styles.CardTitle.Render(title)
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
	if value == "-" {
		return fmt.Sprintf("%s %s", labelText, m.theme.Styles.Muted.Render(value))
	}
	return fmt.Sprintf("%s %s", labelText, m.theme.Styles.Accent.Render(value))
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
