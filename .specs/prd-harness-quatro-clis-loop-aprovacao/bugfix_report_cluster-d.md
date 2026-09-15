# Relatorio de Bugfix

- Total de bugs no escopo: 3
- Corrigidos: 3
- Testes de regressao adicionados: 3
- Pendentes: nenhum
- Estado final: done

## Bugs

### BUG-005

- ID: BUG-005
- Severidade: minor
- Origem: RF-28/RF-29 (tarefa 9.0), subtarefa 9.12 do task file `task-9.0-hooks-e-paridade.md` — review adversarial da tarefa 9.0 (achado medium)
- Estado: fixed
- Causa raiz: o alvo `test-hooks-live` foi introduzido no `Makefile` sem o bloco de comentario de escopo que o alvo irmao `test-acp-live` ja possui, deixando a declaracao "camada 3 nao e gate de merge" presente apenas no workflow (`.github/workflows/hooks-live.yml`), nao no `Makefile`.
- Arquivos alterados: `Makefile`
- Teste de regressao: nao aplicavel (mudanca documental); verificacao manual do conteudo do `Makefile` confirmando o bloco de comentario espelhando `test-acp-live:106-109`.
- Validacao: leitura do `Makefile` apos a edicao confirmando o comentario imediatamente acima de `test-hooks-live` com o mesmo estilo/conteudo (escopo nightly, build tag protegendo compilacao, "nao e gate de merge").

### BUG-006

- ID: BUG-006
- Severidade: minor
- Origem: RF-28/RF-29 (tarefa 9.0) — review adversarial da tarefa 9.0 (achado medium)
- Estado: fixed
- Causa raiz: `TestMandatoryMatrixWrittenConfigContainsNativeKeyAndValidator` verificava a presenca da chave nativa e do validador esperado via `strings.Contains` sobre o conteudo bruto do arquivo de config escrito, sem escopar a busca pela chave estrutural correspondente. Isso reproduz a mesma classe de fragilidade do defeito original da tarefa (substring pode sobreviver textualmente sem estar de fato ligada ao ponto canonico correto).
- Arquivos alterados: `internal/install/hooks_parity_matrix_test.go`
- Teste de regressao: o proprio `TestMandatoryMatrixWrittenConfigContainsNativeKeyAndValidator` foi reescrito para:
  - Claude/Codex/Copilot (`.claude/settings.local.json`, `.codex/hooks.json`, `.github/hooks/governance.json`, todos JSON): `json.Unmarshal` do conteudo escrito, navegacao estrutural ate `hooks[<nativeKey>]`, e coleta recursiva de apenas os valores string dentro desse subarvore (`jsonHookKeyScope`/`collectJSONStringValues`) — a chave nativa precisa existir de fato como chave JSON, e o validador precisa aparecer dentro do subarvore escopado, nao em qualquer lugar do arquivo.
  - OpenCode (`.opencode/plugin/governance.js`, JS): parsing estrutural via casamento de chave de handler async (`"tool.execute.before": async ...`) + extracao de corpo de funcao por balanceamento de chaves (`findJSHandlerBody`/`findJSFunctionBody`/`extractBalancedBraces`), seguindo uma camada de chamadas locais (`runCanonicalValidator`, `runSessionEndValidator`) e resolvendo identificadores de constante (`CANONICAL_PRE_TOOL_SCRIPT`, `CANONICAL_SESSION_END_SCRIPT`) contra suas declaracoes `const NOME = "valor"` no arquivo — em vez de checar a string do script em qualquer parte do arquivo.
  - Mantidas as 12 celulas (4 agentes x 3 pontos canonicos) e o mesmo conjunto de validadores aceitaveis, sem reducao de cobertura.
- Validacao: `go test ./internal/install/... -run TestMandatoryMatrixWrittenConfigContainsNativeKeyAndValidator -v -count=1` -> pass.

### BUG-010

- ID: BUG-010
- Severidade: minor (low, mandato desta rodada exige 0 ressalvas)
- Origem: RF-10 (tarefa 7.0) — review adversarial da tarefa 7.0 (achado low)
- Estado: fixed
- Causa raiz: `ValidateFixedArgsFormat` era aplicada apenas ao `FixedArgs` da spec principal em `newSpec`/`newSpecWithBootstrap`, nunca ao `FixedArgs` de cada `FallbackLauncher` do registro, deixando o gate assimetrico entre a spec principal e o fallback.
- Arquivos alterados: `internal/runtime/specs/spec.go`, `internal/runtime/specs/format_gate_test.go`
- Detalhe de design: uma aplicacao literal e ingenua de `ValidateFixedArgsFormat` ao array inteiro de `FallbackLauncher.FixedArgs` quebraria os 4 fallbacks reais hoje registrados (Claude, Codex, Copilot, OpenCode), pois todos seguem a convencao `npx --yes <pacote>[@versao] [args-do-binario...]` — um flag (`--yes`) sempre seguido de um item posicional (o pacote), que e uma gramatica distinta e legitima da convencao "prefixo posicional, depois flags" usada pelo `FixedArgs` da spec principal. Por isso foi adicionada `ValidateFallbackFixedArgsFormat`, que identifica o primeiro item posicional do array (o pivo — equivalente ao pacote/alvo resolvido pelo launcher) e aplica a mesma invariante de `ValidateFixedArgsFormat` apenas ao segmento apos o pivo (os argumentos efetivamente repassados ao binario resolvido, semanticamente equivalentes ao `FixedArgs` da spec principal). Isso preserva os 4 fallbacks reais (todos validos sob essa leitura) e ainda rejeita um fallback genuinamente malformado, ex.: `["--yes", "pkg@1.0.0", "--flag", "acp"]` (item posicional apos flag dentro do segmento pos-pivo).
- Teste de regressao:
  - `TestNewSpecPanicsOnInvalidFallbackFixedArgsFormat` (`internal/runtime/specs/format_gate_test.go`): `newSpec` com um `FallbackLauncher.FixedArgs` malformado (`["--yes", "artificial-pkg@0.0.0", "--flag", "acp"]`) deve panicar.
  - `TestValidateFallbackFixedArgsFormatAcceptsNpxPrefixConvention` (`internal/runtime/specs/format_gate_test.go`): confirma que a convencao `npx --yes <pkg> [acp|--acp]` (os 3 formatos reais hoje em uso: Claude/Codex, OpenCode, Copilot) permanece aceita, e que o caso malformado acima e rejeitado por `ValidateFallbackFixedArgsFormat` diretamente (sem depender do panic).
  - Confirmado que os 4 fallbacks reais do registro (`claude.go`, `codex.go`, `copilot.go`, `opencode.go`) continuam construindo sem panic via `go test ./internal/runtime/specs/... -run TestRegistry -v` (dentro da suite completa do pacote).
- Validacao: `go test ./internal/runtime/specs/... -count=1 -v` -> pass (inclui os testes novos e os 4 agentes reais do registro).

## Comandos Executados

- `go build ./...` -> ok, sem erros.
- `go vet ./...` -> ok, sem achados.
- `go test ./internal/install/... ./internal/runtime/specs/... -count=1 -v` -> `Go test: 191 passed in 2 packages`.
- `make check-scripts-sync` -> `Validadores em sync: 24 / Drift / missing: 0` (nao afetado pelas mudancas, confirmado).
- `go test ./... -count=1` -> `Go test: 2905 passed in 75 packages`.

## Riscos Residuais

- BUG-010: `ValidateFallbackFixedArgsFormat` nao detecta uma ordem trocada entre pacote e subcomando ANTES do pivo (ex.: subcomando aparecendo como primeiro item positional em vez do pacote), porque a funcao so audita o segmento apos o primeiro item posicional. Esse cenario especifico nao foi levantado pela review adversarial e exigiria modelar explicitamente a gramatica de cada launcher (ex.: `npx` vs. futuros launchers com convencao diferente) — fora do escopo minimo desta correcao.
- BUG-006: a resolucao estrutural do `governance.js` do OpenCode segue apenas uma camada de chamadas de funcao (handler -> funcao local chamada diretamente) para localizar o identificador de constante do script; uma futura refatoracao do plugin que adicione mais niveis de indirecao entre o handler e o uso da constante do script exigiria estender `jsHookKeyScope` para mais saltos.
