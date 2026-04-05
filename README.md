# ORQ

CLI para orquestrar workflows de agentes de IA no terminal com execução step-by-step, Human-in-the-Loop e persistência local.

[![Go Version](https://img.shields.io/badge/go-1.26.1-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## O que é

O ORQ resolve o problema de operar múltiplos agentes de IA de forma isolada e sem controle de fluxo. Em vez de alternar manualmente entre CLIs diferentes, a ferramenta executa um workflow declarativo, conecta a saída aprovada de um step ao próximo e mantém estado e artefatos locais para auditoria e retomada.

Hoje o projeto suporta na V1 execução via `claude` e `copilot`, além de gerenciamento de assets instaláveis para os ecossistemas Claude, Gemini, Codex e Copilot. A arquitetura de providers permanece extensível para evoluções futuras.

## Funcionalidades

- Workflows declarativos em YAML com steps, providers, templates e validação estrutural.
- Workflow built-in `dev-workflow` para fluxo de desenvolvimento assistido por IA.
- Providers plugáveis com suporte V1 para `claude` e `copilot`.
- Human-in-the-Loop em cada step com ações canônicas: `approve`, `edit`, `redo` e `exit`.
- Persistência local em `.orq/runs/<run-id>` com retomada por `orq continue`.
- Extração de Markdown e JSON estruturado do output dos providers.
- Validação de JSON com schema por step quando configurado.
- Retry automático para falhas de provider antes de pausar a execução.
- Logs por run e armazenamento de artefatos brutos, aprovados e validados.
- Catálogo embutido de workflows e comando para listagem de workflows disponíveis.
- Subsistema de instalação para assets de providers com preview, apply, list, verify, update e remove.
- Inventário de instalação por escopo `project` e `global`.
- Verificação estrutural e, em alguns casos, funcional dos assets instalados.

## Como funciona

O fluxo principal é:

1. O usuário executa `orq run <workflow>` com `--input` ou `--file`.
2. O ORQ carrega o workflow embutido, valida a definição e verifica se os providers necessários estão disponíveis no `PATH`.
3. Para cada step, resolve templates como `{{input}}` e `{{steps.<name>.output}}`.
4. Executa o provider correspondente via subprocesso com timeout e perfil de invocação compatível com a CLI instalada.
5. Processa o output, extraindo Markdown e, quando houver, JSON estruturado.
6. Exibe o resultado para decisão humana: aprovar, editar, refazer ou sair.
7. Persiste estado, artefatos e logs em `.orq/`.
8. Continua para o próximo step ou pausa a run para retomada posterior.

## Instalação

### Requisitos

- Go `1.26.1+` para build local ou `go install`
- Os providers do workflow padrão instalados no `PATH`:
  - `claude`
  - `copilot`
- `task` é opcional, mas é a interface preferida para desenvolvimento local

### Via `go install`

```bash
go install github.com/jailtonjunior/orchestrator/cmd/orq@latest
```

### Build local

```bash
git clone https://github.com/jailtonjunior/orchestrator.git
cd orchestrator
task build
```

O binário gerado é `./orq`.

### Release local

```bash
task release:snapshot
```

## Uso rápido

### Ver ajuda

```bash
orq --help
```

### Listar workflows embutidos

```bash
orq list
```

### Executar um workflow

```bash
orq run dev-workflow --input "Sistema de notificações push para mobile"
```

Ou:

```bash
orq run dev-workflow --file requisitos.md
```

### Retomar uma execução pausada

```bash
orq continue
```

Para uma run específica:

```bash
orq continue --run-id <run-id>
```

## Comandos da CLI

Os comandos expostos hoje são:

- `orq run <workflow>`: executa um workflow built-in.
- `orq continue`: retoma a última run pausada ou pendente.
- `orq list`: lista workflows embutidos.
- `orq install`: instala e gerencia assets suportados pelo ORQ.

## Workflow padrão: `dev-workflow`

O catálogo embutido contém hoje o workflow `dev-workflow`, com quatro steps:

| Step | Provider | Objetivo |
| --- | --- | --- |
| `prd` | `claude` | Gerar um PRD detalhado a partir do input |
| `techspec` | `claude` | Gerar Tech Spec com base no PRD aprovado |
| `tasks` | `claude` | Decompor PRD + Tech Spec em tasks implementáveis |
| `execute` | `copilot` | Propor um plano de execução mediado pelo runtime |

O último step exige um JSON estruturado com:

- `summary`
- `commands` com `executable` e `args`
- `files` opcional com `path` e `content`

O prompt do step `execute` proíbe explicitamente `git commit`, `git push` e `gh pr create`.

## Human-in-the-Loop

Após cada execução de step, o ORQ entra em modo de decisão humana. As ações canônicas do runtime são:

- `approve`: aceita o output e avança.
- `edit`: abre o editor externo para ajustar o conteúdo aprovado.
- `redo`: refaz o step com o mesmo input.
- `exit`: pausa a execução.

Esse modelo permite revisão obrigatória entre steps sem perder o estado da run.

## Formato de workflow

Os workflows são definidos em YAML. Exemplo compatível com o runtime:

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
    provider: codex
    input: |
      Com base na analise aprovada:
      {{steps.analise.output}}
      Proponha a implementacao.
    output:
      markdown: required
```

### Campos relevantes

| Campo | Obrigatório | Descrição |
| --- | --- | --- |
| `name` | Sim | Identificador único do step |
| `provider` | Sim | `claude` ou `copilot` na V1; a arquitetura permanece extensível |
| `input` | Sim | Template resolvido antes da execução |
| `output.markdown` | Sim | Define que o output em Markdown é obrigatório |
| `output.json_schema` | Não | Nome lógico do schema |
| `schema` | Não | JSON Schema inline para validação |
| `capabilities` | Não | Capacidades opcionais do provider |
| `timeout` | Não | Timeout do step |

### Variáveis de template

| Variável | Descrição |
| --- | --- |
| `{{input}}` | Input informado pelo usuário |
| `{{steps.<name>.output}}` | Markdown aprovado de um step anterior |

## Persistência e artefatos

Cada run é armazenada em `.orq/runs/<run-id>/`.

```text
.orq/
└── runs/
    └── <run-id>/
        ├── state.json
        ├── artifacts/
        │   └── <step>/
        │       ├── raw.md
        │       ├── approved.md
        │       ├── structured.json
        │       └── validation.json
        └── logs/
            └── run.log
```

Arquivos principais:

- `state.json`: snapshot da run, status e ponteiros para artefatos.
- `raw.md`: saída bruta do provider.
- `approved.md`: versão aprovada ou editada pelo usuário.
- `structured.json`: JSON extraído do output.
- `validation.json`: relatório de validação do schema.
- `run.log`: log append-only da execução.

## Providers suportados

### Runtime

| Provider | Binário esperado | Timeout padrão | Observações |
| --- | --- | --- | --- |
| Claude | `claude` | 5 min | Detecta perfil compatível com `--output-format json` |
| Copilot | `copilot` | 10 min | Executa em modo `--yolo` quando suportado |
| Gemini | `gemini` | 5 min | Requer CLI compatível com `-p` e `--yolo` |
| Codex | `codex` | 5 min | Usa `codex exec` e sandbox default `read-only` |

Todos os providers são validados antes da run. Se um binário não estiver disponível ou a versão instalada for incompatível com o perfil esperado, a execução falha com mensagem explícita.

### Instalação de assets

O comando `orq install` gerencia assets de providers por escopo:

- `project`: inventário em `.orq/install/inventory.json`
- `global`: inventário em `~/.local/state/orq/install/inventory.json` ou diretório equivalente do sistema

Operações disponíveis:

```bash
orq install
orq install update
orq install remove
orq install list
orq install verify
```

Filtros suportados:

- `--provider`
- `--asset`
- `--kind`
- `--project`
- `--global`

Flags adicionais para operações mutáveis:

- `--conflict abort|skip|overwrite`
- `--yes`

### O que o subsistema de install gerencia

O catálogo atual descobre assets instaláveis a partir do repositório, principalmente em:

- `.claude/commands/`
- `.claude/skills/`, quando existir
- `.gemini/commands/` e `.gemini/skills/`, quando existirem
- `.codex/skills/`, quando existir
- `AGENTS.md`
- `.github/copilot-instructions.md`, quando existir

Mapeamentos importantes:

- Claude: instala commands e skills em `.claude/`
- Gemini: instala commands e skills em `.gemini/`
- Codex: instala skills em `.codex/` e reconcilia `config.toml`
- Copilot:
  - no projeto, usa `.claude/commands`, `.claude/skills` e `AGENTS.md`
  - no escopo global, usa `.copilot/skills` e `.copilot/copilot-instructions.md`

Exemplos:

```bash
orq install list --provider claude
orq install --provider codex --kind skill --yes
orq install verify --global --provider copilot
orq install remove --provider gemini --asset reviewer
```

## Arquitetura

O projeto segue uma organização de CLI em Go com composição explícita:

```text
cmd/orq/              entrypoint
internal/bootstrap/   wiring das dependências
internal/cli/         comandos Cobra e rendering
internal/runtime/     engine de execução e domínio da run
internal/workflows/   parser, catálogo, templates e validação
internal/providers/   adapters das CLIs externas
internal/output/      extração e validação de output estruturado
internal/install/     catálogo, planner, inventory e adapters de install
internal/state/       persistência de runs
internal/platform/    filesystem, subprocesso, editor e clock
internal/hitl/        interação humana no terminal
```

Princípios adotados:

- CLI-first, sem backend HTTP.
- Clean Architecture e Ports and Adapters.
- Providers executam o step, mas não controlam o fluxo do workflow.
- `Run` é o aggregate root natural da execução.
- Persistência local simples com filesystem + JSON.

## Desenvolvimento

### Tooling preferido

- `task`
- `gofumpt`
- `golangci-lint`
- `goreleaser`

### Comandos

```bash
task build
task test
task lint
task fmt
task generate
task release:snapshot
```

O `task test` executa:

```bash
go test -race -cover ./...
```

## Estado atual do projeto

Pontos importantes para quem estiver usando ou contribuindo:

- O nome da CLI é `orq`.
- O entrypoint atual é `cmd/orq`.
- O workflow embutido disponível hoje é `dev-workflow`.
- O repositório já suporta `gemini` e `codex` no runtime; o README antigo citava apenas Claude e Copilot.
- O subsistema de `install` já faz parte da interface pública da CLI.

## Licença

MIT. Consulte [LICENSE](LICENSE).
