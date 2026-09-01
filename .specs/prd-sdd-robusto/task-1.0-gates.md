# Tarefa 1.0: Fechar escapes críticos de validadores e hooks

## Requisitos

RF-01, RF-03, RF-04, RF-05.

## Critérios de Sucesso

- A suíte adversarial é parte de `make test-validators` e CI.
- Hook rejeita path absoluto, traversal e symlink externo.
- Testes comprovam os quatro falsos positivos históricos como falhas.

## Skills Necessárias

Nenhuma além das auto-carregadas (governance + linguagem).
