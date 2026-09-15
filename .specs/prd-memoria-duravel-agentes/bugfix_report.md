# Relatorio de Bugfix

- Total de bugs no escopo: 4
- Corrigidos: 4
- Testes de regressao adicionados: 4
- Pendentes: nenhum
- Estado final: done

## Bugs

- ID: BUG-WIN-01
- Severidade: major
- Origem: RF-01 (`internal/runtime/memory/durable/scope_test.go`, `TestActivePathResolvesForPRDLayer`)
- Estado: fixed
- Causa raiz: o teste comparava `Scope.ActivePath()` contra um literal hardcoded com barra normal (`"/project/.specs/prd-x/memory/MEMORY.md"`). A implementacao (`internal/runtime/memory/durable/layer.go:82`, `activePath` -> `filepath.Join`) usa o separador nativo do SO. `filepath.Clean`/`Join` do Go normalizam **qualquer** `/` de entrada para o separador nativo tambem no Windows (`internal/filepathlite.IsPathSeparator` reconhece `/` e `\` e `Clean` reemite sempre `Separator`), entao em Windows o resultado real e `\project\.specs\prd-x\memory\MEMORY.md` — exatamente o valor visto no log do CI — enquanto o teste esperava barra normal. Bug do teste, nao da implementacao.
- Arquivos alterados: `internal/runtime/memory/durable/scope_test.go`
- Teste de regressao: o proprio `TestActivePathResolvesForPRDLayer`, agora comparando contra `filepath.Join(scope.TasksDir, "memory", "MEMORY.md")` em vez de literal.
- Validacao: `go test ./internal/runtime/memory/durable/... -run TestScopeSuite -count=1` (darwin, verde); `GOOS=windows GOARCH=amd64 go vet ./internal/runtime/memory/durable` e `GOOS=windows GOARCH=amd64 go test -c` (compila).
- Confianca de correcao real no Windows: **alta**. Verifiquei o codigo-fonte de `path/filepath`/`internal/filepathlite` do toolchain instalado (Go 1.27.1) e confirmei que `Clean`/`Join` no Windows tratam `/` como separador valido e reemitem sempre `\` — o mesmo padrao do literal `\project\...` no log do CI. Nao ha ambiguidade de comportamento aqui.

- ID: BUG-WIN-02
- Severidade: major
- Origem: RF-16 (`internal/runtime/memory/durable/layer_test.go`, `TestConsolidateRefusesWriteThroughExternalSymlink`, teste adicionado nesta mesma sessao para BUG-B)
- Estado: fixed
- Causa raiz: o teste construia o link do symlink fake com um literal hardcoded (`"/repo/.specs/prd-x/memory"`, barra normal) enquanto `fs.RefuseExternalSymlink` (`internal/fs/symlink_guard.go:27`) monta os caminhos-ancestrais a verificar via `symlinkAncestors`, que usa `filepath.Join` (separador nativo). Em Windows os ancestrais gerados usam `\`, mas o mapa `FakeFileSystem.Links` guardava a chave crua com `/` (sem normalizacao) porque `Symlink(target, link)` (`internal/fs/fake.go:59`) apenas faz `f.Links[link] = target`, sem `filepath.Clean`. `IsSymlink` (`internal/fs/fake.go:122`) faz `_, ok := f.Links[path]` — comparacao exata de string, sem normalizar. Resultado: em Windows o lookup nunca casava, `IsSymlink` sempre retornava `false`, a guarda nunca disparava, e o `Consolidate` escrevia atraves do symlink externo sem erro — exatamente "An error is expected but got nil" do log.
- Arquivos alterados: `internal/runtime/memory/durable/layer_test.go` (construcao do argumento `link` de `filesystem.Symlink(...)` e das asserções de `Exists` passou a usar `filepath.Join` a partir da mesma `tasksDir`, em vez de literal com barra fixa)
- Teste de regressao: o proprio `TestConsolidateRefusesWriteThroughExternalSymlink`, agora com `filesystem.Symlink("/outside/evil", filepath.Join(tasksDir, "memory"))`.
- Validacao: `go test ./internal/runtime/memory/durable/... -run TestLayerSuite -count=1` (darwin, verde); `GOOS=windows GOARCH=amd64 go vet` e `go test -c` (compila).
- Confianca de correcao real no Windows: **media-alta**. A leitura de codigo confirma que, com a chave do link construida via `filepath.Join` (que normaliza para `\` em Windows, igual aos ancestrais de `symlinkAncestors`), a comparacao exata em `IsSymlink` volta a casar — a mesma logica que ja funciona hoje em Unix (onde `/` e nativo e portanto os dois lados ja coincidiam por acidente). Nao alterei `internal/fs/fake.go` (producao de teste compartilhada por muitos outros testes) porque a causa raiz observavel esta no teste: ele hardcodava um separador que nao corresponde ao produzido pela implementacao no SO alvo. Risco residual: se algum outro teste em outro arquivo tambem chamar `FakeFileSystem.Symlink`/`IsSymlink` com literais de barra fixa, o mesmo bug pode recorrer sob Windows sem que eu tenha auditado exaustivamente todos os chamadores (busquei apenas os quatro achados deste escopo).

- ID: BUG-WIN-03
- Severidade: major
- Origem: RF-16 (`internal/runtime/memory/durable/layer_test.go`, `TestConsolidateSurfacesLockContentionWithoutCorruptingPage`)
- Estado: fixed
- Causa raiz: mesma classe de defeito do BUG-WIN-02, mas no locker de teste. `lockPath` era um literal hardcoded (`"/repo/.specs/prd-x/memory/MEMORY.lock"`, barra normal) usado como chave em `locker.refuseOn[lockPath] = true`. Em runtime, `layer.Consolidate` obtem o lock path via `scope.lockPath()` -> `sidecarPath` -> `filepath.Join` (separador nativo). Em Windows o `lockPath` real e `\repo\.specs\prd-x\memory\MEMORY.lock`, que nao bate com a chave `/repo/...` inserida em `refuseOn` por `inMemoryLayerLocker.Lock` (`internal/runtime/memory/durable/layer_test.go:29`, tambem comparacao exata de string). O lock nunca era recusado, `Consolidate` seguia e escrevia a pagina normalmente — exatamente as duas asserções falhando no log (`errors.Is(err, ErrLayerLocked)` false e `Exists(...)` true).
- Arquivos alterados: `internal/runtime/memory/durable/layer_test.go` (`lockPath` construido via `filepath.Join(tasksDir, "memory", "MEMORY.lock")`, `tasksDir` extraida para variavel compartilhada com o `scope`)
- Teste de regressao: o proprio `TestConsolidateSurfacesLockContentionWithoutCorruptingPage`.
- Validacao: `go test ./internal/runtime/memory/durable/... -run TestLayerSuite -count=1` (darwin, verde); `GOOS=windows GOARCH=amd64 go vet`/`go test -c` (compila).
- Confianca de correcao real no Windows: **alta**. `scope.lockPath()` e o `lockPath` do teste passam a ser construidos pela mesma funcao (`filepath.Join`) a partir do mesmo `tasksDir`, entao produzem literalmente a mesma string em qualquer SO — nao ha mais dependencia de qual separador o codigo de producao escolhe.

- ID: BUG-WIN-04
- Severidade: minor
- Origem: RF-20 (`internal/runtime/memory/durable/facade_test.go`, `TestBuildContextReportsRecoveryLatencyDegradation`)
- Estado: fixed
- Causa raiz: o teste configurava `RecoveryLatencyLimit: time.Nanosecond` e esperava que `BuildContext` sempre demorasse >= 1ns de tempo de parede (`facade.go:187-198`, `withRecoveryLatency` compara `elapsed := time.Since(start)` contra o limite). Isso nao e um bug de path/SO como os tres anteriores: e uma comparacao de timing de escala nanossegundo contra um corpo de funcao muito barato (leituras em `FakeFileSystem` puramente em memoria, sem I/O real), que pode legitimamente medir `elapsed == 0` dependendo da granularidade e do overhead de chamada do relogio monotonico do runtime Go na maquina/SO que executa o teste. Nao consigo provar com certeza absoluta qual e o mecanismo exato no runner Windows do CI (sem acesso a um runner Windows real), mas a natureza do teste — limite de 1ns contra trabalho em memoria — e inerentemente fragil em qualquer plataforma, e o log mostra exatamente a falha esperada desse desenho (`RecoveryDegraded` ficou `false`).
- Arquivos alterados: `internal/runtime/memory/durable/facade_test.go` (novo tipo de teste `slowReadLayer`, um decorator que embute `durable.Layer` e sobrescreve `Read` para dormir `50ms` antes de delegar; `RecoveryLatencyLimit` ajustado de `time.Nanosecond` para `5 * time.Millisecond`)
- Teste de regressao: o proprio `TestBuildContextReportsRecoveryLatencyDegradation`, agora forcando uma demora real e generosa (50ms, muito acima da granularidade historica mais grosseira ja documentada em timers Windows, ~15.6ms) contra um limite de 5ms, tornando o resultado determinístico independente da resolucao do relogio da plataforma.
- Validacao: `go test ./internal/runtime/memory/durable/... -run TestFacadeSuite -count=1` (darwin, verde); `GOOS=windows GOARCH=amd64 go vet`/`go test -c` (compila).
- Confianca de correcao real no Windows: **media**. A leitura de codigo confirma que `time.Sleep` bloqueia por pelo menos a duracao pedida (o SO pode dormir mais, nunca menos), entao com margem de 10x entre delay (50ms) e limite (5ms) o teste deveria passar em qualquer granularidade de timer plausivel. Nao tenho como validar isso rodando de fato em um runner Windows, e reconheco a tensao com a diretriz geral do projeto de evitar "testes frageis baseados em sleep" (`agent-governance/references/testing.md`, secao Proibido): aqui o uso de `Sleep` nao e para sincronizar concorrencia de forma adivinhada, e sim para exercitar deterministicamente um comportamento que a propria feature (RF-20, degradacao de latencia) define em termos de tempo real decorrido — a alternativa (manter um limiar de nanossegundos contra trabalho em memoria) e o que efetivamente já causou a flakiness observada. Registro essa tensao como suposicao assumida, nao como certeza.

## Comandos Executados
- `go build ./...` -> OK, sem output
- `go vet ./...` -> OK, sem output
- `go test ./... -count=1` -> `Go test: 3806 passed in 78 packages`, exit 0 (suite completa, darwin, sem regressao)
- `go test ./internal/runtime/memory/durable/... -run 'TestScopeSuite|TestLayerSuite|TestFacadeSuite' -count=1` -> `52 passed`, exit 0
- `GOOS=windows GOARCH=amd64 go vet ./internal/runtime/memory/durable/...` -> OK, sem output
- `GOOS=windows GOARCH=amd64 go build ./internal/runtime/memory/durable/...` -> OK, sem output
- `GOOS=windows GOARCH=amd64 go test -c -o durable_windows_test.exe ./internal/runtime/memory/durable` -> compilou com sucesso (binario de 15.5MB gerado, depois removido do scratchpad)
- `gofmt -l` nos 3 arquivos alterados -> sem saida (formatados)

## Riscos Residuais
- BUG-WIN-02: nao auditei todos os demais usos de `FakeFileSystem.Symlink`/`IsSymlink` no repositorio alem dos quatro achados deste escopo; se outro teste em outro arquivo tambem hardcodar separador fixo ao chamar `Symlink`, o mesmo padrao de falha pode recorrer sob Windows. Correcao aplicada apenas no teste que falhou no CI, nao em `internal/fs/fake.go` (decisao deliberada: menor mudanca segura, ver causa raiz do BUG-WIN-02).
- BUG-WIN-04: sem acesso a um runner Windows real, a confianca na correcao e "media", nao "alta". A margem de 10x entre o delay artificial (50ms) e o limite (5ms) foi escolhida para superar com folga a pior granularidade de timer historicamente documentada em Windows (~15.6ms), mas isso e inferencia sobre documentacao publica do Go/Windows, nao observacao direta neste ambiente.
- Nenhum dos quatro achados exigiu mudanca de comportamento publico (todas as correcoes ficaram confinadas a arquivos `_test.go`).
