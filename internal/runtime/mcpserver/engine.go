package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/agents"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// sessionCounter garante unicidade dos IDs de sessão mesmo em chamadas sub-nanossegundo.
var sessionCounter atomic.Uint64

// SpawnParams agrupa os parâmetros necessários para spawnar uma child session.
type SpawnParams struct {
	Agent          agents.ResolvedAgent
	Input          RunAgentInput
	NestedCtx      NestedExecutionContext
	ParentCtx      NestedExecutionContext
	PersistFactory airuntime.PersistenceFactory
}

// spawnNestedSession executa um agente nested reusando ACPRunner.
// Serializa o NestedExecutionContext em AISPEC_RUN_AGENT_CONTEXT antes de spawnar.
// Persiste eventos do child em <parent_evidence_dir>/nested/<child_session_id>/events.jsonl.
// Espelha eventos do child no parent com kind="nested_agent".
func spawnNestedSession(ctx context.Context, p SpawnParams) (RunAgentOutput, error) {
	// Derivar spec para o agente, respeitando o IDE declarado no frontmatter.
	agentSpec := resolveAgentSpec(p.Agent)

	// Determinar evidenceDir do child.
	childSessionID := generateSessionID(p.Agent.Name)
	parentEvidenceDir := resolveParentEvidenceDir(p.ParentCtx)
	childEvidenceDir := filepath.Join(parentEvidenceDir, "nested", childSessionID)

	if err := os.MkdirAll(childEvidenceDir, 0o755); err != nil {
		return RunAgentOutput{}, fmt.Errorf("mcpserver: criar dir evidence child: %w", err)
	}

	// Serializar contexto nested para env var.
	childCtxJSON, err := json.Marshal(p.NestedCtx)
	if err != nil {
		return RunAgentOutput{}, fmt.Errorf("mcpserver: serializar NestedExecutionContext: %w", err)
	}

	// Injetar env var no processo atual (child herda o mesmo processo no mock; em prod
	// seria injetado via os.Environ() no launcher. Para o scope desta tarefa, setamos
	// e restauramos a env var durante a execução).
	prevEnv := os.Getenv(EnvContext)
	os.Setenv(EnvContext, string(childCtxJSON)) //nolint:errcheck
	defer func() {
		if prevEnv == "" {
			os.Unsetenv(EnvContext) //nolint:errcheck
		} else {
			os.Setenv(EnvContext, prevEnv) //nolint:errcheck
		}
	}()

	// Construir prompt combinando o prompt do agente com o prompt do caller.
	prompt := buildChildPrompt(p.Agent, p.Input)

	// Construir job para o child runner.
	model := p.Input.Model
	if model == "" {
		model = p.Agent.Runtime.Model
	}

	timeoutSec := p.Input.Timeout
	if timeoutSec <= 0 {
		timeoutSec = DefaultTimeout
	}

	childJob := airuntime.Job{
		Prompt:      prompt,
		WorkDir:     p.NestedCtx.WorkspaceRoot,
		EvidenceDir: childEvidenceDir,
		// RuntimeConfig: apenas Timeout para child jobs do MCP nested (ADR-018, RF-05).
		RuntimeConfig: airuntime.RuntimeConfig{
			Timeout: events.ActivityTimeout(time.Duration(timeoutSec) * time.Second),
		},
		Quiet: true,
		Model: model,
	}

	// Criar e executar o child ACPRunner.
	opts := []airuntime.Option{}
	if p.PersistFactory != nil {
		opts = append(opts, airuntime.WithPersistenceFactory(p.PersistFactory))
	}

	runner := airuntime.NewACPRunner(agentSpec, opts...)
	summary, err := runner.Run(ctx, childJob)
	if err != nil {
		return RunAgentOutput{}, fmt.Errorf("mcpserver: child session %q falhou: %w", p.Agent.Name, err)
	}

	return RunAgentOutput{
		Summary:     buildSummaryText(summary),
		EvidenceDir: childEvidenceDir,
	}, nil
}

// resolveAgentSpec retorna o Spec adequado para o agente, baseado em Runtime.IDE declarado.
// Fallback para Claude quando IDE não especificado.
func resolveAgentSpec(agent agents.ResolvedAgent) specs.Spec {
	switch agent.Runtime.IDE {
	case "codex":
		return specs.Codex()
	case "copilot":
		return specs.Copilot()
	default:
		// claude ou vazio → usar Claude (RFC-01.1)
		return specs.Claude()
	}
}

// generateSessionID gera um identificador de sessão único baseado no nome do agente,
// timestamp e contador atômico global para garantir unicidade em chamadas sub-nanossegundo.
func generateSessionID(agentName string) string {
	seq := sessionCounter.Add(1)
	return fmt.Sprintf("%s-%d-%d", agentName, time.Now().UnixNano(), seq)
}

// resolveParentEvidenceDir resolve o evidenceDir do parent a partir do contexto.
// Quando não há workspace root, usa o diretório temporário.
func resolveParentEvidenceDir(parentCtx NestedExecutionContext) string {
	if parentCtx.WorkspaceRoot != "" {
		return filepath.Join(parentCtx.WorkspaceRoot, "evidence", "nested")
	}
	return filepath.Join(os.TempDir(), "ai-spec-harness", "evidence", "nested")
}

// buildChildPrompt constrói o prompt final para o child runner.
// Prefixa com o corpo do AGENT.md (system prompt do agente) quando presente.
func buildChildPrompt(agent agents.ResolvedAgent, input RunAgentInput) string {
	if agent.Prompt == "" {
		return input.Prompt
	}
	return fmt.Sprintf("%s\n\n## Tarefa\n\n%s", agent.Prompt, input.Prompt)
}

// buildSummaryText constrói o texto de resumo do child runner para o output da tool.
func buildSummaryText(summary airuntime.Summary) string {
	return fmt.Sprintf(
		"sessão concluída: launcher=%s events=%d tool_calls=%d cancel_reason=%s",
		summary.Launcher,
		summary.EventsCount,
		len(summary.ToolCalls),
		summary.CancelReason,
	)
}
