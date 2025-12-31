# Events (SSE)

reproq-tui can consume an optional Server-Sent Events stream for recent task
activity and errors. If events are not configured, the UI shows a placeholder.

## Endpoint

- URL: configurable via `--events-url`
- Content-Type: text/event-stream
- Auth: use `--auth-token` to send `Authorization: Bearer <token>`, or `--header` for custom headers. TUI login uses a signed bearer token stored locally.
- Optional filters: `?queue=<name>&worker_id=<id>&task_id=<id>` (supported by reproq-worker).

## SSE format

Each event should be sent as JSON in a `data:` line, for example:

```
data: {"ts":"2024-01-01T12:00:00Z","level":"error","type":"task_failed","msg":"task failed","queue":"default","task_id":"abc","worker_id":"w1"}
```

Multiple `data:` lines are concatenated with newlines until a blank line is
received.

## Event schema

Field | Type | Notes
----- | ---- | -----
ts | string or float | RFC3339 or unix seconds
level | string | info, warn, error
type | string | event type string
msg | string | human readable message
queue | string | optional
task_id | string | optional
worker_id | string | optional
metadata | object | optional key/value map

## UI behavior

- Events are buffered in a ring buffer and filtered by the search input.
- The events pane highlights warn/error levels.
- Reconnects use exponential backoff with jitter.
- The `/` filter supports `queue:`, `worker:`, and `task:` tokens that are sent to the SSE server.
