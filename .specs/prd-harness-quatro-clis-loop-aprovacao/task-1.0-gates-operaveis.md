# Tarefa 1.0: Tornar os gates capazes de rodar e de dizer a verdade

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Esta é a única aresta sem predecessora do grafo. Enquanto ela não existir, as demais tarefas são
validadas por gates que ou não executam, ou executam mentindo, ou bloqueiam ativamente a entrega:

1. O gate de referências de caminho (`scripts/check-spec-paths.sh:56`) usa `mapfile`, builtin de
   bash 4+. O macOS entrega bash 3.2 em `/bin/bash`, onde `mapfile: command not found`. Como o script
   roda sob `set -uo pipefail` **sem `-e`**, o erro não aborta: `prd_dirs` fica vazio, o script cai no
   ramo "Nenhum PRD sob gestao SDD encontrado" e **sai 0**. RF-63 é vazio até que isso seja corrigido —
   o gate não reprova nada porque não chega a verificar nada.
2. O mesmo defeito de portabilidade existe em `.agents/skills/object-calisthenics-go/scripts/list-go-files.sh:9`
   e nos seus 3 espelhos. Ali o script usa `set -euo pipefail`, então aborta ruidosamente — não é falha
   silenciosa, mas quebra a skill inteira no macOS.
3. A notação `(planejado)` que a techspec promete na seção "Arquivos Relevantes e Dependentes" **não tem
   implementação**: `grep planejado scripts/check-spec-paths.sh` não retorna nada. Consequência direta:
   assim que o gate voltar a rodar (item 1), ele reprovará a própria techspec por citar
   `internal/approval`, `internal/approval/mocks` e `internal/approval/portas.go`, que são caminhos
   deliberadamente futuros.
4. `cmd/ai_spec_harness/cli_contract_test.go:181-184` contém um bloco literal que **falha se o esquema
   do CLI não contiver a string `gemini`**. É a aresta que bloqueia toda a entrega: qualquer commit que
   remova o agente descontinuado quebra o build.

Os quatro itens são a mesma fatia porque tocam o mesmo par de arquivos de gate e o mesmo teste de gate,
e porque nenhum deles tem valor isolado: consertar a execução sem a notação `(planejado)` produz um gate
vermelho por design, e implementar a notação sem consertar a execução produz código morto.

<requirements>
- RF-08: o gate de contrato de CI que exige a string do agente removido é desarmado **antes de qualquer
  outra alteração**, sob pena de bloquear a própria entrega.
- RF-63: o gate que reprova artefato de contrato citando caminho inexistente permanece verde após a
  entrega — o que exige, primeiro, que ele efetivamente execute.
- A substituição de `mapfile` deve preservar semântica byte-idêntica de conteúdo do array, inclusive
  para caminhos com espaço, e não pode introduzir dependência de bash 4+ por outra via (`readarray`,
  `declare -A`, `${var^^}`, `&>>`).
- O desarme do gate de contrato **não** pode virar ausência de verificação: o laço remanescente sobre
  `_runtimeACPCatalog` passa trivialmente com catálogo vazio, então precisa de guarda de vacuidade.
- A direção inversa do gate (nenhum agente aposentado pode aparecer no schema) **não** é ligada aqui —
  pertence à etapa 10 da ordem de build da Fase 4 descrita na techspec.
- Os 4 espelhos de `list-go-files.sh` precisam permanecer sincronizados; divergência entre eles é falha
  de gate própria (`make check-skills-sync check-scripts-sync`).
- Sem alteração de comportamento observável dos gates além da correção: um artefato que hoje deveria
  reprovar continua reprovando.
</requirements>

## Subtarefas

- [ ] 1.1 Substituir `mapfile -t prd_dirs < <(...)` em `scripts/check-spec-paths.sh:56` por laço
      `while IFS= read -r linha; do prd_dirs+=("$linha"); done < <(...)`, compatível com bash 3.2.
      Inicializar o array explicitamente antes do laço, porque `set -u` recusa expansão de array não
      inicializado em bash 3.2.
- [ ] 1.2 Aplicar a mesma substituição em `.agents/skills/object-calisthenics-go/scripts/list-go-files.sh:9`
      e propagar aos 3 espelhos: `.claude/skills/object-calisthenics-go/scripts/list-go-files.sh`,
      `.github/skills/object-calisthenics-go/scripts/list-go-files.sh` e
      `internal/embedded/assets/.agents/skills/object-calisthenics-go/scripts/list-go-files.sh`.
- [ ] 1.3 Auditar os demais scripts de gate em busca de outros builtins de bash 4+ (`mapfile`,
      `readarray`, `declare -A`, expansão de caso `${v^^}`/`${v,,}`) e registrar as ocorrências
      encontradas no relatório, corrigindo as que estejam em caminho de gate.
- [ ] 1.4 Implementar a notação `(planejado)` em `scripts/check-spec-paths.sh`: um caminho citado no
      artefato e imediatamente seguido do marcador `(planejado)` é aceito sem resolver no repositório;
      qualquer outro caminho continua obrigatoriamente resolvível.
- [ ] 1.5 Acrescentar caso de teste a `tests/scripts/check-spec-paths_test.sh` cobrindo os três
      resultados: caminho existente passa; caminho inexistente **sem** marcador reprova; caminho
      inexistente **com** `(planejado)` passa.
- [ ] 1.6 Remover o bloco literal de `cmd/ai_spec_harness/cli_contract_test.go:181-184` que exige
      `strings.Contains(schemaContent, "gemini")`.
- [ ] 1.7 Adicionar guarda de vacuidade ao laço `for tool := range _runtimeACPCatalog` no mesmo teste:
      catálogo vazio é falha explícita, não sucesso trivial.
- [ ] 1.8 Executar `make check-spec-paths`, `make check-skills-sync` e `make check-scripts-sync` no
      macOS e capturar a saída como evidência de que o gate agora **executa** e reporta PRDs reais.

## Detalhes de Implementação

Seguir a techspec desta pasta, seção **"Sequenciamento de Desenvolvimento" → fase `F0 — Gates
operáveis`**, que define o escopo exato dos três itens e justifica por que a fase não constava da
versão anterior do documento (os dois defeitos foram descobertos por verificação empírica posterior).

O desarme do gate de contrato é a **etapa 1 da "Ordem de build da Fase 4 (remoção)"** na mesma techspec,
que estabelece literalmente que "o bloco literal é removido e o loop existente ganha guarda de vacuidade"
e que "a direção inversa (nenhum agente aposentado pode aparecer no schema) só é ligada na etapa 10".
Não antecipar a etapa 10 aqui.

A notação `(planejado)` é definida na techspec, seção **"Arquivos Relevantes e Dependentes"**: "Caminhos
ainda não existentes são marcados como `(planejado)` — notação que os distingue de referência a caminho
inexistente por erro, mantendo o gate de referências útil". Os três caminhos que a techspec já cita sob
essa promessa e que hoje o gate acusaria são `internal/approval` (planejado), `internal/approval/mocks`
(planejado) e `internal/approval/portas.go` (planejado).

A dependência dura desta tarefa sobre todas as demais está registrada em `tasks.md`, seção
**"Dependências Críticas"**, primeiro item.

## Critérios de Sucesso

- `bash scripts/check-spec-paths.sh` executado em macOS com bash 3.2 (`/bin/bash --version` registrado
  na evidência) **não** imprime `mapfile: command not found` e **não** imprime
  `Nenhum PRD sob gestao SDD encontrado`; a saída lista os diretórios de PRD efetivamente encontrados.
  Comando de verificação: `bash scripts/check-spec-paths.sh 2>&1 | tee evidence/check-spec-paths.log`.
- `grep -c mapfile scripts/check-spec-paths.sh` retorna `0`.
- `grep -rc mapfile .agents/skills/object-calisthenics-go/scripts/list-go-files.sh .claude/skills/object-calisthenics-go/scripts/list-go-files.sh .github/skills/object-calisthenics-go/scripts/list-go-files.sh internal/embedded/assets/.agents/skills/object-calisthenics-go/scripts/list-go-files.sh`
  retorna `0` para os quatro arquivos.
- `bash .agents/skills/object-calisthenics-go/scripts/list-go-files.sh | wc -l` retorna valor `> 0` em
  macOS, com exit code `0`.
- `grep -c planejado scripts/check-spec-paths.sh` retorna valor `> 0` (hoje retorna `0`).
- `bash tests/scripts/check-spec-paths_test.sh` passa e a sua saída nomeia o caso novo da notação
  `(planejado)`; o caso de caminho inexistente **sem** marcador continua reprovando com exit code `1`.
- `go test ./cmd/ai_spec_harness/ -run TestCLI_Contract -count=1` passa; `grep -n '"gemini"'
  cmd/ai_spec_harness/cli_contract_test.go` não retorna a asserção removida das linhas 181-184.
- O teste de contrato falha quando `_runtimeACPCatalog` está vazio — verificável por execução com
  catálogo esvaziado localmente e registro do output, revertido em seguida.
- `make check-skills-sync check-scripts-sync` passa, provando que os 4 espelhos foram sincronizados.
- `go build ./... && go vet ./... && go test ./... -count=1` verde.

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

Cobertura obrigatória:

- `tests/scripts/check-spec-paths_test.sh`: caso novo para `(planejado)`, mais preservação dos casos
  existentes de caminho válido e de caminho inexistente sem marcador.
- `cmd/ai_spec_harness/cli_contract_test.go`: guarda de vacuidade do catálogo ACP, exercitada.
- Execução real dos gates em macOS (`bash --version` 3.2) como prova de integração — é o único teste que
  distingue "gate corrigido" de "gate que continua saindo 0 sem verificar".
- `make check-skills-sync check-scripts-sync` como gate de sincronia dos 4 espelhos.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `scripts/check-spec-paths.sh:56` — `mapfile` sob `set -uo pipefail` sem `-e`: falha silenciosa que
  faz o gate sair 0 sem verificar nada.
- `tests/scripts/check-spec-paths_test.sh` — recebe o caso de teste da notação `(planejado)`.
- `.agents/skills/object-calisthenics-go/scripts/list-go-files.sh:9` — mesmo `mapfile`, aqui sob
  `set -euo pipefail:2`, que aborta ruidosamente e derruba a skill inteira no macOS.
- `.claude/skills/object-calisthenics-go/scripts/list-go-files.sh` — espelho.
- `.github/skills/object-calisthenics-go/scripts/list-go-files.sh` — espelho.
- `internal/embedded/assets/.agents/skills/object-calisthenics-go/scripts/list-go-files.sh` — espelho.
- `cmd/ai_spec_harness/cli_contract_test.go:181-184` — bloco literal a remover; laço em `:175-179`
  recebe a guarda de vacuidade.
- `docs/cli-schema.json` — artefato lido pelo teste de contrato (não alterado nesta tarefa).
- `Makefile:79-83` — alvo `check-spec-paths`, que encadeia script e teste.
- `scripts/check-skills-sync.sh`, `scripts/check-scripts-sync.sh` — gates de sincronia dos espelhos.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — seções "Sequenciamento de
  Desenvolvimento", "Ordem de build da Fase 4 (remoção)" e "Arquivos Relevantes e Dependentes".
