---
name: create-tasks
version: 1.5.0
description: Cria tarefas incrementais de implementação a partir de um PRD e de uma especificação técnica. Use quando documentos de produto e técnicos aprovados precisarem ser decompostos em itens de trabalho ordenados e testáveis, incluindo declaração de skills processuais especializadas necessárias por tarefa. Não use para mudanças diretas de código, descoberta de funcionalidade ou revisão de branch.
---

# Criar Tarefas

## Procedimentos

**Etapa 1: Validar os documentos de origem**
1. Confirmar que o contrato de carga base definido em `AGENTS.md` foi cumprido.
2. Confirmar que `tasks/prd-<feature-slug>/prd.md` e `tasks/prd-<feature-slug>/techspec.md` existem.
2. Ler os dois arquivos por completo antes de propor itens de trabalho.
3. Parar com `needs_input` se qualquer documento estiver ausente ou contraditório o suficiente para bloquear o planejamento.

**Etapa 2: Extrair fatias de entrega**
1. Identificar requisitos, decisões técnicas, pontos de integração, dependências e áreas de risco.
2. Agrupar o trabalho em fatias que entreguem valor verificável.
3. Preferir a sequência `domain -> interfaces/ports -> use cases -> adapters/repositories -> handlers -> integration`, salvo quando a especificação técnica justificar outra ordem.

**Etapa 3: Propor primeiro o plano de tarefas em alto nível**
1. Ler `assets/tasks-template.md` e `assets/task-template.md` antes de redigir.
2. Produzir uma lista de alto nível com no máximo `${AI_MAX_TASKS_PER_PRD:-10}` tarefas (configurável via env ou `.claude/config.yaml` chave `max_tasks_per_prd`). Default conservador é 10 para forçar consolidação de PRDs grandes em fatias coerentes; PRDs com escopo justificadamente maior podem aumentar — documentar a justificativa em `## Riscos de Integração` da `tasks.md`.
3. Para cada tarefa, incluir objetivo, entregável e dependências.
4. Parar e aguardar aprovação antes de gerar os arquivos finais.
5. Se a aprovação não estiver disponível na sessão atual, retornar `needs_input` e não escrever os arquivos de tarefa.

**Etapa 4: Gerar os artefatos detalhados de tarefa**
1. Após a aprovação, criar `tasks/prd-<feature-slug>/tasks.md` a partir de `assets/tasks-template.md`.
2. Criar um arquivo por tarefa usando `assets/task-template.md`.
3. Dar a cada tarefa critérios de aceitação explícitos, arquivos relevantes e expectativas de teste.
4. Garantir que cada tarefa seja executável de forma independente e revisável objetivamente.
5. Ao escrever `tasks.md`, calcular e inserir os comentários de rastreabilidade de spec no cabeçalho:
   - `<!-- spec-hash-prd: $(sha256sum tasks/prd-<feature-slug>/prd.md | awk '{print $1}') -->`
   - `<!-- spec-hash-techspec: $(sha256sum tasks/prd-<feature-slug>/techspec.md | awk '{print $1}') -->`
   Estes hashes permitem detectar drift posterior via `bash scripts/check-spec-drift.sh`.

**Etapa 4.1: Preencher skills processuais necessárias (descoberta agnóstica, mandatória)**

Os templates (`assets/tasks-template.md` e `assets/task-template.md`) já contêm os placeholders mandatórios:
- `tasks-template.md` tem a coluna `Skills` na tabela.
- `task-template.md` tem a seção `## Skills Necessárias` entre `## Critérios de Sucesso` e `## Testes da Tarefa`.

Sua tarefa nesta etapa é **preencher** esses placeholders com detecção agnóstica em runtime — não inventar campos novos nem omitir os existentes.

1. Listar o diretório `.agents/skills/` para enumerar todas as skills disponíveis no projeto. Ignorar as auto-carregadas em runtime — lista **exata e explícita**:
   - Governance/orquestração: `agent-governance`, `execute-task`, `execute-all-tasks`, `bugfix`, `review`, `refactor`.
   - Linguagem (detectadas pelo diff em `execute-task` Stage 2): `go-implementation`, `node-implementation`, `python-implementation`, `object-calisthenics-go`.
   - **Não usar glob `*-implementation`** — overmatch pode ignorar skills futuras com esse sufixo que não sejam de linguagem (ex.: `react-implementation` se um dia for criada como skill de framework, e não de linguagem). A enumeração explícita acima é a fonte de verdade; atualizar quando uma nova skill de linguagem for adicionada ao projeto.
   - Skills não-listadas acima são candidatas à seção `## Skills Necessárias` se a description casar semanticamente com o objetivo da tarefa.
2. Para cada skill restante, ler `description` no frontmatter de `.agents/skills/<skill>/SKILL.md`.
3. Para cada tarefa proposta, comparar semanticamente o objetivo/critérios de aceitação com as descrições das skills disponíveis. Identificar skills cujo gatilho seja claramente acionado pela tarefa.
4. **Preenchimento mandatório dos placeholders (formato estrito, F28):**
   - Em `tasks.md` coluna `Skills`: lista separada por vírgula dos nomes de skill detectados; ou `—` se nenhuma extra for necessária. A coluna **deve estar preenchida em todas as linhas**, nunca em branco. Cada nome deve casar regex `^[a-z0-9-]+$` (lowercase, dígitos, hífen).
   - Em cada `task-X.Y-*.md` seção `## Skills Necessárias`: uma linha por skill com formato canônico estrito:
     - Regex obrigatório por linha: `^- \`([a-z0-9-]+)\` — .+$` (hífen, espaço, backtick, nome em backticks, espaço, em-dash, espaço, justificativa não-vazia).
     - Variantes proibidas: `* skill-name` (asterisco), `- skill-name` (sem backticks), `- "skill-name"` (aspas em vez de backticks), `- \`skill-name\`: justificativa` (dois-pontos em vez de em-dash), `- \`skill-name\` - justificativa` (hífen em vez de em-dash).
     - Se nenhuma skill processual extra for necessária, substituir os bullets de exemplo pelo conteúdo único exato: `Nenhuma além das auto-carregadas (governance + linguagem).` (sem variações como "Nenhuma." ou "N/A").
     - A seção **deve estar presente em todos os task files**.
   - Manter os comentários HTML (`<!-- ... -->`) dos placeholders no arquivo final como guard rails contra futuras alucinações de re-geração.
   - `execute-task` Stage 2 item 5 aplica os mesmos regex; entrada malformada → `failed: malformed Skills Necessárias entry: <linha>`. Não há tolerância — formato canônico é o contrato.
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
   - **Dependência cross-PRD** (opcional): use prefixo `<outro-slug>/` antes do id. Exemplo: `1.0, foundations/3.0, observability/2.0`. O orquestrador (`execute-all-tasks`) interpreta o prefixo como referência ao PRD em `tasks/prd-<outro-slug>/tasks.md` e exige que aquela tarefa esteja `done` antes de tornar a atual `ready`.
   - Regex aceito: `^(—|(\w[\w-]*\/)?\d+\.\d+(,\s*(\w[\w-]*\/)?\d+\.\d+)*)$`. Valor fora do regex → `failed: malformed dependencies on task <id>`.
3. Identificar paralelismo seguro apenas quando ele não esconder risco de integração. Formato canônico OBRIGATÓRIO da coluna `Paralelizável` (case-sensitive, com til e maiúscula em `Não`):
   - `—`: tarefa sem par paralelo (default ou primeira da fase).
   - `Não`: tarefa explicitamente sequencial — não paralelizar mesmo se deps permitirem. **Atenção**: deve ser exatamente `Não` com N maiúsculo e til em ã. Não usar `não` (minúsculo), `nao` (sem til), `NÃO` (todo maiúsculo), `No`, `false`, `n`.
   - `Com <id>` ou `Com <id>, <id>, ...`: paralelizável especificamente com as tarefas listadas. **Atenção**: deve ser exatamente `Com ` com C maiúsculo. Não usar `com`, `COM`, `paralelo com`, `&`.
   - **NÃO usar**: `Sim`, `yes`, `parallel`, `Possivelmente`, `talvez`, `pode`, abreviações, ou variações em outras línguas.
   - Regex case-sensitive aplicado por `execute-all-tasks`: `^(—|Não|Com\s+\d+\.\d+(,\s*\d+\.\d+)*)$`. Valores fora → `failed: malformed Paralelizável on <id>`. **Não há tolerância case-insensitive** — divergência indica geração inconsistente que precisa ser corrigida na origem, não mascarada pelo parser.
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

**Etapa 6: Reportar o resultado**
1. Listar os arquivos gerados.
2. Destacar dependências críticas e tarefas paralelizáveis.
3. Confirmar que Etapa 5.5 retornou ok.
4. Retornar estado final `done` quando os arquivos forem gerados ou `needs_input` quando a aprovação ainda for necessária.

## Tratamento de Erros

* Se o PRD e a especificação técnica divergirem sobre o escopo, pausar e expor o conflito em vez de codificar os dois nas tarefas.
* Se uma tarefa proposta misturar preocupações não relacionadas, dividi-la antes de escrever os arquivos.
* Se o plano exceder 10 itens principais, consolidar ou reagrupar o trabalho até que cada tarefa represente uma fatia coerente de entrega, e não um micro-passo.

## Resolução de paths

Todo caminho `tasks/prd-<slug>/` referenciado neste documento resolve para `${AI_TASKS_ROOT:-tasks}/${AI_PRD_PREFIX:-prd-}<slug>/`. Defaults preservam o layout histórico. Customização via `.claude/config.yaml` ou `.agents/config.yaml` (chaves `tasks_root`, `prd_prefix`). `scripts/lib/check-invocation-depth.sh` exporta `AI_TASKS_ROOT` e `AI_PRD_PREFIX` para garantir paridade entre Claude Code, Codex, Gemini e Copilot.
