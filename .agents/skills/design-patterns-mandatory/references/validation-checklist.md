# Checklist Final de Validacao

Cada item abaixo deve passar antes da entrega.

## Decisao
- O problema foi descrito de forma objetiva.
- O pattern foi escolhido por evidencia, nao por analogia.
- A compatibilidade com a codebase cita `path:line` ou declara `greenfield`.
- A alternativa simples rejeitada foi registrada.
- Patterns proximos foram rejeitados explicitamente.
- `nao aplicar padrao` foi usado quando a evidenica nao justificou o custo.

## Economia
- O pattern reduz custo de mudanca, duplicacao ou acoplamento recorrente.
- O numero de tipos, wrappers ou factories adicionados e proporcional ao problema.
- Nao ha abstracao sem segunda variante plausivel.

## Eficiencia
- O ganho prometido em execucao ou manutencao tem mecanismo claro.
- Nao ha penalidade desnecessaria de indirecao em hot path.
- O desenho reduz branching, dependencia concreta ou churn.

## Robustez
- Contratos publicos e invariantes foram preservados.
- O fluxo continua rastreavel e testavel.
- Falhas novas introduzidas pelo pattern foram reconhecidas e mitigadas.

## Entrega
- O bundle segue `assets/pattern-decision-template.md`.
- `python3 scripts/validate_pattern_bundle.py <bundle.md>` retorna `SUCCESS`.
- O plano de testes cobre positivo, negativo e regressao.
- O mapeamento por paradigma esta alinhado com a linguagem real.
