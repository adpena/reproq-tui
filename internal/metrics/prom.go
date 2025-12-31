package metrics

import (
	"context"
	"io"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/adpena/reproq-tui/pkg/client"
	"github.com/adpena/reproq-tui/pkg/models"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

func Scrape(ctx context.Context, httpClient *client.Client, url string, catalog Catalog) (models.MetricSnapshot, error) {
	start := time.Now()
	resp, err := httpClient.Get(ctx, url)
	if err != nil {
		return models.MetricSnapshot{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return models.MetricSnapshot{}, client.StatusError{URL: url, Code: resp.StatusCode}
	}
	metricFamilies, err := parseMetrics(resp.Body)
	if err != nil {
		return models.MetricSnapshot{}, err
	}
	values := extractCatalog(metricFamilies, catalog)
	return models.MetricSnapshot{
		CollectedAt: time.Now(),
		Latency:     time.Since(start),
		Values:      values,
	}, nil
}

func parseMetrics(reader io.Reader) (map[string]*dto.MetricFamily, error) {
	parser := expfmt.NewTextParser(model.NameValidationScheme)
	return parser.TextToMetricFamilies(reader)
}

func extractCatalog(families map[string]*dto.MetricFamily, catalog Catalog) map[string]float64 {
	values := map[string]float64{}
	selectors := catalog.Selectors
	if selectors == nil {
		selectors = compileSelectors(catalog.Mapping)
	}
	for key, selector := range selectors {
		values[key] = extractMetricValue(families, selector, key == MetricLatencyP95)
	}
	return values
}

func extractMetricValue(families map[string]*dto.MetricFamily, selector Selector, isP95 bool) float64 {
	if selector.Name == "" {
		return math.NaN()
	}
	family, ok := families[selector.Name]
	if !ok {
		return math.NaN()
	}
	filtered := filterMetrics(family.Metric, selector.Labels)
	switch family.GetType() {
	case dto.MetricType_GAUGE:
		return sumGauge(filtered)
	case dto.MetricType_COUNTER:
		return sumCounter(filtered)
	case dto.MetricType_SUMMARY:
		if isP95 {
			if val, ok := summaryQuantile(filtered, 0.95); ok {
				return val
			}
		}
		return sumSummary(filtered)
	case dto.MetricType_HISTOGRAM:
		if isP95 {
			if val, ok := histogramQuantile(filtered, 0.95); ok {
				return val
			}
		}
		return sumHistogram(filtered)
	default:
		return math.NaN()
	}
}

func sumGauge(metrics []*dto.Metric) float64 {
	total := 0.0
	for _, metric := range metrics {
		total += metric.GetGauge().GetValue()
	}
	return total
}

func sumCounter(metrics []*dto.Metric) float64 {
	total := 0.0
	for _, metric := range metrics {
		total += metric.GetCounter().GetValue()
	}
	return total
}

func sumSummary(metrics []*dto.Metric) float64 {
	total := 0.0
	for _, metric := range metrics {
		total += metric.GetSummary().GetSampleSum()
	}
	return total
}

func sumHistogram(metrics []*dto.Metric) float64 {
	total := 0.0
	for _, metric := range metrics {
		total += metric.GetHistogram().GetSampleSum()
	}
	return total
}

func summaryQuantile(metrics []*dto.Metric, quantile float64) (float64, bool) {
	var totalCount uint64
	total := 0.0
	found := false
	for _, metric := range metrics {
		count := metric.GetSummary().GetSampleCount()
		for _, q := range metric.GetSummary().GetQuantile() {
			if q.GetQuantile() == quantile {
				found = true
				total += float64(count) * q.GetValue()
				totalCount += count
				break
			}
		}
	}
	if !found || totalCount == 0 {
		return math.NaN(), false
	}
	return total / float64(totalCount), true
}

func histogramQuantile(metrics []*dto.Metric, quantile float64) (float64, bool) {
	if len(metrics) == 0 {
		return math.NaN(), false
	}
	bucketCounts := map[float64]uint64{}
	var totalCount uint64
	for _, metric := range metrics {
		hist := metric.GetHistogram()
		if hist == nil {
			continue
		}
		totalCount += hist.GetSampleCount()
		for _, bucket := range hist.GetBucket() {
			bucketCounts[bucket.GetUpperBound()] += bucket.GetCumulativeCount()
		}
	}
	if totalCount == 0 || len(bucketCounts) == 0 {
		return math.NaN(), false
	}
	bounds := sortedBounds(bucketCounts)
	target := float64(totalCount) * quantile
	var prevCount uint64
	for _, bound := range bounds {
		count := bucketCounts[bound]
		if float64(count) >= target {
			if count == prevCount {
				return bound, true
			}
			lowerCount := float64(prevCount)
			upperCount := float64(count)
			ratio := (target - lowerCount) / (upperCount - lowerCount)
			return bound * ratio, true
		}
		prevCount = count
	}
	return math.NaN(), false
}

func filterMetrics(metrics []*dto.Metric, labels map[string]string) []*dto.Metric {
	if len(labels) == 0 {
		return metrics
	}
	filtered := make([]*dto.Metric, 0, len(metrics))
	for _, metric := range metrics {
		if matchesLabels(metric, labels) {
			filtered = append(filtered, metric)
		}
	}
	return filtered
}

func matchesLabels(metric *dto.Metric, labels map[string]string) bool {
	if len(labels) == 0 {
		return true
	}
	for key, expected := range labels {
		found := false
		for _, pair := range metric.GetLabel() {
			if pair.GetName() == key && pair.GetValue() == expected {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func sortedBounds(counts map[float64]uint64) []float64 {
	bounds := make([]float64, 0, len(counts))
	for bound := range counts {
		bounds = append(bounds, bound)
	}
	sort.Float64s(bounds)
	return bounds
}
