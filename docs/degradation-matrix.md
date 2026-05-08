# Matriz de degradação

Documento canônico que descreve o comportamento do fluxo `execute-task → review → bugfix` quando componentes esperados estão ausentes ou parcialmente disponíveis. Todo fallback é **explícito e anotado**: relatórios e validators emitem `degraded:<reason>:<tool>` em vez de operar silenciosamente.

Os 4 agentes suportados (Claude Code, Codex, Gemini, Copilot) seguem a mesma matriz; quando uma limitação é tool-specific, está marcada na coluna **Tool-specific**.

## Convenções

- **Componente:** artefato cuja ausência aciona degradação.
- **Comportamento:** o que o fluxo faz quando ele falta.
- **Impacto:** quais garantias se perdem (correção, paridade, evidência, etc.).
- **Remediação:** comando ou ação para restaurar o estado normal.
- **Tool-specific:** `—` quando aplica a todos; nome da tool quando específico.
- **Anotação no relatório:** literal emitido em `## Degradação` quando o fluxo opera em fallback.

Severidades:
- `block` — fluxo aborta com `blocked` ou `failed`. Não há fallback seguro.
- `warn`  — fluxo segue com fallback documentado; relatório anotado.
- `best-effort` — limitação técnica conhecida (ex.: ADR-007); paridade `BestEffort`.

---

## Tabela

| # | Componente | Severidade | Comportamento | Impacto | Remediação | Tool-specific | Anotação no relatório |
|---|---|---|---|---|---|---|---|
| 1 | `ai-spec` CLI no PATH | block | `execute-task` Etapa 1 não consegue rodar `ai-spec check-spec-drift` nem `ai-spec skills --verify`; aborta com `blocked: ai-spec-cli-missing` | Sem gate de cobertura de RF e sem gate de lockfile | Instalar via `go install github.com/JailtonJunior94/ai-spec-harness@latest` ou usar binário do release | — | `degraded:ai-spec-cli-missing:<tool>` |
| 2 | `AGENTS.md` na raiz | block | Skills não atendem o contrato de carga base; `execute-task` Etapa 2 retorna `needs_input` | Carga de governança incompleta; risco de divergência entre tools | Restaurar via `ai-spec install --tools all` ou copiar de release | — | `degraded:agents-md-missing:<tool>` |
| 3 | `.claude/rules/governance.md` | warn | Precedência transversal não resolvível em conflitos; skills aplicam regras locais sem o tie-breaker canônico | Decisões de severidade podem divergir entre tools | `ai-spec install --tools all` | — | `degraded:governance-md-missing:<tool>` |
| 4 | `scripts/lib/check-invocation-depth.sh` | block | Sem proteção de profundidade; sem export de env vars (`AI_TASKS_ROOT`, etc.); risco de loop infinito review↔bugfix | Paths não resolvem via config; loop pode escalar custo | Reinstalar bundle | — | `degraded:invocation-depth-script-missing:<tool>` |
| 5 | `.claude/scripts/validate-task-evidence.sh` | block | Etapa 5 de `execute-task` não consegue validar evidência; status da task não vai para `done` | Tasks fechariam sem evidência verificada | Reinstalar bundle | — | `degraded:evidence-validator-missing:<tool>` |
| 6 | `.claude/scripts/validate-bugfix-evidence.sh` | block | `bugfix` Etapa 5 não valida relatório; sai com `failed` | Bugs marcados `fixed` sem evidência | Reinstalar bundle | — | `degraded:bugfix-validator-missing:<tool>` |
| 7 | `.claude/skills/bugfix/scripts/validate-bug-input.py` | warn | Validação de schema canônico de bug pula JSON Schema; usa fallback manual equivalente | Cobertura de validação reduzida (sem mensagens detalhadas de campo) | Instalar `jsonschema` (`pip install jsonschema`) ou reinstalar bundle | — | `degraded:bug-schema-validator-fallback:<tool>` |
| 8 | `skills-lock.json` (ADR-005) | warn | Sem gate de drift; skills upstream podem mudar comportamento sem detecção | Paridade comportamental entre versões não verificável | `ai-spec skill-bump --regenerate-lock` | — | `degraded:skills-lock-missing:<tool>` |
| 9 | `.claude/config.yaml` / `.agents/config.yaml` | warn | `LoadRuntime` retorna defaults compatíveis (`tasks/prd-<slug>/`); env vars exportadas com defaults | Customização de paths indisponível; comportamento default | Criar config.yaml com chaves desejadas | — | `degraded:config-yaml-missing:<tool>` (apenas quando defaults conflitarem com layout do projeto) |
| 10 | Skill da linguagem (`go-implementation` / `node-implementation` / `python-implementation`) | warn | `execute-task` Etapa 2 carrega apenas governança transversal; refs de linguagem não disparam | Heurísticas Go/Node/Python não se aplicam ao diff | `ai-spec install --langs <lang>` | — | `degraded:lang-skill-missing:<lang>:<tool>` |
| 11 | `internal/parity` invariantes (ADR-008) | block | `doctor` reporta paridade incalculável; instalação não é validável entre tools | Drift entre tools indetectável | Reinstalar binário atualizado | — | `degraded:parity-invariants-unavailable:<tool>` |
| 12 | `Taskfile.yml` / `Makefile` | warn | Etapa 3 cai no fallback nativo (`go test ./... && go vet ./...`, `pnpm test`, `pytest`) | Comandos canônicos do projeto não usados; risco de divergência local | Adicionar `Taskfile.yml` com targets `test`, `lint`, `fmt` | — | `degraded:project-validation-entrypoint-missing:<tool>` |
| 13 | Cobertura ≥ `coverage_threshold` | warn | `validate-task-evidence.sh` falha se delta < `-2.0%`; relatório anotado e task volta a `pending` | Regressão de cobertura silenciosa | Adicionar testes e re-rodar `task test` | — | `degraded:coverage-regression:<pkg>:<delta>` |
| 14 | `CLAUDE.md` | best-effort | Claude Code carrega via `AGENTS.md` apenas; comportamento ainda paritário via skills | Sem espelho específico para Claude; OK porque AGENTS.md é canônico | `ai-spec install --tools claude` | claude | `degraded:claude-md-missing` |
| 15 | `.codex/config.toml` | best-effort | Codex usa `AGENTS.md` direto; gates injetados via wrapper CLI | Sem prelúdio específico Codex | `ai-spec install --tools codex` | codex | `degraded:codex-config-missing` |
| 16 | `GEMINI.md` | best-effort | Gemini carrega via `AGENTS.md`; skills via `.gemini/commands/*.toml` | Sem instruções inline customizadas | `ai-spec install --tools gemini` | gemini | `degraded:gemini-md-missing` |
| 17 | `COPILOT.md` | best-effort | Copilot é stateless (ADR-007); cada invocação reinjeta gates via `ai-spec wrapper copilot` | Custo de prompt maior; semântica preservada | `ai-spec install --tools copilot` | copilot | `degraded:copilot-md-missing` |
| 18 | `yq` para parser YAML em scripts | warn | `check-invocation-depth.sh` usa fallback awk para chaves planas | Schema aninhado não é suportado em fallback | `brew install yq` ou `apt install yq` | — | `degraded:yq-missing-fallback-awk` |
| 19 | `git` no PATH | block | `doctor`, `check-invocation-depth.sh` e `reviewer` não conseguem resolver raiz nem capturar diff | Fluxo inteiro indisponível | Instalar git | — | `degraded:git-missing:<tool>` |
| 20 | `golangci-lint` (Go) ou equivalente | warn | Etapa 4 marca lint como `skipped`; testes ainda obrigatórios | Lint não verificado; review pode emitir achados estilísticos extras | `brew install golangci-lint` | — | `degraded:lint-tool-missing:<tool>` |

---

## Política de promoção `BestEffort` → `Common`

Limitações tecnicamente irredutíveis (linhas 14–17) são `BestEffort` em `internal/parity/parity.go`. Quando uma limitação puder ser eliminada (ex.: Copilot ganhar suporte stateful), a entrada correspondente é promovida a `Common` no mesmo PR que adiciona a capacidade — nunca antes.

## Como o fluxo emite anotações de degradação

1. `internal/taskloop/runloop.go` chama `report.AppendDegradation(reason, tool)` ao detectar fallback em qualquer Etapa.
2. `internal/taskloop/report.go` agrupa anotações em uma seção `## Degradação` no relatório de execução, formato:
   ```
   ## Degradação
   - degraded:agents-md-missing:claude
   - degraded:lint-tool-missing:claude
   ```
3. `validate-task-evidence.sh` reconhece anotações `degraded:` e exibe `warn` (não `fail`) — exceto para entradas marcadas `block` na tabela acima, em que o fluxo já não chega à validação.
4. `doctor` referencia esta matriz em cada `fail` com link relativo a `docs/degradation-matrix.md`.

## Verificação rápida

```bash
# Forçar degradação (renomear AGENTS.md temporariamente)
mv AGENTS.md AGENTS.md.bak
ai-spec doctor . | grep "agents-md-missing"
# Restaurar
mv AGENTS.md.bak AGENTS.md
```

## Referências

- ADR-003 — paridade semântica entre tools
- ADR-007 — Copilot CLI stateless workaround
- ADR-008 — paridade multi-tool com invariantes (29 → 40 invariantes)
- `internal/parity/parity.go` — implementação dos invariantes
- `scripts/lib/check-invocation-depth.sh` — entrypoint de gates e env vars
