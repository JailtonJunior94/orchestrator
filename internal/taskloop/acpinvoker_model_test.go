package taskloop_test

import (
	"context"
	"strings"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/taskloop"
)

type instantHandshakeWaiter struct{}

func (instantHandshakeWaiter) Wait(_ context.Context) error { return nil }

func (instantHandshakeWaiter) Path() string { return "/fake/handshake/sentinel" }

type instantHandshakeWaiterFactory struct{}

func (instantHandshakeWaiterFactory) NewWaiter() (client.HandshakeWaiter, func() error, error) {
	return instantHandshakeWaiter{}, func() error { return nil }, nil
}

func buildOpenCodeTestRunner(t *testing.T, ctx context.Context, script *acpfake.Script) *airuntime.ACPRunner {
	t.Helper()
	return airuntime.NewACPRunner(specs.NewCatalog().OpenCode(),
		airuntime.NewCatalog().WithProber(&testProber{
			launcher: specs.NewBinaryLauncher("/fake/opencode"),
		}),
		airuntime.NewCatalog().WithClientFactory(&testClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.NewCatalog().WithPersistenceFactory(&testPersistenceFactory{}),
		airuntime.NewCatalog().WithRenderer(&testDiscardRenderer{}),
		airuntime.NewCatalog().WithHandshakeWaiterFactory(instantHandshakeWaiterFactory{}),
	)
}

func TestACPInvokerPropagatesModelToWindowClass(t *testing.T) {
	prompt := strings.Repeat("palavra ", 5000)

	t.Run("large window model unlocks large budget", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		script := acpfake.NewScript().AppendAgentMessage("ok").AppendSessionEnd()
		invoker := taskloop.NewACPInvoker(buildOpenCodeTestRunner(t, ctx, script), true, 0)

		_, _, _, err := invoker.Invoke(ctx, prompt, workDirWithAgentsMD(t), "google/gemini-2.5-pro")
		if err != nil {
			t.Fatalf("Invoke com modelo de janela grande falhou: %v", err)
		}
	})

	t.Run("empty model keeps standard budget", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		script := acpfake.NewScript().AppendAgentMessage("ok").AppendSessionEnd()
		invoker := taskloop.NewACPInvoker(buildOpenCodeTestRunner(t, ctx, script), true, 0)

		_, _, _, err := invoker.Invoke(ctx, prompt, workDirWithAgentsMD(t), "")
		if err == nil {
			t.Fatal("Invoke sem modelo deveria estourar o budget standard de opencode")
		}
		if !strings.Contains(err.Error(), "token budget") {
			t.Fatalf("erro esperado de token budget, obteve: %v", err)
		}
	})
}
