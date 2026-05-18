# Biblioteca de Prompts

> Prompts canonicos para uso com as skills centrais do `orchestrator`.
>
> Este documento prioriza prompts copiaveis, com baixo desvio e foco em execucao deterministica. Para contratos completos de cada skill, consulte o [Guia de uso das skills](skills-usage-guide.md).

## Como Usar Esta Biblioteca

Todo prompt efetivo deve deixar explicito:

1. a skill ou etapa desejada
2. o contexto obrigatorio a ser lido
3. as restricoes nao negociaveis
4. o formato de saida esperado

Use o menor prompt que ainda mantenha o agente dentro do trilho. Se o contexto estiver instavel, suba do minimo para o robusto em vez de pedir "mais capricho".

## Prompts de Planejamento

### `create-prd`

#### Prompt Minimo

```text
Use a skill create-prd para definir a feature [nome da feature].

Entradas:
- problema: [descricao objetiva]
- persona afetada: [quem sente o problema]
- fora de escopo: [o que nao entra]
- restricoes tecnicas: [limites conhecidos]

Quero no resultado:
- problema e objetivo
- requisitos funcionais numerados
- requisitos nao funcionais
- criterios de aceite
- riscos e pontos em aberto
```

#### Prompt Robusto

```text
Use a skill create-prd para definir a feature [nome da feature].

Contexto:
- repositorio: [nome ou caminho]
- problema de negocio: [descricao objetiva]
- persona ou sistema afetado: [quem sofre o problema]
- valor esperado: [resultado esperado se a feature funcionar]
- fora de escopo: [o que esta explicitamente excluido]
- restricoes tecnicas: [performance, integracoes, compliance, rollout]

Regras:
- nao desenhar implementacao nesta etapa
- nao misturar requisito com solucao tecnica
- explicitar suposicoes quando o contexto nao fechar

Quero no resultado:
- problema claro e verificavel
- objetivos e nao objetivos
- RFs numerados com criterio de aceite
- RNFs mensuraveis
- riscos, mitigacoes e questoes em aberto
```

### `create-technical-specification`

#### Prompt Minimo

```text
Use a skill create-technical-specification com base no PRD aprovado em `tasks/<prd-folder>/prd.md`.

Contexto tecnico:
- stack: [linguagem, framework, banco]
- arquitetura atual: [camadas e contratos relevantes]
- restricoes: [o que nao pode mudar]

Quero no resultado:
- desenho de implementacao
- interfaces e arquivos afetados
- estrategia de erros
- estrategia de testes
```

#### Prompt Robusto

```text
Use a skill create-technical-specification com base no PRD aprovado em `tasks/<prd-folder>/prd.md`.

Contexto tecnico obrigatorio:
- stack: [linguagem, framework, banco, runtime]
- arquitetura atual: [handler -> service -> repository, hexagonal, etc.]
- contratos publicos existentes: [endpoints, CLI, arquivos, interfaces]
- fronteiras que devem ser preservadas: [modulos, pacotes, integracoes]
- referencias necessarias: [ddd, error-handling, testing, seguranca]

Regras:
- nao alterar comportamento publico sem justificativa explicita
- definir responsabilidades por camada antes de propor arquivos
- incluir riscos tecnicos e estrategia de validacao

Quero no resultado:
- modelagem e fluxo de implementacao
- contratos e pontos de integracao
- riscos, trade-offs e ADRs necessarias
- plano de testes e validacoes
```

### `create-tasks`

#### Prompt Minimo

```text
Use a skill create-tasks para decompor a spec aprovada em tasks executaveis.

Arquivos de entrada:
- `tasks/<prd-folder>/prd.md`
- `tasks/<prd-folder>/techspec.md`

Quero no resultado:
- `tasks.md` com ordem e dependencias
- `task-*.md` pequenos, claros e testaveis
```

#### Prompt Robusto

```text
Use a skill create-tasks para decompor a spec aprovada em tasks executaveis.

Arquivos de entrada:
- PRD: `tasks/<prd-folder>/prd.md`
- Tech spec: `tasks/<prd-folder>/techspec.md`

Regras de decomposicao:
- uma responsabilidade principal por task
- dependencias explicitas e sem ciclo
- criterio de sucesso verificavel por task
- testes e validacoes declarados em cada task
- nao gerar tasks vagas, abertas ou grandes demais para uma sessao

Quero no resultado:
- `tasks.md` com ordem, paralelismo seguro e skills necessarias
- um task file por item com objetivo, arquivos relevantes, testes e criterio de pronto
```

## Prompt Unitario para `execute-task`

Use estas variantes para executar uma unica task por vez.

### Versao Curta

```text
Use a skill execute-task para implementar exatamente a task `tasks/prd-<slug>/<task-file>.md`.

Leia antes de agir:
- `AGENTS.md`
- `tasks/prd-<slug>/prd.md`
- `tasks/prd-<slug>/techspec.md`
- `tasks/prd-<slug>/tasks.md`
- `tasks/prd-<slug>/<task-file>.md`

Regras:
- execute somente o escopo da task
- preserve arquitetura e contratos publicos
- implemente testes junto com a mudanca
- rode validacao proporcional
- so conclua com evidencia real

Quero no resultado:
- implementacao da task
- testes e validacoes executados
- execution report
- status final: `done`, `blocked`, `failed` ou `needs_input`
```

### Versao Padrao

Use como default. E o melhor equilibrio entre clareza, custo e confianca.

```text
Use a skill execute-task para implementar exatamente a task `tasks/prd-<slug>/<task-file>.md`.

Contexto obrigatorio:
- Leia `AGENTS.md` na raiz do repositorio antes de qualquer acao.
- Leia por completo:
  - `tasks/prd-<slug>/prd.md`
  - `tasks/prd-<slug>/techspec.md`
  - `tasks/prd-<slug>/tasks.md`
  - `tasks/prd-<slug>/<task-file>.md`
- Execute somente o escopo definido no task file.
- Preserve contratos publicos, arquitetura atual e convencoes do repositorio.
- Nao introduza abstracoes, dependencias ou mudancas fora da task sem necessidade explicita.
- Carregue apenas as skills e referencias realmente necessarias para esta task.

Criterios de execucao nao negociaveis:
- implementar producao e testes juntos
- seguir exatamente os criterios de sucesso e testes definidos no task file
- rodar validacao proporcional antes de concluir
- registrar evidencias reais de validacao
- atualizar status apenas se a task estiver realmente concluida

Quero no resultado:
- implementacao completa da task, sem expandir escopo
- testes e validacoes executados com resultado explicito
- caminho do execution report gerado
- status final canonico: `done`, `blocked`, `failed` ou `needs_input`
- se houver bloqueio, explicar objetivamente a causa e nao improvisar fora da spec
```

### Versao Rigorosa

Use em tasks criticas, sensiveis ou quando houve historico de desvio.

```text
Use a skill execute-task para implementar exatamente a task `tasks/prd-<slug>/<task-file>.md`.

Leitura obrigatoria antes de qualquer edicao:
- `AGENTS.md`
- `tasks/prd-<slug>/prd.md`
- `tasks/prd-<slug>/techspec.md`
- `tasks/prd-<slug>/tasks.md`
- `tasks/prd-<slug>/<task-file>.md`

Restricoes mandatorias:
- nao saia do escopo do task file
- nao mude contratos publicos sem justificativa explicita na spec
- nao crie abstracoes, dependencias ou refactors laterais sem demanda concreta da task
- nao pule testes
- nao marque a task como concluida sem evidencia real de validacao
- se encontrar ambiguidade material, pare com `needs_input` em vez de improvisar

Criterios de execucao:
- implementar codigo e testes juntos
- seguir exatamente os criterios de sucesso, subtarefas e testes definidos no task file
- rodar primeiro validacao direcionada e depois validacao proporcional ao risco
- registrar no resultado os comandos executados e o desfecho de cada validacao
- manter o diff restrito ao minimo necessario para concluir a task

Formato de saida esperado:
- resumo curto do que foi implementado
- lista dos testes e validacoes executados
- caminho do execution report
- status final canonico: `done`, `blocked`, `failed` ou `needs_input`
- se nao for `done`, explicar a causa raiz objetiva do bloqueio
```

### Como Escolher a Variante

| Situacao | Variante recomendada |
| --- | --- |
| task bem especificada, baixo risco | versao curta |
| execucao normal, uso diario | versao padrao |
| task critica, ambigua ou com historico de desvio | versao rigorosa |

## Prompts para Escalada de Execucao

### `task-loop`

`task-loop` e um fluxo de CLI, nao uma skill para ser chamada diretamente no prompt. A entrada canonica aqui e o comando, nao uma instrucao textual ao agente:

```bash
ai-spec task-loop --tool codex --max-iterations 2 tasks/prd-<slug>
```

Use esse modo somente quando:

- o bundle ja estiver maduro o suficiente para lote curto
- a ordem e as dependencias em `tasks.md` estiverem claras
- a primeira task relevante ja tiver sido validada via `execute-task`

### `execute-all-tasks`

#### Prompt Minimo

```text
Use a skill execute-all-tasks para executar o PRD `tasks/prd-<slug>/` inteiro.

Contexto:
- o bundle ja passou por task unitaria e esta maduro para throughput
- dependencias e paralelismo ja estao declarados em `tasks.md`

Regras:
- respeite o DAG do bundle
- use subagentes fresh por task
- pare na primeira falha bloqueante

Quero no resultado:
- resumo por wave ou task
- status final consolidado
- checkpoints, execution reports e bloqueios relevantes
```

## Antipadroes que Aumentam Desvio

Evite:

- pedir varias etapas no mesmo prompt, como `create-prd + create-tasks + execute-task`
- usar frases vagas como "faca tudo", "analise completo" ou "resolva ponta a ponta"
- omitir paths exatos de `prd.md`, `techspec.md`, `tasks.md` e `task-*.md`
- pedir para o agente "decidir a melhor arquitetura" quando a etapa ainda e de execucao de task
- autorizar implicitamente expansao de escopo com frases como "ajuste o que for preciso ao redor"
- pedir conclusao sem evidencias, por exemplo "se parecer ok, marque como done"

## Exemplo Real Neste Repositorio

Exemplo para a task `4.0` do bundle `governance-playbook-evolution`:

```text
Use a skill execute-task para implementar exatamente a task `tasks/prd-governance-playbook-evolution/task-4.0-prompt-library.md`.

Leitura obrigatoria antes de qualquer edicao:
- `AGENTS.md`
- `tasks/prd-governance-playbook-evolution/prd.md`
- `tasks/prd-governance-playbook-evolution/techspec.md`
- `tasks/prd-governance-playbook-evolution/tasks.md`
- `tasks/prd-governance-playbook-evolution/task-4.0-prompt-library.md`

Restricoes mandatorias:
- nao saia do escopo do task file
- nao mude contratos publicos sem justificativa explicita na spec
- nao crie abstracoes, dependencias ou refactors laterais sem demanda concreta da task
- nao pule testes
- nao marque a task como concluida sem evidencia real de validacao
- se encontrar ambiguidade material, pare com `needs_input` em vez de improvisar

Formato de saida esperado:
- resumo curto do que foi implementado
- lista dos testes e validacoes executados
- caminho do execution report
- status final canonico: `done`, `blocked`, `failed` ou `needs_input`
```

## Notas Operacionais

- Para uma unica task, prefira `execute-task` direto.
- Para lote pequeno e ainda incerto, use `task-loop` com poucas iteracoes.
- Para PRD inteiro com DAG madura, use `execute-all-tasks`.
- Se a task nao estiver estavel o suficiente para a versao curta, suba imediatamente para a versao padrao ou rigorosa.
