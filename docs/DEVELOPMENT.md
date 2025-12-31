# Development

## Prerequisites

 - Go 1.24.0+
- golangci-lint (for `make lint`)

## Common commands

```
make fmt
make test
make lint
make build
make run
```

## Demo mode

```
reproq-tui demo
```

## Local dashboard

```
reproq-tui dashboard --worker-metrics-url http://localhost:9100/metrics
```

## Quick setup (optional)

```
reproq-tui setup --worker-url http://localhost:9100 --django-url http://localhost:8000
reproq-tui dashboard
```

## Notes

- Run tests before submitting changes: `go test ./...`
- Keep UI changes deterministic so chart tests remain stable.
- Update UI golden snapshots with `UPDATE_GOLDEN=1 go test ./internal/ui -run TestDashboardViewGolden`.

## Releases

- Tags start at `v0.0.101` and increment patch (`v0.0.102`, `v0.0.103`, ...).
- Create a tag and push:
  `git tag v0.0.101 && git push origin v0.0.101`
- The release workflow runs GoReleaser, updates GitHub releases, and publishes Homebrew formula updates.
- For Homebrew publishing, set a `HOMEBREW_TAP_GITHUB_TOKEN` secret with access to `adpena/homebrew-tap`.
