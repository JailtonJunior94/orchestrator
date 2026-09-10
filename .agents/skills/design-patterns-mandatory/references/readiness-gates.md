# Readiness Gates

Todos os gates abaixo devem passar antes de a decisao sair como `done`.

## Gate 1: Suficiencia de Evidencia
- Existe codigo, diff, pseudocodigo ou descricao estrutural suficiente.
- Existe ao menos um `path:line` ou declaracao explicita de `greenfield`.
- Os sinais canonicos foram mapeados sem extrapolacao.

## Gate 2: Alternativa Simples
- A alternativa mais simples foi descrita.
- A alternativa simples so perdeu por custo total pior, nao por preferencia estetica.
- Se a alternativa simples venceu, o status final e `reject` ou `nao aplicar padrao`.

## Gate 3: Pattern Primario
- O pattern primario pertence ao catalogo fechado da skill.
- Os sinais fortes do pattern estao presentes.
- Os sinais de exclusao nao estao presentes.
- O custo estrutural e proporcional ao problema.

## Gate 4: Pattern Complementar
- So existe pattern complementar quando o primario nao fecha a solucao sozinho.
- O complementar nao duplica responsabilidade do primario.
- O complementar tem justificativa explicita no bundle.

## Gate 5: Implementacao e Testes
- O mapeamento por paradigma e compativel com a linguagem real.
- O plano de implementacao preserva contratos publicos e invariantes.
- O plano de testes cobre positivo, negativo, regressao e falha operacional relevante.

## Gate 6: Estados Finais
- `done`: bundle completo, seletor executado, validador executado e blockers vazios.
- `needs_input`: falta evidencia material para decidir com seguranca.
- `blocked`: existe impedimento externo que impede completar a decisao.
- `failed`: houve erro estrutural ou contradicao nao resolvida no bundle.
