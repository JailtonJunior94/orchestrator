# Tarefa 1.0: Scaffolding e Tooling

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Criar a estrutura base do projeto Go, módulo, task runner, release config e CI. Tudo depende desta tarefa.

<requirements>
- Inicializar módulo Go com `go mod init`
- Criar estrutura de diretórios conforme tech spec
- Configurar Taskfile.yml com tasks: build, test, lint, fmt, generate, release:snapshot
- Configurar .goreleaser.yaml para macOS, Linux e Windows (amd64, arm64)
- Criar entrypoint stub em `cmd/orq/main.go` com version injection via ldflags
- Garantir que `task build` e `task test` funcionam com o stub
</requirements>

## Subtarefas

- [ ] 1.1 Executar `go mod init` com module path adequado e adicionar dependências iniciais (cobra, yaml.v3, uuid)
- [ ] 1.2 Criar estrutura de diretórios: `cmd/orq/`, `internal/cli/`, `internal/bootstrap/`, `internal/workflows/`, `internal/runtime/domain/`, `internal/providers/`, `internal/state/`, `internal/hitl/`, `internal/platform/`, `internal/output/`, `pkg/`
- [ ] 1.3 Criar `cmd/orq/main.go` com root command Cobra e version injection (`version`, `commit`, `date` via ldflags)
- [ ] 1.4 Criar `Taskfile.yml` conforme referência na tech spec
- [ ] 1.5 Criar `.goreleaser.yaml` conforme referência na tech spec
- [ ] 1.6 Verificar que `task build`, `task test` e `task lint` executam sem erro

## Detalhes de Implementação

Referir seções "Sequenciamento de Desenvolvimento" (item 1) e "Taskfile.yml / GoReleaser Config" em `techspec.md`.

- `main.go` deve injetar `version`, `commit` e `date` via ldflags do GoReleaser
- Root command deve ter `Use: "orq"` e description alinhada ao PRD
- Subcomandos serão adicionados em tarefas posteriores

## Critérios de Sucesso

- `go build ./...` compila sem erro
- `go test ./...` passa (mesmo sem testes ainda)
- `task build` gera binário `orq`
- `task test` executa sem erro
- `.goreleaser.yaml` válido (verificável com `goreleaser check`)
- Estrutura de diretórios conforme tech spec

## Testes da Tarefa

- [ ] `go build ./cmd/orq` compila
- [ ] `./orq --version` exibe versão (ou "dev")
- [ ] `task build && task test` executam com sucesso

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO DONE</critical>

## Arquivos Relevantes
- `cmd/orq/main.go`
- `go.mod`, `go.sum`
- `Taskfile.yml`
- `.goreleaser.yaml`
