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
)

type Catalog struct {
	Mapping   map[string]string
	Selectors map[string]Selector
}

func DefaultCatalog() Catalog {
	mapping := map[string]string{
		MetricQueueDepth:       "reproq_queue_depth",
		MetricTasksTotal:       "reproq_tasks_processed_total",
		MetricTasksFailed:      "reproq_tasks_processed_total{status=\"failure\"}",
		MetricTasksRunning:     "reproq_tasks_running",
		MetricWorkerCount:      "reproq_workers",
		MetricConcurrencyInUse: "reproq_concurrency_in_use",
		MetricConcurrencyLimit: "reproq_concurrency_limit",
		MetricLatencyP95:       "reproq_exec_duration_seconds",
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
