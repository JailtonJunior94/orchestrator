# Hooks Live

## Esta camada NAO e gate de merge

A camada 3 (`hooks_live`) **nao e gate de merge**. Ela roda como job nightly no ambiente protegido
`hooks-live` (`.github/workflows/hooks-live.yml`), fora do gate de PR (`.github/workflows/test.yml`),
e e protegida pela build tag `hooks_live` — `make test` e `go test ./...` nao a compilam. A mesma
declaracao consta no alvo `test-hooks-live` do `Makefile` e no workflow; ela e repetida aqui porque a
techspec exige a declaracao explicita tambem no codigo do pacote (R-STYLE-001.2 proibe comentarios em
codigo, portanto a declaracao vive neste README).

Consequencia pratica: uma celula vermelha aqui **nao bloqueia PR**. Por isso nenhuma celula pode ser
pulada ou ter a assercao afrouxada para ficar verde — uma celula que nao pode ser provada nesta
maquina deve **falhar com diagnostico preciso**, que e o unico valor que esta camada entrega.


Esta camada executa os quatro CLIs reais contra um projeto temporario instalado pelo harness. Ela nao
interpreta comandos de hook com `bash -c`, nao carrega plugin manualmente e nao produz `DispatchProof`.
Cada célula comprova o contrato nativo do seu evento: pré-tool não cria a mutação Go bloqueada **e** emite
o diagnóstico do validador canônico, pós-tool permite a mutação e emite o diagnóstico do validador, e
session-end emite o diagnóstico de tarefa ativa sem `APPROVED`. Ausência de binário, falha de autenticação
ou ausência desse efeito falha o teste.

Requisitos locais:

- `AISPEC_HOOKS_LIVE=1`.
- `claude`, `codex`, `copilot` e `opencode` no `PATH` e autenticados.
- As sessões preservam `HOME` e `CODEX_HOME` do perfil autenticado. Para Codex, o teste lê os hashes
  correntes por `hooks/list`, cria um perfil `hooks-live-*.config.toml` temporário no diretório já usado
  pelo CLI e o passa por `codex exec --profile`; o perfil contém somente
  `projects.<path>.trust_level = "trusted"` e `[hooks.state]`, sendo removido por `t.Cleanup`. Para
  Copilot, o fixture materializa `.github/settings.json` com as definições de `hooks` instaladas e usa
  `--add-dir <fixture>` para acesso a arquivos mais o opt-in documentado
  `GITHUB_COPILOT_PROMPT_MODE_REPO_HOOKS=true` para carregar os hooks de repositório (ver seção do
  contrato nativo do Copilot abaixo). Nenhum segredo é lido, copiado, gravado ou logado pelo teste.
- O Copilot roda com `--stream off --output-format json`, evitando o limite de 90 segundos sem saída
  observável.
- Provider/modelo configurado para OpenCode.

As invocações não interativas usam somente permissões documentadas e delimitadas ao fixture: Claude
`--permission-mode acceptEdits`, Codex `--approve-for-me` (que seleciona `workspace-write`), Copilot
`--add-dir <fixture> --allow-all-tools` e OpenCode `--auto`. O teste não aceita negações de sandbox ou
permissão como evidência: cada célula ainda precisa produzir o efeito e o diagnóstico do hook nativo.

O workflow noturno instala os quatro CLIs e exige `ANTHROPIC_API_KEY`, `OPENAI_API_KEY` e `GH_TOKEN`.
O estado de trust do Codex é derivado e gravado pelo próprio teste em um perfil temporário; não há
credencial persistida pelo teste. O fixture efêmero é inicializado como repositório Git.

O job usa o environment protegido `hooks-live`. Os três segredos devem ser configurados nele; a
ausência de qualquer um falha antes de chamar uma CLI e, portanto, não degrada para evidência sintética.

Verificacao:

```bash
AISPEC_HOOKS_LIVE=1 make test-hooks-live
```

## Evidencia positiva de disparo (nao inferir negacao de ausencia de mutacao)

A assercao pre-ferramenta exige **evidencia positiva** de que o hook nativo disparou: o diagnostico do
validador canonico precisa aparecer na saida da CLI. Arquivo intacto **nao** e prova de negacao — e
igualmente consistente com uma edicao que o modelo nunca tentou (quota esgotada, erro de provider,
recusa do modelo). Os tres estados sao diagnosticados separadamente:

- `nunca dispatchou` — alvo intacto e nenhum diagnostico: a celula nao prova nada.
- `nunca dispatchou e mutou` — nenhum gate entre o agente e o arquivo.
- `dispatchou e mutou` — hook rodou mas nao negou.

As strings esperadas vivem em `diagnostics.go` e sao amarradas ao script que as emite pelo guard
`TestCanonicalDiagnosticsStayInSyncWithTheirEmittingScript` (roda em `go test ./...`, sem build tag).
Se o texto do validador canonico mudar, o guard falha em vez de as assercoes live silenciosamente
pararem de provar disparo.

## Contrato nativo do Codex — trust precede a listagem

Verificado empiricamente contra `codex` 0.154.0 via `codex app-server` + RPC `hooks/list` em fixtures
controlados:

- Hooks de projeto sao lidos **tanto** de `[[hooks.*]]` em `.codex/config.toml` **quanto** de
  `.codex/hooks.json`; ambos retornam `source: "project"`. Declarar os dois no mesmo projeto carrega
  os dois e emite o warning `loading hooks from both ...; prefer a single representation for this layer`.
- Enquanto o projeto **nao esta confiado**, `hooks/list` nao expoe nenhum hook de projeto:
  `Project-local config, hooks, and exec policies are disabled ... until the project is trusted`.

Por isso o setup e em duas fases: primeiro confia-se o projeto em um `CODEX_HOME` temporario e isolado
(somente `projects.<path>.trust_level`), le-se os hashes correntes por `hooks/list`, e so entao grava-se
o perfil `hooks-live-*.config.toml` com `projects` + `[hooks.state]` no `CODEX_HOME` autenticado. O
`CODEX_HOME` temporario nao contem credencial alguma.

## Contrato nativo do Copilot — `--add-dir` nao basta

Verificado empiricamente contra `copilot` 1.0.83: a condicao que carrega hooks de repositorio em modo
prompt e `COPILOT_ALLOW_ALL=true || GITHUB_COPILOT_PROMPT_MODE_REPO_HOOKS=true || folderTrustIsTrusted(cwd)`.

Dois fixtures identicos (mesmo `.github/settings.json`, mesma flag `--add-dir`) produzem resultados
diferentes: o fixture sob `/Users/<user>` (coberto por `trustedFolders`) emite no log
`Loading repo hooks in prompt mode (folder is trusted or opt-in set)`; o fixture sob `TMPDIR` nao emite
nada. Ou seja, `--add-dir` concede acesso a arquivos e a `.github/agents`, **nao** confianca de pasta.

Como o fixture e efemero e vive em `TMPDIR`, e como o PRD proibe alterar a configuracao do usuario, o
teste usa o opt-in por variavel de ambiente `GITHUB_COPILOT_PROMPT_MODE_REPO_HOOKS=true`, escopado ao
processo filho. Isso carrega os hooks do repositorio sem tocar `~/.copilot/config.json`.

## Limites ambientais conhecidos

Estes limites fazem celulas falharem por causa externa ao harness. A celula **falha** (nunca e pulada),
e o diagnostico distingue "hook nao disparou" de "hook disparou e nao bloqueou".

- **Copilot — quota esgotada.** `statusCode: 402`, `{"message":"You have exceeded your monthly quota","code":"quota_exceeded"}`,
  `premium_interactions: 1500/1500`. O modelo nunca chega a chamar uma ferramenta, portanto nenhum hook
  pode disparar, mesmo com os hooks de repositorio carregados. Reseta em `resetDate` da conta.
- **OpenCode — erro de provider.** `{"name":"UnknownError","data":{"message":"Unexpected server error. Check server logs for details."}}`
  (HTTP 500 do provider do modelo configurado). Nao e falha de hook.
