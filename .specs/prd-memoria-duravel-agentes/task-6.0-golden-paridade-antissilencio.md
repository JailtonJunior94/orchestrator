# Tarefa 6.0: Trava de regressão — golden byte-a-byte e fim da degradação silenciosa

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Instalar a rede de proteção **antes** de existir feature. Duas entregas na mesma fatia porque são a mesma preocupação de produto — nada regride e nada se desliga em silêncio — e porque a correção que toca `internal/runtime/runner.go` entra junto do gate de bytes que protege esse arquivo.

Hoje nada trava os bytes do prompt: os testes existentes usam `strings.Contains` (`internal/runtime/runner_memory_test.go:49`), não igualdade. E hoje uma falha ao gravar memória é **invisível**: o erro de despacho é descartado em `internal/runtime/runner.go:428` e `:221`.

<requirements>
- RF-29: com a feature desativada, o prompt final é byte-idêntico ao produzido pela versão atual.
- RF-30: desativação e degradação registradas explicitamente; desligamento silencioso proibido.
- **Ancoragem obrigatória:** o golden captura o prompt em `hooks.PointPromptPostBuild`, despachado em `internal/runtime/runner.go:290`, cujo evento carrega ponteiro para o prompt final. Ancorar em `injectMemoryContext` ou em `internal/runtime/runner.go:490` é proibido: 7.0 substitui o call site de `:149` e o gate ficaria verde enquanto o prompt real mudasse.
- A correção do descarte de erro registra via log e `Summary`, **não** via a seção de evidência de memória — do contrário cria dependência circular com 8.0.
</requirements>

## Subtarefas

- [x] 6.1 Implementar hook de captura registrado em `hooks.PointPromptPostBuild` para uso exclusivo de teste.
- [x] 6.2 Criar o golden com os quatro casos: ausência total de memória; somente workflow; workflow e task na ordem correta; diretiva de compactação anexada.
- [x] 6.3 Substituir o descarte de erro em `internal/runtime/runner.go:428` (fim de sessão) por registro explícito, sem abortar a sessão.
- [x] 6.4 Substituir o descarte de erro em `internal/runtime/runner.go:221` (pós-revisão) da mesma forma.
- [x] 6.5 Tornar observável o descompasso de tipo em `internal/runtime/hooks/memory_persist.go:57`, mantendo `buildMemoryContent` e o modo de escrita byte-idênticos.

## Detalhes de Implementação

Ver techspec.md, seção "Abordagem de Testes", bloco "Paridade (RF-29)" — a ancoragem está decidida ali. Ver MD-004 (`adr-004-optin-paridade-byte-a-byte.md`), seções Decisão e Plano de Implementação, passos 4 a 6.

## Critérios de Sucesso

- O golden falha se qualquer byte do prompt final mudar, nos quatro casos.
- O golden é ancorado no ponto de ciclo de vida, não em função interna — provado por inspeção do teste.
- Falha de despacho passa a ser registrada e a compor o `Summary`; teste prova que o registro ocorre.
- A sessão **não** aborta quando a gravação de memória falha: comportamento preservado, visibilidade adicionada.
- `buildMemoryContent` produz saída byte-idêntica à atual; os testes de `internal/runtime/hooks/memory_persist_test.go` passam **sem alteração de expectativa**.
- A suíte existente de memória, hooks e runner permanece verde sem alteração de expectativa.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [x] Testes unitários
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/runtime/runner.go` — linhas 221 e 428 (descarte de erro), 290 (ponto de ancoragem)
- `internal/runtime/hooks/memory_persist.go` — linha 57 (asserção de tipo silenciosa)
- `internal/runtime/hooks/dispatcher.go` — `PromptBuildEvent`, apenas leitura
- `internal/runtime/summary.go` — campo para a falha registrada
- `internal/runtime/runner_memory_test.go` e `internal/runtime/hooks/memory_persist_test.go` — devem passar sem alteração
