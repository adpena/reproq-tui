package auth

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreSaveLoadClear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	store := NewStore(path)

	token := Token{Value: "token-123", ExpiresAt: time.Now().Add(2 * time.Hour)}
	if err := store.Save(token); err != nil {
		t.Fatalf("save token: %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("load token: %v", err)
	}
	if loaded.Value != token.Value {
		t.Fatalf("expected token %q, got %q", token.Value, loaded.Value)
	}
	if loaded.ExpiresAt.IsZero() || loaded.ExpiresAt.Unix() != token.ExpiresAt.Unix() {
		t.Fatalf("expected expires_at %v, got %v", token.ExpiresAt, loaded.ExpiresAt)
	}
	if err := store.Clear(); err != nil {
		t.Fatalf("clear token: %v", err)
	}
	if _, err := store.Load(); err == nil {
		t.Fatalf("expected load error after clear")
	}
}
