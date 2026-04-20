# ai-spec-harness — Governança para GitHub Copilot

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
make coverage        # relatorio de cobertura (minimo 70%)
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
- **Cobertura:** threshold minimo de 70% enforced no CI (cobertura atual: ~75.5%)

## Padroes Importantes

- `internal/fs/fake.go` — usar FakeFileSystem em testes unitarios, nunca OS real
- `internal/output.Printer` — injetar em todo Service, usar `io.Discard` em testes
- Build tag `integration` para testes que tocam o filesystem real
- Skills externas rastreadas em `skills-lock.json` com hash SHA-256
- Harness auto-instalado neste repo (self-dogfooding)

## Governança

Regras transversais, precedencia e restrições operacionais estao definidas em:

- `AGENTS.md` — instrucao de sessao e contrato de carga base
- `.agents/skills/agent-governance/SKILL.md` — governanca para analise e alteracao de codigo
- `.claude/rules/governance.md` — precedencia e politica de evidencia

Regras essenciais:
1. Ler `AGENTS.md` e `.agents/skills/agent-governance/SKILL.md` antes de editar codigo.
2. Toda alteracao deve ser justificavel pelo PRD, por regra explicita ou por necessidade tecnica demonstravel.
3. Preservar estilo, arquitetura e fronteiras existentes antes de propor mudancas.
4. Validar mudancas com comandos proporcionais ao risco.
5. Nao inventar contexto ausente.
6. Nao executar acoes destrutivas sem pedido explicito.

## Mecanismo de Carregamento de Contexto

O GitHub Copilot possui **dois modos distintos** com mecanismos diferentes:

### Copilot Chat (VS Code / GitHub.com) — Suporte Nativo

O arquivo `.github/copilot-instructions.md` e carregado automaticamente como contexto de repositorio pelo GitHub Copilot Chat na extensao VS Code (v1.143+) e no GitHub.com. Este e o mecanismo nativo para fornecer instrucoes de repositorio ao Copilot.

- **Arquivo de contexto automatico:** `.github/copilot-instructions.md` (ja existe neste repositorio)
- **Escopo:** instruido automaticamente em todas as conversas do Copilot Chat no repositorio
- **Este arquivo (`COPILOT.md`):** nao e carregado automaticamente — serve como documentacao de governanca para leitura humana e referencia via `#file:COPILOT.md` no chat

Para referenciar este arquivo explicitamente no Copilot Chat: `#file:COPILOT.md`

### gh copilot CLI (`gh copilot suggest` / `gh copilot explain`) — Sem Suporte Nativo

O `gh copilot` CLI **nao le** `.github/copilot-instructions.md` nem qualquer arquivo de contexto automaticamente. Cada invocacao e stateless.

**Workaround recomendado para o CLI:**
```bash
# Incluir contexto manualmente no prompt
gh copilot suggest "$(cat .github/copilot-instructions.md)\n\nTarefa: <descricao>"

# Ou via pipe para tarefas de explicacao
cat internal/install/install.go | gh copilot explain
```

## Limitações Conhecidas

Em comparacao com Claude Code (CLAUDE.md), Gemini CLI (GEMINI.md) e Codex (CODEX.md):

| Capacidade | Claude Code | Gemini CLI | Codex | Copilot Chat | gh copilot CLI |
|---|---|---|---|---|---|
| Arquivo de contexto automatico | CLAUDE.md | GEMINI.md | AGENTS.md | `.github/copilot-instructions.md` | **Nao suportado** |
| Hooks de pre/pos execucao | Sim (`.claude/settings.json`) | Sim (`.gemini/hooks/`) | Sim (`.codex/hooks/`) | Nao | Nao |
| Sistema de skills/comandos | Sim (`.claude/skills/`) | Sim (`.gemini/commands/`) | Sim (`.codex/config.toml`) | Nao | Nao |
| Estado de sessao persistente | Sim | Sim | Sim | Sim (por sessao de chat) | **Nao — stateless** |
| Carregamento sob demanda de refs | Sim | Sim | Sim | Manual (`#file:`) | **Nao** |

**Resumo das limitacoes:**
1. `gh copilot` CLI nao suporta arquivo de contexto automatico — contexto deve ser injetado manualmente.
2. Nao ha mecanismo de hooks (pre/pos execucao) equivalente ao `.claude/settings.json` ou `.gemini/hooks/`.
3. Nao ha sistema de skills ou comandos customizados — o `gh copilot suggest` aceita apenas prompt livre.
4. Sessoes do `gh copilot` CLI sao stateless — sem memoria de contexto entre invocacoes.
5. Copilot Chat (VS Code/GitHub.com) suporta `.github/copilot-instructions.md` automaticamente, mas nao tem extensibilidade via hooks ou skills.
6. Enforcement de governanca depende exclusivamente do usuario seguir instrucoes procedurais — nao ha bloqueio automatico.
