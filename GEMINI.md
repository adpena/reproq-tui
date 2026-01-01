# reproq-tui

Modern, realtime terminal dashboard for Reproq Worker (and optional Reproq Django). It polls the worker metrics and health endpoints, optionally consumes SSE events, and renders a responsive Bubble Tea + Lip Gloss UI.

## Project Overview

- **Purpose**: To provide a terminal-based interface for monitoring Reproq Worker nodes and Reproq Django instances in real-time.
- **Key Technologies**: Go, Bubble Tea (TUI framework), Lip Gloss (styling), Prometheus (metrics parsing), SSE (Server-Sent Events).
- **Architecture**:
    - **Pollers**: Fetch metrics, health, and stats asynchronously.
    - **Parsers**: Process Prometheus text format into metric snapshots.
    - **Ring Buffers**: Store metric samples for rolling windows (1m/5m/15m).
    - **UI**: Renders charts, sparklines, and status bars using the Bubble Tea model.

## Building and Running

### Prerequisites
- Go 1.24.2+
- `golangci-lint` (for linting)

### Key Commands

- **Build**: `make build`
  - Compiles the binary to `bin/reproq-tui`.
- **Run**: `make run`
  - Runs the application directly from source.
- **Test**: `make test`
  - Runs all unit and integration tests.
- **Lint**: `make lint`
  - Runs `golangci-lint` to ensure code quality.
- **Format**: `make fmt`
  - Formats code using `go fmt`.
- **Install**: `go install github.com/adpena/reproq-tui/cmd/reproq-tui@latest`

### Running the Dashboard

- **Demo Mode**: `reproq-tui demo` (Runs a mock server + UI)
- **Standard Run**: `reproq-tui dashboard --worker-metrics-url http://localhost:9100/metrics`
- **Setup**: `reproq-tui setup --worker-url http://localhost:9100 --django-url http://localhost:8000`

## Development Conventions

- **Aesthetics & UX**: Maintain a modern, high-contrast "Obsidian" aesthetic. Prioritize the use of whitespace, subtle borders, and vivid accent colors (Cyan, Emerald, Amber) to make the TUI feel "alive" and professional.
- **Observability**: Every feature must consider the telemetry it consumes. Deep observability (DB pool stats, system telemetry, task heatmaps) is a first-class citizen.
- **Style**: Follow standard Go formatting (`go fmt`).
- **Testing**:
    - Run tests before submitting changes.
    - Keep UI changes deterministic for chart tests.
    - **Golden Files**: Update UI golden snapshots with:
      `UPDATE_GOLDEN=1 go test ./internal/ui -run TestDashboardViewGolden`
- **Releases**: Tags start at `v0.0.101` and follow semantic versioning.
- **Architecture**:
    - `cmd/reproq-tui`: Entry point.
    - `internal/ui`: Bubble Tea models and views.
    - `internal/metrics`: Prometheus parsing and storage.
    - `internal/events`: SSE client.
    - `internal/charts`: TUI chart renderers.

## UI Layout Notes

- **Queues detail**: paused queues appear under the standard queue summary to keep operational status adjacent to throughput; labels stay muted to avoid overpowering primary metrics.
- **Databases detail**: per-database rollups live in a dedicated view to avoid cluttering the primary panels while still exposing multi-DB routing health.
- **Color usage**: we keep the Obsidian accent palette for status emphasis (paused/critical items are highlighted sparingly) so the UI remains calm during normal operations.

## Documentation

For more detailed information, refer to:
- `docs/ARCHITECTURE.md`: System design and data flow.
- `docs/DEVELOPMENT.md`: Developer guide.
- `docs/METRICS.md`: Metric definitions.
- `docs/EVENTS.md`: SSE event documentation.
