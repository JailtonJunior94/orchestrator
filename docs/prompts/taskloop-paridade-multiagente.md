# Prompt enriquecido: paridade rigorosa do task loop multiagente

## Objetivo

Garantir, de forma rigorosa e critica, que `internal/taskloop` execute o fluxo de tasks de forma semanticamente equivalente para `claude`, `codex`, `copilot-cli` e `gemini`, preservando contratos publicos, evidencias de validacao e comportamento esperado entre ferramentas.

O fluxo deve executar tasks do bundle `tasks/prd-modernizacao-cli-task-loop-iterativo`, revisar o diff apos cada ciclo relevante e corrigir achados criticos antes de prosseguir.

## Prompt original resumido

Executar tasks usando `execute-task` com o modelo:

```text
Use a skill execute-task para implementar a task tasks/<prd-folder>/<N>_task.md.
```

Depois de concluir cada task, limpar a sessao e executar a proxima task disponivel. Para revisao, usar `review` sobre o diff atual. Se houver bug, usar `bugfix` para corrigir achados criticos.

## Ambiguidades tratadas

- "Funcionar de forma IGUAL" foi definido como paridade semantica observavel entre ferramentas, nao igualdade byte a byte de logs ou prompts internos.
- "Proxima task disponivel" foi definido como a proxima task ainda nao concluida no bundle informado, respeitando ordenacao numerica do nome do arquivo.
- "Limpar a sessao" foi definido como iniciar uma nova invocacao isolada por task, sem reutilizar memoria conversacional, mantendo apenas contexto persistido em arquivos.
- "Review a cada task" foi definido como revisao do diff acumulado apos a task implementada, antes de corrigir bugs ou avancar para outra task.
- "Bug" foi restringido a achados criticos da revisao ou regressao demonstravel por teste/validacao.
- O problema observado em `claude` e `codex` foi definido como falha de ciclo: a ferramenta nao encerra corretamente a task atual, nao limpa a sessao e nao inicia a proxima task disponivel. A causa raiz deve ser analisada antes de implementar qualquer correcao.

## Prompt enriquecido para implementacao do task loop

Use este prompt para orientar a implementacao rigorosa do fluxo em `internal/taskloop`.

```text
Voce esta trabalhando no repositorio `orchestrator`.

Objetivo:
Fazer `internal/taskloop` executar tasks de forma semanticamente equivalente para as ferramentas `claude`, `codex`, `copilot-cli` e `gemini`.

Definicao de paridade:
- A mesma task elegivel deve gerar o mesmo tipo de prompt operacional para todas as ferramentas.
- O ciclo implementacao -> revisao -> bugfix, quando aplicavel, deve existir para todas as ferramentas.
- O estado persistido de conclusao, evidencia e proxima task deve ser independente da ferramenta usada.
- Diferencas de adaptador, flags, auth, limite de contexto ou workaround especifico da ferramenta nao podem alterar a semantica do fluxo.
- Nao exigir igualdade byte a byte de output textual, apenas igualdade de contrato, estado final, evidencias e decisoes de controle.

Escopo obrigatorio:
- Superficie principal: `internal/taskloop`.
- Bundle de referencia: `tasks/prd-modernizacao-cli-task-loop-iterativo`.
- Ferramentas-alvo: `claude`, `codex`, `copilot-cli`, `gemini`.
- Manter compatibilidade com comandos e contratos publicos existentes.

Antes de alterar codigo:
1. Leia `AGENTS.md`.
2. Leia `.agents/skills/agent-governance/SKILL.md`.
3. Leia a task alvo antes de qualquer alteracao.
4. Inspecione a arquitetura atual de `internal/taskloop`, incluindo adapters, contratos, persistencia de estado, selecao de task, montagem de prompt e captura de evidencia.
5. Identifique explicitamente os contratos publicos que nao podem mudar sem ADR.

Problema observado que exige analise de causa raiz:
- Em `claude` e `codex`, o task loop nao esta terminando a task atual e iniciando a proxima task disponivel.
- Antes de propor correcao, determine se a causa esta em encerramento de subprocesso, criterio de sucesso/falha, persistencia de estado, deteccao de task concluida, limpeza de sessao, montagem do proximo prompt, diferenca de adapter por ferramenta ou tratamento de output.
- Registre a causa raiz encontrada e a evidencia que a comprova, como teste falhando, log, diff de estado persistido ou comportamento reproduzido.
- A correcao deve atacar a causa raiz, nao apenas adicionar uma chamada manual para avancar task.

Prompt dinamico de implementacao por task:

Use a skill execute-task para implementar a task tasks/<prd-folder>/<N>_task.md.

Contexto obrigatorio:
- Leia o arquivo de task antes de iniciar qualquer alteracao.
- Arquitetura: descreva a camada e os contratos relevantes de `internal/taskloop` antes da implementacao.
- Referencias a carregar: `go-implementation`, `ddd`, `tests` ou outras referencias somente quando a task exigir.

Criterios de execucao nao negociaveis:
- preservar contratos publicos existentes; nenhuma assinatura publica muda sem ADR.
- nenhuma interface nova sem fronteira real justificada.
- usar `context.Context` em todas as operacoes de IO novas ou alteradas.
- cobrir todos os cenarios do criterio de pronto com testes table-driven.
- registrar evidencia de conclusao no arquivo de task, incluindo output de teste e lint quando aplicavel.
- nao fechar a task sem evidencia de validacao.

Ao concluir uma task:
1. Persistir evidencia no arquivo da task.
2. Encerrar a invocacao corrente.
3. Iniciar uma nova invocacao isolada, sem depender de memoria conversacional anterior.
4. Selecionar a proxima task disponivel no bundle `tasks/prd-modernizacao-cli-task-loop-iterativo`, respeitando ordem numerica.
5. Repetir o mesmo fluxo ate nao haver task pendente ou ate ocorrer bloqueio registrado.

Prompt de revisao apos cada task ou lote coerente:

Use a skill review para revisar o diff atual.

Contexto da implementacao:
- Tasks executadas: liste as tasks do bundle executadas nesta invocacao.
- Skill usada na implementacao: `execute-task`.
- Areas de risco: performance, seguranca, contratos publicos, concorrencia, filesystem, subprocessos, isolamento de sessao e paridade entre ferramentas.

Focos obrigatorios da revisao:
- corretude: a implementacao atende todos os RFs e criterios de aceite do PRD?
- regressao: alguma mudanca quebra contrato publico ou comportamento existente?
- seguranca: ha injecao de dependencia insegura, dado sensivel exposto ou validacao faltando?
- testes: todos os cenarios do criterio de pronto estao cobertos?
- divida tecnica introduzida: o que precisara de refactor futuro?

Saidas esperadas da revisao:
- lista de achados por categoria: critico, importante, sugestao.
- para cada achado critico: arquivo, linha, descricao e correcao sugerida.
- veredicto final: aprovado, aprovado com ressalvas ou reprovado.

Se a revisao encontrar achados criticos, use o prompt de bugfix:

Use a skill bugfix para corrigir os achados criticos da revisao.

Achados a corrigir, conforme saida da skill review:
- [arquivo:linha] Achado 1: [descricao do problema]
- [arquivo:linha] Achado 2: [descricao do problema]

Comportamento esperado apos a correcao:
- Descreva o comportamento correto esperado para cada achado.

Invariantes que nao podem mudar:
- contratos publicos: liste assinaturas, comandos, tipos ou formatos persistidos que nao podem ser alterados.
- tipos de erro: liste erros que o restante do sistema depende.
- comportamento de outros fluxos nao afetados pelos achados.

Regras de execucao nao negociaveis para bugfix:
- identificar a causa raiz de cada achado antes de escrever qualquer linha de codigo.
- adicionar ou corrigir testes de regressao que provem que o bug nao pode regredir.
- nao alterar comportamento fora do escopo dos achados listados.
- ao finalizar, rodar testes, rodar lint ou vet conforme custo e risco, e registrar o output como evidencia.
- nao declarar o bugfix concluido sem evidencia de que a causa raiz foi eliminada.

Saidas esperadas do bugfix:
- diff com a correcao minima necessaria.
- testes de regressao adicionados ou corrigidos.
- evidencia de validacao, incluindo `go test ./...` e `go vet ./...` quando proporcionais ao risco.
- descricao da causa raiz de cada achado corrigido.

Criterios de aceite finais para `internal/taskloop`:
- Existe teste cobrindo equivalencia do fluxo para `claude`, `codex`, `copilot-cli` e `gemini`.
- A selecao da proxima task nao depende da ferramenta.
- A limpeza de sessao entre tasks e explicita, testavel e documentada no comportamento do task loop.
- O bug observado em `claude` e `codex` esta reproduzido por teste ou evidencia controlada, com causa raiz documentada antes da correcao.
- Apos concluir uma task em `claude` e `codex`, o fluxo encerra a invocacao atual, limpa a sessao e inicia a proxima task disponivel sem depender de intervencao manual.
- A montagem dos prompts de implementacao, revisao e bugfix e deterministica a partir de task, bundle e contexto persistido.
- Falhas de uma ferramenta geram estado e evidencia consistentes, sem mascarar erro como sucesso.
- Nenhuma alteracao de contrato publico foi feita sem ADR.
- Toda task concluida contem evidencia de validacao.

Formato de resposta final esperado:
- Resumo curto das tasks executadas.
- Arquivos alterados.
- Evidencia de validacao com comandos e resultado.
- Achados de revisao e status de correcao.
- Bloqueios ou suposicoes, se houver.
```

## Justificativa das adicoes

| Adicao | Motivo |
|--------|--------|
| Definicao de paridade semantica | Evita exigir igualdade textual impossivel entre CLIs diferentes e foca no contrato verificavel. |
| Escopo explicito em `internal/taskloop` | Reduz risco de refatoracao ampla fora do objetivo. |
| Sequencia implementacao, revisao e bugfix | Transforma o pedido em um fluxo operacional deterministico. |
| Isolamento de sessao por task | Formaliza a limpeza de contexto exigida pelo objetivo. |
| Analise de causa raiz para Claude e Codex | Garante que a falha de nao encerrar uma task e iniciar a proxima seja reproduzida e corrigida na origem. |
| Criterios de aceite finais | Torna mensuravel quando a mudanca esta pronta. |
| Formato de resposta final | Garante evidencia e rastreabilidade para fechamento. |

## Variante curta para uso direto

```text
Use a skill execute-task para implementar a proxima task pendente em `tasks/prd-modernizacao-cli-task-loop-iterativo`, garantindo que `internal/taskloop` mantenha paridade semantica entre `claude`, `codex`, `copilot-cli` e `gemini`.

Antes de corrigir, analise a causa raiz do problema observado em `claude` e `codex`: a task atual nao termina, a sessao nao e limpa e a proxima task disponivel nao inicia. Verifique encerramento de subprocesso, criterio de sucesso/falha, persistencia de estado, deteccao de task concluida, limpeza de sessao, montagem do proximo prompt e diferencas entre adapters.

Antes de editar, leia `AGENTS.md`, `.agents/skills/agent-governance/SKILL.md` e o arquivo da task. Preserve contratos publicos, nao crie interfaces sem fronteira real, use `context.Context` em IO, cubra criterios de pronto com testes table-driven e registre evidencia no arquivo da task.

Apos concluir a task, revise o diff com a skill `review`. Se houver achados criticos, corrija com a skill `bugfix`, adicionando testes de regressao e evidencia de validacao. Depois, limpe a sessao e execute a proxima task pendente do bundle.
```
