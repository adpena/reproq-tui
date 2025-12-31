package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func setTestConfigHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("APPDATA", dir)
	t.Setenv(envPrefix+"CONFIG", "")
	return dir
}

func TestLoadPrecedence(t *testing.T) {
	setTestConfigHome(t)
	cmd := &cobra.Command{Use: "test"}
	RegisterFlags(cmd)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("worker_metrics_url: http://config\ninterval: 5s\nwindow: 10m\nauth_token: filetoken\ndjango_stats_url: http://stats-file\nstats_interval: 9s\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv(envPrefix+"WORKER_METRICS_URL", "http://env")
	t.Setenv(envPrefix+"WINDOW", "2m")
	t.Setenv(envPrefix+"AUTH_TOKEN", "envtoken")
	t.Setenv(envPrefix+"DJANGO_STATS_URL", "http://stats-env")
	t.Setenv(envPrefix+"STATS_INTERVAL", "7s")

	if err := cmd.Flags().Set("config", cfgPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}
	if err := cmd.Flags().Set("worker-metrics-url", "http://flag"); err != nil {
		t.Fatalf("set metrics flag: %v", err)
	}
	if err := cmd.Flags().Set("interval", "1s"); err != nil {
		t.Fatalf("set interval flag: %v", err)
	}
	if err := cmd.Flags().Set("auth-token", "flagtoken"); err != nil {
		t.Fatalf("set auth token flag: %v", err)
	}
	if err := cmd.Flags().Set("django-stats-url", "http://stats-flag"); err != nil {
		t.Fatalf("set stats url flag: %v", err)
	}
	if err := cmd.Flags().Set("stats-interval", "3s"); err != nil {
		t.Fatalf("set stats interval flag: %v", err)
	}

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.WorkerMetricsURL != "http://flag" {
		t.Fatalf("expected flag override, got %s", cfg.WorkerMetricsURL)
	}
	if cfg.Interval != time.Second {
		t.Fatalf("expected interval 1s, got %s", cfg.Interval)
	}
	if cfg.Window != 2*time.Minute {
		t.Fatalf("expected window from env, got %s", cfg.Window)
	}
	if got := cfg.Headers["Authorization"]; got != "Bearer flagtoken" {
		t.Fatalf("expected auth header from flag, got %q", got)
	}
	if cfg.DjangoStatsURL != "http://stats-flag" {
		t.Fatalf("expected stats url from flag, got %s", cfg.DjangoStatsURL)
	}
	if cfg.StatsInterval != 3*time.Second {
		t.Fatalf("expected stats interval from flag, got %s", cfg.StatsInterval)
	}
}

func TestLoadDefaults(t *testing.T) {
	setTestConfigHome(t)
	cmd := &cobra.Command{Use: "test"}
	RegisterFlags(cmd)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("worker_metrics_url: http://config\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := cmd.Flags().Set("config", cfgPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Interval != time.Second {
		t.Fatalf("expected default interval, got %s", cfg.Interval)
	}
	if cfg.HealthInterval != 500*time.Millisecond {
		t.Fatalf("expected default health interval, got %s", cfg.HealthInterval)
	}
	if cfg.Timeout != 2*time.Second {
		t.Fatalf("expected default timeout, got %s", cfg.Timeout)
	}
	if cfg.StatsInterval != 5*time.Second {
		t.Fatalf("expected default stats interval, got %s", cfg.StatsInterval)
	}
	if !cfg.AutoLogin {
		t.Fatalf("expected default auto_login to be true")
	}
	if cfg.WorkerHealthURL == "" || !strings.HasSuffix(cfg.WorkerHealthURL, "/healthz") {
		t.Fatalf("expected derived health url ending with /healthz, got %s", cfg.WorkerHealthURL)
	}
}

func TestLoadWorkerURLDerivesEndpoints(t *testing.T) {
	setTestConfigHome(t)
	cmd := &cobra.Command{Use: "test"}
	RegisterFlags(cmd)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("worker_url: http://worker.local/api\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := cmd.Flags().Set("config", cfgPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.WorkerMetricsURL != "http://worker.local/api/metrics" {
		t.Fatalf("expected metrics url derived from worker url, got %s", cfg.WorkerMetricsURL)
	}
	if cfg.WorkerHealthURL != "http://worker.local/api/healthz" {
		t.Fatalf("expected health url derived from worker url, got %s", cfg.WorkerHealthURL)
	}
}

func TestWorkerURLDoesNotOverrideExplicitMetrics(t *testing.T) {
	setTestConfigHome(t)
	cmd := &cobra.Command{Use: "test"}
	RegisterFlags(cmd)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("worker_url: http://worker.local\nworker_metrics_url: http://metrics.local/metrics\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := cmd.Flags().Set("config", cfgPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.WorkerMetricsURL != "http://metrics.local/metrics" {
		t.Fatalf("expected explicit metrics url, got %s", cfg.WorkerMetricsURL)
	}
}

func TestLoadDjangoURLDerivesStats(t *testing.T) {
	setTestConfigHome(t)
	cmd := &cobra.Command{Use: "test"}
	RegisterFlags(cmd)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("worker_metrics_url: http://config\ndjango_url: http://django.local\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := cmd.Flags().Set("config", cfgPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DjangoStatsURL != "http://django.local/reproq/stats/" {
		t.Fatalf("expected derived stats url, got %s", cfg.DjangoStatsURL)
	}
}

func TestLoadDjangoStatsDerivesURL(t *testing.T) {
	setTestConfigHome(t)
	cmd := &cobra.Command{Use: "test"}
	RegisterFlags(cmd)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("worker_metrics_url: http://config\ndjango_stats_url: http://django.local/reproq/stats/\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := cmd.Flags().Set("config", cfgPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DjangoURL != "http://django.local" {
		t.Fatalf("expected derived django url, got %s", cfg.DjangoURL)
	}
}

func TestAutoLoginFromFileAndFlag(t *testing.T) {
	setTestConfigHome(t)
	cmd := &cobra.Command{Use: "test"}
	RegisterFlags(cmd)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("worker_metrics_url: http://config\nauto_login: false\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := cmd.Flags().Set("config", cfgPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AutoLogin {
		t.Fatalf("expected auto_login false from config")
	}

	if err := cmd.Flags().Set("auto-login", "true"); err != nil {
		t.Fatalf("set auto-login flag: %v", err)
	}
	cfg, err = Load(cmd)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.AutoLogin {
		t.Fatalf("expected auto_login true from flag")
	}
}

func TestLoadAuthTokenEnvFallback(t *testing.T) {
	setTestConfigHome(t)
	cmd := &cobra.Command{Use: "test"}
	RegisterFlags(cmd)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("worker_metrics_url: http://config\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := cmd.Flags().Set("config", cfgPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}

	t.Setenv("METRICS_AUTH_TOKEN", "fallbacktoken")

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Headers["Authorization"]; got != "Bearer fallbacktoken" {
		t.Fatalf("expected auth header from METRICS_AUTH_TOKEN, got %q", got)
	}
}

func TestAuthTokenDoesNotOverrideHeader(t *testing.T) {
	setTestConfigHome(t)
	cmd := &cobra.Command{Use: "test"}
	RegisterFlags(cmd)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("worker_metrics_url: http://config\nheaders:\n  - \"Authorization: Bearer header\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := cmd.Flags().Set("config", cfgPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}
	if err := cmd.Flags().Set("auth-token", "flagtoken"); err != nil {
		t.Fatalf("set auth token flag: %v", err)
	}

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Headers["Authorization"]; got != "Bearer header" {
		t.Fatalf("expected header to remain, got %q", got)
	}
}

func TestLoadDefaultConfigPath(t *testing.T) {
	setTestConfigHome(t)
	cmd := &cobra.Command{Use: "test"}
	RegisterFlags(cmd)

	cfgPath, err := defaultConfigPath()
	if err != nil {
		t.Fatalf("default config path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte("worker_metrics_url: http://config\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.WorkerMetricsURL != "http://config" {
		t.Fatalf("expected config from default path, got %s", cfg.WorkerMetricsURL)
	}
}
