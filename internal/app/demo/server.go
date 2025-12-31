package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Server struct {
	Listener   net.Listener
	HTTPServer *http.Server
	BaseURL    string
	MetricsURL string
	HealthURL  string
	EventsURL  string
	StatsURL   string
	state      *state
}

func Start() (*Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	baseURL := fmt.Sprintf("http://%s", listener.Addr().String())
	state := newState()

	mux := http.NewServeMux()
	server := &Server{
		Listener:   listener,
		BaseURL:    baseURL,
		MetricsURL: baseURL + "/metrics",
		HealthURL:  baseURL + "/healthz",
		EventsURL:  baseURL + "/events",
		StatsURL:   baseURL + "/stats",
		state:      state,
	}

	mux.HandleFunc("/metrics", server.handleMetrics)
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/healthz", server.handleHealth)
	mux.HandleFunc("/events", server.handleEvents)
	mux.HandleFunc("/stats", server.handleStats)

	server.HTTPServer = &http.Server{
		Handler: mux,
	}
	go server.HTTPServer.Serve(listener)
	go state.run()

	return server, nil
}

func (s *Server) Close(ctx context.Context) error {
	s.state.stop()
	return s.HTTPServer.Shutdown(ctx)
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	s.state.writeMetrics(w)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	s.state.mu.RLock()
	health := s.state.health
	s.state.mu.RUnlock()

	payload := map[string]string{
		"status":  health.status,
		"version": "demo",
		"build":   "dev",
		"commit":  "demo",
		"message": health.message,
	}
	if health.httpStatus != http.StatusOK {
		w.WriteHeader(health.httpStatus)
	}
	if health.json {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(health.message))
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	ticker := time.NewTicker(1200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			event := s.state.nextEvent()
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	s.state.mu.RLock()
	queueDepth := s.state.queueDepth
	tasksRunning := s.state.tasksRunning
	tasksTotal := s.state.tasksTotal
	tasksFailed := s.state.tasksFailed
	workerCount := s.state.workerCount
	concurrencyLimit := s.state.concurrencyLimit
	s.state.mu.RUnlock()

	ready := int64(math.Round(queueDepth * 0.6))
	if ready < 0 {
		ready = 0
	}
	waiting := int64(math.Round(queueDepth)) - ready
	if waiting < 0 {
		waiting = 0
	}
	running := int64(math.Round(tasksRunning))
	if running < 0 {
		running = 0
	}
	failed := int64(math.Round(tasksFailed))
	if failed < 0 {
		failed = 0
	}
	success := int64(math.Round(tasksTotal)) - failed
	if success < 0 {
		success = 0
	}
	queues := map[string]map[string]int64{}
	queueNames := []string{"default", "fast", "slow"}
	perQueueReady := ready / int64(len(queueNames))
	perQueueWaiting := waiting / int64(len(queueNames))
	perQueueRunning := running / int64(len(queueNames))
	perQueueFailed := failed / int64(len(queueNames))
	for i, name := range queueNames {
		qReady := perQueueReady
		qWaiting := perQueueWaiting
		qRunning := perQueueRunning
		qFailed := perQueueFailed
		if i == 0 {
			qReady += ready % int64(len(queueNames))
			qWaiting += waiting % int64(len(queueNames))
			qRunning += running % int64(len(queueNames))
			qFailed += failed % int64(len(queueNames))
		}
		queues[name] = map[string]int64{
			"READY":   qReady,
			"WAITING": qWaiting,
			"RUNNING": qRunning,
			"FAILED":  qFailed,
		}
	}

	workers := make([]map[string]interface{}, 0, int(workerCount))
	workerTotal := int(math.Max(1, math.Round(workerCount)))
	perWorker := int(math.Max(1, math.Round(concurrencyLimit/float64(workerTotal))))
	for i := 0; i < workerTotal; i++ {
		workers = append(workers, map[string]interface{}{
			"worker_id":    fmt.Sprintf("worker-%d", i+1),
			"hostname":     fmt.Sprintf("demo-%d", i+1),
			"concurrency":  perWorker,
			"queues":       []string{"default", "fast", "slow"},
			"last_seen_at": time.Now().Add(-time.Duration(i) * 8 * time.Second),
			"version":      "demo",
		})
	}

	periodic := []map[string]interface{}{
		{
			"name":        "cleanup",
			"cron_expr":   "*/5 * * * *",
			"enabled":     true,
			"next_run_at": time.Now().Add(2 * time.Minute),
		},
		{
			"name":        "sync-analytics",
			"cron_expr":   "*/15 * * * *",
			"enabled":     true,
			"next_run_at": time.Now().Add(6 * time.Minute),
		},
	}

	payload := map[string]interface{}{
		"tasks": map[string]int64{
			"READY":      ready,
			"WAITING":    waiting,
			"RUNNING":    running,
			"FAILED":     failed,
			"SUCCESSFUL": success,
		},
		"queues":   queues,
		"workers":  workers,
		"periodic": periodic,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

type state struct {
	mu               sync.RWMutex
	rnd              *rand.Rand
	startedAt        time.Time
	queueDepth       float64
	tasksTotal       float64
	tasksFailed      float64
	tasksRunning     float64
	workerCount      float64
	concurrencyInUse float64
	concurrencyLimit float64
	latencyP95       float64
	health           healthState
	stopCh           chan struct{}
}

func newState() *state {
	return &state{
		rnd:              rand.New(rand.NewSource(time.Now().UnixNano())),
		startedAt:        time.Now(),
		queueDepth:       120,
		tasksTotal:       4200,
		tasksFailed:      42,
		tasksRunning:     18,
		workerCount:      3,
		concurrencyInUse: 18,
		concurrencyLimit: 32,
		latencyP95:       0.32,
		health: healthState{
			status:     "ok",
			message:    "healthy",
			httpStatus: http.StatusOK,
			json:       true,
		},
		stopCh: make(chan struct{}),
	}
}

func (s *state) run() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.tick()
		case <-s.stopCh:
			return
		}
	}
}

func (s *state) stop() {
	close(s.stopCh)
}

func (s *state) tick() {
	s.mu.Lock()
	defer s.mu.Unlock()
	shift := s.rnd.Float64()*6 - 3
	s.queueDepth = clamp(s.queueDepth+shift, 10, 300)
	throughput := clamp(s.rnd.NormFloat64()*2+6, 1, 12)
	s.tasksTotal += throughput
	fail := s.rnd.Float64() < 0.08
	if fail {
		s.tasksFailed += clamp(s.rnd.Float64()*1.2, 0.2, 1.2)
	}
	s.tasksRunning = clamp(s.tasksRunning+shift/6, 4, s.concurrencyLimit)
	s.concurrencyInUse = s.tasksRunning
	s.workerCount = clamp(s.workerCount+s.rnd.NormFloat64()*0.1, 2, 6)
	s.latencyP95 = clamp(0.2+math.Abs(s.rnd.NormFloat64()*0.15), 0.08, 1.2)
	s.rotateHealth()
}

func (s *state) writeMetrics(w http.ResponseWriter) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fmt.Fprintf(w, "# HELP reproq_queue_depth Queue depth\n")
	fmt.Fprintf(w, "# TYPE reproq_queue_depth gauge\n")
	fmt.Fprintf(w, "reproq_queue_depth %.0f\n", s.queueDepth)

	fmt.Fprintf(w, "# HELP reproq_tasks_running Running tasks\n")
	fmt.Fprintf(w, "# TYPE reproq_tasks_running gauge\n")
	fmt.Fprintf(w, "reproq_tasks_running %.0f\n", s.tasksRunning)

	fmt.Fprintf(w, "# HELP reproq_tasks_processed_total Total tasks processed\n")
	fmt.Fprintf(w, "# TYPE reproq_tasks_processed_total counter\n")
	success := s.tasksTotal - s.tasksFailed
	if success < 0 {
		success = 0
	}
	fmt.Fprintf(w, "reproq_tasks_processed_total{status=\"success\",queue=\"default\"} %.0f\n", success)
	fmt.Fprintf(w, "reproq_tasks_processed_total{status=\"failure\",queue=\"default\"} %.0f\n", s.tasksFailed)

	fmt.Fprintf(w, "# HELP reproq_workers Worker count\n")
	fmt.Fprintf(w, "# TYPE reproq_workers gauge\n")
	fmt.Fprintf(w, "reproq_workers %.0f\n", s.workerCount)

	fmt.Fprintf(w, "# HELP reproq_concurrency_in_use Concurrency in use\n")
	fmt.Fprintf(w, "# TYPE reproq_concurrency_in_use gauge\n")
	fmt.Fprintf(w, "reproq_concurrency_in_use %.0f\n", s.concurrencyInUse)

	fmt.Fprintf(w, "# HELP reproq_concurrency_limit Concurrency limit\n")
	fmt.Fprintf(w, "# TYPE reproq_concurrency_limit gauge\n")
	fmt.Fprintf(w, "reproq_concurrency_limit %.0f\n", s.concurrencyLimit)

	fmt.Fprintf(w, "# HELP reproq_exec_duration_seconds Task execution duration\n")
	fmt.Fprintf(w, "# TYPE reproq_exec_duration_seconds histogram\n")
	s.writeExecHistogram(w)
}

func (s *state) writeExecHistogram(w http.ResponseWriter) {
	bounds := []float64{0.05, 0.1, 0.2, 0.35, 0.5, 0.75, 1.0, 1.5, 2.0}
	total := int64(math.Max(1, math.Round(s.tasksTotal)))
	p95 := s.latencyP95
	p95Index := len(bounds) - 1
	for idx, bound := range bounds {
		if bound >= p95 {
			p95Index = idx
			break
		}
	}
	var cumulative int64
	for idx, bound := range bounds {
		ratio := 1.0
		switch {
		case idx < p95Index:
			step := float64(idx+1) / float64(p95Index+1)
			ratio = 0.2 + 0.7*step
		case idx == p95Index:
			ratio = 0.95
		default:
			tail := float64(idx-p95Index) / float64(len(bounds)-p95Index)
			ratio = 0.95 + 0.05*tail
		}
		count := int64(math.Round(float64(total) * ratio))
		if count < cumulative {
			count = cumulative
		}
		if count > total {
			count = total
		}
		cumulative = count
		fmt.Fprintf(w, "reproq_exec_duration_seconds_bucket{le=\"%s\"} %d\n", formatBound(bound), count)
	}
	fmt.Fprintf(w, "reproq_exec_duration_seconds_bucket{le=\"+Inf\"} %d\n", total)
	fmt.Fprintf(w, "reproq_exec_duration_seconds_sum %.4f\n", p95*float64(total))
	fmt.Fprintf(w, "reproq_exec_duration_seconds_count %d\n", total)
}

func formatBound(bound float64) string {
	return strconv.FormatFloat(bound, 'f', -1, 64)
}

type healthState struct {
	status     string
	message    string
	httpStatus int
	json       bool
}

func (s *state) rotateHealth() {
	uptime := time.Since(s.startedAt)
	switch {
	case uptime%(40*time.Second) < 5*time.Second:
		s.health = healthState{
			status:     "degraded",
			message:    "transient latency spike",
			httpStatus: http.StatusOK,
			json:       true,
		}
	case uptime%(75*time.Second) < 4*time.Second:
		s.health = healthState{
			status:     "down",
			message:    "database unavailable",
			httpStatus: http.StatusServiceUnavailable,
			json:       s.rnd.Float64() < 0.5,
		}
	default:
		s.health = healthState{
			status:     "ok",
			message:    "healthy",
			httpStatus: http.StatusOK,
			json:       true,
		}
	}
}

func (s *state) nextEvent() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	level := "info"
	msg := "task completed"
	etype := "task_completed"
	if s.rnd.Float64() < 0.2 {
		level = "warn"
		msg = "retrying task"
		etype = "task_retry"
	}
	if s.rnd.Float64() < 0.08 {
		level = "error"
		msg = "task failed"
		etype = "task_failed"
	}
	return map[string]interface{}{
		"ts":        time.Now().Format(time.RFC3339Nano),
		"level":     level,
		"type":      etype,
		"msg":       msg,
		"queue":     fmt.Sprintf("queue-%d", 1+s.rnd.Intn(3)),
		"worker_id": fmt.Sprintf("worker-%d", 1+s.rnd.Intn(4)),
	}
}

func clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
