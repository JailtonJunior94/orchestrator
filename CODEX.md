# ai-spec-harness — Codex

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

1. Ler `AGENTS.md` no inicio da sessao — e a instrucao principal de sessao para este repositorio.
2. `.agents/skills/` e a fonte de verdade dos fluxos procedurais.
3. `.codex/config.toml` lista as skills habilitadas para resolucao e upgrade via harness.
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

### Capacidades

O Codex nao suporta hooks de pre/pos-edicao equivalentes ao Claude Code (PreToolUse/PostToolUse). O arquivo `.codex/config.toml` lista apenas as skills disponiveis para resolucao pelo harness — nao ha mecanismo para interceptar operacoes de edicao de arquivo.

| Mecanismo | Suporte |
|-----------|---------|
| PreToolUse hooks | Nao suportado |
| PostToolUse hooks | Nao suportado |
| Instrucoes de sessao (AGENTS.md) | Suportado via system prompt |
| Skills via config.toml | Suportado |

### Workaround recomendado

Como o Codex nao suporta hooks automaticos, use as seguintes alternativas para manter compliance:

1. **Variavel de ambiente pre-sessao:** Configurar `GOVERNANCE_PRELOAD_CONFIRMED=1` antes de iniciar uma sessao Codex confirma explicitamente que o contrato de carga base sera seguido.

2. **Hook pre-commit no git:** Adicionar ao `.git/hooks/pre-commit` uma chamada a `bash .gemini/hooks/validate-governance.sh` (os scripts Gemini tambem funcionam em linha de comando) para capturar edicoes indevidas em arquivos de governanca antes do commit.

   ```bash
   # .git/hooks/pre-commit
   git diff --cached --name-only | while read file; do
     bash .gemini/hooks/validate-governance.sh "$file" || exit 1
   done
   ```

3. **Instrucao explicita no AGENTS.md:** O Codex le `AGENTS.md` como system prompt — o contrato de carga base esta documentado nele e o modelo e instruido a segui-lo antes de editar codigo.

### Gap registrado

Codex nao oferece enforcement automatico de governanca. A compliance depende inteiramente do modelo seguir as instrucoes procedurais do `AGENTS.md` e das skills carregadas. Este gap esta documentado para rastreabilidade.

## Orientacoes Especificas para Codex

O Codex le `AGENTS.md` como instrucao de sessao e `.codex/config.toml` como metadados de skills para resolucao pelo harness. Para manter compliance:

1. Ao iniciar uma sessao, confirmar que `AGENTS.md` e `.agents/skills/agent-governance/SKILL.md` foram lidos.
2. Descobrir skills disponiveis consultando os paths listados em `.codex/config.toml`.
3. Seguir as etapas procedurais do SKILL.md de cada skill invocada.
4. Ao final da tarefa, executar os comandos de validacao descritos na secao Validacao do `AGENTS.md`.
5. Enforcement depende do modelo seguir as instrucoes — nao ha bloqueio automatico.
