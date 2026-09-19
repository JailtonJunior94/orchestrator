# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Rótulo:** ADR-006
- **Título:** Atomicidade e detecção de corrupção dos artefatos operacionais de checkpoint e evidência
- **Data:** 2026-09-18
- **Status:** Proposta
- **Decisores:** dono do repositório
- **Relacionados:** [PRD](prd.md) RF-39, RF-40, RF-41, RF-42, RF-43, RF-44, RF-49 · [techspec](techspec.md) · [ADR-002 desta feature](adr-002-resultado-tipado-traducao-exit-code.md) · [ADR-004 desta feature](adr-004-prova-dispatch-por-celula.md) · [MD-003 do PRD de memória durável](../prd-memoria-duravel-agentes/adr-003-escrita-atomica-lock-camada-lease.md) · [ADR-002 do repositório](../../docs/adr/002-fake-filesystem-testes.md)

## Contexto

O PRD exige, de RF-39 a RF-44, que o checkpoint persista estado mínimo de retomada, que sua escrita
seja atômica, que seja idempotente, que estado corrompido seja detectado e reportado em vez de
consumido, que uma tarefa iniciada em uma CLI possa ser retomada por outra, e que nada de sensível
seja persistido. RF-49 estende a proibição de segredo a telemetria, logs e evidência. O levantamento
do código de produção mostra que a primitiva necessária já existe, que ela é usada por uma minoria
dos caminhos, e que o artefato de onde toda a evidência do produto depende é justamente o menos
protegido.

### A primitiva atômica existe e está correta

`internal/fs/fs.go:173` implementa `WriteFileAtomic` no `OSFileSystem` com o padrão correto:
`os.CreateTemp(dir, ".tmp-*")` no **mesmo diretório de destino** (`internal/fs/fs.go:181`), escrita
(`internal/fs/fs.go:189`), `Sync()` com erro verificado (`internal/fs/fs.go:193`), `Close()` com erro
verificado (`internal/fs/fs.go:197`) e `os.Rename` (`internal/fs/fs.go:200`). O temporário é removido
por `defer` em `internal/fs/fs.go:186`. O método está declarado na interface `FileSystem` em
`internal/fs/fs.go:27` e implementado no dublê em `internal/fs/fake.go:145`, onde é sinônimo de
`WriteFile` porque o dublê é um mapa em memória. Não existe arquivo `internal/fs/os.go`: a
implementação real vive em `internal/fs/fs.go`, ao lado da interface.

`internal/fs/fs.go:163` é a escrita não atômica: após `MkdirAll` e um `Chmod` defensivo, delega a
`os.WriteFile` em `internal/fs/fs.go:170`, que abre com `O_TRUNC`. Um crash entre o truncate e o fim
da escrita deixa o arquivo truncado ou vazio.

### A primitiva é usada por apenas cinco pacotes de produção

- `internal/runtime/memory/durable/layer.go:453` — persistência da camada de memória durável.
- `internal/runtime/memory/durable/handoff_lease.go:147` — gravação do lease de bastão.
- `internal/txn/transaction.go:254` (commit) e `:289` (rollback), com o método de fachada em `:442`.
- `internal/tracking/tracker.go:203`, que apenas delega em `:204`.
- `cmd/ai_spec_harness/memory.go:237`, `:369` e `:394` — exportação, backup e migração de memória.

Todo o restante do repositório grava por `WriteFile` ou por `os.WriteFile` direto.

### O artefato que sustenta toda a evidência é o menos protegido

`internal/runtime/persistence/jsonl.go` declara-se "writer JSONL append-only" em
`internal/runtime/persistence/jsonl.go:15`, mas não faz append. O struct mantém o arquivo inteiro em
memória no campo `buf []byte` (`internal/runtime/persistence/jsonl.go:23`); o construtor lê o arquivo
completo em `internal/runtime/persistence/jsonl.go:38` e o carrega no buffer em
`internal/runtime/persistence/jsonl.go:40`; e cada evento faz `append` em memória
(`internal/runtime/persistence/jsonl.go:57`) seguido de `WriteFile` do conteúdo **completo**
(`internal/runtime/persistence/jsonl.go:58`).

Como `WriteFile` do `OSFileSystem` é `os.WriteFile` (`internal/fs/fs.go:170`), que trunca, um crash
entre o truncate e o fim da escrita não perde o último evento: perde a **sessão inteira**. A janela
de risco cresce linearmente com o número de eventos já registrados, porque o volume reescrito a cada
evento é o arquivo todo. O custo é O(n) por evento e O(n²) por sessão.

O comentário em `internal/runtime/persistence/jsonl.go:46` afirma que "falha de escrita propaga erro
sem corromper o conteúdo anterior". Isso é verdade apenas para o `FakeFileSystem`, cuja escrita é uma
atribuição de mapa. Para o filesystem real a afirmação é **falsa**: o truncate já aconteceu quando a
escrita falha.

A justificativa declarada para esse desenho está em `internal/runtime/persistence/jsonl.go:21-22`:
"necessário pois FakeFileSystem não suporta append". Uma limitação do dublê de teste determinou a
forma do código de produção.

O `sync.Mutex` declarado em `internal/runtime/persistence/jsonl.go:20` e tomado em
`internal/runtime/persistence/jsonl.go:54` protege apenas goroutines do mesmo processo. Duas CLIs
apontadas ao mesmo `evidenceDir` carregam buffers no construtor
(`internal/runtime/persistence/jsonl.go:38`), divergem a partir dali e sobrescrevem-se mutuamente: o
último a escrever vence e apaga em silêncio os eventos do outro. Não há detecção, não há merge, não
há erro.

### Outras escritas não atômicas relevantes

- `internal/runtime/persistence/report.go:45` lê o `execution_report.md` e `:69` grava o conteúdo
  enriquecido com `WriteFile` — read-modify-write sem atomicidade nem lock.
- `internal/runtime/persistence/toolcalls.go:23` grava `tool_calls.md` com `WriteFile`.
- `cmd/ai_spec_harness/seal_evidence.go:72` grava o `result.json` **selado** com `os.WriteFile`
  direto, fora da abstração. O artefato que carrega o digest do patch é gravado sem atomicidade.
- `internal/taskloop/task_status_writer.go:48` grava o arquivo da tarefa e `:91` grava `tasks.md`,
  ambos com `WriteFile`.
- `internal/taskloop/reservations.go:138` grava o arquivo da tarefa e `:228` acrescenta linhas em
  `tasks.md`, ambos com `WriteFile`.
- `internal/runtime/memory/store.go:153` grava o `MEMORY.md` legado com `os.WriteFile` direto no modo
  `WriteModeReplace`; `internal/runtime/memory/store.go:155` usa `O_APPEND` no modo `WriteModeAppend`.

### Oito caminhos chamam `os.WriteFile` fora da abstração

Mudar a interface `fs.FileSystem` não alcança estes caminhos, porque eles não passam por ela:

| Caminho | Artefato |
|---------|----------|
| `cmd/ai_spec_harness/seal_evidence.go:72` | `result.json` selado |
| `cmd/ai_spec_harness/update_version.go:36` | `VERSION` |
| `internal/changelog/changelog.go:157` | `CHANGELOG.md` |
| `internal/runtime/approval_adapters.go:124` | artefato do ciclo de aprovação |
| `internal/runtime/approval_adapters.go:192` | artefato do ciclo de aprovação |
| `internal/runtime/memory/store.go:153` | `MEMORY.md` legado |
| `internal/specdrift/sync.go:60` | `tasks.md` na sincronização de spec-hash |
| `internal/embedded/embedded.go:61` | materialização de assets embutidos |

### O checkpoint de wave não é atômico nem idempotente, e nenhum código Go o lê

Em `.agents/hooks/post-wave.sh`, a criação do cabeçalho usa `cat > "$PARTIAL_MD.tmp"`
(`.agents/hooks/post-wave.sh:45`) seguido de `mv` (`.agents/hooks/post-wave.sh:54`). O rename é
atômico, mas o nome do temporário é **fixo**: duas invocações concorrentes no mesmo PRD colidem no
mesmo `.tmp`.

O append da wave é o bloco `{ ... } >> "$PARTIAL_MD"` entre `.agents/hooks/post-wave.sh:58` e
`.agents/hooks/post-wave.sh:70`. São cinco a seis `echo`/`cat` separados, cada um uma escrita
distinta no descritor: um crash no meio deixa seção parcial. O `>>` garante ordenação, não
atomicidade da seção.

O append é incondicional: não há verificação de que a wave já foi registrada. Reexecutar
`post-wave.sh` com o mesmo `<wave-id>` duplica a seção. Isso viola RF-41 diretamente.

Nenhum código Go lê `_orchestration_report.partial.md`. O rename final para
`_orchestration_report.md` é instruído em prosa ao LLM em
`.agents/skills/execute-all-tasks/SKILL.md:128`, com a leitura do parcial em
`.agents/skills/execute-all-tasks/SKILL.md:127` e o desempate entre parcial e final em
`.agents/skills/execute-all-tasks/SKILL.md:129`. A conclusão do checkpoint depende de o modelo
executar a instrução.

### Divergência de extensão do checkpoint, com quatro consumidores

| Consumidor | Extensão esperada |
|-----------|-------------------|
| `.agents/skills/execute-task/SKILL.md:84` | `.checkpoints/<num>.yaml` (via `.yaml.tmp`) |
| `internal/sdd/state.go:556` | `.checkpoints/<taskID>.json` |
| `.claude/hooks/subagent-stop-wrapper.sh:115` | `.checkpoints/<task_id>.json` |
| `.claude/hooks/post-execute-task.sh:221` | `.checkpoints/<TASK_ID>.yaml` (gate F25) |

O gate F25 checa `.yaml` e o importador Go lê `.json`. O comentário em
`internal/taskloop/isolation.go:307` também registra `<num>.yaml`. Dependendo do caminho exercitado,
ou o gate falha, ou a importação de estado falha com "checkpoint v2 ausente"
(`internal/sdd/state.go:558`).

### O lock do orquestrador é anônimo

`internal/taskloop/orchestrator_lock_unix.go:12` abre o arquivo de lock e
`internal/taskloop/orchestrator_lock_unix.go:16` aplica `syscall.Flock` com `LOCK_EX|LOCK_NB`. Nada é
escrito no arquivo: sem PID, sem hostname, sem timestamp, sem identificação da CLI. Quem encontra o
lock recebe `ErrWriterLocked` (`internal/taskloop/orchestrator_lock_unix.go:19`) e não tem como saber
se o dono está vivo.

No Windows, `internal/taskloop/orchestrator_lock_windows.go:13` usa `O_CREATE|O_EXCL`, sem flock. O
próprio comentário do arquivo, em `internal/taskloop/orchestrator_lock_windows.go:10-12`, admite que
a retomada exige "remover explicitamente um lock órfão após confirmação operacional": um crash deixa
lock órfão **permanente**, porque não há liberação pelo kernel ao fechar o processo.

### O mecanismo que resolveria isso já existe, no pacote errado

`internal/runtime/memory/durable/liveness.go:14` monta o `ProcessRef` corrente com PID e hostname
(`internal/runtime/memory/durable/liveness.go:15` e `:18`), e o resultado da sonda carrega o campo
`Reliable` (`internal/runtime/memory/durable/liveness.go:24`).
`internal/runtime/memory/durable/process_unix.go:21` faz `syscall.Kill(ref.PID, 0)` e distingue os
casos: `EPERM` significa processo vivo de outro dono
(`internal/runtime/memory/durable/process_unix.go:25-26`), `ESRCH` significa morto
(`internal/runtime/memory/durable/process_unix.go:27-28`), e qualquer outro erro retorna
`Reliable:false` (`internal/runtime/memory/durable/process_unix.go:30`). Quando o hostname do
`ProcessRef` diverge do local, a sonda recusa-se a opinar e devolve `Reliable:false`
(`internal/runtime/memory/durable/process_unix.go:17-18`).

`internal/runtime/memory/durable/handoff_lease.go` implementa lease com dono
(`internal/runtime/memory/durable/handoff_lease.go:12` e `:102`), razões de transferência por prazo
vencido ou dono morto (`internal/runtime/memory/durable/handoff_lease.go:49-50`), e retry com
deadline (`internal/runtime/memory/durable/handoff_lease.go:89-90`, `:173`, `:183`), gravando o
arquivo do lease atomicamente em `internal/runtime/memory/durable/handoff_lease.go:147`.

Nada disso é usado pelo orquestrador. O pacote `durable` tem a solução completa; o
`internal/taskloop` tem o problema.

### A exceção positiva

`internal/sdd/state.go` reimplementa escrita atômica com chamadas diretas de `os`:
`os.CreateTemp(filepath.Dir(path), ".sdd-state-*")` em `internal/sdd/state.go:299`, `Sync` em
`internal/sdd/state.go:311` e `os.Rename` em `internal/sdd/state.go:318` (há ainda um `Sync` de
diretório em `internal/sdd/state.go:280`). Funciona, e é o motivo de `sdd-state.json` ser o artefato
operacional mais confiável do repositório. Mas é código duplicado que não passa pela interface
`fs.FileSystem` e portanto não é exercitável com o `FakeFileSystem`, contrariando o ADR-002 do
repositório.

### Sanitização de segredos existe em um único lugar

`internal/runtime/memory/durable/sanitization_policy.go:35` define o catálogo mínimo com seis
padrões: chave privada PEM (`:38`), token de provedor (`:42`), JWT (`:46`), header `Authorization`
(`:50`), segredo de dotenv (`:54`) e credencial em connection string (`:58`). O método
`Sanitize` está em `internal/runtime/memory/durable/sanitization_policy.go:68`.

O único ponto de produção que o aplica é `internal/runtime/memory/durable/facade.go:231`, com a
política injetada em `internal/runtime/memory/durable/facade.go:77` e `:103`. Não cobre:
`events.jsonl` (`internal/runtime/persistence/jsonl.go:58`), `execution_report.md`
(`internal/runtime/persistence/report.go:69`), `tool_calls.md`
(`internal/runtime/persistence/toolcalls.go:23`), o `MEMORY.md` legado
(`internal/runtime/memory/store.go:153`) e o `.partial.md` do post-wave, que faz `cat` literal do
arquivo YAML de resultados em `.agents/hooks/post-wave.sh:64`. RF-44 e RF-49 estão descobertos nesses
cinco caminhos.

### A ADR MD-003 está desatualizada em relação ao próprio código

`.specs/prd-memoria-duravel-agentes/adr-003-escrita-atomica-lock-camada-lease.md:8` registra status
**Proposta**, mas os seis passos do seu plano de implementação estão entregues: `WriteFileAtomic` na
interface e no dublê (`internal/fs/fs.go:27`, `internal/fs/fs.go:173`, `internal/fs/fake.go:145`),
mocks regenerados (`internal/fs/mocks/file_system.go`), detecção de processo vivo por build tag
(`internal/runtime/memory/durable/process_unix.go`, `process_windows.go`), lease com prazo
(`internal/runtime/memory/durable/lease_policy.go`, `handoff_lease.go`), agregado de camada com lock
(`internal/runtime/memory/durable/layer.go`, `layer_lock_unix.go`, `layer_lock_windows.go`) e testes
de concorrência com processos reais
(`internal/runtime/memory/durable/multiprocess_integration_test.go`,
`handoff_lease_multiprocess_integration_test.go`, `layer_lock_orphan_integration_test.go`). É a ADR
que precisa ser promovida, não o código que precisa mudar.

### Restrição que condiciona a ordem de execução

`internal/taskloop/orchestrator.go:456` monta o conjunto de exclusões do patch semântico:
evidências, artefato do patch, `sdd-state.json` e `.sdd-orchestrate.lock`
(`internal/taskloop/orchestrator.go:495`), backup de migração
(`internal/taskloop/orchestrator.go:502`) e `.checkpoints/**`
(`internal/taskloop/orchestrator.go:507-511`). O padrão `.tmp-*` **não está** nesse conjunto.
Simetricamente, `internal/taskloop/isolation.go:322` tolera apenas `memory`, `.checkpoints` e
`.partials` como diretórios gerenciados pela stack, e a tolerância é por diretório, não por padrão de
nome de arquivo.

`WriteFileAtomic` cria o temporário no diretório de **destino** (`internal/fs/fs.go:181`). Migrar
qualquer escrita sob `.specs/` antes de ajustar essas duas listas faz o `.tmp-*` aparecer como
untracked no patch semântico, alterar o `PatchSHA256` e disparar o fail-closed de verificação de
patch — um falso positivo que pararia a entrega inteira. A janela é curta mas real, e um crash deixa
o temporário no disco indefinidamente.

## Decisão

### 1. Pré-requisito duro: tolerar `.tmp-*` antes de qualquer migração

Antes de migrar uma única escrita sob `.specs/`, acrescentar o padrão `.tmp-*` ao conjunto `excluded`
construído em `internal/taskloop/orchestrator.go:456`, ao lado dos artefatos operacionais já tratados
em `internal/taskloop/orchestrator.go:495` e `:507-511`, e estender a tolerância de
`internal/taskloop/isolation.go:322` para reconhecer o padrão de nome `.tmp-*` além dos três
diretórios hoje listados. Sem essa etapa zero, a primeira migração quebra o gate de patch.

### 2. Migrar por ordem de risco crescente, nunca em lote

**(a) Artefatos terminais e write-once, sem leitor concorrente.** O `result.json` selado
(`cmd/ai_spec_harness/seal_evidence.go:72`, que também sai do `os.WriteFile` direto para a
abstração), o `execution_report.md` (`internal/runtime/persistence/report.go:69`) e o `tool_calls.md`
(`internal/runtime/persistence/toolcalls.go:23`). São gravados uma vez ao final do ciclo; a troca de
`WriteFile` por `WriteFileAtomic` é semanticamente neutra e não altera nenhum teste com o dublê,
porque `internal/fs/fake.go:145` já é equivalente a `internal/fs/fake.go:140`.

**(b) `tasks.md`.** Maior ganho e maior risco: `internal/taskloop/task_status_writer.go:91`,
`internal/taskloop/reservations.go:228` e `internal/specdrift/sync.go:60`. A migração deve unificar a
exclusão mútua com o `flock -x` que a skill já tenta por fora
(`.agents/skills/execute-all-tasks/SKILL.md:124`), reaproveitando o mecanismo de lock existente. Não
criar um segundo esquema de exclusão concorrente ao da skill: dois esquemas independentes sobre o
mesmo arquivo não compõem.

**(c) `events.jsonl` por último, e como redesenho, não como troca de chamada.** Estender a interface
`fs.FileSystem` (`internal/fs/fs.go:22-40`) com append real, implementar no `OSFileSystem` sobre
`os.OpenFile` com `O_APPEND` e no `FakeFileSystem` como concatenação no mapa, e reescrever
`internal/runtime/persistence/jsonl.go` para não manter `buf`
(`internal/runtime/persistence/jsonl.go:23`) nem reler o arquivo no construtor
(`internal/runtime/persistence/jsonl.go:38`).

### 3. Não migrar o modo append de `internal/runtime/memory/store.go`

`internal/runtime/memory/store.go:155` já usa `O_APPEND`, que o kernel serializa corretamente para
escritas menores que `PIPE_BUF` no mesmo arquivo. Converter esse caminho para read-modify-write
atômico **pioraria** a concorrência, trocando serialização do kernel por uma corrida de leitura e
reescrita. Apenas `internal/runtime/memory/store.go:153` (`WriteModeReplace`) migra, e migra para a
abstração antes de migrar para atômico.

### 4. Detecção de corrupção onde hoje há zero

Os artefatos operacionais não têm nenhuma verificação de integridade. Introduzir, para o JSONL de
eventos: marcador explícito de fim de sessão e contagem de linhas esperada, verificáveis na leitura.
O consumo de um JSONL truncado ou com contagem divergente deve **recusar explicitamente** o arquivo e
reportar, nunca carregá-lo para o buffer e escrever por cima — que é exatamente o que
`internal/runtime/persistence/jsonl.go:38` a `:58` faz hoje. Isso fecha RF-42.

Para o checkpoint de wave, tornar o append de `.agents/hooks/post-wave.sh:58-70` condicional à
ausência de uma seção com o mesmo `<wave-id>`, e trocar o temporário de nome fixo
`.agents/hooks/post-wave.sh:45` por `mktemp` no mesmo diretório. Isso fecha RF-41.

### 5. Promover `ProcessRef` e lease para pacote compartilhado

Mover `internal/runtime/memory/durable/liveness.go`, `process_unix.go`, `process_windows.go` e o
núcleo de `handoff_lease.go` para um pacote compartilhado, mantendo `durable` como consumidor, e usar
o resultado em `internal/taskloop/orchestrator_lock_unix.go:12` e
`internal/taskloop/orchestrator_lock_windows.go:13`: escrever no arquivo de lock o PID, o hostname, o
identificador da CLI e o timestamp; ao encontrar lock ocupado, sondar o dono. Um dono com `ESRCH` e
`Reliable:true` libera o lock; `Reliable:false` mantém o fail-closed atual com mensagem que nomeia o
dono. Isso resolve o lock órfão permanente do Windows admitido em
`internal/taskloop/orchestrator_lock_windows.go:10-12` e viabiliza RF-43.

### 6. Unificar a extensão do checkpoint

Um único formato, com os quatro consumidores atualizados **no mesmo lote**:
`.agents/skills/execute-task/SKILL.md:84`, `internal/sdd/state.go:556`,
`.claude/hooks/subagent-stop-wrapper.sh:115` e `.claude/hooks/post-execute-task.sh:221`, mais o
comentário de `internal/taskloop/isolation.go:307`. JSON é a escolha, porque é o formato que o código
Go já parseia (`internal/sdd/state.go:556-565`) e o que o wrapper de hook já extrai; a skill e o gate
F25 passam a escrever e checar `.json`. Migração parcial reintroduz a divergência com outro sinal.

### 7. Estender a `SanitizationPolicy` existente aos cinco caminhos descobertos

Aplicar `internal/runtime/memory/durable/sanitization_policy.go:68` a `events.jsonl`
(`internal/runtime/persistence/jsonl.go`), `execution_report.md`
(`internal/runtime/persistence/report.go:69`), `tool_calls.md`
(`internal/runtime/persistence/toolcalls.go:23`), o `MEMORY.md` legado
(`internal/runtime/memory/store.go:153`) e o YAML copiado literalmente em
`.agents/hooks/post-wave.sh:64`. Reusar o catálogo de seis padrões, não criar um segundo.

## Alternativas Consideradas

### A1 — Trocar todos os `WriteFile` por `WriteFileAtomic` de uma vez

**Vantagem:** uma única tarefa, um único diff, sem faseamento.

**Desvantagens:** `WriteFileAtomic` cria inode novo e publica por `os.Rename`
(`internal/fs/fs.go:181` e `:200`). Isso descarta dono, grupo, ACL e atributos estendidos do arquivo
original, **quebra hard link** — e o instalador espelha hooks e scripts entre cinco diretórios
(`.agents/`, `.claude/`, `.github/`, `internal/embedded/assets/` e o destino instalado) — e
**destrói symlink de destino**, porque o rename substitui o link em vez de escrever através dele,
que é o comportamento atual de `internal/fs/fs.go:170`. O repositório tem `Symlink`
(`internal/fs/fs.go:100`), `IsSymlink` (`internal/fs/fs.go:146`), `EvalSymlinks`
(`internal/fs/fs.go:151`) e um guarda dedicado em `internal/fs/symlink_guard.go`, o que confirma que
symlinks são caso de uso real e não hipótese.

**Motivo da rejeição:** raio de regressão desproporcional ao ganho, sobre caminhos de instalação que
não têm o problema que a decisão quer resolver.

### A2 — Não fazer nada, aceitando o risco

**Vantagem:** custo zero, risco zero de regressão.

**Desvantagens:** RF-40 e RF-42 ficam sem cobertura, e a evidência — que é a razão de existir do
produto, por ser o que sustenta o selo de `cmd/ai_spec_harness/seal_evidence.go:72` — continua sendo
o artefato menos protegido do repositório, com janela de perda total crescendo a cada evento
(`internal/runtime/persistence/jsonl.go:58`).

**Motivo da rejeição:** requisitos funcionais explícitos sem cobertura.

### A3 — Trocar apenas a linha do `jsonl.go` para `WriteFileAtomic`

**Vantagem:** uma linha (`internal/runtime/persistence/jsonl.go:58`), nenhum teste quebrado, torna a
afirmação do comentário de `internal/runtime/persistence/jsonl.go:46` verdadeira.

**Desvantagens:** mantém o rewrite O(n) por evento e o buffer integral em memória
(`internal/runtime/persistence/jsonl.go:23`), e não resolve a sobrescrita mútua entre processos,
porque dois processos com buffers divergentes continuam publicando arquivos completos e
incompatíveis por rename — a atomicidade torna a perda *limpa*, não menor. Pior: dá aparência de
correção e remove o incentivo para o redesenho.

**Motivo da rejeição:** rejeitada explicitamente. Corrige o sintoma visível e preserva a causa.

### A4 — Introduzir banco de dados ou WAL para o estado operacional

**Vantagem:** atomicidade, durabilidade e concorrência resolvidas por um componente maduro.

**Desvantagens:** dependência nova, formato binário não auditável por diff, incompatível com o
princípio de que evidência é texto versionado e inspecionável.

**Motivo da rejeição:** o PRD declara "banco de dados introduzido apenas para hooks" como fora de
escopo em `.specs/prd-hooks-canonicos-vendor-neutral/prd.md:484`.

### A5 — Manter o lock anônimo e apenas documentar a recuperação manual

**Vantagem:** nenhuma mudança de código; basta um runbook.

**Desvantagens:** no Windows o lock órfão bloqueia **toda** retomada até intervenção humana, como o
próprio comentário de `internal/taskloop/orchestrator_lock_windows.go:10-12` reconhece, e sem
identidade do dono a documentação não pode sequer instruir como confirmar que a remoção é segura.
RF-43 (retomada cross-CLI) fica inviável, porque a segunda CLI não tem como distinguir dono vivo de
dono morto.

**Motivo da rejeição:** transfere ao operador uma decisão que o código já sabe tomar, com o
mecanismo pronto em `internal/runtime/memory/durable/process_unix.go:21`.

## Consequências

### Benefícios Esperados

- A evidência passa a sobreviver a crash: o `events.jsonl`, o `execution_report.md`, o
  `tool_calls.md` e o `result.json` selado deixam de ter janela de truncamento.
- Corrupção passa a ser detectada e reportada em vez de propagada: um JSONL truncado é recusado, não
  carregado para o buffer e reescrito por cima, fechando RF-42.
- O checkpoint de wave torna-se idempotente, fechando RF-41, e o `.partial.md` deixa de duplicar
  seções em reexecução.
- A retomada cross-CLI ganha dono identificável, viabilizando RF-43, e o Windows deixa de ter lock
  órfão permanente.
- O custo por evento do JSONL cai de O(n) para O(1), e o custo por sessão de O(n²) para O(n).
- A cobertura de sanitização passa de um para seis caminhos, fechando RF-44 e RF-49 nos artefatos
  operacionais.

### Trade-offs e Custos

- `WriteFileAtomic` exige espaço em disco para duas cópias simultâneas do arquivo
  (`internal/fs/fs.go:189` escreve o conteúdo completo no temporário antes do rename de `:200`). Para
  um `events.jsonl` de sessão longa isso **dobra o pico** de uso de disco no diretório de evidência.
- `os.CreateTemp` no mesmo diretório (`internal/fs/fs.go:181`) evita `EXDEV` no rename apenas se o
  diretório de destino não for mount point nem estiver em overlay filesystem — cenário comum em
  container de CI, onde `.specs/` ou `evidence/` podem estar montados separadamente. Nesse caso o
  rename falha e a escrita inteira falha, onde hoje ela teria sucesso.
- O rename gera evento de filesystem `RENAME` e não `MODIFY`, e substitui o inode. Observadores em
  tempo real que mantêm o arquivo aberto — `tail -f`, watchers de editor, coletores de log —
  **perdem o handle** e param de ver escritas novas.
- Append real exige **mudança de interface** em `fs.FileSystem` (`internal/fs/fs.go:22-40`), o que
  **não é aditivo** para implementadores externos e obriga a regenerar os mocks
  (`internal/fs/mocks/file_system.go`, gate `make check-mocks`). Isso contraria diretamente a
  mitigação central de MD-003, cujo passo 1
  (`.specs/prd-memoria-duravel-agentes/adr-003-escrita-atomica-lock-camada-lease.md`) foi desenhado
  como adição compatível. **O conflito é real e fica registrado aqui**: a etapa (c) do plano rompe a
  premissa de aditividade de MD-003, e por isso é a última e a única que exige revert coordenado.
- Perde-se a simetria atual entre `internal/fs/fake.go:140` e `internal/fs/fake.go:145`: com append
  real o dublê passa a ter três semânticas de escrita, e os testes precisam distinguir qual delas
  estão exercitando.

### Riscos e Mitigações

| Risco | Impacto | Mitigação | Verificação que precede a migração |
|-------|---------|-----------|-----------------------------------|
| `.tmp-*` aparece no patch semântico e altera o `PatchSHA256` | Fail-closed de verificação de patch aborta a entrega inteira, com falso positivo | Etapa zero: incluir `.tmp-*` em `internal/taskloop/orchestrator.go:456` e em `internal/taskloop/isolation.go:322` antes de tudo | Teste que grava atomicamente sob `.specs/` e prova que o patch e o isolamento não acusam |
| Quebra de hard link em caminhos espelhados pelo instalador | Cinco diretórios deixam de compartilhar inode; `make check-skills-sync` e `check-hooks-sync` divergem | Não migrar caminhos de instalação; escopo restrito a artefatos operacionais e de evidência | `find` por `-links +1` nos diretórios alvo antes de cada etapa que toque caminho de instalação |
| Destruição de symlink de destino por rename | Escrita que hoje atravessa o link passa a substituí-lo | Mesma restrição de escopo; `internal/fs/symlink_guard.go` e `IsSymlink` (`internal/fs/fs.go:146`) como guarda prévia | `lstat` do destino antes da primeira escrita atômica em cada caminho migrado |
| `EXDEV` em diretório montado ou overlay de CI | Escrita falha onde hoje sucede; job de CI quebra | Fallback documentado para `WriteFile` com erro explícito, nunca silencioso | Job de integração que monta `evidence/` em filesystem distinto e exercita a escrita |
| Pico de disco dobrado no JSONL | OOM de disco em sessão longa | Append real (etapa c) elimina o problema na origem; até lá o JSONL permanece na etapa (c), não migrado | Medir tamanho do `events.jsonl` de uma sessão real antes de decidir a ordem |
| Observador em tempo real perde o handle | Diagnóstico ao vivo deixa de funcionar | Documentar em `docs/troubleshooting.md`; recomendar releitura por path | Levantar quais hooks e workflows fazem `tail` sobre esses artefatos |
| Mudança de interface não aditiva quebra implementadores e mocks | `make check-mocks` e compilação de consumidores externos | Etapa (c) isolada, executada por último, com regeneração de mocks no mesmo commit | `make check-mocks` e `make test` verdes no commit da mudança de interface |
| Dois esquemas de exclusão concorrentes sobre `tasks.md` | Deadlock ou corrida entre o `flock` da skill e o lock novo | Unificar no mesmo mecanismo em vez de somar | Teste de concorrência com dois processos reais gravando `tasks.md` |

**Rollback:** cada etapa é reversível isoladamente, porque a migração é por caminho e a troca de
`WriteFile` por `WriteFileAtomic` é local a uma linha. A exceção é a etapa (c): a mudança de
interface exige revert coordenado da interface (`internal/fs/fs.go`), do dublê
(`internal/fs/fake.go`), dos mocks (`internal/fs/mocks/file_system.go`) e do consumidor
(`internal/runtime/persistence/jsonl.go`) no mesmo commit.

## Plano de Implementação

1. **Etapa zero — tolerar `.tmp-*`.** Incluir o padrão no conjunto `excluded` de
   `internal/taskloop/orchestrator.go:456` e na tolerância de `internal/taskloop/isolation.go:322`.
   Nenhuma outra mudança neste commit. Bloqueia todas as etapas seguintes.
2. **Verificação de hard link e symlink.** Levantar, com `find -links +1` e `lstat`, quais caminhos
   alvo são hard link ou symlink. Qualquer caminho que seja um dos dois sai do escopo da migração.
   Precede as etapas 3 a 5.
3. **Etapa (a) — artefatos terminais.** Migrar `cmd/ai_spec_harness/seal_evidence.go:72` para a
   abstração e para `WriteFileAtomic`; migrar `internal/runtime/persistence/report.go:69` e
   `internal/runtime/persistence/toolcalls.go:23`. Depende de 1 e 2.
4. **Unificação do checkpoint.** Consolidar em `.json` nos quatro consumidores
   (`.agents/skills/execute-task/SKILL.md:84`, `internal/sdd/state.go:556`,
   `.claude/hooks/subagent-stop-wrapper.sh:115`, `.claude/hooks/post-execute-task.sh:221`) e no
   comentário de `internal/taskloop/isolation.go:307`, no mesmo lote. Rodar
   `make check-skills-sync check-hooks-sync test-hooks`. Independente de 3.
5. **Idempotência e temporário único do post-wave.** Tornar o append de
   `.agents/hooks/post-wave.sh:58-70` condicional ao `<wave-id>` e trocar
   `.agents/hooks/post-wave.sh:45` por `mktemp`. Rodar `make check-hooks-sync test-hooks`.
   Independente de 3 e 4.
6. **Etapa (b) — `tasks.md`.** Migrar `internal/taskloop/task_status_writer.go:91`,
   `internal/taskloop/reservations.go:228` e `internal/specdrift/sync.go:60`, unificando com o
   `flock` de `.agents/skills/execute-all-tasks/SKILL.md:124`. Depende de 1, 2 e 3.
7. **Promoção de `ProcessRef` e lease.** Extrair de `internal/runtime/memory/durable/` para pacote
   compartilhado e adotar em `internal/taskloop/orchestrator_lock_unix.go:12` e
   `internal/taskloop/orchestrator_lock_windows.go:13`, escrevendo identidade no arquivo de lock.
   Independente de 3 a 6.
8. **Extensão da sanitização.** Aplicar
   `internal/runtime/memory/durable/sanitization_policy.go:68` aos cinco caminhos descobertos.
   Depende de 7 apenas se a política for extraída junto.
9. **Etapa (c) — redesenho do JSONL.** Estender `fs.FileSystem` com append real, implementar em
   `OSFileSystem` e `FakeFileSystem`, regenerar mocks, reescrever
   `internal/runtime/persistence/jsonl.go` sem `buf` nem releitura, e adicionar marcador de fim de
   sessão com contagem de linhas. Última etapa, commit único com mocks.

**Critério de adoção concluída:** nenhum artefato operacional ou de evidência é gravado por
truncamento sem atomicidade; o JSONL registra eventos em tempo constante; um checkpoint truncado é
recusado com erro nomeado; o lock do orquestrador identifica o dono em ambas as plataformas; e os
quatro consumidores do checkpoint concordam na extensão.

## Monitoramento e Validação

**Critérios de sucesso:**

- Teste que trunca `events.jsonl` no meio de uma linha, reabre o writer e exige **detecção explícita
  e recusa**, provando que o comportamento atual de `internal/runtime/persistence/jsonl.go:38`
  (carregar para o buffer e escrever por cima) não persiste.
- Teste que grava atomicamente sob `.specs/` e prova que o `.tmp-*` não aparece no patch semântico
  produzido por `internal/taskloop/orchestrator.go:456` nem dispara violação de isolamento em
  `internal/taskloop/isolation.go`.
- Teste de concorrência com dois processos reais gravando o mesmo `tasks.md` e o mesmo
  `events.jsonl`, sem perda de linha, no molde de
  `internal/runtime/memory/durable/multiprocess_integration_test.go`.
- Teste que simula crash com lock retido e prova que a segunda CLI identifica o dono morto por
  `ESRCH` e retoma, e que um dono vivo continua fail-closed.
- Teste de reexecução de `.agents/hooks/post-wave.sh` com o mesmo `<wave-id>` provando ausência de
  seção duplicada.
- `make test`, `make integration`, `make check-mocks`, `make check-skills-sync check-hooks-sync` e
  `make test-hooks` verdes; cobertura mantida acima dos gates de 75% total e 70% por pacote crítico.

**Sinais a acompanhar:** contagem de arquivos `.tmp-*` remanescentes nos diretórios de evidência após
execução (indica crash durante rename); ocorrências de `EXDEV` nos logs de CI; tamanho do
`events.jsonl` por sessão.

**Critério de revisão da decisão:** se a detecção de corrupção passar a rejeitar arquivos válidos —
JSONL íntegro recusado por contagem de linhas divergente, por exemplo em sessão interrompida
legitimamente — a heurística está errada e deve ser corrigida ou afrouxada antes de virar gate
bloqueante. Rejeição de arquivo válido é sintoma de heurística mal calibrada, não de corrupção real.

## Impacto em Documentação e Operação

- [`docs/evidence-gates.md`](../../docs/evidence-gates.md): registrar as garantias de atomicidade por
  artefato e o novo comportamento de recusa de JSONL corrompido.
- [`docs/troubleshooting.md`](../../docs/troubleshooting.md): procedimento de recuperação de lock
  órfão com identidade do dono, em Unix e Windows; e o que fazer quando um `.tmp-*` sobra no
  diretório de evidência.
- [`docs/degradation-matrix.md`](../../docs/degradation-matrix.md): registrar a degradação por
  `EXDEV` em filesystem montado ou overlay e o fallback declarado.
- `.agents/skills/execute-task/SKILL.md` e `.agents/skills/execute-all-tasks/SKILL.md`: extensão do
  checkpoint unificada e remoção da instrução de rename em prosa quando ela deixar de ser a única
  garantia.
- [`MD-003`](../prd-memoria-duravel-agentes/adr-003-escrita-atomica-lock-camada-lease.md): promover o
  status de **Proposta** para **Aceita**, já que os seis passos do seu plano estão entregues, e
  acrescentar nota apontando esta ADR como a que rompe a premissa de aditividade da interface.
- `AGENTS.md`: nada a alterar; a decisão não muda comandos nem convenções.

## Revisão Futura

Revisar quando o harness passar a suportar execução concorrente de múltiplas CLIs no mesmo diretório
de evidência como caso de uso de primeira classe. Nesse cenário, o append real do JSONL deixa de ser
suficiente — ordenação entre processos e merge de sessões passam a ser requisito — e a decisão de
manter evidência como texto plano versionado deve ser reavaliada contra A4.

Gatilhos adicionais de revisão: aparecimento recorrente de `EXDEV` em CI, que invalidaria a premissa
de que o temporário no mesmo diretório é suficiente; ou adoção de um filesystem de destino que não
garanta rename atômico.
