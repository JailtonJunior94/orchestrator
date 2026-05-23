# Tarefa 1.0: config.Resolver — hierarquia global+projeto, upward-walk, precedência

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Introduzir uma camada de configuração em cascata mantendo `config.Runtime` como modelo único,
resolvida por um novo `config.Resolver`: built-in < global (`~/.aispec/config.yaml`) < projeto
(arquivo mais próximo via upward-walk) < overrides explícitos (flags). `LoadRuntime` passa a ser um
wrapper fino sobre o Resolver, preservando comportamento atual. Conforme
[ADR-016](adr-016-config-hierarquico-universal.md).

<requirements>
- Resolver com merge campo a campo (cada fonte só sobrescreve campos não-zero).
- Global resolvido via `os.UserHomeDir` (nunca hardcoded — R-SEC-001); ausência de global é não-fatal.
- Upward-walk a partir do CWD, parando em marcador de projeto (`.git/`, `.aispec/`, `.claude/`, `.agents/`)
  ou no boundary do FS; candidatos de projeto em ordem `.aispec/config.yaml` → `.claude/config.yaml` → `.agents/config.yaml`.
- Precedência determinística: flags > projeto > global > built-in.
- Adicionar chaves operacionais opcionais ao `Runtime` (zero-value => default/F1): `timeout`,
  `max_retries`, `retry_backoff_multiplier`, `concurrent`, `batch_size`, `default_tool`.
- Persistência permanece file-only; `~/.aispec/` guarda apenas configuração (RF-17).
- Regressão zero: sem global e a partir da raiz do repo, resultado byte-idêntico ao `LoadRuntime` legado.
</requirements>

## Subtarefas

- [ ] 1.1 Criar `internal/config/resolver.go` com a interface `Resolver` e implementação concreta.
- [ ] 1.2 Implementar resolução de path global (`os.UserHomeDir` + `.aispec/config.yaml`) com ausência não-fatal.
- [ ] 1.3 Implementar `findProjectConfig(cwd)` por upward-walk com marcadores e ordem de candidatos.
- [ ] 1.4 Implementar merge campo-a-campo reusando/estendendo `applyRuntimeDefaults`.
- [ ] 1.5 Estender `config.Runtime` com as novas chaves YAML opcionais.
- [ ] 1.6 Reescrever `LoadRuntime(repoRoot)` como wrapper sobre o Resolver (compat).
- [ ] 1.7 Testes table-driven + FakeFileSystem (precedência, upward-walk, ausência de global, malformado, regressão legado).

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" (interface `Resolver`, modelo `Runtime` estendido) e
[ADR-016](adr-016-config-hierarquico-universal.md). Reusar `internal/config/runtime.go`
(`DefaultRuntime`, `applyRuntimeDefaults`, `runtimeCandidates`) e `internal/fs` (FakeFileSystem).

## Critérios de Sucesso

- Precedência `flags > projeto > global > built-in` comprovada por teste determinístico.
- Upward-walk encontra config de projeto a partir de subdiretório profundo.
- Sem `$HOME`/global ausente: degrada para projeto+defaults sem erro; arquivo malformado propaga erro descritivo.
- `LoadRuntime` legado: saída idêntica para projetos sem config global (RF-16).
- `make test` e `make lint` verdes; cobertura ≥ 75% no pacote alterado.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários (table-driven + FakeFileSystem): precedência, upward-walk, global ausente, malformado, regressão legado.
- [ ] Testes de integração: não obrigatórios nesta tarefa (resolução coberta por unit + FakeFileSystem).

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/config/resolver.go` (novo)
- `internal/config/runtime.go` (estender `Runtime`, reescrever `LoadRuntime`)
- `internal/config/runtime_test.go` / `internal/config/resolver_test.go`
- `internal/fs/fake.go` (FakeFileSystem em testes)
