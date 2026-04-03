# Tarefa 8.0: Engine de Execução

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar o engine de execução que orquestra o workflow completo: resolução de template → execução de provider → processamento de output → persistência de estado → coordenação HITL. Este é o componente central que integra todos os anteriores.

<requirements>
- Interface Engine conforme tech spec: Run(), Continue()
- Execução sequencial e determinística dos steps (F3.1)
- Contexto isolado por step (F3.2)
- Resolução de template antes de enviar ao provider (F3.3)
- Persistência de estado após cada step (F3.4)
- Retomada a partir do último step pendente (F3.5)
- Retry automático para JSON inválido (máx 2 retries — F5.6)
- Coordenação com HITL entre steps
- Provider não controla fluxo (F3.7)
- Logging estruturado com slog
</requirements>

## Subtarefas

- [ ] 8.1 Definir interface Engine em `internal/runtime/engine.go`
- [ ] 8.2 Implementar DefaultEngine que compõe: workflow parser, template resolver, provider factory, output processor, state store, HITL prompter
- [ ] 8.3 Implementar fluxo Run(): carregar workflow → validar → criar Run → executar steps sequencialmente
- [ ] 8.4 Implementar fluxo de step: resolver template → executar provider → processar output → HITL → persistir
- [ ] 8.5 Implementar fluxo Continue(): carregar run pendente → retomar do último step não completado
- [ ] 8.6 Implementar retry automático para JSON inválido (máx 2 tentativas, depois HITL)
- [ ] 8.7 Implementar tratamento de ações HITL: approve (avança), edit (substitui output), redo (re-executa), exit (pausa e persiste)
- [ ] 8.8 Implementar logging estruturado com slog (run_id, workflow, step, provider, duration_ms)
- [ ] 8.9 Testes de integração: happy path completo com doubles
- [ ] 8.10 Testes: pause/continue
- [ ] 8.11 Testes: retry de JSON inválido
- [ ] 8.12 Testes: edit e redo via HITL
- [ ] 8.13 Testes: falha de provider

## Detalhes de Implementação

Referir seções "Interfaces Chave" (Engine), "Fluxo de dados principal" e "Decisões Chave" em `techspec.md`.

- Engine recebe todas dependências por construtor (composition root fará wiring)
- Fluxo de step conforme Template Method por composição: resolve → execute → process → persist → HITL
- Se output JSON inválido: tentar correção automática → se falhar, retry com provider (máx 2) → se falhar, HITL
- Estado do Run transiciona via métodos do aggregate root (nunca manipulação direta)
- `{{steps.<name>.output}}` resolve para Markdown do output do step (decisão técnica #9)
- Latência entre output do provider e prompt HITL deve ser < 2s (timing com slog)

## Critérios de Sucesso

- Pipeline completo executa end-to-end com provider fake
- Pause/continue funciona corretamente
- Retry automático de JSON inválido funciona até o limite
- Ações HITL (approve, edit, redo, exit) funcionam corretamente
- Estado persiste após cada step
- `go test ./internal/runtime/...` passa

## Testes da Tarefa

- [ ] Teste de integração: happy path (4 steps, todos aprovados)
- [ ] Teste: pause no step 2, continue retoma no step 2
- [ ] Teste: JSON inválido → retry → sucesso
- [ ] Teste: JSON inválido → retry exausto → HITL
- [ ] Teste: ação edit substitui output e avança
- [ ] Teste: ação redo re-executa step
- [ ] Teste: falha de provider → mensagem clara
- [ ] Teste: estado persistido após cada step

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO DONE</critical>

## Arquivos Relevantes
- `internal/runtime/engine.go`
- `internal/runtime/domain/` (dependência)
- `internal/workflows/` (dependência)
- `internal/providers/` (dependência)
- `internal/output/` (dependência)
- `internal/state/` (dependência)
- `internal/hitl/` (dependência)
- `internal/runtime/*_test.go`
