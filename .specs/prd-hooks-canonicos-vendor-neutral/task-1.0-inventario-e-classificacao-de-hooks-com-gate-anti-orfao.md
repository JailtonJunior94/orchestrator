# Tarefa 1.0: Inventario e classificacao de hooks com gate anti-orfao

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Produzir o inventário versionado de **todo hook e todo script que atue como hook** no repositório,
com os 11 campos exigidos por RF-01/RF-02, classificação `KEEP|MODIFY|REMOVE|ON-DEMAND` justificada
(RF-03), e um gate de CI que falha quando surge hook sem entrada no inventário ou entrada apontando
para arquivo inexistente (RF-05). A tarefa é a pré-condição declarada da Fase 1 da User Story:
**nenhum hook novo é criado antes do inventário existir**, motivo pelo qual 1.0 precede 4.0.

O inventário é gerado a partir do código (pacote `internal/hookinventory/`) e projetado em dois
artefatos: `docs/hook-inventory.md` (legível por humano) e `testdata/hook-inventory.json`
(verificável por máquina). Documento e JSON nunca são editados à mão; divergência entre eles é falha
de gate.

Cobre **RF-01, RF-02, RF-03, RF-04, RF-05, RF-70**. Sem dependências — paralelizável com 2.0 e 3.0.

<requirements>
- Varrer, sem exceção, as nove origens de hook do repositório: `.agents/hooks/` (7 scripts),
  `.claude/hooks/` (8 scripts), `.codex/hooks/`, `.github/hooks/`, `.agents/scripts/` (10 scripts),
  `.claude/scripts/`, `internal/runtime/hooks/` (6 hooks Go), `.opencode/plugin/governance.js` e
  `.agents/lib/`. Script que atua como hook conta como hook mesmo sem estar registrado em settings.
- Cada entrada carrega os 11 campos de RF-02: nome, localização, evento, provedor, objetivo,
  policy/gate associado, bloqueante ou não, custo aproximado, falha esperada (aberta ou fechada),
  cobertura de testes e duplicação.
- Cada entrada carrega classificação `KEEP|MODIFY|REMOVE|ON-DEMAND` com justificativa de uma linha
  ancorada em evidência do repositório (RF-03).
- **RF-04 (restrição bloqueante):** nenhum hook classificado `REMOVE` pode ser removido nesta tarefa
  nem em qualquer outra tarefa deste PRD sem que o invariante que ele protege tenha cobertura prévia
  no core. A classificação `REMOVE` é diagnóstico, não autorização de remoção.
- O gate anti-órfão é bidirecional: falha se existe arquivo de hook sem entrada no inventário **e**
  falha se existe entrada apontando para arquivo inexistente.
- Documento em PT-BR; código em inglês, zero comentários (R-STYLE-001, hard).
- Zero regressão: todos os gates da seção de Critérios de Sucesso passam ao fim da tarefa.
</requirements>

## Subtarefas

- [ ] 1.1 Criar `internal/hookinventory/` com o modelo de entrada (os 11 campos de RF-02 mais o campo
      de classificação de RF-03) e o scanner das nove origens. Varredura por caminho declarado, nunca
      por heurística de nome, para que um hook renomeado não desapareça do inventário.
- [ ] 1.2 Implementar as duas projeções a partir da mesma fonte em memória: `docs/hook-inventory.md`
      e `testdata/hook-inventory.json`. Projeções derivadas, nunca fontes — segue o mesmo princípio
      de fonte única do `adr-001-hook-contract-fonte-unica-projecoes.md` desta pasta.
- [ ] 1.3 Preencher a coluna **falha esperada** marcando explicitamente como `FAIL-OPEN` os itens
      verificados abaixo, cada um com o `arquivo:linha` do ponto de escape:
      - `.agents/hooks/subagent-stop-wrapper.sh:30,35,40,87,101` — cinco `exit 0` de degradação
        silenciosa (`HOOKS_DIR` vazio, hook não instalado, input vazio, `report_path` ausente,
        `prd_slug`/`task_id` fora da convenção);
      - `.agents/hooks/validate-governance.sh:33,38` — `parse-hook-input.sh` ausente vira aviso com
        `exit 0`, e `file_path` vazio aprova;
      - `.agents/scripts/hook-prereq-gate.sh:50,60` — validador ausente degrada para no-op e
        `resolve-references.sh` ausente encerra em `exit 0`;
      - `.agents/hooks/validate-preload.sh:54,65` — os dois `if [[ -f "$gate" ]]` **sem `else`**:
        gate ausente é simplesmente pulado, sem bloqueio e sem registro;
      - `.agents/scripts/git-operation-gate.sh:39-41` — `command_text` vazio produz `exit 0`
        (tratado em profundidade pela tarefa 2.0);
      - `scripts/git-hooks/pre-commit:41,54,58,61,66` — bloco 3 do spec-drift gate, marcado
        "permissivo" no próprio arquivo, com dois `exit 0` de degradação.
- [ ] 1.4 Preencher a coluna **cobertura de testes** marcando como `SEM TESTE` os itens verificados:
      `.claude/hooks/post-wave.sh` (nenhuma referência em `scripts/test-hooks.sh` nem em
      `tests/`; só aparece em listas de cópia e sync) e
      `.agents/scripts/validate-governance-references.sh`.
      **Correção de premissa registrada:** `scripts/git-hooks/pre-commit` **tem** cobertura —
      `scripts/test-hooks.sh:538` define `PRE_COMMIT_HOOK` e exercita os cenários `B3-*`. Classificar
      como `COM TESTE (parcial: apenas bloco 3)`, não como `SEM TESTE`.
- [ ] 1.5 Marcar `.agents/scripts/validate-governance-references.sh` como **órfão de invocação**
      (candidato a `REMOVE` ou `ON-DEMAND`, sem remoção nesta tarefa por RF-04). Evidência: grep
      repo-wide encontra apenas o próprio arquivo, seu espelho em `.claude/scripts/`, as listas de
      cópia em `scripts/sync-skills.sh:173`, `scripts/check-scripts-sync.sh:27` e
      `internal/install/install.go`, e a menção documental em `docs/evidence-gates.md:20`. Nenhum
      call-site executável. É o caso mais nítido de falsa sensação de enforcement do repositório.
- [ ] 1.6 Marcar `.claude/hooks/validate-token-budget.sh` como **LOCAL-ONLY inerte**: a exclusão do
      espelhamento é decisão registrada na allowlist `CLAUDE_LOCAL_ONLY_HOOKS`
      (`scripts/check-hooks-sync.sh:154,158`), o arquivo nunca é registrado em settings e não roda.
      Registrar a decisão no campo de justificativa em vez de tratá-lo como divergência de sync.
- [ ] 1.7 Registrar as lacunas de sync que RF-05 corrige, com a premissa corrigida:
      - `scripts/git-hooks/pre-commit` — **fora de qualquer lista de sync** (grep em `scripts/*.sh`
        só o encontra em `test-hooks.sh:538`, que é teste, não sync);
      - `.github/hooks/governance.json` — **fora de qualquer lista de sync** (zero ocorrências de
        `governance.json` em `scripts/*.sh`);
      - `check-invocation-depth.sh` — **correção da premissa**: o espelho
        `internal/embedded/assets/scripts/lib/check-invocation-depth.sh` **existe**, mas `diff`
        contra o canônico `.agents/lib/check-invocation-depth.sh` retorna **DIVERGENTE** (2.4K vs
        2.1K) e `scripts/check-skills-sync.sh:112` cobre apenas o par
        `.agents/lib/` ↔ `scripts/lib/` (`:102,105`), deixando o espelho embarcado sem gate. A lacuna
        é de **conteúdo divergente sem gate**, não de ausência de arquivo.
- [ ] 1.8 Registrar, como achado transversal do inventário, que `.agents/policies/` contém
      `code-style.md` e `governance.md` mas **nenhum hook inventariado lê esses arquivos** — todos os
      gates são auto-contidos por regex ou constante embutida. O campo `policy/gate associado` é
      preenchido com a policy que o hook *deveria* observar, e a divergência entre policy declarada e
      enforcement real vira entrada de rastreabilidade consumida pela tarefa 6.0.
- [ ] 1.9 Atender RF-70 ligando o inventário à integridade já existente: registrar, por entrada de
      hook crítico, o hash correspondente em `skills-lock.json` e garantir que alteração em hook
      crítico seja detectável pelo check `Integridade de skills (lock)`
      (`internal/doctor/doctor.go:257` — `checkSkillIntegrity`, cujo `Name` literal está em `:261`).
- [ ] 1.10 Expor o comando `ai-spec hooks inventory`, registrado em `cmd/ai_spec_harness/root.go`
      seguindo o padrão de um arquivo por subcomando.
- [ ] 1.11 Criar `scripts/check-hooks-inventory.sh` (gate anti-órfão bidirecional), ligá-lo a um alvo
      do `Makefile` na vizinhança dos demais `check-*-sync` (`Makefile:78-93`) e adicioná-lo a
      `.github/workflows/test.yml`. **Armadilha ativa herdada:** gate que não entra em nenhum job do
      CI é gate órfão — a ligação ao workflow faz parte desta tarefa, não da 13.0.
- [ ] 1.12 Escrever os testes unitários e de integração da tarefa e executar todos os gates
      declarados em Critérios de Sucesso, persistindo as saídas como evidência.

## Detalhes de Implementação

Referenciar, sem duplicar:

- `techspec.md` — `### Visão Geral dos Componentes` (linha 42) e `### Relacionamentos e Fluxo de
  Dados` (linha 76) para o lugar de `internal/hookinventory/` no desenho; `### Interfaces Chave`
  (linha 116) e `### Modelos de Dados` (linha 245) para o formato das entradas; `### Rastreabilidade
  — RF × decisão × arquivo × teste` (linha 409) para o mapeamento de RF-01..RF-05 e RF-70;
  `### Ordem de Build` (linha 453) para a precedência 1.0 → 4.0.
- `adr-001-hook-contract-fonte-unica-projecoes.md` desta pasta — princípio de fonte única com
  projeções derivadas, aplicado aqui ao par `docs/hook-inventory.md` + `testdata/hook-inventory.json`.
- `tasks.md` — `## Dependências Críticas` (bloco "1.0 precede 4.0") e `## Riscos de Integração`
  (armadilha "Gate órfão").
- `AGENTS.md` — tabela de gates por área tocada; `ADR-001` (assets via `go:embed`) para o
  espelhamento em `internal/embedded/assets/`; `ADR-002` (`FakeFileSystem` em teste unitário).

Convenções obrigatórias: injetar `internal/fs.FileSystem` e `internal/output.Printer` via construtor;
zero estado global; erros com `fmt.Errorf("context: %w", err)` em inglês; testes table-driven.

## Critérios de Sucesso

- `docs/hook-inventory.md` e `testdata/hook-inventory.json` existem, são gerados por
  `internal/hookinventory/` e são idênticos em conteúdo semântico (teste de round-trip).
- Toda entrada tem os 11 campos de RF-02 preenchidos e classificação de RF-03 com justificativa.
- As nove origens de hook estão cobertas; contagem por origem confere com o repositório
  (`.agents/hooks/` 7, `.claude/hooks/` 8, `.agents/scripts/` 10, `internal/runtime/hooks/` 6).
- Os seis itens `FAIL-OPEN` da subtarefa 1.3 e os itens `SEM TESTE` da 1.4 aparecem marcados, com
  `arquivo:linha`.
- `scripts/check-hooks-inventory.sh` falha (exit não-zero) em ambos os cenários sintéticos: hook novo
  sem entrada, e entrada apontando para arquivo inexistente. Prova executada, não apenas afirmada.
- O gate está ligado ao `Makefile` **e** a um job de `.github/workflows/test.yml` — sem gate órfão.
- `ai-spec hooks inventory` executa e regenera os dois artefatos de forma determinística.
- Nenhum hook classificado `REMOVE` foi removido (RF-04). Verificável por `git status`: nenhuma
  deleção de arquivo de hook no diff da tarefa.
- **Gates de não-regressão (inegociáveis), todos verdes:**
  - `make test lint vet` (mínimo transversal)
  - `make check-hooks-sync` — a tarefa toca `.agents/hooks/`/`.claude/hooks/`
  - `make check-scripts-sync` — a tarefa toca `.agents/scripts/` e adiciona `scripts/`
  - `make test-hooks` — comportamento dos hooks shell preservado
  - `make coverage` — 75% total e 70% por pacote crítico, `internal/hookinventory/` incluído

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
  - `internal/hookinventory/` — scanner por origem com `FakeFileSystem` (ADR-002), table-driven:
    origem vazia, origem com hook não registrado, entrada órfã, campo obrigatório ausente.
  - Round-trip: gerar Markdown e JSON da mesma fonte e assertar equivalência semântica.
  - Classificação: `REMOVE` sem cobertura prévia do invariante no core não autoriza remoção (RF-04).
  - Ligação RF-70: alteração simulada em hook crítico é detectada via `skills-lock.json`.
- [ ] Testes de integração
  - `tests/integration/hook_inventory_gate_test.go` — gate anti-órfão bidirecional sobre repositório
    temporário (`t.TempDir()`), build tag `integration`, exercitando os dois modos de falha.
  - Cobertura das nove origens contra o repositório real, para que uma origem nova não escape.
  - **Guarda anti-órfão do próprio gate:** o pacote de integração novo precisa cair em um job de
    `.github/workflows/test.yml`; asserção de presença no workflow acompanha o teste.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

A criar:
- `internal/hookinventory/` — modelo, scanner e projeções
- `docs/hook-inventory.md` — projeção legível
- `testdata/hook-inventory.json` — projeção verificável
- `scripts/check-hooks-inventory.sh` — gate anti-órfão bidirecional
- `cmd/ai_spec_harness/hooks_inventory.go` — subcomando `ai-spec hooks inventory`
- `tests/integration/hook_inventory_gate_test.go`

A modificar:
- `Makefile:78-93` — novo alvo na vizinhança dos `check-*-sync`
- `.github/workflows/test.yml` — ligação do gate a um job existente
- `cmd/ai_spec_harness/root.go` — registro do subcomando
- `scripts/sync-skills.sh`, `scripts/check-scripts-sync.sh`, `scripts/check-skills-sync.sh:102-112`,
  `scripts/check-hooks-sync.sh:154-158` — lacunas de sync de RF-05 (subtarefa 1.7)
- `internal/embedded/assets/` — espelhos exigidos pelo ADR-001

A inventariar (leitura; nenhuma remoção nesta tarefa, RF-04):
- `.agents/hooks/` — `post-execute-task.sh`, `post-wave.sh`, `pre-execute-all-tasks.sh`,
  `subagent-stop-wrapper.sh:30,35,40,87,101`, `validate-governance.sh:33,38`,
  `validate-preload.sh:54,65`, `validate-session-end.sh`
- `.claude/hooks/` — os sete acima mais `validate-token-budget.sh` (LOCAL-ONLY, inerte)
- `.codex/hooks/`, `.github/hooks/governance.json`
- `.agents/scripts/` — `git-operation-gate.sh:39-41`, `hook-prereq-gate.sh:50,60`,
  `resolve-references.sh`, `validate-bugfix-evidence.sh`, `validate-governance-references.sh`
  (órfão), `validate-refactor-evidence.sh`, `validate-review-evidence.sh`, `validate-session-end.sh`,
  `validate-skill-prerequisites.sh`, `validate-task-evidence.sh`
- `.claude/scripts/` — espelho de `.agents/scripts/`
- `internal/runtime/hooks/` — `governance.go`, `memory_events.go`, `memory_evidence.go`,
  `memory_persist.go`, `spec_drift.go`, `token_budget.go`
- `.opencode/plugin/governance.js`
- `.agents/lib/` — `parse-hook-input.sh`, `check-invocation-depth.sh`
- `scripts/git-hooks/pre-commit:41,54,58,61,66`
- `.agents/policies/code-style.md`, `.agents/policies/governance.md` (nenhum hook os lê)
- `internal/doctor/doctor.go:257,261` — check `Integridade de skills (lock)` para RF-70
- `skills-lock.json`
- `scripts/test-hooks.sh:538` — cobertura existente de `scripts/git-hooks/pre-commit`
- `docs/evidence-gates.md:20` — menção documental de `validate-governance-references.sh`
