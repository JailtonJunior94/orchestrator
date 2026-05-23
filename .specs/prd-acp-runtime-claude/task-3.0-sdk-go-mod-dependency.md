# Tarefa 3.0: SDK Dependency (go.mod)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Adicionar a dependência `github.com/coder/acp-go-sdk` ao `go.mod` na **última versão estável com tag semântica** publicada no momento da execução, sem pseudo-version de commit e sem `replace`. Rodar `go mod tidy`, sincronizar a constante `ClaudeSDKVersion` em `internal/runtime/specs/claude.go` (entregue na task 2.0) via `scripts/sync-acp-sdk-version.sh`, e registrar a primeira entrada de auditoria em `audit/skill-upgrade-decisions.md` (ou arquivo equivalente) seguindo `.specs/templates/skill-upgrade-decision.md`.

Esta task é deliberadamente pequena e isolada para que a mudança de dependência seja revisável separadamente das tasks que **usam** o SDK (4.0 e 8.0).

<requirements>
- Versão pinada exata (`vX.Y.Z`) no `go.mod`; sem `latest`, sem pseudo-version.
- `go mod tidy` produz `go.sum` consistente.
- `scripts/sync-acp-sdk-version.sh` (da task 2.0) atualiza `ClaudeSDKVersion` para a versão recém-adicionada.
- Entrada de auditoria criada com motivador, critério de aceitação e data.
- Build do projeto continua passando (`make verify`) mesmo sem nenhum código usar o SDK ainda.
</requirements>

## Subtarefas

- [ ] 3.1 Pesquisar a última versão estável tagged de `github.com/coder/acp-go-sdk` (`go list -m -versions github.com/coder/acp-go-sdk` ou GitHub releases). Registrar a versão escolhida e justificativa em commit.
- [ ] 3.2 Executar `go get github.com/coder/acp-go-sdk@vX.Y.Z` para pinar a versão exata.
- [ ] 3.3 Executar `go mod tidy` e validar `go.sum`.
- [ ] 3.4 Executar `scripts/sync-acp-sdk-version.sh` (task 2.0) para sincronizar `ClaudeSDKVersion` em `internal/runtime/specs/claude.go`.
- [ ] 3.5 Criar entrada em `audit/skill-upgrade-decisions.md` (ou `audit/2026-05-XX-acp-go-sdk-introduction.md` conforme convenção do repo) usando `.specs/templates/skill-upgrade-decision.md` como base. Campos obrigatórios: skill=`coder/acp-go-sdk`, versão anterior=`-`, versão nova=`vX.Y.Z`, motivador=`PRD acp-runtime-claude / ADR-009`, critério de aceitação=`make verify passa; RF-04 atendido nas tasks subsequentes`, data.
- [ ] 3.6 Validar com `go build ./...` que o projeto continua compilando.

## Detalhes de Implementação

Ver:
- `techspec.md` §"Pontos de Integração" → "github.com/coder/acp-go-sdk"
- `prd.md` §"Restrições Técnicas de Alto Nível" → "Dependência nova obrigatória"
- `.specs/adr/009-acp-protocol-adoption.md` §"Plano de Implementação" item 5
- `AGENTS.md` §"Upgrades de Skills Externas" para o formato de entrada de auditoria

## Critérios de Sucesso

- `go.mod` tem a linha `require github.com/coder/acp-go-sdk vX.Y.Z` (X.Y.Z = versão exata, sem hash).
- `go.sum` é consistente.
- `internal/runtime/specs/claude.go` tem `ClaudeSDKVersion` igual à versão do `go.mod`.
- `audit/...` tem entrada com motivador, critério de aceitação, data.
- `make verify` passa.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] `go build ./...` sem erros
- [ ] `go mod verify` sem warnings
- [ ] `go vet ./...` sem warnings
- [ ] Teste manual: alterar `go.mod` para versão diferente, rodar `scripts/sync-acp-sdk-version.sh`, conferir que `ClaudeSDKVersion` atualiza; reverter para a versão pinada.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `go.mod` (modificado)
- `go.sum` (modificado por `go mod tidy`)
- `internal/runtime/specs/claude.go` (modificado: constante `ClaudeSDKVersion` sincronizada)
- `audit/...` (novo arquivo de auditoria)

## Validações (DoD)

- [ ] `make verify` executado com sucesso e capturado em `evidence/task-3.0/execution_report.md`
- [ ] `go list -m github.com/coder/acp-go-sdk` retorna exatamente a versão pinada
- [ ] `git diff go.mod go.sum` contém apenas a adição da dependência (sem efeitos colaterais)
- [ ] Entrada de auditoria existe e segue o template
- [ ] Commit semântico `chore(deps): pin github.com/coder/acp-go-sdk vX.Y.Z (ADR-009)`
