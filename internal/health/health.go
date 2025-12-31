package health

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/adpena/reproq-tui/pkg/client"
	"github.com/adpena/reproq-tui/pkg/models"
)

type payload struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Build   string `json:"build"`
	Commit  string `json:"commit"`
	Message string `json:"message"`
}

func Fetch(ctx context.Context, httpClient *client.Client, url string) (models.HealthStatus, error) {
	start := time.Now()
	resp, err := httpClient.Get(ctx, url)
	if err != nil {
		return models.HealthStatus{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	status := models.HealthStatus{
		Healthy:   resp.StatusCode == http.StatusOK,
		Status:    strings.ToLower(http.StatusText(resp.StatusCode)),
		CheckedAt: time.Now(),
		Latency:   time.Since(start),
	}
	if len(body) > 0 && (strings.Contains(resp.Header.Get("Content-Type"), "application/json") || body[0] == '{') {
		var parsed payload
		if err := json.Unmarshal(body, &parsed); err == nil {
			status.Status = strings.ToLower(parsed.Status)
			status.Version = parsed.Version
			status.Build = parsed.Build
			status.Commit = parsed.Commit
			status.Message = parsed.Message
			if status.Status == "ok" || status.Status == "healthy" {
				status.Healthy = true
			}
		}
	}
	if resp.StatusCode >= 400 {
		return status, client.StatusError{URL: url, Code: resp.StatusCode}
	}
	return status, nil
}
