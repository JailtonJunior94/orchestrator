---
name: execute-task
version: 1.4.0
depends_on: [review, bugfix, agent-governance]
description: Executa uma tarefa de implementação aprovada via codificação, validação, revisão e captura de evidências. Carrega skills processuais declaradas em `## Skills Necessárias` (formato canônico estrito) + skills de linguagem inferidas do diff. Use quando um task file estiver pronto para implementação. Não use para planejamento.
---

# Executar Tarefa

## Procedimentos

**Etapa 1: Validar elegibilidade**
1. `source scripts/lib/check-invocation-depth.sh || { echo "failed: depth limit exceeded"; exit 1; }`. Fallback: `bash` + `eval`.
2. **Pre-flight gates condicionais (F8)**:
   - `AI_PREFLIGHT_DONE=1` exportada → pular gates (orquestrador já validou).
   - Senão: `ai-spec skills --verify` (`blocked` se ≠0); `ai-spec check-spec-drift tasks/prd-<slug>/tasks.md` (`blocked` se RF não coberto).
3. Derivar `<slug>` do path. Ambíguo → `needs_input`.
4. Confirmar `tasks.md`, task file alvo, `prd.md`, `techspec.md` presentes.
5. Selecionar primeira tarefa elegível só se usuário não escolheu.
6. Confirmar deps em `done` → senão `blocked`.

**Etapa 2: Carregar contexto**
1. Ler task file, `prd.md`, `techspec.md` por completo.
2. **Coerência temporal**: prd/techspec modificado após tasks.md → avisar drift; `needs_input` se usuário não confirmar.
3. Confirmar AGENTS.md base contract.
4. **Detecção de linguagem (F1)**:
   - Inspecionar `Arquivos Relevantes`: Go (`*.go`), Node (`*.ts/.tsx/.js/.jsx/.mjs`), Python (`*.py`).
   - Linguagem detectada → ler `.agents/skills/<linguagem>-implementation/SKILL.md`. Skill ausente → `needs_input`.
   - **Tarefas non-code** (docs, configs, SQL, shell, MD): nenhuma skill de linguagem; prosseguir.
5. **Skills processuais declaradas (F6+F16+F28)**:
   - Parsear seção `## Skills Necessárias` (gerada por `create-tasks` v1.4+).
   - **Regex estrito**: `^- \`([a-z0-9-]+)\` — .+$`. Linha não-canônica → `failed: malformed Skills Necessárias entry on task <id>: <linha>`.
   - Conteúdo único exato `Nenhuma além das auto-carregadas (governance + linguagem).` = vazio. Variações → `failed: malformed Skills Necessárias section`.
   - Ler coluna `Skills` em `tasks.md` (`—` = vazio).
   - **Sync gate (sem união silenciosa)**: divergente → `failed: skills sync drift on task <id> — file=<S_file> table=<S_table>`.
   - Ambas vazias: prosseguir (retrocompatível).
   - UMA fonte vazia outra preenchida: `failed: skills declaration missing in <fonte>`.
   - Para cada skill: validar `.agents/skills/<skill>/SKILL.md` existe (`needs_input` se não); ler description + procedimentos; refs sob demanda.
   - **Regras agnósticas**: nunca inferir por heurística textual; nunca carregar não-declaradas; descoberta via `ls .agents/skills/`.
6. Mapear objetivo, critérios, subtarefas, arquivos-alvo antes de editar.

**Etapa 3: Implementação**
1. Seguir ordem das subtarefas. Implementar testes junto com produção.
2. Resolver entrypoint (parar no primeiro): `task test|lint|fmt` → `make test|lint|fmt` → nativo (`go test ./... && go vet`, `pnpm test && pnpm lint`, `pytest && ruff check`). Nenhum → `needs_input`.
3. Validação direcionada após cada subtarefa, não só no final.
4. Registrar comandos e arquivos. `needs_input` se decisão obrigatória bloquear.

**Etapa 4: Validação + revisão (F24)**
1. Seguir Etapa 4 de `agent-governance`.
2. Teste/lint do pacote afetado (mandatório). Suíte completa (`hard`) se diff cruzar pacote, alterar API pública, ou tocar config compartilhada.
3. Verificar critérios com evidência explícita.
4. Invocar `review` com prd.md + techspec.md como contexto.
5. **Mapear veredito**:
   - `APPROVED` → Etapa 5.
   - `APPROVED_WITH_REMARKS` → **inspecionar severidade (F24)**: parsear remarks procurando `[critical]`, `[security]`, `[blocker]`, `[high]` (case-insensitive). Tag crítica → escalar para `blocked: APPROVED_WITH_REMARKS contém remark crítico — <remark>`; NÃO seguir Etapa 5; NÃO bugfix; devolver ao humano. Sem tag crítica → Etapa 5; remarks vão para "Riscos Residuais".
   - `REJECTED` com bugs canônicos → `bugfix` no escopo, rerodar validações + nova review.
   - `REJECTED` sem formato canônico → `failed`.
   - `BLOCKED` → `blocked`; **não** invocar `bugfix`.
6. Final aceito apenas: `APPROVED`, OU `APPROVED_WITH_REMARKS` confirmado sem remarks críticos.

**Etapa 5: Persistir evidências (F25 checkpoint)**
1. Salvar `tasks/prd-<slug>/[num]_execution_report.md` (overwrite com `# Generated: <ISO-8601 UTC>` no header — F36) a partir de `assets/task-execution-report-template.md`.
2. Rodar validador de evidências (resolver: `.claude/scripts/...` → `.agents/scripts/...` → `scripts/...`). Nenhum → `failed`. Falha → `blocked`; não mutar tasks.md.
3. **Checkpoint YAML antes de mutar tasks.md (F25)**:
   - `mkdir -p tasks/prd-<slug>/.checkpoints/`.
   - Escrever `.checkpoints/<num>.yaml.tmp` com `status`, `report_path`, `summary`, `timestamp` (ISO-8601 UTC).
   - `mv -n .yaml.tmp .yaml` atômico. Completo ou inexistente, nunca parcial.
4. **Só após checkpoint persistido**, mutar tasks.md para `done`.
5. **Lock atômico em tasks.md (F3+F32)** quando invocador é `execute-all-tasks` em wave paralela:
   - POSIX: `flock -x -w 30 tasks/prd-<slug>/tasks.md.lock -c '<edit>'`.
   - Sem `flock`: temp + `mv -n` atômico.
   - Fallback final (Windows nativo, containers minimal): escrever em `tasks/prd-<slug>/.partials/tasks.md.<num>.partial`; orquestrador consolida na sua Etapa 5.
   - Lock falha em 30s → `failed: tasks.md lock timeout`.

**Etapa 6: Encerrar**
Retornar `done`, `blocked`, `failed` ou `needs_input` (canônico) com path do relatório, validações e veredito do reviewer.

## Paralelismo e Subagentes

Spawn APENAS se: (1) saída excede o que principal precisa reter, (2) trabalho independente, (3) custo de spawn < custo de bruto no contexto. Não spawnar para: arquivo já carregado; sequencial dependente; paralelas tool calls (Bash/Edit) já resolvem.

Aplicação: Etapa 2 (refs grandes multi-linguagem), Etapa 3 (subtarefas em pacotes distintos). Etapa 4: `task test`+`task lint` paralelos via Bash, sem subagente. Etapa 5: sempre inline. Registrar em "Comandos Executados" como `subagent[<desc>] -> <resumo>`.

## Tratamento de Erros

* Task file desatualizado vs código/spec → parar e expor antes de editar.
* Validação falha → uma remediação limitada; falha mais profunda → `failed` com comando bloqueante + diagnóstico.
* Respeitar depth limit de `agent-governance`. Cadeia review → bugfix → review é máxima.

## Resolução de paths

`tasks/prd-<slug>/` resolve para `${AI_TASKS_ROOT:-tasks}/${AI_PRD_PREFIX:-prd-}<slug>/`. Configurar em `.claude/config.yaml`/`.agents/config.yaml` (`tasks_root`, `prd_prefix`, `evidence_dir`, `coverage_threshold`, `language_default`). Vars exportadas por `scripts/lib/check-invocation-depth.sh`. `AI_TOOL` validado contra `{claude, codex, gemini, copilot}`; inválido → unset (modo agnóstico).
