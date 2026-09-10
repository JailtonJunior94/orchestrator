# Tarefa 9.0: Comando `memory` com seis subcomandos, incluindo migração

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Entregar a superfície de operação humana. Ela **não passa pela fachada**: acessa os colaboradores diretamente, porque precisa de granularidade total — promover, arquivar, buscar, compactar — que a porta estreita esconde de propósito. Isso é uso correto de Facade, não violação.

A migração é um dos seis subcomandos e vem na mesma fatia porque compartilha exatamente os mesmos arquivos e o mesmo gate; separá-la imporia duas edições do schema de CLI e duas execuções do gate bidirecional.

<requirements>
- RF-31: `ai-spec memory` com `show`, `search`, `export`, `compact`, `migrate` e `handoff`. **Nenhum comando existente pode ter contrato ou output alterados.**
- RF-32: a inspeção rastreia cada fato até a sessão de origem.
- RF-19: busca por texto e por entidade, scan determinístico, sem rede e sem LLM.
- RF-33: migração somente por comando explícito, preservando 100% do conteúdo, com backup verificável antes de qualquer conversão, e recusa de conteúdo já migrado.
- Molde obrigatório: comando pai sem `RunE` mais subcomandos, seguindo `cmd/ai_spec_harness/telemetry.go`; wiring de `output.New` e `fs.NewOSFileSystem()` seguindo `cmd/ai_spec_harness/skills.go`; código de saída via `newExitError`.
</requirements>

## Subtarefas

- [ ] 9.1 Criar o comando pai e registrá-lo em `cmd/ai_spec_harness/root.go`.
- [ ] 9.2 Implementar `show`: fatos ativos por camada, com origem e contradições sinalizadas.
- [ ] 9.3 Implementar `search` sobre o scan determinístico de 5.0.
- [ ] 9.4 Implementar `export` produzindo artefato autocontido e portátil.
- [ ] 9.5 Implementar `compact` invocando a política determinística de 3.0.
- [ ] 9.6 Implementar `handoff`: estado, reivindicação e liberação do bastão de 4.0.
- [ ] 9.7 Implementar `migrate` com backup verificável e recusa de reaplicação.
- [ ] 9.8 Atualizar `docs/cli-schema.json` com o comando e todos os subcomandos e flags.
- [ ] 9.9 Atualizar `docs/troubleshooting.md`.

## Detalhes de Implementação

Ver techspec.md, seção "Endpoints de API" (tabela de subcomandos e molde). Ver MD-004, seção Decisão, parágrafo sobre migração, e MD-001, passo 6 do Plano de Implementação.

## Critérios de Sucesso

- `cmd/ai_spec_harness/cli_contract_test.go` passa nas duas direções: nada no schema sem implementação, nada implementado sem schema.
- Nenhum comando existente teve contrato ou output alterados.
- `migrate` grava backup verificável **antes** de converter, e recusa conteúdo já migrado com erro tipado.
- `migrate` preserva 100% do conteúdo existente, provado por comparação antes e depois.
- `show` exibe a origem de cada fato e sinaliza contradições.
- `search` funciona sem rede e sem chave de API.
- Erro com código de saída específico usa `newExitError`, não `os.Exit`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `cmd/ai_spec_harness/memory.go` — arquivo novo
- `cmd/ai_spec_harness/root.go` — registro do comando
- `cmd/ai_spec_harness/telemetry.go` — molde de comando pai com subcomandos, apenas leitura
- `cmd/ai_spec_harness/skills.go` — molde de wiring, apenas leitura
- `cmd/ai_spec_harness/exit_error.go` — contrato de código de saída, apenas leitura
- `docs/cli-schema.json` e `cmd/ai_spec_harness/cli_contract_test.go` — gate bidirecional
- `internal/output/output.go` — `Printer`, apenas leitura
- `docs/troubleshooting.md`
