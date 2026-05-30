# Relatório de Execução de Tarefa

## Tarefa
- ID:
- Título:
- Arquivo: .specs/prd-<slug>/NN_<nome>.md
- Estado: pending | in_progress | needs_input | blocked | failed | done

## Contexto Carregado
- PRD: [caminho do prd.md consultado]
- TechSpec: [caminho do techspec.md consultado]
- Governança: [skills e references carregadas]

## Comandos Executados
- [comando] -> [resultado]

## Arquivos Alterados
- [caminho]

## Resultados de Validação
- Testes: pass | fail | blocked
- Lint: pass | fail | blocked
- Veredito do Revisor: APPROVED | APPROVED_WITH_REMARKS | REJECTED | BLOCKED

## Critérios de Aceite
<!-- Um item por critério da task file (## Critérios de Sucesso / ## Critérios de Aceite).
     Formato: `- [texto do critério] -> comprovado: <evidência verificável>`.
     Todo critério deve ter comprovação física (comando, arquivo, teste, output). -->
- [critério 1] -> comprovado: [evidência]

## Definition of Done (DoD)
- [ ] Todos os critérios de aceite acima comprovados com evidência física.
- [ ] Testes da tarefa criados e executados (`Testes: pass` com comando correspondente em Comandos Executados).
- [ ] Lint/vet/build sem regressão.
- [ ] Estado de tasks.md sincronizado com este relatório.

## Diff Reviewed

sha={{.DiffSHA}}
verdict={{.Verdict}}
tool={{.Tool}}

## Coverage

package={{.CoveragePackage}}
delta={{.CoverageDelta}}

## Suposições
- [suposição]

## Riscos Residuais
- [risco]

## Conflitos de Regra
- [rule_id_1 vs rule_id_2 | regra_escolhida | razão_de_precedência | "none" se sem conflitos]
