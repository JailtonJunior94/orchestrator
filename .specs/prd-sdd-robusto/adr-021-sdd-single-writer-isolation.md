# ADR-021 — Escritor único e isolamento obrigatório

## Status

Aceita.

## Contexto e Decisão

Atualizações concorrentes de tasks e relatórios causam lost update. O orquestrador terá um lock por
PRD e só habilitará escrita paralela em worktrees isoladas com ownership disjunto comprovado.

## Consequências

Há menor paralelismo em runtimes sem isolamento, em troca de não produzir falso `done`.
