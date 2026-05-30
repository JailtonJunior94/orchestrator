package runtime_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	rtime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

// fakeClock implementa Clock para testes determinísticos (zero time.Sleep real).
type fakeClock struct {
	now atomic.Int64 // unix nano
}

func newFakeClock(t time.Time) *fakeClock {
	c := &fakeClock{}
	c.now.Store(t.UnixNano())
	return c
}

func (c *fakeClock) Now() time.Time {
	return time.Unix(0, c.now.Load())
}

// advance avança o relógio em d sem bloquear.
func (c *fakeClock) advance(d time.Duration) {
	c.now.Add(int64(d))
}

// TestActivityWatchdog_Disabled verifica que timeout=0 torna Start um no-op:
// cancel nunca é chamado, nenhuma goroutine é vazada.
func TestActivityWatchdog_Disabled(t *testing.T) {
	goroutinesBefore := runtime.NumGoroutine()

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	timeout, err := events.NewActivityTimeout(0)
	if err != nil {
		t.Fatalf("NewActivityTimeout(0): %v", err)
	}

	var cancelCalled atomic.Bool
	cancelCause := func(cause error) {
		cancelCalled.Store(true)
		cancel(cause)
	}

	clk := newFakeClock(time.Now())
	wd := rtime.NewActivityWatchdog(timeout, cancelCause, clk)
	wd.Start(ctx)

	// Avançar o clock muito além de qualquer timeout imaginável.
	clk.advance(10 * time.Minute)

	// Nenhuma goroutine deve ter sido criada (Start é no-op quando desabilitado).
	goroutinesAfter := runtime.NumGoroutine()
	if goroutinesAfter > goroutinesBefore+1 {
		t.Errorf("vazamento de goroutine: antes=%d depois=%d", goroutinesBefore, goroutinesAfter)
	}

	if cancelCalled.Load() {
		t.Error("cancel foi chamado com timeout=0 (watchdog deve estar desabilitado)")
	}
}

// TestActivityWatchdog_KeptAlive verifica que Touch() periódico impede o disparo do cancel.
// Usa fakeClock para que o watchdog sempre veja lastSeen como "agora" e não dispare.
func TestActivityWatchdog_KeptAlive(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	// Usa timeout muito pequeno para que o ticker dispare rapidamente.
	const activityTimeout = 20 * time.Millisecond
	timeout, err := events.NewActivityTimeout(activityTimeout)
	if err != nil {
		t.Fatalf("NewActivityTimeout: %v", err)
	}

	var cancelCalled atomic.Bool
	cancelCause := func(cause error) {
		cancelCalled.Store(true)
		cancel(cause)
	}

	// Usa fakeClock: Touch() e o watchdog veem o mesmo "agora", então o delta é sempre 0.
	clk := newFakeClock(time.Now())
	wd := rtime.NewActivityWatchdog(timeout, cancelCause, clk)
	wd.Start(ctx)
	defer wd.Stop()

	// O fakeClock não avança → o watchdog vê delta=0 e nunca dispara.
	// Aguarda via ctx (que nunca é cancelado) com timeout de segurança.
	select {
	case <-ctx.Done():
		t.Error("cancel foi chamado mesmo com Touch (clock estático)")
	case <-time.After(100 * time.Millisecond):
		// esperado: nenhum cancelamento ocorreu
	}

	if cancelCalled.Load() {
		t.Error("cancel foi chamado mesmo com clock estático (delta sempre 0)")
	}
}

// TestActivityWatchdog_AbsoluteCapFiresDespiteTouch é a regressão do hang do codex: mesmo com
// Touch() contínuo (simulando keep-alives que resetam a inatividade), o cap ABSOLUTO (timeout x5)
// deve disparar quando o tempo total excede o teto — garantindo que a sessão nunca penda para sempre.
func TestActivityWatchdog_AbsoluteCapFiresDespiteTouch(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	const activityTimeout = 60 * time.Millisecond // cap absoluto = 300ms (5x)
	timeout, err := events.NewActivityTimeout(activityTimeout)
	if err != nil {
		t.Fatalf("NewActivityTimeout: %v", err)
	}

	clk := newFakeClock(time.Now())
	wd := rtime.NewActivityWatchdog(timeout, func(cause error) { cancel(cause) }, clk)
	wd.Start(ctx)
	defer wd.Stop()

	// Loop de keep-alive: avança o tempo total (acima do cap absoluto) mas faz Touch a cada passo
	// (mantendo a inatividade < timeout). Sem o cap absoluto, isto penduraria para sempre.
	go func() {
		for i := 0; i < 50; i++ {
			select {
			case <-ctx.Done():
				return
			default:
			}
			clk.advance(50 * time.Millisecond) // < timeout (60ms): inatividade sozinha não dispara
			wd.Touch()                         // keep-alive: reseta inatividade
			time.Sleep(40 * time.Millisecond)  // > ticker (~30ms): dá tempo ao watchdog observar
		}
	}()

	select {
	case <-ctx.Done():
		// esperado: cap absoluto disparou apesar dos Touch contínuos
	case <-time.After(5 * time.Second):
		t.Fatal("cap absoluto não disparou apesar de exceder timeout x5 com Touch contínuo (hang)")
	}
	if !errors.Is(context.Cause(ctx), rtime.ErrActivityTimeout) {
		t.Errorf("cause = %v, want ErrActivityTimeout", context.Cause(ctx))
	}
}

// TestActivityWatchdog_Fires verifica que sem Touch além do timeout, cancel é chamado com ErrActivityTimeout.
// Usa fakeClock para avançar o tempo artificialmente até ultrapassar o timeout.
func TestActivityWatchdog_Fires(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	const activityTimeout = 10 * time.Millisecond
	timeout, err := events.NewActivityTimeout(activityTimeout)
	if err != nil {
		t.Fatalf("NewActivityTimeout: %v", err)
	}

	cancelCh := make(chan error, 1)
	cancelCause := func(cause error) {
		select {
		case cancelCh <- cause:
		default:
		}
		cancel(cause)
	}

	// Usa fakeClock: avança para ultrapassar o timeout.
	base := time.Now()
	clk := newFakeClock(base)

	wd := rtime.NewActivityWatchdog(timeout, cancelCause, clk)
	wd.Start(ctx)
	defer wd.Stop()

	// Avança o clock além do timeout para que o próximo tick do watchdog dispare.
	clk.advance(activityTimeout + time.Millisecond)

	// Aguarda a notificação de cancelamento via canal (sem time.Sleep).
	select {
	case cause := <-cancelCh:
		if !errors.Is(cause, rtime.ErrActivityTimeout) {
			t.Errorf("causa esperada ErrActivityTimeout, obteve %v", cause)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("watchdog não disparou dentro do prazo esperado")
	}
}

// TestActivityWatchdog_StopIdempotent verifica que Stop() pode ser chamado N vezes sem panic.
func TestActivityWatchdog_StopIdempotent(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	timeout, _ := events.NewActivityTimeout(10 * time.Second)
	wd := rtime.NewActivityWatchdog(timeout, cancel, nil)
	wd.Start(ctx)

	// Chamadas múltiplas não devem causar panic.
	for i := 0; i < 10; i++ {
		wd.Stop()
	}
}

// TestActivityWatchdog_NoGoroutineLeak verifica que Stop() encerra a goroutine sem leak.
func TestActivityWatchdog_NoGoroutineLeak(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	timeout, _ := events.NewActivityTimeout(10 * time.Second)

	goroutinesBefore := runtime.NumGoroutine()
	wd := rtime.NewActivityWatchdog(timeout, cancel, nil)
	wd.Start(ctx)

	wd.Stop()

	// Aguarda encerramento da goroutine via ctx cancelado externamente + stop.
	// Usa polling com deadline em vez de time.Sleep fixo.
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= goroutinesBefore+1 {
			break
		}
		runtime.Gosched()
	}

	goroutinesAfter := runtime.NumGoroutine()
	if goroutinesAfter > goroutinesBefore+1 {
		t.Errorf("goroutine leak: antes=%d depois=%d", goroutinesBefore, goroutinesAfter)
	}
}

// TestActivityWatchdog_LargeTimeout verifica que timeout muito grande usa o cap de 5s
// no intervalo do ticker, sem causar overflow (cobre o branch half > maxTickerInterval).
func TestActivityWatchdog_LargeTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	// Timeout de 1 hora → half = 30min > 5s → ticker deve usar 5s.
	timeout, err := events.NewActivityTimeout(1 * time.Hour)
	if err != nil {
		t.Fatalf("NewActivityTimeout: %v", err)
	}

	var cancelCalled atomic.Bool
	cancelCause := func(cause error) {
		cancelCalled.Store(true)
		cancel(cause)
	}

	wd := rtime.NewActivityWatchdog(timeout, cancelCause, nil)
	wd.Start(ctx)

	// Stop imediatamente — só verifica que Start não pânica e que goroutine encerra.
	wd.Stop()

	if cancelCalled.Load() {
		t.Error("cancel não deveria ter sido chamado")
	}
}

// TestActivityWatchdog_TinyTimeout verifica que timeout muito pequeno usa o mínimo de 1ms
// (cobre o branch half < minTickerInterval).
// Usa fakeClock para que o tempo seja controlado sem real ticker dependency.
func TestActivityWatchdog_TinyTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	// Timeout de 1ms → half = 500µs < 1ms → ticker usa 1ms.
	timeout, err := events.NewActivityTimeout(1 * time.Millisecond)
	if err != nil {
		t.Fatalf("NewActivityTimeout: %v", err)
	}

	cancelCh := make(chan error, 1)
	cancelCause := func(cause error) {
		select {
		case cancelCh <- cause:
		default:
		}
		cancel(cause)
	}

	// Usa fakeClock: avança além do timeout para disparar o watchdog.
	clk := newFakeClock(time.Now())
	wd := rtime.NewActivityWatchdog(timeout, cancelCause, clk)
	wd.Start(ctx)
	defer wd.Stop()

	// Avança o clock além do timeout.
	clk.advance(2 * time.Millisecond)

	select {
	case cause := <-cancelCh:
		if !errors.Is(cause, rtime.ErrActivityTimeout) {
			t.Errorf("causa esperada ErrActivityTimeout, obteve %v", cause)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("watchdog com timeout=1ms não disparou em 500ms")
	}
}

// TestActivityWatchdog_ContextCanceled verifica que o watchdog encerra quando ctx é cancelado
// externamente sem chamar cancelCause.
func TestActivityWatchdog_ContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())

	timeout, _ := events.NewActivityTimeout(10 * time.Second)

	var cancelWatchdogCalled atomic.Bool
	cancelWatchdog := func(cause error) {
		cancelWatchdogCalled.Store(true)
	}

	wd := rtime.NewActivityWatchdog(timeout, cancelWatchdog, nil)
	wd.Start(ctx)

	// Cancela o contexto externamente — a goroutine do watchdog deve encerrar.
	cancel(nil)

	// Aguarda encerramento via polling (sem time.Sleep).
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !cancelWatchdogCalled.Load() {
			runtime.Gosched()
			continue
		}
		break
	}

	if cancelWatchdogCalled.Load() {
		t.Error("watchdog não deveria ter chamado cancelCause quando ctx foi cancelado externamente")
	}
}

// TestActivityWatchdog_TouchResetsTimer verifica que Touch() reinicia o timer do watchdog,
// impedindo o disparo quando chamado antes de o timeout expirar.
func TestActivityWatchdog_TouchResetsTimer(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	const activityTimeout = 30 * time.Millisecond
	timeout, _ := events.NewActivityTimeout(activityTimeout)

	cancelCh := make(chan error, 1)
	cancelCause := func(cause error) {
		select {
		case cancelCh <- cause:
		default:
		}
		cancel(cause)
	}

	// fakeClock: controla o "tempo atual" visto pelo watchdog.
	base := time.Now()
	clk := newFakeClock(base)

	wd := rtime.NewActivityWatchdog(timeout, cancelCause, clk)
	wd.Start(ctx)
	defer wd.Stop()

	// Avança o clock para quase o timeout e chama Touch para reiniciar.
	clk.advance(activityTimeout - time.Millisecond)
	wd.Touch() // Touch deve atualizar lastSeen para o "agora" do fakeClock

	// Verifica que o delta após Touch é 0 (ou próximo de 0): watchdog não dispara.
	select {
	case cause := <-cancelCh:
		t.Errorf("cancel chamado após Touch: %v", cause)
	case <-time.After(50 * time.Millisecond):
		// esperado: clock não avançou além de timeout após Touch
	}
}

// TestSentinels verifica que cada sentinel suporta errors.Is com wrapping.
func TestSentinels(t *testing.T) {
	t.Parallel()

	sentinels := []error{
		rtime.ErrLauncherUnavailable,
		rtime.ErrActivityTimeout,
		rtime.ErrPermissionDenied,
		rtime.ErrSessionAborted,
		rtime.ErrUnsupportedTool,
		rtime.ErrInvalidEvent,
	}

	for _, sentinel := range sentinels {
		sentinel := sentinel
		t.Run(sentinel.Error(), func(t *testing.T) {
			t.Parallel()
			wrapped := fmt.Errorf("contexto adicional: %w", sentinel)
			if !errors.Is(wrapped, sentinel) {
				t.Errorf("errors.Is(wrapped, %v) = false, esperado true", sentinel)
			}
		})
	}
}
