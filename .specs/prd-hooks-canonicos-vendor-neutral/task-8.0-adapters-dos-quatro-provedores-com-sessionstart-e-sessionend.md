# Tarefa 8.0: Adapters dos quatro provedores com SessionStart e SessionEnd reais

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Declarar os 20 pares `(provedor, evento)` — 4 provedores × 5 eventos canônicos — com estado
explícito para cada um, corrigir os caminhos de configuração nativa que hoje divergem da
documentação oficial, e fechar o furo de confronto do Claude, onde o gate de paridade passa sem ter
o que ler.

O ponto central: o repositório chama de `session-end` algo que, em todas as quatro CLIs, é fim de
**turno**, não fim de sessão. Manter essa nomenclatura é falsa paridade, que o princípio P07 proíbe.
Esta tarefa separa os dois conceitos e declara explicitamente onde não há suporte.

Cobre RF-51, RF-52, RF-53, RF-54, RF-55, RF-56, RF-58, RF-69, RF-71 e RF-72. Depende da tarefa 5.0
— os adapters só podem declarar `SessionStart` e o `SessionEnd` real depois que `NewEnforcement`
(`internal/runtime/specs/enforcement.go:137-141`) deixar de exigir exatamente os três pontos de
`canonicalPoints` (`:18-22`); ampliar antes disso causa `panic` no init do catálogo
(`internal/runtime/specs/registry.go:294-296`). Paralelizável com 10.0.

<requirements>
- Os 20 pares `(provedor, evento)` declarados com estado explícito. Ausência de suporte é um estado
  nomeado (`SupportUnsupported`), nunca omissão.
- Proibida paridade simulada (princípio P07): onde o provedor não tem o ponto, declarar
  `SupportUnsupported`; onde há aproximação, declarar `SupportAdapter` com `limitation` textual.
- `.claude/settings.json` versionado passa a receber a fiação canônica e o registro marca a fonte
  como `required: true`, fechando RF-52.
- Caminhos de configuração nativa corrigidos conforme a documentação oficial de cada CLI.
- Limitações que não são corrigíveis pelo harness devem ser DECLARADAS, não contornadas.
- RF-56: gate que FALHA quando lógica de decisão de policy aparecer dentro de um adapter.
- Código em inglês, zero comentários (R-STYLE-001, hard). Documento e relatório em PT-BR.
</requirements>

## Subtarefas

- [ ] 8.1 Confirmar que a tarefa 5.0 está `done` e que `NewEnforcement` aceita o conjunto ampliado
      de pontos sem `panic` no init do catálogo (`registry.go:294-296`).
- [ ] 8.2 Renomear conceitualmente o ponto hoje chamado `session-end` para `BeforeComplete` nas
      projeções, preservando o mapeamento nativo existente de `cliHookKeyVocabulary`
      (`internal/runtime/specs/registry.go:17-38`): `Stop`/`SubagentStop` (Claude, Codex),
      `agentStop` (Copilot), `session.idle` (OpenCode).
- [ ] 8.3 Declarar `SessionEnd` como evento canônico distinto e mapeá-lo aos hooks nativos
      `SessionEnd`/`sessionEnd` de Claude, Codex e Copilot, que o repositório hoje IGNORA.
- [ ] 8.4 Declarar `SessionStart` e mapeá-lo ao hook nativo homônimo de Claude, Codex e Copilot.
- [ ] 8.5 Declarar o par `(opencode, SessionStart)` como `SupportAdapter` com `limitation`: no
      OpenCode não existe hook de início de sessão; a aproximação é o barramento `event` com
      `event.type === "session.created"`.
- [ ] 8.6 Declarar o par `(opencode, SessionEnd)` como `SupportUnsupported` e o par
      `(opencode, BeforeComplete)` como `SupportAdapter` com `limitation`: `session.idle` NÃO
      bloqueia. OpenCode é o limitante da matriz.
- [ ] 8.7 Corrigir `internal/runtime/specs/registry.go:77-92`: `.claude/settings.json` (`:79`) passa
      a `required: true`; a fonte `.claude/settings.local.json` (`:80`) permanece como origem
      opcional local.
- [ ] 8.8 Alterar `internal/install/install.go` para escrever a fiação canônica em
      `.claude/settings.json` versionado, além do merge atual em `.claude/settings.local.json`
      (`install.go:935,952`, dry-run em `:857`).
- [ ] 8.9 Corrigir o caminho do Copilot em `registry.go:87`: `.github/settings.json` →
      `.github/copilot/settings.json`, conforme a documentação oficial.
- [ ] 8.10 Ampliar as fontes reconhecidas do Codex em `registry.go:82-84`: além de
      `.codex/config.toml`, aceitar `.codex/hooks.json`, `~/.codex/hooks.json` e
      `~/.codex/config.toml`.
- [ ] 8.11 Declarar, sem tentar corrigir, a limitação do Codex: hooks untrusted são pulados em
      SILÊNCIO após alteração de hash (bug público `openai/codex#46210`). Ligar a declaração ao
      check já existente `Trust de hooks do Codex` (`internal/doctor/doctor.go:530-560`), exposto
      por `ai-spec doctor --codex-trust` (`cmd/ai_spec_harness/doctor.go:28,34`), que passa a fazer
      parte do gate de release.
- [ ] 8.12 Declarar a limitação do Copilot: `preToolUse` é fail-closed para exit 2 e para
      non-zero, mas FAIL-OPEN no timeout.
- [ ] 8.13 Revisar a precondição `PreconditionTrustedFolder` do Copilot
      (`internal/runtime/specs/registry.go:273`): a documentação de `trustedFolders` NÃO afirma que
      gateia execução de hooks, ao contrário do trusted hash do Codex (`:265`). A precondição
      declarada no repositório não tem respaldo documental — reclassificar ou registrar a lacuna.
- [ ] 8.14 Declarar que `AfterTool` NÃO bloqueia em nenhuma das quatro CLIs e que o único ponto de
      negação real é `BeforeTool`.
- [ ] 8.15 Registrar como oportunidade FORA DE ESCOPO: `PermissionRequest`/`permissionRequest`
      existe em Claude, Codex e Copilot como segundo ponto de negação real e o repositório não o usa
      em nenhum provedor.
- [ ] 8.16 Implementar o gate de RF-56: análise estática que FALHA quando lógica de decisão de
      policy aparece dentro de um adapter (`TestAdapterContainsNoPolicyDecision`, conforme
      `techspec.md` na linha de rastreabilidade de RF-56).
- [ ] 8.17 Criar a suíte por adapter (RF-58) cobrindo os 20 pares, com dispatch provado por célula
      conforme [`adr-001-hook-contract-fonte-unica-projecoes.md`](adr-001-hook-contract-fonte-unica-projecoes.md).
- [ ] 8.18 Garantir RF-71 e RF-72: a instalação toca apenas os provedores configurados e não
      introduz daemon nem processo residente.
- [ ] 8.19 Atualizar os testes que codificam os três pontos antigos e que a tarefa 5.0 deixou
      alinhados, sem reabrir `panic` no init do catálogo.
- [ ] 8.20 Executar os gates da seção "Critérios de Sucesso" e anexar as saídas ao
      `execution_report.md`.

## Detalhes de Implementação

Referência canônica: [`techspec.md`](techspec.md), seção "Rastreabilidade — RF × decisão × arquivo ×
teste" — linha `RF-51, RF-52` (`.claude/settings.json` versionado e `required`, provados por
`native_config_gate_test.go`), linha `RF-53 a RF-55` (adapters Codex, OpenCode e Copilot em
`specs/registry.go`, suíte por adapter), linha `RF-56`
(`TestAdapterContainsNoPolicyDecision`), linha `RF-58` (dispatch proof por célula) e linha
`RF-71, RF-72` (`install.go`, `install_test.go`). O modelo de evento canônico e a razão de
`SupportUnsupported` ser um estado próprio estão em
[`adr-001-hook-contract-fonte-unica-projecoes.md`](adr-001-hook-contract-fonte-unica-projecoes.md)
— não duplicar aqui.

### Fatos de pesquisa em documentação oficial (fonte registrada na ADR-001)

- O ponto que o repositório chama de `session-end` mapeia, em `internal/runtime/specs/registry.go:17-38`,
  para `Stop`/`SubagentStop` (Claude, Codex), `agentStop` (Copilot) e `session.idle` (OpenCode).
  **TODOS são fim de TURNO**, isto é `BeforeComplete`, não fim de sessão. Claude, Codex e Copilot
  têm `SessionEnd`/`sessionEnd` nativos e distintos, que o repositório IGNORA.
- `SessionStart` existe nativamente em Claude, Codex e Copilot. No OpenCode não existe como hook; a
  aproximação é o barramento `event` com `event.type === "session.created"`.
- **OpenCode é o limitante**: não tem `SessionEnd`, e seu `BeforeComplete` (`session.idle`) NÃO
  bloqueia. Declarar `SupportUnsupported` e `SupportAdapter` com `limitation` — nunca paridade
  simulada (princípio P07).
- `AfterTool` NÃO bloqueia em nenhuma das quatro CLIs. O único ponto de negação real é `BeforeTool`,
  mais `PermissionRequest`/`permissionRequest`, que existe em Claude, Codex e Copilot e que o
  repositório não usa em nenhuma. Registrar como oportunidade fora de escopo.

### RF-52 — o furo do Claude

`internal/runtime/specs/registry.go:77-92` declara as fontes de configuração nativa por provedor. As
duas do Claude, `.claude/settings.json` (`:79`) e `.claude/settings.local.json` (`:80`), estão com
`required: false` implícito, enquanto Codex (`:83`) e OpenCode (`:90`) são `required: true`. Como o
instalador grava em `settings.local.json` — não versionado — (`internal/install/install.go:935`, com
merge em `:952` e mensagem de dry-run em `:857`), o gate de paridade **não tem o que ler e passa sem
verificar nada**.

Correção: o instalador passa a escrever a fiação canônica em `.claude/settings.json` versionado, e o
registro marca essa fonte como `required: true`.

### Caminhos errados ou incompletos a corrigir

| Provedor | Declarado no repositório | Correto |
|---|---|---|
| Copilot | `.github/settings.json` (`registry.go:87`) | `.github/copilot/settings.json` |
| Codex | somente `.codex/config.toml` (`registry.go:83`) | também `.codex/hooks.json`, `~/.codex/hooks.json`, `~/.codex/config.toml` |

### Limitações a DECLARAR, não corrigir

- **Codex**: pula hooks untrusted em SILÊNCIO após alteração de hash (bug público
  `openai/codex#46210`). `ai-spec doctor --codex-trust` (`cmd/ai_spec_harness/doctor.go:28,34`) já
  detecta via RPC read-only `hooks/list`, materializado no check `Trust de hooks do Codex`
  (`internal/doctor/doctor.go:530-560`), e passa a ser parte do gate de release.
- **Copilot**: `preToolUse` é fail-closed para exit 2 e para non-zero, mas FAIL-OPEN no timeout.
- **Copilot, `trustedFolders`**: a documentação oficial NÃO afirma que gateia execução de hooks, ao
  contrário do trusted hash do Codex. A precondição declarada em
  `internal/runtime/specs/registry.go:273` (`PreconditionTrustedFolder`) não tem respaldo
  documental — registrar a lacuna em vez de sustentar a afirmação.

### RF-56

O gate deve falhar quando lógica de decisão de policy aparecer dentro de um adapter. Adapter traduz
entre o contrato canônico e o vocabulário nativo do provedor; decisão de policy vive fora dele.

## Critérios de Sucesso

- Os 20 pares `(provedor, evento)` declarados, cada um com estado explícito; nenhum par ausente por
  omissão.
- Nenhum par declarado como suportado sem hook nativo correspondente ou sem `limitation` textual —
  prova de ausência de paridade simulada (P07).
- `.claude/settings.json` versionado contém a fiação canônica após `ai-spec install .`, e o gate de
  paridade do Claude falha se o arquivo estiver ausente ou incompleto (antes da correção, passava
  vazio).
- `.github/copilot/settings.json` e as quatro fontes do Codex reconhecidas pelo registro.
- Gate de RF-56 verde no estado corrigido e comprovadamente vermelho contra um adapter de teste que
  contenha decisão de policy.
- `ai-spec doctor --codex-trust` incorporado ao gate de release, com resultado registrado.
- **Não-regressão (inegociável):** `make test lint vet` verdes, sem exceção e sem teste marcado como
  skip para passar.
- `make check-hooks-sync` verde (espelhos de hooks entre `.agents/`, `.claude/`, `.github/` e
  `internal/embedded/`).
- `make integration` verde, com a suíte por adapter (RF-58) incluída em um job que o CI executa —
  pacote de integração novo sem job é gate órfão.
- `make coverage` respeita os gates de 75% total e 70% por pacote crítico.
- Nenhum `panic` no init do catálogo de agentes em qualquer ponto da suíte.

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

- `internal/runtime/specs/registry.go` — `cliHookKeyVocabulary` (`:17-38`), `agentHookArtifacts`
  (`:40-`), `agentNativeConfigs` (`:77-92`) com Claude (`:79-80`), Codex (`:83`), Copilot (`:87`) e
  OpenCode (`:90`); preconditions do Codex (`:265`) e do Copilot (`:273`); `panic` de init
  (`:294-296`)
- `internal/runtime/specs/enforcement.go` — `CanonicalPoint` (`:12-16`), `canonicalPoints`
  (`:18-22`), validação de cobertura em `NewEnforcement` (`:137-141`)
- `internal/runtime/specs/native_config.go` — formatos e leitura das fontes nativas
- `internal/install/install.go` — escrita de `.claude/settings.local.json` (`:935`), merge (`:952`),
  dry-run (`:857`), `.github/settings.json` (`:1220`)
- `internal/doctor/doctor.go` — check `Trust de hooks do Codex` (`:530-560`)
- `cmd/ai_spec_harness/doctor.go` — flag `--codex-trust` (`:28`, `:34`)
- `tests/integration/` — suíte por adapter (RF-58) e job de CI correspondente
- [`adr-001-hook-contract-fonte-unica-projecoes.md`](adr-001-hook-contract-fonte-unica-projecoes.md)
  — eventos canônicos, projeções e `SupportUnsupported`
- [`techspec.md`](techspec.md) — rastreabilidade RF-51 a RF-56, RF-58, RF-69, RF-71, RF-72
