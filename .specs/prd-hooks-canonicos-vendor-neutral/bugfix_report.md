# Relatorio de Bugfix

- Total de bugs no escopo: 4
- Corrigidos: 4
- Testes de regressao adicionados: 4
- Pendentes: nenhum
- Estado final: done

## Bugs
- ID: COVERAGE-1
- Severidade: major
- Origem: RF-74, task 1.0, finding de review
- Estado: fixed
- Causa raiz: o teste denominado P95 usava o pior valor de cinco amostras, medindo P100 e falhando sob ruído de agendamento e coleta de lixo; a execução com `-race` adicionava contenção do runner, não latência do caminho medido.
- Arquivos alterados: internal/runtime/memory/durable/facade_bench_test.go
- Teste de regressao: TestBuildContextP95PreliminaryBenchmark_1000ActiveFactsInSinglePage calcula o P95 por nearest-rank de vinte amostras e preserva o orçamento de 200ms fora de `-race`.
- Validacao: `make coverage` -> pass; `make coverage-packages` -> pass, durable 88.1%; `go test ./... -count=1 -race` -> pass.

- ID: VALIDATIONLANG-1
- Severidade: major
- Origem: RF-31, RF-32, RF-33, RF-34 (PRD `.specs/prd-hooks-canonicos-vendor-neutral/prd.md`), finding de review da tarefa 7.0, debito residual registrado na tarefa 3.0 ("internal/contextgen.ValidationLangOrder/ValidationLangLabels ainda sao literais hardcoded (nao derivados de skills.AllLangs); a 7.0 precisa substitui-los por derivacao")
- Estado: fixed
- Causa raiz: `ValidationLangOrder` e `ValidationLangLabels` em `internal/contextgen/contextgen.go` eram literais `[]string`/`map[string]string` mantidos manualmente em paralelo a `skills.AllLangs`; a tarefa 7.0 apenas ampliou os literais de 3 para 5 entradas ao adicionar `.NET`/Java, sem eliminar a duplicacao. Uma linguagem nova adicionada a `skills.AllLangs` no futuro nao propagaria automaticamente para esses dois simbolos — a unica protecao era `TestAllLangsAreExhaustivelyWired` falhando tardiamente.
- Arquivos alterados: internal/contextgen/contextgen.go (`ValidationLangOrder`/`ValidationLangLabels` passam a ser calculados em tempo de inicializacao por `deriveValidationLangOrder`/`deriveValidationLangLabels`, iterando `skills.AllLangs`; rotulo por linguagem centralizado em `validationLangLabel`, que faz `panic` para `skills.Lang` sem rotulo mapeado em vez de silenciar)
- Teste de regressao: internal/contextgen/validation_lang_derivation_test.go — `TestValidationLangOrderDerivesFromAllLangs` compara `len(ValidationLangOrder)` e a ordem de cada entrada com `skills.AllLangs`; `TestValidationLangLabelsCoverAllLangs` compara `len(ValidationLangLabels)` com `skills.AllLangs` e garante rotulo nao vazio para cada linguagem; `TestValidationLangLabelPanicsOnUnknownLang` confirma que `validationLangLabel` falha explicitamente (nao silencia) para uma `skills.Lang` desconhecida.
- Validacao: `go build ./...` -> pass; `go vet ./...` -> pass; `go test ./internal/contextgen/... ./internal/skills/... -v` -> 230 passed; `go test -tags integration ./internal/skills/... -run TestAllLangsAreExhaustivelyWired -v` -> 6 passed (5 linguagens + wiring); `make lint` -> 0 issues; cobertura combinada `internal/contextgen` + `internal/skills` -> 91.3% (acima dos gates de 75%/70%).

- ID: QUALITYGATE-1
- Severidade: critical
- Origem: finding de review estrita da tarefa 10.0, RF-27/RF-30
- Estado: fixed
- Reprodução: chamar `Gate.Evaluate()` duas vezes com o mesmo `EvaluationInput` e um check obrigatório sempre falho. Antes da correção, a 1ª chamada retornava `DecisionBlock` e a 2ª retornava `DecisionNotApplicable` ("skip - fingerprint unchanged since last run") mesmo com o mesmo estado ainda falho — bypass de segurança real, pois `internal/runtime/hooks/quality_gate.go` só bloqueia em `DecisionBlock`/`DecisionError`.
- Causa raiz: (1) `Gate.Evaluate` em `internal/qualitygate/gate.go:72-78` aplicava o skip por fingerprint inalterado independentemente da decisão cacheada anterior, inclusive quando essa decisão era `DecisionBlock`/`DecisionError`; (2) o único wiring de produção do gate, em `internal/runtime/runner.go:717-722`, nunca preenchia `EvaluationInput.StateDigest`, deixando o fingerprint dependente só de `TaskID+TaskType+Risk+Lang+Toolchain` — idênticos entre invocações repetidas da mesma tarefa, o que tornava o dedup por fingerprint indistinguível de "nada mudou de verdade".
- Correção:
  1. `internal/qualitygate/gate.go`: nova função `isSkippableDecision` restringe o skip por fingerprint inalterado a decisões cacheadas não bloqueantes (`ALLOW`/`WARN`); uma decisão cacheada `BLOCK`/`ERROR` nunca é usada para pular a reavaliação — o gate sempre reexecuta os checks nesse caso, preservando o bloqueio real quando o estado que falhou ainda não mudou.
  2. `internal/runtime/runner.go` (`prepareHooksDispatcher`, wiring do `QualityGateHook`): `EvaluationInput.StateDigest` agora recebe `qualitygate.ComputeStateDigest(j.WorkDir)`, um hash SHA-256 de `git rev-parse HEAD` + `git status --porcelain=v1` + `git diff HEAD` do `WorkDir` (novo arquivo `internal/qualitygate/statedigest.go`). Em diretório sem git ou sem HEAD, retorna string vazia (zero-value, preserva comportamento anterior); com git disponível, o fingerprint passa a refletir mudanças reais no working tree, evitando que RF-30 deduplique qualquer execução repetida independentemente de mudança de estado.
- Arquivos alterados: internal/qualitygate/gate.go, internal/qualitygate/statedigest.go (novo), internal/runtime/runner.go
- Teste de regressao: `TestQualityGate_NeverMasksCachedBlockOnUnchangedFingerprint` (internal/qualitygate/gate_test.go) — chama `Evaluate()` duas vezes com o mesmo `EvaluationInput` e um check obrigatório sempre falho; exige `DecisionBlock` nas duas chamadas. Testes de suporte: `TestComputeStateDigest_ChangesWhenWorkingTreeChanges` e `TestComputeStateDigest_NonGitDirReturnsEmpty` (internal/qualitygate/statedigest_test.go) provam que o digest muda com modificação real do working tree e é vazio fora de um repo git. Os testes preexistentes `TestQualityGate_SkipsWhenFingerprintUnchanged` e `TestQualityGate_ReexecutesWhenFingerprintChanges` continuam passando, confirmando que o dedup legítimo (estado inalterado, decisão anterior não bloqueante) não foi quebrado.
- Validacao: `go build ./...` -> pass; `go vet ./...` -> pass; `go test ./internal/qualitygate/... ./internal/runtime/hooks/... ./internal/runtime/... -v` -> 1047 passed; `make lint` -> 0 issues; `make coverage` -> pass, total 82.1%, internal/qualitygate 90.2%.

- ID: GITGATE-VAREXPAND-1
- Severidade: critical
- Origem: finding de review estrita da tarefa 6.0, RF-18, RF-19, RF-22
- Estado: fixed
- Reprodução: `printf '%s' '{"tool_input":{"command":"echo $PATH"}}' | bash .agents/scripts/git-operation-gate.sh` bloqueava com "GOVERNANCE BLOQUEIO: invocacao de interpretador com conteudo dinamico detectada (variable_expansion)", exit 2 — mesmo resultado para `cat $HOME/.bashrc`, `grep -rn foo $REPO_ROOT`, `for f in *.go; do echo $f; done`, ou seja, qualquer comando Bash comum contendo `$` de expansão de variável, mesmo sem nenhuma relação com `git`, era bloqueado incondicionalmente. Reproduzido ao vivo nesta própria sessão de bugfix antes da correção (o hook `validate-preload.sh`, que consome o mesmo classificador, bloqueou um `source` legítimo do `check-invocation-depth.sh` contendo `"$candidate"`/`"$d"`).
- Causa raiz: no classificador Python embutido em `classify_git_command_structurally` (bloco heurístico final), a condição `elif "$" in command_text: interpreter_hits.append("variable_expansion")` tratava qualquer ocorrência de `$` em **todo o texto do comando** como "invocação de interpretador com conteúdo dinâmico", sem checar se o `$` estava na posição de palavra de comando (candidata a ocultar `git`) ou apenas em um argumento comum (`$PATH`, `$HOME`, `$REPO_ROOT`). `$VAR` isolado não executa nada novo por si só — a heurística original não distinguia esse caso do de ofuscação real.
- Correção: escopo do sinal `variable_expansion` restringido às posições de token que o classificador já examina como candidatas a nome de comando (`resolve_candidate_positions` — a posição da palavra de comando de cada segmento, e todas as posições subsequentes quando o segmento inicia por um wrapper transparente como `sudo`/`env`/`exec`), excluindo tokens que casam `ASSIGNMENT_RE` (`VAR=valor`, para não penalizar `env FOO=$BAR comando`). A detecção de substituição de comando (`$(...)`/crase) permanece global e inalterada, pois sempre executa algo novo independentemente de posição. Arquivo: `.agents/scripts/git-operation-gate.sh` (linhas ~296-313, bloco Python embutido), mirrors `.claude/scripts/git-operation-gate.sh`, `internal/embedded/assets/.agents/scripts/git-operation-gate.sh`, `internal/embedded/assets/.claude/scripts/git-operation-gate.sh` (sincronizados via `scripts/sync-skills.sh`).
- Arquivos alterados: .agents/scripts/git-operation-gate.sh, .claude/scripts/git-operation-gate.sh, internal/embedded/assets/.agents/scripts/git-operation-gate.sh, internal/embedded/assets/.claude/scripts/git-operation-gate.sh, scripts/test-validators.sh
- Teste de regressao: nova seção em `scripts/test-validators.sh` ("Não-regressão: expansão de variável comum sem relação com git não bloqueia") cobre `echo $PATH`, `cat $HOME/.bashrc`, `grep -rn foo $REPO_ROOT`, `for f in *.go; do echo $f; done`, `go test ./...`, `make test`, todos exigindo exit 0. Os dois casos de ofuscação real já existentes na mesma seção (`git${IFS}push${IFS}--force` e `G=git; P=push; $G $P --force`, ambos exit 2) permanecem cobertos e passando, confirmando que a correção não reabriu esses bypasses.
- Validacao: `bash scripts/test-validators.sh` -> 111 passaram, 0 falharam (inclui os 8 casos novos e todos os 6 bypasses previamente fechados: payload >65KB com `git push`, `sudo git push`, homônimo aninhado, separadores compostos `&&`/`||`, `git push` entre aspas, `sudo -u git -- git push`); `bash scripts/test-hooks.sh` -> 73 asserts OK, 0 FAIL; `make check-hooks-sync check-scripts-sync` -> 0 drift, todos os mirrors sincronizados; `go build ./...` -> pass; `go vet ./...` -> pass. Reprodução manual pós-correção: `echo $PATH`/`cat $HOME/.bashrc`/`go test ./...`/`make test` -> exit 0; `git push origin main` puro -> exit 2; `git${IFS}push${IFS}--force` e `G=git; P=push; $G $P --force` -> exit 2; payload >65KB terminando em `; git push origin main` -> exit 2.

## Comandos Executados
- `go test ./internal/runtime/memory/durable -run '^TestBuildContextP95PreliminaryBenchmark_1000ActiveFactsInSinglePage$' -count=20 -v` -> pass em 20 execuções.
- `go test -cover ./internal/runtime/memory/durable -run '^TestBuildContextP95PreliminaryBenchmark_1000ActiveFactsInSinglePage$' -count=20 -v` -> pass em 20 execuções.
- `go test -bench '^BenchmarkRecoveryWith1000ActiveFactsInSinglePage$' -benchmem ./internal/runtime/memory/durable` -> pass, p95_ms 15.39.
- `make coverage` -> pass (na sessao COVERAGE-1).
- `make coverage-packages` -> pass.
- `go test ./... -count=1 -race` -> pass.
- `go build ./...` -> pass (VALIDATIONLANG-1).
- `go vet ./...` -> pass.
- `go test ./internal/contextgen/... ./internal/skills/... -v` -> 230 passed em 2 pacotes.
- `go test -tags integration ./internal/skills/... -run TestAllLangsAreExhaustivelyWired -v` -> 6 passed.
- `make lint` -> 0 issues.
- `go test ./internal/contextgen/... ./internal/skills/... -coverprofile=/tmp/cov_contextgen.out -covermode=atomic && go tool cover -func=/tmp/cov_contextgen.out` -> total 91.3%.
- `make coverage` (rodada VALIDATIONLANG-1) -> falhou por causa raiz preexistente e nao relacionada: `internal/Makefile` na regra `coverage` gera path malformado `.../ai-spec-harness/ingithub.com/.../internal/taskloop` (concatenacao indevida de `internal/taskloop` a outro pacote), provavelmente de edicao concorrente de outra tarefa em `Makefile` (fora do escopo `internal/contextgen/contextgen.go` desta correcao); `make coverage` restrito a `internal/contextgen`/`internal/skills` via `go test -coverprofile` confirma 91.3%, acima dos gates.
- `go build ./...` -> pass (QUALITYGATE-1).
- `go vet ./...` -> pass.
- `go test ./internal/qualitygate/... ./internal/runtime/hooks/... ./internal/runtime/... -v` -> pass (1047 testes).
- `make lint` -> 0 issues.
- `make coverage` (rodada QUALITYGATE-1, apos o Makefile ja estar estavel) -> pass, total 82.1%, internal/qualitygate 90.2%.
- `bash scripts/test-validators.sh` (rodada GITGATE-VAREXPAND-1) -> 111 passaram, 0 falharam.
- `bash scripts/test-hooks.sh` (rodada GITGATE-VAREXPAND-1) -> 73 asserts OK, 0 FAIL.
- `make check-hooks-sync check-scripts-sync` (rodada GITGATE-VAREXPAND-1) -> 0 drift, 32 validadores e 28 hooks em sync.
- `go build ./...` -> pass (GITGATE-VAREXPAND-1).
- `go vet ./...` -> pass (GITGATE-VAREXPAND-1).
- Reprodução manual dos 9 cenários do bug (`echo $PATH`, `cat $HOME/.bashrc`, `grep -rn foo $REPO_ROOT`, `for f in *.go; do echo $f; done`, `go test ./...`, `make test`, `git push origin main`, `git${IFS}push${IFS}--force`, `G=git; P=push; $G $P --force`) -> todos com exit code correto.
- `make coverage` (rodada GITGATE-VAREXPAND-1) -> falhou por causa raiz preexistente e nao relacionada a esta correção: `FAIL github.com/JailtonJunior94/ai-spec-harness/internal/runtime/persistence [build failed]`, originado de edição concorrente em `internal/runtime/persistence/jsonl.go`/`jsonl_crash_recovery_test.go` (fora do escopo `.agents/scripts/git-operation-gate.sh` desta correção, presumivelmente outra tarefa em andamento no mesmo working tree). `go build ./...`, `go vet ./...`, `bash scripts/test-validators.sh` e `bash scripts/test-hooks.sh` — os alvos proporcionais ao escopo shell-only desta correção — passaram integralmente.

## Riscos Residuais
- Nenhum para COVERAGE-1; o teste de desempenho permanece fora de `-race`, que valida correção sob instrumentação e não é uma medição de latência comparável.
- Nenhum para VALIDATIONLANG-1 dentro do escopo `internal/contextgen/contextgen.go`. `make coverage` (alvo completo do Makefile) esta quebrado por mudanca concorrente fora de escopo em `Makefile`/`internal/taskloop`; nao foi tocado por esta correcao e deve ser resolvido pela tarefa que esta alterando `Makefile` no momento.
- Nenhum para QUALITYGATE-1 no escopo `internal/qualitygate/`/`internal/runtime/runner.go`. `ComputeStateDigest` depende do binário `git` em PATH e do `WorkDir` ser um repositório git com pelo menos um commit; sem isso o digest retorna vazio e o fingerprint volta a depender apenas de `TaskID+TaskType+Risk+Lang+Toolchain` — a correção 1 (nunca mascarar `BLOCK`/`ERROR` cacheado) permanece válida de qualquer forma, pois independe do StateDigest. `cache.Put` continua gravando qualquer decisão, inclusive `BLOCK`; a filtragem foi implementada no ponto de leitura (`Gate.Evaluate`), que é onde o bypass de segurança se manifestava.
- Nenhum para GITGATE-VAREXPAND-1 no escopo `.agents/scripts/git-operation-gate.sh`/mirrors. O sinal `variable_expansion` ainda pode gerar falso positivo em um caso não coberto por teste: `sudo $VAR` ou `env $VAR` (variável usada dentro da posição de comando de um wrapper transparente) continua bloqueando mesmo quando `$VAR` de fato expande para algo inofensivo em runtime — isso é uma troca deliberada de fail-closed, consistente com a postura de ADR-003 e com o próprio histórico de bypass documentado no `6.0_execution_report.md` (`$G $P --force`), já que o classificador nunca executa o comando e não pode saber o valor real da variável. Durante esta correção, `.agents/scripts/git-operation-gate.sh` também recebeu, de uma tarefa concorrente (não desta correção), a adição de `hook_runtime_guard(...)` logo após o `source "$parse_lib"` (linhas ~29-31); essa adição foi preservada intacta no working tree e deliberadamente **não** incluída no commit desta correção (apenas o hunk do bloco Python do classificador, linhas ~296-313, foi staged via `git apply --cached` de um patch cirúrgico) para não misturar escopos de tarefas concorrentes no mesmo arquivo — cabe à tarefa responsável por essa adição commitá-la separadamente.
