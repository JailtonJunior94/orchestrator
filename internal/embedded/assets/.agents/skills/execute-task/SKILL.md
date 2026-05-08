---
name: execute-task
version: 1.0.0
depends_on: [review, bugfix, agent-governance]
description: Executa uma tarefa de implementação aprovada por meio de codificação, validação, revisão e captura de evidências. Use quando um arquivo de tarefa estiver pronto para implementação e fechamento com testes, lint e evidência de revisão. Não use para planejamento, refatorações amplas sem tarefa ou exploração especulativa de código.
---

# Executar Tarefa

## Procedimentos

**Etapa 1: Validar a elegibilidade da tarefa**
1. Verificar profundidade de invocação a partir da raiz do repositório: `source scripts/lib/check-invocation-depth.sh || { echo "failed: depth limit exceeded"; exit 1; }`. Se sourcing não for possível, usar o fallback documentado no script (`bash scripts/lib/check-invocation-depth.sh` + `eval`). Parar se o limite for atingido.
2. Verificar drift de lockfile de skills: `ai-spec skills --verify || { echo "blocked: skills lock drift"; exit 1; }`. Abortar com `blocked` se o comando retornar exit ≠ 0.
3. Derivar `<feature-slug>` a partir do caminho da tarefa fornecida pelo usuário (`tasks/prd-<feature-slug>/<num>_*.md`). Se múltiplos PRDs casarem, se o caminho não permitir inferência única, ou se o usuário só passar um número de tarefa sem PRD, retornar `needs_input` listando os candidatos.
4. Confirmar que `tasks/prd-<feature-slug>/tasks.md`, o arquivo de tarefa alvo, `prd.md` e `techspec.md` estão presentes.
5. Executar gate de cobertura de RF: `ai-spec check-spec-drift tasks/prd-<feature-slug>/tasks.md` — parar com `blocked` se houver RFs não cobertos.
6. Selecionar a primeira tarefa elegível apenas quando o usuário não tiver escolhido uma explicitamente.
7. Confirmar que todas as dependências da tarefa estão em `done`.
8. Parar com `needs_input` ou `blocked` se a tarefa não for elegível para execução.

**Etapa 2: Carregar o contexto de implementação**
1. Ler por completo o arquivo de tarefa selecionado, `prd.md` e `techspec.md`.
2. Verificar coerência temporal: se `prd.md` ou `techspec.md` foram modificados após a criação de `tasks.md`, avisar o usuário que os artefatos de origem podem ter divergido e perguntar se deseja continuar ou re-gerar as tarefas. Parar com `needs_input` se o usuário não confirmar.
3. Confirmar que o contrato de carga base definido em `AGENTS.md` foi cumprido.
4. Para cada linguagem afetada pelo diff, ler `.agents/skills/<linguagem>-implementation/SKILL.md` (ex.: `go`, `node`, `python`) e carregar apenas as referências exigidas pela mudança. Se a skill da linguagem não existir, parar com `needs_input`.
5. Mapear objetivo da tarefa, critérios de aceitação, subtarefas e arquivos-alvo antes de editar.

**Etapa 3: Executar a etapa de implementação**
1. Seguir a ordem das subtarefas definida no arquivo de tarefa.
2. Implementar testes junto com as mudanças de produção.
3. Resolver o entrypoint de validação na seguinte ordem (parar no primeiro disponível):
   - `task test|lint|fmt` (Taskfile.yml presente)
   - `make test|lint|fmt` (Makefile com esses targets)
   - comando nativo da linguagem afetada: `go test ./... && go vet ./...` (Go), `pnpm test && pnpm lint` ou `npm test && npm run lint` (Node), `pytest && ruff check` (Python)
   - se nenhum se aplicar, parar com `needs_input` solicitando o entrypoint canônico do projeto — não adivinhar
4. Rodar validação direcionada após subtarefas relevantes, não apenas no final.
5. Registrar comandos executados e arquivos alterados para o relatório.
6. Parar com `needs_input` se uma decisão obrigatória ou entrada faltante bloquear a conclusão segura.

**Etapa 4: Executar a etapa de validação e aprovação**
1. Seguir Etapa 4 de `.agents/skills/agent-governance/SKILL.md`.
2. Rodar teste e lint do pacote/módulo afetado de forma obrigatória. Rodar a suíte completa do projeto (`hard`) quando qualquer um destes gatilhos for verdadeiro: o diff cruza fronteira de pacote, altera APIs públicas/exportadas, ou modifica configuração compartilhada (build, CI, dependências, governança). Para diffs amplos que não disparem nenhum gatilho `hard`, rodar suíte completa fica como `guideline` — decidir por julgamento do agente.
3. Verificar cada critério de aceitação com evidência explícita.
4. Invocar a habilidade `review` para o diff produzido e incluir `prd.md` e `techspec.md` como contexto de revisão.
5. Mapear o veredito de `review`:
   - `APPROVED` ou `APPROVED_WITH_REMARKS` → seguir para Etapa 5.
   - `REJECTED` com bugs no formato canônico → invocar `bugfix` dentro do escopo da tarefa, rerodar validações e nova revisão.
   - `REJECTED` sem formato canônico → parar com `failed`, anexando o motivo do revisor.
   - `BLOCKED` → parar com `blocked`, anexando o motivo do revisor; **não** invocar `bugfix`.
6. Aceitar apenas `APPROVED` ou `APPROVED_WITH_REMARKS` como veredito final.

**Etapa 5: Persistir as evidências**
1. Ler `assets/task-execution-report-template.md`.
2. Salvar o relatório como `tasks/prd-<feature-slug>/[num]_execution_report.md` antes de qualquer atualização de status.
3. Rodar o validador de evidências contra o relatório salvo, resolvendo o caminho na ordem `.claude/scripts/validate-task-evidence.sh` → `.agents/scripts/validate-task-evidence.sh` → `scripts/validate-task-evidence.sh`. Parar com `failed` se nenhum existir.
4. Se o validador de evidências falhar, retornar `blocked` descrevendo a evidência ausente. **Não** atualizar o status da tarefa em `tasks.md`.
5. Atualizar o status da tarefa em `tasks.md` para `done` somente após implementação, validação, revisão aprovadora final e validador de evidência terem retornado sucesso.

**Etapa 6: Encerrar explicitamente**
1. Informar o status da tarefa, os resultados de validação, o veredito do revisor e o caminho do relatório.
2. Retornar `done`, `blocked`, `failed` ou `needs_input` usando apenas nomes de estado canônicos.

## Paralelismo e Subagentes

Spawnar subagente é uma decisão de custo/benefício. O agente principal deve aplicar a regra abaixo antes de delegar:

**Spawnar subagente APENAS quando todos forem verdadeiros:**
1. A saída esperada do trabalho excede o que o agente principal precisa reter (ex.: extrair 1 fato de arquivo grande; varredura em múltiplos pacotes; build verboso de cuja saída só interessa o veredito).
2. O trabalho é independente (não consome dado intermediário de outro passo da mesma etapa).
3. O custo de spawn (overhead de prompt + saída resumida) é menor que o custo de manter o resultado bruto no contexto principal.

**Não spawnar quando:**
- A edição é alvo de arquivo já carregado no contexto principal.
- A operação é sequencial e depende imediatamente do resultado anterior.
- Múltiplas chamadas de ferramenta paralelas (Bash, Edit, Read concorrentes) já resolvem o caso sem subagente.

**Pontos de aplicação dentro desta skill:**
- Etapa 2: se a tarefa toca múltiplas linguagens e cada SKILL de linguagem tem refs grandes, delegar a leitura/seleção de refs por linguagem a subagentes paralelos. Para uma única linguagem, ler inline.
- Etapa 3: se houver subtarefas independentes em pacotes distintos sem ordem obrigatória, delegar cada bloco a um subagente. Para subtarefas sequenciais ou dentro de um único pacote, executar inline.
- Etapa 4: rodar `task test` e `task lint` (ou equivalentes) em paralelo via chamadas Bash concorrentes — sem subagente. Delegar a subagente apenas se a saída for muito verbosa e o agente principal só precisar do veredito (`pass|fail` + falhas relevantes).
- Etapa 5: validador de evidência é leve e síncrono — sempre inline.

**Critério de fechamento:** o relatório final deve registrar quais blocos foram delegados a subagentes e por quê, em "Comandos Executados" (formato: `subagent[<descrição>] -> <resumo>`).

## Tratamento de Erros

* Se o arquivo de tarefa estiver desatualizado em relação ao codebase ou à especificação técnica, parar e expor o descompasso antes de editar código.
* Se as validações falharem, tentar apenas uma remediação limitada. Se a falha apontar para um problema de desenho mais profundo, parar e retornar `failed` com o comando bloqueante exato e um diagnóstico curto.
* Respeitar o limite de profundidade de invocação definido em `.agents/skills/agent-governance/SKILL.md`. Se review invocar bugfix e bugfix precisar de nova review, esta é a profundidade máxima — não re-invocar bugfix a partir dessa segunda review.

## Resolução de paths

Todo caminho `tasks/prd-<slug>/` referenciado neste documento resolve para `${AI_TASKS_ROOT:-tasks}/${AI_PRD_PREFIX:-prd-}<slug>/`. Os defaults preservam o layout histórico. Para customizar (monorepos, convenções próprias), declare em `.claude/config.yaml` ou `.agents/config.yaml`:

```yaml
tasks_root: tasks
prd_prefix: prd-
evidence_dir: ""
coverage_threshold: 70
language_default: ""
```

As variáveis `AI_TASKS_ROOT`, `AI_PRD_PREFIX`, `AI_EVIDENCE_DIR`, `AI_COVERAGE_THRESHOLD`, `AI_LANGUAGE_DEFAULT` e `AI_TOOL` são exportadas automaticamente por `scripts/lib/check-invocation-depth.sh` (Etapa 1). Skills, validators e o runtime `internal/taskloop` consomem a mesma fonte, garantindo paridade exata entre Claude Code, Codex, Gemini e Copilot.
