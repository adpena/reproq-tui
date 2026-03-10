# Reproq TUI

`reproq-tui` is a realtime terminal dashboard for the Reproq task stack. It gives operators and developers a fast, keyboard-driven view of queue depth, throughput, latency, failures, worker health, and optional Django-side rollups without opening a browser.

The project is designed for teams running [Reproq Worker](https://github.com/adpena/reproq-worker) directly or alongside [Reproq Django](https://github.com/adpena/reproq-django). It works well for local development, production debugging, incident response, and operator-facing demos.

## Why It Exists

Reproq Worker exposes rich operational signals, but raw metrics endpoints are not the best day-to-day interface for humans. `reproq-tui` turns those signals into an interactive terminal experience that is:

- Fast to launch
- Friendly over SSH
- Useful during incidents
- Easy to hand to engineers or operators who do not want to live in Prometheus queries

## What It Visualizes

`reproq-tui` can combine multiple sources of runtime information in one place:

- Worker metrics from `/metrics`
- Worker health from `/healthz`
- Optional SSE events from `/events`
- Optional Django rollups from `/reproq/stats/`
- Saved local config and auth state for repeat runs

The dashboard supports rolling windows, theme fallbacks, filters, overlays, and snapshot export.

## Highlights

- Realtime queue, throughput, latency, and error views
- Bubble Tea + Lip Gloss interface with responsive panels
- Optional Django-aware overlays for paused queues and worker rollups
- Optional SSE stream support with reconnect and backoff
- Interactive setup flow for first-time users
- Local auth/token storage for repeat usage
- Cross-platform release targets for macOS, Linux, and Windows

## Installation

### Go

```bash
go install github.com/adpena/reproq-tui/cmd/reproq-tui@latest
```

### Homebrew

```bash
brew tap adpena/tap
brew install reproq-tui
```

### Release Installer

```bash
curl -fsSL https://github.com/adpena/reproq-tui/releases/latest/download/install.sh | bash
```

### Manual Downloads

Prebuilt binaries are published on the releases page:

`https://github.com/adpena/reproq-tui/releases`

## Quick Start

### Fastest Path: Direct Worker Metrics

If you already know the worker endpoint:

```bash
reproq-tui dashboard --worker-url http://localhost:9100
```

Or point directly at the metrics endpoint:

```bash
reproq-tui dashboard --worker-metrics-url http://localhost:9100/metrics
```

If metrics are protected:

```bash
reproq-tui dashboard \
  --worker-metrics-url http://localhost:9100/metrics \
  --auth-token "$METRICS_AUTH_TOKEN"
```

### Recommended Path: Django-Assisted Setup

If you run `reproq-django`, the smoothest onboarding flow is:

1. Set `REPROQ_TUI_SECRET` on `reproq-django`.
2. Set the same `REPROQ_TUI_SECRET` on `reproq-worker` if you want the issued JWT to authorize `/metrics`, `/healthz`, and `/events`.
3. Optionally set `REPROQ_TUI_WORKER_INTERNAL_URL` or `REPROQ_TUI_WORKER_URL` on `reproq-django` so it can hand the worker endpoints to the TUI automatically.
4. Launch `reproq-tui` and paste the base Django URL when prompted.

```bash
reproq-tui
```

Or be explicit:

```bash
reproq-tui dashboard --django-url http://localhost:8000
```

When Django is configured, the TUI can bootstrap from `/reproq/tui/config/`, start the login flow, and avoid prompting for worker URLs unless it truly needs them.

## Authentication

### Recommended: TUI Login

The preferred flow is a one-time login mediated by `reproq-django`:

1. Configure `REPROQ_TUI_SECRET` in Django.
2. Optionally share the same secret with the worker.
3. Launch the TUI and press `l` if prompted.
4. Open the approval URL in a browser, authenticate, and approve the session.

The TUI stores the resulting token locally and reuses it on later runs until you log out.

Stored auth defaults to:

`~/.config/reproq-tui/auth.json`

Override with:

`REPROQ_TUI_AUTH_FILE`

### Alternative: Static Bearer Token

If you prefer a simpler deployment:

1. Protect `reproq-worker` metrics endpoints with `METRICS_AUTH_TOKEN` or the equivalent worker config.
2. Configure the same token on `reproq-django` if you want protected stats there too.
3. Pass the token to the TUI:

```bash
export METRICS_AUTH_TOKEN=your-token
reproq-tui dashboard --worker-metrics-url http://localhost:9100/metrics
```

You can also provide custom headers via repeated `--header "Key: Value"` flags or config file entries.

## Common Workflows

### Worker + Django stats

```bash
reproq-tui dashboard \
  --worker-metrics-url http://localhost:9100/metrics \
  --django-stats-url http://localhost:8000/reproq/stats/ \
  --auth-token "$METRICS_AUTH_TOKEN"
```

### Worker + health + events

```bash
reproq-tui dashboard \
  --worker-metrics-url http://localhost:9100/metrics \
  --worker-health-url http://localhost:9100/healthz \
  --events-url http://localhost:9100/events
```

### Demo mode

```bash
reproq-tui demo
```

Demo mode starts a mock backend and emits `reproq_*` metrics that match the default catalog, making it useful for screenshots, quick exploration, and smoke testing.

## Setup Checklist

- Ensure the worker exposes `/metrics`
- Expose `/healthz` and `/events` if you want richer operational views
- Configure `reproq-django` if you want the guided login/bootstrap flow
- Decide whether you want JWT-based TUI login or a static bearer token
- Save your preferred defaults in a config file if you run the tool often

An automation helper is also included:

```bash
scripts/setup.sh --worker-url http://worker:9100 --django-url http://django:8000
```

## Configuration

Configuration precedence is:

1. CLI flags
2. Environment variables
3. Config file
4. Built-in defaults

If a config file exists at the platform default location, it is loaded automatically. Override with:

- `--config`
- `REPROQ_TUI_CONFIG`

Example config:

```yaml
worker_url: http://localhost:9100
django_url: http://localhost:8000
events_url: http://localhost:9100/events
django_stats_url: http://localhost:8000/reproq/stats/
interval: 1s
health_interval: 500ms
stats_interval: 5s
window: 5m
theme: auto
auto_login: true
timeout: 2s
auth_token: TOKEN
headers:
  - "X-Reproq-Token: TOKEN"
metrics:
  queue_depth: worker_queue_depth
  tasks_total: worker_tasks_total
```

Frequently used environment variables:

- `REPROQ_TUI_WORKER_URL`
- `REPROQ_TUI_WORKER_METRICS_URL`
- `REPROQ_TUI_WORKER_HEALTH_URL`
- `REPROQ_TUI_EVENTS_URL`
- `REPROQ_TUI_DJANGO_URL`
- `REPROQ_TUI_DJANGO_STATS_URL`
- `REPROQ_TUI_AUTH_TOKEN`
- `REPROQ_TUI_HEADERS`
- `REPROQ_TUI_THEME`
- `REPROQ_TUI_LOG_FILE`

## Relationship to the Reproq Stack

- [`reproq-django`](https://github.com/adpena/reproq-django) handles task definition, enqueueing, Django Admin integration, and TUI login/bootstrap endpoints.
- [`reproq-worker`](https://github.com/adpena/reproq-worker) executes tasks, exposes runtime metrics, and powers the operational data shown in the TUI.

If you want the best operator experience, run all three together.

## When To Use It

`reproq-tui` is a good fit when you want:

- Lightweight observability during local development
- A terminal-first operator workflow
- Quick visibility into task system health without opening a browser
- A simple way to demo Reproq’s operational story

If you need long-term time series, alerts, and dashboards for large teams, pair it with your usual observability stack rather than treating it as a replacement.

## Further Reading

- `docs/ARCHITECTURE.md`
- `docs/METRICS.md`
- `docs/EVENTS.md`
- `docs/DEVELOPMENT.md`

## Development

For local development:

```bash
make fmt
make test
make lint
make build
```

## Troubleshooting

- If metrics are timing out, increase `--timeout` and verify network reachability.
- If the wrong metric names appear, map them explicitly in config or review `docs/METRICS.md`.
- If colors or layout look wrong, set `--theme dark` or `--theme light` and verify your terminal supports modern box-drawing/truecolor output.
- If auth works in Django but not on worker endpoints, verify the shared secret or bearer token is configured consistently across services.

## License

Apache License 2.0. See [LICENSE](LICENSE).
