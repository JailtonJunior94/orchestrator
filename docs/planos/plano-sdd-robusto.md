# Plano de evolução para um SDD robusto e verificável

## Diagnóstico executivo

**Nota geral atual: 5,0/10.**

O fluxo possui bons fundamentos — PRD-first, hashes, isolamento, revisão, evidências e testes — mas parte importante dessas garantias existe apenas como instrução textual. Há inconsistências entre skills, hooks e validadores que permitem falso `done`, evidência autodeclarada, drift silencioso e falhas de concorrência.

| Componente | Nota | Diagnóstico principal |
|---|---:|---|
| `create-prd` | 6,5 | Bom contrato funcional, mas aprovação, NFRs mensuráveis e invalidação downstream são frágeis. |
| `create-technical-specification` | 6,0 | Boa intenção arquitetural; falta matriz RF→decisão→teste e maior neutralidade tecnológica. |
| `create-tasks` | 6,0 | Boa decomposição, mas há contradições no carregamento de skills e DAG invertido no template. |
| `execute-all-tasks` | 3,0 | Orquestração depende excessivamente de obediência do modelo; contratos de checkpoint, profundidade e concorrência são inconsistentes. |
| `execute-task` | 4,0 | Tem gates relevantes, porém evidência, revisão independente, DiffSHA e atualização concorrente não são confiáveis. |
| `review` | 5,0 | Boa taxonomia, mas revisão no mesmo contexto, roteamento por linguagem e diff de worktree são insuficientes. |
| `bugfix` | 5,0 | Exige causa raiz e regressão, mas não garante fail-before/pass-after nem revisão independente por defeito. |

A meta “zero alucinação/zero regressão” será tratada de forma operacional: **zero escapes conhecidos nas suítes determinísticas e adversariais, execução fail-closed e estado `needs_input` quando não houver evidência suficiente**. Nenhum processo baseado em modelos deve prometer risco matematicamente zero.

A direção segue as recomendações oficiais: prompts enxutos, contratos explícitos, ferramentas restritas e avaliação contínua da OpenAI ([model guidance](https://developers.openai.com/api/docs/guides/latest-model), [eval best practices](https://developers.openai.com/api/docs/guides/evaluation-best-practices), [agent evals](https://developers.openai.com/api/docs/guides/agent-evals)); além de simplicidade, isolamento, avaliação independente e hooks determinísticos da Anthropic ([effective agents](https://www.anthropic.com/engineering/building-effective-agents), [agent evals](https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents), [long-running agents](https://www.anthropic.com/engineering/harness-design-long-running-apps), [Claude hooks](https://code.claude.com/docs/en/hooks-guide), [subagents](https://code.claude.com/docs/en/sub-agents)).

## Plano de implementação

### Fase 0 — Corrigir violações críticas

- Incluir `tests/scripts/validate-task-evidence_test.sh` no gate oficial `make test-validators` e no CI. Hoje ele falha em 4 de 6 cenários, embora o gate oficial passe.
- Tornar obrigatórios e semanticamente validados:
  - digest imutável do patch;
  - `verdict` da revisão;
  - ferramenta/revisor permitido;
  - ausência de regressão de cobertura;
  - evidência individual para cada critério de aceite.
- Corrigir containment de caminhos no [post-execute-task.sh](/Users/jailtonjunior/Git/orchestrator/.agents/hooks/post-execute-task.sh:140): usar `realpath`, rejeitar `..`, caminhos absolutos, symlinks externos e qualquer destino fora do repositório.
- Unificar checkpoint, resultado do agente e estado da tarefa em um único schema versionado. Eliminar a contradição em que o checkpoint contém quatro campos, o hook aceita três e depois exige um arquivo que o fluxo manda remover.
- Corrigir:
  - direção do DAG no [tasks-template.md](/Users/jailtonjunior/Git/orchestrator/.agents/skills/create-tasks/assets/tasks-template.md:49);
  - classificação hardcoded de skills no [task-template.md](/Users/jailtonjunior/Git/orchestrator/.agents/skills/create-tasks/assets/task-template.md:29);
  - profundidade cross-PRD maior que três;
  - vocabulário de status e semântica de `partial`;
  - divergência entre F35 opt-in e execução default-on.
- Substituir documentação desatualizada sobre Codex: agentes customizados em `.codex/agents/*.toml` são atualmente nativos ([documentação de subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents)).
- Remover o encadeamento impossível `execute-all → execute-task → review → bugfix` sob o limite atual de profundidade. `review` não poderá mais ignorar o guard com `|| true`.

### Fase 1 — Transformar governança textual em contratos executáveis

- Criar `.specs/prd-<slug>/sdd-state.json`, gerado exclusivamente pelo CLI, contendo:
  - `schema_version`, `run_id`;
  - hashes aprovados de PRD, techspec e tasks;
  - estados `draft`, `approved`, `stale`, `executing`, `blocked`, `done`;
  - RFs/NFRs, DAG, aprovações e referências de evidência.
- Introduzir:
  - `ai-spec validate <prd-dir>`;
  - `ai-spec approve <artifact>`;
  - `ai-spec invalidate <prd-dir> --from <artifact>`;
  - `ai-spec orchestrate <prd-dir>`.
- Alterar `sync-spec-hash`: ele não poderá atualizar hashes downstream aprovados após mudança upstream. Mudança no PRD marca techspec e tasks como `stale`; mudança na techspec invalida tasks.
- Centralizar parsing, validação e transições de estado em Go. Hooks shell passam a ser adaptadores finos, sem `grep` como fonte de verdade.
- Usar resultados JSON validados por schema entre agentes. Conteúdo inválido, incompleto ou não comprovado resulta em `blocked`/`needs_input`, nunca em inferência otimista.

### Fase 2 — Fortalecer PRD, techspec e tasks

- PRD:
  - adicionar responsável, aprovador, estado, revisão e histórico;
  - separar RFs de NFRs numerados e mensuráveis;
  - registrar métricas de sucesso, fora de escopo, premissas e perguntas abertas;
  - invalidar automaticamente artefatos downstream após alteração material.
- Techspec:
  - adicionar matriz obrigatória `RF/NFR → decisão técnica → tarefa → teste`;
  - documentar baseline, compatibilidade, migração, rollback, segurança, observabilidade e falhas;
  - criar ADR apenas para decisões transversais, irreversíveis ou de alto custo;
  - substituir exemplos exclusivamente Go por contratos neutros e variantes carregadas sob demanda.
- Tasks:
  - validar cobertura por vínculo estrutural, substituindo a busca textual atual em [specdrift.go](/Users/jailtonjunior/Git/orchestrator/internal/specdrift/specdrift.go:26);
  - exigir IDs de requisitos, testes e critérios de aceite por tarefa;
  - validar DAG, ciclos, dependências e ownership de arquivos;
  - derivar skills apenas do `category` do frontmatter;
  - usar `parallel_group` explícito, calculado a partir de dependências e ownership disjunto.

### Fase 3 — Orquestração segura e portável

- Implementar `ai-spec orchestrate` como máquina de estados idempotente:
  - eventos append-only;
  - escrita atômica;
  - tentativas identificadas por `run_id/task_id/attempt`;
  - retomada após crash sem falso `done`;
  - um único escritor do estado global.
- Executar tarefas que escrevem em worktrees isoladas. Permitir paralelismo apenas quando dependências e ownership forem disjuntos; conflitos potenciais tornam a execução sequencial.
- Detectar capacidades dos runtimes em vez de manter tabelas de versão dentro dos prompts:
  - Codex: agentes nativos;
  - Claude: subagent isolado, sem tentar spawn recursivo, pois subagents Claude não criam outros subagents;
  - fallback inline somente para operações read-only. Escrita que exige isolamento deve falhar fechada.
- Aplicar timeout, cancelamento, retry e orçamento no processo/orquestrador, não por promessa textual do agente.
- Substituir `DiffSHA` baseado em `HEAD` por:
  - SHA base;
  - digest SHA-256 do patch completo;
  - digest do estado final relevante;
  - opcionalmente commit SHA, apenas quando a criação de commit estiver autorizada.

### Fase 4 — Revisão e bugfix independentes

- Executar revisão em contexto fresco, read-only e separado do executor. O revisor recebe PRD, techspec, tarefa, patch completo e resultados de testes, mas não o raciocínio do implementador.
- Roteamento multi-language por arquivos realmente alterados; eliminar fallback arbitrário para Go.
- Revisar o patch cumulativo incluindo mudanças não commitadas, staged e novos arquivos.
- Tornar o veredito determinístico: qualquer achado crítico/alto bloqueia; ambiguidades viram `needs_input`, sem pergunta interativa durante automação.
- Bugfix:
  - aceitar entrada natural e normalizá-la internamente;
  - exigir origem vinculada a RF, contrato ou issue;
  - persistir prova fail-before e pass-after do teste de regressão;
  - executar revisão fresca depois da correção;
  - gerar relatório por `run_id`, evitando sobrescrita.
- Exceção PRD-first: defeito que apenas restaura comportamento já especificado não cria novo PRD, mas deve referenciar o RF existente. Comportamento novo ou alteração pública exige atualização e nova aprovação do PRD.

### Fase 5 — Redução de prompts e avaliação contínua

- Reduzir as skills a objetivo, entradas, saídas, stop conditions e ferramentas permitidas. Mover schemas, transições e validações para o CLI.
- Remover repetições, tabelas de capacidades estáticas e regras já asseguradas deterministicamente.
- Criar corpus inicial de 20–50 falhas reais/adversariais e separar:
  - capability evals;
  - regression evals;
  - testes determinísticos;
  - avaliações de trajetória e resultado;
  - calibração humana periódica.
- Executar múltiplas tentativas nos evals com modelos, ferramentas e CLIs suportados; registrar qualidade, escapes, tokens, latência e custo.
- Manter AGENTS/CLAUDE concisos: instruções são contexto, enquanto invariantes obrigatórios devem residir em hooks, schemas e executáveis ([Claude memory](https://code.claude.com/docs/en/memory), [Codex AGENTS.md](https://learn.chatgpt.com/docs/agent-configuration/agents-md)).

## Interfaces públicas e compatibilidade

- Novos comandos: `validate`, `approve`, `invalidate` e `orchestrate`.
- Novo contrato versionado `sdd-state.json`; Markdown continua sendo a interface humana.
- Novo resultado JSON de execução/revisão com `schema_version`, identidade da execução, patch digest, testes, critérios, veredito e evidências.
- Compatibilidade:
  - leitores aceitam o formato Markdown atual durante duas versões menores;
  - formato legado opera em modo warning-only, sem execução automática confiável;
  - novos projetos usam schema v2 estrito;
  - migração não altera arquivos sem dry-run e confirmação explícita.

## Testes e critérios de aceite

- Tornar verdes todos os testes existentes, incluindo os quatro casos atualmente aceitos incorretamente pelo validador.
- Adicionar testes table-driven, property/fuzz e integração para:
  - path traversal e symlink escape;
  - evidência falsa, ausente, duplicada ou reaproveitada;
  - hashes stale e aprovação indevida;
  - DAG invertido, ciclos e dependências cross-PRD;
  - crash em cada transição de checkpoint;
  - concorrência e conflito de arquivos;
  - dirty worktree e diff não commitado;
  - projetos Go, Node, Python, .NET e multi-language;
  - runtime sem suporte a subagent;
  - revisão e bugfix cumulativos.
- Matriz CI em Ubuntu, macOS e Windows, com smoke tests dos adaptadores Codex, Claude, Gemini e Copilot.
- Critérios de saída:
  - 100% de rastreabilidade estrutural RF/NFR→decisão→task→teste→evidência;
  - zero path escape, lost update e falso `done` no corpus adversarial;
  - nenhum artefato `done` com hash stale, teste não comprovado ou revisão bloqueante;
  - todos os gates standalone incorporados ao CI;
  - nenhuma regressão superior a 10% em custo/latência sem ganho de qualidade aprovado;
  - revisão independente sem achados críticos/altos antes da adoção do modo estrito.

## Premissas

- A prioridade é confiabilidade sobre velocidade; paralelismo só será habilitado quando comprovadamente seguro.
- Markdown permanece legível e versionável, mas deixa de ser a única fonte de estado operacional.
- As mudanças serão entregues incrementalmente: primeiro fechar escapes críticos, depois introduzir schema/orquestrador e, por último, simplificar prompts.
- A análise que originou este plano foi read-only: nenhuma alteração havia sido feita no workspace durante a auditoria.
