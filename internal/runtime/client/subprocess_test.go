package client_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// skipIfNotAvailable pula o teste se o binário não estiver disponível.
func skipIfNotAvailable(t *testing.T, bin string) {
	t.Helper()
	if _, err := exec.LookPath(bin); err != nil {
		t.Skipf("binário %s não disponível no PATH: %v", bin, err)
	}
}

// TestAcpClient_Open_RealProcess: Open com processo real (cat) exercita startProcess + killProcess.
// cat fica aguardando stdin indefinidamente; Close() deve encerrá-lo.
func TestAcpClient_Open_RealProcess(t *testing.T) {
	t.Parallel()
	skipIfNotAvailable(t, "cat")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	factory := client.NewDefaultClientFactory()
	c := factory.New(t.TempDir())

	// Usa "cat" — fica lendo stdin e escrevendo em stdout.
	// Open vai falhar (cat não fala ACP), mas exercita startProcess.
	launcher := specs.NewBinaryLauncher("cat")
	err := c.Open(ctx, launcher, "prompt")
	// É esperado que falhe na negociação (cat não implementa ACP).
	// O que importa é que não entre em pânico e que Close() funcione.
	if err != nil {
		t.Logf("Open retornou erro esperado: %v", err)
	}

	// Close deve encerrar o processo.
	if closeErr := c.Close(); closeErr != nil {
		t.Logf("Close: %v", closeErr)
	}
}

// TestAcpClient_Close_WithRunningProcess: exercita killProcess com processo real.
func TestAcpClient_Close_WithRunningProcess(t *testing.T) {
	t.Parallel()
	skipIfNotAvailable(t, "sleep")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	factory := client.NewDefaultClientFactory()
	c := factory.New(t.TempDir())

	// sleep 100 permanece vivo; Open falha na negociação mas spawn funciona.
	launcher := specs.NewBinaryLauncher("sleep", "100")
	err := c.Open(ctx, launcher, "prompt")
	if err != nil {
		t.Logf("Open retornou erro esperado: %v", err)
	}

	start := time.Now()
	if closeErr := c.Close(); closeErr != nil {
		t.Logf("Close: %v", closeErr)
	}
	elapsed := time.Since(start)

	// Close deve completar em menos de 3s (SIGTERM + 2s max).
	if elapsed > 3*time.Second {
		t.Errorf("Close demorou %v; esperado < 3s", elapsed)
	}
}
