package ui

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/adpena/reproq-tui/internal/metrics"
	"github.com/adpena/reproq-tui/pkg/models"
)

func (m *Model) latestValue(key string) float64 {
	buf, ok := m.series[key]
	if !ok {
		return math.NaN()
	}
	latest, ok := buf.Latest()
	if !ok {
		return math.NaN()
	}
	return latest.Value
}

func (m *Model) seriesValues(key string) []float64 {
	return valuesFromSamples(m.seriesSamples(key))
}

func (m *Model) seriesSamples(key string) []models.Sample {
	buf, ok := m.series[key]
	if !ok {
		return nil
	}
	cutoff := metrics.WindowCutoff(m.currentWindow(), time.Now())
	return buf.ValuesSince(cutoff)
}

func (m *Model) currentThroughput() float64 {
	return latestValueFrom(m.seriesSamples(seriesThroughput))
}

func (m *Model) currentErrorRatio() float64 {
	return metrics.Ratio(m.seriesSamples(metrics.MetricTasksFailed), m.seriesSamples(metrics.MetricTasksTotal))
}

func (m *Model) currentSuccessRatio() float64 {
	ratio := m.currentErrorRatio()
	if math.IsNaN(ratio) {
		return math.NaN()
	}
	return 1 - ratio
}

func (m *Model) currentLatencyP95() float64 {
	return m.latestValue(metrics.MetricLatencyP95)
}

func (m *Model) queueTrend() float64 {
	samples := m.seriesSamples(metrics.MetricQueueDepth)
	if len(samples) < 2 {
		return math.NaN()
	}
	return samples[len(samples)-1].Value - samples[0].Value
}

func valuesFromSamples(samples []models.Sample) []float64 {
	values := make([]float64, 0, len(samples))
	for _, sample := range samples {
		values = append(values, sample.Value)
	}
	return values
}

func latestValueFrom(samples []models.Sample) float64 {
	if len(samples) == 0 {
		return math.NaN()
	}
	return samples[len(samples)-1].Value
}

func (m *Model) statsAvailable() bool {
	return m.statsEnabled && !m.lastStats.FetchedAt.IsZero()
}

func (m *Model) statsSnapshot() *models.DjangoStats {
	if !m.statsAvailable() {
		return nil
	}
	copy := m.lastStats
	if copy.Tasks == nil {
		copy.Tasks = map[string]int64{}
	}
	return &copy
}

func (m *Model) statsTaskCount(status string) (int64, bool) {
	if !m.statsAvailable() || m.lastStats.Tasks == nil {
		return 0, false
	}
	key := strings.ToUpper(strings.TrimSpace(status))
	if key == "" {
		return 0, false
	}
	val, ok := m.lastStats.Tasks[key]
	return val, ok
}

func (m *Model) statsWaitingCount() (int64, bool) {
	if !m.statsAvailable() {
		return 0, false
	}
	waiting, ok := m.statsTaskCount("WAITING")
	if !ok {
		waiting = 0
	}
	callback, ok := m.statsTaskCount("WAITING_CALLBACK")
	if !ok {
		callback = 0
	}
	if waiting == 0 && callback == 0 {
		_, okWaiting := m.statsTaskCount("WAITING")
		_, okCallback := m.statsTaskCount("WAITING_CALLBACK")
		if !okWaiting && !okCallback {
			return 0, false
		}
	}
	return waiting + callback, true
}

func (m *Model) statsWorkerCount() (int64, bool) {
	if !m.statsAvailable() {
		return 0, false
	}
	return int64(len(m.lastStats.Workers)), true
}

func (m *Model) statsQueueCount() (int64, bool) {
	queues := m.statsQueueNames()
	if len(queues) == 0 {
		return 0, false
	}
	return int64(len(queues)), true
}

func (m *Model) statsQueueNames() []string {
	if !m.statsAvailable() {
		return nil
	}
	seen := map[string]struct{}{}
	for _, worker := range m.lastStats.Workers {
		for _, queue := range worker.Queues {
			queue = strings.TrimSpace(queue)
			if queue == "" {
				continue
			}
			seen[queue] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	queues := make([]string, 0, len(seen))
	for queue := range seen {
		queues = append(queues, queue)
	}
	sort.Strings(queues)
	return queues
}

func (m *Model) statsPeriodicCount() (int64, bool) {
	if !m.statsAvailable() {
		return 0, false
	}
	return int64(len(m.lastStats.Periodic)), true
}

func (m *Model) statsNextPeriodic() (models.PeriodicTask, bool) {
	if !m.statsAvailable() {
		return models.PeriodicTask{}, false
	}
	periodic := m.statsPeriodicByNextRun()
	for _, task := range periodic {
		if !task.Enabled || task.NextRunAt.IsZero() {
			continue
		}
		return task, true
	}
	return models.PeriodicTask{}, false
}

func (m *Model) statsWorkersByRecent() []models.WorkerInfo {
	if !m.statsAvailable() {
		return nil
	}
	workers := append([]models.WorkerInfo(nil), m.lastStats.Workers...)
	sort.Slice(workers, func(i, j int) bool {
		return workers[i].LastSeenAt.After(workers[j].LastSeenAt)
	})
	return workers
}

func (m *Model) statsWorkerStatusCounts() (int, int, bool) {
	if !m.statsAvailable() {
		return 0, 0, false
	}
	now := m.referenceTime()
	cutoff := m.workerActiveCutoff(now)
	active := 0
	stale := 0
	for _, worker := range m.lastStats.Workers {
		if worker.LastSeenAt.IsZero() || worker.LastSeenAt.Before(cutoff) {
			stale++
			continue
		}
		active++
	}
	return active, stale, true
}

func (m *Model) splitWorkersByStatus(workers []models.WorkerInfo, now time.Time) ([]models.WorkerInfo, []models.WorkerInfo) {
	cutoff := m.workerActiveCutoff(now)
	active := make([]models.WorkerInfo, 0, len(workers))
	stale := make([]models.WorkerInfo, 0, len(workers))
	for _, worker := range workers {
		if worker.LastSeenAt.IsZero() || worker.LastSeenAt.Before(cutoff) {
			stale = append(stale, worker)
			continue
		}
		active = append(active, worker)
	}
	return active, stale
}

func (m *Model) workerActiveCutoff(now time.Time) time.Time {
	window := 90 * time.Second
	interval := m.cfg.StatsInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if candidate := interval * 6; candidate > window {
		window = candidate
	}
	return now.Add(-window)
}

func (m *Model) statsPeriodicByNextRun() []models.PeriodicTask {
	if !m.statsAvailable() {
		return nil
	}
	periodic := append([]models.PeriodicTask(nil), m.lastStats.Periodic...)
	sort.Slice(periodic, func(i, j int) bool {
		return periodic[i].NextRunAt.Before(periodic[j].NextRunAt)
	})
	return periodic
}

type queueSummary struct {
	Name    string
	Total   int64
	Ready   int64
	Waiting int64
	Running int64
	Failed  int64
}

func (m *Model) statsQueueSummaries() []queueSummary {
	if !m.statsAvailable() || len(m.lastStats.Queues) == 0 {
		return nil
	}
	out := make([]queueSummary, 0, len(m.lastStats.Queues))
	for name, counts := range m.lastStats.Queues {
		summary := queueSummary{Name: name}
		for status, count := range counts {
			switch strings.ToUpper(status) {
			case "READY":
				summary.Ready += count
			case "WAITING", "WAITING_CALLBACK":
				summary.Waiting += count
			case "RUNNING":
				summary.Running += count
			case "FAILED":
				summary.Failed += count
			}
			summary.Total += count
		}
		out = append(out, summary)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Total == out[j].Total {
			return out[i].Name < out[j].Name
		}
		return out[i].Total > out[j].Total
	})
	return out
}

func (m *Model) queueDepthValue() float64 {
	value := m.latestValue(metrics.MetricQueueDepth)
	if !math.IsNaN(value) {
		return value
	}
	ready, okReady := m.statsTaskCount("READY")
	waiting, okWaiting := m.statsWaitingCount()
	if !okReady && !okWaiting {
		return math.NaN()
	}
	return float64(ready + waiting)
}

func (m *Model) runningCountValue() float64 {
	value := m.latestValue(metrics.MetricTasksRunning)
	if !math.IsNaN(value) {
		return value
	}
	if running, ok := m.statsTaskCount("RUNNING"); ok {
		return float64(running)
	}
	return math.NaN()
}

func (m *Model) workerCountValue() float64 {
	value := m.latestValue(metrics.MetricWorkerCount)
	if !math.IsNaN(value) {
		return value
	}
	if count, ok := m.statsWorkerCount(); ok {
		return float64(count)
	}
	return math.NaN()
}

func (m *Model) referenceTime() time.Time {
	if !m.lastStats.FetchedAt.IsZero() {
		return m.lastStats.FetchedAt
	}
	if !m.lastScrapeAt.IsZero() {
		return m.lastScrapeAt
	}
	return time.Now()
}
