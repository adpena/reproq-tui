package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

var ErrNotFound = errors.New("auth token not found")

type Token struct {
	Value     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	DjangoURL string    `json:"django_url,omitempty"`
}

func (t Token) Valid(now time.Time) bool {
	if t.Value == "" {
		return false
	}
	if t.ExpiresAt.IsZero() {
		return true
	}
	return now.Before(t.ExpiresAt)
}

type Store struct {
	path string
}

func DefaultStore() *Store {
	if val := os.Getenv("REPROQ_TUI_AUTH_FILE"); val != "" {
		return &Store{path: val}
	}
	root := ""
	if dir, err := os.UserConfigDir(); err == nil {
		root = dir
	}
	if root == "" {
		root = os.TempDir()
	}
	return &Store{path: filepath.Join(root, "reproq-tui", "auth.json")}
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Load() (Token, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Token{}, ErrNotFound
		}
		return Token{}, err
	}
	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return Token{}, err
	}
	return token, nil
}

func (s *Store) Save(token Token) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func (s *Store) Clear() error {
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
