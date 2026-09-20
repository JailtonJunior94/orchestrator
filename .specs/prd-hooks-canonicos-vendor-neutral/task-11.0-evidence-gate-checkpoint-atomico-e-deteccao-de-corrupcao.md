# Tarefa 11.0: Evidence gate, checkpoint atomico, deteccao de corrupcao e sanitizacao

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Tornar atômicos, idempotentes, detectáveis quando corrompidos e sanitizados os artefatos
operacionais de checkpoint e evidência. Cobre RF-35 a RF-44 e RF-49.

O racional completo, o levantamento linha a linha e a ordem de migração estão em
[ADR-006](adr-006-atomicidade-artefatos-operacionais.md) — **referenciar, não duplicar**. A
semântica do resultado tipado que o evidence gate devolve vem de
[ADR-002](adr-002-resultado-tipado-traducao-exit-code.md). Sequenciamento em `techspec.md:510-517`
(Fase 7).

Estado verificado do repositório, que define o tamanho real do problema:

- A primitiva **existe e está correta**: `internal/fs/fs.go:173` cria o temporário `.tmp-*` no
  diretório de destino (`:181`), escreve (`:189`), faz `Sync()` com erro verificado (`:193`),
  `Close()` com erro verificado (`:197`) e `os.Rename` (`:200`). Está declarada na interface em
  `internal/fs/fs.go:27` e implementada no dublê em `internal/fs/fake.go:145`.
- É usada por apenas **quatro pacotes de produção**: `internal/txn/transaction.go:254,289,442`,
  `internal/runtime/memory/durable/{layer.go:453,handoff_lease.go:147}`,
  `internal/tracking/tracker.go:204-205` e `cmd/ai_spec_harness/memory.go:237,369,394`.
- O artefato que sustenta **toda** a evidência é o menos protegido: `internal/runtime/persistence/jsonl.go`
  declara-se "writer JSONL append-only" em `:15` mas **não faz append**. Mantém o arquivo inteiro em
  memória no campo `buf []byte` (`:21-23`), lê o arquivo completo no construtor (`:38`, carregado em
  `:40`) e a cada evento faz `append` em memória (`:57`) seguido de `WriteFile` do conteúdo
  **completo** (`:58`). Como `WriteFile` do `OSFileSystem` é `os.WriteFile` com `O_TRUNC`
  (`internal/fs/fs.go:170`), um crash não perde o último evento: perde a **sessão inteira**. O custo
  é O(n) por evento e O(n²) em bytes ao longo da run.
- O doc-comment de `internal/runtime/persistence/jsonl.go:46` — "Falha de escrita propaga erro sem
  corromper o conteúdo anterior" — é **falso** para o filesystem real.
- A justificativa declarada desse desenho está em `internal/runtime/persistence/jsonl.go:21-22`: o
  `FakeFileSystem` "não suporta append". É uma limitação do dublê de teste ditando o design de
  produção.

<requirements>
- RF-35: pacote de evidência validado deterministicamente antes da conclusão; ausente ou inválido
  impede conclusão aprovada.
- RF-36: integridade da evidência validada por fingerprint quando aplicável.
- RF-37: declaração textual do agente não substitui evidência.
- RF-38: evidência produzida sob um provedor é validável sob qualquer outro; o gate permanece
  independente de provedor e de LLM.
- RF-39: checkpoint persiste o estado mínimo de retomada, sem depender do histórico privado da
  sessão do provedor.
- RF-40: escrita de checkpoint atômica — escrita parcial não produz checkpoint válido.
- RF-41: checkpoints idempotentes e reexecutáveis sem efeito colateral.
- RF-42: estado corrompido é detectado e reportado explicitamente, nunca consumido como válido.
- RF-43: tarefa iniciada em uma CLI é retomável por outra quando o workflow permitir.
- RF-44: checkpoint não persiste segredo nem contexto sensível desnecessário.
- RF-49: chaves, tokens e segredos fora de telemetria, logs e evidência.
- R-STYLE-001 (hard): código em inglês, zero comentários, sem prefixo `_` em identificador Go.
- Zero regressão: nenhum gate ou teste existente muda de comportamento observável sem registro.
</requirements>

## Subtarefas

- [ ] 11.1 **Etapa zero, bloqueante.** Confirmar que a tarefa 3.0 já acrescentou `.tmp-*` às
      exclusões de `internal/taskloop/orchestrator.go:456-512` e à tolerância de
      `internal/taskloop/isolation.go:322`. Sem isso, o temporário criado por `WriteFileAtomic`
      entra no patch semântico, altera o `PatchSHA256` e dispara o fail-closed de
      `internal/taskloop/orchestrator.go:352-354`. Se 3.0 não estiver `done`, esta tarefa reporta
      `blocked`. Ordem obrigatória registrada em [ADR-006](adr-006-atomicidade-artefatos-operacionais.md)
      e em `techspec.md:455-462`.
- [ ] 11.2 Inventariar e registrar os **oito** call-sites que chamam `os.WriteFile` **direto**, fora
      da abstração `fs.FileSystem`: `internal/changelog/changelog.go:157`,
      `internal/runtime/approval_adapters.go:124`, `internal/runtime/approval_adapters.go:192`,
      `internal/runtime/memory/store.go:153`, `internal/specdrift/sync.go:60`,
      `internal/embedded/embedded.go:61`, `cmd/ai_spec_harness/seal_evidence.go:72` e
      `cmd/ai_spec_harness/update_version.go:36`. Classificar cada um em migrar / não migrar, com
      justificativa.
- [ ] 11.3 **Migração — etapa 1: artefatos terminais e write-once sem leitor concorrente.**
      `cmd/ai_spec_harness/seal_evidence.go:72` (result.json selado),
      `internal/runtime/persistence/report.go:69` (`execution_report.md`) e
      `internal/runtime/persistence/toolcalls.go:23` (`tool_calls.md`). Antes de cada migração,
      verificar hard link e symlink no destino.
- [ ] 11.4 **Migração — etapa 2: `tasks.md`.** `internal/taskloop/task_status_writer.go:48,91` e
      `internal/taskloop/reservations.go:138,228`. Unificar com o `flock` que
      `.agents/skills/execute-all-tasks/SKILL.md:124` já tenta por fora, em prosa, no prompt do
      subagente.
- [ ] 11.5 **Migração — etapa 3, por ÚLTIMO e como REDESENHO:** `internal/runtime/persistence/jsonl.go`.
      Append real, com **extensão da interface `fs.FileSystem`** (`internal/fs/fs.go:20-32`) e do
      `FakeFileSystem` (`internal/fs/fake.go`) para suportar append. **Não** é troca de chamada:
      trocar `WriteFile` por `WriteFileAtomic` na linha `:58` esconde o problema sem resolver o
      rewrite O(n) por evento nem a sobrescrita entre processos. Remover o campo `buf` (`:23`) e a
      leitura integral do construtor (`:38,40`). Corrigir o doc-comment falso de `:46` — e, sob
      R-STYLE-001, **remover** os comentários das linhas tocadas em vez de reescrevê-los.
- [ ] 11.6 **NÃO MIGRAR:** `internal/runtime/memory/store.go:153` no modo append. O `O_APPEND` do
      kernel já serializa corretamente; converter para read-modify-write pioraria a concorrência.
      Registrar a decisão explicitamente no `execution_report.md`.
- [ ] 11.7 Implementar detecção de corrupção (RF-42) em `jsonl.go`: linha inválida é reportada como
      erro tipado, nunca consumida como válida nem silenciosamente descartada.
- [ ] 11.8 Corrigir `.agents/hooks/post-wave.sh`. Três defeitos verificados: o header usa temporário
      de nome **fixo** `"$PARTIAL_MD.tmp"` seguido de `mv` (`:45`, `:54`), que colide sob
      concorrência; o append da wave usa `>>` em cinco syscalls dentro de um bloco `{ … } >>`
      (`:58-70`), portanto **não atômico**; e a operação **não é idempotente** — reexecutar a mesma
      wave duplica a seção. Além disso, **nenhum código Go lê o arquivo**: o rename final
      `.partial.md` → `_orchestration_report.md` é instruído em prosa ao LLM em
      `.agents/skills/execute-all-tasks/SKILL.md:126-129`.
- [ ] 11.9 Unificar a extensão do checkpoint, hoje divergente entre **quatro** consumidores:
      `.agents/skills/execute-task/SKILL.md:84` grava `.checkpoints/<num>.yaml.tmp`;
      `internal/sdd/state.go:556` lê `.checkpoints/<taskID>.json`;
      `.claude/hooks/subagent-stop-wrapper.sh:115` procura `.checkpoints/${task_id}.json`; e o gate
      F25 de `.agents/hooks/post-execute-task.sh:221-227` checa `.checkpoints/${TASK_ID}.yaml`.
      Escolher uma extensão, migrar os quatro no mesmo lote e provar por teste que nenhum caminho
      ficou órfão.
- [ ] 11.10 Substituir o lock **anônimo** do orquestrador. `internal/taskloop/orchestrator_lock_unix.go:13-23`
       abre o arquivo e faz `syscall.Flock` sem escrever **nada** — sem PID, sem hostname, sem
       timestamp. Em `internal/taskloop/orchestrator_lock_windows.go:15` usa `O_CREATE|O_EXCL` e o
       comentário de `:11-13` admite o lock órfão permanente ("uma retomada requer remover
       explicitamente um lock órfão após confirmação operacional"). O `ProcessRef` e a prova de
       liveness que resolvem isso **já existem**: `internal/runtime/memory/durable/liveness.go:8,14,34`
       e `internal/runtime/memory/durable/process_unix.go:11`; o lease está em
       `internal/runtime/memory/durable/handoff_lease.go:147`. Estão no pacote errado — promover
       para um pacote consumível pelo `taskloop` sem inverter a dependência.
- [ ] 11.11 Estender a sanitização (RF-49). Hoje existe em **um** lugar:
       `internal/runtime/memory/durable/sanitization_policy.go:64,68`, aplicada exclusivamente em
       `internal/runtime/memory/durable/facade.go:231`. **Não cobre** os cinco caminhos exigidos:
       `events.jsonl`, `execution_report.md`, `tool_calls.md`, o `MEMORY.md` legado e o
       `.partial.md` do post-wave — que faz `cat` **literal** do YAML de resultados
       (`.agents/hooks/post-wave.sh:64`).
- [ ] 11.12 Evidence gate (RF-35 a RF-38): validação determinística, independente de provedor e de
       LLM, com fingerprint de integridade. Preservar o selo existente —
       `internal/taskloop/orchestrator.go:126-130` calcula sha256 do patch `base..commit`, é
       write-once (`:205`) e falha duro na divergência (`:352-354`) —, corrigindo apenas o caminho
       de escrita não atômico em `cmd/ai_spec_harness/seal_evidence.go:72`.
- [ ] 11.13 Rodar os gates de não-regressão e anexar as saídas ao `execution_report.md`.

## Detalhes de Implementação

- Ordem de migração, justificativa de cada etapa e lista completa de call-sites:
  [ADR-006](adr-006-atomicidade-artefatos-operacionais.md). Não reproduzir o levantamento aqui.
- Fase 7 do sequenciamento: `techspec.md:510-517`.
- Riscos R-04, R-05 e R-06: `techspec.md:591-594`.
- Semântica do resultado do evidence gate:
  [ADR-002](adr-002-resultado-tipado-traducao-exit-code.md).
- Lacuna de sanitização registrada em `techspec.md:565-571`.

### Riscos de não-regressão a registrar e verificar antes de cada migração

1. `WriteFileAtomic` cria **inode novo**: perde dono, grupo, ACL e xattr; **quebra hard link**; e
   **destrói symlink de destino** por rename, onde hoje se escreve através dele.
2. Esta tarefa **não toca caminhos de instalação**. `internal/install/install.go` espelha hooks
   entre cinco diretórios (`.agents/`, `.claude/`, `.codex/`, `.github/`,
   `internal/embedded/assets/`); migrar esses caminhos é fora de escopo.
3. Verificar **hard link e symlink** no destino antes de cada migração individual, não em bloco.
4. `os.Rename` gera evento de filesystem `RENAME`, **não** `MODIFY`: observadores em tempo real
   perdem o handle. Verificar `internal/taskloop/agent_livewriter.go` antes da etapa 11.3.
5. `os.CreateTemp` no mesmo diretório (`internal/fs/fs.go:181`) evita `EXDEV` **apenas** se o
   diretório não for mount point nem overlay — cenário comum em container de CI. Registrar o
   comportamento esperado nessa condição.

## Critérios de Sucesso

- Nenhuma escrita de artefato terminal, de `tasks.md` ou de evento passa por `WriteFile` truncante
  nos caminhos migrados; teste que falha se um call-site migrado regredir.
- `jsonl.go` sem campo `buf` e sem leitura integral no construtor; teste de recuperação de crash
  (`jsonl_crash_recovery_test.go`) provando que apenas o último registro parcial é perdido, nunca a
  sessão.
- Interface `fs.FileSystem` e `FakeFileSystem` com append real, com paridade provada entre dublê e
  implementação de sistema operacional.
- Checkpoint com extensão única; teste que enumera os quatro consumidores e falha se algum divergir.
- Lock do orquestrador com `ProcessRef` gravado e liveness verificável; teste de lock órfão que hoje
  não existe.
- Sanitização aplicada aos cinco caminhos de RF-49; teste com segredo sintético por caminho.
- `.agents/hooks/post-wave.sh` idempotente sob reexecução da mesma wave e sem temporário de nome
  fixo; caso correspondente em `scripts/test-validators.sh`.
- `internal/runtime/memory/store.go:153` **não** migrado, com a decisão registrada.
- **Não-regressão (inegociável):** `make test lint vet` verdes.
- `make check-scripts-sync` verde — espelhos de `.agents/scripts/` e `.agents/hooks/` sincronizados
  (`Makefile:84`).
- `make check-mocks` verde após a extensão da interface `fs.FileSystem` (`Makefile:17`,
  `mockery.yml`).
- `make integration` e `make coverage` verdes, respeitando 75% total e 70% por pacote crítico
  (`Makefile:26`, `Makefile:42`, `Makefile:47`).
- Verificação de hard link e symlink anexada como evidência, uma por caminho migrado.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

**Núcleo da migração**

- `internal/fs/fs.go:20-32,27,163-204,170,173,181,189,193,197,200` — interface e implementação.
- `internal/fs/fake.go:140-148` — dublê.
- `internal/runtime/persistence/jsonl.go:15,21-23,38,40,46,57-58,61` — redesenho.
- `internal/runtime/persistence/report.go:69`, `internal/runtime/persistence/toolcalls.go:23`.
- `cmd/ai_spec_harness/seal_evidence.go:72`.
- `internal/taskloop/task_status_writer.go:48,91`, `internal/taskloop/reservations.go:138,228`.

**Checkpoint e lock**

- `.agents/skills/execute-task/SKILL.md:84`, `internal/sdd/state.go:556`,
  `.claude/hooks/subagent-stop-wrapper.sh:115`, `.agents/hooks/post-execute-task.sh:221-227`.
- `internal/taskloop/orchestrator_lock_unix.go:13-23`,
  `internal/taskloop/orchestrator_lock_windows.go:11-13,15`.
- `internal/runtime/memory/durable/liveness.go:8,14,34`,
  `internal/runtime/memory/durable/process_unix.go:11`,
  `internal/runtime/memory/durable/handoff_lease.go:147`.

**Sanitização**

- `internal/runtime/memory/durable/sanitization_policy.go:64,68`,
  `internal/runtime/memory/durable/facade.go:231`.

**Shell**

- `.agents/hooks/post-wave.sh:45,54,58-70,64` + espelho `.claude/hooks/post-wave.sh`.
- `.agents/skills/execute-all-tasks/SKILL.md:124,126-129`.

**Pré-requisito da tarefa 3.0, a verificar**

- `internal/taskloop/orchestrator.go:352-354,456-512`, `internal/taskloop/isolation.go:322`.

**A verificar, não migrar**

- `internal/runtime/memory/store.go:153`, `internal/install/install.go`,
  `internal/taskloop/agent_livewriter.go`.

**Documentos de referência**

- `.specs/prd-hooks-canonicos-vendor-neutral/prd.md:301-319,332` (RF-35 a RF-44, RF-49).
- `.specs/prd-hooks-canonicos-vendor-neutral/techspec.md:455-462,510-517,565-571,591-594`.
- `.specs/prd-hooks-canonicos-vendor-neutral/adr-006-atomicidade-artefatos-operacionais.md`.
- `.specs/prd-hooks-canonicos-vendor-neutral/adr-002-resultado-tipado-traducao-exit-code.md`.
