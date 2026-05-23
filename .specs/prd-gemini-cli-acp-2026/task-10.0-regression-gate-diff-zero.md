# Tarefa 10.0: Regression gate — diff-zero checks + suite completa Claude/Codex/Copilot verde

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Gate final antes de merge do PR de F0..F5-Gemini. Valida que (a) **diff zero** em módulos protegidos listados em RF-32 (`internal/runtime/specs/spec.go`, `client/client.go`, `persistence/`, `watchdog.go`, `mcpserver/`, `hooks/`, `memory/store.go`, `runner_autoreview.go`); (b) suíte completa de testes de Claude, Codex e Copilot (`go test ./internal/runtime/specs/{claude,codex,copilot}_test.go`, `acp_integration_test.go`, etc.) permanece 100% verde; (c) `golangci-lint run ./...` sem regressão; (d) `go test ./...` end-to-end verde.

Esta task **não escreve código** — apenas valida. Se qualquer check falhar, marca como `failed` e cria tasks de correção apontando o módulo divergente.

<requirements>
- Diff zero validado via `git diff --stat <arquivo>` retornando exatamente 0 linhas para cada módulo protegido em RF-32.
- Suite completa `go test ./...` verde (não apenas testes Gemini específicos).
- `golangci-lint run ./...` sem warnings novos comparados ao baseline da branch base.
- Regressão para Claude/Codex/Copilot validada via execução isolada de suites por driver.
- Smoke test integration suite (`go test -tags integration ./tests/integration/...`) verde quando dependências externas disponíveis; skip limpo quando ausentes.
- Cobertura de testes não decresce significativamente (`go test -coverprofile=cover.out ./...` — diff aceitável: -2% global).
- Cross-PRD validation: se F2/F3/F5-Claude foram pré-requisitos, validar que continuam funcionais (não regredidos por F2/F3/F5-Gemini).
</requirements>

## Subtarefas

- [ ] 10.1 Validar que 9.0 está `done`.
- [ ] 10.2 Executar `git diff --stat` para cada arquivo em lista RF-32:
  - `internal/runtime/specs/spec.go`
  - `internal/runtime/specs/claude.go`, `codex.go`, `copilot.go`
  - `internal/runtime/client/client.go`
  - `internal/runtime/persistence/` (toda a árvore)
  - `internal/runtime/watchdog.go`
  - `internal/runtime/mcpserver/` (toda a árvore)
  - `internal/runtime/hooks/dispatcher.go`, `governance.go`, demais hooks
  - `internal/runtime/memory/store.go`
  - `internal/runtime/runner_autoreview.go`
  - `internal/specdrift/specdrift.go`
  - `internal/agents/registry.go`
- [ ] 10.3 Para cada arquivo com diff > 0: ABORTAR com `failed` e criar task remediation `task-10.1-fix-diff-<modulo>.md`.
- [ ] 10.4 Executar `go test ./internal/runtime/specs/claude_test.go ./internal/runtime/specs/codex_test.go ./internal/runtime/specs/copilot_test.go` e validar 100% verde.
- [ ] 10.5 Executar `go test -run TestACPIntegration_Claude ./internal/runtime/...` e validar 100% verde.
- [ ] 10.6 Executar `go test -run TestACPIntegration_Codex ./internal/runtime/...` e validar 100% verde.
- [ ] 10.7 Executar `go test -run TestACPIntegration_Copilot ./internal/runtime/...` e validar 100% verde.
- [ ] 10.8 Executar `go test ./...` end-to-end e validar 100% verde.
- [ ] 10.9 Executar `golangci-lint run ./...` e validar sem warnings novos.
- [ ] 10.10 (Opcional, se integration tests disponíveis) Executar `go test -tags integration ./tests/integration/...` validando suítes Claude/Codex/Copilot/Gemini.
- [ ] 10.11 Gerar relatório consolidado em `_validation_report.md` (na pasta da PRD) com tabela de resultados de cada validação.

## Detalhes de Implementação

Ver techspec.md:
- §"Considerações Técnicas / Arquivos Relevantes / Inalterados (diff zero — RF-32)" — lista exata de arquivos a validar.
- §"Abordagem de Testes / Regressão Obrigatória" — descrição dos checks.
- §"Mapeamento RF → Componente → Teste" — RF-30, RF-32.

Esta task é o equivalente do "gate de aceitação" antes do PR seguir para review humana.

## Critérios de Sucesso

- `git diff --stat <cada-arquivo-RF-32>` retorna **zero linhas** para todos os módulos protegidos.
- `go test ./...` retorna `ok` em todos os pacotes; nenhum `FAIL` ou `panic`.
- `golangci-lint run ./...` retorna `0 issues` ou apenas issues pré-existentes (não introduzidas por F0..F5-Gemini).
- Suítes Claude/Codex/Copilot rodam isoladas com 100% verde.
- Smoke test Gemini (integration tag) verde quando `gemini` disponível.
- `_validation_report.md` gerado na pasta `.specs/prd-gemini-cli-acp-2026/` com tabela:

```markdown
| Check | Resultado | Notas |
|---|---|---|
| diff zero — specs/spec.go | ✅ 0 linhas | — |
| diff zero — client/client.go | ✅ 0 linhas | — |
| ... | ... | ... |
| go test claude suite | ✅ 100% verde | 47 testes |
| go test codex suite | ✅ 100% verde | 52 testes |
| go test copilot suite | ✅ 100% verde | 38 testes |
| go test gemini suite | ✅ 100% verde | 35 testes (1.0+4.0) |
| go test ./... | ✅ 100% verde | 312 testes total |
| golangci-lint | ✅ 0 issues novos | baseline preservado |
```

### Definition of Done

1. Todos os checks de diff-zero passam (zero linhas em N módulos).
2. `go test ./...` 100% verde end-to-end.
3. `golangci-lint run ./...` sem regressão.
4. Suítes Claude/Codex/Copilot isoladas 100% verdes (RF-30).
5. `_validation_report.md` gerado e revisado.
6. Cross-PRD F2/F3/F5-Claude continuam funcionais (testes verdes).
7. Task 9.0 confirmada `done` antes do início.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] `git diff --stat` para cada módulo RF-32 → 0 linhas
- [ ] `go test ./internal/runtime/specs/{claude,codex,copilot}_test.go` → 100% verde
- [ ] `go test -run TestACPIntegration_{Claude,Codex,Copilot} ./internal/runtime/...` → 100% verde
- [ ] `go test ./...` end-to-end → 100% verde
- [ ] `golangci-lint run ./...` → sem warnings novos
- [ ] `_validation_report.md` gerado com tabela consolidada
- [ ] (Opcional) `go test -tags integration ./tests/integration/...` → verde quando dependências disponíveis

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **NOVO**: `.specs/prd-gemini-cli-acp-2026/_validation_report.md` (relatório consolidado)
- **REFERÊNCIA (não modificar)**: todos os arquivos da lista RF-32 (diff-zero validation)
