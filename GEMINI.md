# ai-spec-harness — Gemini CLI

Use `AGENTS.md` como fonte canonica das regras deste repositorio.

## Stack

- **Linguagem:** Go 1.26+
- **CLI framework:** spf13/cobra
- **Testes:** go test (unit com FakeFileSystem, integration com build tag `integration`)
- **Release:** GoReleaser + GitHub Actions + Homebrew Cask
- **Dependencias diretas:** cobra, jsonschema/v6

## Comandos

```bash
make test            # testes unitarios
make integration     # testes de integracao
make lint            # golangci-lint
make build           # compila binario
make vet             # go vet
```

## Instrucoes

1. Ler `AGENTS.md` no inicio da sessao.
2. `.agents/skills/` e a fonte de verdade dos fluxos procedurais.
3. `.gemini/commands/` sao adaptadores finos que apontam para a habilidade correta.
4. Em tarefas de execucao, carregar apenas `AGENTS.md`, `agent-governance` e a skill operacional da linguagem ou atividade afetada.
5. Skills de planejamento (`analyze-project`, `create-prd`, `create-technical-specification`, `create-tasks`) entram apenas quando a tarefa pedir esse fluxo explicitamente.
6. Carregar referencias adicionais apenas quando a tarefa exigir.
7. Preservar estilo, arquitetura e fronteiras existentes antes de propor mudancas.
8. Validar mudancas com comandos proporcionais ao risco.

## Convencoes

- **Idioma do codigo:** comentarios, erros e mensagens em portugues (PT-BR)
- **Commits:** Conventional Commits com tipo em ingles e corpo em portugues
- **Testes:** table-driven, FakeFileSystem para unit, t.TempDir() para integration
- **DI:** toda dependencia injetada via construtor, zero estado global
- **Erros:** sempre envolver com `fmt.Errorf("contexto: %w", err)`
- **Pacotes internos:** cada pacote em `internal/` com responsabilidade unica
- **Interfaces:** definir no pacote consumidor quando possivel

## Estrutura

```
cmd/ai_spec_harness/   # comandos cobra (CLI)
internal/              # logica de negocio (34 pacotes)
  fs/                  # abstracao de filesystem (OSFileSystem + FakeFileSystem)
  output/              # printer estruturado (Info, Step, Warn, Error, Debug)
  config/              # DTOs de configuracao
  skills/              # tipos de dominio (Tool, Lang, LinkMode)
  install/             # servico de instalacao
  upgrade/             # servico de upgrade
  detect/              # deteccao de linguagens e ferramentas
  metrics/             # estimativa de tokens e custo
  parity/              # paridade semantica multi-agente
  specdrift/           # deteccao de drift entre specs
  evidence/            # validacao de relatorios
  taskloop/            # orquestracao de execucao de tasks
testdata/              # fixtures, snapshots, projetos-exemplo
.agents/skills/        # skills externas (gerenciadas via skills-lock.json)
internal/embedded/     # assets embarcados (go:embed) - baseline de governanca
```

## CI

- **test.yml:** unit + integration + golangci-lint em ubuntu-24.04 e macos-15
- **release.yml:** semver-next automatico, GoReleaser multi-plataforma
- **release-dry-run.yml:** validacao de release sem side-effects
- **Cobertura:** threshold minimo de 70% enforced no CI (cobertura atual: ~75.5%)
  - Gerar relatorio local: `make coverage`

## Padroes importantes

- `internal/fs/fake.go` — usar FakeFileSystem em testes unitarios, nunca OS real
- `internal/output.Printer` — injetar em todo Service, usar `io.Discard` em testes
- Build tag `integration` para testes que tocam o filesystem real
- Skills externas rastreadas em `skills-lock.json` com hash SHA-256
- Harness auto-instalado neste repo (self-dogfooding)

## Hooks de Governanca

O Gemini CLI nao suporta hooks automaticos (PreToolUse/PostToolUse) nativos como o Claude Code. Os scripts em `.gemini/hooks/` sao utilitarios manuais, nao acionados automaticamente.

### Scripts disponíveis

| Script | Equivalente Claude | Uso |
|--------|-------------------|-----|
| `.gemini/hooks/validate-preload.sh` | `.claude/hooks/validate-preload.sh` | Lembrete de carga base antes de editar codigo |
| `.gemini/hooks/validate-governance.sh` | `.claude/hooks/validate-governance.sh` | Aviso ao modificar arquivos de governanca |

### Como invocar manualmente

```bash
# Verificar contrato de carga antes de editar
bash .gemini/hooks/validate-preload.sh path/to/file.go

# Verificar apos editar arquivo de governanca
bash .gemini/hooks/validate-governance.sh .agents/skills/review/SKILL.md
```

### Invocacao via flag --hook (se suportado pela versao do Gemini CLI)

```bash
gemini --hook "bash .gemini/hooks/validate-preload.sh {file}" \
       --hook "bash .gemini/hooks/validate-governance.sh {file}"
```

### Variaveis de controle

| Variavel | Valores | Efeito |
|----------|---------|--------|
| `GEMINI_PRELOAD_MODE` | `fail` (default) / `warn` | Controla se validate-preload bloqueia ou apenas avisa |
| `GEMINI_GOVERNANCE_MODE` | `fail` (default) / `warn` | Controla se validate-governance bloqueia ou apenas avisa |
| `GOVERNANCE_PRELOAD_CONFIRMED` | `0` (default) / `1` | Bypass do bloqueio de preload quando contrato ja foi confirmado |

### Limitacao conhecida

Sem hooks automaticos, a compliance depende de seguir as instrucoes procedurais manualmente. Use `GOVERNANCE_PRELOAD_CONFIRMED=1` em sessoes longas onde o contrato ja foi confirmado no inicio.

## Orientacoes Especificas para Gemini

1. Ao iniciar uma tarefa, ler `AGENTS.md` e `.agents/skills/agent-governance/SKILL.md` como contexto base antes de editar codigo.
2. Usar `@<command>` para invocar o comando TOML correspondente a skill desejada em `.gemini/commands/`.
3. Seguir as etapas procedurais do SKILL.md carregado pelo comando como se fossem instrucoes sequenciais.
4. Ao final da tarefa, executar os comandos de validacao descritos na secao Validacao do `AGENTS.md`.
5. Nao confiar em enforcement automatico — a compliance depende de seguir as instrucoes procedurais manualmente.
