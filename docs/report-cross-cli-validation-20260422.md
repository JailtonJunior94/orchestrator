# Relatorio de Validacao Cross-CLI — Task Loop

- **Data:** 2026-04-22
- **Binario:** `./ai-spec` (compilado via `make build`)
- **Versao CLIs:** claude 2.1.104, copilot 1.0.34, gemini 0.38.2, codex 0.122.0

---

## 1. Resumo da Matriz de Dry-Run

| Bundle | claude | copilot | gemini | codex | Obs |
|--------|--------|---------|--------|-------|-----|
| `maturidade` | 18 tasks elegiveis | 18 tasks elegiveis | 18 tasks elegiveis | 18 tasks elegiveis | Paridade total |
| `prd-interactive-task-loop` | 0 elegiveis | 0 elegiveis | 0 elegiveis | 0 elegiveis | Tasks 1.0/2.0 `blocked`, 3.0/4.0 `pending` com dependencias em bloqueadas |
| `prd-task-loop-modelos-por-papel` | 0 elegiveis | 0 elegiveis | 0 elegiveis | 0 elegiveis | 7 `done`, 1 `blocked` (8.0) |
| `prd-task-loop-sequential-execution` | 0 elegiveis | 0 elegiveis | 0 elegiveis | 0 elegiveis | Todas 5 `done` |
| `prd-telemetry-feedback` | 0 elegiveis | 0 elegiveis | 0 elegiveis | 0 elegiveis | Todas 4 `done` |

**Resultado dry-run:** As 4 ferramentas produzem output identico. O harness nao apresenta divergencia de parsing entre CLIs na fase de dry-run.

---

## 2. Execucao Real — Bundle `maturidade` (--max-iterations 1)

Unico bundle com tasks elegiveis. Os outros 4 bundles tinham tasks em estado terminal.

| Ferramenta | Task | Pre-Status | Post-Status | Exit Code | Duracao | Resultado |
|-----------|------|------------|-------------|-----------|---------|-----------|
| **claude** | 1.0 | pending | pending | 1 | 0s | **bug** |
| **copilot** | 1.0 | pending | blocked | 0 | 3m36s | **inconclusive** |
| **codex** | 2.0 | pending | done | 0 | 3m01s | **pass** |
| **gemini** | 1.0 | pending | (em execucao) | — | >15min | **inconclusive** |

### Observacoes

1. **Codex pegou task 2.0** (nao 1.0) porque copilot ja havia alterado task 1.0 para `blocked` no tasks.md antes do codex iniciar o parsing. Isso comprova que o harness le o estado real do arquivo antes de cada iteracao.

2. **Race condition parcial:** copilot e gemini iniciaram concorrentemente na mesma task 1.0. Copilot terminou primeiro e alterou o status. Gemini continuou executando sobre um estado ja alterado. O harness nao implementa lock de task.

---

## 3. Bugs Identificados

### BUG-001: Claude CLI requer autenticacao interativa

- **Ferramenta:** claude
- **Bundle:** maturidade
- **Comando:** `./ai-spec task-loop --tool claude --max-iterations 1 --report-path /tmp/report-claude-maturidade.md tasks/maturidade`
- **Comportamento esperado:** O agente executa a task ou falha com erro informativo
- **Comportamento observado:** Exit code 1 em 0s. Output: `Not logged in · Please run /login`
- **Evidencia:** Relatorio em `/tmp/report-claude-maturidade.md`, linha 66: `Not logged in · Please run /login`
- **Impacto:** Claude e inutilizavel quando invocado como sub-processo de outra sessao Claude Code. Afeta 100% dos cenarios de self-testing.
- **Hipotese de causa raiz:** A CLI Claude Code, quando invocada via `ai-spec task-loop`, roda em processo separado sem herdar credenciais da sessao pai. A autenticacao e per-process e requer `/login` interativo.
- **Severidade:** alta (bloqueante para self-testing)
- **Proxima acao:** Documentar como restricao conhecida. Avaliar se `claude --api-key` ou variavel de ambiente `ANTHROPIC_API_KEY` permite execucao nao-interativa.

### BUG-002: Copilot marca task como `blocked` por falha de spec-drift

- **Ferramenta:** copilot
- **Bundle:** maturidade
- **Comando:** `./ai-spec task-loop --tool copilot --max-iterations 1 --report-path /tmp/report-copilot-maturidade.md tasks/maturidade`
- **Comportamento esperado:** Copilot executa a task ou relata falha com causa especifica
- **Comportamento observado:** Exit code 0, duracao 3m36s. Task 1.0 alterada para `blocked`. O agente executou `check-spec-drift` e encontrou divergencia, interpretando como bloqueio.
- **Evidencia:** Relatorio em `/tmp/report-copilot-maturidade.md`. Output do agente mostra: `check-spec-drift` falhou, agente decidiu marcar como `blocked`.
- **Impacto:** medio. A decisao de marcar como `blocked` e do agente (copilot), nao do harness. O harness registrou corretamente a transicao. Porem, a interpretacao do copilot sobre `spec-drift` como bloqueio e discutivel — outros agentes poderiam tratar como warning.
- **Hipotese de causa raiz:** O bundle `maturidade` pode ter hashes de spec desatualizados. O copilot seguiu o `execute-task` SKILL.md que instrui pausar em caso de spec-drift, mas interpretou de forma conservadora.
- **Severidade:** media (comportamento do agente, nao do harness)
- **Proxima acao:** Verificar se hashes de spec em `tasks/maturidade/tasks.md` estao atualizados. Se spec-drift for esperado, considerar tornar o gate configurable (soft warning vs hard block).

### BUG-003: Gemini excede timeout razoavel sem output de progresso

- **Ferramenta:** gemini
- **Bundle:** maturidade
- **Comando:** `./ai-spec task-loop --tool gemini --max-iterations 1 --report-path /tmp/report-gemini-maturidade.md tasks/maturidade`
- **Comportamento esperado:** Execucao completa em tempo razoavel (<10min) ou falha com timeout explicito
- **Comportamento observado:** Gemini rodou por >15min na task 1.0 sem produzir output de progresso no relatorio. O processo node permaneceu ativo com CPU ~0%.
- **Evidencia:** Output do harness parou em `-> iteracao 1: executando task 1.0`. Processo node (PID 37895) ativo com 5s de CPU em 15+ minutos.
- **Impacto:** medio. A ausencia de streaming de progresso torna impossivel diagnosticar se o gemini esta trabalhando, travado, ou aguardando API.
- **Hipotese de causa raiz:** Gemini CLI pode nao suportar streaming de output para stdout quando invocado via sub-processo. Tambem pode estar executando implementacao real da task (leitura de multiplos arquivos, implementacao de codigo) de forma lenta mas funcional.
- **Severidade:** media (UX/observabilidade, nao funcionalidade)
- **Proxima acao:** Aguardar conclusao e verificar resultado. Avaliar se `gemini` suporta flag de verbosidade que emita progresso para stdout.

---

## 4. Divergencias Observadas entre Ferramentas

| Aspecto | claude | copilot | gemini | codex |
|---------|--------|---------|--------|-------|
| **Dry-run parsing** | identico | identico | identico | identico |
| **Autenticacao** | bloqueado (sub-processo) | OK | OK | OK |
| **Tempo de execucao** | 0s (falhou) | 3m36s | >15min (em andamento) | 3m01s |
| **Interpretacao spec-drift** | N/A | conservadora (blocked) | em andamento | permissiva (ignorou, completou) |
| **Output de progresso** | mensagem de erro unica | trace detalhado de acoes | sem output | output minimo |
| **Resultado funcional** | falha | blocked | inconclusive | pass |

---

## 5. Classificacao Final

| Bundle | Ferramenta | Classificacao |
|--------|-----------|---------------|
| maturidade | claude | **blocked** — auth impossibilita execucao como sub-processo |
| maturidade | copilot | **inconclusive** — task marcada `blocked` por spec-drift, comportamento do agente |
| maturidade | codex | **pass** — task 2.0 executada e completada com sucesso |
| maturidade | gemini | **inconclusive** — exit 0 em 16m35s, status inalterado, MCP issues no inicio |
| prd-interactive-task-loop | todas | **pass** (dry-run) — nenhuma task elegivel, comportamento correto |
| prd-task-loop-modelos-por-papel | todas | **pass** (dry-run) — todas terminais, comportamento correto |
| prd-task-loop-sequential-execution | todas | **pass** (dry-run) — todas done, comportamento correto |
| prd-telemetry-feedback | todas | **pass** (dry-run) — todas done, comportamento correto |

---

## 6. Riscos Residuais

1. **Lock de task ausente:** O harness nao implementa mutual exclusion sobre tasks. Execucoes concorrentes de ferramentas diferentes podem competir pela mesma task. Observado: copilot e gemini rodaram task 1.0 simultaneamente.

2. **Bundles sem tasks elegiveis:** 4 dos 5 bundles nao tinham tasks para executar. A validacao real ficou limitada ao bundle `maturidade`. Para cobertura completa, seria necessario resetar tasks ou usar bundles dedicados a teste.

3. **Self-testing com Claude:** Testar o harness com `--tool claude` de dentro de uma sessao Claude Code nao funciona. Esse cenario precisa de documentacao ou workaround.

---

## 7. Conclusao

O **harness** (`./ai-spec task-loop`) funciona corretamente em todos os cenarios testados:
- Dry-run produz output identico para as 4 CLIs
- Parsing de tasks.md e deteccao de elegibilidade sao consistentes
- Relatorios sao gerados com estrutura e evidencia
- Transicoes de status sao registradas fielmente

As divergencias observadas sao **do comportamento dos agentes** (auth, interpretacao de spec-drift, velocidade de execucao), nao do harness em si.

**Veredicto do harness:** pass
**Veredicto cross-CLI:** parcial — 1 pass (codex), 1 blocked (claude/auth), 2 inconclusivos (copilot/spec-drift, gemini/tempo)

---

## 8. Correcoes Aplicadas

### BUG-001: Deteccao de erro de autenticacao + terminacao antecipada

**Arquivos alterados:**
- `internal/taskloop/agent.go` — funcoes `isAuthError()`, `authGuidance()`, interface `LiveOutputSetter`
- `internal/taskloop/taskloop.go` — deteccao de auth error no loop com `break` e stop reason especifico
- `internal/taskloop/agent_test.go` — testes para `isAuthError`, `authGuidance`, `LiveOutputSetter`
- `internal/taskloop/taskloop_test.go` — `TestExecuteAuthErrorEarlyTermination`, `TestExecuteNonAuthErrorContinuesLoop`

**Comportamento:**
- Quando agente retorna exit != 0 com padrao de auth no output, o loop e encerrado imediatamente
- Stop reason: `"abortado: <tool> nao esta autenticado"`
- Note da iteracao: orientacao especifica por ferramenta (ex: "execute 'claude' em terminal separado e faca login com '/login'")
- Erros nao-auth continuam com o comportamento anterior (skip e continua)

### BUG-002: Hashes de spec ausentes no bundle maturidade

**Arquivos alterados:**
- `tasks/maturidade/tasks.md` — adicionados comentarios `<!-- spec-hash-prd: ... -->` e `<!-- spec-hash-techspec: ... -->`

**Comportamento:**
- `check-spec-drift` agora encontra hashes validos no tasks.md do bundle maturidade
- Elimina falso-positivo de spec-drift que levava copilot a marcar tasks como `blocked`

### BUG-003: Streaming de output do agente para o terminal

**Arquivos alterados:**
- `internal/taskloop/agent.go` — campo `liveOut io.Writer` e `SetLiveOutput()` em todos os invokers
- `internal/taskloop/agent_unix.go` — `runCmd` aceita `liveOut io.Writer`, usa `io.MultiWriter` quando non-nil
- `internal/taskloop/agent_windows.go` — paridade com unix
- `internal/taskloop/taskloop.go` — configura `LiveOutputSetter` com `os.Stderr` antes do loop; campo `liveOutOverride` no Service para testes

**Comportamento:**
- Stdout do agente e transmitido em tempo real para stderr enquanto e capturado no buffer para o relatorio
- O usuario ve progresso do agente durante a execucao (resolve gemini sem output por 16+ minutos)
- Em dry-run, streaming nao e configurado (agente nao e invocado)
- Em testes, `liveOutOverride` permite injecao de `io.Discard`

### Validacao

- `go build ./...` — compila sem erros
- `go vet ./internal/taskloop/...` — sem warnings
- `go test ./internal/taskloop/... -count=1` — 100% pass (incluindo 5 novos testes)
