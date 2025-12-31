package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/adpena/reproq-tui/pkg/client"
)

type TUIConfig struct {
	WorkerURL        string
	WorkerMetricsURL string
	WorkerHealthURL  string
	EventsURL        string
	LowMemoryMode    bool
}

func FetchConfig(ctx context.Context, httpClient *client.Client, baseURL string) (TUIConfig, error) {
	configURL, err := joinURL(baseURL, "/reproq/tui/config/")
	if err != nil {
		return TUIConfig{}, err
	}
	resp, err := httpClient.Get(ctx, configURL)
	if err != nil {
		return TUIConfig{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return TUIConfig{}, client.StatusError{URL: configURL, Code: resp.StatusCode}
	}
	var payload struct {
		WorkerURL        string `json:"worker_url"`
		WorkerMetricsURL string `json:"worker_metrics_url"`
		WorkerHealthURL  string `json:"worker_health_url"`
		EventsURL        string `json:"events_url"`
		LowMemoryMode    bool   `json:"low_memory_mode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return TUIConfig{}, err
	}
	return TUIConfig{
		WorkerURL:        strings.TrimSpace(payload.WorkerURL),
		WorkerMetricsURL: strings.TrimSpace(payload.WorkerMetricsURL),
		WorkerHealthURL:  strings.TrimSpace(payload.WorkerHealthURL),
		EventsURL:        strings.TrimSpace(payload.EventsURL),
		LowMemoryMode:    payload.LowMemoryMode,
	}, nil
}
