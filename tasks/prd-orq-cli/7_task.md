# Tarefa 7.0: State (Persistência em .orq/)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar persistência de runs, artefatos e state.json no filesystem dentro de `.orq/runs/<id>/`.

<requirements>
- Interface Store conforme tech spec: CreateRun, SaveRun, LoadRun, FindLatestContinuable, SaveArtifact, LoadArtifact
- Implementação FileStore que persiste em `.orq/runs/<run-id>/`
- state.json com schema_version para evolução futura
- Artefatos individuais por step (Markdown + JSON)
- Diretório por run com identificador único para evitar sobrescrita
- Serialização/deserialização de entidades de domínio para JSON
- Testes de integração com t.TempDir()
</requirements>

## Subtarefas

- [ ] 7.1 Definir interface Store em `internal/state/store.go`
- [ ] 7.2 Implementar DTOs de serialização (Run → JSON, JSON → Run) em `internal/state/dto.go`
- [ ] 7.3 Implementar FileStore em `internal/state/file_store.go`: CreateRun, SaveRun, LoadRun com state.json
- [ ] 7.4 Implementar FindLatestContinuable para suportar `orq continue`
- [ ] 7.5 Implementar SaveArtifact e LoadArtifact para artefatos de step (prd.md, techspec.md, etc.)
- [ ] 7.6 Criar estrutura de diretórios `.orq/runs/<id>/` e `.orq/runs/<id>/logs/`
- [ ] 7.7 Testes de integração: salvar e carregar run com t.TempDir()
- [ ] 7.8 Testes de integração: salvar e carregar artefatos
- [ ] 7.9 Testes: FindLatestContinuable com múltiplos runs

## Detalhes de Implementação

Referir seções "Interfaces Chave" (Store), "State JSON Schema" e "Estrutura de Diretório .orq/" em `techspec.md`.

- state.json deve conter: run_id, workflow, input, status, created_at, updated_at, schema_version, steps[]
- Cada step em state.json: name, provider, status, attempts e refs separadas para `raw_output`, `approved_markdown`, `structured_json` e `validation_report`
- Refs de artefatos apontam para arquivos dentro do diretório da run, preservando separação entre raw, aprovado, estruturado e validação
- DTOs convertem entre entidades de domínio (campos não exportados) e representação JSON
- Schema version = 1 para V1, permite evolução sem quebra (F7.6)
- Paths com `filepath.Join` (cross-platform)

## Critérios de Sucesso

- SaveRun cria diretório e state.json válido
- LoadRun reconstrói entidade Run corretamente
- FindLatestContinuable encontra a run compatível mais recente com status pendente/pausado
- Artefatos são salvos e carregados por step
- Múltiplos runs coexistem sem sobrescrita
- `go test ./internal/state/...` passa

## Testes da Tarefa

- [ ] Testes de integração SaveRun + LoadRun roundtrip
- [ ] Testes de integração SaveArtifact + LoadArtifact
- [ ] Testes FindLatestContinuable (nenhum run, um run, múltiplos runs)
- [ ] Teste de schema_version no state.json
- [ ] Teste de estrutura de diretórios criada

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO DONE</critical>

## Arquivos Relevantes
- `internal/state/store.go`
- `internal/state/file_store.go`
- `internal/state/dto.go`
- `internal/state/*_test.go`
