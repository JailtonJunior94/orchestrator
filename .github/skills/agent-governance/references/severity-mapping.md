# Mapeamento de Severidade (review ↔ bug-schema)

<!-- TL;DR
Tabela canônica que mapeia a severidade de achados da skill `review` (critical/high/medium/low) para a severidade do `bug-schema.json` consumido por `bugfix` (critical/major/minor). Evita perda de informação no handoff review→bugfix.
Keywords: severidade, severity, mapping, review, bugfix, bug-schema, critical, high, medium, low, major, minor
Load complete when: a skill `review` emite bugs acionáveis para `bugfix`, ou `bugfix` consome a lista de bugs e precisa interpretar a severidade.
-->

- Rule ID: R-SEV-001
- Severidade: hard
- Escopo: Handoff `review` → `bugfix` via `bug-schema.json`.

## Objetivo

`review` classifica achados em 4 níveis (`critical`, `high`, `medium`, `low`); `bug-schema.json`
aceita 3 (`critical`, `major`, `minor`). Sem uma tabela canônica, o handoff perde informação ou
diverge entre execuções. Esta referência fixa o mapeamento.

## Tabela canônica

| `review` (4 níveis) | `bug-schema` (3 níveis) | Veredito típico |
|---|---|---|
| `critical` | `critical` | `REJECTED` (bloqueante) |
| `high`     | `major`    | `REJECTED` (bloqueante) |
| `medium`   | `minor`    | `APPROVED_WITH_REMARKS` |
| `low`      | `minor`    | `APPROVED_WITH_REMARKS` |

## Regras

1. `review` atribui a severidade de 4 níveis ao achado e, ao emitir o bug no formato
   `bug-schema.json`, traduz para o nível de 3 usando a tabela acima.
2. `critical` e `high` são bloqueantes (`REJECTED`); `medium` e `low` viram remarks.
3. `medium` e `low` colapsam ambos em `minor` no schema — preservar o nível original de 4 no campo
   de descrição/impacto do bug para não perder granularidade.
4. `bugfix` interpreta `critical`/`major` como correção obrigatória no escopo; `minor` pode ser
   corrigido ou registrado como risco residual conforme o orçamento.

## Anti-padrões

- Emitir bug com severidade fora do enum do schema (`high`/`medium`/`low` cru no JSON).
- Rebaixar `high` para `minor` (perda de bloqueio).
- Perder o nível original de 4 ao colapsar `medium`/`low` em `minor` sem registrar.
