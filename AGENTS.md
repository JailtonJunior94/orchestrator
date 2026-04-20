# Regras para Agentes de IA

Este diretório centraliza regras para uso com agentes de IA em tarefas reais de análise, alteração e validação de código.

## Objetivo

Use estas instruções para manter consistência, segurança e qualidade ao trabalhar com código, configuração, validação e evolução de sistemas.

## Modo de trabalho

1. Entender o contexto antes de editar qualquer arquivo.
2. Preferir a menor mudança segura que resolva a causa raiz.
3. Preservar arquitetura, convenções e fronteiras existentes.
4. Não introduzir abstrações ou dependências sem demanda concreta.
5. Atualizar testes quando houver mudança de comportamento.
6. Rodar validações proporcionais à mudança.
7. Registrar bloqueios e suposições quando o contexto estiver incompleto.
8. Evitar reescritas amplas e overengineering; risco de regressão é restrição principal.

## Contrato de carga base

Toda skill que altera código deve carregar, como primeiro passo, a seguinte base obrigatória — essa instrução é reforçada em cada SKILL.md como medida defensiva:

1. Ler este `AGENTS.md`.
2. Ler `.agents/skills/agent-governance/SKILL.md`.

Essa base define governança para análise, alteração e validação, carregamento sob demanda de regras de DDD, erros, segurança e testes, e critérios mínimos de preservação arquitetural, risco e validação proporcional.

Skills individuais devem declarar apenas cargas adicionais específicas ao seu contexto.

## Regras por Linguagem

| Linguagem | Skill a carregar |
|-----------|-----------------|
| Go | `.agents/skills/go-implementation/SKILL.md` |
| Node/TypeScript | `.agents/skills/node-implementation/SKILL.md` |
| Python | `.agents/skills/python-implementation/SKILL.md` |
| Revisão/refatoração Go (OC) | `.agents/skills/object-calisthenics-go/SKILL.md` |
| Correção de bugs | `.agents/skills/bugfix/SKILL.md` |

## Governança por Ferramenta

| Arquivo | Ferramenta |
|---------|-----------|
| `CLAUDE.md` | Claude Code (hooks, rules, agents) |
| `GEMINI.md` | Gemini CLI (commands, orientações procedurais) |
| `CODEX.md` | Codex (config.toml, instrução de sessão) |
| `COPILOT.md` | GitHub Copilot (Chat e gh copilot CLI); contexto via `.github/copilot-instructions.md` |

Esses arquivos são suplementares a este `AGENTS.md`. A fonte de verdade dos fluxos procedurais permanece em `.agents/skills/`.

## Referências

Cada skill lista suas próprias referências em `references/` com gatilhos de carregamento no respectivo `SKILL.md`. Não duplicar a listagem aqui — consultar o SKILL.md da skill ativa para saber quais referências carregar e em que condição.

## Validação

Antes de concluir uma alteração, seguir Etapa 4 de `.agents/skills/agent-governance/SKILL.md`.

## Upgrades de Skills Externas

Toda atualização de hash em `skills-lock.json` deve ser acompanhada de um registro
de decisão salvo em `audit/`. Use o template em `tasks/templates/skill-upgrade-decision.md`.

Campos obrigatórios: skill, versão anterior, versão nova, motivador, critério de aceitação, data.

Sem registro de motivador: upgrade não aprovado para merge.

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
make coverage        # relatorio de cobertura
make bench           # benchmarks
```

## Convencoes

| Aspecto | Regra |
|---------|-------|
| Idioma | PT-BR (comentarios, erros, mensagens) |
| Commits | Conventional Commits: tipo em ingles, corpo em portugues |
| Testes | table-driven; FakeFileSystem (unit); t.TempDir() (integration) |
| DI | injetar via construtor; zero estado global |
| Erros | `fmt.Errorf("contexto: %w", err)` |
| Pacotes | um por responsabilidade em `internal/` |
| Interfaces | definir no pacote consumidor quando possivel |

## Estrutura

```
cmd/ai_spec_harness/
internal/
  fs/
  output/
  config/
  skills/
  install/
  upgrade/
  detect/
  metrics/
  parity/
  specdrift/
  evidence/
  taskloop/
testdata/
.agents/skills/
internal/embedded/
```

## CI

- **test.yml:** unit + integration + golangci-lint em ubuntu-24.04 e macos-15
- **release.yml:** semver-next automatico, GoReleaser multi-plataforma
- **release-dry-run.yml:** validacao de release sem side-effects
- **Cobertura:** threshold minimo de 75%; gerar local: `make coverage`

## Padroes Importantes

- `internal/fs/fake.go` — FakeFileSystem em testes unitarios, nunca OS real
- `internal/output.Printer` — injetar em todo Service; `io.Discard` em testes
- Build tag `integration` para testes que tocam o filesystem real
- Skills externas rastreadas em `skills-lock.json` com hash SHA-256
- Harness auto-instalado neste repo (self-dogfooding)

## Restrições

Nao inventar contexto ausente. Nao assumir versao sem verificar. Nao alterar comportamento publico sem registrar. Adaptar exemplos ao contexto real.

## Documentacao

- [Guia de troubleshooting](docs/troubleshooting.md) — problemas comuns com sintoma, causa, solucao e verificacao
- [Ciclo de telemetria](docs/telemetry-feedback-cycle.md) — feedback loop com GOVERNANCE_TELEMETRY
