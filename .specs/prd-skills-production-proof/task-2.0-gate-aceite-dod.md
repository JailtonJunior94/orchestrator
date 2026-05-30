# Tarefa 2.0: Gate de aceite/DoD — template + validate-task-evidence.sh + DiffSHA default-on

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Fechar o principal vetor de falso positivo de `done`: cada critério de aceite da task file deve
ter evidência verificável no report, `Testes: pass` sem prova é rejeitado, e DiffSHA (F35) passa a
default-on. Cobre RF-01..RF-04. Ver techspec "Gate de Critérios de Aceite" e "DiffSHA default-on".

<requirements>
- Seção obrigatória `## Critérios de Aceite` + item de DoD no `task-execution-report-template.md`.
- `validate-task-evidence.sh` extrai cada critério da task file (`## Critérios de Sucesso`/`## Critérios de Aceite`) e falha se algum não estiver comprovado no report.
- `Testes: pass` exige comando de teste correspondente em `## Comandos Executados`; sem ele → falha.
- `AI_VALIDATE_GIT_HISTORY` default `1` em `post-execute-task.sh` (opt-out via `=0`).
- Tasks legadas sem seção de critérios → aviso não-fatal (zero regressão, RF-22).
</requirements>

## Subtarefas

- [ ] 2.1 Adicionar `## Critérios de Aceite` (1 item por critério, status verificável) + DoD ao template.
- [ ] 2.2 Estender `validate-task-evidence.sh`: resolver task file via campo `Arquivo:`, extrair critérios, exigir comprovação.
- [ ] 2.3 Rejeitar prova fraca de teste (`Testes: pass` sem comando associado).
- [ ] 2.4 Mudar default de `AI_VALIDATE_GIT_HISTORY` para `1` em `post-execute-task.sh`.
- [ ] 2.5 Referenciar o gate na Etapa 4.3 de `execute-task/SKILL.md`.

## Detalhes de Implementação

Ver techspec.md. Compatibilidade: ausência de seção de critérios na task → warning, não erro.

## Critérios de Sucesso

- Report com critério de aceite não comprovado → validador `exit 1`.
- Report com todos os critérios comprovados → validador `exit 0`.
- `Testes: pass` sem comando de teste → `exit 1`.
- Task legada sem seção de critérios → `exit 0` com aviso visível.
- `AI_VALIDATE_GIT_HISTORY` ausente comporta-se como `1`; `=0` restaura o opt-out.

## Skills Necessárias

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes shell (fixtures): casos a/b/c/d da techspec "Abordagem de Testes".
- [ ] Confirmar contagem de eventos inalterada com DiffSHA default-on.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `.agents/scripts/validate-task-evidence.sh`
- `.agents/skills/execute-task/assets/task-execution-report-template.md`
- `.agents/hooks/post-execute-task.sh`
- `.agents/skills/execute-task/SKILL.md`
