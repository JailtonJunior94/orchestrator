# ORQ

CLI para orquestrar workflows de agentes de IA no terminal, com execução step-by-step, Human-in-the-Loop e persistência local para auditoria e retomada.

[![Go Version](https://img.shields.io/badge/go-1.26.2-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Sobre

O ORQ resolve o problema de operar múltiplos agentes de IA sem controle explícito de fluxo. Em vez de alternar manualmente entre CLIs, copiar saídas entre ferramentas e perder contexto da execução, você descreve um workflow e o runtime coordena os steps, encadeia o output aprovado e persiste tudo em `.orq/`.

O projeto já suporta runtime com `claude`, `copilot`, `gemini` e `codex`, além de um subsistema de instalação de assets para ecossistemas compatíveis. O foco continua sendo uma experiência `CLI-first`, auditável e extensível.

## Funcionalidades

- Workflows declarativos em YAML com validação estrutural.
- Execução step-by-step com aprovação humana obrigatória entre etapas.
- Persistência local de estado, artefatos e logs por run.
- Retomada de execução com `orq continue`.
- Extração de Markdown e JSON estruturado do output dos providers.
- Validação de JSON Schema por step quando configurado.
- Retry automático antes de pausar a run por falha.
- Catálogo embutido de workflows.
- Comando `orq install` para instalar, atualizar, remover e verificar assets.

## Instalação

### Requisitos

- Go `1.26.2+` para build local ou `go install`
- Pelo menos os providers usados no workflow instalado no `PATH`
- `task` é opcional, mas é a interface preferida para desenvolvimento local

Para o workflow embutido `dev-workflow`, o caminho padrão usa:

- `claude` nos steps `prd`, `techspec` e `tasks`
- `copilot` no step `execute`

### Homebrew

```bash
brew install jailtonjunior/tap/orq
```

Para atualizar:

```bash
brew upgrade orq
```

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

O binário gerado será `./orq`.

## Primeiros 5 Minutos

Se você quer apenas validar que o ORQ está funcionando, siga este fluxo:

### 1. Verifique a CLI

```bash
orq --help
```

### 2. Liste os workflows disponíveis

```bash
orq list
```

Se quiser evitar a interface interativa:

```bash
orq --no-tui list
```

### 3. Execute o workflow padrão

```bash
orq run dev-workflow --input "Criar sistema de notificações push para mobile"
```

Ou usando arquivo:

```bash
orq run dev-workflow --file requisitos.md
```

### 4. Revise cada step

Após cada execução, o ORQ entra em Human-in-the-Loop. Você decide como seguir:

- `approve`: aceita a saída e avança
- `edit`: abre o editor para ajustar o conteúdo aprovado
- `redo`: executa o step novamente
- `exit`: pausa a run para retomar depois

### 5. Retome se necessário

```bash
orq continue
```

Para uma run específica:

```bash
orq continue --run-id <run-id>
```

## Passo a Passo de Uso

### Fluxo 1: executar um workflow do zero

1. Confira os workflows disponíveis com `orq list`.
2. Escolha um workflow, por exemplo `dev-workflow`.
3. Passe o input inline com `--input` ou via arquivo com `--file`.
4. Revise o output de cada step no momento da aprovação.
5. Ao concluir, consulte os artefatos persistidos em `.orq/runs/<run-id>/`.

Exemplo:

```bash
orq run dev-workflow --input "Adicionar autenticação por magic link"
```

### Fluxo 2: executar em ambiente não interativo ou mais simples

Se você quiser reduzir dependência de TUI:

```bash
orq --no-tui run dev-workflow --input "Adicionar exportação CSV"
```

Para desabilitar animações:

```bash
orq --no-animation run dev-workflow --input "Adicionar exportação CSV"
```

Também é possível desabilitar animações via variável de ambiente:

```bash
ORQ_NO_ANIMATION=1 orq run dev-workflow --input "Adicionar exportação CSV"

# usar um binário alternativo para o provider logical `claude`
ORQ_CLAUDE_BINARY=claude orq run dev-workflow --input "Adicionar exportação CSV"
```

### Fluxo 3: retomar uma run pausada

1. Saia de uma execução usando `exit` no Human-in-the-Loop.
2. Retome a última run pausada com `orq continue`.
3. Se houver mais de uma run, informe `--run-id`.

### Fluxo 4: inspecionar a execução

Cada run é armazenada em:

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

- `state.json`: snapshot da run e status atual
- `raw.md`: saída original do provider
- `approved.md`: conteúdo aprovado ou editado pelo usuário
- `structured.json`: JSON extraído do output, quando existir
- `validation.json`: resultado da validação de schema
- `run.log`: log append-only da execução

## Comandos da CLI

### `orq run <workflow>`

Executa um workflow embutido.

Flags principais:

- `--input`: input inline
- `--file`, `-f`: input vindo de arquivo

Exemplos:

```bash
orq run dev-workflow --input "Criar módulo de cobrança recorrente"
orq run dev-workflow --file prompt.md
```

### `orq continue`

Retoma a última run pausada.

Flag principal:

- `--run-id`: retoma uma run específica

Exemplo:

```bash
orq continue --run-id <run-id>
```

### `orq list`

Lista os workflows disponíveis. Em TTY, pode abrir a interface interativa; com `--no-tui`, imprime em texto puro.

Exemplo:

```bash
orq --no-tui list
```

### `orq install`

Gerencia assets compatíveis com os ecossistemas suportados pelo projeto.

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

Exemplos:

```bash
orq install list --provider claude
orq install --provider codex --kind skill --yes
orq install verify --global --provider copilot
orq install remove --provider gemini --asset reviewer
```

## Workflow Padrão

O catálogo embutido inclui hoje o workflow `dev-workflow`, com quatro steps:

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

## Como Funciona

O fluxo principal de execução é:

1. Você executa `orq run <workflow>` com `--input` ou `--file`.
2. O ORQ carrega o workflow, valida a definição e checa os providers necessários.
3. O runtime resolve templates como `{{input}}` e `{{steps.<name>.output}}`.
4. Cada provider é executado via subprocesso com timeout e perfil compatível.
5. O output é processado em Markdown e, quando existir, JSON estruturado.
6. O Human-in-the-Loop decide se aprova, edita, refaz ou pausa.
7. O estado e os artefatos são persistidos em `.orq/`.
8. A run segue para o próximo step ou fica pronta para retomada posterior.

## Human-in-the-Loop

Após cada step, o runtime entra em modo de decisão humana com ações canônicas:

- `approve`
- `edit`
- `redo`
- `exit`

Esse modelo permite revisão obrigatória entre etapas sem perder o histórico da execução.

## Formato de Workflow

Os workflows são definidos em YAML. Exemplo compatível com o runtime:

````yaml
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
````

Campos relevantes:

| Campo | Obrigatório | Descrição |
| --- | --- | --- |
| `name` | Sim | Identificador único do step |
| `provider` | Sim | Provider do step, como `claude`, `copilot`, `gemini` ou `codex` |
| `input` | Sim | Template resolvido antes da execução |
| `output.markdown` | Sim | Define que o output em Markdown é obrigatório |
| `output.json_schema` | Não | Nome lógico do schema |
| `schema` | Não | JSON Schema inline para validação |
| `capabilities` | Não | Capacidades opcionais do provider |
| `timeout` | Não | Timeout do step |

Variáveis de template:

| Variável | Descrição |
| --- | --- |
| `{{input}}` | Input informado pelo usuário |
| `{{steps.<name>.output}}` | Markdown aprovado de um step anterior |

## Providers Suportados

### Runtime

| Provider | Binário esperado | Timeout padrão | Observações |
| --- | --- | --- | --- |
| Claude | `claude` | 5 min | Detecta perfil compatível com `--output-format json` |
| Copilot | `copilot` | 10 min | Executa em modo `--yolo` quando suportado |
| Gemini | `gemini` | 5 min | Requer CLI compatível com `-p` e `--yolo` |
| Codex | `codex` | 5 min | Usa `codex exec` e sandbox default `read-only` |

Todos os providers são validados antes da run. Se um binário não estiver disponível ou a versão instalada for incompatível com o perfil esperado, a execução falha com mensagem explícita.

### Instalação de assets

O comando `orq install` gerencia assets por escopo:

- `project`: inventário em `.orq/install/inventory.json`
- `global`: inventário em `~/.local/state/orq/install/inventory.json` ou diretório equivalente do sistema

O catálogo atual descobre assets principalmente em:

- `.claude/commands/`
- `.claude/skills/`
- `.gemini/commands/`
- `.gemini/skills/`
- `.codex/skills/`
- `AGENTS.md`
- `.github/copilot-instructions.md`

Mapeamentos importantes:

- Claude: instala commands e skills em `.claude/`
- Gemini: instala commands e skills em `.gemini/`
- Codex: instala skills em `.codex/` e reconcilia `config.toml`
- Copilot:
  - no projeto, usa `.claude/commands`, `.claude/skills` e `AGENTS.md`
  - no escopo global, usa `.copilot/skills` e `.copilot/copilot-instructions.md`

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

- CLI-first, sem backend HTTP
- Clean Architecture e Ports and Adapters
- Providers executam o step, mas não controlam o fluxo do workflow
- `Run` é o aggregate root natural da execução
- Persistência local simples com filesystem + JSON

## Desenvolvimento

### Tooling preferido

- `task`
- `gofumpt`
- `golangci-lint`
- `goreleaser`

### Comandos úteis

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

## Contribuição

Para contribuir localmente:

```bash
git clone https://github.com/jailtonjunior/orchestrator.git
cd orchestrator
task fmt
task lint
task test
task build
```

Pontos úteis para quem está chegando:

- O nome da CLI é `orq`
- O entrypoint é `cmd/orq`
- O workflow embutido principal é `dev-workflow`
- O runtime já suporta `claude`, `copilot`, `gemini` e `codex`
- O subsistema `install` faz parte da interface pública da CLI

## Licença

MIT. Consulte [LICENSE](LICENSE).
