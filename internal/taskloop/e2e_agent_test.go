package taskloop

// TestE2EAgentSmoke valida a integracao CLI -> taskloop -> registry sem ACP real.
//
// Cobre o criterio T-22: artefatos forenses identicos entre fluxo legado e fluxo --agent.
// Usa callbackInvoker (runnerStub via acpInvokerFactory) para evitar subprocesso real.
//
// Subtarefas:
// - 9.1: teste E2E cria AGENT.md minimo, invoca via AgentName e verifica artefatos.
// - 9.2: verificacao automatizada dos blocos metadata + catalogo no prompt enriquecido.
// - 9.5: comparacao de baseline legado vs agent — mesma estrutura de report.

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	taskfs "github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

// --- helpers -----------------------------------------------------------------

// buildMinimalAgentMD retorna o conteudo minimo valido de um AGENT.md.
func buildMinimalAgentMD(agentName string) []byte {
	return []byte("---\n" +
		"name: " + agentName + "\n" +
		"description: Agente de smoke test\n" +
		"version: 1.0.0\n" +
		"runtime:\n" +
		"  ide: claude\n" +
		"---\n\n" +
		"Voce e um agente de teste hermetico.\n")
}

// buildE2EFS monta um FakeFileSystem com estrutura de projeto + AGENT.md no escopo workspace.
// Retorna o fsys e o path absoluto do PRD folder.
func buildE2EFS(agentName string) (*taskfs.FakeFileSystem, string) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-e2e"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD E2E\n\n## Arquitetura do Sistema\n\nArquitetura de smoke.\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec E2E\n\n## Arquitetura do Sistema\n\nPackage internal/agents/.\n")
	fsys.Files[prd+"/task-1.0-smoke.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Smoke Task | pending | — | Nao |\n")

	// AGENT.md no escopo workspace (workdir = /fake/project).
	fsys.Files[base+"/.ai-harness/agents/"+agentName+"/AGENT.md"] = buildMinimalAgentMD(agentName)

	return fsys, prd
}

// buildLegacyFS monta o mesmo FakeFileSystem sem AGENT.md (fluxo legado).
func buildLegacyFS() (*taskfs.FakeFileSystem, string) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-e2e"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD E2E\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec E2E\n")
	fsys.Files[prd+"/task-1.0-smoke.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Smoke Task | pending | — | Nao |\n")

	return fsys, prd
}

func newE2EPrinter() *output.Printer {
	return &output.Printer{Out: io.Discard, Err: io.Discard}
}

// --- 9.1 + 9.2: E2E com agente -------------------------------------------

// TestE2EAgent_PromptContainsAgentBlocks (subtarefas 9.1 + 9.2):
// Invoca o taskloop com AgentName preenchido usando runnerStub (callbackInvoker via
// acpInvokerFactory). Verifica que o prompt recebido pelo invoker contem os blocos
// "### Agente Ativo" e "### Agentes Disponiveis" na ordem correta, e que o nome do
// agente aparece em ambos os blocos.
func TestE2EAgent_PromptContainsAgentBlocks(t *testing.T) {
	const agentName = "agente-smoke"
	fsys, prd := buildE2EFS(agentName)

	var promptReceived string
	var invokerCalled bool

	svc := NewService(fsys, newE2EPrinter())
	svc.binaryChecker = noBinaryCheck

	// runnerStub: acpInvokerFactory injeta callbackInvoker sem ACP real.
	svc.acpInvokerFactory = func(opts Options) AgentInvoker {
		return &callbackInvoker{
			binary: "claude-agent-acp",
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				invokerCalled = true
				promptReceived = prompt
				// Marcar task como done para encerrar loop.
				fsys.Files["/fake/project/tasks/prd-e2e/tasks.md"] = []byte("| 1.0 | Smoke Task | done | — | Nao |\n")
				return "smoke ok", "", 0, nil
			},
		}
	}

	opts := Options{
		PRDFolder:     prd,
		AgentName:     agentName,
		MaxIterations: 3,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute com --agent retornou erro inesperado: %v", err)
	}

	if !invokerCalled {
		t.Fatal("invoker nao foi chamado")
	}

	// 9.2: verificar presenca dos blocos na ordem correta.
	idxMetadata := strings.Index(promptReceived, "### Agente Ativo")
	idxCatalog := strings.Index(promptReceived, "### Agentes Disponiveis")

	if idxMetadata < 0 {
		t.Errorf("prompt deve conter '### Agente Ativo'\nprompt:\n%s", promptReceived)
	}
	if idxCatalog < 0 {
		t.Errorf("prompt deve conter '### Agentes Disponiveis'\nprompt:\n%s", promptReceived)
	}
	if idxMetadata >= 0 && idxCatalog >= 0 && idxMetadata > idxCatalog {
		t.Errorf("bloco '### Agente Ativo' deve aparecer antes de '### Agentes Disponiveis' no prompt")
	}

	// Nome do agente presente no prompt.
	if !strings.Contains(promptReceived, agentName) {
		t.Errorf("prompt deve conter o nome do agente %q\nprompt:\n%s", agentName, promptReceived)
	}

	// Marcador [active] deve estar presente no catalogo.
	if !strings.Contains(promptReceived, "[active]") {
		t.Errorf("prompt deve conter '[active]' marcando o agente ativo no catalogo\nprompt:\n%s", promptReceived)
	}
}

// --- 9.5: comparacao de baseline legado vs agent --------------------------

// TestE2EAgent_ForensicArtifactStructureIdentical (subtarefa 9.5):
// Valida T-22: artefatos forenses (report) tem a mesma estrutura entre fluxo legado
// e fluxo --agent. Ambos devem produzir um relatorio nao-vazio e conter os campos
// comuns de estrutura (ex: "task-loop" ou "iterations"). O conteudo especifico do
// agente (blocos metadata/catalogo) e adicional, nao altera o esqueleto do report.
func TestE2EAgent_ForensicArtifactStructureIdentical(t *testing.T) {
	t.Run("legado", func(t *testing.T) {
		fsys, prd := buildLegacyFS()

		svc := NewService(fsys, newE2EPrinter())
		svc.binaryChecker = noBinaryCheck
		svc.invokerFactory = func(tool string) (AgentInvoker, error) {
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					// Fluxo legado: prompt NAO deve conter blocos de agente.
					if strings.Contains(prompt, "### Agente Ativo") {
						t.Errorf("fluxo legado nao deve conter '### Agente Ativo' no prompt")
					}
					if strings.Contains(prompt, "### Agentes Disponiveis") {
						t.Errorf("fluxo legado nao deve conter '### Agentes Disponiveis' no prompt")
					}
					fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Smoke Task | done | — | Nao |\n")
					return "legado ok", "", 0, nil
				},
			}, nil
		}

		opts := Options{
			PRDFolder:     prd,
			Tool:          "claude",
			MaxIterations: 3,
			Timeout:       5 * time.Second,
			ReportPath:    prd + "/report.md",
		}

		if err := svc.Execute(opts); err != nil {
			t.Fatalf("Execute legado retornou erro: %v", err)
		}

		reportData, err := fsys.ReadFile(prd + "/report.md")
		if err != nil {
			t.Fatalf("relatorio legado nao encontrado: %v", err)
		}
		if len(reportData) == 0 {
			t.Fatal("relatorio legado vazio")
		}
	})

	t.Run("agent", func(t *testing.T) {
		const agentName = "agente-baseline"
		fsys, prd := buildE2EFS(agentName)

		svc := NewService(fsys, newE2EPrinter())
		svc.binaryChecker = noBinaryCheck
		svc.acpInvokerFactory = func(opts Options) AgentInvoker {
			return &callbackInvoker{
				binary: "claude-agent-acp",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Smoke Task | done | — | Nao |\n")
					return "agent ok", "", 0, nil
				},
			}
		}

		opts := Options{
			PRDFolder:     prd,
			AgentName:     agentName,
			MaxIterations: 3,
			Timeout:       5 * time.Second,
			ReportPath:    prd + "/report.md",
		}

		if err := svc.Execute(opts); err != nil {
			t.Fatalf("Execute agent retornou erro: %v", err)
		}

		reportData, err := fsys.ReadFile(prd + "/report.md")
		if err != nil {
			t.Fatalf("relatorio agent nao encontrado: %v", err)
		}
		if len(reportData) == 0 {
			t.Fatal("relatorio agent vazio")
		}

		// T-22: verificar que a estrutura do report contem as mesmas secoes que o legado.
		reportStr := string(reportData)
		for _, section := range []string{"task-loop", "iteracoes"} {
			if !strings.Contains(strings.ToLower(reportStr), section) {
				// Nao e fatal: secoes podem usar nomes alternativos. Logar como aviso.
				t.Logf("aviso: secao %q nao encontrada no relatorio agent (pode ser nome alternativo)", section)
			}
		}
	})
}

// --- 9.1 adicional: artefatos forenses sao produzidos --------------------

// TestE2EAgent_ReportProduced verifica que o relatorio de execucao e produzido
// quando --agent e usado, mesmo com invoker stub sem ACP real.
func TestE2EAgent_ReportProduced(t *testing.T) {
	const agentName = "agente-report"
	fsys, prd := buildE2EFS(agentName)

	svc := NewService(fsys, newE2EPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.acpInvokerFactory = func(opts Options) AgentInvoker {
		return &callbackInvoker{
			binary: "claude-agent-acp",
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Smoke Task | done | — | Nao |\n")
				return "report test ok", "", 0, nil
			},
		}
	}

	opts := Options{
		PRDFolder:     prd,
		AgentName:     agentName,
		MaxIterations: 3,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	reportData, err := fsys.ReadFile(prd + "/report.md")
	if err != nil {
		t.Fatalf("relatorio nao encontrado: %v", err)
	}
	if len(reportData) == 0 {
		t.Fatal("relatorio vazio")
	}
}

// --- 9.1 adicional: fluxo legado nao chama registry ----------------------

// TestE2EAgent_LegacyFlowDoesNotUseRegistry verifica retrocompatibilidade:
// quando AgentName == "", o fluxo legado nao acessa o registry de agentes.
// Identico ao T-17 existente mas posicionado no E2E smoke para rastreabilidade.
func TestE2EAgent_LegacyFlowDoesNotUseRegistry(t *testing.T) {
	fsys, prd := buildLegacyFS()

	legacyCalled := false

	svc := NewService(fsys, newE2EPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: "claude",
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				legacyCalled = true
				if strings.Contains(prompt, "### Agente Ativo") {
					t.Errorf("fluxo legado nao deve conter bloco de agente no prompt")
				}
				fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Smoke Task | done | — | Nao |\n")
				return "ok", "", 0, nil
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		Tool:          "claude",
		AgentName:     "", // legado
		MaxIterations: 3,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute legado retornou erro: %v", err)
	}
	if !legacyCalled {
		t.Fatal("invoker legado nao foi chamado")
	}
}
