package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	envPrefix = "REPROQ_TUI_"
)

type Config struct {
	WorkerURL          string
	WorkerMetricsURL   string
	WorkerHealthURL    string
	EventsURL          string
	DjangoURL          string
	DjangoStatsURL     string
	Interval           time.Duration
	HealthInterval     time.Duration
	StatsInterval      time.Duration
	Window             time.Duration
	Theme              string
	AutoLogin          bool
	Headers            map[string]string
	AuthToken          string
	Timeout            time.Duration
	InsecureSkipVerify bool
	Metrics            map[string]string
	LogFile            string
}

type fileConfig struct {
	WorkerURL          string            `yaml:"worker_url" toml:"worker_url"`
	WorkerMetricsURL   string            `yaml:"worker_metrics_url" toml:"worker_metrics_url"`
	WorkerHealthURL    string            `yaml:"worker_health_url" toml:"worker_health_url"`
	EventsURL          string            `yaml:"events_url" toml:"events_url"`
	DjangoURL          string            `yaml:"django_url" toml:"django_url"`
	DjangoStatsURL     string            `yaml:"django_stats_url" toml:"django_stats_url"`
	Interval           string            `yaml:"interval" toml:"interval"`
	HealthInterval     string            `yaml:"health_interval" toml:"health_interval"`
	StatsInterval      string            `yaml:"stats_interval" toml:"stats_interval"`
	Window             string            `yaml:"window" toml:"window"`
	Theme              string            `yaml:"theme" toml:"theme"`
	AutoLogin          *bool             `yaml:"auto_login" toml:"auto_login"`
	Headers            []string          `yaml:"headers" toml:"headers"`
	AuthToken          string            `yaml:"auth_token" toml:"auth_token"`
	Timeout            string            `yaml:"timeout" toml:"timeout"`
	InsecureSkipVerify bool              `yaml:"insecure_skip_verify" toml:"insecure_skip_verify"`
	Metrics            map[string]string `yaml:"metrics" toml:"metrics"`
	LogFile            string            `yaml:"log_file" toml:"log_file"`
}

type flagValues struct {
	ConfigFile         string
	WorkerURL          string
	WorkerMetricsURL   string
	WorkerHealthURL    string
	EventsURL          string
	DjangoURL          string
	DjangoStatsURL     string
	Interval           time.Duration
	HealthInterval     time.Duration
	StatsInterval      time.Duration
	Window             time.Duration
	Theme              string
	AutoLogin          bool
	Headers            []string
	AuthToken          string
	Timeout            time.Duration
	InsecureSkipVerify bool
	Metrics            []string
	LogFile            string
	IntervalSet        bool
	HealthIntervalSet  bool
	StatsIntervalSet   bool
	WindowSet          bool
	ThemeSet           bool
	AutoLoginSet       bool
	TimeoutSet         bool
}

func DefaultConfig() Config {
	return Config{
		Interval:       time.Second,
		HealthInterval: 500 * time.Millisecond,
		StatsInterval:  5 * time.Second,
		Window:         5 * time.Minute,
		Theme:          "auto",
		AutoLogin:      true,
		Headers:        map[string]string{},
		Timeout:        2 * time.Second,
		Metrics:        map[string]string{},
	}
}

func RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().String("config", "", "Path to YAML/TOML config file")
	cmd.Flags().String("worker-url", "", "Base worker URL (derives /metrics and /healthz)")
	cmd.Flags().String("worker-metrics-url", "", "Worker Prometheus/OpenMetrics URL (required)")
	cmd.Flags().String("worker-health-url", "", "Worker health URL (default derived from metrics host)")
	cmd.Flags().String("events-url", "", "Events SSE URL")
	cmd.Flags().String("django-url", "", "Base Django URL (derives /reproq/stats/ and auth endpoints)")
	cmd.Flags().String("django-stats-url", "", "Django stats API URL (optional)")
	cmd.Flags().Duration("interval", time.Second, "Metrics poll interval")
	cmd.Flags().Duration("health-interval", 500*time.Millisecond, "Health poll interval")
	cmd.Flags().Duration("stats-interval", 5*time.Second, "Django stats poll interval")
	cmd.Flags().Duration("window", 5*time.Minute, "Default timeseries window")
	cmd.Flags().String("theme", "auto", "Theme: auto, dark, or light")
	cmd.Flags().Bool("auto-login", true, "Auto-start login flow when auth is required")
	cmd.Flags().StringArray("header", []string{}, "Request header in 'Key: Value' form (repeatable)")
	cmd.Flags().String("auth-token", "", "Bearer token for metrics/health/events (adds Authorization header)")
	cmd.Flags().Duration("timeout", 2*time.Second, "HTTP request timeout")
	cmd.Flags().Bool("insecure-skip-verify", false, "Skip TLS verification (dev only)")
	cmd.Flags().StringArray("metric", []string{}, "Metric mapping in 'canonical=actual' form (repeatable)")
	cmd.Flags().String("log-file", "", "Write debug logs to file")
}

func Load(cmd *cobra.Command) (Config, error) {
	flags, err := readFlags(cmd)
	if err != nil {
		return Config{}, err
	}

	cfg := DefaultConfig()
	filePath := flags.ConfigFile
	if filePath == "" {
		filePath = strings.TrimSpace(os.Getenv(envPrefix + "CONFIG"))
	}
	if filePath == "" {
		if defaultPath, err := defaultConfigPath(); err == nil {
			if _, statErr := os.Stat(defaultPath); statErr == nil {
				filePath = defaultPath
			} else if !os.IsNotExist(statErr) {
				return Config{}, statErr
			}
		}
	}
	if filePath != "" {
		if err := applyFile(&cfg, filePath); err != nil {
			return Config{}, err
		}
	}
	applyEnv(&cfg)
	applyFlags(&cfg, flags)
	applyAuthToken(&cfg)
	if cfg.WorkerMetricsURL == "" {
		cfg.WorkerMetricsURL = deriveMetricsURL(cfg.WorkerURL)
	}
	if cfg.DjangoURL == "" {
		cfg.DjangoURL = deriveDjangoURL(cfg.DjangoStatsURL)
	}
	if cfg.DjangoStatsURL == "" {
		cfg.DjangoStatsURL = deriveDjangoStatsURL(cfg.DjangoURL)
	}
	if cfg.WorkerHealthURL == "" {
		cfg.WorkerHealthURL = deriveHealthURL(cfg.WorkerMetricsURL)
	}
	if cfg.WorkerMetricsURL == "" {
		return Config{}, errors.New("worker metrics URL is required (--worker-metrics-url or --worker-url)")
	}
	if err := validateURLs(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func defaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return "", fmt.Errorf("unable to resolve user config dir")
	}
	return filepath.Join(dir, "reproq-tui", "config.yaml"), nil
}

func readFlags(cmd *cobra.Command) (flagValues, error) {
	var flags flagValues
	var err error
	flags.ConfigFile, err = cmd.Flags().GetString("config")
	if err != nil {
		return flags, err
	}
	flags.WorkerURL, err = cmd.Flags().GetString("worker-url")
	if err != nil {
		return flags, err
	}
	flags.WorkerMetricsURL, err = cmd.Flags().GetString("worker-metrics-url")
	if err != nil {
		return flags, err
	}
	flags.WorkerHealthURL, err = cmd.Flags().GetString("worker-health-url")
	if err != nil {
		return flags, err
	}
	flags.EventsURL, err = cmd.Flags().GetString("events-url")
	if err != nil {
		return flags, err
	}
	flags.DjangoURL, err = cmd.Flags().GetString("django-url")
	if err != nil {
		return flags, err
	}
	flags.DjangoStatsURL, err = cmd.Flags().GetString("django-stats-url")
	if err != nil {
		return flags, err
	}
	flags.Interval, err = cmd.Flags().GetDuration("interval")
	if err != nil {
		return flags, err
	}
	flags.IntervalSet = cmd.Flags().Changed("interval")
	flags.HealthInterval, err = cmd.Flags().GetDuration("health-interval")
	if err != nil {
		return flags, err
	}
	flags.HealthIntervalSet = cmd.Flags().Changed("health-interval")
	flags.StatsInterval, err = cmd.Flags().GetDuration("stats-interval")
	if err != nil {
		return flags, err
	}
	flags.StatsIntervalSet = cmd.Flags().Changed("stats-interval")
	flags.Window, err = cmd.Flags().GetDuration("window")
	if err != nil {
		return flags, err
	}
	flags.WindowSet = cmd.Flags().Changed("window")
	flags.Theme, err = cmd.Flags().GetString("theme")
	if err != nil {
		return flags, err
	}
	flags.ThemeSet = cmd.Flags().Changed("theme")
	flags.AutoLogin, err = cmd.Flags().GetBool("auto-login")
	if err != nil {
		return flags, err
	}
	flags.AutoLoginSet = cmd.Flags().Changed("auto-login")
	flags.Headers, err = cmd.Flags().GetStringArray("header")
	if err != nil {
		return flags, err
	}
	flags.AuthToken, err = cmd.Flags().GetString("auth-token")
	if err != nil {
		return flags, err
	}
	flags.Timeout, err = cmd.Flags().GetDuration("timeout")
	if err != nil {
		return flags, err
	}
	flags.TimeoutSet = cmd.Flags().Changed("timeout")
	flags.InsecureSkipVerify, err = cmd.Flags().GetBool("insecure-skip-verify")
	if err != nil {
		return flags, err
	}
	flags.Metrics, err = cmd.Flags().GetStringArray("metric")
	if err != nil {
		return flags, err
	}
	flags.LogFile, err = cmd.Flags().GetString("log-file")
	if err != nil {
		return flags, err
	}
	return flags, nil
}

func applyFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	var fc fileConfig
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &fc); err != nil {
			return fmt.Errorf("parse yaml config: %w", err)
		}
	case ".toml":
		if err := toml.Unmarshal(data, &fc); err != nil {
			return fmt.Errorf("parse toml config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config extension: %s", filepath.Ext(path))
	}
	applyFileConfig(cfg, fc)
	return nil
}

func applyFileConfig(cfg *Config, fc fileConfig) {
	cfg.WorkerURL = firstNonEmpty(cfg.WorkerURL, fc.WorkerURL)
	cfg.WorkerMetricsURL = firstNonEmpty(cfg.WorkerMetricsURL, fc.WorkerMetricsURL)
	cfg.WorkerHealthURL = firstNonEmpty(cfg.WorkerHealthURL, fc.WorkerHealthURL)
	cfg.EventsURL = firstNonEmpty(cfg.EventsURL, fc.EventsURL)
	cfg.DjangoURL = firstNonEmpty(cfg.DjangoURL, fc.DjangoURL)
	cfg.DjangoStatsURL = firstNonEmpty(cfg.DjangoStatsURL, fc.DjangoStatsURL)
	if d := parseDuration(fc.Interval); d > 0 {
		cfg.Interval = d
	}
	if d := parseDuration(fc.HealthInterval); d > 0 {
		cfg.HealthInterval = d
	}
	if d := parseDuration(fc.StatsInterval); d > 0 {
		cfg.StatsInterval = d
	}
	if d := parseDuration(fc.Window); d > 0 {
		cfg.Window = d
	}
	cfg.Theme = firstNonEmpty(cfg.Theme, fc.Theme)
	if fc.AutoLogin != nil {
		cfg.AutoLogin = *fc.AutoLogin
	}
	if len(fc.Headers) > 0 {
		cfg.Headers = mergeHeaders(cfg.Headers, fc.Headers)
	}
	cfg.AuthToken = firstNonEmpty(cfg.AuthToken, fc.AuthToken)
	if d := parseDuration(fc.Timeout); d > 0 {
		cfg.Timeout = d
	}
	if fc.InsecureSkipVerify {
		cfg.InsecureSkipVerify = true
	}
	if len(fc.Metrics) > 0 {
		for k, v := range fc.Metrics {
			cfg.Metrics[k] = v
		}
	}
	cfg.LogFile = firstNonEmpty(cfg.LogFile, fc.LogFile)
}

func applyEnv(cfg *Config) {
	if val := strings.TrimSpace(os.Getenv(envPrefix + "WORKER_URL")); val != "" {
		cfg.WorkerURL = val
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "WORKER_METRICS_URL")); val != "" {
		cfg.WorkerMetricsURL = val
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "WORKER_HEALTH_URL")); val != "" {
		cfg.WorkerHealthURL = val
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "EVENTS_URL")); val != "" {
		cfg.EventsURL = val
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "DJANGO_URL")); val != "" {
		cfg.DjangoURL = val
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "DJANGO_STATS_URL")); val != "" {
		cfg.DjangoStatsURL = val
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "INTERVAL")); val != "" {
		if d := parseDuration(val); d > 0 {
			cfg.Interval = d
		}
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "HEALTH_INTERVAL")); val != "" {
		if d := parseDuration(val); d > 0 {
			cfg.HealthInterval = d
		}
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "STATS_INTERVAL")); val != "" {
		if d := parseDuration(val); d > 0 {
			cfg.StatsInterval = d
		}
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "WINDOW")); val != "" {
		if d := parseDuration(val); d > 0 {
			cfg.Window = d
		}
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "THEME")); val != "" {
		cfg.Theme = val
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "AUTO_LOGIN")); val != "" {
		cfg.AutoLogin = parseBool(val)
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "HEADERS")); val != "" {
		cfg.Headers = mergeHeaders(cfg.Headers, splitComma(val))
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "AUTH_TOKEN")); val != "" {
		cfg.AuthToken = val
	} else if val := strings.TrimSpace(os.Getenv("METRICS_AUTH_TOKEN")); val != "" {
		cfg.AuthToken = val
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "TIMEOUT")); val != "" {
		if d := parseDuration(val); d > 0 {
			cfg.Timeout = d
		}
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "INSECURE_SKIP_VERIFY")); val != "" {
		cfg.InsecureSkipVerify = parseBool(val)
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "METRICS")); val != "" {
		for k, v := range parseKeyValueList(splitComma(val)) {
			cfg.Metrics[k] = v
		}
	}
	if val := strings.TrimSpace(os.Getenv(envPrefix + "LOG_FILE")); val != "" {
		cfg.LogFile = val
	}
}

func applyFlags(cfg *Config, flags flagValues) {
	cfg.WorkerURL = firstNonEmpty(cfg.WorkerURL, flags.WorkerURL)
	cfg.WorkerMetricsURL = firstNonEmpty(cfg.WorkerMetricsURL, flags.WorkerMetricsURL)
	cfg.WorkerHealthURL = firstNonEmpty(cfg.WorkerHealthURL, flags.WorkerHealthURL)
	cfg.EventsURL = firstNonEmpty(cfg.EventsURL, flags.EventsURL)
	cfg.DjangoURL = firstNonEmpty(cfg.DjangoURL, flags.DjangoURL)
	cfg.DjangoStatsURL = firstNonEmpty(cfg.DjangoStatsURL, flags.DjangoStatsURL)
	if flags.IntervalSet && flags.Interval > 0 {
		cfg.Interval = flags.Interval
	}
	if flags.HealthIntervalSet && flags.HealthInterval > 0 {
		cfg.HealthInterval = flags.HealthInterval
	}
	if flags.StatsIntervalSet && flags.StatsInterval > 0 {
		cfg.StatsInterval = flags.StatsInterval
	}
	if flags.WindowSet && flags.Window > 0 {
		cfg.Window = flags.Window
	}
	if flags.ThemeSet {
		cfg.Theme = firstNonEmpty(cfg.Theme, flags.Theme)
	}
	if flags.AutoLoginSet {
		cfg.AutoLogin = flags.AutoLogin
	}
	if len(flags.Headers) > 0 {
		cfg.Headers = mergeHeaders(cfg.Headers, flags.Headers)
	}
	cfg.AuthToken = firstNonEmpty(cfg.AuthToken, flags.AuthToken)
	if flags.TimeoutSet && flags.Timeout > 0 {
		cfg.Timeout = flags.Timeout
	}
	if flags.InsecureSkipVerify {
		cfg.InsecureSkipVerify = true
	}
	if len(flags.Metrics) > 0 {
		for k, v := range parseKeyValueList(flags.Metrics) {
			cfg.Metrics[k] = v
		}
	}
	cfg.LogFile = firstNonEmpty(cfg.LogFile, flags.LogFile)
}

func deriveMetricsURL(workerURL string) string {
	if workerURL == "" {
		return ""
	}
	parsed, err := url.Parse(workerURL)
	if err != nil {
		return ""
	}
	parsed.Path = joinPath(parsed.Path, "/metrics")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func deriveHealthURL(metricsURL string) string {
	if metricsURL == "" {
		return ""
	}
	parsed, err := url.Parse(metricsURL)
	if err != nil {
		return ""
	}
	basePath := strings.TrimSuffix(parsed.Path, "/")
	if strings.HasSuffix(basePath, "/metrics") {
		basePath = strings.TrimSuffix(basePath, "/metrics")
	}
	parsed.Path = joinPath(basePath, "/healthz")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func deriveDjangoStatsURL(djangoURL string) string {
	if djangoURL == "" {
		return ""
	}
	parsed, err := url.Parse(djangoURL)
	if err != nil {
		return ""
	}
	parsed.Path = joinPath(parsed.Path, "/reproq/stats/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func deriveDjangoURL(statsURL string) string {
	if statsURL == "" {
		return ""
	}
	parsed, err := url.Parse(statsURL)
	if err != nil {
		return ""
	}
	basePath := strings.TrimSuffix(parsed.Path, "/")
	if strings.HasSuffix(basePath, "/reproq/stats") {
		basePath = strings.TrimSuffix(basePath, "/reproq/stats")
	}
	parsed.Path = basePath
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func DeriveDjangoStatsURL(djangoURL string) string {
	return deriveDjangoStatsURL(djangoURL)
}

func DeriveDjangoURL(statsURL string) string {
	return deriveDjangoURL(statsURL)
}

func validateURLs(cfg Config) error {
	for name, val := range map[string]string{
		"worker url":         cfg.WorkerURL,
		"worker metrics url": cfg.WorkerMetricsURL,
		"worker health url":  cfg.WorkerHealthURL,
		"events url":         cfg.EventsURL,
		"django url":         cfg.DjangoURL,
		"django stats url":   cfg.DjangoStatsURL,
	} {
		if val == "" {
			continue
		}
		if _, err := url.ParseRequestURI(val); err != nil {
			return fmt.Errorf("invalid %s: %w", name, err)
		}
	}
	return nil
}

func joinPath(basePath, suffix string) string {
	if suffix == "" {
		return basePath
	}
	basePath = strings.TrimSuffix(basePath, "/")
	if basePath == "" {
		return suffix
	}
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	return basePath + suffix
}

func parseDuration(val string) time.Duration {
	if strings.TrimSpace(val) == "" {
		return 0
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0
	}
	return d
}

func parseBool(val string) bool {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func applyAuthToken(cfg *Config) {
	token := strings.TrimSpace(cfg.AuthToken)
	if token == "" {
		return
	}
	if hasAuthorizationHeader(cfg.Headers) {
		return
	}
	if cfg.Headers == nil {
		cfg.Headers = map[string]string{}
	}
	value := token
	if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
		value = "Bearer " + token
	}
	cfg.Headers["Authorization"] = value
}

func hasAuthorizationHeader(headers map[string]string) bool {
	for key := range headers {
		if strings.EqualFold(key, "Authorization") {
			return true
		}
	}
	return false
}

func mergeHeaders(base map[string]string, headers []string) map[string]string {
	if base == nil {
		base = map[string]string{}
	}
	for k, v := range parseHeaderList(headers) {
		base[k] = v
	}
	return base
}

func parseHeaderList(headers []string) map[string]string {
	out := map[string]string{}
	for _, header := range headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}
		out[key] = val
	}
	return out
}

func parseKeyValueList(items []string) map[string]string {
	out := map[string]string{}
	for _, item := range items {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	return out
}

func splitComma(val string) []string {
	parts := strings.Split(val, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func firstNonEmpty(current, next string) string {
	if strings.TrimSpace(next) == "" {
		return current
	}
	return next
}
