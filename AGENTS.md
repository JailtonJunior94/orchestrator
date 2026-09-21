# Regras para Agentes de IA

`ai-spec-harness` e uma CLI em Go que instala, valida, inspeciona e atualiza governanca operacional
para ferramentas de IA (Claude Code, Codex, GitHub Copilot, OpenCode) em repositorios de software.
O modulo Go chama-se `ai-spec-harness`; o binario publicado chama-se **`ai-spec`**. Este arquivo e a
fonte canonica para todos os agentes; `CLAUDE.md`, `CODEX.md` e `COPILOT.md` sao suplementos.

## Modo de trabalho

1. Entender o contexto antes de editar qualquer arquivo.
2. Preferir a menor mudanca segura que resolva a causa raiz.
3. Preservar arquitetura, convencoes e fronteiras existentes.
4. Nao introduzir abstracoes ou dependencias sem demanda concreta.
5. Atualizar testes quando houver mudanca de comportamento.
6. Rodar validacoes proporcionais a mudanca.
7. Registrar bloqueios e suposicoes quando o contexto estiver incompleto.
8. Evitar reescritas amplas e overengineering; risco de regressao e restricao principal.

## Contrato de carga base

Toda skill que altera codigo deve carregar, como primeiro passo, a seguinte base obrigatoria — essa
instrucao e reforcada em cada SKILL.md como medida defensiva:

1. Ler este `AGENTS.md`.
2. Ler `.agents/skills/agent-governance/SKILL.md`.

Essa base define governanca para analise, alteracao e validacao, carregamento sob demanda de regras
de DDD, erros, seguranca e testes, e criterios minimos de preservacao arquitetural, risco e validacao
proporcional. Skills individuais declaram apenas cargas adicionais especificas ao seu contexto.

## Regras por Linguagem

| Linguagem | Skill a carregar |
|-----------|-----------------|
| Go | `.agents/skills/go-implementation/SKILL.md` |
| Node/TypeScript | `.agents/skills/node-implementation/SKILL.md` |
| Python | `.agents/skills/python-implementation/SKILL.md` |
| .NET/C# | `.agents/skills/dotnet-csharp-implementation/SKILL.md` |
| Revisao/refatoracao Go (OC) | `.agents/skills/object-calisthenics-go/SKILL.md` |
| Correcao de bugs | `.agents/skills/bugfix/SKILL.md` |

## Invariantes de Governanca (obrigatorias)

1. **Protocolo PRD-First:** toda alteracao de comportamento ou nova funcionalidade inicia com criacao ou atualizacao de um PRD (`create-prd`). Proibido implementar codigo sem requisito funcional (RF) mapeado.
2. **Ancora de confianca (spec-hash):** a integridade Requisito -> Arquitetura -> Implementacao e garantida por SHA-256. Ao editar um PRD, rodar `ai-spec sync-spec-hash`. `execute-task` e `execute-all-tasks` validam drift via `ai-spec check-spec-drift` e interrompem em caso de inconsistencia.
3. **Isolamento de contexto:** operar com o minimo de contexto necessario. Subagentes sao obrigatorios para tarefas de execucao.
4. **Evidencia obrigatoria:** tarefa so e `done` apos persistir `execution_report.md` com evidencias fisicas (logs, testes, outputs) da validacao.

## Governanca por Ferramenta

| Arquivo | Ferramenta |
|---------|-----------|
| `CLAUDE.md` | Claude Code — importa este arquivo via `@AGENTS.md`; hooks, rules, agents em `.claude/` |
| `CODEX.md` | Codex — `.codex/config.toml`, instrucao de sessao |
| `COPILOT.md` | GitHub Copilot — contexto via `.github/copilot-instructions.md` |
| — | OpenCode carrega `AGENTS.md` e `.agents/skills/` nativamente; plugin de governanca em `.opencode/plugin/` (RF-13) |

A fonte de verdade dos fluxos procedurais permanece em `.agents/skills/`. Cada skill lista suas
`references/` com gatilhos de carregamento no proprio `SKILL.md`; nao duplicar aqui.

## Stack

- **Linguagem:** Go 1.27 (`go.mod`: `go 1.27.1`)
- **CLI:** spf13/cobra + pflag
- **Dependencias diretas:** coder/acp-go-sdk (runtime ACP), pkoukk/tiktoken-go, santhosh-tekuri/jsonschema/v6, gopkg.in/yaml.v3, stretchr/testify (testes)
- **Testes:** `go test` — unit com FakeFileSystem, integration com build tag `integration`
- **Release:** GoReleaser + GitHub Actions + Homebrew Formula (`brews:` em `.goreleaser.yaml`)

## Comandos

```bash
make test            # testes unitarios
make integration     # testes de integracao (build tag integration)
make lint            # golangci-lint
make vet             # go vet
make build           # compila ./ai-spec
make coverage        # cobertura total (gate 75%) + por pacote critico (gate 70%)
make bench           # benchmarks
```

Gates que um agente deve rodar quando tocar a area correspondente:

| Area tocada | Gate |
|-------------|------|
| `.agents/skills/`, `.agents/lib/`, `.agents/hooks/` | `make check-skills-sync check-hooks-sync` (espelhos em `.claude/`, `.github/`, `internal/embedded/`) |
| `.agents/scripts/` (validadores) | `make check-scripts-sync test-validators` |
| `.agents/policies/` | `make check-policies-sync` (espelhos em `.claude/rules/` e equivalentes) |
| `.claude/hooks/`, `.agents/hooks/` | `make test-hooks` |
| `prd.md`, `techspec.md`, `tasks.md` de PRD sob SDD | `make check-spec-paths` |
| Mocks (`mockery.yml`) | `make check-mocks` |

Comandos `ai-spec` do fluxo SDD (todos registrados em `cmd/ai_spec_harness/root.go`):

```bash
ai-spec install . [--tools claude,codex,copilot,opencode] [--global] [--mode copy] [--dry-run]
ai-spec verify . | ai-spec verify --global      # current/missing/drifted por skill/agente
ai-spec sync-spec-hash | ai-spec check-spec-drift | ai-spec validate-sdd
ai-spec task-loop --tool <claude|codex|copilot|opencode> --runtime acp .specs/prd-X
ai-spec seal-evidence .specs/prd-X/result.json --prd-dir .specs/prd-X [--verify]
ai-spec telemetry report | ai-spec memory status | ai-spec lint .
```

## Convencoes

| Aspecto | Regra |
|---------|-------|
| Idioma (codigo) | **Ingles obrigatorio** em identificadores, pacotes, mensagens de erro, nomes de teste e de arquivo. Ver `.claude/rules/code-style.md` (R-STYLE-001, hard) |
| Comentarios | **Zero comentarios no codigo** (linha, bloco e doc-comments). Regra hard, inegociavel |
| Idioma (docs) | PT-BR em artefatos `.md`, relatorios, ADRs, PRDs, changelog |
| Commits | Conventional Commits: tipo em ingles, corpo em portugues |
| Testes | table-driven; FakeFileSystem (unit); `t.TempDir()` (integration) |
| DI | injetar via construtor; zero estado global |
| Erros | `fmt.Errorf("context: %w", err)` — texto em ingles |
| Pacotes | um por responsabilidade em `internal/`; interfaces no pacote consumidor |

## Estrutura

```
main.go, cmd/ai_spec_harness/   comandos Cobra, um arquivo por subcomando
internal/                        um pacote por responsabilidade; pontos de entrada:
  fs/, output/                   FakeFileSystem e Printer — injetados em todo Service
  config/                        cascata de configuracao (resolver.go)
  install/, uninstall/, upgrade/, detect/, contextgen/ (gera AGENTS.md/CLAUDE.md alvo)
  embedded/assets/               assets distribuidos via go:embed (ADR-001)
  runtime/                       ACP runner, hooks Go, memory, mcpserver, events
  taskloop/, sdd/, specdrift/, evidence/, traceability/, approval/, parity/, lint/, telemetry/
.agents/skills|scripts|hooks|lib fonte canonica (espelhada em .claude/, .github/, internal/embedded/)
.specs/                          PRDs, techspecs, tasks; .specs/adr/ ADRs 009+
docs/                            guias; docs/adr/ ADRs 000-008
testdata/, tests/, evals/        fixtures, snapshots, integracao, evals SDD
```

## Validadores de evidencia

- Canonicos em `.agents/scripts/`, espelhados em `.claude/scripts/` e `internal/embedded/assets/` via `scripts/sync-skills.sh`. Skills resolvem em cascata `.agents/scripts/` -> `.claude/scripts/` -> `scripts/`.
- `validate-task-evidence.sh` (DoD + criterios de aceite + prova de testes), `validate-bugfix-evidence.sh` (`--no-rf` para opt-out), `validate-refactor-evidence.sh`, `validate-review-evidence.sh`.
- Gate de criterios de aceite e **fail-closed** desde 0.31.0. `AI_SDD_STRICT_EVIDENCE=0` reabre o legado apenas para migracao e e ruidoso de proposito.
- Regex nos validadores: nunca `Crit[eé]rios` (falha silenciosa em mawk); usar `Crit(e|é)rios`. `scripts/test-validators.sh` caso "a2" trava isso sob `LC_ALL=C`.
- `SKILL.md` pode declarar `category: governance|language|processual`; `governance`/`language` sao auto-carregadas.
- Racional, historico (BUG-127, `check-spec-paths`) e selo de evidencia: [`docs/evidence-gates.md`](docs/evidence-gates.md).

## CI

| Workflow | Gatilho | Conteudo |
|----------|---------|----------|
| `test.yml` | push/PR | jobs `unit` + `lint` (matriz ubuntu-24.04, macos-15), `integration`, `sdd-evals`; cobertura 75% total e 70% por pacote critico (`scripts/check-package-coverage.sh`) |
| `codeql.yml` | push main / PR | analise CodeQL |
| `release.yml` | apos `Tests` concluir em `main` | semver-next automatico, GoReleaser multi-plataforma |
| `release-dry-run.yml` | manual (`workflow_dispatch`) | validacao de release sem side-effects |
| `acp-live.yml`, `hooks-live.yml` | nightly / manual | testes ao vivo contra CLIs reais; nao sao gate de merge |
| `test-setup-action.yml` | PR em `.github/actions/setup-ai-spec/` | smoke da action |

## Configuracao

Precedencia deterministica: `flags CLI > workspace (.aispec/config.yaml > .claude/config.yaml > .agents/config.yaml, upward-walk) > global (~/.aispec/config.yaml) > defaults`.
Chaves: `timeout`, `max_retries`, `retry_backoff_multiplier`, `concurrent`, `batch_size`, `default_tool`, `durable_memory_enabled`. Zero-value preserva o comportamento F1. Ver [`docs/config-hierarchy.md`](docs/config-hierarchy.md).

## Padroes Importantes

- `internal/fs/fake.go` — FakeFileSystem em testes unitarios, nunca OS real (ADR-002)
- `internal/output.Printer` — injetar em todo Service; `io.Discard` em testes
- Skills externas rastreadas em `skills-lock.json` com hash SHA-256 (ADR-005); todo bump exige registro em `audit/` pelo template `.specs/templates/skill-upgrade-decision.md` (skill, versao anterior, versao nova, motivador, criterio de aceitacao, data)
- Harness auto-instalado neste repo (self-dogfooding); manifesto em `.ai_spec_harness.json`

## Restricoes

Nao inventar contexto ausente. Nao assumir versao sem verificar. Nao alterar comportamento publico
sem registrar. Nao executar git destrutivo nem publicar remoto sem pedido explicito. Adaptar
exemplos ao contexto real.

## Documentacao

- [Guia de Instalacao Universal](docs/guia-instalacao-universal.md) · [Hierarquia de Configuracao](docs/config-hierarchy.md) · [Troubleshooting](docs/troubleshooting.md)
- [Referencia do task-loop](docs/task-loop-reference.md) · [Capacidades do runtime Claude](docs/runtime-claude-capabilities.md) · [Gates de evidencia](docs/evidence-gates.md)
- [Ciclo de telemetria](docs/telemetry-feedback-cycle.md) · [Matriz de degradacao](docs/degradation-matrix.md)
- [Hooks canonicos, eventos e policies](docs/hooks-canonicos.md) — inclui a semantica exata de `AfterTool` (informa apos a edicao, nunca bloqueia previamente) referenciada por `CLAUDE.md`.

## ADRs

Template: [`docs/adr/000-template.md`](docs/adr/000-template.md). Consultar antes de mudancas estruturais.

| ADR | Titulo | Status |
|-----|--------|--------|
| [001](docs/adr/001-go-embed-baseline.md) | Assets via go:embed | Aceita |
| [002](docs/adr/002-fake-filesystem-testes.md) | FakeFileSystem vs afero | Aceita |
| [003](docs/adr/003-paridade-semantica.md) | Invariantes semanticas vs diff textual | Aceita |
| [004](docs/adr/004-lazy-loading-referencias.md) | References sob demanda | Aceita |
| [005](docs/adr/005-skills-lock-sha256.md) | Lock file SHA-256 | Aceita |
| [006](docs/adr/006-telemetria-feedback-cycle.md) | Telemetria opt-in append-only | Aceita |
| [007](docs/adr/007-copilot-cli-stateless-workaround.md) | Copilot injecao manual | Substituida por 012 |
| [008](docs/adr/008-parity-multi-tool-invariants.md) | 29 invariantes 3 niveis | Aceita |
| [009](.specs/adr/009-acp-protocol-adoption.md) | Adocao do ACP via coder/acp-go-sdk | Aceita |
| [011](.specs/adr/011-agent-registry-declarativo.md) | Agent registry declarativo | Proposta |
| [012](.specs/adr/012-copilot-cli-acp-native.md) | Copilot CLI como runtime ACP nativo | Aceita |
| [013](.specs/adr/013-codex-cli-acp-native.md) | Codex CLI como runtime ACP nativo | Proposta |
| [014](.specs/adr/014-claude-cli-acp-native.md) | Claude CLI como runtime ACP nativo | Proposta |
| [015](.specs/adr/015-gemini-cli-acp-native.md) | Gemini CLI ACP nativo — agente removido | Substituida |
| [020](.specs/adr/020-opencode-acp-subcomando.md) | OpenCode ACP via subcomando | Aceita |
| [MD-001](.specs/prd-memoria-duravel-agentes/adr-001-fachada-porta-unica-memoria.md) | Fachada como porta unica do runtime para memoria duravel | Proposta |
| [MD-002](.specs/prd-memoria-duravel-agentes/adr-002-fato-pagina-roundtrip-lossless.md) | Fato por chave semantica + hash; Pagina Markdown round-trip lossless | Proposta |
| [MD-003](.specs/prd-memoria-duravel-agentes/adr-003-escrita-atomica-lock-camada-lease.md) | Escrita atomica, lock por camada, lease de bastao | Proposta |
| [MD-004](.specs/prd-memoria-duravel-agentes/adr-004-optin-paridade-byte-a-byte.md) | Opt-in + paridade byte-a-byte + fim da degradacao silenciosa | Proposta |
| [MD-005](.specs/prd-memoria-duravel-agentes/adr-005-evidencia-metricas-memoria.md) | Eventos pelo dispatcher existente, metricas pelo mapa de campos extra | Proposta |

ADR-010, 016-019, PP-001 e PP-002 foram removidos do repositorio no commit `91f8cb9`
(diretorios `.specs/prd-acp-runtime-claude/`, `.specs/prd-fundacao-portatil/`,
`.specs/prd-skills-production-proof/`); o conteudo sobrevive apenas no historico git.
