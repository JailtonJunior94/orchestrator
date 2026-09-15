# Formato de Página da Memória Durável

Este documento descreve o formato de arquivo Markdown usado pela memória durável de agentes
(`internal/runtime/memory/durable`, pacote de domínio; ADR [MD-002](../.specs/prd-memoria-duravel-agentes/adr-002-fato-pagina-roundtrip-lossless.md)).
O objetivo é permitir edição manual segura em editor de texto, Obsidian ou `grep`, sem quebrar a
leitura pelo harness e sem perder conteúdo escrito por uma pessoa.

## Estrutura geral

Uma Página é um arquivo `.md` com três partes, nesta ordem quando gerada pelo harness:

1. **Frontmatter da página** — bloco YAML entre `---` no início do arquivo.
2. **Conteúdo de autoria humana** — qualquer texto Markdown livre, escrito por uma pessoa.
3. **Seções de Fato** — cada uma iniciada por um cabeçalho `### Fact: <chave>` seguido de um
   bloco de código cercado com a etiqueta `fact-metadata`.

```markdown
---
identity: minha-pagina
layer: task
origin_session: sess-123
date: "2026-09-11T10:00:00Z"
format_version: 1
---

Anotações livres da pessoa que edita este arquivo. Este texto nunca é reescrito
pelo harness — apenas preservado.

### Fact: payment.retry
```fact-metadata
hash: sha256:...
durability: durable
state: active
origin:
  session: sess-123
  cli: claude
  task: "2.0"
  date: "2026-09-11T10:00:00Z"
content: |
  O pagamento é reprocessado até três vezes antes de falhar definitivamente.
links: []
```
```

## Regras de edição manual

- **Não editar dentro do bloco cercado por ` ```fact-metadata ... ``` `.** Esse bloco é dado
  estruturado (YAML) e é a única fonte de verdade sobre identidade, durabilidade, estado e
  origem de um Fato. Editar manualmente é seguro apenas para ajustar o texto em `content:`
  (bloco de escalar YAML), respeitando a indentação.
- **Texto fora das seções `### Fact: ...` é seu.** Qualquer parágrafo, lista, tabela ou nota
  escrita fora de uma seção de Fato é tratada como conteúdo de autoria humana (RF-37): é
  preservado byte a byte, contado no orçamento de contexto, e sinalizado para compactação
  humana quando a página crescer demais — nunca reescrito ou apagado automaticamente.
- **O cabeçalho `### Fact: <chave>` precisa vir imediatamente seguido da cerca
  ` ```fact-metadata `**, sem linha em branco entre os dois. Se essa sequência exata não for
  encontrada, o harness trata o texto como conteúdo humano comum — o que é seguro (nada é
  perdido), mas o bloco deixa de ser reconhecido como Fato até a formatação ser corrigida.
- **Não duplicar `### Fact: ` fora de uma seção real.** Um cabeçalho `### Fact: algo` sem a
  cerca `fact-metadata` na linha seguinte não quebra nada, mas também não vira um Fato — fica
  como texto comum.
- **YAML malformado dentro de `fact-metadata` (ou no frontmatter da página) isola a página**:
  o harness recusa a leitura com um erro tipado (`ErrPageUnreadable`) em vez de adivinhar ou
  corromper o conteúdo. Corrija a sintaxe YAML e a página volta a ser lida normalmente.

## Campos do frontmatter da página

| Campo | Significado |
|---|---|
| `identity` | Identificador da página (RF-05) |
| `layer` | Camada de memória: `task`, `prd` ou `project` |
| `origin_session` | Sessão que originou ou gravou por último a página |
| `date` | Data/hora da última escrita, formato RFC 3339 |
| `format_version` | Versão do formato de página (atualmente `1`) |

## Campos de um Fato (`fact-metadata`)

| Campo | Significado |
|---|---|
| `hash` | `sha256:<hex>` do conteúdo normalizado do Fato — parte da identidade (RF-12) |
| `durability` | `ephemeral`, `prd` ou `durable` — enum fechado; ausente/vazio é inválido (RF-07) |
| `state` | `proposed`, `active`, `contradicted`, `promoted` ou `archived` |
| `origin` | `session`, `cli`, `task`, `date` — rastreabilidade até a sessão de origem (RF-31, RF-32) |
| `content` | Texto do Fato, em escalar de bloco YAML (`\|`) |
| `links` | Lista de ligações tipadas: `replaces`, `causes`, `fixes` ou `contradicts` (RF-06) |

A chave semântica (`### Fact: <chave>`) fica no cabeçalho da seção, não dentro do bloco YAML —
é o identificador declarado sobre o que o Fato afirma algo (RF-12).

## Garantia de round-trip

A leitura (`Parse`) e a escrita (`Serialize`) de uma Página são funções puras sobre `[]byte`.
Toda escrita gerada pelo harness reexecuta a própria leitura sobre a saída produzida e recusa a
operação com `ErrRoundTripNotPreserved` se o conteúdo de autoria humana não bater byte a byte
com o que foi lido — essa verificação é interna e acontece antes de qualquer gravação em disco.
Isso significa, na prática: **editar uma página à mão e rodar uma sessão em seguida nunca altera
um único byte do que a pessoa escreveu fora das seções de Fato.**

Uma limitação deliberada: se conteúdo humano estiver fisicamente intercalado entre duas seções
de Fato, `Serialize` recusa a escrita com `ErrHumanBlockInterleaved` em vez de reagrupar o
conteúdo humano em um único bloco. O harness nunca reposiciona byte de autoria humana sem
autorização explícita — a página precisa ser reformatada manualmente (movendo o texto para antes
da primeira seção de Fato) antes de aceitar nova consolidação.

Outra normalização exigida: quando há Fatos a anexar, o conteúdo humano precisa já terminar com
uma quebra de linha. Se não terminar, `Serialize` recusa com `ErrHumanContentNotNormalized` em
vez de acrescentar a quebra de linha silenciosamente — a decisão de normalizar cabe a quem grava
a página, nunca ao domínio por trás da verificação de round-trip.
