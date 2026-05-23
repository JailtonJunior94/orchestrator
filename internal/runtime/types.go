package runtime

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// RuntimeConfig agrupa os parâmetros operacionais de uma sessão ACP.
// Embutido em Job por composição (ADR-018). Todos os campos têm zero-value
// inerte que preserva o comportamento F1 (regressão zero).
type RuntimeConfig struct {
	// Timeout é o timeout de inatividade do watchdog. Zero = desabilitado (F1).
	// Mapeia o campo ActivityTimeout anterior de Job.
	Timeout events.ActivityTimeout
	// MaxRetries é o número máximo de tentativas extras após falha transitória.
	// 0 = uma tentativa (comportamento F1 preservado).
	MaxRetries int
	// RetryBackoffMultiplier é o fator de espera exponencial entre retentativas.
	// <=0 = sem espera (F1).
	RetryBackoffMultiplier float64
	// Concurrent é o grau máximo de paralelismo no runloop.
	// <=0 é normalizado para 1 (sequencial, F1) por ApplyDefaults.
	Concurrent int
	// BatchSize é o tamanho máximo do lote de tasks enviadas por iteração.
	// <=0 é normalizado para 1 (F1) por ApplyDefaults.
	BatchSize int
}

// ApplyDefaults normaliza os campos zero-value de RuntimeConfig para seus
// valores funcionais mínimos, preservando o comportamento F1 em todos os casos.
// Timeout, MaxRetries, RetryBackoffMultiplier: zero-value já é inerte — sem alteração.
// Concurrent <=0 → 1 (sequencial). BatchSize <=0 → 1.
func (c *RuntimeConfig) ApplyDefaults() {
	if c.Concurrent <= 0 {
		c.Concurrent = 1
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 1
	}
}

// Job descreve os parâmetros de uma execução ACP.
type Job struct {
	// Prompt é o texto enviado ao agente como entrada.
	Prompt string
	// WorkDir é o diretório de trabalho do agente.
	WorkDir string
	// EvidenceDir é o diretório onde artefatos de evidência são salvos.
	EvidenceDir string
	// RuntimeConfig embute os parâmetros operacionais unificados (ADR-018).
	// Timeout substitui o campo ActivityTimeout anterior; zero-value preserva F1.
	RuntimeConfig
	// Quiet suprime o output do renderer para o usuário.
	Quiet bool

	// Model é o identificador do modelo a ser usado pelo agente.
	// Para Codex: ex. "gpt-5.5" (default specs.DefaultCodexModel).
	// Default "" (string vazia) é tratado como no-op por codexBootstrapArgs.
	// Ignorado por Claude/Copilot via no-op em Spec.BootstrapArgs. ADR-013 D-02.
	Model string

	// ReasoningEffort define o nível de esforço de raciocínio do agente Codex.
	// Valores aceitos: "low", "medium", "high". Default "" omite o flag
	// (Codex usa o padrão do modelo). Ignorado por Claude/Copilot via no-op
	// em Spec.BootstrapArgs. ADR-013 D-02.
	ReasoningEffort string

	// AccessMode define o nível de acesso ao sistema de arquivos solicitado ao
	// agente Codex. Default specs.AccessModeRestricted ("restricted").
	// specs.AccessModeFull ("full") deve ser usado apenas em ambientes isolados.
	// Ignorado por Claude/Copilot via no-op em Spec.BootstrapArgs. ADR-013 D-02.
	AccessMode specs.AccessMode

	// AddDirs lista diretórios adicionais que o agente Codex pode acessar além
	// do WorkDir. Requer SupportsAddDirs=true no compozy. Default nil (sem
	// diretórios extras). Ignorado por Claude/Copilot via no-op em
	// Spec.BootstrapArgs. ADR-013 D-02.
	AddDirs []string

	// TaskFileName é o nome do arquivo de task ativo (ex: "task-3.0-...md").
	// Necessário para resolver memory.ReadTask em F3-Claude. Default "" = sem task file.
	TaskFileName string

	// MCPNested habilita o spawn do servidor MCP interno (F2-Claude).
	// Quando true, spawna mcpserver.Server em goroutine antes de c.Open e
	// injeta --mcp-server no launcher para que o Claude possa invocar run_agent.
	// Default false preserva comportamento F1-Claude.
	MCPNested bool

	// NoNormalize desabilita a normalização de tool-calls (F2-Claude, debug).
	// Quando true, events.BuildNormalizedToolCall não é chamado no loop de eventos.
	// Default false = normalização sempre ativa; raw_name e normalized_name gravados lado a lado.
	NoNormalize bool

	// MemoryLimits define os limites de compactação para o memory store 2-tier (F3-Claude).
	// Zero-value em cada campo significa usar os defaults de memory.DefaultLimits().
	// Propagado pelo task_loop via --memory-*-limit-* flags.
	MemoryLimits memory.Limits

	// DisableHooks desabilita TODOS os hooks Go in-process (F3-Claude, debug).
	// Quando true, governance, token_budget e memory_persist não são registrados.
	// Default false = hooks ativos (comportamento F3).
	// HARD: não afeta .claude/hooks/*.sh — shell hooks continuam ativos no modo interativo.
	DisableHooks bool

	// TasksDir é o caminho do diretório de tasks do PRD ativo (ex: "tasks/prd-foo").
	// Necessário para instanciar o memory.Store (ReadWorkflow + ReadTask).
	// Default "" = memory store desabilitado (regressão F1/F2 preservada).
	TasksDir string
	// MemoryLimitsExplicit indica que MemoryLimits veio de override explicito do usuario.
	// WindowLarge deve preservar esses limites em vez de aplicar defaults ampliados.
	MemoryLimitsExplicit bool

	// SkipDriftGuard desabilita apenas o SpecDriftHook (ADR-022, RG-01/RG-02).
	// Quando true, o hook spec_drift retorna nil imediatamente sem verificar hash.
	// Útil em ambientes de CI sem PRD rastreável ou durante desenvolvimento inicial.
	// Default false = guard ativo quando TasksDir != "".
	// NOTA: --disable-hooks (DisableHooks) desabilita TODOS os hooks, incluindo spec_drift.
	// SkipDriftGuard desabilita SOMENTE o spec_drift, mantendo governance/token_budget ativos.
	SkipDriftGuard bool

	// WindowClass é a classe da janela de contexto da CLI ativa (ADR-023).
	// Propagado da Spec pelo ACPRunner antes de Run(); não deve ser setado pelo usuário.
	// Zero-value (WindowStandard) preserva comportamento F1 exato (sem regressão).
	WindowClass specs.WindowClass

	// AutoReview habilita o auto-review opt-in (F5-Claude).
	// Quando true, após session end sem erro, spawna nova ACPRunner com prompt da
	// skill review (.agents/skills/review/SKILL.md) + diff acumulado da task.
	// HARD: default false; child session de review tem AutoReview=false forçado (anti-recursão).
	// HARD: Claude NÃO modifica internal/wrapper/ValidTools (ADR-014 §D-07).
	AutoReview bool
}
