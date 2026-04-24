package taskloop

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type fakeHeartbeatTicker struct {
	ch      chan time.Time
	stopped atomic.Bool
}

func newFakeHeartbeatTicker() *fakeHeartbeatTicker {
	return &fakeHeartbeatTicker{ch: make(chan time.Time, 4)}
}

func (t *fakeHeartbeatTicker) Chan() <-chan time.Time {
	return t.ch
}

func (t *fakeHeartbeatTicker) Stop() {
	t.stopped.Store(true)
}

func TestHeartbeatPublishesTicksWithoutSleep(t *testing.T) {
	ticker := newFakeHeartbeatTicker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var published []time.Time
	done := startHeartbeat(ctx, func(time.Duration) heartbeatTicker {
		return ticker
	}, time.Second, func(at time.Time) error {
		published = append(published, at)
		if len(published) == 2 {
			cancel()
		}
		return nil
	})

	first := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	second := first.Add(5 * time.Second)
	ticker.ch <- first
	ticker.ch <- second

	if err := <-done; err != nil {
		t.Fatalf("startHeartbeat() erro inesperado: %v", err)
	}
	if len(published) != 2 {
		t.Fatalf("publicacoes = %d, want 2", len(published))
	}
	if !published[0].Equal(first) || !published[1].Equal(second) {
		t.Fatalf("timestamps publicados = %v", published)
	}
	if !ticker.stopped.Load() {
		t.Fatal("ticker deveria ter sido parado ao encerrar o heartbeat")
	}
}

func TestHeartbeatPropagatesPublishError(t *testing.T) {
	ticker := newFakeHeartbeatTicker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wantErr := errors.New("falha ao publicar heartbeat")
	done := startHeartbeat(ctx, func(time.Duration) heartbeatTicker {
		return ticker
	}, time.Second, func(time.Time) error {
		return wantErr
	})

	ticker.ch <- time.Now()

	if err := <-done; !errors.Is(err, wantErr) {
		t.Fatalf("erro = %v, want %v", err, wantErr)
	}
	if !ticker.stopped.Load() {
		t.Fatal("ticker deveria ter sido parado quando publish falha")
	}
}
