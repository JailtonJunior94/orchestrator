# Orchestrator

Orquestrador de workflows multi-step para agentes de IA no terminal.

[![Go Version](https://img.shields.io/badge/go-1.26.1-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Sobre

**Orchestrator** resolve o problema de usar agentes de IA (Claude CLI, Copilot CLI) de forma isolada e sem controle. Em vez de executar cada ferramenta separadamente e copiar resultados entre elas, o Orchestrator permite definir pipelines declarativos em YAML onde cada etapa alimenta a próxima, com aprovação humana obrigatória antes de avançar.

O resultado: workflows completos de desenvolvimento -- do PRD ao código -- executados de forma sequencial, auditável e com estado persistido para retomada a qualquer momento.

## Funcionalidades

- **Workflows declarativos em YAML** -- defina pipelines multi-step com providers, templates e schemas
- **Providers plugáveis** -- suporte nativo a Claude CLI e Copilot CLI, extensível para novos agentes
- **Human-in-the-Loop (HITL)** -- aprove, edite, refaça ou pause a execução a cada step
- **Persistência de estado** -- retome workflows pausados com `orchestrator continue`
- **Validação de output** -- extração de JSON com validação via JSON Schema
- **Retry automático** -- até 2 tentativas em falhas recuperáveis antes de pausar para intervenção humana
- **Resolução de templates** -- variáveis `{{input}}` e `{{steps.<name>.output}}` conectam steps entre si
- **Cross-platform** -- binários para macOS, Linux e Windows (amd64 e arm64)
- **Auditoria completa** -- artefatos brutos, aprovados, JSON estruturado e logs por execução

## Instalacao

### Via Go Install

```bash
go install github.com/jailtonjunior/orchestrator/cmd/orchestrator@latest
```

### Via Release (binario)

Baixe o binário para sua plataforma na [página de releases](https://github.com/jailtonjunior/orchestrator/releases) e adicione ao seu `PATH`.

### Build Local

```bash
git clone https://github.com/jailtonjunior/orchestrator.git
cd orchestrator
task build
```

O binário `orchestrator` será gerado na raiz do projeto.

### Requisitos

- **Go 1.26.1+** (para build local)
- **Claude CLI** (`claude`) e/ou **Copilot CLI** (`copilot`) instalados e disponíveis no `PATH`
- **Task** (opcional, para automação de desenvolvimento)

## Uso

### Executar um workflow

```bash
# Com input inline
orchestrator run dev-workflow --input "Sistema de notificações push para mobile"

# Com input de arquivo
orchestrator run dev-workflow --file requisitos.md
```

O workflow `dev-workflow` vem embutido e executa 4 steps:

| Step | Provider | Descricao |
|------|----------|-----------|
| `prd` | Claude | Gera um PRD detalhado a partir do input |
| `techspec` | Claude | Gera Tech Spec com base no PRD aprovado |
| `tasks` | Claude | Decompoe em tasks implementaveis |
| `execute` | Copilot | Propoe plano de execucao com comandos e arquivos |

A cada step, voce escolhe uma acao:

```
[A] Aprovar   - aceita o output e avanca
[E] Editar    - abre editor externo para ajustar o output
[R] Refazer   - re-executa o step com o mesmo input
[S] Sair      - pausa a execucao para retomar depois
```

### Retomar um workflow pausado

```bash
# Retoma o run mais recente
orchestrator continue

# Retoma um run especifico
orchestrator continue --run-id 01JQ...
```

### Listar workflows disponiveis

```bash
orchestrator list
```

## Como Funciona

### Fluxo de Execucao

```
Input do usuario
      |
      v
  Parse do workflow YAML
      |
      v
  Para cada step:
      |
      +---> Resolve templates ({{input}}, {{steps.*.output}})
      |
      +---> Executa provider via subprocess
      |
      +---> Processa output (extrai JSON, valida schema)
      |
      +---> Exibe resultado + prompt HITL
      |
      +---> Acao do usuario (aprovar/editar/refazer/sair)
      |
      +---> Persiste artefatos e estado
      |
      v
  Proximo step ou fim
```

### Persistencia de Estado

Toda execucao e salva em `.orq/runs/<run-id>/`:

```
.orq/
└── runs/
    └── 01JQABC.../
        ├── state.json              # Metadados do run e status dos steps
        ├── artifacts/
        │   ├── prd/
        │   │   ├── raw.md          # Output original do provider
        │   │   ├── approved.md     # Markdown aprovado pelo usuario
        │   │   ├── structured.json # JSON extraido e validado
        │   │   └── validation.json # Relatorio de validacao
        │   ├── techspec/
        │   ├── tasks/
        │   └── execute/
        └── logs/
            └── run.log             # Log estruturado da execucao
```

### Workflows Customizados

Workflows sao arquivos YAML com a seguinte estrutura:

```yaml
name: meu-workflow
steps:
  - name: analise
    provider: claude
    input: |
      Analise o seguinte requisito: {{input}}
      Responda com Markdown e inclua um bloco ```json``` valido.
    output:
      markdown: required
      json_schema: analise/v1
    schema: |
      {
        "type": "object",
        "required": ["summary"],
        "properties": {
          "summary": { "type": "string" }
        }
      }

  - name: implementacao
    provider: copilot
    input: |
      Com base na analise:
      {{steps.analise.output}}
      Proponha a implementacao.
    output:
      markdown: required
```

#### Campos de um Step

| Campo | Obrigatorio | Descricao |
|-------|-------------|-----------|
| `name` | Sim | Identificador unico do step |
| `provider` | Sim | `claude` ou `copilot` |
| `input` | Sim | Template com variaveis resolvidas antes da execucao |
| `output.markdown` | Sim | `required` -- output deve ser Markdown |
| `output.json_schema` | Nao | Nome logico do schema para referencia |
| `schema` | Nao | JSON Schema inline para validacao do bloco JSON |
| `capabilities` | Nao | Capacidades opcionais do provider (ex: `filesystem_execution`) |
| `timeout` | Nao | Timeout de execucao do provider |

#### Variaveis de Template

| Variavel | Descricao |
|----------|-----------|
| `{{input}}` | Input fornecido pelo usuario via `--input` ou `--file` |
| `{{steps.<name>.output}}` | Markdown aprovado de um step anterior |

## Providers

### Claude CLI

- **Binario:** `claude`
- **Timeout padrao:** 5 minutos
- **Deteccao automatica** de versao e perfil de invocacao

### Copilot CLI

- **Binario:** `copilot`
- **Timeout padrao:** 10 minutos
- **Capacidade opcional:** `filesystem_execution` (degrada graciosamente se indisponivel)

Ambos os providers sao verificados no `PATH` antes da execucao. Se um provider nao estiver disponivel, o Orchestrator informa qual binario e esperado e como resolver.

## Arquitetura

```
cmd/orchestrator/     Entrypoint e injecao de versao
internal/
  bootstrap/          Wiring de dependencias
  cli/                Comandos Cobra (run, continue, list, install)
  runtime/
    application/      Services de orquestracao
    domain/           Agregados, entidades e maquinas de estado
    engine.go         Engine de execucao de workflows
  workflows/          Parser YAML, validador, catalogo e templates
  providers/          Adapters de Claude CLI e Copilot CLI
  state/              Persistencia em .orq/
  hitl/               Prompt interativo no terminal
  output/             Extracao, correcao e validacao de JSON
  platform/           Abstracoes de OS (subprocess, editor, filesystem)
```

Principios:
- **Clean Architecture** -- dependencias apontam para dentro (domain nao conhece infra)
- **Composicao sobre heranca** -- interfaces minimas, structs pequenas
- **Strategy Pattern** -- providers, renderers e politicas de retry sao plugaveis
- **State Pattern** -- transicoes de estado de Run e Step sao explicitas e centralizadas

## Desenvolvimento

### Requisitos

- Go 1.26.1+
- [Task](https://taskfile.dev/) (task runner)
- [golangci-lint](https://golangci-lint.run/) (linter)
- [gofumpt](https://github.com/mvdan/gofumpt) (formatter)

### Comandos

```bash
# Build
task build

# Testes com race detector e cobertura
task test

# Lint
task lint

# Formatar codigo
task fmt

# Pipeline completo (lint + test + build)
task

# Release snapshot local
task release:snapshot
```

### Executando os Testes

```bash
go test -race -cover ./...
```

Os testes usam doubles deterministicos para subprocess, filesystem e terminal. Nenhum teste chama providers reais.

## Roadmap

- [ ] Suporte a Gemini CLI como provider
- [ ] Suporte a OpenAI Codex CLI como provider
- [ ] Execucao paralela de steps independentes
- [ ] Workflows definidos pelo usuario em `.orq/workflows/`
- [ ] Dashboard TUI para acompanhar execucoes
- [ ] Plugin system para hooks pre/pos step

## Licenca

Orchestrator esta licenciado sob a licenca MIT. Consulte o arquivo [`LICENSE`](LICENSE) para mais informacoes.
