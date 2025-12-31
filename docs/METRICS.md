# Metrics Catalog

reproq-tui uses a small canonical catalog to map worker metrics into the UI. The
canonical keys are stable across versions; the actual metric names are
configurable via flags, env, or config file.

## Canonical keys

Key | Type | Description
--- | ---- | -----------
queue_depth | gauge | total queue depth across queues
tasks_total | counter | total tasks processed
tasks_failed_total | counter | total failed tasks
tasks_running | gauge | currently running tasks
worker_count | gauge | active workers
concurrency_in_use | gauge | active concurrency slots
concurrency_limit | gauge | total concurrency capacity
latency_p95 | summary or histogram | p95 task execution latency

## Default mapping

The default mapping matches `reproq-worker` metrics (v0.0.133+). If your worker
emits different names, use the mapping options below to align the catalog.

- queue_depth -> reproq_queue_depth
- tasks_total -> reproq_tasks_processed_total
- tasks_failed_total -> reproq_tasks_processed_total{status="failure"}
- tasks_running -> reproq_tasks_running
- worker_count -> reproq_workers
- concurrency_in_use -> reproq_concurrency_in_use
- concurrency_limit -> reproq_concurrency_limit
- latency_p95 -> reproq_exec_duration_seconds

## Mapping via flags

```
reproq-tui dashboard \
  --worker-metrics-url http://localhost:9100/metrics \
  --metric queue_depth=worker_queue_depth \
  --metric tasks_total=worker_tasks_total \
  --metric tasks_failed_total=worker_tasks_total{status="failure"}
```

## Mapping via env

```
REPROQ_TUI_METRICS=queue_depth=worker_queue_depth,tasks_total=worker_tasks_total,tasks_failed_total=worker_tasks_total{status=\"failure\"}
```

## Mapping via config file

```yaml
metrics:
  queue_depth: worker_queue_depth
  tasks_total: worker_tasks_total
  tasks_failed_total: worker_tasks_total{status="failure"}
```

## Missing metrics

If a metric is missing, the UI shows "-" and continues running. Counters that
are missing will simply report no derived rates.

`queue_depth`, `tasks_running`, `worker_count`, `concurrency_in_use`, and
`concurrency_limit` are provided by `reproq-worker` (v0.0.133+) via lightweight
DB-derived gauges. If you are running an older worker, map these to equivalent
metrics or leave them unmapped.

## Latency quantiles

If `latency_p95` is a summary, the p95 quantile is read directly. If
it is a histogram, p95 is approximated using bucket counts.
