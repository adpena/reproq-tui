package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/adpena/reproq-tui/pkg/client"
	"github.com/adpena/reproq-tui/pkg/models"
)

func Fetch(ctx context.Context, httpClient *client.Client, url string) (models.DjangoStats, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return models.DjangoStats{}, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return models.DjangoStats{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return models.DjangoStats{}, fmt.Errorf("stats status %d", resp.StatusCode)
	}
	var stats models.DjangoStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return models.DjangoStats{}, err
	}
	stats.FetchedAt = time.Now()
	if stats.Tasks == nil {
		stats.Tasks = map[string]int64{}
	}
	if stats.Queues == nil {
		stats.Queues = map[string]map[string]int64{}
	}
	if stats.QueueControls == nil {
		stats.QueueControls = []models.QueueControl{}
	}
	if stats.Databases == nil {
		stats.Databases = []models.DatabaseStats{}
	}
	return stats, nil
}
