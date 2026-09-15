# Tarefa 3.0: Quatro políticas stateless — relevância, orçamento, sanitização e compactação

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar as quatro políticas determinísticas do subsistema como structs stateless com métodos, **sem interface**, no idioma que o pacote já adota em `internal/runtime/memory/window_policy.go:37`. Cada uma tem implementação única e nenhuma segunda variante plausível — transformá-las em interface criaria abstração vazia.

A política de lease **não** entra aqui: é estado persistido com arquivo, prazo, referência de processo e build tags, e vive em 4.0.

<requirements>
- RF-17: seleção por relevância à task ativa, não injeção integral.
- RF-18: teto global derivado da `WindowClass` com cotas por camada e cessão de sobra.
- RF-15: sanitização com catálogo mínimo obrigatório; redação de trecho registrada; recusa quando não isolável.
- RF-13: compactação determinística que decide o conjunto a arquivar, preservando bloco humano.
- Resultado reproduzível para a mesma entrada: nenhuma política pode depender de tempo, mapa não ordenado ou aleatoriedade.
</requirements>

## Subtarefas

- [x] 3.1 Política de relevância: ordenação determinística a partir de camada, durabilidade, chave, task ativa e marcação de contradição.
- [x] 3.2 Política de orçamento: teto por `specs.WindowClass`, cotas por camada (projeto 50%, PRD 30%, task 20%) e cessão de sobra.
- [x] 3.3 Política de sanitização com o catálogo mínimo de RF-15: chave privada PEM, tokens com prefixo reconhecível, JWT, cabeçalho de autorização, valores de `.env` e string de conexão com credencial.
- [x] 3.4 Extensibilidade do catálogo por configuração.
- [x] 3.5 Política de compactação: conjunto determinístico a arquivar, preservando bloco humano e reportando limite inalcançável.
- [x] 3.6 Instância compartilhada exportada por política, com receiver por valor, seguindo `window_policy.go:20`.

## Detalhes de Implementação

Ver techspec.md, seção "Interfaces Chave" (bloco `PoliticaOrcamento`) e "Abordagem de Testes". Ver MD-001, seção Decisão, parágrafo sobre políticas sem interface.

## Critérios de Sucesso

- Nenhuma política virou interface sem segunda implementação real.
- Mesma entrada produz mesma saída, provado por teste que executa duas vezes e compara.
- Cotas respeitadas, sobra cedida, e omissões declaradas ao atingir o teto.
- Cada padrão do catálogo mínimo tem teste próprio, positivo e negativo.
- Trecho sensível não isolável resulta em recusa com erro tipado, sem persistir nada.
- Compactação preserva bloco humano intacto e reporta limite inalcançável em vez de silenciá-lo.
- Nenhuma política usa tempo, aleatoriedade ou iteração de mapa sem ordenação.

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
- [x] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- Quatro arquivos de política no pacote criado em 2.0 — ver techspec.md, "Visão Geral dos Componentes"
- `internal/runtime/memory/window_policy.go` — molde obrigatório de domain service stateless
- `internal/runtime/specs/window.go` — `WindowClass`, apenas leitura
- `internal/metrics/metrics.go` — `ToolBudgets` e `ToolBudgetsLarge`, apenas leitura
