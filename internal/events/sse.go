package events

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/adpena/reproq-tui/pkg/client"
	"github.com/adpena/reproq-tui/pkg/models"
)

const (
	backoffMin = time.Second
	backoffMax = 30 * time.Second
)

type listenOptions struct {
	min     time.Duration
	max     time.Duration
	jitter  func(time.Duration) time.Duration
	sleep   func(context.Context, time.Duration) bool
	connect func(context.Context, *client.Client, string, chan<- models.Event) error
}

func Listen(ctx context.Context, httpClient *client.Client, url string, out chan<- models.Event) {
	listenWithOptions(ctx, httpClient, url, out, listenOptions{})
}

func listenWithOptions(ctx context.Context, httpClient *client.Client, url string, out chan<- models.Event, opts listenOptions) {
	min := opts.min
	max := opts.max
	if min <= 0 {
		min = backoffMin
	}
	if max <= 0 {
		max = backoffMax
	}
	jitter := opts.jitter
	if jitter == nil {
		jitter = func(base time.Duration) time.Duration {
			if base <= 0 {
				return 0
			}
			return time.Duration(rand.Int63n(int64(base/2) + 1))
		}
	}
	sleep := opts.sleep
	if sleep == nil {
		sleep = defaultSleep
	}
	connectFn := opts.connect
	if connectFn == nil {
		connectFn = connect
	}

	backoff := min
	for {
		if ctx.Err() != nil {
			return
		}
		if err := connectFn(ctx, httpClient, url, out); err != nil && ctx.Err() == nil {
			wait := backoff + jitter(backoff)
			if !sleep(ctx, wait) {
				return
			}
			backoff *= 2
			if backoff > max {
				backoff = max
			}
			continue
		}
		backoff = min
	}
}

func connect(ctx context.Context, httpClient *client.Client, url string, out chan<- models.Event) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return client.StatusError{URL: url, Code: resp.StatusCode}
	}

	reader := bufio.NewReader(resp.Body)
	var dataLines []string
	for {
		if ctx.Err() != nil {
			return nil
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			line = strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(line, "data:") {
				dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
			if len(dataLines) > 0 {
				payload := strings.Join(dataLines, "\n")
				if event, ok := parseEvent(payload); ok {
					out <- event
				}
			}
			return err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if len(dataLines) > 0 {
				payload := strings.Join(dataLines, "\n")
				if event, ok := parseEvent(payload); ok {
					out <- event
				}
			}
			dataLines = dataLines[:0]
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
}

func defaultSleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func parseEvent(payload string) (models.Event, bool) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return models.Event{}, false
	}
	event := models.Event{
		Timestamp: time.Now(),
		Level:     toString(raw["level"]),
		Type:      toString(raw["type"]),
		Message:   toString(raw["msg"]),
		Queue:     toString(raw["queue"]),
		TaskID:    toString(raw["task_id"]),
		WorkerID:  toString(raw["worker_id"]),
		Metadata:  map[string]string{},
	}
	if ts, ok := raw["ts"]; ok {
		if parsed, ok := parseTime(ts); ok {
			event.Timestamp = parsed
		}
	}
	if meta, ok := raw["metadata"].(map[string]interface{}); ok {
		for key, val := range meta {
			event.Metadata[key] = toString(val)
		}
	}
	return event, true
}

func toString(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		return ""
	}
}

func parseTime(val interface{}) (time.Time, bool) {
	switch t := val.(type) {
	case string:
		if parsed, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return parsed, true
		}
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			return parsed, true
		}
	case float64:
		sec := int64(t)
		nsec := int64((t - float64(sec)) * float64(time.Second))
		return time.Unix(sec, nsec), true
	}
	return time.Time{}, false
}
