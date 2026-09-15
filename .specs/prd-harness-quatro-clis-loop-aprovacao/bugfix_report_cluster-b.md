# Relatorio de Bugfix

- Total de bugs no escopo: 2
- Corrigidos: 2
- Testes de regressao adicionados: 2
- Pendentes: nenhum
- Estado final: done

## Bugs

### BUG-002
- ID: BUG-002
- Severidade: major
- Origem: RF-19/RF-20 (tarefa 8.0 — instalador OpenCode), review adversarial da tarefa 8.0 (Achado 1, medium -> major)
- Estado: fixed
- Causa raiz: `internal/install/install.go` chamava `specs.ValidatePermissionBlock(permission)` e `specs.ValidatePermissionBlock(decodeWrittenPermission(merged))` sem o parametro variadico `toolsThatMustExist`. Como `ValidatePermissionBlock` (`internal/runtime/specs/opencode.go:40`) so reprova "deny" total em ferramenta obrigatoria quando essa lista e informada, o gate de defesa-em-profundidade contra um default futuro que negue uma ferramenta necessaria ficava inerte em producao — so disparava em testes unitarios que passavam a lista manualmente.
- Arquivos alterados:
  - `internal/install/install.go`: adiciona `var openCodeRequiredTools = []string{"bash", "edit", "write", "multiedit", "patch"}` (mesma lista de `MUTATING_TOOLS` do plugin OpenCode em `internal/embedded/assets/.opencode/plugin/governance.js`, para consistencia cross-artefato); passa `openCodeRequiredTools...` nas duas chamadas de `specs.ValidatePermissionBlock` em `installOpenCode`; introduz `var openCodePermissionFactory = specs.DefaultOpenCodePermission` como seam de teste para o default de permissao (usado no lugar da chamada direta a `specs.DefaultOpenCodePermission()`), permitindo simular um default corrompido no teste de regressao sem alterar o comportamento de producao.
  - `internal/runtime/specs/opencode_config_test.go`: novo teste de regressao provando que o default real passa mesmo com a lista de ferramentas obrigatorias.
  - `internal/install/install_opencode_test.go`: novo teste de regressao que exercita o caminho real de producao (`svc.Execute` -> `installOpenCode`) com um default artificialmente corrompido.
- Teste de regressao:
  - `TestValidatePermissionBlockAcceptsDefaultOpenCodePermissionWithRequiredTools` (`internal/runtime/specs/opencode_config_test.go`): confirma que `specs.DefaultOpenCodePermission()` continua passando na validacao quando `bash, edit, write, multiedit, patch` sao informados como obrigatorios — prova que o gate ativado nao quebra o default real hoje em producao (defaults finos, nunca deny total).
  - `TestInstall_OpenCode_FailsWhenDefaultPermissionWildcardDeniesRequiredTool` (`internal/install/install_opencode_test.go`): sobrescreve `openCodePermissionFactory` para retornar `{"edit": "deny"}`, executa `svc.Execute` com `Tools: [ToolOpenCode]` e confirma que o erro retornado satisfaz `errors.Is(err, specs.ErrOpenCodePermissionWildcardDeny)` e que `opencode.json` NAO e escrito — prova o caminho real de producao via `install.go`, nao apenas o teste isolado que ja existia em `opencode_config_test.go`.
- Validacao: `go build ./...`, `go vet ./...`, `go test ./internal/install/... -count=1 -v` (75 passed), `go test ./internal/runtime/specs/... -count=1 -v` (114 passed), `go test ./... -count=1` (2901 passed em 75 pacotes).

### BUG-007
- ID: BUG-007
- Severidade: minor (mas viola regra `hard` R-STYLE-001.3 — mandato desta rodada exige 0 ressalvas)
- Origem: R-STYLE-001.3 (`.claude/rules/code-style.md`), achado recorrente nas revisoes adversariais das tarefas 6.0-7.0 e 10.0
- Estado: fixed
- Causa raiz: `_orchestratorHooks` (linha 889 original) e `_agentsScriptsFiles` (linha 901 original) mantinham prefixo `_` em identificador Go de pacote — proibido por R-STYLE-001.3 ("Go nao usa `_` como prefixo de identificador... vale tambem para nomes que hoje existem no repo com esse prefixo: renomear ao tocar"). Confirmado via `git blame` que ambas as regioes foram efetivamente tocadas por este trabalho (linha do comentario acima de `_orchestratorHooks` e a entrada `validate-session-end.sh` dentro de `_agentsScriptsFiles` apareciam como "Not Committed Yet" antes desta correcao).
- Arquivos alterados:
  - `internal/install/install.go`: renomeado `_orchestratorHooks` -> `orchestratorHooks` e `_agentsScriptsFiles` -> `agentsScriptsFiles` na declaracao (linhas ~882, ~889) e em todos os usos (`copyAgentsScripts`, `copyToolValidationHooks`/hooks loop, `expectedInstalledPaths`, total de 5 pontos). `grep -rn "_orchestratorHooks|_agentsScriptsFiles" internal/install/` confirmou apenas esses 5 pontos, todos corrigidos; nenhuma outra referencia no pacote `install` ou no restante do repositorio.
  - Comentarios em PT-BR presentes nos blocos efetivamente tocados por este trabalho (acima das duas declaracoes) foram removidos ao tocar as linhas, em conformidade com R-STYLE-001.2 (zero comentarios em codigo criado ou editado) — regra hard que precede e absorve a exigencia de traducao de R-STYLE-001.1 para este caso, pois remover o comentario elimina tanto a violacao de idioma quanto a de presenca de comentario. Nenhum outro comentario PT-BR fora dessas linhas tocadas foi alterado (varredura restrita ao diff, conforme mandado pela skill).
- Teste de regressao: nao aplicavel — troca de nome de identificador privado ao pacote, sem mudanca de comportamento observavel. A cobertura existente de `internal/install/install_opencode_test.go` (`TestInstall_OpenCode_ShipsCanonicalValidators`) e da suite completa de `internal/install/...` (75 testes) já exercita os call-sites renomeados e confirma que nenhuma referencia ficou quebrada.
- Validacao: `go build ./...` (sem erros), `go vet ./...` (sem erros), `go test ./internal/install/... -count=1 -v` (75 passed), `go test ./... -count=1` (2901 passed em 75 pacotes).

## Comandos Executados
- `go build ./...` -> sem erros
- `go vet ./...` -> sem erros
- `go test ./internal/install/... -count=1 -v` -> 75 passed
- `go test ./internal/runtime/specs/... -count=1 -v` -> 114 passed
- `go test ./... -count=1` -> 2901 passed em 75 pacotes
- `grep -rn "_orchestratorHooks\|_agentsScriptsFiles" internal/` -> nenhuma ocorrencia remanescente (confirmado apos rename)
- `git blame -L 880,930 internal/install/install.go` -> confirmou quais linhas dos blocos BUG-007 foram tocadas nesta sessao (task 8.0/10.0) antes da correcao

## Riscos Residuais
- `openCodePermissionFactory` e um seam de teste (`var` sobrescrevivel) introduzido exclusivamente para permitir o teste de regressao de BUG-002 exercitar o caminho real de producao sem lookup de PATH real nem side effects de disco fora do FakeFileSystem. Nao e exposto via API publica nem documentado como ponto de extensao — segue o padrao ja usado no arquivo para injecao de dependencias em testes (`lookPather`, `agentDetect`, `langDetect`).
- A lista `openCodeRequiredTools` duplica textualmente `MUTATING_TOOLS` de `internal/embedded/assets/.opencode/plugin/governance.js` (JS, fora do grafo de compilacao Go). Nao ha mecanismo automatico de deteccao de drift entre as duas listas; um teste futuro poderia comparar as duas fontes se o script JS ganhar um parser Go dedicado. Risco baixo hoje pois ambas as listas sao pequenas e estaveis (ferramentas mutantes canonicas do harness).
