package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/adpena/reproq-tui/pkg/client"
)

type Pairing struct {
	Code             string
	VerifyURL        string
	ExpiresAt        time.Time
	WorkerURL        string
	WorkerMetricsURL string
	WorkerHealthURL  string
	EventsURL        string
	LowMemoryMode    bool
}

type PairStatus struct {
	Status    string
	Token     string
	ExpiresAt time.Time
}

func StartPair(ctx context.Context, httpClient *client.Client, baseURL string) (Pairing, error) {
	pairURL, err := joinURL(baseURL, "/reproq/tui/pair/")
	if err != nil {
		return Pairing{}, err
	}
	resp, err := httpClient.Get(ctx, pairURL)
	if err != nil {
		return Pairing{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Pairing{}, client.StatusError{URL: pairURL, Code: resp.StatusCode}
	}
	var payload struct {
		Code             string `json:"code"`
		VerifyURL        string `json:"verify_url"`
		ExpiresAt        int64  `json:"expires_at"`
		WorkerURL        string `json:"worker_url"`
		WorkerMetricsURL string `json:"worker_metrics_url"`
		WorkerHealthURL  string `json:"worker_health_url"`
		EventsURL        string `json:"events_url"`
		LowMemoryMode    bool   `json:"low_memory_mode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Pairing{}, err
	}
	return Pairing{
		Code:             payload.Code,
		VerifyURL:        payload.VerifyURL,
		ExpiresAt:        time.Unix(payload.ExpiresAt, 0),
		WorkerURL:        strings.TrimSpace(payload.WorkerURL),
		WorkerMetricsURL: strings.TrimSpace(payload.WorkerMetricsURL),
		WorkerHealthURL:  strings.TrimSpace(payload.WorkerHealthURL),
		EventsURL:        strings.TrimSpace(payload.EventsURL),
		LowMemoryMode:    payload.LowMemoryMode,
	}, nil
}

func CheckPair(ctx context.Context, httpClient *client.Client, baseURL, code string) (PairStatus, error) {
	statusURL, err := joinURL(baseURL, fmt.Sprintf("/reproq/tui/pair/%s/", code))
	if err != nil {
		return PairStatus{}, err
	}
	resp, err := httpClient.Get(ctx, statusURL)
	if err != nil {
		return PairStatus{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return PairStatus{}, client.StatusError{URL: statusURL, Code: resp.StatusCode}
	}
	var payload struct {
		Status    string `json:"status"`
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return PairStatus{}, err
	}
	status := PairStatus{Status: payload.Status, Token: payload.Token}
	if payload.ExpiresAt > 0 {
		status.ExpiresAt = time.Unix(payload.ExpiresAt, 0)
	}
	return status, nil
}

func joinURL(baseURL, suffix string) (string, error) {
	if strings.TrimSpace(baseURL) == "" {
		return "", fmt.Errorf("base url is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	basePath := strings.TrimSuffix(parsed.Path, "/")
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	if strings.HasSuffix(basePath, "/reproq") && strings.HasPrefix(suffix, "/reproq/") {
		suffix = strings.TrimPrefix(suffix, "/reproq")
	}
	parsed.Path = basePath + suffix
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}
