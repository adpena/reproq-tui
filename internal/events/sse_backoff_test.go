package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adpena/reproq-tui/pkg/client"
	"github.com/adpena/reproq-tui/pkg/models"
)

func TestListenBackoffDoublesAndCaps(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var waits []time.Duration
	connect := func(ctx context.Context, _ *client.Client, _ string, _ chan<- models.Event) error {
		return errors.New("fail")
	}
	sleep := func(ctx context.Context, d time.Duration) bool {
		waits = append(waits, d)
		if len(waits) >= 4 {
			cancel()
			return false
		}
		return true
	}

	done := make(chan struct{})
	go func() {
		listenWithOptions(ctx, client.New(client.Options{}), "http://example", make(chan models.Event), listenOptions{
			min:     time.Second,
			max:     4 * time.Second,
			jitter:  func(time.Duration) time.Duration { return 0 },
			sleep:   sleep,
			connect: connect,
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listener did not exit")
	}

	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 4 * time.Second}
	if len(waits) != len(want) {
		t.Fatalf("expected %d waits, got %d", len(want), len(waits))
	}
	for i := range want {
		if waits[i] != want[i] {
			t.Fatalf("wait %d: got %s, want %s", i, waits[i], want[i])
		}
	}
}

func TestListenBackoffResetsAfterSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var waits []time.Duration
	attempt := 0
	connect := func(ctx context.Context, _ *client.Client, _ string, _ chan<- models.Event) error {
		attempt++
		switch attempt {
		case 1:
			return errors.New("fail")
		case 2:
			return nil
		default:
			return errors.New("fail")
		}
	}
	sleep := func(ctx context.Context, d time.Duration) bool {
		waits = append(waits, d)
		if len(waits) >= 2 {
			cancel()
			return false
		}
		return true
	}

	done := make(chan struct{})
	go func() {
		listenWithOptions(ctx, client.New(client.Options{}), "http://example", make(chan models.Event), listenOptions{
			min:     time.Second,
			max:     5 * time.Second,
			jitter:  func(time.Duration) time.Duration { return 0 },
			sleep:   sleep,
			connect: connect,
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listener did not exit")
	}

	want := []time.Duration{time.Second, time.Second}
	if len(waits) != len(want) {
		t.Fatalf("expected %d waits, got %d", len(want), len(waits))
	}
	for i := range want {
		if waits[i] != want[i] {
			t.Fatalf("wait %d: got %s, want %s", i, waits[i], want[i])
		}
	}
}

func TestListenBackoffJitterApplied(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var waits []time.Duration
	connect := func(ctx context.Context, _ *client.Client, _ string, _ chan<- models.Event) error {
		return errors.New("fail")
	}
	sleep := func(ctx context.Context, d time.Duration) bool {
		waits = append(waits, d)
		cancel()
		return false
	}

	done := make(chan struct{})
	go func() {
		listenWithOptions(ctx, client.New(client.Options{}), "http://example", make(chan models.Event), listenOptions{
			min:     2 * time.Second,
			max:     10 * time.Second,
			jitter:  func(base time.Duration) time.Duration { return base / 2 },
			sleep:   sleep,
			connect: connect,
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listener did not exit")
	}

	if len(waits) != 1 {
		t.Fatalf("expected 1 wait, got %d", len(waits))
	}
	if waits[0] != 3*time.Second {
		t.Fatalf("expected jittered wait 3s, got %s", waits[0])
	}
}
