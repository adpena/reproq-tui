# Reproq TUI Agent Guidelines

As an agent working on `reproq-tui`, you must adhere to the following standards:

## 1. Aesthetic Excellence
- The UI must feel modern, minimal, and performant, similar to high-end CLI tools like `gemini-cli`.
- Use the "Obsidian" theme palette defined in `internal/theme/theme.go`.
- Use whitespace intentionally to separate concerns without heavy box borders.
- Utilize sparklines, gauges, and high-resolution charts for trend visualization.

## 2. Metrics & Observability
- Surface actionable metrics: latency, throughput, error rates, and resource utilization.
- Ensure all metrics are properly parsed and handled (including NaN/Inf cases).
- Support for "Pulse" metrics in a hero row for at-a-glance monitoring.

## 3. Engineering Rigor
- **Test-Driven Development**: Write tests for new logic before implementation.
- **Golden Files**: Always update and verify UI golden snapshots after view changes using `UPDATE_GOLDEN=1 go test ./internal/ui`.
- **Linting**: Ensure `golangci-lint` passes.
- **Performance**: Profile for CPU/Memory usage during high event volumes to ensure the TUI remains responsive.

## 4. Documentation
- Keep `GEMINI.md` and `docs/` up to date with architectural shifts.
- Explain the *why* behind UI layout decisions and color choices.