# Repository Guidelines

## Project Structure & Module Organization
- `cmd/reproq-tui/`: CLI entry point (`main.go`).
- `internal/app/`: CLI wiring and demo server (`demo`).
- `internal/ui/`: Bubble Tea model/update/view, keybindings, layout.
- `internal/metrics/`, `internal/health/`, `internal/events/`, `internal/stats/`: polling, parsing, and SSE/client logic.
- `internal/charts/` and `internal/theme/`: rendering helpers and styling.
- `pkg/client/` and `pkg/models/`: shared HTTP client and data structs.
- `docs/`: architecture, metrics, events, and development notes.
- `scripts/install.sh`: curl-based installer used in releases.

## Build, Test, and Development Commands
- `make fmt`: run `gofmt` on the repo.
- `make test`: run `go test ./...`.
- `make lint`: run `golangci-lint`.
- `make build`: build the `reproq-tui` binary.
- `make run`: run the dashboard (prompts for Django URL/worker URL if not configured).
- `reproq-tui demo`: run the mock server and dashboard UI.

## Coding Style & Naming Conventions
- Go code must be `gofmt`-formatted; follow idiomatic Go naming.
- Keep packages focused and small; avoid cyclic dependencies.
- File names are lowercase with underscores when needed (Go convention).
- Prefer ASCII in source files unless the file already uses Unicode.

## Testing Guidelines
- Tests live alongside code as `*_test.go`.
- Run `go test ./...` for the full suite.
- Keep UI output deterministic to avoid flaky chart and view tests.
- Add unit tests for new parsing/derivation logic and update integration tests when wiring changes.

## Commit & Pull Request Guidelines
- No strict commit convention is enforced; use concise, sentence-style summaries and keep commits focused.
- PRs should include a clear description, why the change is needed, and testing notes (commands run).
- Update docs when flags, endpoints, or config behavior changes.

## Security & Configuration Tips
- Use `--auth-token` or `REPROQ_TUI_AUTH_TOKEN` for protected endpoints; never commit tokens.
- Prefer secure worker metrics endpoints (auth + allowlist); use `--insecure-skip-verify` only for dev.

## Release & Distribution
- Tag releases starting at `v0.0.101` and increment patch versions (`v0.0.102`, `v0.0.103`).
- Pushing a tag triggers GoReleaser to publish GitHub releases and Homebrew formula updates.
