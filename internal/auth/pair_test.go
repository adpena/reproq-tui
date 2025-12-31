package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adpena/reproq-tui/pkg/client"
)

func TestPairFlow(t *testing.T) {
	pairCode := "abc123"
	tokenValue := "token-xyz"
	expiresAt := time.Now().Add(10 * time.Minute).Unix()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/reproq/tui/pair/":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code":       pairCode,
				"verify_url": server.URL + "/reproq/tui/authorize/?code=" + pairCode,
				"expires_at": expiresAt,
			})
		case "/reproq/tui/pair/" + pairCode + "/":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     "approved",
				"token":      tokenValue,
				"expires_at": expiresAt,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	httpClient := client.New(client.Options{Timeout: 2 * time.Second})
	ctx := context.Background()

	pair, err := StartPair(ctx, httpClient, server.URL)
	if err != nil {
		t.Fatalf("start pair: %v", err)
	}
	if pair.Code != pairCode {
		t.Fatalf("expected code %q, got %q", pairCode, pair.Code)
	}

	status, err := CheckPair(ctx, httpClient, server.URL, pair.Code)
	if err != nil {
		t.Fatalf("check pair: %v", err)
	}
	if status.Status != "approved" {
		t.Fatalf("expected approved, got %s", status.Status)
	}
	if status.Token != tokenValue {
		t.Fatalf("expected token %q, got %q", tokenValue, status.Token)
	}
}
