package models

import "time"

type Sample struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type MetricSnapshot struct {
	CollectedAt time.Time          `json:"collected_at"`
	Latency     time.Duration      `json:"latency"`
	Values      map[string]float64 `json:"values"`
}

type HealthStatus struct {
	Healthy   bool          `json:"healthy"`
	Status    string        `json:"status"`
	Version   string        `json:"version,omitempty"`
	Build     string        `json:"build,omitempty"`
	Commit    string        `json:"commit,omitempty"`
	Message   string        `json:"message,omitempty"`
	CheckedAt time.Time     `json:"checked_at"`
	Latency   time.Duration `json:"latency"`
}

type Event struct {
	Timestamp time.Time         `json:"ts"`
	Level     string            `json:"level"`
	Type      string            `json:"type"`
	Message   string            `json:"msg"`
	Queue     string            `json:"queue,omitempty"`
	TaskID    string            `json:"task_id,omitempty"`
	WorkerID  string            `json:"worker_id,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type DjangoStats struct {
	Tasks         map[string]int64            `json:"tasks"`
	Queues        map[string]map[string]int64 `json:"queues"`
	Workers       []WorkerInfo                `json:"workers"`
	Periodic      []PeriodicTask              `json:"periodic"`
	QueueControls []QueueControl              `json:"queue_controls,omitempty"`
	WorkerHealth  *WorkerHealth               `json:"worker_health,omitempty"`
	Scheduler     *SchedulerStatus            `json:"scheduler,omitempty"`
	TopFailing    []FailingTask               `json:"top_failing"`
	Databases     []DatabaseStats             `json:"databases,omitempty"`
	FetchedAt     time.Time                   `json:"fetched_at,omitempty"`
}

type SchedulerStatus struct {
	Mode            string `json:"mode"`
	LowMemory       bool   `json:"low_memory"`
	BeatEnabled     bool   `json:"beat_enabled"`
	BeatConfigured  bool   `json:"beat_configured"`
	PgCronAvailable bool   `json:"pg_cron_available"`
	Warning         string `json:"warning,omitempty"`
}

type FailingTask struct {
	TaskPath string `json:"task_path"`
	Count    int64  `json:"count"`
}

type WorkerInfo struct {
	WorkerID    string    `json:"worker_id"`
	Hostname    string    `json:"hostname"`
	Concurrency int       `json:"concurrency"`
	Queues      []string  `json:"queues"`
	LastSeenAt  time.Time `json:"last_seen_at"`
	Version     string    `json:"version"`
}

type PeriodicTask struct {
	Name      string    `json:"name"`
	CronExpr  string    `json:"cron_expr"`
	Enabled   bool      `json:"enabled"`
	NextRunAt time.Time `json:"next_run_at"`
}

type WorkerHealth struct {
	Alive int `json:"alive"`
	Dead  int `json:"dead"`
}

type QueueControl struct {
	QueueName string    `json:"queue_name"`
	Paused    bool      `json:"paused"`
	PausedAt  time.Time `json:"paused_at"`
	Reason    string    `json:"reason"`
	UpdatedAt time.Time `json:"updated_at"`
	Database  string    `json:"database,omitempty"`
}

type DatabaseStats struct {
	Alias    string                      `json:"alias"`
	Tasks    map[string]int64            `json:"tasks"`
	Queues   map[string]map[string]int64 `json:"queues"`
	Workers  []WorkerInfo                `json:"workers"`
	Periodic []PeriodicTask              `json:"periodic"`
}
