# Tarefa 3.0: Estender Job com ReasoningEffort/AccessMode/AddDirs

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Estender o value object `runtime.Job` (`internal/runtime/runner.go`) com três campos opcionais para suportar parâmetros Codex-específicos quando consumidos por `Spec.BootstrapArgs(...)`:

- `ReasoningEffort string` — `"low"`, `"medium"`, `"high"`; default `""` (que `codexBootstrapArgs` traduz para omitir o flag).
- `AccessMode specs.AccessMode` — `AccessModeRestricted` (default) ou `AccessModeFull`.
- `AddDirs []string` — diretórios adicionais para Codex (`SupportsAddDirs=true` no compozy). Default `nil`.

**Defaults preservam comportamento atual** para Claude/Copilot: quando `BootstrapArgs(...)` no-op retorna `nil` (tarefa 1.0), os valores em `Job` são ignorados.

Paralelizável com tarefa 2.0 (`specs/codex.go`) — arquivos disjuntos (`runner.go::Job` vs `specs/codex.go`), ambas dependem apenas de 1.0.

<requirements>
- 3 campos novos em Job (`ReasoningEffort`, `AccessMode`, `AddDirs`).
- Tipo `AccessMode` importado de `specs` (não duplicar definição).
- Defaults: "", AccessModeRestricted, nil.
- Job continua sendo value object (sem ponteiros, sem mutação após criação).
- Construtores de Job (se existentes via fixtures) atualizados para zero-init dos novos campos.
- Diff zero em internal/runtime/persistence/, watchdog.go, client/, events/.
- Testes de regressão `runner_test.go` permanecem 100% verdes sem alteração de comportamento.
</requirements>

## Subtarefas

- [ ] 3.1 Adicionar `ReasoningEffort string` ao struct `Job` em `internal/runtime/runner.go`.
- [ ] 3.2 Adicionar `AccessMode specs.AccessMode` ao struct `Job` (importar `specs`).
- [ ] 3.3 Adicionar `AddDirs []string` ao struct `Job`.
- [ ] 3.4 Adicionar comentários go-doc em cada novo campo documentando defaults e quem consome (`Codex.BootstrapArgs`).
- [ ] 3.5 Verificar que zero-value de Job é válido para Claude/Copilot: `Job{}` produz `BootstrapArgs(nil)` resultado `nil` (no-op).
- [ ] 3.6 Rodar `go test ./internal/runtime/...` → 100% verde (regressão).
- [ ] 3.7 Confirmar diff zero em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/`, `internal/runtime/events/`.

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → bloco `internal/runtime/runner.go — Job estendido` e §"Sequenciamento de Desenvolvimento" → item 3. Decisão registrada em ADR-013 D-02 (campos opcionais preservam Claude/Copilot).

Anti-padrão: NÃO alterar a assinatura de construtores existentes de `Job` se forem usados via literal — preservar comportamento de zero-value.

## Critérios de Sucesso

- `internal/runtime/runner.go::Job` ganha 3 campos novos.
- Zero-value `Job{}` continua válido (Claude/Copilot fluem inalterados).
- Importação de `specs.AccessMode` em `runner.go` sem ciclo de dependência.
- `runner_test.go` 100% verde sem alteração de teste.
- Diff zero em módulos invariantes.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-19 (regressão Claude/Copilot): suíte de `runner_test.go` 100% verde com defaults dos novos campos.
- [ ] Teste novo: `Job{}.AccessMode == specs.AccessModeRestricted` (ou string zero-value que mapeia para restricted no consumo).
- [ ] Teste novo: `Job{ReasoningEffort: "high", AccessMode: specs.AccessModeFull, AddDirs: []string{"/x"}}` constrói corretamente.
- [ ] `go vet ./internal/runtime/...` sem warnings.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] `Job` em `runner.go` tem campos `ReasoningEffort string`, `AccessMode specs.AccessMode`, `AddDirs []string` com go-doc.
- [ ] Import de `specs` em `runner.go` sem ciclo (verificar via `go build ./...`).
- [ ] `Job{}` zero-value compila e flui em testes Claude/Copilot sem alteração de saída.
- [ ] **Atenção**: zero-value de `specs.AccessMode` (string) é `""`, não `"restricted"`. Tarefa 4.0 deve tratar `""` como equivalente a `AccessModeRestricted` no consumo OU este task adiciona helper `Job.EffectiveAccessMode() AccessMode` retornando default `AccessModeRestricted` quando vazio. Decisão registrada na implementação.
- [ ] `go test ./internal/runtime/...` → 100% verde.
- [ ] `go vet ./...` → sem warnings.
- [ ] `git diff --stat internal/runtime/persistence/ internal/runtime/watchdog.go internal/runtime/client/ internal/runtime/events/` → vazio.

## Arquivos Relevantes

- `internal/runtime/runner.go` (modificar: struct Job)
- `internal/runtime/runner_test.go` (validar regressão; adicionar caso de zero-value)
- `internal/runtime/specs/spec.go` (consumir tipo AccessMode)
- ADR-013 §"Decisão" → D-02
- techspec.md §"Design de Implementação" → bloco `runner.go — Job estendido`
