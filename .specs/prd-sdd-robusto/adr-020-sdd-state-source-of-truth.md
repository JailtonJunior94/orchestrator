# ADR-020 — Estado SDD versionado como fonte operacional

## Status

Aceita.

## Contexto e Decisão

Markdown é legível, mas parsing textual e hashes sincronizáveis permitem drift silencioso. O CLI
passará a persistir e validar `sdd-state.json` v2; Markdown permanece uma projeção humana.

## Consequências

Transições se tornam auditáveis e fail-closed; é necessário manter leitor legado warning-only por
duas versões menores.
