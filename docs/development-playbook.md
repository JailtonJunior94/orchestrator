# Playbook Mestre de Desenvolvimento

> Fonte operacional principal para escolher o fluxo certo entre planejamento, execucao unitaria e execucao em lote.
>
> Use o [README](../README.md) como entrada executiva. Use este playbook para decidir a proxima etapa e os guias especializados apenas quando precisar de detalhe operacional.

## Objetivo

Reduzir a decisao a poucos passos:

1. entender em que etapa do fluxo voce esta
2. escolher a skill certa para esta etapa
3. evitar subir para lote antes de o bundle estar maduro

## Fluxo Mestre

O fluxo recomendado parte do menor contexto necessario e so aumenta throughput quando o bundle ja provou qualidade.

| Etapa | Quando usar | Saida esperada |
| --- | --- | --- |
| `create-prd` | o problema, escopo ou criterio de sucesso ainda nao estao fechados | `prd.md` com RFs, RNFs, criterios de aceite e fora de escopo |
| `create-technical-specification` | o PRD foi aprovado e falta transformar requisito em plano tecnico | `techspec.md` com decisoes, arquivos afetados, riscos e validacao |
| `create-tasks` | a tech spec ja esta executavel e precisa ser fatiada | `tasks.md` + `task-*.md` pequenos, ordenados e validaveis |
| `execute-task` | voce vai executar uma task isolada, sensivel, exploratoria ou ainda incerta | implementacao de uma unica task com testes, review e execution report |
| `task-loop` | existe um lote pequeno ou progressivo de tasks maduras e voce quer throughput controlado | execucao iterativa com pausas e lote curto |
| `execute-all-tasks` | o PRD inteiro esta maduro, com DAG clara e paralelismo honesto | orquestracao completa com subagentes fresh e halt-first |

## Regra de Escalada

Suba de etapa somente quando a anterior estiver fechada:

- nao use `create-technical-specification` sem `prd.md` aprovado
- nao use `create-tasks` sem `techspec.md` coerente
- nao use `execute-task` sem `tasks.md` e task file validos
- nao use `task-loop` ou `execute-all-tasks` enquanto a primeira task relevante ainda falha, muda demais de escopo ou depende de improviso

## Arvore de Decisao

```text
O trabalho ainda nao tem escopo e criterio de sucesso claros?
-> Sim: use create-prd
-> Nao:
   A arquitetura, contratos e validacoes ainda estao em aberto?
   -> Sim: use create-technical-specification
   -> Nao:
      As tasks ainda nao foram decompostas ou estao grandes demais?
      -> Sim: use create-tasks
      -> Nao:
         Voce vai executar uma unica task, ou ainda precisa medir qualidade real?
         -> Sim: use execute-task
         -> Nao:
            O lote e pequeno, progressivo ou ainda pede supervisao frequente?
            -> Sim: use task-loop
            -> Nao: use execute-all-tasks
```

## Como Escolher o Modo de Execucao

### 1. Execucao Unitaria

Use `execute-task` como default quando:

- ha apenas uma task a executar
- a task toca contrato sensivel, fronteira arquitetural ou area ainda pouco estabilizada
- voce quer medir qualidade real de testes, review e evidencias antes de aumentar throughput
- o bundle ainda esta em `warning` no scorecard

Sinal de prontidao: o caminho da task esta claro, os testes da task existem e a validacao cabe em uma sessao unica.

### 2. Lote Pequeno

Use `task-loop` quando:

- ha poucas tasks elegiveis e independentes
- voce quer avancar em lotes curtos, com possibilidade de parar e revisar entre iteracoes
- a decomposicao ja esta boa, mas o bundle ainda nao justificou orquestracao completa

Sinal de prontidao: `tasks.md` ordenado, dependencias claras, `dry-run` util e baixa chance de conflito entre tasks consecutivas.

### 3. Orquestracao Completa

Use `execute-all-tasks` quando:

- o PRD inteiro precisa ser executado
- existem pelo menos algumas tasks maduras, com dependencias declaradas e paralelismo confiavel
- o custo de abrir sessao por task e menor que o custo de supervisao manual

Sinal de prontidao: bundle em estado `pass` ou proximo disso, primeira task ja validada na pratica e nenhuma ambiguidade material aberta na spec.

## Diferenca Entre `task-loop` e `execute-all-tasks`

| Pergunta | `task-loop` | `execute-all-tasks` |
| --- | --- | --- |
| Qual problema resolve melhor? | ganho incremental com supervisao frequente | throughput maximo sobre um PRD inteiro |
| Unidade de controle | iteracoes curtas | waves do DAG |
| Quando usar | lote pequeno, ainda observando qualidade | bundle maduro, dependencias e paralelismo confiaveis |
| Tolerancia a incerteza | maior | menor |
| Requisito operacional | `tasks.md` utilizavel | `tasks.md` canonico e pronto para orquestracao |

## Navegacao Minima do Nucleo Operacional

Depois de decidir o fluxo, use estes pontos de apoio:

- prompts copiaveis: [Biblioteca de prompts](prompt-library.md)
- checks obrigatorios antes de executar: [Checklist de Preflight e Readiness](preflight-checklist.md)
- criterio rapido de prontidao do bundle: [Scorecard de qualidade e confianca](quality-scorecard.md)
- detalhes de lote progressivo: [Referencia do task-loop](task-loop-reference.md)
- detalhes da orquestracao completa: [Guia Completo: `execute-all-tasks`](execute-all-tasks-guide.md)

## Bundle Canonico de Referencia

O bundle de referencia deste projeto deve viver em `.specs/prd-<slug>/`, com `prd.md`, `techspec.md`, `tasks.md`, `task-*.md` e execution report rastreavel.

Se esse bundle ainda nao existir na branch atual, trate isso como sinal de que o fluxo documental ainda nao fechou ate a etapa exemplar. Nesta feature, o contrato desse bundle esta definido na task `6.0`, em `.specs/prd-governance-playbook-evolution/task-6.0-canonical-reference-bundle.md`.

## Sequencia Recomendada em Poucos Passos

1. Leia o `README.md` para identificar o ponto de entrada.
2. Use este playbook para decidir entre planejamento, task unica, lote pequeno ou orquestracao completa.
3. Confirme readiness com o scorecard e com o preflight.
4. Copie o prompt adequado e execute no modo escolhido.
5. So aumente throughput depois que a primeira execucao relevante produzir evidencia confiavel.

## Limites Deliberados

- Este playbook decide fluxo; ele nao substitui os contratos detalhados das skills.
- Este playbook aponta para prompts, preflight e bundle; ele nao replica esses conteudos integralmente.
- Se houver conflito entre este playbook e uma `SKILL.md`, a skill ativa continua sendo a fonte de verdade operacional.
