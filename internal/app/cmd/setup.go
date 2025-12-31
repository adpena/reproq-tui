package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type setupConfig struct {
	WorkerURL        string `yaml:"worker_url,omitempty"`
	WorkerMetricsURL string `yaml:"worker_metrics_url,omitempty"`
	WorkerHealthURL  string `yaml:"worker_health_url,omitempty"`
	EventsURL        string `yaml:"events_url,omitempty"`
	DjangoURL        string `yaml:"django_url,omitempty"`
	DjangoStatsURL   string `yaml:"django_stats_url,omitempty"`
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Write a config file and optionally sign in",
	RunE: func(cmd *cobra.Command, _ []string) error {
		workerURL, _ := cmd.Flags().GetString("worker-url")
		metricsURL, _ := cmd.Flags().GetString("worker-metrics-url")
		healthURL, _ := cmd.Flags().GetString("worker-health-url")
		djangoURL, _ := cmd.Flags().GetString("django-url")
		djangoStatsURL, _ := cmd.Flags().GetString("django-stats-url")
		eventsURL, _ := cmd.Flags().GetString("events-url")
		configPath, _ := cmd.Flags().GetString("config")
		loginEnabled, _ := cmd.Flags().GetBool("login")
		openBrowser, _ := cmd.Flags().GetBool("open-browser")
		authFile, _ := cmd.Flags().GetString("auth-file")
		force, _ := cmd.Flags().GetBool("force")

		explicitWorkerURL := workerURL != ""
		if !explicitWorkerURL {
			workerURL = deriveWorkerURL(metricsURL)
		}
		if workerURL == "" && metricsURL == "" {
			return errors.New("worker-url or worker-metrics-url is required")
		}
		metricsURLTrim := strings.TrimSuffix(strings.TrimSpace(metricsURL), "/")
		deriveEvents := eventsURL == "" && workerURL != "" && (explicitWorkerURL || metricsURLTrim == "" || strings.HasSuffix(metricsURLTrim, "/metrics"))
		if deriveEvents {
			eventsURL = joinPath(workerURL, "/events")
		}
		if configPath == "" {
			path, err := defaultConfigPath()
			if err != nil {
				return err
			}
			configPath = path
		}
		if !force {
			if _, err := os.Stat(configPath); err == nil {
				return fmt.Errorf("config file already exists: %s (use --force to overwrite)", configPath)
			}
		}

		cfg := setupConfig{
			WorkerMetricsURL: metricsURL,
			WorkerHealthURL:  healthURL,
			EventsURL:        eventsURL,
			DjangoURL:        djangoURL,
			DjangoStatsURL:   djangoStatsURL,
		}
		if explicitWorkerURL {
			cfg.WorkerURL = workerURL
		}
		payload, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(configPath, payload, 0o600); err != nil {
			return err
		}

		fmt.Printf("Wrote %s\n", configPath)
		fmt.Printf("Run: reproq-tui dashboard --config %s\n", configPath)

		if loginEnabled && djangoURL != "" {
			opts := loginOptions{
				DjangoURL:   djangoURL,
				Timeout:     2 * time.Second,
				Poll:        time.Second,
				MaxWait:     10 * time.Minute,
				OpenBrowser: openBrowser,
				AuthFile:    authFile,
			}
			store := authStore(authFile)
			if err := runLoginFlow(opts, store); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	setupCmd.Flags().String("config", "", "Path to write config file (default: user config dir)")
	setupCmd.Flags().String("worker-url", "", "Base worker URL (required if worker-metrics-url is not set)")
	setupCmd.Flags().String("worker-metrics-url", "", "Worker metrics URL (used to derive worker-url)")
	setupCmd.Flags().String("worker-health-url", "", "Worker health URL (optional)")
	setupCmd.Flags().String("events-url", "", "Events SSE URL (default derived from worker-url)")
	setupCmd.Flags().String("django-url", "", "Base Django URL (optional)")
	setupCmd.Flags().String("django-stats-url", "", "Django stats URL (optional)")
	setupCmd.Flags().Bool("login", true, "Run login flow after writing config")
	setupCmd.Flags().Bool("open-browser", true, "Open the approval URL in a browser")
	setupCmd.Flags().String("auth-file", "", "Override auth token store path")
	setupCmd.Flags().Bool("force", false, "Overwrite the config file if it exists")
	RootCmd.AddCommand(setupCmd)
}

func defaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return "", fmt.Errorf("unable to resolve user config dir")
	}
	return filepath.Join(dir, "reproq-tui", "config.yaml"), nil
}

func deriveWorkerURL(metricsURL string) string {
	if strings.TrimSpace(metricsURL) == "" {
		return ""
	}
	metricsURL = strings.TrimSuffix(metricsURL, "/")
	if strings.HasSuffix(metricsURL, "/metrics") {
		return strings.TrimSuffix(metricsURL, "/metrics")
	}
	return metricsURL
}

func joinPath(base, suffix string) string {
	base = strings.TrimSuffix(base, "/")
	if base == "" {
		return suffix
	}
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	return base + suffix
}
