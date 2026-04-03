# Especificação Técnica — Orchestrator CLI

## Resumo Executivo

O Orchestrator é uma CLI em Go exposta ao usuário como binário `orq`, capaz de orquestrar workflows declarativos (YAML) com múltiplos providers (Claude CLI, Copilot CLI) e controle humano (HITL) em cada etapa. A arquitetura segue as regras do repositório: `cmd/orq/` como entrypoint, `internal/bootstrap` como composition root, application services por módulo e domínio isolado de detalhes de terminal, filesystem e subprocess.

A implementação prioriza domínio e engine de execução, seguidos por parser de workflow, adapters de provider, persistência em `.orq/`, camada HITL de terminal e, por fim, comandos Cobra. Build, validação e automação local usam `Taskfile.yml`; distribuição cross-platform usa GoReleaser.

## Arquitetura do Sistema

### Visão Geral dos Componentes

```text
cmd/orq/main.go           → entrypoint, version injection, bootstrap
internal/cli/             → comandos Cobra (run, continue, list), flags, rendering
internal/bootstrap/       → composition root, wiring de dependências
internal/runtime/
  application/            → serviços de run/continue/list
  domain/                 → Run aggregate, StepExecution, estados, regras
internal/workflows/       → parser YAML, validator, template resolver, catálogo embed.FS
internal/providers/       → contratos e adapters Claude CLI / Copilot CLI
internal/state/           → persistência de runs, artefatos e state.json em .orq/
internal/hitl/            → prompt interativo, approve/edit/redo/exit
internal/output/          → parsing de provider output, extração JSON, schema validation
internal/platform/        → subprocess, editor, filesystem, clock, path helpers
pkg/                      → tipos realmente compartilhados entre módulos
```

**Fluxo principal**:

```text
CLI (Cobra) → bootstrap → runtime application service
  → workflow parser + validator
  → engine
    → template resolver
    → provider adapter
    → output processor
    → state store
    → HITL
    → próximo step ou pausa/fim
```

**Direção de dependências**: `cmd/` -> `internal/bootstrap` -> `internal/<module>/application` -> `internal/<module>/domain`, com adapters dependendo dos contratos internos.

## Design de Implementação

### Interfaces Chave

```go
// internal/providers/provider.go
type Provider interface {
    Name() string
    Available(ctx context.Context) error
    Execute(ctx context.Context, input ExecutionRequest) (ExecutionResult, error)
}

type ExecutionRequest struct {
    Prompt           string
    Timeout          time.Duration
    WorkingDirectory string
    Mode             CapabilityMode
}

type ExecutionResult struct {
    Stdout   string
    Stderr   string
    ExitCode int
    Duration time.Duration
    Metadata map[string]string
}
```

```go
// internal/runtime/application/service.go
type Service interface {
    Run(ctx context.Context, workflowName string, input InputSource) (*RunResult, error)
    Continue(ctx context.Context) (*RunResult, error)
    ListWorkflows(ctx context.Context) ([]WorkflowSummary, error)
}
```

```go
// internal/state/store.go
type Store interface {
    CreateRun(ctx context.Context, run *Run) error
    SaveRun(ctx context.Context, run *Run) error
    LoadRun(ctx context.Context, runID string) (*Run, error)
    FindLatestContinuable(ctx context.Context) (*Run, error)
    SaveArtifact(ctx context.Context, runID string, artifact Artifact) error
    LoadArtifact(ctx context.Context, runID string, key ArtifactKey) (*Artifact, error)
}
```

```go
// internal/hitl/prompter.go
type Prompter interface {
    Prompt(ctx context.Context, view StepReviewView) (Action, error)
}

type Action int

const (
    ActionApprove Action = iota
    ActionEdit
    ActionRedo
    ActionExit
)
```

```go
// internal/workflows/parser.go
type Parser interface {
    Parse(ctx context.Context, data []byte) (*Workflow, error)
}

type Validator interface {
    Validate(ctx context.Context, workflow *Workflow) error
}

type TemplateResolver interface {
    Resolve(ctx context.Context, template string, vars TemplateVars) (string, error)
}
```

### Modelos de Dados

#### Entidades de Domínio

```go
// internal/runtime/domain/run.go
type Run struct {
    id        RunID
    workflow  WorkflowName
    input     string
    status    RunStatus
    steps     []StepExecution
    createdAt time.Time
    updatedAt time.Time
}

func (r *Run) Start() error
func (r *Run) ApproveStep(stepName StepName) error
func (r *Run) EditStep(stepName StepName, editedMarkdown string, editedJSON []byte) error
func (r *Run) RetryStep(stepName StepName) error
func (r *Run) Pause() error
func (r *Run) Resume() error
func (r *Run) MarkStepFailed(stepName StepName, err error) error
func (r *Run) MarkStepCompleted(stepName StepName, result StepResult) error
func (r *Run) CurrentStep() (*StepExecution, error)
```

```go
// internal/runtime/domain/step.go
type StepExecution struct {
    name          StepName
    provider      ProviderName
    status        StepStatus
    resolvedInput string
    attempts      int
    result        StepResult
    failure       string
}

type StepResult struct {
    RawOutputRef          ArtifactKey
    ApprovedMarkdownRef   ArtifactKey
    StructuredJSONRef     ArtifactKey
    ValidationReportRef   ArtifactKey
    SchemaName            string
    SchemaVersion         string
    ValidationStatus      ValidationStatus
    EditedByHuman         bool
}
```

#### Value Objects

```go
// internal/runtime/domain/values.go
type RunStatus string

const (
    RunPending   RunStatus = "pending"
    RunRunning   RunStatus = "running"
    RunPaused    RunStatus = "paused"
    RunFailed    RunStatus = "failed"
    RunCompleted RunStatus = "completed"
    RunCancelled RunStatus = "cancelled"
)

type StepStatus string

const (
    StepPending         StepStatus = "pending"
    StepRunning         StepStatus = "running"
    StepWaitingApproval StepStatus = "waiting_approval"
    StepApproved        StepStatus = "approved"
    StepRetrying        StepStatus = "retrying"
    StepFailed          StepStatus = "failed"
    StepSkipped         StepStatus = "skipped"
)

type ValidationStatus string

const (
    ValidationNotApplicable ValidationStatus = "not_applicable"
    ValidationPending       ValidationStatus = "pending"
    ValidationPassed        ValidationStatus = "passed"
    ValidationCorrected     ValidationStatus = "corrected"
    ValidationFailed        ValidationStatus = "failed"
)
```

#### Workflow YAML Schema

```yaml
# workflows/dev-workflow.yaml
name: dev-workflow
steps:
  - name: prd
    provider: claude
    input: |
      Gere um PRD detalhado para: {{input}}
    output:
      markdown: required
      json_schema: prd/v1

  - name: techspec
    provider: claude
    input: |
      Com base no PRD abaixo, gere uma Tech Spec:
      {{steps.prd.output}}
    output:
      markdown: required
      json_schema: techspec/v1

  - name: tasks
    provider: claude
    input: |
      Decomponha em tasks implementáveis:
      PRD: {{steps.prd.output}}
      TechSpec: {{steps.techspec.output}}
    output:
      markdown: required
      json_schema: tasks/v1

  - name: execute
    provider: copilot
    input: |
      A partir das tasks aprovadas abaixo, proponha e execute o plano conforme capabilities do adapter:
      {{steps.tasks.output}}
    capabilities:
      filesystem_execution: optional
    output:
      markdown: required
      json_schema: execute/v1
    schema: |
      {
        "type":"object",
        "required":["summary","commands"],
        "properties":{
          "summary":{"type":"string"},
          "commands":{
            "type":"array",
            "items":{
              "type":"object",
              "required":["executable","args"],
              "properties":{
                "executable":{"type":"string"},
                "args":{"type":"array","items":{"type":"string"}}
              }
            }
          },
          "files":{
            "type":"array",
            "items":{
              "type":"object",
              "required":["path","content"],
              "properties":{
                "path":{"type":"string"},
                "content":{"type":"string"}
              }
            }
          }
        }
      }
```

`{{steps.<name>.output}}` sempre resolve para o Markdown aprovado do step anterior. O JSON estruturado permanece disponível por referência explícita no runtime, não por interpolação textual.

#### State JSON Schema

```json
{
  "run_id": "01JQ1234567890ABCDEFG",
  "workflow": "dev-workflow",
  "input": "criar API de login",
  "status": "paused",
  "schema_version": 1,
  "created_at": "2026-04-03T10:00:00Z",
  "updated_at": "2026-04-03T10:05:00Z",
  "current_step": "techspec",
  "steps": [
    {
      "name": "prd",
      "provider": "claude",
      "status": "approved",
      "attempts": 1,
      "result": {
        "raw_output_ref": "artifacts/prd/raw.md",
        "approved_markdown_ref": "artifacts/prd/approved.md",
        "structured_json_ref": "artifacts/prd/structured.json",
        "validation_report_ref": "artifacts/prd/validation.json",
        "schema_name": "prd/v1",
        "schema_version": "1",
        "validation_status": "passed",
        "edited_by_human": false
      }
    },
    {
      "name": "techspec",
      "provider": "claude",
      "status": "waiting_approval",
      "attempts": 1,
      "result": {
        "raw_output_ref": "artifacts/techspec/raw.md",
        "approved_markdown_ref": "artifacts/techspec/candidate.md",
        "structured_json_ref": "artifacts/techspec/structured.json",
        "validation_report_ref": "artifacts/techspec/validation.json",
        "schema_name": "techspec/v1",
        "schema_version": "1",
        "validation_status": "corrected",
        "edited_by_human": false
      }
    }
  ]
}
```

#### Estrutura de Diretório `.orq/`

```text
.orq/
└── runs/
    └── <run-id>/
        ├── state.json
        ├── artifacts/
        │   ├── prd/
        │   │   ├── raw.md
        │   │   ├── approved.md
        │   │   ├── structured.json
        │   │   └── validation.json
        │   ├── techspec/
        │   ├── tasks/
        │   └── execute/
        └── logs/
            └── run.log
```

O diretório `.orq/` fica no `cwd` onde o comando é executado. Cada run tem subdiretório próprio para evitar sobrescrita e permitir auditoria e retomada.

## Pontos de Integração

### Claude CLI (Provider)

- Adapter de subprocess encapsulado em `internal/providers/claude`.
- O comando efetivo não fica hardcoded na spec; ele é resolvido por uma configuração do adapter com smoke test de compatibilidade por versão do CLI.
- Entrada pode ser enviada por stdin, argumento ou arquivo temporário, conforme a capability disponível na versão do binário detectado.
- Timeout default: 5 minutos por step.
- Validação: `exec.LookPath("claude")` no bootstrap do provider.

### Copilot CLI (Provider)

- Adapter de subprocess encapsulado em `internal/providers/copilot`.
- A forma exata de invocação para modo de planejamento e modo de execução é tratada como detalhe do adapter, porque flags como `--yolo` precisam de validação por versão e por SO.
- Timeout default: 10 minutos por step de execução.
- Validação: `exec.LookPath("copilot")` no bootstrap do provider.
- Capability opcional: `filesystem_execution`. Quando indisponível ou insegura, o step `execute` opera apenas como geração de plano e comandos aprováveis pelo runtime.

### Editor Externo (HITL Edit)

- Comando: `$EDITOR` ou fallback para `vi` (Unix) / `notepad` (Windows).
- Fluxo: gravar artefato temporário -> abrir editor -> ler conteúdo editado -> revalidar JSON se aplicável -> persistir novos refs -> limpar temporário.
- Adapter: `internal/platform/editor.go`.

## Abordagem de Testes

### Testes Unitários

- Domínio (`Run`, `StepExecution`): transições de estado, invariantes, approve/edit/redo/pause/continue.
- `WorkflowParser` + `Validator`: YAML válido/inválido, referências inexistentes, providers inválidos, campos futuros ignorados sem breaking change.
- `TemplateResolver`: resolução de `{{input}}` e `{{steps.<name>.output}}`.
- `OutputProcessor`: extração de JSON de Markdown, correção segura, validação por schema e diferenciação entre raw/approved/structured.
- `State` serialization: compatibilidade de `state.json` com `schema_version`.

### Testes de Integração

- `FileStore` + filesystem temporário: criação de `.orq/runs/<id>/`, persistência de artifacts e retomada.
- `Engine` + provider fake: happy path, pause/continue, retry, edit, redo, schema failure, fallback para HITL.
- `Editor` adapter: edição de artefato e revalidação.
- Auditoria do `execute`: confirmar logging de comandos disparados pelo runtime, artefatos produzidos e ações HITL.

### Testes E2E

- `orq run` com providers fake retornando outputs pré-definidos.
- `orq continue` retomando run pausada.
- Matrix CI: `ubuntu`, `macos`, `windows`.

### Matriz Requisito → Decisão Técnica → Teste

| Requisito | Decisão Técnica | Estratégia de Teste |
|---|---|---|
| F1.1-F1.7 | Cobra em `internal/cli/` e entrypoint `cmd/orq/` | Integração dos comandos com doubles |
| F2.1-F2.7 | Parser + validator + `embed.FS` | Table-driven com YAML válido/inválido |
| F3.1-F3.7 | Engine sequencial e state machine explícita | Unitário e integração para todas as transições |
| F4.1-F4.8 | Interface `Provider` + adapters version-aware | Doubles de subprocess e smoke tests opcionais |
| F5.1-F5.9 | `OutputProcessor` + artifacts separados | Unitário para parse/correção/schema/persistência |
| F6.1-F6.7 | `Prompter` + editor adapter | Integração com stdin simulado |
| F7.1-F7.7 | `FileStore` em `.orq/runs/<id>/` | Integração com `t.TempDir()` |
| F8.1-F8.7 | Execução mediada pelo runtime + audit log explícito | Integração para comandos e logs do execute |

## Sequenciamento de Desenvolvimento

1. Scaffolding e tooling: `go mod init`, `Taskfile.yml`, `.goreleaser.yaml`, diretórios base, CI.
2. Domínio em `internal/runtime/domain/`.
3. Workflows em `internal/workflows/`.
4. Platform adapters (`subprocess`, `editor`, `filesystem`, `clock`).
5. Providers (`claude`, `copilot`) com resolução de capabilities.
6. Output processor e schema registry.
7. State store em `.orq/runs/<id>/`.
8. Runtime application service e engine.
9. HITL.
10. CLI (`cmd/orq`, `internal/cli`).
11. Release.

## Dependências Técnicas

| Dependência | Versão | Propósito |
|---|---|---|
| Go | 1.24.0 | Linguagem |
| `github.com/spf13/cobra` | `v1.9.x` | Framework CLI |
| `gopkg.in/yaml.v3` | `v3.0.x` | Parser YAML |
| `github.com/google/uuid` | `v1.6.x` | IDs de run |
| `log/slog` | stdlib | Logging estruturado |
| GoReleaser | `v2.x` | Release cross-platform |
| Task | `v3.x` | Task runner local |
| `golangci-lint` | `v1.64.x` | Linter |

Nenhuma dependência de rede para o core. Apenas os providers requerem conectividade.

## Monitoramento e Observabilidade

### Logging

- Logger injetado via bootstrap, nunca global.
- Campos mínimos: `run_id`, `workflow`, `step`, `provider`, `operation`, `duration_ms`.
- Níveis:
  - `INFO`: início/fim de step, decisão HITL, persistência de artifacts.
  - `WARN`: retry automático, JSON corrigido, capability degradada.
  - `ERROR`: falha de provider, persistência, validação.
  - `DEBUG`: command args sanitizados, paths normalizados, output truncado.
- Output: `logs/run.log` dentro do diretório do run.
- Sensibilidade: não logar prompts completos nem secrets; truncar output e registrar apenas metadados quando possível.

### Auditoria

- O step `execute` audita apenas operações mediadas pelo Orchestrator:
  - comandos disparados pelo runtime;
  - artifacts produzidos ou alterados pelo runtime;
  - decisões HITL e inputs de aprovação;
  - capabilities do provider usadas na execução.
- Mutações realizadas autonomamente por um provider externo fora do controle do runtime não entram como garantia de auditoria completa da V1.

## Considerações Técnicas

### Decisões Chave

| # | Decisão | Justificativa | Alternativa Rejeitada |
|---|---|---|---|
| 1 | `cmd/orq` e binário `orq` | Alinha com as regras hard do repositório | `cmd/orchestrator` |
| 2 | `.orq/` no `cwd` | Contrato arquitetural explícito do projeto | `.orchestrator/` |
| 3 | Run isolada em `.orq/runs/<id>/` | Evita sobrescrita e simplifica continue | Layout plano por projeto |
| 4 | Artifacts separados para raw/approved/json/validation | Fecha o contrato de F5 e F7 | `output_ref` único |
| 5 | Adapter de provider version-aware | Evita acoplamento a flags não validadas | Comandos hardcoded na spec |
| 6 | Auditoria apenas de operações mediadas | Implementável e verificável | Prometer captura total de mutações do provider |
| 7 | `Taskfile.yml` como runner oficial | Cross-platform e já presente no repo | `make` |

### Riscos Conhecidos

| Risco | Impacto | Mitigação |
|---|---|---|
| Flags do Copilot CLI mudarem entre versões | Adapter quebra | Resolver comando por capability/version detection |
| Limites de stdin/stdout dos CLIs | Truncamento | Fallback para arquivo temporário |
| Provider retornar JSON incorrigível | Step pausa | Correção segura + 2 retries + HITL |
| Capability de execução autônoma divergir por SO | Comportamento inconsistente | Desabilitar capability quando smoke test falhar |
| Usuário editar JSON de forma inválida no HITL | Run inconsistente | Revalidar antes de aprovar |

### Conformidade com Padrões

| Regra | Conformidade |
|---|---|
| `R-ARCH-001` | Fluxo `cmd/orq -> bootstrap -> application -> domain`, persistência em `.orq/` |
| `R-CLI-001` | Comandos `orq run`, `orq continue`, `orq list` |
| `R-ERR-001` | Retry máximo 2, wrapping e mensagens acionáveis |
| `R-SEC-001` | Sem shell implícito, sem segredos persistidos, paths normalizados |
| `R-TEST-001` | Doubles determinísticos, sem dependência de CLIs reais |
| `R-GOV-001` | Estados canônicos e HITL restrito a `approve/edit/redo/exit` |

### Taskfile.yml (Referência)

```yaml
version: '3'

vars:
  BINARY: orq
  MAIN: ./cmd/orq

tasks:
  build:
    desc: Build the CLI binary
    cmds:
      - go build -o {{.BINARY}} {{.MAIN}}

  test:
    desc: Run all tests
    cmds:
      - go test -race -cover ./...

  lint:
    desc: Run linter
    cmds:
      - golangci-lint run ./...

  fmt:
    desc: Format code
    cmds:
      - gofumpt -w .

  generate:
    desc: Run go generate
    cmds:
      - go generate ./...

  release:snapshot:
    desc: Local snapshot release
    cmds:
      - goreleaser release --snapshot --clean

  default:
    desc: Lint, test, and build
    cmds:
      - task: lint
      - task: test
      - task: build
```

### GoReleaser Config (Referência)

```yaml
version: 2
builds:
  - main: ./cmd/orq
    binary: orq
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}
archives:
  - formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]
checksum:
  name_template: 'checksums.txt'
```

### Arquivos Relevantes e Dependentes

| Arquivo | Propósito |
|---|---|
| `tasks/prd-orq-cli/prd.md` | PRD fonte desta spec |
| `.claude/rules/*.md` | Regras de conformidade |
| `Taskfile.yml` | Task runner oficial |
| `.goreleaser.yaml` | Release config |
| `cmd/orq/main.go` | Entrypoint |
| `go.mod` | Module definition |
