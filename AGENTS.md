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
| .NET/C# | `.agents/skills/dotnet-csharp-implementation/SKILL.md` |
| Revisão/refatoração Go (OC) | `.agents/skills/object-calisthenics-go/SKILL.md` |
| Correção de bugs | `.agents/skills/bugfix/SKILL.md` |

## Invariantes de Governança (Obrigatórias)

Para garantir a confiabilidade em qualquer projeto instrumentado por este harness, as seguintes regras são **mandatórias** e não admitem desvios:

1. **Protocolo PRD-First:** Toda e qualquer alteração de comportamento ou nova funcionalidade deve obrigatoriamente iniciar com a criação ou atualização de um PRD (`create-prd`). É proibido implementar código sem um requisito funcional (RF) mapeado.
2. **Âncora de Confiança (Spec-Hash):** A integridade entre Requisito -> Arquitetura -> Implementação é garantida por hashes SHA-256. 
   - Ao editar um PRD, você deve sincronizar os hashes rodando `ai-spec sync-spec-hash`.
   - As skills de execução (`execute-task`, `execute-all-tasks`) devem validar o drift via `ai-spec check-spec-drift` e interromper a execução em caso de inconsistência.
3. **Isolamento de Contexto:** Agentes devem operar com o mínimo de contexto necessário. O uso de subagentes para tarefas de execução é obrigatório para evitar a poluição da sessão principal e minimizar alucinações.
4. **Evidência Obrigatória:** Uma tarefa só é considerada concluída (`done`) após a persistência de um relatório de execução (`execution_report.md`) que contenha evidências físicas (logs, testes, outputs) da validação.

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
de decisão salvo em `audit/`. Use o template em `.specs/templates/skill-upgrade-decision.md`.

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

### Instalacao e Verificacao (Fundacao Portatil)

```bash
# Instalar em um projeto (auto-deteccao de agentes)
ai-spec-harness install .

# Instalar selecionando ferramentas manualmente
ai-spec-harness install . --tools claude,gemini

# Instalar globalmente em ~/.aispec (escopo global, opt-in)
ai-spec-harness install --global

# Verificar estado dos assets instalados (current/missing/drifted)
ai-spec-harness verify .
ai-spec-harness verify --global

# Instalar em modo copy (sem symlinks)
ai-spec-harness install . --mode copy

# Simular sem executar
ai-spec-harness install . --dry-run
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
.agents/scripts/   # validadores de evidencia canonicos (tool-neutros)
.agents/hooks/     # hooks do orquestrador (canonico)
.agents/lib/       # shell libs vendoradas
internal/embedded/
```

### Validadores de evidencia (paridade cross-CLI)

Os validadores canonicos vivem em `.agents/scripts/` (tool-neutros) e sao espelhados para
`.claude/scripts/` e `internal/embedded/assets/{.claude,.agents}/scripts/` via `scripts/sync-skills.sh`
(gate: `make check-scripts-sync`). O instalador copia-os para `.agents/scripts/` do projeto destino
**sempre** (independente dos tools), garantindo que projetos so-Gemini/Codex/Copilot tenham os
mesmos gates que Claude. As skills resolvem em cascata `.agents/scripts/` -> `.claude/scripts/` -> `scripts/`.

- `validate-task-evidence.sh` — gate anti-falso-positivo (DoD + cada criterio de aceite + prova forte de testes).
- `validate-bugfix-evidence.sh` — rastreabilidade de origem default-on (`--no-rf` para opt-out).
- `validate-refactor-evidence.sh` — evidencia de nao-regressao.
- `validate-review-evidence.sh` — evidencia do modo `--auto-review` (veredito + severidade).

### Selo de evidencia (RF-14)

A prova de fechamento e verificada contra a arvore de trabalho viva, que deixa de existir quando o
trabalho e commitado — por isso ela nao e re-auditavel depois. `ai-spec seal-evidence` fecha essa
lacuna gravando `commit_sha` e `commit_patch_sha256` (o patch recomputado em `base..commit` com as
mesmas exclusoes do fechamento):

```bash
ai-spec seal-evidence .specs/prd-x/result.json --prd-dir .specs/prd-x   # selar apos commitar
ai-spec seal-evidence .specs/prd-x/result.json --prd-dir .specs/prd-x --verify
```

O selo exige que o commit descenda da base registrada e recusa reselagem. A verificacao nao toca a
arvore de trabalho, entao permanece valida indefinidamente. Limite: o selo torna a evidencia
imutavel e reverificavel dali em diante, mas nao prova que o commit e byte-identico a arvore do
fechamento — essa arvore ja nao existe quando o selo e aplicado.

`AI_SDD_STRICT_EVIDENCE=1` fecha os escapes de compatibilidade de
`validate-task-evidence.sh` (NFR-01): um relatorio cuja task file nao seja resolvivel pelo campo
`Arquivo:`, ou cuja task nao declare secao de criterios, passa a falhar em vez de emitir aviso.
Sem a variavel o comportamento warning-only da janela de compatibilidade e preservado. Essa
janela fecha em **v0.31.0**: NFR-01 concede warning-only por duas versoes menores, o fluxo SDD
entrou em `0.29.0`, e a partir de `0.31.0` o modo estrito passa a ser o padrao. O validador ja
anuncia o prazo em toda execucao que usa o escape.

As expressoes regulares desses validadores nao podem usar classes de bracket com caracteres
multibyte (`Crit[eé]rios`): em `awk` byte-oriented (mawk, padrao nos runners Linux) elas nunca
casam e o gate se desliga silenciosamente. Usar alternacao (`Crit(e|é)rios`). O caso "a2" de
`scripts/test-validators.sh` trava essa invariante executando o gate sob `LC_ALL=C`.

### Metadado `category` no frontmatter

Cada SKILL.md pode declarar `category: governance|language|processual`. `governance`/`language` sao
auto-carregadas em runtime; `processual` (ou ausente) e declarada por tarefa. `create-tasks` deriva a
lista de skills auto-carregadas desse metadado, nao de prosa hardcoded.

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

## Configuracao

O harness usa hierarquia de config em cascata com precedência determinística:

```
flags CLI  >  workspace (.claude/config.yaml)  >  global (~/.aispec/config.yaml)  >  defaults built-in
```

- **Config global:** `~/.aispec/config.yaml` — defaults reutilizaveis entre projetos (opt-in).
- **Config de projeto:** descoberta por upward-walk a partir do CWD; candidatos: `.aispec/config.yaml` > `.claude/config.yaml` > `.agents/config.yaml`.
- **Compatibilidade:** sem config global e a partir da raiz do repo, comportamento identico ao atual (F1, zero regressao).

Chaves operacionais aceitas (zero-value preserva comportamento F1):
`timeout`, `max_retries`, `retry_backoff_multiplier`, `concurrent`, `batch_size`, `default_tool`.

Ver [`docs/config-hierarchy.md`](docs/config-hierarchy.md) para referencia completa.

## Documentacao

- [Guia de Instalacao Universal](docs/guia-instalacao-universal.md) — bootstrap portatil em qualquer codebase, deteccao automatica, escopo global, verify
- [Hierarquia de Configuracao](docs/config-hierarchy.md) — precedencia flags > workspace > global > built-in, upward-walk, chaves disponíveis
- [Guia de troubleshooting](docs/troubleshooting.md) — problemas comuns com sintoma, causa, solucao e verificacao
- [Ciclo de telemetria](docs/telemetry-feedback-cycle.md) — feedback loop com GOVERNANCE_TELEMETRY

## ADRs

| ADR | Titulo | Status |
|-----|--------|--------|
| [001](.specs/adr/001-go-embed-baseline.md) | Assets via go:embed | Aceita |
| [002](.specs/adr/002-fake-filesystem-testes.md) | FakeFileSystem vs afero | Aceita |
| [003](.specs/adr/003-paridade-semantica.md) | Invariantes semanticas vs diff textual | Aceita |
| [004](.specs/adr/004-lazy-loading-referencias.md) | References sob demanda | Aceita |
| [005](.specs/adr/005-skills-lock-sha256.md) | Lock file SHA-256 | Aceita |
| [006](docs/adr/006-telemetria-feedback-cycle.md) | Telemetria opt-in append-only | Aceita |
| [007](docs/adr/007-copilot-cli-stateless-workaround.md) | Copilot injecao manual | Substituida por ADR-012 |
| [008](docs/adr/008-parity-multi-tool-invariants.md) | 29 invariantes 3 niveis | Aceita |
| [ADR-009](.specs/adr/009-acp-protocol-adoption.md) | Adocao do ACP via coder/acp-go-sdk para invocacao de agentes | Aceita |
| [ADR-010](.specs/prd-acp-runtime-claude/adr-010-event-tagged-union.md) | runtime.Event como struct tagged union | Aceita |
| [ADR-012](.specs/adr/012-copilot-cli-acp-native.md) | Copilot CLI como runtime ACP nativo | Aceita |
| [ADR-013](.specs/adr/013-codex-cli-acp-native.md) | Codex CLI como runtime ACP nativo | Aceita |
| [ADR-015](.specs/adr/015-gemini-cli-acp-native.md) | Gemini CLI ACP nativo (ADR-015) | Proposta |
| [ADR-016](.specs/prd-fundacao-portatil/adr-016-config-hierarquico-universal.md) | Config hierarquico universal (global+projeto, upward-walk, precedencia) | Proposta |
| [ADR-017](.specs/prd-fundacao-portatil/adr-017-fallback-launcher-chain.md) | Generalizacao de fallback launchers (cadeia generica ordenada) | Proposta |
| [ADR-018](.specs/prd-fundacao-portatil/adr-018-runtimeconfig-retry-backpressure.md) | RuntimeConfig unificado, retry com backoff e sessao ACP com backpressure observavel | Proposta |
| [ADR-019](.specs/prd-fundacao-portatil/adr-019-instalador-portatil-detect-verify.md) | Instalador portatil: auto-deteccao de agentes, escopo global e verify file-first | Proposta |
| [PP-001](.specs/prd-skills-production-proof/adr-001-validadores-canonicos-agents-scripts.md) | Validadores de evidencia canonicos em `.agents/scripts/` (tool-neutros, cascata) | Aceita |
| [PP-002](.specs/prd-skills-production-proof/adr-002-hooks-nativos-paridade-cross-cli.md) | Hooks nativos de bloqueio nos 4 CLIs (paridade cross-CLI 2026) | Aceita |
