# Tarefa 8.0: Suíte de paridade RP-03 + gate CI (RG-03) + plano de sunset legacy

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Fechar a entrega validando a paridade de ponta a ponta: promover as invariantes ADR-008 a uma suíte executável que asserta igualdade entre as 4 CLIs (RP-03), torná-la gate de CI obrigatório por CLI (RG-03) e registrar o plano de sunset do legacy mode (RIN-05, sem remoção neste ciclo).

<requirements>
- Suíte `internal/parity` derivada de ADR-008: task fixture executada (via `acpfake`) contra os 4 drivers asserta igualdade do conjunto de `normalized_name` e da forma de evento. Determinística (sem rede real).
- Gate de CI obrigatório por CLI em `.github/workflows/test.yml` (RG-03).
- Telemetria opt-in (RG-04) confirmada inalterada no fluxo de paridade.
- Plano de sunset (ADR-026): reforçar mensagens de depreciação dos entrypoints legados (`copilotInvoker`/`codexInvoker`, `internal/wrapper`) e esboçar guia de migração `legacy → --runtime acp` em `docs/`. **Não remover código legado.**
</requirements>

## Subtarefas

- [ ] 8.1 Suíte `internal/parity` com fixture única e asserção cross-CLI (4 drivers).
- [ ] 8.2 Gate de CI por CLI em `test.yml`.
- [ ] 8.3 Reforçar depreciação dos entrypoints legados (mensagem única apontando `--runtime acp`).
- [ ] 8.4 Guia de migração legacy→ACP em `docs/`.
- [ ] 8.5 Registrar tarefa futura de remoção com o critério de ADR-026.

## Detalhes de Implementação

Ver techspec.md §"Abordagem de Testes" (E2E/Paridade) e §"Sequenciamento". ADRs: [022](adr-022-guard-governanca-runtime-spec-hash.md) (RG-03), [026](adr-026-sunset-legacy-mode.md) (RIN-05). Reusar `internal/parity` e `acpfake` existentes.

## Critérios de Sucesso

- Mesma fixture → mesmo conjunto de `normalized_name` e forma de evento nas 4 CLIs.
- CI falha se a paridade quebrar em qualquer CLI.
- Entrypoints legados emitem aviso claro e único; guia de migração publicado.
- Nenhum código legado removido neste ciclo (escopo ADR-026).
- `make test` + suíte `internal/parity` verdes.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/parity/*` (suíte RP-03)
- `.github/workflows/test.yml` (gate por CLI)
- `internal/taskloop/agent.go` (depreciação `copilotInvoker`/`codexInvoker`)
- `internal/wrapper/wrapper.go` (depreciação)
- `docs/` (guia de migração)
