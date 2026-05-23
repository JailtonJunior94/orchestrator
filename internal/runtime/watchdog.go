package runtime

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

const (
	maxTickerInterval = 5 * time.Second
	minTickerInterval = time.Millisecond

	// absoluteSessionCapFactor define o teto absoluto da sessão como múltiplo do ActivityTimeout.
	// O cap de inatividade (sem progresso reconhecido) é resetado por Touch; o cap ABSOLUTO conta
	// desde o início e NUNCA é resetado. Garante que uma sessão não penda para sempre quando a CLI
	// emite eventos keep-alive/periódicos que resetam a inatividade sem encerrar o turn (observado
	// no codex-acp: completa a task mas a sessão nunca termina). Fator generoso para não matar
	// trabalho legítimo: trabalho real conclui em ~1x; o cap só age em sessões realmente penduradas.
	absoluteSessionCapFactor = 5
)

// ActivityWatchdog monitora o intervalo entre eventos e cancela o contexto (via
// context.CancelCauseFunc) quando esse intervalo excede o ActivityTimeout configurado.
//
// Uso típico:
//
//	wd := NewActivityWatchdog(timeout, cancel, clock)
//	wd.Start(ctx)
//	defer wd.Stop()
//	// em cada evento recebido:
//	wd.Touch()
type ActivityWatchdog struct {
	timeout   events.ActivityTimeout
	cancel    context.CancelCauseFunc
	lastSeen  atomic.Int64 // unix nano do último Touch() (cap de inatividade)
	startedAt atomic.Int64 // unix nano de Start() (cap absoluto; nunca resetado por Touch)
	clock     Clock
	stopCh    chan struct{}
	stopOnce  sync.Once
}

// NewActivityWatchdog cria um ActivityWatchdog.
// Se clock for nil, usa RealClock().
func NewActivityWatchdog(timeout events.ActivityTimeout, cancel context.CancelCauseFunc, clock Clock) *ActivityWatchdog {
	if clock == nil {
		clock = RealClock()
	}
	w := &ActivityWatchdog{
		timeout: timeout,
		cancel:  cancel,
		clock:   clock,
		stopCh:  make(chan struct{}),
	}
	// Inicializa lastSeen com o instante de criação para evitar disparo imediato.
	w.lastSeen.Store(clock.Now().UnixNano())
	return w
}

// Touch atualiza atomicamente o timestamp do último evento recebido.
func (w *ActivityWatchdog) Touch() {
	w.lastSeen.Store(w.clock.Now().UnixNano())
}

// Start lança a goroutine de monitoramento. Se o timeout estiver desabilitado (0),
// retorna imediatamente sem criar goroutine (RF-07).
//
// A goroutine encerra quando ctx é cancelado ou Stop() é chamado.
func (w *ActivityWatchdog) Start(ctx context.Context) {
	if w.timeout.Disabled() {
		return
	}

	// startedAt marca o início do monitoramento para o cap absoluto (não resetado por Touch).
	w.startedAt.Store(w.clock.Now().UnixNano())
	absoluteCap := w.timeout.Duration() * absoluteSessionCapFactor

	interval := w.tickerInterval()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case <-ticker.C:
				now := w.clock.Now()
				// Cap de inatividade: sem progresso reconhecido por timeout (resetado por Touch).
				last := time.Unix(0, w.lastSeen.Load())
				if now.Sub(last) > w.timeout.Duration() {
					w.cancel(ErrActivityTimeout)
					return
				}
				// Cap absoluto: sessão total excedeu timeout × fator, independentemente de keep-alives.
				started := time.Unix(0, w.startedAt.Load())
				if now.Sub(started) > absoluteCap {
					w.cancel(ErrActivityTimeout)
					return
				}
			}
		}
	}()
}

// Stop encerra a goroutine de monitoramento de forma idempotente.
// Pode ser chamado múltiplas vezes sem panic ou race condition.
func (w *ActivityWatchdog) Stop() {
	w.stopOnce.Do(func() {
		close(w.stopCh)
	})
}

// tickerInterval calcula o intervalo do ticker: min(timeout/2, 5s), nunca menor que 1ms.
func (w *ActivityWatchdog) tickerInterval() time.Duration {
	half := w.timeout.Duration() / 2
	if half > maxTickerInterval {
		half = maxTickerInterval
	}
	if half < minTickerInterval {
		half = minTickerInterval
	}
	return half
}
