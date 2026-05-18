# Scorecard de Qualidade e Confianca

> Critério canônico para avaliar se um bundle `prd.md + techspec.md + tasks.md` está pronto para execução com agentes.
>
> Este documento mede qualidade e readiness. Ele não substitui os contratos detalhados do [Guia de uso das skills](skills-usage-guide.md) nem a visão executiva do [README](../README.md).

## Objetivo

Usar um critério curto, repetível e objetivo para classificar um bundle em três estados:

- `pass`: pronto para executar com risco controlado
- `warning`: executável, mas com lacunas que aumentam retrabalho
- `fail`: não executar antes de corrigir a base documental

## Quando Usar

Use o scorecard:

1. antes da primeira execução com `execute-task`
2. antes de trocar de `execute-task` para `task-loop` ou `execute-all-tasks`
3. quando houver dúvida se o bundle está suficientemente estável
4. em revisão de governança ou auditoria de prontidão

## Modelo de Pontuação

Cada dimensão recebe um peso. Classifique a dimensão e converta em pontos:

| Status | Pontos atribuídos |
| --- | --- |
| `pass` | 100% do peso |
| `warning` | 50% do peso |
| `fail` | 0% do peso |

### Pesos por Dimensão

| Dimensão | Peso |
| --- | --- |
| Qualidade do PRD | 25 |
| Qualidade da Tech Spec | 25 |
| Qualidade das Tasks | 20 |
| Readiness de Execução | 20 |
| Evidência e Rastreabilidade | 10 |
| **Total** | **100** |

## Dimensões de Avaliação

### 1. Qualidade do PRD

O PRD deve definir o problema, o escopo e os critérios de aceite sem deixar a execução dependente de inferência.

| Status | Critérios objetivos |
| --- | --- |
| `pass` | Problema, objetivos, fora de escopo, RFs e RNFs estão claros; cada RF é testável; riscos principais estão nomeados. |
| `warning` | O problema está claro, mas há RF sem critério testável, risco pouco explicitado ou fora de escopo incompleto. |
| `fail` | O PRD não delimita o problema, mistura solução com descoberta ou deixa lacunas que exigem improviso material. |

**Evidências mínimas esperadas**

- RFs numerados e coerentes com a entrega
- critérios de aceite verificáveis
- fora de escopo explícito

### 2. Qualidade da Tech Spec

A tech spec deve transformar o PRD em um plano técnico executável sem mudar contratos por acidente.

| Status | Critérios objetivos |
| --- | --- |
| `pass` | Há decisões de implementação, arquivos e integrações relevantes, estratégia de validação e mapeamento PRD → implementação. |
| `warning` | A direção técnica existe, mas falta clareza em validações, impacto por arquivo ou fronteiras de mudança. |
| `fail` | A spec não orienta a implementação, contradiz o PRD ou deixa decisões críticas em aberto sem registrar bloqueio. |

**Evidências mínimas esperadas**

- resumo executivo coerente com o PRD
- arquivos relevantes ou superfícies afetadas
- abordagem de testes ou validação proporcional

### 3. Qualidade das Tasks

As tasks devem ser pequenas, ordenadas e executáveis em uma sessão de agente sem expandir escopo.

| Status | Critérios objetivos |
| --- | --- |
| `pass` | `tasks.md` explicita dependências; cada task tem objetivo, subtarefas, critérios de sucesso, testes e arquivos relevantes. |
| `warning` | A decomposição existe, mas há tasks grandes demais, dependências pouco claras ou critérios de pronto genéricos. |
| `fail` | Há tasks ambíguas, sem testes definidos, sem dependências confiáveis ou impossíveis de executar isoladamente. |

**Evidências mínimas esperadas**

- coluna `Skills` coerente entre `tasks.md` e task file
- critérios de sucesso objetivos
- comandos ou validações definidos por task

### 4. Readiness de Execução

O bundle precisa estar operacionalmente pronto para um agente executar sem depender de adivinhação de fluxo.

| Status | Critérios objetivos |
| --- | --- |
| `pass` | Binário `ai-spec`, artefatos obrigatórios e sequência operacional estão claros; a escolha entre `execute-task`, `task-loop` e `execute-all-tasks` é objetiva. |
| `warning` | O fluxo é usável, mas há fricção operacional: prompts incompletos, preflight disperso ou escolha de modo ainda incerta. |
| `fail` | Não está claro como executar, quais gates rodar ou qual skill usar; o executor depende de tentativa e erro. |

**Evidências mínimas esperadas**

- prompt ou instrução operacional copiável
- checklist de preflight ou gates equivalentes
- regra objetiva para escolher o modo de execução

### 5. Evidência e Rastreabilidade

O bundle deve permitir provar o que foi planejado, executado e validado.

| Status | Critérios objetivos |
| --- | --- |
| `pass` | PRD, tech spec, tasks e relatórios se referenciam corretamente; há hash de spec quando aplicável e evidência real de validação. |
| `warning` | Existe rastreabilidade parcial, mas faltam hashes, relatório de execução ou prova explícita de validação. |
| `fail` | Não é possível provar relação entre requisito, task e validação; o bundle depende de relato informal. |

**Evidências mínimas esperadas**

- `spec-hash-prd` e hashes relacionados quando o fluxo exigir
- relatório de execução ou evidência equivalente
- comandos de validação com resultado observável

## Interpretação do Resultado

Depois de somar a pontuação:

| Faixa | Classificação |
| --- | --- |
| 85 a 100 | `pass` |
| 60 a 84 | `warning` |
| abaixo de 60 | `fail` |

### Hard Stops

Mesmo com pontuação alta, não execute o bundle como `pass` se qualquer um destes itens ocorrer:

- PRD ou tech spec contradiz o escopo da task
- task não define testes nem critérios de sucesso
- fluxo de execução depende de improviso material
- não existe evidência mínima de validação ou rastreabilidade

Nesses casos, o resultado operacional é `fail`.

## Como Aplicar na Prática

1. Leia `prd.md`, `techspec.md`, `tasks.md` e pelo menos a task imediata.
2. Classifique cada dimensão como `pass`, `warning` ou `fail`.
3. Some os pontos usando os pesos deste documento.
4. Verifique se houve algum hard stop.
5. Registre a decisão antes de executar.

### Modelo de Registro

```text
Bundle: tasks/prd-<slug>/
PRD: pass (25/25)
Tech Spec: warning (12,5/25)
Tasks: pass (20/20)
Readiness de Execução: warning (10/20)
Evidência e Rastreabilidade: pass (10/10)
Total: 77,5/100
Classificação final: warning
Decisão: executar somente via execute-task até reduzir lacunas
```

## Uso Recomendado por Faixa

| Classificação | Ação recomendada |
| --- | --- |
| `pass` | Pode seguir com `execute-task`; considere `task-loop` ou `execute-all-tasks` só se a decomposição também estiver madura. |
| `warning` | Execute uma task por vez com `execute-task` e trate lacunas antes de aumentar throughput. |
| `fail` | Corrija PRD, tech spec, tasks ou evidências antes de qualquer execução. |

## Limites Deliberados

- Este scorecard não redefine contratos de cada skill.
- Este scorecard não substitui review técnico ou validação real.
- Este scorecard mede prontidão operacional; não garante sucesso da implementação.
