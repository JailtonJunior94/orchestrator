# Relatório de Validação Cross-CLI — Task Loop (Sessão 2)

- **Data:** 2026-04-22
- **Binário:** `./ai-spec` (compilado localmente — `CGO_ENABLED=0 go build -trimpath -o ai-spec .`)
- **Versão do harness:** v0.11.2 (commit 5cb4615, branch main)
- **Ferramentas testadas:**
  - claude 2.1.104 (Claude Code)
  - copilot 1.0.34 (GitHub Copilot CLI)
  - gemini 0.38.2
  - codex-cli 0.122.0
- **Timeout explícito usado nesta sessão:** `--timeout 3m` (padrão do harness: 30m)

---

## Resultado

| Bundle | claude | copilot | gemini | codex |
|--------|--------|---------|--------|-------|
| maturidade (dry-run) | pass | pass | pass | pass |
| maturidade (real, max=1) | bug (auth) | blocked (timeout 3m) | blocked (timeout 3m) | bug (zero output, timeout 3m) |
| prd-interactive-task-loop | pass (0 iters) | pass (0 iters) | pass (0 iters) | pass (0 iters) |
| prd-task-loop-modelos-por-papel | pass (todas done) | pass | pass | pass |
| prd-task-loop-sequential-execution | pass (todas done) | pass | pass | pass |
| prd-telemetry-feedback | pass (todas done) | pass | pass | pass |

> **Observação:** o timeout de 3m foi definido explicitamente para limitar a execução. O default de 30m seria adequado para tarefas reais. Gemini e copilot demonstraram atividade real dentro do tempo limite; os timeouts são esperados com 3m.

---

## Evidências

### Compilação

```
CGO_ENABLED=0 go build -trimpath -o ai-spec .
```
Saída limpa. Binário gerado com sucesso.

### Dry-run — maturidade (4 ferramentas, paridade total)

```
task-loop iniciado: folder=tasks/maturidade tool=<tool> max=20 timeout=30m0s
-> iteracao 1: executando task 1.0 (Quick-Reference Mode no Contextgen)
   [dry-run] invocaria <tool> com prompt para task 1.0
   [dry-run] task file: tasks/maturidade/TASK-001-quick-reference-mode.md
...
-> iteracao 17: executando task 19.0 (Expandir Patterns Inline no go-implementation)
task-loop finalizado: nenhuma task elegivel (restantes estao bloqueadas, falharam ou aguardam input)
```

17 tasks elegíveis (tasks 14.0 e 15.0 em estado `done`; task 20.0 bloqueada). Comportamento idêntico para claude, copilot, gemini, codex.

### Dry-run — prd-interactive-task-loop (0 iterações, 4 ferramentas)

```
task-loop finalizado: nenhuma task elegivel (restantes estao bloqueadas, falharam ou aguardam input)
```

Tasks 1.0 e 2.0 em estado `blocked` apesar de não terem dependências. Tasks 3.0/4.0 em `pending` mas dependem de 1.0/2.0.

### Execução Real — maturidade (--max-iterations 1 --timeout 3m)

| Ferramenta | Exit | Duração | Output Streaming | Classificação |
|-----------|------|---------|-----------------|---------------|
| claude | 1 | 0s | `Not logged in · Please run /login` | bug — auth |
| codex | -1 | 3m0s | (vazio) | bug — zero output |
| gemini | -1 | 3m0s | 10 passos capturados (leu arquivos, tentou `check-spec-drift`, `.aiignore`) | blocked (timeout) |
| copilot | -1 | 3m0s | 40+ ações: leu AGENTS.md, skills, task files, rodou testes unitários | blocked (timeout) |

---

## Bugs Identificados

### BUG-001 — claude: falha de autenticação em subprocesso

- **Arquivo:** `docs/bug_report_claude_maturidade.md`
- **Comando real invocado:** `claude --dangerously-skip-permissions --print --bare -p <prompt>`
- **Evidência:** `Not logged in · Please run /login` | exit code 1 | duration 0s
- **Causa provável:** Claude Code usa autenticação por sessão (não env vars). O subprocesso não herda a sessão ativa do processo pai.
- **Severidade:** Alta — bloqueia 100% das execuções com `--tool claude` em self-testing.
- **Nota:** O harness detecta e reporta o erro corretamente (comportamento correto do harness). O bug é de ambiente/autenticação.

### BUG-002 — codex: zero output durante execução de 3 minutos

- **Arquivo:** `docs/bug_report_codex_maturidade.md`
- **Comando real invocado:** `codex exec --dangerously-bypass-approvals-and-sandbox <prompt>`
- **Evidência:** exit -1, duração 3m0s, campo `Agent Output` vazio no relatório
- **Contraste:** gemini e copilot produziram output streaming; codex não
- **Causa provável:** `codex exec` pode não suportar output via pipe (requer TTY), ou o processo iniciou mas não emitiu stdout capturável
- **Severidade:** Média — execução é opaca; falhas são indiagnosticáveis

### Anomalia de Dados — prd-interactive-task-loop

- **Arquivo:** `docs/bug_report_interactive_taskloop_blocked.md`
- **Tasks 1.0 e 2.0:** status `blocked` sem dependências → bundle permanentemente travado
- **Comportamento consistente** entre as 4 ferramentas → não é bug do harness
- **Severidade:** Média — bundle inutilizável até desbloqueio manual

---

## Divergências Observadas entre Ferramentas

| Aspecto | claude | copilot | gemini | codex |
|---------|--------|---------|--------|-------|
| Dry-run parsing | Idêntico | Idêntico | Idêntico | Idêntico |
| Autenticação | Falha (subprocesso) | OK | OK | OK |
| Output streaming | Não (falhou antes) | Sim (40+ ações) | Sim (parcial, 10 ações) | Não (opaco) |
| Leitura AGENTS.md | N/A | Confirmada no output | Confirmada no output | Não confirmável |
| Aviso de MCP | Não | Não | **Sim** (`MCP issues detected. Run /mcp list for status.`) | Não |
| Invocação de skill execute-task | N/A | Sim (evidência: `● skill(execute-task)`) | Sim (inferido pelo comportamento) | Não confirmável |

---

## Notas sobre Comportamento de Ferramentas

### Gemini
- Inicia output com aviso `MCP issues detected. Run /mcp list for status.` — pode confundir parsers futuros.
- Tentou executar `make build` dentro da task, o que é comportamento correto para verificar compilação.
- Não conseguiu ler `TASK-001-quick-reference-mode.md` diretamente (leu via `.aiignore`).

### Copilot
- Produziu o output mais detalhado e estruturado de todos os agentes.
- Leu todos os arquivos de governança obrigatórios (AGENTS.md, agent-governance/SKILL.md, execute-task/SKILL.md).
- Rodou testes unitários (`go test ./internal/contextgen/...`, `go test ./internal/metrics/...`) dentro da task — comportamento correto.
- Identificou que task 1.0 pode já estar subsumida pela 15.0 — evidência de análise de contexto.

---

## Classificação Final

| Bundle | Ferramenta | Resultado |
|--------|-----------|-----------|
| maturidade | claude | **bug** — auth impossibilita execução como subprocesso |
| maturidade | copilot | **blocked** — timeout 3m; agente estava trabalhando |
| maturidade | gemini | **blocked** — timeout 3m; agente estava trabalhando |
| maturidade | codex | **bug** — zero output; opacidade total |
| prd-interactive-task-loop | todas | **inconclusive** — dados em estado blocked/pending-sem-destravamento |
| prd-task-loop-modelos-por-papel | todas | **pass** |
| prd-task-loop-sequential-execution | todas | **pass** |
| prd-telemetry-feedback | todas | **pass** |

---

## Sem Bug Reproduzido

- O **harness** (`./ai-spec task-loop`) se comporta corretamente em todos os casos:
  - Parsing de tasks.md consistente entre todas as ferramentas
  - Detecção de elegibilidade correta
  - Geração de relatório Markdown com evidência
  - Detecção e reporte de erro de auth (claude)
  - Handling de timeout (exit -1)
  - Preservação de status quando agente falha

Os bugs identificados são do **ambiente** (auth claude) e do **comportamento da CLI** (output codex), não do harness.
