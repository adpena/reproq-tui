package metrics

const (
	MetricQueueDepth       = "queue_depth"
	MetricTasksTotal       = "tasks_total"
	MetricTasksFailed      = "tasks_failed_total"
	MetricTasksRunning     = "tasks_running"
	MetricWorkerCount      = "worker_count"
	MetricConcurrencyInUse = "concurrency_in_use"
	MetricConcurrencyLimit = "concurrency_limit"
	MetricLatencyP95       = "latency_p95"

	// New telemetry metrics
	MetricWorkerMemUsage       = "worker_mem_usage"
	MetricDBPoolConnections    = "db_pool_conns"
	MetricDBPoolWait           = "db_pool_wait"
)

type Catalog struct {
	Mapping   map[string]string
	Selectors map[string]Selector
}

func DefaultCatalog() Catalog {
	mapping := map[string]string{
		MetricQueueDepth:        "reproq_queue_depth",
		MetricTasksTotal:        "reproq_tasks_processed_total",
		MetricTasksFailed:       "reproq_tasks_processed_total{status=\"failure\"}",
		MetricTasksRunning:      "reproq_tasks_running",
		MetricWorkerCount:       "reproq_workers",
		MetricConcurrencyInUse:  "reproq_concurrency_in_use",
		MetricConcurrencyLimit:  "reproq_concurrency_limit",
		MetricLatencyP95:        "reproq_exec_duration_seconds",
		MetricWorkerMemUsage:    "reproq_worker_mem_usage_bytes",
		MetricDBPoolConnections: "reproq_db_pool_connections_in_use",
		MetricDBPoolWait:        "reproq_db_pool_wait_count_total",
	}
	return Catalog{
		Mapping:   mapping,
		Selectors: compileSelectors(mapping),
	}
}

func NewCatalog(overrides map[string]string) Catalog {
	catalog := DefaultCatalog()
	for key, val := range overrides {
		if val != "" {
			catalog.Mapping[key] = val
		}
	}
	catalog.Selectors = compileSelectors(catalog.Mapping)
	return catalog
}

func (c Catalog) Name(key string) string {
	if val, ok := c.Mapping[key]; ok && val != "" {
		return val
	}
	return ""
}
