package taskloop

import (
	"context"
	"time"
)

const defaultHeartbeatInterval = 5 * time.Second

type heartbeatTicker interface {
	Chan() <-chan time.Time
	Stop()
}

type heartbeatTickerFactory func(interval time.Duration) heartbeatTicker

type realtimeHeartbeatTicker struct {
	ticker *time.Ticker
}

func newRealtimeHeartbeatTicker(interval time.Duration) heartbeatTicker {
	return &realtimeHeartbeatTicker{ticker: time.NewTicker(interval)}
}

func (t *realtimeHeartbeatTicker) Chan() <-chan time.Time {
	return t.ticker.C
}

func (t *realtimeHeartbeatTicker) Stop() {
	t.ticker.Stop()
}

func startHeartbeat(
	ctx context.Context,
	factory heartbeatTickerFactory,
	interval time.Duration,
	publish func(time.Time) error,
) <-chan error {
	done := make(chan error, 1)
	if publish == nil {
		close(done)
		return done
	}
	if factory == nil {
		factory = newRealtimeHeartbeatTicker
	}
	if interval <= 0 {
		interval = defaultHeartbeatInterval
	}

	ticker := factory(interval)
	go func() {
		defer close(done)
		defer ticker.Stop()

		for {
			// Draina ticks ja enfileirados antes de honrar o cancelamento para nao
			// perder o ultimo heartbeat observado ao encerrar a invocacao.
			select {
			case at := <-ticker.Chan():
				if err := publish(at); err != nil {
					done <- err
					return
				}
				continue
			default:
			}

			select {
			case <-ctx.Done():
				return
			case at := <-ticker.Chan():
				if err := publish(at); err != nil {
					done <- err
					return
				}
			}
		}
	}()

	return done
}
