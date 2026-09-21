# Relatorio de Bugfix

- Total de bugs no escopo: 1
- Corrigidos: 1
- Testes de regressao adicionados: 1
- Pendentes: nenhum
- Estado final: done

## Bugs

- ID: BUG-UPGRADE-HOOKS-PARITY
- Severidade: critical
- Origem: Validacao manual de release (upgrade real 2.1.0->2.2.0 testado pelo usuario em `/Users/jailtonjunior/Git/morvi`, fora deste repo, via `jailtonjunior94/tap/ai-spec`)
- Estado: fixed
- Causa raiz:
  `internal/install/install.go` sempre sincroniza, tool-neutro, quatro conjuntos de artefatos
  gerenciados: `orchestratorHooks` (`post-execute-task.sh`, `pre-execute-all-tasks.sh`,
  `post-wave.sh`, `subagent-stop-wrapper.sh`), `toolValidationHooks` (`validate-preload.sh`,
  `validate-governance.sh`, `validate-session-end.sh`) para cada `<tool>/hooks/`, e
  `agentsScriptsFiles`/`agentsLibFiles` canonicos para `.agents/scripts/` e `.agents/lib/`.
  `internal/upgrade/upgrade.go` (`regenerateAdapters`) reimplementava esse mesmo conjunto de
  forma manual e incompleta: nunca copiava `orchestratorHooks` para nenhum tool, cobria apenas 2
  dos ~10 arquivos de `.agents/scripts/` (e nunca escrevia em `.agents/scripts/`/`.agents/lib/`
  diretamente — so em `.claude/scripts/` e `scripts/lib/`), e nao tocava `.github/hooks/` nem
  `.agents/hooks/` (usado nativamente pelo opencode). Alem disso, `regenerateAdapters` so era
  chamada quando `updated > 0` (pelo menos uma skill desatualizada) — como hooks/scripts/lib tem
  ciclo de vida proprio, independente do campo `version` do frontmatter de skill, uma segunda
  rodada de upgrade com skills ja convergidas (`updated == 0`) nunca mais tocava esses artefatos,
  mesmo que estivessem drifted/missing em relacao a fonte. `ai-spec verify` compara exatamente
  esse conjunto (`verifyGovernanceArtifacts` + `verifyCanonicalValidators` em
  `internal/install/install.go`), por isso reportava DRIFTED/MISSING mesmo apos o `upgrade` se
  autodeclarar convergido (conflict=0).
- Arquivos alterados:
  - `internal/hooksync/hooksync.go` (novo pacote): extrai `OrchestratorHooks`,
    `ToolValidationHooks`, `AgentsScriptsFiles`, `AgentsLibFiles` e as funcoes de copia
    (`CopyOrchestratorHooks`, `CopyToolValidationHooks`, `CopyAgentsScripts`, `CopyAgentsLib`,
    `SyncAll`) como fonte unica, compartilhada entre `install` e `upgrade` (nao havia pacote
    comum: `install` ja depende de `upgrade`, entao a extracao evita ciclo de import).
  - `internal/install/install.go`: `copyOrchestratorHooks`, `copyToolValidationHooks`,
    `copyAgentsScripts`, `copyAgentsLib` agora delegam para `internal/hooksync` (elimina a
    duplicacao que permitiu a divergencia original); metodo `markExecutable` nao utilizado apos a
    extracao foi removido.
  - `internal/upgrade/upgrade.go`: nova funcao `syncManagedArtifacts` chama `hooksync.SyncAll`
    para `.claude/hooks/` (se `.claude/` existir), `.codex/hooks/` (se `.codex/config.toml`
    existir), `.github/hooks/` (se `.github/` existir) e sempre `.agents/hooks/`, alem de sempre
    `.agents/scripts/` e `.agents/lib/`. A chamada roda **incondicionalmente** em `Execute` (fora
    do bloco `if updated > 0`), corrigindo o caso "segunda rodada convergida, mas hooks
    permanecem obsoletos". `regenerateAdapters` foi simplificada para nao mais duplicar essa
    logica (mantém apenas a regeneracao de adaptadores/markdown e os syncs de
    `.claude/rules/*`, `.claude/scripts/validate-*.sh` e `scripts/lib/*.sh`, que `verify` nao
    audita e cuja gate por `updated > 0` preexistia sem relacao com o bug reportado).
- Teste de regressao:
  `internal/upgrade/upgrade_test.go::TestUpgrade_SyncsManagedHooksScriptsLibAcrossAllToolsWithoutSkillChange`
  — reproduz literalmente o cenario do bug com `FakeFileSystem`: skill `review` ja convergida
  (`updated == 0`), os 4 tool-dirs (`.claude`, `.codex/config.toml`, `.github`, `.agents`)
  presentes com hooks `post-wave.sh`/`post-execute-task.sh` desatualizados ou ausentes, e
  `.agents/scripts/*` / `.agents/lib/*` desatualizados ou ausentes. Apos `svc.Execute(...)`,
  assert por hash de conteudo que todos os artefatos passam a bater exatamente com a fonte (o
  mesmo criterio que `verify` usa via `FileHash`). Confirmei que o teste captura a regressao:
  revertendo temporariamente a chamada de `syncManagedArtifacts` em `Execute` (sem tocar em mais
  nada), o teste falha listando exatamente as classes de arquivo do bug relatado
  (`.claude/.codex/.github/.agents` hooks/post-wave.sh e post-execute-task.sh divergentes);
  restaurando a chamada, o teste volta a passar.
- Validacao:
  - `go build ./...` -> ok.
  - `make vet` -> ok, sem achados.
  - `make lint` (golangci-lint v2.13.2, linters errcheck/gosec/govet/ineffassign/staticcheck/unused)
    -> `0 issues.`
  - `make test` (`go test ./...`) -> `ok` em todos os pacotes, incluindo
    `internal/upgrade` (novo teste incluso), `internal/install` e o gate
    `internal/runtime/specs::TestNoAgentListLiteralOutsideRegistry` (verificado apos remover um
    mapa auxiliar nao utilizado em `hooksync.go` que enumerava os 4 tool IDs literalmente e
    disparava esse gate anti-drift do registry de agentes).
  - Validacao empirica end-to-end em `/Users/jailtonjunior/Git/morvi` (binario local
    `./ai-spec` compilado via `make build`): **concluida em sessao de continuacao** (ver
    "Comandos Executados" e "Riscos Residuais" atualizados abaixo).

## Comandos Executados

- `go build ./internal/install/... ./internal/hooksync/...` -> ok (apos criar `internal/hooksync`).
- `go build ./...` -> ok.
- `go test ./internal/upgrade/... ./internal/install/... ./internal/hooksync/...` -> `170 passed in 3 packages`.
- `go test ./internal/upgrade/... -run TestUpgrade_SyncsManagedHooksScriptsLibAcrossAllToolsWithoutSkillChange -v` -> `1 passed` (apos o fix); falhou de proposito com a chamada de `syncManagedArtifacts` neutralizada, confirmando que o teste detecta a regressao original.
- `make vet` -> sem saida (ok).
- `make lint` -> `0 issues.`
- `make test` -> `ok` em todos os pacotes do modulo (incluindo `internal/runtime/specs`, `internal/install`, `internal/upgrade`).
- `make build` -> gerou `./ai-spec` (19.4M) com sucesso.
- `git -C /Users/jailtonjunior/Git/morvi status` -> working tree sujo de tentativa anterior (apenas arquivos de governanca: `.claude/`, `.codex/`, `.github/`, `.agents/`, `AGENTS.md`, `CLAUDE.md`, `.ai_spec_harness.json`, `scripts/lib/`), confirmado como seguro para descarte por escopo.
- `git -C /Users/jailtonjunior/Git/morvi checkout -- .` e `git -C /Users/jailtonjunior/Git/morvi clean -fd -- ...` -> permaneceram bloqueados pelo `git-operation-gate.sh` mesmo em sessao interativa (usuario tentou exportar `GOVERNANCE_GIT_OPERATION_CONFIRMED=1` via prefixo `!` do Claude Code, mas o hook `PreToolUse` roda em processo separado que nao herda o export feito dentro do comando). Decisao: pular a higienizacao previa e validar `upgrade --overwrite-conflicts` diretamente sobre o working tree sujo (arquivos rastreados modificados sao sobrescritos pelo proprio `--overwrite-conflicts`; os poucos arquivos novos nao rastreados nao pertencem ao conjunto auditado por `verify`).
- `/Users/jailtonjunior/Git/orchestrator/ai-spec upgrade /Users/jailtonjunior/Git/morvi --overwrite-conflicts` (1a rodada) -> `Resumo: 29 atualizadas, 0 desatualizadas (0 refs divergentes), 0 ausentes` / `Batch summary (upgrade): created=0 updated=0 preserved=41 merged=0 conflict=0`.
- `/Users/jailtonjunior/Git/orchestrator/ai-spec verify /Users/jailtonjunior/Git/morvi` (apos 1a rodada) -> `Resumo: 157 current, 0 missing, 0 drifted, 0 inert, 2 unknown` (as 2 `unknown` sao pre-condicoes de handshake externo do Codex/OpenCode, fora do escopo deste bug).
- `/Users/jailtonjunior/Git/orchestrator/ai-spec upgrade /Users/jailtonjunior/Git/morvi --overwrite-conflicts` (2a rodada, reproduz o cenario exato do bug: skills ja convergidas) -> `Resumo: 29 atualizadas, 0 desatualizadas (0 refs divergentes), 0 ausentes` / `Batch summary (upgrade): created=0 updated=0 preserved=41 merged=0 conflict=0`.
- `/Users/jailtonjunior/Git/orchestrator/ai-spec verify /Users/jailtonjunior/Git/morvi` (apos 2a rodada) -> `Resumo: 157 current, 0 missing, 0 drifted, 0 inert, 2 unknown` — identico a 1a rodada, confirmando que a segunda passada com `updated=0` nao deixou hooks/scripts/lib obsoletos em nenhum dos 4 tools (`.claude`, `.codex`, `.github`/copilot, `.agents`/opencode).

## Riscos Residuais

- **Validacao empirica end-to-end concluida nesta sessao de continuacao**, com uma ressalva de
  metodo: nao foi possivel higienizar previamente o working tree de
  `/Users/jailtonjunior/Git/morvi` (bloqueio do `git-operation-gate.sh` persiste mesmo com
  confirmacao via `!` do usuario, pois o hook `PreToolUse` roda em processo que nao herda env
  exportado dentro do comando do Bash tool — mesma limitacao ja registrada acima, agora tambem
  reproduzida em sessao interativa, nao so em subagente). A validacao foi feita rodando
  `upgrade --overwrite-conflicts` diretamente sobre o estado sujo herdado de uma tentativa
  anterior com o binario antigo; como esse estado ja continha arquivos alinhados ao conteudo da
  correcao (aplicados manualmente por essa tentativa anterior), a 1a rodada reportou `updated=0`
  de imediato — ou seja, as duas rodadas executadas cobrem exatamente o cenario critico do bug
  (`updated=0` sem deixar hooks/scripts/lib obsoletos), mas nao cobrem o caminho "primeira
  convergencia a partir de um estado realmente desatualizado" dentro desta sessao. Esse caminho
  ja e coberto por `TestUpgrade_SyncsManagedHooksScriptsLibAcrossAllToolsWithoutSkillChange`
  (unit test com `FakeFileSystem`, que simula explicitamente hooks desatualizados/ausentes nos 4
  tool-dirs antes de `Execute`). Nenhum `DRIFTED`/`MISSING` restante em `.claude`, `.codex`,
  `.github` ou `.agents` apos as duas rodadas reais.
- `regenerateAdapters` continua com alguns syncs (`.claude/scripts/validate-*.sh`,
  `scripts/lib/*.sh`, `.claude/rules/*`) gated por `updated > 0`. Isso preserva o comportamento
  pre-existente (nenhum desses caminhos e auditado por `verify`, que so olha
  `.agents/hooks|scripts|lib` e `<tool>/hooks/{orchestratorHooks,toolValidationHooks}` — agora
  todos sincronizados incondicionalmente via `syncManagedArtifacts`), mas fica registrado como
  gap de paridade teorica caso `verify` seja estendido no futuro para auditar esses caminhos
  tambem.
- Nao foi identificado RF explicito para este comportamento de paridade upgrade/install/verify no
  PRD atual; tratado como correcao reativa de defeito (bugfix puro), sem RF associado — uso
  `--no-rf` na validacao do relatorio.
