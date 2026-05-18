# Checklist de Preflight e Readiness

> Fonte canonica para decidir se voce pode executar agora ou se deve parar antes de chamar `execute-task`, `task-loop` ou `execute-all-tasks`.
>
> Use este checklist junto com o [Playbook Mestre de Desenvolvimento](development-playbook.md) e o [Scorecard de Qualidade e Confianca](quality-scorecard.md). Este documento nao substitui a `SKILL.md` ativa; ele reduz a triagem operacional antes da execucao.

## Objetivo

Consolidar os checks que evitam falso positivo operacional:

1. confirmar que o binario e os artefatos existem
2. detectar drift de spec e lock de skills antes da execucao
3. separar o que bloqueia de verdade do que apenas recomenda cautela
4. adaptar o preflight ao modo escolhido: task unica, lote curto ou orquestracao completa

## Sequencia Curta e Executavel

Use esta sequencia antes de executar qualquer fluxo:

```bash
command -v ai-spec
test -f tasks/prd-<slug>/prd.md
test -f tasks/prd-<slug>/techspec.md
test -f tasks/prd-<slug>/tasks.md
ai-spec skills check
ai-spec check-spec-drift tasks/prd-<slug>/tasks.md
```

Se qualquer comando acima falhar, pare. Corrija a causa raiz antes de seguir para a execucao.

## Checks Bloqueantes

Os itens abaixo sao gate de entrada. Se falharem, o status operacional e "nao executar".

| Check | Como verificar | Se falhar |
| --- | --- | --- |
| Binario `ai-spec` disponivel | `command -v ai-spec` | pare; instale o binario antes de continuar |
| Bundle minimo existe | `test -f tasks/prd-<slug>/prd.md && test -f tasks/prd-<slug>/techspec.md && test -f tasks/prd-<slug>/tasks.md` | pare; faltam artefatos obrigatorios |
| Lock de skills verificavel | `ai-spec skills check` | pare se o comando retornar erro; trate drift ou instalacao incompleta |
| Spec sem drift | `ai-spec check-spec-drift tasks/prd-<slug>/tasks.md` | pare; bundle fora de sincronia entre PRD, tech spec e tasks |
| Task file alvo existe | `test -f tasks/prd-<slug>/task-X.Y-<nome>.md` | pare; nao execute `execute-task` sem task file real |

## Checks Recomendados

Os itens abaixo nao substituem os gates bloqueantes. Se falharem, a execucao ainda pode ser possivel, mas com risco maior.

| Check | Como verificar | Impacto se ignorar |
| --- | --- | --- |
| Ferramenta autenticada e pronta | validar a sessao interativa da tool antes de subprocessos ou wrappers | risco de timeout, login interativo ou falha no meio da execucao |
| Sessao fresca para lote | confirmar ausencia de subagentes ativos ou contexto residual | risco de confusao de contexto e concorrencia acidental |
| `Paralelizável` honesto | revisar `tasks.md` antes de `execute-all-tasks` | risco de conflito entre tasks ou lock desnecessario |
| Prompt canonico em maos | copiar da [Biblioteca de Prompts](prompt-library.md) | risco de desvio de escopo e resposta incompleta |
| Telemetria habilitada quando fizer sentido | `echo ${GOVERNANCE_TELEMETRY:-0}` | perde trilha adicional de observabilidade do fluxo |

## Preflight por Fluxo

### 1. Antes de `execute-task`

Use quando voce vai executar uma unica task ou ainda precisa medir qualidade real.

- [ ] `tasks/prd-<slug>/prd.md`, `techspec.md`, `tasks.md` e o `task-*.md` alvo existem
- [ ] `ai-spec skills check` retorna 0
- [ ] `ai-spec check-spec-drift tasks/prd-<slug>/tasks.md` retorna 0
- [ ] a task alvo tem criterio de sucesso e testes explicitados
- [ ] nao ha ambiguidade material aberta na spec

Sinal de prosseguir: voce consegue apontar a task exata e os comandos de validacao sem improviso.

### 2. Antes de `task-loop`

Use quando o lote e pequeno ou progressivo e ainda pede supervisao frequente.

- [ ] todos os gates de `execute-task` passaram
- [ ] `tasks.md` tem dependencias explicitas e sem ciclo aparente
- [ ] as proximas tasks elegiveis estao pequenas o suficiente para uma sessao cada
- [ ] o lote pode ser executado com pausa de revisao entre iteracoes
- [ ] o `dry-run` ou a leitura manual do bundle nao indica status invalido

Sinal de prosseguir: o bundle esta utilizavel em lote curto, mas voce ainda quer controlar ritmo e observacao.

### 3. Antes de `execute-all-tasks`

Use somente quando o PRD inteiro ja estiver maduro e o paralelismo for real.

- [ ] todos os gates de `task-loop` passaram
- [ ] a primeira task relevante ja foi validada na pratica ou o bundle esta claramente maduro
- [ ] a coluna `Paralelizável` esta revisada com honestidade
- [ ] a ferramenta escolhida esta pronta para subagentes ou subprocessos no ambiente atual
- [ ] nao existe ambiguidade material em tasks que serao executadas em paralelo

Sinal de prosseguir: o custo de coordenacao manual ja ficou maior que o custo de orquestrar o DAG inteiro.

## Como Classificar o Resultado

| Estado | Significado | Acao |
| --- | --- | --- |
| `pass` | todos os checks bloqueantes passaram e os recomendados nao levantam risco relevante | pode executar no fluxo escolhido |
| `warning` | os gates passaram, mas ha risco operacional moderado | prefira `execute-task` ou reduza o lote |
| `fail` | um ou mais checks bloqueantes falharam | nao execute; corrija a base antes |

## Comandos Reais Deste Repositorio

Os comandos abaixo refletem a CLI instalada hoje neste repositorio:

```bash
ai-spec skills check
ai-spec check-spec-drift tasks/prd-<slug>/tasks.md
ai-spec lint .
```

Nota importante: este ambiente expõe `ai-spec skills check`, nao `ai-spec skills --verify`. Para preflight operacional, trate `skills check` como o comando canonico real do binario atual.

## Quando Parar Sem Improvisar

Pare e volte para a base documental se qualquer um destes sinais aparecer:

- PRD, tech spec e tasks apontam direcoes diferentes
- a task nao define teste nem criterio de sucesso verificavel
- o fluxo depende de adivinhar qual skill usar ou em que ordem executar
- o ambiente da ferramenta nao suporta autenticacao ou subprocesso no modo escolhido

## Referencias de Apoio

- [Playbook Mestre de Desenvolvimento](development-playbook.md)
- [Scorecard de Qualidade e Confianca](quality-scorecard.md)
- [Biblioteca de Prompts](prompt-library.md)
- [Referencia do task-loop](task-loop-reference.md)
- [Guia Completo: `execute-all-tasks`](execute-all-tasks-guide.md)
- [Matriz de degradacao](degradation-matrix.md)
- [Guia de troubleshooting](troubleshooting.md)
