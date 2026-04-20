# Guia de Resolucao de Problemas — ai-spec-harness

Este guia cobre os problemas mais comuns encontrados por usuarios e agentes ao trabalhar com o `ai-spec-harness`. Cada problema inclui sintoma, causa raiz, solucao passo a passo e comando de verificacao.

---

## Problema: Instalacao falha — binario nao encontrado ou permissao negada

**Sintoma:** Apos instalar via `brew install` ou `go install`, o terminal retorna `command not found: ai-spec` ou `permission denied` ao executar o binario.

**Causa:** O diretorio do Homebrew ou do `$GOPATH/bin` nao esta no `PATH` do shell. Em instalacoes manuais (download direto), o binario pode nao ter permissao de execucao.

**Solucao:**

1. Verifique onde o binario foi instalado:
   ```bash
   which ai-spec || which ai-spec-harness
   ls -la $(brew --prefix)/bin/ai-spec 2>/dev/null
   ls -la $GOPATH/bin/ai-spec-harness 2>/dev/null
   ```
2. Adicione o prefixo do Homebrew ao `PATH` no arquivo de inicializacao do shell:
   ```bash
   # ~/.zshrc ou ~/.bashrc
   export PATH="$(brew --prefix)/bin:$PATH"
   alias ai-spec-harness="ai-spec"
   ```
3. Recarregue o shell:
   ```bash
   source ~/.zshrc   # ou source ~/.bashrc
   ```
4. Para binarios baixados manualmente, conceda permissao de execucao:
   ```bash
   chmod +x ai-spec
   sudo mv ai-spec /usr/local/bin/ai-spec
   ```
5. No macOS, se o Gatekeeper bloquear o binario com aviso de malware, remova a quarentena:
   ```bash
   xattr -dr com.apple.quarantine $(which ai-spec)
   ```

**Verificacao:** `ai-spec version`

---

## Problema: Lint falha com erro de schema

**Sintoma:** `ai-spec lint .` retorna erro como `schema version mismatch`, `schemaVersion invalido` ou `campo obrigatorio ausente` ao validar o manifesto `.ai_spec_harness.json`.

**Causa:** O manifesto foi gerado por uma versao anterior do CLI e nao esta compativel com o `cli-schema.json` atual embarcado no binario. Isso ocorre com frequencia apos um `brew upgrade ai-spec` sem re-executar o `install` ou `upgrade` no projeto alvo.

**Solucao:**

1. Verifique a versao do CLI instalada:
   ```bash
   ai-spec version
   ```
2. Inspecione o manifesto atual do projeto alvo para identificar o campo divergente:
   ```bash
   cat .ai_spec_harness.json
   ```
3. Execute o upgrade para regenerar o manifesto com o schema atual:
   ```bash
   ai-spec upgrade . --source <caminho-para-governanca> --langs go
   ```
4. Se o problema persistir, reinstale a governanca:
   ```bash
   ai-spec install . --source <caminho-para-governanca> --tools claude,gemini,codex,copilot --langs go
   ```

**Verificacao:** `ai-spec lint .`

---

## Problema: Parity check falha inesperadamente

**Sintoma:** Em CI, a verificacao de paridade multi-agente retorna falha ou aviso inesperado. Localmente o resultado pode ser diferente. A mensagem menciona um invariante `BestEffort` como `GEMINI-MD` ou `COPILOT-MD`.

**Causa:** Invariantes classificados como `BestEffort` verificam conformidade procedural que nao tem enforcement automatico (por exemplo, presenca de documentacao opcional em `GEMINI.md` ou `.github/copilot-instructions.md`). Em CI, o projeto alvo pode nao ter esses arquivos, fazendo com que o check emita aviso mesmo sem bloquear o build.

**Solucao:**

1. Identifique quais invariantes falharam e o nivel de enforcement:
   ```bash
   ai-spec inspect .
   ```
2. Invariantes `BestEffort` geram `Warning=true` e nao devem bloquear o CI — verifique se o pipeline esta tratando avisos como erros.
3. Para silenciar avisos legitimos, certifique-se de que os arquivos opcionais existam no projeto alvo:
   ```bash
   # exemplo: garantir presenca do arquivo de instrucoes do Copilot
   ls .github/copilot-instructions.md
   ```
4. Se o arquivo nao for relevante para o projeto, documente a excecao no `AGENTS.md` do repositorio alvo.

**Verificacao:** `ai-spec inspect . 2>&1 | grep -i warning`

---

## Problema: Hash mismatch no skills-lock.json

**Sintoma:** O CLI ou um hook de governanca retorna erro como `hash diverge do registrado em skills-lock.json` ou `skill modificada apos o lock`. O CI bloqueia o merge.

**Causa:** O conteudo de uma skill em `.agents/skills/` foi alterado diretamente sem atualizar o `skills-lock.json`. O lock file rastreia o SHA-256 de cada skill externa; qualquer modificacao sem atualizacao do hash e detectada como adulteracao.

**Solucao:**

1. Identifique qual skill esta com hash divergente:
   ```bash
   cat skills-lock.json
   ```
2. Se a modificacao foi intencional, atualize o hash apos registrar a decisao de upgrade em `audit/`:
   - Crie o registro de decisao usando o template em `tasks/templates/skill-upgrade-decision.md`.
   - Preencha os campos obrigatorios: `skill`, `versao anterior`, `versao nova`, `motivador`, `criterio de aceitacao`, `data`.
3. Atualize o hash no `skills-lock.json` com o SHA-256 recalculado do conteudo atual da skill:
   ```bash
   shasum -a 256 .agents/skills/<nome-da-skill>/SKILL.md
   ```
4. Atualize o campo `computedHash` correspondente no `skills-lock.json`.
5. Se a modificacao foi acidental, restaure o arquivo original:
   ```bash
   git checkout -- .agents/skills/<nome-da-skill>/
   ```

**Verificacao:** `ai-spec lint . && git diff skills-lock.json`

---

## Problema: Telemetria nao registra eventos

**Sintoma:** `ai-spec telemetry summary` nao exibe dados ou mostra zero eventos mesmo apos executar skills. O arquivo de log de telemetria nao e criado.

**Causa:** A variavel de ambiente `GOVERNANCE_TELEMETRY` nao esta definida. A telemetria e opt-in — sem essa variavel, nenhum evento e gravado.

**Solucao:**

1. Exporte a variavel antes de executar o CLI:
   ```bash
   export GOVERNANCE_TELEMETRY=1
   ```
2. Para ativar permanentemente, adicione ao arquivo de inicializacao do shell:
   ```bash
   # ~/.zshrc ou ~/.bashrc
   export GOVERNANCE_TELEMETRY=1
   ```
3. Recarregue o shell e execute um comando para gerar um evento:
   ```bash
   source ~/.zshrc
   GOVERNANCE_TELEMETRY=1 ai-spec telemetry log create-prd
   ```
4. Consulte o resumo de uso:
   ```bash
   ai-spec telemetry summary
   ```

**Verificacao:** `ai-spec telemetry summary` — deve exibir pelo menos um evento registrado.

---

## Problema: Detect nao encontra linguagem do projeto

**Sintoma:** `ai-spec inspect .` ou `ai-spec install` nao detecta a linguagem do projeto (Go, Node, Python). A saida mostra `langs: []` ou a linguagem esperada esta ausente.

**Causa:** O arquivo de manifesto de linguagem (`go.mod`, `package.json`, `requirements.txt`, `pyproject.toml`) nao esta na raiz do diretorio alvo. O detector do pacote `internal/detect` busca esses arquivos apenas no nivel raiz, nao em subdiretorios.

**Solucao:**

1. Verifique se o arquivo de manifesto existe na raiz:
   ```bash
   ls go.mod package.json requirements.txt pyproject.toml 2>/dev/null
   ```
2. Se o projeto usa uma estrutura com subdiretorios (ex: monorepo), passe o subdiretorio correto como alvo:
   ```bash
   ai-spec install ./servico-go --source <governanca> --langs go
   ```
3. Se necessario, force a linguagem explicitamente via flag `--langs`:
   ```bash
   ai-spec install . --source <governanca> --tools claude --langs go,node
   ```
4. Para repositorios multi-linguagem, liste todas as linguagens presentes:
   ```bash
   ai-spec install . --source <governanca> --tools all --langs go,node,python
   ```

**Verificacao:** `ai-spec inspect . | grep -i lang`

---

## Problema: Upgrade nao detecta mudancas

**Sintoma:** `ai-spec upgrade . --source <governanca> --check` retorna `nenhuma mudanca detectada` ou `ja na versao mais recente`, mas o repositorio fonte foi atualizado.

**Causa:** O comando `--check` compara o estado instalado com o estado atual da fonte. Se a instalacao foi feita via `copy` (snapshot fisico), o upgrade so detecta mudancas quando os arquivos copiados diferem dos arquivos da fonte. Se o repositorio fonte nao foi atualizado localmente (git pull pendente), o CLI nao tem como saber sobre mudancas remotas.

**Solucao:**

1. Atualize o repositorio fonte de governanca:
   ```bash
   cd <caminho-para-governanca> && git pull
   ```
2. Execute o check novamente a partir do projeto alvo:
   ```bash
   cd <projeto-alvo>
   ai-spec upgrade . --source <caminho-para-governanca> --check
   ```
3. Se quiser forcado o upgrade mesmo sem diff detectado:
   ```bash
   ai-spec upgrade . --source <caminho-para-governanca> --langs go
   ```
4. Para instalacoes via `symlink`, as mudancas na fonte ja se refletem automaticamente — nao e necessario re-executar o upgrade.

**Verificacao:** `ai-spec inspect . && ai-spec doctor .`

---

## Problema: Cobertura de testes abaixo do threshold no CI

**Sintoma:** O CI falha com mensagem como `cobertura atual X% abaixo do minimo de 75%` ou o job de testes e cancelado com erro de threshold. O badge de cobertura fica vermelho.

**Causa:** Novo codigo foi adicionado sem testes correspondentes, ou testes existentes foram removidos. O threshold minimo enforced no CI e 75% de cobertura total.

**Solucao:**

1. Gere o relatorio de cobertura local para identificar os pacotes com baixa cobertura:
   ```bash
   make coverage
   ```
2. Para ver a cobertura por pacote com threshold de 70%:
   ```bash
   make coverage-packages
   ```
3. Abra o relatorio HTML para inspecionar as linhas nao cobertas:
   ```bash
   go test -coverprofile=coverage.out ./...
   go tool cover -html=coverage.out
   ```
4. Adicione testes para os pacotes abaixo do threshold. Lembre-se das convencoes:
   - Testes unitarios usam `FakeFileSystem` (nunca o OS real)
   - Testes de integracao usam `t.TempDir()` com build tag `integration`
   - Testes devem ser table-driven
5. Valide localmente antes de abrir PR:
   ```bash
   make test && make integration
   ```

**Verificacao:** `make coverage` — a linha `total:` deve indicar >= 75%.

---

## Problema: GoReleaser falha ao executar localmente

**Sintoma:** `goreleaser release --snapshot --clean` retorna erro como `goreleaser: command not found`, `missing required tool`, erro de configuracao em `.goreleaser.yaml`, ou falha ao criar os artefatos de release.

**Causa:** O GoReleaser nao esta instalado localmente, ou a versao instalada e incompativel com a configuracao do projeto. Localmente, o GoReleaser requer que a tag Git exista quando executado sem `--snapshot`.

**Solucao:**

1. Instale o GoReleaser via Homebrew:
   ```bash
   brew install goreleaser/tap/goreleaser
   goreleaser --version
   ```
2. Para testar o release sem publicar e sem precisar de tag:
   ```bash
   goreleaser release --snapshot --clean
   ```
3. Para validar apenas a configuracao sem gerar artefatos:
   ```bash
   goreleaser check
   ```
4. Se quiser simular o pipeline de CI localmente, use o dry-run:
   ```bash
   goreleaser release --snapshot --clean --skip=publish
   ```
5. Verifique se `GITHUB_TOKEN` nao e necessario para o modo snapshot:
   ```bash
   unset GITHUB_TOKEN
   goreleaser release --snapshot --clean
   ```

**Verificacao:** `goreleaser release --snapshot --clean` — deve gerar artefatos em `dist/` sem erros.

---

## Problema: Copilot CLI nao carrega contexto do repositorio

**Sintoma:** O `gh copilot suggest` ignora as instrucoes de governanca do repositorio. As respostas do Copilot CLI nao seguem as skills ou convencoes do projeto.

**Causa:** O `gh copilot` CLI e stateless — ele nao le `.github/copilot-instructions.md` nem qualquer arquivo de contexto automaticamente. Cada invocacao e independente. Apenas o Copilot Chat na extensao VS Code (v1.143+) carrega esse arquivo automaticamente.

**Solucao:**

1. Verifique se o arquivo de instrucoes existe:
   ```bash
   ls .github/copilot-instructions.md
   ```
2. Para o CLI `gh copilot`, injete o contexto manualmente na invocacao:
   ```bash
   gh copilot suggest "$(cat .github/copilot-instructions.md)

   Tarefa: <descricao da tarefa>"
   ```
3. Para fluxos repetitivos, crie um alias ou script que inclua o contexto automaticamente:
   ```bash
   # ~/.zshrc
   alias copilot-with-ctx='gh copilot suggest "$(cat .github/copilot-instructions.md)\n\nTarefa: "'
   ```
4. Use o wrapper do CLI para validar pre-condicoes e emitir a instrucao correta:
   ```bash
   ai-spec wrapper copilot execute-task .
   ```
5. Para o Copilot Chat no VS Code, confirme que o arquivo esta em `.github/copilot-instructions.md` (e nao em outro caminho) e que a extensao esta na versao 1.143 ou superior.

**Verificacao:** `cat .github/copilot-instructions.md && ai-spec wrapper copilot execute-task .`

---

## Problema: Hook de governanca nao executa — permissao negada

**Sintoma:** Ao executar comandos do Claude Code, o hook `.claude/hooks/validate-governance.sh` nao e invocado, ou retorna `permission denied`. O terminal pode exibir `bash: .claude/hooks/validate-governance.sh: Permission denied`.

**Causa:** O script de hook nao tem permissao de execucao. Isso ocorre apos clonar o repositorio em sistemas onde as permissoes de execucao nao sao preservadas, ou apos editar o arquivo em um editor que remove a flag `+x`.

**Solucao:**

1. Verifique as permissoes atuais dos hooks:
   ```bash
   ls -la .claude/hooks/
   ```
2. Conceda permissao de execucao aos scripts de hook:
   ```bash
   chmod +x .claude/hooks/validate-governance.sh
   chmod +x .claude/hooks/validate-preload.sh
   ```
3. Confirme que os scripts sao executaveis:
   ```bash
   ls -la .claude/hooks/
   ```
4. Para evitar que o problema se repita apos operacoes de git, configure o repositorio para preservar permissoes:
   ```bash
   git config core.fileMode true
   ```
5. Adicione os hooks ao git com a permissao correta:
   ```bash
   git add .claude/hooks/validate-governance.sh
   git update-index --chmod=+x .claude/hooks/validate-governance.sh
   ```

**Verificacao:** `.claude/hooks/validate-governance.sh` — deve executar sem erros de permissao.

---

## Problema: Snapshot desatualizado nos testes

**Sintoma:** Testes falham com mensagem como `output diverge do snapshot` ou `snapshot nao encontrado`. O CI bloqueia com diff de snapshot. Localmente, `make test` reporta o arquivo de snapshot com conteudo diferente do esperado.

**Causa:** O comportamento de um comando foi alterado (por exemplo, saida de `ai-spec inspect` ou `ai-spec wrapper`) sem atualizar os arquivos de snapshot em `testdata/`. Os snapshots sao fixtures de saida esperada usados para detectar regressoes de output.

**Solucao:**

1. Identifique quais snapshots estao desatualizados executando os testes com output detalhado:
   ```bash
   go test ./... -v 2>&1 | grep -A5 "diverge do snapshot"
   ```
2. Atualize os snapshots com a variavel de ambiente `UPDATE_SNAPSHOTS=1`:
   ```bash
   UPDATE_SNAPSHOTS=1 go test ./...
   ```
3. Revise os diffs dos arquivos de snapshot atualizados para confirmar que as mudancas sao intencionais:
   ```bash
   git diff testdata/
   ```
4. Se as mudancas forem intencionais (ex: nova feature que altera output), commite os snapshots atualizados junto com o codigo:
   ```bash
   git add testdata/
   git commit -m "test: atualizar snapshots apos mudanca de output em <comando>"
   ```
5. Se as mudancas nao forem esperadas, investigue a regressao antes de atualizar.

**Verificacao:** `make test` — todos os testes devem passar sem mencionar `diverge do snapshot`.
