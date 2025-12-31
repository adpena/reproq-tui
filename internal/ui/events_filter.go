package ui

import (
	"net/url"
	"strings"
)

type sseFilter struct {
	queue    string
	workerID string
	taskID   string
	local    string
}

func parseSSEFilter(input string) sseFilter {
	filter := sseFilter{}
	fields := strings.Fields(input)
	local := make([]string, 0, len(fields))
	for _, field := range fields {
		key, val, ok := splitFilterToken(field)
		if !ok {
			local = append(local, field)
			continue
		}
		switch key {
		case "queue":
			filter.queue = val
		case "worker", "worker_id":
			filter.workerID = val
		case "task", "task_id":
			filter.taskID = val
		default:
			local = append(local, field)
		}
	}
	filter.local = strings.TrimSpace(strings.Join(local, " "))
	return filter
}

func splitFilterToken(token string) (string, string, bool) {
	sep := strings.IndexAny(token, ":=")
	if sep <= 0 || sep >= len(token)-1 {
		return "", "", false
	}
	key := strings.ToLower(strings.TrimSpace(token[:sep]))
	val := strings.TrimSpace(token[sep+1:])
	if key == "" || val == "" {
		return "", "", false
	}
	return key, val, true
}

func buildEventsURL(base string, filter sseFilter) string {
	if base == "" {
		return ""
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return base
	}
	query := parsed.Query()
	query.Del("queue")
	query.Del("worker_id")
	query.Del("task_id")
	if filter.queue != "" {
		query.Set("queue", filter.queue)
	}
	if filter.workerID != "" {
		query.Set("worker_id", filter.workerID)
	}
	if filter.taskID != "" {
		query.Set("task_id", filter.taskID)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
