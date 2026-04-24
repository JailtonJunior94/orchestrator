# Referencia do task-loop

> Guia consolidado de execucao com o `task-loop`. Cobre flags, heuristicas, alternativas e comparativos.
>
> Para instalar o CLI, consulte o [README](../README.md). Para o pipeline completo de skills, consulte o [Guia de uso das skills](skills-usage-guide.md).

## Fluxo recomendado

1. Primeiro valide se as tasks estao pequenas, ordenadas e com dependencias claras.
2. Rode um `dry-run` para confirmar qual task sera escolhida primeiro.
3. Comece com poucas iteracoes para observar a qualidade do lote inicial.
4. So depois aumente `max-iterations` e `timeout` para execucao mais longa.
5. Sempre salve o relatorio final em caminho explicito quando estiver trabalhando em uma feature relevante.

## Comandos recomendados

Inspecao inicial:

```bash
ai-spec task-loop --tool codex --dry-run tasks/prd-payments-list
```

Primeiro lote pequeno:

```bash
ai-spec task-loop --tool codex --max-iterations 2 tasks/prd-payments-list
```

Execucao mais longa com relatorio salvo:

```bash
ai-spec task-loop \
  --tool codex \
  --max-iterations 8 \
  --timeout 1h \
  --report-path ./task-loop-report-payments.md \
  tasks/prd-payments-list
```

## Quando cada flag ajuda

### Modo simples

| Flag | Quando usar |
| --- | --- |
| `--tool` | escolher o agente unico (claude, codex, gemini, copilot) — modo simples |
| `--dry-run` | validar ordem e elegibilidade das tasks antes de gastar ciclo de agente |
| `--max-iterations` | controlar lote inicial, reduzir risco e evitar rodar tasks demais de uma vez; `0` executa sem limite de iteracoes ate nao restar tasks pendentes |
| `--timeout` | dar mais tempo para tasks grandes ou com validacoes demoradas |
| `--report-path` | manter rastreabilidade e facilitar revisao posterior |

### Modo avancado: executor, reviewer e bugfix automatico

O modo avancado permite configurar executor e reviewer com ferramentas e modelos distintos. `--tool` e `--executor-tool` sao mutuamente exclusivos.

Quando o reviewer retorna exit code != 0 (achados criticos), o loop invoca automaticamente uma fase de bugfix usando o mesmo executor, com prompt estruturado a partir dos achados. A fase de bugfix tem guardrails de isolamento proprios e o resultado e registrado separadamente no report avancado. O fluxo por iteracao e: **executor → reviewer → bugfix (se necessario)**.

```bash
# Executor: Codex; Reviewer: Claude Opus (template padrao embutido)
ai-spec task-loop \
  --executor-tool codex \
  --reviewer-tool claude \
  --reviewer-model claude-opus-4-6 \
  tasks/prd-payments-list

# Com modelos explicitamente definidos para ambos os papeis
ai-spec task-loop \
  --executor-tool claude \
  --executor-model claude-sonnet-4-6 \
  --reviewer-tool claude \
  --reviewer-model claude-opus-4-6 \
  --report-path ./task-loop-report.md \
  tasks/prd-payments-list

# Dry-run avancado: exibe perfis, compatibilidade e preview do prompt de revisao
ai-spec task-loop \
  --executor-tool gemini \
  --reviewer-tool claude \
  --dry-run \
  tasks/prd-payments-list
```

| Flag avancada | Quando usar |
| --- | --- |
| `--executor-tool` | ferramenta do agente executor (obrigatorio no modo avancado) |
| `--executor-model` | modelo especifico do executor; "" usa default da ferramenta |
| `--reviewer-tool` | ferramenta do agente reviewer; omitir desativa a etapa de revisao |
| `--reviewer-model` | modelo especifico do reviewer |
| `--reviewer-prompt-template` | path de template `.tmpl` customizado para o prompt de revisao |
| `--executor-fallback-model` | modelo de fallback nativo do executor (suporte nativo Claude only) |
| `--reviewer-fallback-model` | modelo de fallback nativo do reviewer (suporte nativo Claude only) |
| `--fallback-tool` | ferramenta de fallback para validacao pre-loop |
| `--allow-unknown-model` | aceitar combinacoes ferramenta+modelo fora do catalogo interno sem erro |

> O dry-run no modo avancado exibe status de compatibilidade (✓/✗) para cada perfil e o preview do template resolvido para a primeira task elegivel.

## Heuristicas praticas

- prefira `max-iterations` baixo no inicio de uma feature nova
- aumente o lote apenas quando as primeiras tasks estiverem saindo com boa qualidade
- use `task-loop` quando o pacote de tasks ja estiver maduro; para task isolada ou ainda instavel, prefira `execute-task`
- se a feature for Go e envolver DDD, concorrencia ou risco arquitetural, refine antes em `techspec.md` em vez de delegar ambiguidade ao loop

## Executar uma task especifica (sem task-loop)

Use esse modo quando quiser revisar ou executar uma task isolada sem passar pelo orchestrador. Funciona bem para tasks experimentais, ajustes pontuais ou quando o pacote ainda nao esta maduro o suficiente para execucao em lote.

Inspecione o pacote e escolha a task manualmente:

```bash
ai-spec inspect ../api-pagamentos
```

Emita a instrucao pronta para o agente:

```bash
ai-spec wrapper codex execute-task .
```

Prompt direto para o agente executar uma unica task:

```text
Use a skill execute-task para implementar a task 01_task.md localizada em tasks/prd-payments-list.

Criterios obrigatorios:
- ler o arquivo de task antes de iniciar qualquer alteracao
- preservar a arquitetura e os contratos publicos existentes
- executar os testes e validacoes definidos na propria task
- registrar evidencias de conclusao no arquivo de task
```

## Executar todas as tasks elegiveis (com task-loop)

Use esse modo quando o pacote de tasks ja estiver maduro, ordenado e com dependencias claras.

```bash
# validar antes de gastar ciclo de agente
ai-spec task-loop --tool codex --dry-run tasks/prd-payments-list

# primeiro lote pequeno para observar qualidade
ai-spec task-loop --tool codex --max-iterations 2 tasks/prd-payments-list

# execucao completa com relatorio
ai-spec task-loop \
  --tool codex \
  --max-iterations 8 \
  --timeout 1h \
  --report-path ./task-loop-report-payments.md \
  tasks/prd-payments-list
```

## Quando usar cada abordagem

| Situacao | Abordagem recomendada |
| --- | --- |
| task isolada, experimental ou instavel | `execute-task` direto (sem task-loop) |
| revisar uma unica task antes de continuar o lote | `execute-task` direto |
| pacote maduro, ordenado e com dependencias claras | `task-loop` |
| primeiro lote de uma feature nova — ainda incerto | `task-loop` com `--max-iterations 2` |
| execucao longa com rastreabilidade necessaria | `task-loop` com `--report-path` |
| feature com concorrencia ou risco arquitetural alto | refinar `techspec.md` antes de qualquer execucao |

## Como o task-loop garante isolamento de contexto

Cada iteracao invoca o agente como um processo completamente novo via `exec.CommandContext` com a flag `--print -p <prompt>`. Isso significa:

- nenhum estado da sessao anterior e carregado — o processo inicia do zero
- o contexto injetado e exatamente o minimo obrigatorio: arquivo da task + pasta do PRD (que contem `prd.md` e `techspec.md`)
- a variavel `AI_INVOCATION_DEPTH` e resetada para `0` a cada invocacao, prevenindo aninhamento
- ao terminar ou falhar, o processo e encerrado com `SIGKILL` no grupo inteiro — sem estado residual

### Equivalencia entre ciclo manual e task-loop

| Ciclo manual | O que task-loop faz automaticamente |
| --- | --- |
| abrir nova sessao | spawn de novo processo por task |
| fornecer task file + trecho da techspec | passa `task file` + `prd folder` no prompt |
| executar execute-task com prompt mandatorio | prompt gerado por `BuildPrompt` com instrucao de seguir `SKILL.md` |
| fechar sessao apos evidencia registrada | processo encerrado; status relido de `tasks.md` e do arquivo da task |
| abrir nova sessao para proxima task | proxima iteracao spawn novo processo |

### O que o task-loop executa por tool

```bash
# Claude
claude --dangerously-skip-permissions --print --bare -p "<prompt>"

# Codex
codex exec --dangerously-bypass-approvals-and-sandbox -p "<prompt>"

# Gemini
gemini --yolo -p "<prompt>"

# Copilot
copilot --autopilot --yolo -p "<prompt>"
```

### Quando preferir o ciclo manual

| Situacao | Abordagem |
| --- | --- |
| task com ambiguidade na spec — precisa de input antes de implementar | ciclo manual: resolva a ambiguidade, depois execute |
| task que toca fronteira arquitetural nao documentada na techspec | ciclo manual: adicione o contexto faltante no prompt |
| bundle ainda instavel — tasks sem criterio de pronto claro | ciclo manual com `--max-iterations 1` ate estabilizar |
| bundle maduro, criterios de pronto claros, techspec completa | `task-loop` automatico |

### Sinal de que o task-loop pode ser usado com seguranca

- `tasks.md` tem dependencias explicitas e nenhuma task com escopo aberto
- cada task file tem secao `Criterio de pronto` com comandos de validacao concretos
- `techspec.md` define contratos, tipos de erro e responsabilidade de cada camada
- `dry-run` nao aponta task com status invalido ou dependencia circular

## Alternativas sem o task-loop

### Alternativa 1 — Invocar o agente diretamente por task

Use quando quiser controle total sobre qual task executar sem depender do parser de `tasks.md`.

```bash
# Claude — uma task
claude --dangerously-skip-permissions --print --bare -p \
  "You are executing the \"execute-task\" skill.

First, read AGENTS.md at the repository root to load governance rules and conventions.

Then read and follow the instructions in: .agents/skills/execute-task/SKILL.md

Target task file: tasks/prd-payments-list/01_repository.md
PRD folder: tasks/prd-payments-list

Execute ONLY this task. Follow all skill steps:
1. Validate eligibility
2. Load context (prd.md, techspec.md)
3. Implement
4. Validate (tests, lint)
5. Review
6. Update task status in task file and tasks.md
7. Generate execution report

Update **Status:** in tasks/prd-payments-list/01_repository.md and the corresponding row in tasks/prd-payments-list/tasks.md to reflect the final state."
```

Para Codex ou Gemini, substitua o binario e as flags:

```bash
# Codex
codex exec --dangerously-bypass-approvals-and-sandbox -p "<mesmo prompt>"

# Gemini
gemini --yolo -p "<mesmo prompt>"
```

### Alternativa 2 — Script shell iterando tasks.md

```bash
#!/usr/bin/env bash
set -euo pipefail

PRD_FOLDER="${1:?informe o prd folder}"
TOOL="${2:-claude}"
MAX="${3:-99}"
count=0

while [ "$count" -lt "$MAX" ]; do
  TASK_FILE=$(ls "${PRD_FOLDER}"/[0-9]*_*.md 2>/dev/null \
    | while read -r f; do
        status=$(grep -m1 "^\*\*Status:\*\*" "$f" | awk '{print $2}' | tr -d '[:space:]')
        [ "$status" = "pending" ] && echo "$f" && break
      done)

  [ -z "$TASK_FILE" ] && echo "nenhuma task pendente" && break

  PROMPT="You are executing the \"execute-task\" skill.

First, read AGENTS.md at the repository root to load governance rules and conventions.

Then read and follow the instructions in: .agents/skills/execute-task/SKILL.md

Target task file: ${TASK_FILE}
PRD folder: ${PRD_FOLDER}

Execute ONLY this task. Follow all skill steps:
1. Validate eligibility
2. Load context (prd.md, techspec.md)
3. Implement
4. Validate (tests, lint)
5. Review
6. Update task status in task file and tasks.md
7. Generate execution report

Update **Status:** in ${TASK_FILE} and the corresponding row in ${PRD_FOLDER}/tasks.md to reflect the final state."

  echo "executando: ${TASK_FILE}"
  case "$TOOL" in
    claude)  claude --dangerously-skip-permissions --print --bare -p "$PROMPT" ;;
    codex)   codex exec --dangerously-bypass-approvals-and-sandbox -p "$PROMPT" ;;
    gemini)  gemini --yolo -p "$PROMPT" ;;
    copilot) copilot --autopilot --yolo -p "$PROMPT" ;;
  esac

  count=$((count + 1))
done
```

Uso:

```bash
chmod +x run-tasks.sh
./run-tasks.sh tasks/prd-payments-list claude 3
```

Limitacoes deste script em relacao ao `task-loop`:
- nao valida dependencias entre tasks
- nao gera relatorio estruturado
- nao lida com `blocked`, `failed` ou `needs_input`
- nao tem timeout por task

### Alternativa 3 — `--max-iterations 1` como substituto do ciclo manual

```bash
# executar uma task, revisar, depois rodar novamente
ai-spec task-loop --tool claude --max-iterations 1 tasks/prd-payments-list
```

## Comparativo das abordagens

| Abordagem | Isolamento de sessao | Dependencias entre tasks | Timeout | Relatorio | Quando usar |
| --- | --- | --- | --- | --- | --- |
| `task-loop` completo | automatico | sim | sim | sim | bundle maduro, execucao sem supervisao |
| `task-loop --max-iterations 1` | automatico | sim | sim | sim | execucao task a task com pausa para revisao |
| invocacao direta por task | manual (um processo por chamada) | nao (voce controla a ordem) | nao | nao | task isolada, ambiguidade na spec |
| script shell | manual (um processo por iteracao) | parcial (so status pending) | nao nativo | nao | automacao leve sem dependencias complexas |
