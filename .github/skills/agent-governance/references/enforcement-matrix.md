# Contrato de Enforcement

<!-- TL;DR
O enforcement não é inferido por matriz estática de ferramentas. Use os adaptadores instalados e
o CLI para provar as capacidades locais antes de permitir uma operação sensível.
Keywords: enforcement, hook, adaptador, runtime-capabilities, isolamento, fail-closed
Load complete when: tarefa exige verificar quais regras de governança são tecnicamente impostas.
-->

- Rule ID: R-ENF-001
- Severidade: informativo
- Escopo: enforcement de governança em CLIs de IA.

## Fonte de verdade

Os validadores compartilhados vivem em `.agents/scripts/` e `.agents/hooks/`; os mirrors são
verificados por `check-skills-sync`, `check-hooks-sync` e `check-scripts-sync`. O adaptador deve
executar o gate instalado, propagar sua falha e não tratar documentação, nome de ferramenta ou
versão como prova de suporte.

Para isolamento, escrita concorrente ou paralelismo, consultar:

```bash
ai-spec runtime-capabilities <raiz-do-worktree>
```

O JSON retornado é a evidência local. Sem `isolated_worktrees: true`, escrita concorrente é
bloqueada; apenas o caminho sequencial ou read-only permitido pelo CLI pode continuar.

## Regra operacional

Ausência, erro ou resultado incompleto do adaptador é falha fechada para a capacidade exigida.
Registre no relatório o comando, o JSON ou erro retornado e a decisão aplicada.
