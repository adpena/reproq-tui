# Roadmap

This roadmap tracks the planned evolution of reproq-tui. Each milestone lists
scope, acceptance criteria, risks, and testing expectations.

## v0.1 - Vertical Slice Dashboard

Goals:
- Deliver a usable, modern TUI with polling, charts, and status.
- Provide a demo mode that showcases metrics, health, and optional events.
- Establish config and documentation foundations.

Non-goals:
- No database access or embedded web UI.
- No deep drilldowns beyond a basic details overlay.
- No plugin system or external storage.

Tasks:
- [ ] CLI: `reproq-tui dashboard` with flags, env, config file.
- [ ] Poll /metrics and /healthz with timeouts and retries.
- [ ] Parse Prometheus text/OpenMetrics and build snapshots.
- [ ] Ring buffers for rolling windows (1m/5m/15m).
- [ ] Render status bar, Now cards, and 2+ charts.
- [ ] Demo server with realistic metrics and health variation.
- [ ] Snapshot export to JSON.
- [ ] Documentation for metrics, events, and dev workflow.

Definition of Done (Acceptance Tests):
- `reproq-tui demo` launches and shows live updates within 2 seconds.
- `reproq-tui dashboard --worker-metrics-url ...` renders without panics.
- Metrics parsing tolerates missing metrics (renders "-").
- `go test ./...` passes on macOS/Linux/Windows.

Risks and Mitigation:
- Risk: Metric names differ across worker versions.
  Mitigation: Catalog mapping via config + docs.
- Risk: Slow endpoints degrade UI responsiveness.
  Mitigation: timeouts and background polling with degraded status.
- Risk: Terminal capability mismatches.
  Mitigation: palette fallback for 256/16 colors.

Testing Requirements:
- Unit tests for metrics parsing, catalog mapping, ring buffers, charts.
- Config precedence tests (flags > env > file > defaults).
- Integration test using demo server and headless update loop.

## v0.2 - Drilldowns, Filters, Tables

Goals:
- Add drilldown views for queues, workers, tasks, and errors.
- Add filters and table components for browsing data.
- Improve layout navigation and focus management.

Non-goals:
- No editing or task controls from the TUI.
- No complex alerting or scripting.

Tasks:
- [ ] Table view models for queues/workers/tasks.
- [ ] Filter syntax and search input UX.
- [ ] Scrollable views and sticky headers.
- [ ] Persist filters in snapshot export.

Definition of Done (Acceptance Tests):
- Drilldown views are reachable from the dashboard.
- Filters apply consistently across events and tables.
- Layout remains stable on terminal resize.

Risks and Mitigation:
- Risk: Large datasets cause slow rendering.
  Mitigation: pagination and diff-based rendering where possible.
- Risk: UI complexity grows.
  Mitigation: keep clean boundaries between data and view layers.

Testing Requirements:
- Unit tests for filter parsing and table formatting.
- Snapshot tests for view rendering in different sizes.

## v0.3 - SSE Events + Alerts

Goals:
- Robust SSE client with reconnect/backoff.
- Event filtering and severity highlighting.
- Optional alert rules surfaced in the UI.

Non-goals:
- No persistent alert history.
- No integrations (Slack, PagerDuty, etc.).

Tasks:
- [ ] SSE client reconnection metrics and state.
- [ ] Event buffering with filtering and rate limiting.
- [ ] Basic alert rules (error ratio, queue depth spikes).

Definition of Done (Acceptance Tests):
- Simulated SSE stream reconnects gracefully after disconnects.
- Event pane remains responsive under bursts.

Risks and Mitigation:
- Risk: SSE endpoints differ across deployments.
  Mitigation: configurable event schema in docs.
- Risk: Excessive noise in alerts.
  Mitigation: conservative defaults and user-configurable thresholds.

Testing Requirements:
- Integration tests with simulated SSE drops and reconnects.
- Unit tests for event parsing and buffer behavior.

## v1.0 - Stable Schema + Releases + Hardening

Goals:
- Stable, documented metrics schema and compatibility policy.
- Automated releases with static binaries and changelog.
- Performance and memory hardening for long-running usage.

Non-goals:
- No plugin marketplace or remote extensions.
- No embedded web UI.

Tasks:
- [ ] Versioned metrics catalog and schema docs.
- [ ] Release automation (goreleaser) with checksums.
- [ ] Performance profiling and optimization pass.
- [ ] Long-duration soak tests and memory baselines.

Definition of Done (Acceptance Tests):
- Release builds for macOS/Linux/Windows with verified checksums.
- Metrics compatibility documented with upgrade notes.
- TUI stays responsive after 24h demo run.

Risks and Mitigation:
- Risk: Backwards compatibility breaks.
  Mitigation: schema versioning and explicit change logs.
- Risk: Platform-specific rendering glitches.
  Mitigation: cross-platform CI and manual verification.

Testing Requirements:
- Release artifact verification.
- Long-running soak tests with simulated metrics.
- Lint and static analysis at PR time.
