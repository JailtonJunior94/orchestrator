<!-- spec-hash-prd: bb49d3f0bee58af3d4c8361265864995a94e2f11698da0e410f3fe1bd9b7f349 -->
<!-- MANDATÓRIO: preenchido por `create-technical-specification` Etapa 7.1 com sha256 do PRD consumido.
     Rastreabilidade: `create-tasks` e `execute-task` comparam este hash com o atual do prd.md
     para detectar drift entre techspec e PRD. NÃO remover este comentário ao editar a techspec. -->

# Especificação Técnica — Skills Production-Proof + Paridade Cross-CLI 2026

## Resumo Executivo

A entrega endurece a cadeia de 7 skills de governança até production-proof por meio de
**adições cirúrgicas** a três superfícies: (1) os **validadores shell** de evidência, (2) os
**arquivos SKILL.md** das skills e suas referências canônicas, e (3) os **arquivos de
configuração de hooks** de cada CLI. Não há reescrita de skill nem mudança no núcleo de
runtime Go.

A decisão arquitetural central é mover a fonte de verdade dos validadores de
`.claude/scripts/` para **`.agents/scripts/`** (tool-neutro, portátil), espelhá-la para os
mirrors por-tool e o embedded, e fazer todas as skills resolverem o validador em **cascata**
`.agents/scripts/` → `.claude/scripts/` → `scripts/` — espelhando o padrão já consolidado de
`.agents/lib/` → `scripts/lib/`. Sobre essa base, cada CLI ganha um **hook nativo de
`PreToolUse`/`Stop`** (formato próprio) que invoca o mesmo validador, transformando paridade
"por instrução" em paridade "por enforcement". O Codex é suplementado por `sandbox_mode` +
`approval_policy` por causa da lacuna de route-around documentada nas docs oficiais 2026.

## Arquitetura do Sistema

### Visão Geral dos Componentes

**Validadores (shell, tool-agnósticos) — canônico em `.agents/scripts/`**
- `validate-task-evidence.sh` — estendido: gate de Critérios de Aceite + DoD + rejeição de
  `Testes: pass` sem prova (RF-01..RF-03).
- `validate-bugfix-evidence.sh` — path resolvido em cascata + RF default-on (RF-17, RF-18).
- `validate-review-evidence.sh` — **novo**: valida evidência de `--auto-review` (RF-20).

**Hooks de orquestração (shell) — canônico em `.agents/hooks/`**
- `post-execute-task.sh` — `AI_VALIDATE_GIT_HISTORY` default `1` (RF-04).

**Configs de hooks nativos por-tool (novos)**
- `.claude/settings.json` (`hooks.PreToolUse`/`Stop`/`SubagentStop`).
- `.codex/hooks.json` + bloco `[[hooks.PreToolUse]]` em `.codex/config.toml` + `sandbox_mode`/`approval_policy`.
- `.github/hooks/governance.json` (`version:1`, `preToolUse`/`agentStop`).
- `.gemini/settings.json` (`hooks.BeforeTool`/`AfterAgent`).

**Skills (markdown) e referências**
- `agent-governance/references/multiple-choice-protocol.md` — **novo** (RF-06).
- `agent-governance/references/severity-mapping.md` — **novo** (RF-15).
- `agent-governance/references/enforcement-matrix.md` — reescrito (RF-12).
- `create-prd`, `create-technical-specification`, `create-tasks`, `review`,
  `execute-task`, `execute-all-tasks`, `bugfix` — SKILL.md editados.
- `task-execution-report-template.md` — seção Critérios de Aceite/DoD (RF-01).

**Sync/gate**
- `scripts/sync-skills.sh` estendido para espelhar `.agents/scripts/` (RF-09).
- `scripts/check-scripts-sync.sh` — **novo** gate de drift (RF-13).

### Fluxo de dados (enforcement por hook)

```
agente (qualquer CLI) tenta tool-call/encerrar tarefa
  -> hook nativo do tool dispara (PreToolUse / Stop / agentStop / AfterAgent)
  -> hook invoca validador compartilhado resolvido em cascata (.agents/scripts/ -> ...)
  -> validador confronta critérios de aceite + DoD + DiffSHA + RF
  -> deny/exit!=0 bloqueia conclusão; mensagem acionável devolvida ao agente
```

## Design de Implementação

### Gate de Critérios de Aceite (`validate-task-evidence.sh`)

Contrato de extração: o validador recebe o report e **resolve a task file** a partir do campo
`Arquivo:` do report. Extrai os itens da seção `## Critérios de Aceite` da task file (linhas
`- [ ]`/`- ` ou `AC-nn:`) e exige que cada um apareça como **atendido** na seção
`## Critérios de Aceite` do report (status `atendido`/`[x]` + evidência referenciada).

```sh
# pseudo-shell
task_file="$(grep -E '^- Arquivo:' "$report" | sed 's/^- Arquivo:[[:space:]]*//')"
# extrair ACs da task (entre '## Critérios de Aceite' e próximo heading)
# para cada AC: exigir linha correspondente marcada 'atendido' no report
# falhar (missing=1) se algum AC não comprovado
```

Rejeição de prova fraca (RF-03): além de `Testes: pass`, exigir **pelo menos uma** evidência
associada na seção `## Comandos Executados` (um comando de teste com resultado) quando
`Testes: pass`. Sem comando correspondente → `FALTANDO: evidência de execução de testes`.

Compatibilidade: quando a task file **não declara** `## Critérios de Aceite` (tasks legadas),
emitir aviso não-fatal e manter as checagens atuais — zero regressão (RF-22).

### DiffSHA default-on (`post-execute-task.sh`)

`AI_VALIDATE_GIT_HISTORY:-1` (era `:-0`). Mantém o opt-out explícito via `=0`. O bloco F35
existente permanece; apenas o default muda. Testar contagem de eventos inalterada.

### Cascata de validadores (RF-09, RF-17)

Helper de resolução reutilizável (inline nas skills, igual ao de `check-invocation-depth.sh`):

```sh
_resolve_validator() { # $1 = nome do script
  for d in "$repo_root/.agents/scripts" "$repo_root/.claude/scripts" "$repo_root/scripts"; do
    [[ -f "$d/$1" ]] && { echo "$d/$1"; return 0; }
  done
  return 1
}
```

`bugfix/SKILL.md` Etapa 5.5 passa a usar este helper (em vez do path rígido `.claude/scripts/`).

### Hooks nativos por-tool (RF-10, RF-11)

Todos chamam o mesmo alvo: `bash "$(resolve .agents/scripts/validate-task-evidence.sh)" <report>`
no encerramento, e um guard de `PreToolUse` que valida governança (AGENTS.md presente) antes de
edições. Formatos (das docs oficiais 2026):

- **Claude** `.claude/settings.json`:
  ```json
  { "hooks": { "Stop": [ { "hooks": [ { "type": "command",
    "command": "${CLAUDE_PROJECT_DIR}/.agents/hooks/post-execute-task.sh" } ] } ] } }
  ```
- **Copilot** `.github/hooks/governance.json` (`version:1`, campos `bash`/`powershell`):
  ```json
  { "version": 1, "hooks": { "agentStop": [ { "type": "command",
    "bash": "./.agents/hooks/post-execute-task.sh" } ] } }
  ```
- **Gemini** `.gemini/settings.json` (`hooks.AfterAgent` + `BeforeTool`):
  ```json
  { "hooks": { "AfterAgent": [ { "hooks": [ { "type": "command",
    "command": "$GEMINI_PROJECT_DIR/.agents/hooks/post-execute-task.sh" } ] } ] } }
  ```
- **Codex** `.codex/hooks.json` (`PreToolUse`) + `.codex/config.toml`:
  ```toml
  approval_policy = "on-request"
  sandbox_mode = "workspace-write"
  [[hooks.PreToolUse]]
  matcher = "^(Bash|shell)$"
  [[hooks.PreToolUse.hooks]]
  type = "command"
  command = 'bash "$(git rev-parse --show-toplevel)/.agents/hooks/validate-governance.sh"'
  ```
  Comentário inline documentando que o sandbox/approval cobre a lacuna de route-around.

### Comportamento de hook ausente (RF-05)

`execute-task`/`execute-all-tasks` deixam de degradar para "modo legado" silencioso. Regra:
se o tool é capaz (config de hook presente para o tool ativo via `AI_TOOL`) e o `.sh` não
roda/retorna falha → estado `failed`/`blocked` com mensagem. Sem o config para o tool → aviso
visível (não silencioso), mantendo a validação textual como fallback explícito.

### Protocolo de múltipla escolha (RF-06..RF-08)

`multiple-choice-protocol.md`: 2–5 opções, uma pergunta por turno, marcar "(Recomendado)" na
primeira, gatilho = ambiguidade material. Cada skill alvo ganha uma linha na etapa de
esclarecimento: "Quando houver ambiguidade material, aplicar
`agent-governance/references/multiple-choice-protocol.md`".

### Tabela de severidade (RF-15)

`severity-mapping.md`: `critical→critical`, `high→major`, `medium→minor`, `low→minor`.
`review/SKILL.md` referencia ao emitir bugs no `bug-schema.json`; `bugfix` referencia ao
consumir.

## Pontos de Integração

- **Binário `ai-spec`**: usado para `hash`/`sync-spec-hash`/`check-spec-drift` (já no PATH).
- **CLIs externas (Codex/Gemini)**: validação empírica de subagentes (RF-16) depende de
  binário disponível; sem ele, registrar suposição no report e documentar execução inline.

## Abordagem de Testes

### Testes Unitários

- **Shell (validadores)**: estender `scripts/test-hooks.sh` com casos: (a) AC não comprovado →
  exit 1; (b) AC comprovado → exit 0; (c) `Testes: pass` sem comando → exit 1; (d) task legada
  sem seção AC → exit 0 com aviso. Fixtures em `testdata/`.
- **Go**: nenhum código Go novo no caminho principal; se `AI_VALIDATE_GIT_HISTORY` afetar
  contagem de eventos, teste de contagem em `internal/runtime/...` (FakeFileSystem).

### Testes de Integração

- `make integration`: rodar a cadeia em `t.TempDir()` com repo só-`.agents/` confirmando que a
  cascata resolve os validadores e o gate de aceite bloqueia (RF-09, RF-22).

### Testes E2E

- Validação de portabilidade manual/script: cadeia completa convergindo idêntica nos 4 CLIs
  (RF-22) — registrada como evidência da última tarefa.
- **Matriz de projetos (RF-23)**: exercitar a cadeia em fixtures de projeto **pequeno, médio e
  grande**, **novo e existente**, confirmando gates, paridade cross-CLI e ausência de acoplamento
  a paths do `orchestrator`. Resolução sempre por descoberta agnóstica (`ls .agents/skills/`,
  frontmatter, env exportadas) + cascata.

## Sequenciamento de Desenvolvimento

### Ordem de Build

1. Cascata + canônico em `.agents/scripts/` (base para tudo).
2. Gate de aceite no template + validador + DiffSHA default-on.
3. Referência de múltipla escolha + integração nas skills.
4. Hooks nativos por-tool + matriz + sync gate.
5. Sinergia (review/severidade/subagentes/bugfix path).
6. Economia (RF default-on, skills via metadado, validador de review, budget/kill).
7. Sync mirrors + suíte + portabilidade.

### Dependências Técnicas

- `scripts/sync-skills.sh` deve aprender a espelhar `.agents/scripts/` antes do gate de sync.

## Monitoramento e Observabilidade

- Reaproveitar telemetria opt-in (`GOVERNANCE_TELEMETRY=1`); registrar disparo de gate de
  aceite e de hook bloqueante como entries existentes do relatório, sem novos canais.

## Considerações Técnicas

### Decisões Chave (ADRs)

- **ADR-001** (deste PRD): validadores canônicos em `.agents/scripts/` com cascata e mirror —
  ver `adr-001-validadores-canonicos-agents-scripts.md`.
- **ADR-002**: paridade de enforcement por hooks nativos por-tool chamando validador
  compartilhado; Codex suplementado por sandbox/approval — ver
  `adr-002-hooks-nativos-paridade-cross-cli.md`.

### Riscos Conhecidos

- **Mover validadores** pode quebrar callers que assumem `.claude/scripts/` — mitigar mantendo
  `.claude/scripts/` como mirror e atualizando todos os callers para a cascata.
- **Formatos de hook divergentes** entre tools — mitigar com fixtures por-tool e gate de sync.
- **Codex route-around** — mitigado por sandbox/approval, nunca só pelo hook.
- **Tasks legadas sem seção AC** — mitigado por compatibilidade não-fatal.

### Conformidade com Padrões

- PT-BR; Conventional Commits; FakeFileSystem em unit; DI via construtor; `fmt.Errorf("ctx: %w")`.
- Canônico em `.agents/`; mirrors gerados; gate de drift obrigatório.

### Arquivos Relevantes e Dependentes

- `.agents/scripts/validate-task-evidence.sh` (novo canônico), `.claude/scripts/*` (mirror),
  `internal/embedded/assets/.claude/scripts/*`.
- `.agents/hooks/post-execute-task.sh`; configs de hook por-tool.
- `.agents/skills/{create-prd,create-technical-specification,create-tasks,review,execute-task,execute-all-tasks,bugfix}/SKILL.md`.
- `.agents/skills/agent-governance/references/{multiple-choice-protocol,severity-mapping,enforcement-matrix}.md`.
- `.agents/skills/execute-task/assets/task-execution-report-template.md`.
- `scripts/sync-skills.sh`, `scripts/check-scripts-sync.sh` (novo), `Makefile`.
