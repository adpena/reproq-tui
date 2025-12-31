# Architecture

This document describes the design, package boundaries, and runtime behavior of
reproq-tui.

## Package boundaries

- cmd/reproq-tui
  - Entry point that initializes and runs the Cobra CLI.
- internal/app/cmd
  - CLI commands, flag registration, and runtime wiring.
- internal/app/demo
  - Demo HTTP server for /metrics, /healthz, /events, and /stats.
- internal/config
  - Config loading from flags, env, and optional file.
- internal/metrics
  - Prometheus parsing, metric catalog, ring buffers, and derived metrics.
- internal/health
  - Health endpoint polling and status parsing.
- internal/stats
  - Django stats API polling and JSON decoding.
- internal/auth
  - Pairing flow with reproq-django and persistent token storage.
- internal/events
  - SSE client with reconnect/backoff and event buffer.
- internal/ui
  - Bubble Tea model/update/view, keymap, and layout rendering.
- internal/charts
  - Sparkline, bar, and gauge renderers.
- internal/theme
  - Theme palettes and terminal capability detection.
- pkg/client
  - HTTP client wrapper with headers and timeouts.
- pkg/models
  - Shared model structs for snapshots, health, events, and Django stats.

## Data flow

Pollers -> parsers -> snapshots -> ring buffers -> Bubble Tea model -> view

1) Pollers (internal/ui/update.go)
   - tea.Cmd functions fetch /metrics, /healthz, and /stats asynchronously.
   - Commands return message structs (metricsMsg, healthMsg, statsMsg).

2) Parsers and snapshots (internal/metrics/prom.go)
   - Prometheus text parsing produces a MetricSnapshot.
   - Missing metrics return NaN to keep UI running.

3) Ring buffers (internal/metrics/ring.go)
   - Each metric has a bounded ring buffer of Samples.
   - ValuesSince(window) filters samples for current window.

4) Derived metrics (internal/metrics/derived.go)
   - Rate, delta, and ratio computations derived from counters.

5) Tea model (internal/ui/model.go)
   - Updates ring buffers and caches Django stats for view.

6) View rendering (internal/ui/view.go)
   - The UI composes status bar, cards, charts, and events pane.

## Config resolution

- Defaults load first, then the config file (explicit `--config`, `REPROQ_TUI_CONFIG`,
  or the default config path), then env vars, then flags.
- `reproq-tui setup` writes a starter config to the default path.

## Auth flow (optional)

- The UI initiates pairing with `GET /reproq/tui/pair/` on reproq-django.
- If a Django URL is known, the UI first fetches `/reproq/tui/config/` to
  auto-discover worker metrics/health/events endpoints.
- The user signs in on a dedicated login page and approves the session.
- Django returns a signed token that is stored locally and applied as an `Authorization` header.
- When auto-login is enabled, the UI can trigger pairing after an auth failure.

## Concurrency model

- Bubble Tea Update never blocks on network calls.
- All HTTP requests are executed in tea.Cmd functions (goroutines). 
- SSE events are handled in a goroutine with reconnect/backoff and a buffered
  channel into the model.
- Context cancellation is used for pollers and SSE on shutdown.

## Error handling

- Metrics or health errors set a degraded status and do not crash the UI.
- Scrape errors are surfaced in the status bar and retried on the next poll.
- SSE disconnections trigger reconnects with jittered backoff.

## Portability considerations

- No platform-specific syscalls or file paths.
- Uses standard net/http and context for networking.
- Terminal rendering is done with Bubble Tea and Lip Gloss.

## Theming and terminal capability detection

- Theme mode: auto/dark/light.
- auto mode uses COLORFGBG when available to pick dark vs light.
- Palette colors are selected based on termenv profile:
  - TrueColor uses hex colors.
  - ANSI256 uses 256-color palette entries.
  - ANSI uses 16-color fallbacks.

## Performance notes

- Ring buffers keep memory bounded for long-running sessions.
- View rendering avoids heavy allocations in hot paths.
- Poll intervals and time windows are configurable for tuning.
