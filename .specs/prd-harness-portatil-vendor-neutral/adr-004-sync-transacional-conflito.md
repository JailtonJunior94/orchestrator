# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Sincronização transacional com categoria explícita de conflito e aliases `init`/`sync`
- **Data:** 2026-09-18
- **Status:** Proposta
- **Decisores:** dono do repositório (JailtonJunior94)
- **Relacionados:** [`.specs/prd-harness-portatil-vendor-neutral/prd.md`](prd.md) — RF-22, RF-23, RF-24, RF-25, RF-26, RF-27, RF-28, RF-29

## Contexto

O Bloco D do PRD pede que a distribuição do harness (`install`/`upgrade`) ganhe nomes de entrada
familiares (`init`/`sync`), relato explícito de conflito, não-sobrescrita de arquivo não gerenciado e
aplicação transacional em lote. O código atual não sustenta nenhuma dessas quatro propriedades.

**Superfície de CLI.** `cmd/ai_spec_harness/root.go` registra 38 subcomandos e **não** existem `init`
nem `sync`. `cmd/ai_spec_harness/upgrade.go` expõe `upgrade` com `--check`, `--langs`, `--source`,
`--ref` e `--follow-external-symlinks`. O repositório acabou de publicar a linha v2.0, de modo que
qualquer renomeação atinge um contrato público recém-estabilizado.

**Dependência dura: o manifesto não sabe o que instalou.** `internal/manifest/manifest.go:14-35`
define `checksums` como `map[string]string` indexado por **nome de skill** (valor = sha256 de um
único `SKILL.md`) e `installed_files` como lista de paths **sem hash**. Não existe `path → checksum`.
Pior: `checksums` é efetivamente *write-only* — é escrito em `internal/install/install.go:1582-1593`
e `internal/upgrade/upgrade.go:638`, e nenhum código de produção o lê (as ocorrências de `.Checksums`
são apenas escritas e testes). Sem `path → checksum` registrado, "conflito" não é uma proposição
decidível: RF-24 viraria falso positivo por construção.

O que hoje detecta divergência é **recomputação ao vivo contra a fonte**, não comparação com o
manifesto: `internal/upgrade/upgrade.go:339-341` e `internal/install/install.go:678-689`. Essa
recomputação ainda descarta erros de hash (`sourceHash, _ :=` em
`internal/upgrade/upgrade.go:339-340,352-353`): duas leituras que falham produzem `"" == ""` e um
**falso `StatusOK`** — fail-open exatamente no ponto que deveria ser fail-closed.

**Categoria de conflito inexistente.** Uma busca por conflito/conflict retorna zero ocorrências em
`internal/install`, `internal/upgrade` e `internal/uninstall`. As únicas categorias são
`installed_files` (criado) e `merged_files` (mesclado); o install sequer distingue "atualizado" de
"preservado". O `writeTracker` (`internal/install/write_tracker.go:12-17`) mantém `created` e `merged`
como `map[string]bool`, com `createdPaths()`/`mergedPaths()` em `:160-166`, precedência implícita
merged>created em `record()` (`:49-58`) e `MarkMerged` (`:64-71`) removendo o path de `created`.

**Atomicidade existe só por arquivo, e nem sempre.** `WriteFileAtomic`
(`internal/fs/fs.go:173-203`) faz CreateTemp no mesmo diretório → Write → Sync → Close → Rename. Mas
`WriteFile`, `CopyFile` e `CopyDir` são escritas diretas, sem tmp+rename
(`internal/install/write_tracker.go:104-134`). Não há nenhuma transação de lote.

**Pior caso atual — o argumento central desta ADR.** `internal/upgrade/upgrade.go:193-194` executa
`RemoveAll(skillDst)` e **depois** `CopyDir`. Se a cópia falhar, a skill fica destruída, o processo
emite apenas `Warn`, o loop **continua** e a execução pode terminar com exit 0. E
`if updated == 0 && !versionChanged { return nil }` (`:224-226`) pode nem reescrever o manifesto,
deixando `Checksums` apontando para um estado que não existe mais em disco. O mesmo padrão destrutivo
aparece em `internal/install/install.go:1576-1577`, onde `linkOrCopy` faz `RemoveAll(dst)` antes do
`CopyDir`. Somam-se a isso: o upgrade **não** usa o `writeTracker` (só o install usa), então nada que
ele cria entra em `installed_files`, degradando o uninstall posterior; e uma falha na fase 2 do
install aborta `Execute` antes da fase 4 (`internal/install/install.go:296-316`), deixando o disco com
N arquivos e **zero** rastreio deles.

Premissas preservadas: `HasFileTracking()` (`internal/manifest/manifest.go:39-41`) distingue `nil`
(manifesto legado) de slice vazia — contrato frágil, porém deliberado; `relativize`
(`internal/install/write_tracker.go:30-47`) já rejeita paths fora da raiz e essa contenção deve ser
mantida.

## Decisão

Decisão tomada, não reaberta aqui:

1. **`init` e `sync` entram como aliases** de `install` e `upgrade`, registrados em
   `cmd/ai_spec_harness/root.go`. Zero breaking change: nenhum nome, flag ou comportamento existente
   muda; os aliases herdam integralmente as flags atuais (`--check`, `--langs`, `--source`, `--ref`,
   `--follow-external-symlinks`) e são cobertos pelo teste de contrato de CLI (RF-22).
2. **O manifesto passa a registrar `path → checksum`** para cada arquivo gerenciado. Campo **aditivo**
   e `omitempty` em `internal/manifest/manifest.go`, seguindo o precedente já testado de
   `installed_files`/`merged_files` (`internal/manifest/task_2_0_test.go:9`); manifesto legado sem o
   campo continua válido e é tratado pela mesma lógica de `HasFileTracking()`. O campo `checksums`
   por-skill permanece intocado para não quebrar leitura de manifesto antigo, mas deixa de ser a base
   de qualquer decisão (RF-24).
3. **Conflito vira categoria explícita de primeira classe**: arquivo **gerenciado** cujo checksum em
   disco diverge do registrado para aquele path. Arquivo **não gerenciado** (ausente do manifesto)
   nunca é sobrescrito nem removido — é reportado, não tocado (RF-26). A terceira categoria entra no
   `writeTracker` com **precedência explícita declarada em código**: `conflict` > `merged` > `created`;
   um path marcado em conflito não é rebaixado por escrita posterior.
4. **Conflito aborta o lote inteiro por default** (fail-closed). Aplicar sobre conflito exige flag
   explícita de sobrescrita, e a flag **nomeia cada arquivo sobrescrito** no relatório (RF-27).
5. **A aplicação é transacional**: tudo aplicado ou nada aplicado. O lote é preparado em staging,
   validado, e só então promovido; falha em qualquer ponto reverte o lote e reporta o que foi
   revertido (RF-28). Isso elimina o padrão `RemoveAll` → `CopyDir` como operação irreversível.
6. **O relatório vira tipo estruturado**, no modelo já existente de
   `install.VerifyItem`/`VerifyState`/`VerifyKind` (`internal/install/install.go:36-71`), inclusive o
   campo `Remedy`. Hoje o install só tem saída textual via `internal/output/output.go:10-49`; a
   decisão de conflito não pode depender de parsing de texto.
7. **Erros de hash deixam de ser descartados.** As atribuições `sourceHash, _ :=` em
   `internal/upgrade/upgrade.go:339-340,352-353` passam a propagar com
   `fmt.Errorf("compute source hash: %w", err)`; falha de leitura produz conflito ou erro, nunca
   `StatusOK`.

Partes impactadas: `cmd/ai_spec_harness/root.go`, `internal/manifest/`, `internal/install/`
(`install.go`, `write_tracker.go`), `internal/upgrade/`, `internal/fs/`, `internal/uninstall/`
(beneficiário indireto do rastreio completo).

## Alternativas Consideradas

### (a) Best-effort com backup `.orig`

Ao encontrar divergência, sobrescrever o arquivo e preservar o conteúdo anterior como `<arquivo>.orig`.

- **Vantagens:** implementação trivial; nenhuma mudança no manifesto; nenhum estado transacional.
- **Desvantagens:** o comando sempre "tem sucesso", mesmo destruindo customização; o usuário só
  descobre a perda depois, inspecionando arquivos residuais; polui a árvore com artefatos órfãos que
  ninguém remove; não distingue arquivo gerenciado de não gerenciado.
- **Motivo da rejeição:** contraria diretamente o princípio fail-closed do repositório e o critério
  explícito de RF-26/RF-27 de não sobrescrever sem política explícita. Um backup não é uma política —
  é a ausência de uma.

### (b) Merge 3-way textual por arquivo

Resolver divergência com merge textual entre base registrada, conteúdo local e conteúdo novo.

- **Vantagens:** preservaria edição local em muitos casos sem intervenção humana; familiar a quem vem
  de VCS.
- **Desvantagens:** os arquivos em jogo são artefatos de **governança** (`SKILL.md`, `AGENTS.md`,
  regras, configurações de provedor), não código-fonte com merge driver. Um merge textual bem-sucedido
  pode produzir um artefato sintaticamente inválido, ou pior, semanticamente enfraquecido — e ser
  aceito em silêncio. Marcadores de conflito injetados em um `SKILL.md` seriam carregados como
  instrução por um agente.
- **Motivo da rejeição:** troca uma falha ruidosa e reversível por uma corrupção silenciosa de
  governança. É exatamente o modo de falha que o PRD proíbe ao vedar a promoção automática de conteúdo
  local a regra canônica (RF-27).

### (c) Renomear `install`/`upgrade` para `init`/`sync` com ciclo de depreciação

Adotar os novos nomes como canônicos, marcar os antigos como deprecados e removê-los em uma major
futura.

- **Vantagens:** superfície de CLI menor e terminologia única no longo prazo.
- **Desvantagens:** muda o contrato público da CLI logo após o v2.0.0; quebra scripts de consumidores,
  documentação publicada, a action `setup-ai-spec` e o gate de CI que hoje depende de `upgrade --check`
  (`internal/upgrade/upgrade.go:160-168`); exige ciclo de depreciação, avisos e uma major adicional
  para um ganho puramente cosmético.
- **Motivo da rejeição:** custo de contrato desproporcional ao benefício. Alias entrega o mesmo
  vocabulário ao usuário com risco zero de regressão.

## Consequências

### Benefícios Esperados

- Elimina o pior caso atual: `RemoveAll` seguido de `CopyDir` falho deixa de poder destruir uma skill
  e ainda assim terminar com exit 0.
- Torna "conflito" decidível de fato, com base em `path → checksum` registrado em vez de recomputação
  ao vivo — que não distingue "o usuário editou" de "a fonte mudou".
- Fecha o fail-open de hash: duas leituras falhas deixam de produzir `StatusOK`.
- Fecha a lacuna de rastreio do upgrade: o que ele escreve passa a entrar no rastreamento, o que torna
  o uninstall posterior correto.
- Elimina o estado "disco com N arquivos, manifesto com zero", já que manifesto e árvore passam a ser
  promovidos na mesma transação.
- Relatório estruturado permite gate de CI e teste por asserção de tipo em vez de parsing de texto.
- `init`/`sync` atendem o vocabulário esperado sem nenhuma mudança de contrato.

### Trade-offs e Custos

- Staging de lote custa espaço em disco temporário proporcional ao conjunto instalado e uma passagem
  adicional de escrita antes da promoção.
- Hash por arquivo instalado adiciona custo de I/O e CPU no install e no sync, e aumenta o tamanho do
  manifesto proporcionalmente ao número de arquivos gerenciados.
- Uma terceira categoria no `writeTracker` exige que a precedência, hoje implícita em `record()` e
  `MarkMerged`, seja tornada explícita — refatoração pequena, porém de leitura obrigatória por quem
  mantiver o pacote.
- Fluxos que hoje "passam apesar de erros" passarão a abortar. O ruído inicial é intencional e
  esperado.
- Promoção transacional de uma árvore inteira não é atômica no nível do sistema de arquivos em todos
  os casos: a garantia entregue é "reverte e reporta", não uma barreira de kernel.

### Riscos e Mitigações

| Risco | Impacto | Mitigação | Rollback |
|-------|---------|-----------|----------|
| Falha **durante a reversão** deixa estado intermediário | Alto | Reversão opera sobre staging e sobre um journal de paths promovidos; a ordem de promoção é registrada antes de qualquer escrita destrutiva, e o relatório nomeia exatamente o que foi e o que não foi revertido | Journal em disco permite retomada manual com paths nominados |
| Manifesto legado sem `path → checksum` classifica tudo como conflito | Alto | Campo `omitempty`: ausência significa "sem rastreio de checksum", tratada como o caminho legado de `HasFileTracking()` — nunca como divergência | Comportamento atual preservado por construção para manifesto antigo |
| Falso positivo de conflito por normalização de linha, permissão ou symlink | Médio | Hash sobre bytes de conteúdo com regra única e documentada; `--follow-external-symlinks` mantém a semântica atual; casos de teste por variação de EOL | Flag explícita de sobrescrita, nomeando arquivos |
| `relativize` rejeitando path fora da raiz passa a abortar lote que antes seguia | Médio | Contenção já existente é preservada e o erro passa a ser reportado com o path; sem alargamento de escopo | Nenhum — a contenção é desejada |
| `recordCopiedTree` (`internal/install/write_tracker.go:88-102`) retorna em silêncio quando `ReadDir(src)` falha, deixando arquivos copiados não rastreados | Alto | Propagar o erro com `fmt.Errorf("read source tree: %w", err)`; em regime transacional, árvore não rastreável invalida o lote | Reversão do lote |
| Regressão de performance em repositório grande | Baixo | `make bench` antes e depois; hash calculado uma vez por arquivo por operação | Nenhum — custo linear e mensurável |

## Plano de Implementação

1. **Manifesto:** adicionar o campo aditivo `path → checksum` com `omitempty` em
   `internal/manifest/manifest.go`, com testes espelhando `internal/manifest/task_2_0_test.go:9`
   (round-trip, manifesto legado, slice vazia vs `nil`). Nenhum consumidor novo ainda.
2. **Hash fail-closed:** corrigir `sourceHash, _ :=` em `internal/upgrade/upgrade.go:339-340,352-353` e
   `recordCopiedTree` em `internal/install/write_tracker.go:88-102` para propagar erro com
   `fmt.Errorf("context: %w", err)`. Isolado e testável sozinho.
3. **Escrita do manifesto:** passar install e upgrade a gravar `path → checksum` para cada arquivo
   gerenciado; fazer o upgrade usar o `writeTracker`, hoje exclusivo do install. Ainda sem consumo.
4. **Categoria conflito:** introduzir a terceira categoria no `writeTracker` com precedência explícita
   `conflict > merged > created`, substituindo a precedência implícita de `record()` e `MarkMerged`.
5. **Relatório estruturado:** tipo de relatório no modelo de `install.VerifyItem`/`VerifyState`/
   `VerifyKind` (`internal/install/install.go:36-71`), com `Remedy` preenchido para conflito; a saída
   textual de `internal/output/output.go` passa a renderizar o tipo, não a ser a fonte.
6. **Transação de lote:** staging + promoção + journal de reversão, eliminando `RemoveAll` → `CopyDir`
   como operação irreversível em `internal/upgrade/upgrade.go:193-194` e
   `internal/install/install.go:1576-1577`; remover o atalho
   `if updated == 0 && !versionChanged { return nil }` (`:224-226`) como caminho que pula a escrita do
   manifesto.
7. **Política de conflito:** abortar lote por default; flag explícita de sobrescrita que nomeia cada
   arquivo sobrescrito.
8. **Aliases:** registrar `init` e `sync` em `cmd/ai_spec_harness/root.go`, com teste de contrato de
   CLI cobrindo paridade de flags e de comportamento com `install`/`upgrade`.
9. **Gates:** `make test lint vet coverage`; `make integration`; `make check-mocks` se `mockery.yml`
   for tocado; `make bench` para a passagem de hash.

**Dependências:** os passos 4 a 7 dependem estritamente do passo 1 — sem `path → checksum` no
manifesto, conflito não é decidível. O passo 8 é independente e pode ser entregue em paralelo.

**Critérios de conclusão:** manifesto legado e novo convivem sem regressão; conflito é reportado como
categoria própria e aborta o lote; sobrescrita exige flag e nomeia arquivos; falha no meio do lote
deixa a árvore no estado inicial; `init`/`sync` cobertos por teste de contrato; nenhuma flag ou saída
existente alterada.

## Monitoramento e Validação

- **Sinais:** contagem de conflitos por execução, contagem de lotes revertidos, contagem de erros de
  hash propagados (hoje mascarados). Telemetria opt-in via `GOVERNANCE_TELEMETRY=1`, lida por
  `ai-spec telemetry report`.
- **Gate de CI existente preservado:** `upgrade --check` continua retornando erro quando
  `outdated+missing > 0` (`internal/upgrade/upgrade.go:160-168`); conflito passa a entrar na mesma
  condição de falha.
- **Critérios de sucesso:** (i) nenhum caminho de código consegue destruir um destino sem possibilidade
  de reversão; (ii) `.Checksums` deixa de ser write-only — passa a existir leitura de produção do novo
  campo `path → checksum`; (iii) teste de injeção de falha no meio do lote resulta em árvore idêntica
  à inicial; (iv) teste de arquivo não gerenciado confirma que ele não é tocado; (v) `sync` é
  determinístico — mesma origem e destino, mesmo resultado (RF-29).
- **Critérios de revisão/reversão:** taxa de falso positivo de conflito que force uso rotineiro da flag
  de sobrescrita; custo de staging inviável em repositório grande medido por `make bench`.

## Impacto em Documentação e Operação

- [`docs/guia-instalacao-universal.md`](../../docs/guia-instalacao-universal.md): documentar `init`/`sync`
  como aliases e a política de conflito.
- [`docs/troubleshooting.md`](../../docs/troubleshooting.md): seção de resolução de conflito, incluindo
  o caso do arquivo de regra editado à mão que passa a ser derivado, com a origem canônica onde a
  customização deve viver.
- [`docs/evidence-gates.md`](../../docs/evidence-gates.md): registrar conflito como condição
  fail-closed.
- `AGENTS.md` e `CLAUDE.md`: lista de comandos `ai-spec` passa a citar os aliases.
- `CHANGELOG.md`: entrada de feature aditiva, sem breaking change.
- `.github/actions/setup-ai-spec/`: verificar que a action continua usando os nomes atuais.
- `internal/embedded/assets/` e espelhos (`make check-skills-sync check-hooks-sync`) se algum asset
  citar os comandos.
- Um documento futuro em `docs/sync-transacional.md` (planejado) consolidará o modelo de staging,
  journal e categorias.

## Revisão Futura

- **Marco de revisão:** ao fechar o Bloco D do PRD, ou 90 dias após a primeira release que incluir o
  campo `path → checksum`.
- **Eventos que invalidam premissas:** necessidade real de merge assistido em artefato de governança
  com validação sintática (reabriria a alternativa (b) sob nova forma); adoção de um formato de
  manifesto versionado que torne `HasFileTracking()` desnecessário; custo de staging comprovadamente
  inviável em repositórios grandes.
- **Condições de substituição:** uma nova ADR é exigida caso se decida promover `init`/`sync` a nomes
  canônicos (alternativa (c)), remover `checksums` por-skill do manifesto, ou relaxar o default de
  abortar lote em conflito.
