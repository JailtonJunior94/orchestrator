# Tarefa 8.0: Validar runtime.model contra CompatibilityTable

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Adicionar validação cruzada `runtime.model × runtime.ide` contra `internal/taskloop/compatibility.go:CompatibilityTable` quando ambos os campos estão declarados em `AGENT.md`. Modelos incompatíveis produzem erro acionável, exceto quando `--allow-unknown-model` está ativo (paridade com flag existente — RF-09, suposição A5).

<requirements>
- Validação acontece no fluxo de `ResolveProfileFromAgent` ou em fase adjacente (decisão de design documentada).
- Acessar `CompatibilityTable` via construtor existente (`NewCompatibilityTable`).
- Erro cita o `runtime.ide` e os modelos válidos esperados.
- Flag `--allow-unknown-model` bypassa a validação (mesma semântica de uso atual com `--tool`/`--model`).
- Quando `runtime.model` é vazio, validação é skip (default do CLI subjacente é usado).
</requirements>

## Subtarefas

- [ ] 8.1 Adicionar `ValidateModelForIDE(ide, model string, allowUnknown bool) error` em `internal/agents/` ou em `internal/taskloop/`.
- [ ] 8.2 Acoplar validação ao caminho de `ResolveProfileFromAgent` (ou equivalente da tarefa 6.0).
- [ ] 8.3 Propagar flag `--allow-unknown-model` do CLI (já existente) até o ponto de validação.
- [ ] 8.4 Adicionar teste T-13 com casos positivo (modelo compatível), negativo (incompatível) e bypass (`--allow-unknown-model`).

## Detalhes de Implementação

Ver techspec, seção **Pontos de Integração** (`internal/taskloop/compatibility.go`) e PRD RF-09.

Referência de uso atual da tabela: `internal/taskloop/compatibility.go:NewCompatibilityTable()` retorna `CompatibilityTable`; método `IsSupported(tool, model) bool`.

Quando colocar a validação:
- Opção A (preferida): dentro de `ResolveProfileFromAgent` no taskloop, próxima ao mapping `runtime.ide → specs`.
- Opção B: em `internal/agents/registry.go:Resolve` — rejeitada porque acopla `agents` ao taskloop.

## Critérios de Sucesso

- T-13: modelo incompatível com IDE → erro acionável citando modelos válidos.
- T-13 bypass: `--allow-unknown-model` permite combinação (paridade com flag existente).
- Modelo vazio (`runtime.model: ""`) → sem validação, sem erro.
- Diff zero em `internal/runtime/persistence/*` e `internal/runtime/watchdog.go`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-13 positivo: claude + claude-opus-4-7 → aceito.
- [ ] T-13 negativo: claude + gpt-5.4 → erro citando modelos válidos.
- [ ] T-13 bypass: claude + modelo-novo + `--allow-unknown-model` → aceito.
- [ ] Modelo vazio → sem validação.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/taskloop/profile.go` (modificado — chamada à validação)
- `internal/agents/precedence.go` ou novo `internal/agents/compatibility.go` (a definir na implementação)
- `internal/taskloop/compatibility.go` (referência; sem modificação esperada)
- `cmd/ai_spec_harness/task_loop.go` (modificado — propagação de flag)
