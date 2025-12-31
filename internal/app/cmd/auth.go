package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/adpena/reproq-tui/internal/auth"
	"github.com/adpena/reproq-tui/pkg/client"
	"github.com/spf13/cobra"
)

type loginOptions struct {
	DjangoURL   string
	Timeout     time.Duration
	Poll        time.Duration
	MaxWait     time.Duration
	OpenBrowser bool
	AuthFile    string
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in via reproq-django and store a TUI token",
	RunE: func(cmd *cobra.Command, _ []string) error {
		opts, err := readLoginOptions(cmd)
		if err != nil {
			return err
		}
		store := authStore(opts.AuthFile)
		return runLoginFlow(opts, store)
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear the stored TUI token",
	RunE: func(cmd *cobra.Command, _ []string) error {
		authFile, _ := cmd.Flags().GetString("auth-file")
		store := authStore(authFile)
		if err := store.Clear(); err != nil {
			return err
		}
		fmt.Println("Signed out.")
		return nil
	},
}

func init() {
	loginCmd.Flags().String("django-url", "", "Base Django URL (required)")
	loginCmd.Flags().Duration("timeout", 2*time.Second, "HTTP request timeout")
	loginCmd.Flags().Duration("poll", time.Second, "Polling interval for approval")
	loginCmd.Flags().Duration("max-wait", 10*time.Minute, "Max time to wait for approval")
	loginCmd.Flags().Bool("open-browser", true, "Open the approval URL in a browser")
	loginCmd.Flags().String("auth-file", "", "Override auth token store path")
	RootCmd.AddCommand(loginCmd)

	logoutCmd.Flags().String("auth-file", "", "Override auth token store path")
	RootCmd.AddCommand(logoutCmd)
}

func readLoginOptions(cmd *cobra.Command) (loginOptions, error) {
	opts := loginOptions{}
	opts.DjangoURL, _ = cmd.Flags().GetString("django-url")
	opts.Timeout, _ = cmd.Flags().GetDuration("timeout")
	opts.Poll, _ = cmd.Flags().GetDuration("poll")
	opts.MaxWait, _ = cmd.Flags().GetDuration("max-wait")
	opts.OpenBrowser, _ = cmd.Flags().GetBool("open-browser")
	opts.AuthFile, _ = cmd.Flags().GetString("auth-file")
	if opts.DjangoURL == "" {
		return opts, errors.New("django-url is required")
	}
	if opts.Poll <= 0 {
		opts.Poll = time.Second
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 2 * time.Second
	}
	if opts.MaxWait <= 0 {
		opts.MaxWait = 10 * time.Minute
	}
	return opts, nil
}

func authStore(path string) *auth.Store {
	if path != "" {
		return auth.NewStore(path)
	}
	return auth.DefaultStore()
}

func runLoginFlow(opts loginOptions, store *auth.Store) error {
	httpClient := client.New(client.Options{Timeout: opts.Timeout})
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	pair, err := auth.StartPair(ctx, httpClient, opts.DjangoURL)
	cancel()
	if err != nil {
		return err
	}

	fmt.Printf("Open: %s\n", pair.VerifyURL)
	fmt.Printf("Code: %s\n", pair.Code)
	if opts.OpenBrowser {
		if err := openBrowser(pair.VerifyURL); err != nil {
			fmt.Printf("Browser open failed: %v\n", err)
		}
	}

	deadline := pair.ExpiresAt
	if deadline.IsZero() || time.Now().Add(opts.MaxWait).Before(deadline) {
		deadline = time.Now().Add(opts.MaxWait)
	}

	ticker := time.NewTicker(opts.Poll)
	defer ticker.Stop()

	for {
		if time.Now().After(deadline) {
			return errors.New("login timed out")
		}
		ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
		status, err := auth.CheckPair(ctx, httpClient, opts.DjangoURL, pair.Code)
		cancel()
		if err != nil {
			if client.IsStatus(err, http.StatusNotFound) {
				return errors.New("login expired")
			}
			return err
		}
		switch status.Status {
		case "approved":
			token := auth.Token{Value: status.Token, ExpiresAt: status.ExpiresAt, DjangoURL: opts.DjangoURL}
			if err := store.Save(token); err != nil {
				return err
			}
			if token.ExpiresAt.IsZero() {
				fmt.Println("Signed in.")
			} else {
				fmt.Printf("Signed in (expires %s).\n", token.ExpiresAt.Format(time.RFC3339))
			}
			return nil
		case "pending":
			<-ticker.C
		default:
			return errors.New("login expired")
		}
	}
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
