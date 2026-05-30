---
name: create-tasks
version: 1.7.0
description: Cria tarefas incrementais de implementação a partir de um PRD e de uma especificação técnica. Use quando documentos de produto e técnicos aprovados precisarem ser decompostos em itens de trabalho ordenados e testáveis, incluindo declaração de skills processuais especializadas necessárias por tarefa. Não use para mudanças diretas de código, descoberta de funcionalidade ou revisão de branch.
---

# Criar Tarefas

## Procedimentos

**Etapa 1: Validar os documentos de origem**
1. Confirmar que o contrato de carga base definido em `AGENTS.md` foi cumprido.
2. Confirmar que `.specs/prd-<feature-slug>/prd.md` e `.specs/prd-<feature-slug>/techspec.md` existem.
3. Ler os dois arquivos por completo antes de propor itens de trabalho.
4. Enumerar explicitamente todos os IDs `RF-nn` e `REQ-nn` presentes em `prd.md` e `techspec.md`, mantendo a origem de cada ID (`prd.md`, `techspec.md` ou ambos). Essa enumeração é a base obrigatória da cobertura.
5. Parar com `needs_input` se qualquer documento estiver ausente ou contraditório o suficiente para bloquear o planejamento.

**Etapa 2: Extrair fatias de entrega**
1. Identificar requisitos, decisões técnicas, pontos de integração, dependências e áreas de risco.
2. Garantir que cada ID `RF-nn`/`REQ-nn` enumerado na Etapa 1 seja coberto por pelo menos uma tarefa proposta. Nenhum requisito pode ficar apenas implícito em texto narrativo.
3. Agrupar o trabalho em fatias que entreguem valor verificável.
   - **Fatiamento materialmente ambíguo** (granularidade/ordem/paralelismo com trade-offs reais): aplicar `.agents/skills/agent-governance/references/multiple-choice-protocol.md` (2–5 opções, "(Recomendado)", uma pergunta por turno) antes de fixar o plano.
4. Preferir a sequência `domain -> interfaces/ports -> use cases -> adapters/repositories -> handlers -> integration`, salvo quando a especificação técnica justificar outra ordem.

**Etapa 3: Propor primeiro o plano de tarefas em alto nível**
1. Ler `assets/tasks-template.md` e `assets/task-template.md` antes de redigir.
2. Produzir uma lista de alto nível com no máximo `${AI_MAX_TASKS_PER_PRD:-10}` tarefas (configurável via env ou `.claude/config.yaml` chave `max_tasks_per_prd`). Default conservador é 10 para forçar consolidação de PRDs grandes em fatias coerentes; PRDs com escopo justificadamente maior podem aumentar — documentar a justificativa em `## Riscos de Integração` da `tasks.md`.
3. Para cada tarefa, incluir objetivo, entregável e dependências.
4. Parar e aguardar aprovação antes de gerar os arquivos finais.
5. Se a aprovação não estiver disponível na sessão atual, retornar `needs_input` e não escrever os arquivos de tarefa.

**Etapa 4: Gerar os artefatos detalhados de tarefa**
1. Após a aprovação, criar `.specs/prd-<feature-slug>/tasks.md` a partir de `assets/tasks-template.md`.
2. Preencher a tabela de `tasks.md` de modo que cada linha liste explicitamente os IDs `RF-nn`/`REQ-nn` cobertos pela tarefa (na coluna/descrição de requisitos do template ou no texto da tarefa quando o template não tiver coluna dedicada).
3. Validar a cobertura antes de criar os arquivos individuais: cada ID enumerado na Etapa 1 deve aparecer em pelo menos uma linha da tabela de `tasks.md`.
4. Criar exatamente um arquivo `task-*.md` para cada linha da tabela de `tasks.md`, usando `assets/task-template.md`; não deixar linha sem arquivo correspondente nem arquivo órfão sem linha na tabela.
5. Dar a cada tarefa critérios de aceitação explícitos, arquivos relevantes e expectativas de teste.
6. Garantir que cada tarefa seja executável de forma independente e revisável objetivamente.
7. Não escrever placeholders `PENDING-RUN` para hashes. Ao final da geração de `tasks.md`, sincronizar os comentários de rastreabilidade via CLI portátil usando o caminho do arquivo:
   - `ai-spec sync-spec-hash .specs/prd-<feature-slug>/tasks.md`
   - O comando atualiza `<!-- spec-hash-prd: ... -->` e `<!-- spec-hash-techspec: ... -->` usando SHA-256 implementado em Go, sem depender de `sha256sum`, e remove comentários obsoletos/duplicados.
   Estes hashes permitem detectar drift posterior via `ai-spec check-spec-drift .specs/prd-<feature-slug>/tasks.md` ou `ai-spec check-spec-drift .specs/prd-<feature-slug>`.

**Etapa 4.1: Preencher skills processuais necessárias (descoberta agnóstica, mandatória)**

Os templates (`assets/tasks-template.md` e `assets/task-template.md`) já contêm os placeholders mandatórios:
- `tasks-template.md` tem a coluna `Skills` na tabela.
- `task-template.md` tem a seção `## Skills Necessárias` entre `## Critérios de Sucesso` e `## Testes da Tarefa`.

Sua tarefa nesta etapa é **preencher** esses placeholders com detecção agnóstica em runtime — não inventar campos novos nem omitir os existentes.

1. Listar o diretório `.agents/skills/` para enumerar todas as skills disponíveis no projeto. Classificar cada skill **lendo o campo `category` do frontmatter** de `.agents/skills/<skill>/SKILL.md` (fonte de verdade — não usar lista hardcoded em prosa):
   - `category: governance` → auto-carregada (orquestração/governança). **Ignorar** na declaração por tarefa.
   - `category: language` → auto-carregada por detecção de diff em `execute-task` Stage 2. **Ignorar** na declaração por tarefa.
   - `category: processual` **ou campo ausente** → candidata à seção `## Skills Necessárias` se a `description` casar semanticamente com o objetivo da tarefa.
   - **Não usar glob nem nomes hardcoded** (ex.: `*-implementation`) para classificar — a classificação deriva exclusivamente do metadado `category`. Skills futuras de linguagem/governança devem declarar `category` no frontmatter para serem auto-detectadas como auto-carregadas; sem o campo, são tratadas como `processual` (declaráveis), que é o fallback seguro.
2. Para cada skill restante, ler `description` no frontmatter de `.agents/skills/<skill>/SKILL.md`.
3. Para cada tarefa proposta, comparar semanticamente o objetivo/critérios de aceitação com as descrições das skills disponíveis. Identificar skills cujo gatilho seja claramente acionado pela tarefa.
4. **Preenchimento mandatório dos placeholders (formato estrito, F28):**
   - Em `tasks.md` coluna `Skills`: lista separada por vírgula dos nomes de skill detectados; ou `—` se nenhuma extra for necessária. A coluna **deve estar preenchida em todas as linhas**, nunca em branco. Cada nome deve casar regex `^[a-z0-9-]+$` (lowercase, dígitos, hífen).
   - Em cada `task-X.Y-*.md` seção `## Skills Necessárias`: uma linha por skill com formato canônico estrito:
     - Formato canônico preferencial por linha: `^- \`([a-z0-9-]+)\` — .+$` (hífen, espaço, backtick, nome em backticks, espaço, em-dash, espaço, justificativa não-vazia).
     - Variantes comuns podem ser normalizadas pelo runtime para evitar falso positivo (`:` ou ` - ` entre skill e justificativa, espaços extras), mas o arquivo final deve sair no formato canônico para manter legibilidade e diff estável.
     - Se nenhuma skill processual extra for necessária, substituir os bullets de exemplo pelo conteúdo canônico: `Nenhuma além das auto-carregadas (governance + linguagem).` O runtime aceita variações vazias equivalentes (`Nenhuma.`, `N/A`) como vazio, mas deve reportar warning e preservar o canônico em novos artefatos.
     - A seção **deve estar presente em todos os task files**.
   - Manter os comentários HTML (`<!-- ... -->`) dos placeholders no arquivo final como guard rails contra futuras alucinações de re-geração.
   - `execute-task` Stage 2 item 5 aplica normalização antes da validação; entrada semanticamente ambígua ou skill inexistente continua bloqueante.
5. **Regras agnósticas obrigatórias:**
   - Nunca inventar skill que não exista em `.agents/skills/`. Validar presença lendo o diretório antes de listar.
   - Nunca hardcodar mapeamento; toda detecção parte da `description` do frontmatter da skill descoberta em runtime.
   - Quando o conjunto de skills disponíveis mudar (skills adicionadas/removidas), refazer a detecção. Não cachear suposições.
   - Quando em dúvida, **não declarar** — falso positivo custa contexto do subagent; falso negativo gera warning recuperável quando `execute-task` rodar.

**Etapa 5: Marcar dependências e paralelismo com clareza**
1. Usar apenas estados canônicos para `Status`: `pending`, `in_progress`, `needs_input`, `blocked`, `failed`, `done`.
2. Marcar dependências críticas explicitamente. Formato canônico da coluna `Dependências`:
   - `—` (em-dash unicode) quando nenhuma. NÃO usar hífen comum `-` nem `none`/`N/A`/vazio.
   - Lista separada por vírgula e espaço para múltiplas dependências internas (mesmo PRD): `1.0, 2.0` (decimal id, sempre).
   - **Dependência cross-PRD** (opcional): use prefixo `<outro-slug>/` antes do id. Exemplo: `1.0, foundations/3.0, observability/2.0`. O orquestrador (`execute-all-tasks`) interpreta o prefixo como referência ao PRD em `.specs/prd-<outro-slug>/tasks.md` e exige que aquela tarefa esteja `done` antes de tornar a atual `ready`.
   - Regex aceito: `^(—|(\w[\w-]*\/)?\d+\.\d+(,\s*(\w[\w-]*\/)?\d+\.\d+)*)$`. Valor fora do regex → `failed: malformed dependencies on task <id>`.
3. Identificar paralelismo seguro apenas quando ele não esconder risco de integração. Formato canônico OBRIGATÓRIO da coluna `Paralelizável` (case-sensitive, com til e maiúscula em `Não`):
   - `—`: tarefa sem par paralelo (default ou primeira da fase).
   - `Não`: tarefa explicitamente sequencial — não paralelizar mesmo se deps permitirem. **Atenção**: deve ser exatamente `Não` com N maiúsculo e til em ã. Não usar `não` (minúsculo), `nao` (sem til), `NÃO` (todo maiúsculo), `No`, `false`, `n`.
   - `Com <id>` ou `Com <id>, <id>, ...`: paralelizável especificamente com as tarefas listadas. **Atenção**: deve ser exatamente `Com ` com C maiúsculo. Não usar `com`, `COM`, `paralelo com`, `&`.
   - **NÃO usar**: `Sim`, `yes`, `parallel`, `Possivelmente`, `talvez`, `pode`, abreviações, ou variações em outras línguas.
   - Regex canônico aplicado após normalização por `execute-all-tasks`: `^(—|Não|Com\s+\d+\.\d+(,\s*\d+\.\d+)*)$`. Valores equivalentes (`não`, `nao`, `NÃO`, `com 2.0`, espaços extras) são normalizados com warning para evitar falso positivo. Valores semanticamente ambíguos (`Sim`, `talvez`, `pode`) continuam bloqueantes.
   - Exemplos válidos: `—`, `Não`, `Com 2.0`, `Com 1.0, 3.0, 5.0`. Exemplos inválidos: `Não.` (ponto final), `Com 2.0,3.0` (sem espaço após vírgula), `com 2.0` (c minúsculo), `Sim` (palavra não-canônica).
4. Gerar bloco mermaid `graph TD` em `tasks.md` representando o grafo de dependencias entre tarefas. Formato: `T1["1.0 — Titulo"] --> T2["2.0 — Titulo"]` para cada dependencia.

**Etapa 5.5: Validar sincronia das declarações de skills (mandatório)**

Antes de reportar `done`, validar que **a coluna `Skills` em `tasks.md` e a seção `## Skills Necessárias` em cada `task-X.Y-*.md` estão sincronizadas**, item a item:

1. Para cada linha da tabela em `tasks.md`:
   - Extrair o conjunto `S_table` da coluna `Skills` (vazio se `—`; senão, split por `,` e trim).
   - Ler o `task-X.Y-*.md` correspondente.
   - Extrair o conjunto `S_file` da seção `## Skills Necessárias`: nomes em backticks, vazio se conteúdo for `Nenhuma além das auto-carregadas (governance + linguagem).`
   - Comparar como conjuntos (ordem irrelevante):
     - Se `S_table == S_file`: ok.
     - Se divergente: **parar com `failed: skills sync drift on task <id>`**, reportar `S_table` e `S_file` lado a lado. Não escrever `done`.
2. Esta validação fecha falso positivo onde usuário lê `tasks.md` e vê info diferente do que `execute-task` carregaria. Diferenças indicam alucinação de uma das fontes.

**Etapa 5.6: Validar cobertura e hashes de spec (mandatório)**

1. Rodar `ai-spec check-spec-drift .specs/prd-<feature-slug>` ou `ai-spec check-spec-drift .specs/prd-<feature-slug>/tasks.md` após `sync-spec-hash`.
2. Se o comando reportar IDs faltantes, voltar à tabela de `tasks.md` e aos arquivos `task-*.md` até que todos os `RF-nn`/`REQ-nn` de `prd.md` e `techspec.md` estejam cobertos.
3. Se o comando reportar hash ausente/divergente, rodar novamente `ai-spec sync-spec-hash .specs/prd-<feature-slug>/tasks.md` e repetir a validação.
4. Não reportar `done` enquanto `check-spec-drift` não retornar sem drift.

**Etapa 6: Reportar o resultado**
1. Listar os arquivos gerados.
2. Destacar dependências críticas e tarefas paralelizáveis.
3. Confirmar que Etapa 5.5 retornou ok.
4. Confirmar que Etapa 5.6 retornou ok, sem IDs faltantes e sem hashes `PENDING-RUN`/duplicados.
5. Retornar estado final `done` quando os arquivos forem gerados ou `needs_input` quando a aprovação ainda for necessária.

## Tratamento de Erros

* Se o PRD e a especificação técnica divergirem sobre o escopo, pausar e expor o conflito em vez de codificar os dois nas tarefas.
* Se uma tarefa proposta misturar preocupações não relacionadas, dividi-la antes de escrever os arquivos.
* Se o plano exceder 10 itens principais, consolidar ou reagrupar o trabalho até que cada tarefa represente uma fatia coerente de entrega, e não um micro-passo.

## Resolução de paths

Todo caminho `.specs/prd-<slug>/` referenciado neste documento resolve para `${AI_TASKS_ROOT:-.specs}/${AI_PRD_PREFIX:-prd-}<slug>/`. Defaults preservam o layout histórico. Customização via `.claude/config.yaml` ou `.agents/config.yaml` (chaves `tasks_root`, `prd_prefix`). `check-invocation-depth.sh` exporta `AI_TASKS_ROOT` e `AI_PRD_PREFIX` para garantir paridade entre Claude Code, Codex, Gemini e Copilot — resolução em cascata `.agents/lib/` → `scripts/lib/` (vendor canônico em `.agents/lib/`).
