# Tarefa 9.0: E2E smoke + auditoria de invariantes forenses

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Cenário smoke end-to-end validando integração CLI ↔ taskloop ↔ registry sem ACP real (usa `runnerStub` existente). Auditoria manual + automatizada confirmando que artefatos forenses (`events.jsonl`, `tool_calls.md`, `execution_report.md`) e o `ActivityWatchdog` permanecem byte-idênticos ao fluxo legado quando `--agent` é usado. Este é o gate final de RF-19 e do risco R-05 crítico.

<requirements>
- Teste E2E em diretório temporário cria `AGENT.md` mínimo válido, invoca `task_loop` via `--agent <name>` e verifica:
  - Prompt enriquecido contém os blocos metadata + catálogo na ordem correta.
  - Artefatos forenses são produzidos.
  - Artefatos têm a mesma estrutura/schema que no fluxo `--tool claude` legado.
- Auditoria automatizada: `git diff internal/runtime/persistence/ internal/runtime/watchdog.go` (na branch da feature vs `main`) deve retornar vazio.
- Comparação de baseline: rodar `task_loop --tool claude` (legado) e `task_loop --agent <equivalent>` (novo) sobre mesmo input → artefatos com mesma estrutura.
</requirements>

## Subtarefas

- [ ] 9.1 Criar teste E2E em `internal/taskloop/e2e_agent_test.go` (ou similar) usando `runnerStub`.
- [ ] 9.2 Adicionar verificação automatizada que o prompt enriquecido contém os blocos esperados.
- [ ] 9.3 Executar manualmente `git diff main -- internal/runtime/persistence/ internal/runtime/watchdog.go` e anexar resultado ao execution_report.
- [ ] 9.4 Rodar suíte completa `go test ./...` para garantir zero regressão.
- [ ] 9.5 Executar comparação de baseline: invocar `--tool claude` e `--agent claude-equivalente` com mesmo PRD/task; comparar estruturas de `events.jsonl`, `tool_calls.md`, `execution_report.md`.

## Detalhes de Implementação

Ver techspec, seção **Abordagem de Testes → Testes E2E** e ADR-011 → Riscos R-05 (crítico).

Padrões de `runnerStub`: ver `internal/runtime/runner_test.go` ou equivalente existente para mock de ACPRunner.

Auditoria automatizada (script sugerido em CI/local):
```bash
git fetch origin main
git diff origin/main -- internal/runtime/persistence/ internal/runtime/watchdog.go | wc -l
# Esperado: 0
```

## Critérios de Sucesso

- T-22: artefatos forenses idênticos em estrutura entre fluxo legado e fluxo `--agent`.
- Diff zero em `internal/runtime/persistence/*` e `internal/runtime/watchdog.go` (verificado e documentado no execution_report).
- Suíte completa `go test ./...` verde.
- Teste E2E confirma blocos metadata + catálogo no prompt enriquecido.
- Comparação de baseline documentada no execution_report.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-22: estrutura de artefatos forenses idêntica.
- [ ] Suíte completa `go test ./...` verde.
- [ ] Auditoria de diff zero em persistência forense + watchdog.
- [ ] Smoke E2E com prompt enriquecido validado.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/taskloop/e2e_agent_test.go` (novo)
- `internal/runtime/persistence/*` (verificar — sem modificação esperada)
- `internal/runtime/watchdog.go` (verificar — sem modificação esperada)
