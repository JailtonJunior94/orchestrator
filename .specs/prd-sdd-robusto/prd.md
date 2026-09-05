# PRD — SDD robusto e verificável

<!-- spec-version: 1 -->

## Visão Geral

O ai-spec-harness precisa deixar de depender de instruções textuais para garantir a
confiabilidade do fluxo SDD. Este produto cria contratos versionados e executáveis para
requisitos, aprovação, execução, revisão e evidência, permitindo que automações falhem
fechadas quando não houver prova suficiente.

## Responsáveis e Estado

- Responsável: equipe mantenedora do ai-spec-harness.
- Aprovador: solicitante desta entrega.
- Estado: aprovado para implementação.
- Revisão: 1.
- Histórico: criado a partir de `docs/planos/plano-sdd-robusto.md` em 2026-08-31.

## Objetivos e Métricas de Sucesso

- Impedir `done` sem estado, hash, testes, revisão e evidência válidos.
- Manter 100% de rastreabilidade estrutural RF/NFR → decisão → tarefa → teste → evidência.
- Fazer todos os validadores standalone parte dos gates oficiais e da CI.
- Bloquear 100% dos escapes de caminho, evidência falsa e estado stale no corpus adversarial.
- Não aumentar custo ou latência em mais de 10% sem ganho de qualidade explicitamente aceito.

## Histórias de Usuário

- Como mantenedor, quero aprovar e invalidar artefatos de requisitos para que execução nunca use
  especificação desatualizada.
- Como executor de tarefas, quero um estado recuperável e validado para que uma queda não marque
  trabalho incompleto como concluído.
- Como revisor, quero receber um patch cumulativo e contexto independente para que achados
  bloqueantes não sejam omitidos.
- Como operador de CI, quero gates determinísticos para que o mesmo contrato seja aplicado em
  todas as ferramentas e plataformas suportadas.

## Requisitos Funcionais

- RF-01: `make test-validators` e a CI devem executar a suíte adversarial completa de evidência.
- RF-02: Resultados de execução e revisão devem ser JSON versionado, estritamente validado e
  conter identidade, patch digest, testes, critérios, evidências e veredito.
- RF-03: Nenhuma tarefa pode ficar `done` sem digest imutável do patch, teste comprovado,
  cobertura não regressiva, evidência por critério e revisão aprovada.
- RF-04: Hooks devem rejeitar path traversal, paths absolutos e symlinks que escapem do repositório.
- RF-05: Checkpoint, resultado do agente e estado da tarefa devem usar o mesmo schema e vocabulário.
- RF-06: O CLI deve persistir `sdd-state.json` versionado com hashes, aprovações, requisitos, DAG,
  estados e referências de evidência.
- RF-07: O CLI deve oferecer `validate`, `approve`, `invalidate` e `orchestrate`.
- RF-08: Uma alteração upstream deve invalidar downstream aprovado; sincronização de hash não pode
  ocultar esse drift.
- RF-09: A cobertura de requisitos deve usar vínculos estruturais, não busca textual solta.
- RF-10: Tasks devem validar DAG, ciclos, dependências cross-PRD, ownership e paralelismo seguro.
- RF-11: A orquestração deve ser idempotente, append-only, atômica, com um escritor e retomada
  por `run_id/task_id/attempt`.
- RF-12: Escritas concorrentes devem ocorrer somente em worktrees isoladas e ownership disjunto;
  caso contrário a execução deve ser sequencial.
- RF-13: Capacidades de runtime devem ser detectadas; escrita que exige isolamento sem suporte
  deve falhar fechada.
- RF-14: O patch deve ser identificado por SHA base, SHA-256 do patch completo e digest do estado
  final. A evidência de uma tarefa `done` deve ser selada com o commit SHA que a contém e o digest
  do patch recomputado no range base..commit; sem o selo, a prova vale apenas no instante do
  fechamento e não é re-auditável.
- RF-15: Revisão deve ser fresca, read-only, multi-language e usar diff cumulativo inclusive
  arquivos staged, unstaged e não rastreados.
- RF-16: Bugfix deve persistir origem, fail-before, pass-after e revisão fresca por tentativa.
- RF-17: Skills e documentação devem delegar invariantes ao CLI e não conter tabelas estáticas
  desatualizáveis de capacidades.
- RF-18: Deve existir corpus versionado de evals adversariais, com métricas de qualidade, escapes,
  tokens, latência e custo.

## Requisitos Não Funcionais

- NFR-01: novas validações são fail-closed por padrão. O legado ficou warning-only durante duas
  versões menores a partir de `0.29.0` — janela cumprida em `0.29` e `0.30` — e desde `0.31.0` o
  escape exige opt-out explícito e ruidoso, que não habilita automação confiável.
- NFR-02: toda escrita de estado deve ser atômica e preservar eventos append-only.
- NFR-03: o código mantém Go declarado em `go.mod`, DI explícita, erros contextualizados em PT-BR
  e testes determinísticos.
- NFR-04: CI cobre Ubuntu, macOS e Windows; adaptadores suportados possuem smoke test sem depender
  de credenciais externas.
- NFR-05: Markdown continua como interface humana legível e versionável.

## Fora de Escopo

- Prometer risco matematicamente zero ou validar respostas de modelo por confiança.
- Criar commits, publicar releases ou chamar runtimes remotos reais durante testes.
- Migrar automaticamente projetos legados sem `--dry-run` e confirmação explícita.

## Premissas e Perguntas Abertas

- A autorização explícita para implementar todo o plano também aprova esta decomposição.
- Runtimes externos podem estar indisponíveis em CI; seus adaptadores serão testados por contrato.
- O corpus inicial usará fixtures determinísticas e não fará inferências sobre custo de modelos reais.
