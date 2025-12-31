# reproq-tui

Modern, realtime terminal dashboard for Reproq Worker (and optional Reproq Django).
It polls the worker metrics and health endpoints, optionally consumes SSE events, and
renders a responsive Bubble Tea + Lip Gloss UI.

## Features

- Bubble Tea architecture with responsive, modern panels and theme fallbacks.
- Realtime metrics polling with rolling windows (1m/5m/15m).
- Throughput, queue depth, errors, and latency charts.
- Keyboard-driven UX with help overlay, filters, and snapshot export.
- Optional SSE events stream with reconnect/backoff.
- Cross-platform static binary release targets (macOS/Linux/Windows).

## Install

With Go:

```
go install github.com/adpena/reproq-tui/cmd/reproq-tui@latest
```

Homebrew:

```
brew tap adpena/tap
brew install reproq-tui
```

Curl (macOS/Linux):

```
curl -fsSL https://github.com/adpena/reproq-tui/releases/latest/download/install.sh | bash
```

From releases:

```
https://github.com/adpena/reproq-tui/releases
```

## Quick start

Recommended (one-time setup + auto-login):

1) Set `REPROQ_TUI_SECRET` on reproq-django (and reproq-worker if you want JWT auth for `/metrics`).
2) Run `reproq-tui setup --worker-url http://localhost:9100 --django-url http://localhost:8000`.
3) Run `reproq-tui dashboard` (auto-loads the generated config).

Run the dashboard (base URL):

```
reproq-tui dashboard --worker-url http://localhost:9100
```

Or use the full metrics URL:

```
reproq-tui dashboard --worker-metrics-url http://localhost:9100/metrics
```

Minimal env-based setup:

```
export REPROQ_TUI_WORKER_METRICS_URL=http://localhost:9100/metrics
export METRICS_AUTH_TOKEN=your-token
reproq-tui dashboard
```

If metrics/health are protected, pass a bearer token:

```
reproq-tui dashboard --worker-metrics-url http://localhost:9100/metrics --auth-token $METRICS_AUTH_TOKEN
```

Add Django stats (optional):

```
reproq-tui dashboard \
  --worker-metrics-url http://localhost:9100/metrics \
  --django-stats-url http://localhost:8000/reproq/stats/ \
  --auth-token $METRICS_AUTH_TOKEN
```

The Django stats API accepts `METRICS_AUTH_TOKEN` as a bearer token (or a TUI JWT).

Add health and events endpoints:

```
reproq-tui dashboard \
  --worker-metrics-url http://localhost:9100/metrics \
  --worker-health-url http://localhost:9100/healthz \
  --events-url http://localhost:9100/events
```

Run demo mode (mock server + UI):

```
reproq-tui demo
```

Demo mode emits `reproq_*` metrics that match the default catalog.

## Setup checklist

- Set `REPROQ_TUI_SECRET` on reproq-django (and reproq-worker if you want TUI login to authorize `/metrics`).
- Ensure the worker `/metrics`, `/healthz`, and `/events` are reachable from your machine.
- Run `reproq-tui setup --worker-url http://worker:9100 --django-url http://django:8000`.
- Launch the dashboard and sign in when prompted (press `l` to login/logout). The config file is auto-loaded.

Automation script:

```
scripts/setup.sh --worker-url http://worker:9100 --django-url http://django:8000
```

You can also pass `--worker-metrics-url` if you only have the full metrics URL.

## Authentication & access

### TUI login (recommended)

Configure a single secret in Django (and optionally the worker), then sign in once:

1) Set `REPROQ_TUI_SECRET` on `reproq-django`. This enables the TUI login endpoints.
2) Set the same `REPROQ_TUI_SECRET` on `reproq-worker` so the JWT is accepted for `/metrics`, `/healthz`, and `/events`.
3) Run the TUI with `--django-url` (or `--django-stats-url`) and press `l` to log in.
4) Open the URL shown in the TUI, sign in as a superuser, and approve.

The TUI stores the token locally and reuses it on restart until you log out (`l`).
Stored tokens live in `~/.config/reproq-tui/auth.json` (override with `REPROQ_TUI_AUTH_FILE`).
If your stats endpoint is not `/reproq/stats/`, set `--django-url` explicitly (use the base URL without `/reproq`).
If `--django-url` is not set, press `l` and paste the Django base URL (https optional; backslashes are normalized).

Example:

```
reproq-tui dashboard --worker-url http://localhost:9100 --django-url http://localhost:8000
```

CLI automation:

```
reproq-tui login --django-url http://localhost:8000
reproq-tui logout
```

### Static token (alternative)

Recommended flow (single token everywhere):

1) In `reproq-worker`, set `metrics.auth_token` (config file) or `METRICS_AUTH_TOKEN` (env).
   This protects `/metrics`, `/healthz`, and `/events` with `Authorization: Bearer <token>`.
2) In `reproq-django`, set `METRICS_AUTH_TOKEN` for `/reproq/stats/` (use the same value for worker + stats).
3) In `reproq-tui`, pass `--auth-token` or set `REPROQ_TUI_AUTH_TOKEN` (falls back to `METRICS_AUTH_TOKEN`).

Custom headers:

- Use `--header "X-Reproq-Token: <token>"` or `headers:` in the config file to send a non-Bearer header.
- If you set an `Authorization` header manually, it is respected and `--auth-token` will not override it.

## Configuration

Config sources (highest to lowest precedence): flags, env vars, config file, defaults.

If `~/.config/reproq-tui/config.yaml` (or the platform equivalent) exists, it is
auto-loaded. Use `--config` or `REPROQ_TUI_CONFIG` to override.
`reproq-tui setup` writes to this path by default.

Note: the default catalog matches `reproq-worker` metrics (v0.0.133+). Map
canonical metrics to your worker's metric names via `--metric` or the config file.

Flags (selected):

- `--worker-url` (base URL; derives `/metrics` and `/healthz`)
- `--worker-metrics-url` (required unless `--worker-url` is set)
- `--worker-health-url`
- `--events-url`
- `--django-url`
- `--django-stats-url`
- `--interval` (default `1s`)
- `--health-interval` (default `500ms`)
- `--stats-interval` (default `5s`)
- `--window` (default `5m`)
- `--theme` (`auto`, `dark`, `light`)
- `--auto-login` (default `true`)
- `--header "Key: Value"` (repeatable)
- `--auth-token` (adds `Authorization: Bearer <token>` header)
- `--timeout` (default `2s`)
- `--insecure-skip-verify` (dev only)
- `--metric canonical=actual` (repeatable)
- `--log-file /path/to/reproq-tui.log`

Env vars:

- `REPROQ_TUI_CONFIG`
- `REPROQ_TUI_WORKER_URL`
- `REPROQ_TUI_WORKER_METRICS_URL`
- `REPROQ_TUI_WORKER_HEALTH_URL`
- `REPROQ_TUI_EVENTS_URL`
- `REPROQ_TUI_DJANGO_URL`
- `REPROQ_TUI_DJANGO_STATS_URL`
- `REPROQ_TUI_INTERVAL`
- `REPROQ_TUI_HEALTH_INTERVAL`
- `REPROQ_TUI_STATS_INTERVAL`
- `REPROQ_TUI_WINDOW`
- `REPROQ_TUI_THEME`
- `REPROQ_TUI_AUTO_LOGIN`
- `REPROQ_TUI_AUTH_FILE`
- `REPROQ_TUI_HEADERS` (comma-separated `Key: Value`)
- `REPROQ_TUI_AUTH_TOKEN` (falls back to `METRICS_AUTH_TOKEN`)
- `REPROQ_TUI_TIMEOUT`
- `REPROQ_TUI_INSECURE_SKIP_VERIFY`
- `REPROQ_TUI_METRICS` (comma-separated `canonical=actual`)
- `REPROQ_TUI_LOG_FILE`

Config file (YAML or TOML) with `--config` (use `worker_url` or explicit URLs):

```yaml
worker_url: http://localhost:9100
# worker_metrics_url: http://localhost:9100/metrics
# worker_health_url: http://localhost:9100/healthz
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
  tasks_failed_total: worker_tasks_total{status="failure"}
```

Metric mappings support label selectors in Prometheus format (for example
`reproq_tasks_processed_total{status="failure"}`).

## Keybindings

- `q` quit
- `?` help
- `p` pause/resume
- `r` refresh
- `1/2/3` switch window (1m/5m/15m)
- `tab` next pane (dashboard) or next details tab
- `/` filter input
- `e` toggle events pane
- `t` toggle theme
- `s` export snapshot JSON
- `d` open details (queues/workers/tasks/errors)
- `l` login/logout

Filter tips:
- Use `queue:default`, `worker:worker-1`, or `task:123` to apply server-side SSE filters.
- Combine with text (for example `queue:default error`) to keep local filtering on the remaining text.

## Snapshot export

Press `s` to export a JSON snapshot of current state and recent series points.
Files are written to the current working directory with a timestamped name.

## Documentation

- `docs/ARCHITECTURE.md`
- `docs/METRICS.md`
- `docs/EVENTS.md`
- `docs/DEVELOPMENT.md`

## Recommended terminal settings

- Terminal size: 120x30 or larger.
- Enable truecolor if your terminal supports it.
- Use a monospace font with good box-drawing glyphs.

## Screenshots

- macOS: use `cmd+shift+4`
- Linux: use your desktop screenshot tool or `gnome-screenshot`
- Windows: use `Win+Shift+S`

## Development

Go 1.24.2+ required.

```
make fmt
make test
make lint
make build
```

## Troubleshooting

- Timeouts: increase `--timeout` or check network connectivity.
- Missing metrics: map canonical keys in `docs/METRICS.md`.
- Colors look off: set `--theme dark|light` or check terminal truecolor support.
- Windows terminals: prefer Windows Terminal or a recent PowerShell with UTF-8.

## License

See `LICENSE`.
