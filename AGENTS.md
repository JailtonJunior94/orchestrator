# Regras para Agentes de IA

Este diretório centraliza regras para uso com agentes de IA em tarefas reais de análise, alteração e validação de código.

## Objetivo

Use estas instruções para manter consistência, segurança e qualidade ao trabalhar com código, configuração, validação e evolução de sistemas.

## Modo de trabalho

1. Entender o contexto antes de editar qualquer arquivo.
2. Preferir a menor mudança segura que resolva a causa raiz.
3. Preservar arquitetura, convenções e fronteiras já existentes no contexto analisado.
4. Não introduzir abstrações, camadas ou dependências sem demanda concreta.
5. Atualizar ou adicionar testes quando houver mudança de comportamento.
6. Rodar validações proporcionais à mudança.
7. Registrar bloqueios e suposições explicitamente quando o contexto estiver incompleto.

## Diretrizes de Estrutura

1. Priorize entendimento do código e do contexto atual antes de propor refatorações.
2. Respeite padrões existentes de nomenclatura, organização e tratamento de erro.
3. Defina estrutura simples, evolutiva e com defaults explícitos.
4. Evite reescritas amplas quando uma alteração localizada resolver o problema.
5. Estabeleça contratos, testes e comandos de validação cedo quando eles ainda não existirem.
6. Considere risco de regressão como restrição principal.
7. Evite overengineering disfarçado de arquitetura futura.

## Contrato de carga base

Toda skill que altera código deve carregar, como primeiro passo, a seguinte base obrigatória — essa instrução é reforçada em cada SKILL.md como medida defensiva:

1. Ler este `AGENTS.md`.
2. Ler `.agents/skills/agent-governance/SKILL.md`.

Essa base define governança para análise, alteração e validação, carregamento sob demanda de regras de DDD, erros, segurança e testes, e critérios mínimos de preservação arquitetural, risco e validação proporcional.

Skills individuais devem declarar apenas cargas adicionais específicas ao seu contexto.

## Regras por Linguagem

Para tarefas que alteram código Go, carregar também:

- `.agents/skills/go-implementation/SKILL.md`

Para tarefas que alteram código Node/TypeScript, carregar também:

- `.agents/skills/node-implementation/SKILL.md`

Para tarefas que alteram código Python, carregar também:

- `.agents/skills/python-implementation/SKILL.md`

Para tarefas de revisão ou refatoração incremental de design em Go guiadas por heurísticas de object calisthenics, carregar também:

- `.agents/skills/object-calisthenics-go/SKILL.md`

Para tarefas de correção de bugs com remediação e teste de regressão, carregar também:

- `.agents/skills/bugfix/SKILL.md`

## Governança por Ferramenta

Cada ferramenta suportada tem um arquivo de governança dedicado na raiz do repositório:

- `CLAUDE.md` — instruções específicas para Claude Code (hooks, rules, agents)
- `GEMINI.md` — instruções específicas para Gemini CLI (commands, orientações procedurais)
- `CODEX.md` — instruções específicas para Codex (config.toml, instrução de sessão)
- `COPILOT.md` — instruções específicas para GitHub Copilot (Chat e gh copilot CLI); contexto automático via `.github/copilot-instructions.md`

Esses arquivos são suplementares a este `AGENTS.md` e fornecem contexto de stack, comandos e orientações por ferramenta. A fonte de verdade dos fluxos procedurais permanece em `.agents/skills/`.

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
- **Cobertura:** threshold minimo de 75% enforced no CI (cobertura atual: ~76.7%)
  - Gerar relatorio local: `make coverage`

## Padroes Importantes

- `internal/fs/fake.go` — usar FakeFileSystem em testes unitarios, nunca OS real
- `internal/output.Printer` — injetar em todo Service, usar `io.Discard` em testes
- Build tag `integration` para testes que tocam o filesystem real
- Skills externas rastreadas em `skills-lock.json` com hash SHA-256
- Harness auto-instalado neste repo (self-dogfooding)

## Restrições

1. Não inventar contexto ausente.
2. Não assumir versão de linguagem, framework ou runtime sem verificar.
3. Não alterar comportamento público sem deixar isso explícito.
4. Não usar exemplos como cópia cega; adaptar ao contexto real.
