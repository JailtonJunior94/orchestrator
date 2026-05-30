# Tarefa 7.0: Economia — RF default-on + skills via metadado + validador review + budget/kill

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Endurecimento fino de economia/eficiência: rastreabilidade RF default-on no `bugfix`; lista de
skills auto-carregadas derivada de metadado; validador de evidência para `--auto-review`; orçamento
de tokens por tool + kill de subagente no timeout quando suportado. Cobre RF-18..RF-21.

<requirements>
- `bugfix` passa RF/tasks ao validador por padrão (rastreabilidade default-on).
- `create-tasks` deriva a lista de skills auto-carregadas de metadado (frontmatter), não de prosa.
- Novo `validate-review-evidence.sh` + template para o modo `--auto-review`.
- `execute-all-tasks` aplica orçamento de tokens por tool e mata o subagente no timeout quando o tool suporta kill.
</requirements>

## Subtarefas

- [ ] 7.1 Tornar rastreabilidade RF default-on no validador do `bugfix`.
- [ ] 7.2 Derivar lista de skills auto-carregadas de frontmatter em `create-tasks/SKILL.md`.
- [ ] 7.3 Criar `validate-review-evidence.sh` (canônico em `.agents/scripts/`) + template de review.
- [ ] 7.4 Adicionar orçamento por tool + kill no timeout em `execute-all-tasks/SKILL.md`.

## Detalhes de Implementação

Ver techspec "Economia/eficiência" (RF-18..RF-21). O validador de review espelha a estrutura de
`validate-task-evidence.sh` (simetria de garantia).

## Critérios de Sucesso

- `bugfix` valida rastreabilidade RF sem flag opt-in.
- `create-tasks` não depende de lista hardcoded em prosa para skills auto-carregadas.
- `validate-review-evidence.sh` existe e valida o `review.md` do `--auto-review`.
- `execute-all-tasks` documenta orçamento por tool e mata subagente no timeout quando suportado.

## Skills Necessárias

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Teste shell do validador de review (fixtures válido/inválido).
- [ ] Verificação: `create-tasks` enumera skills via frontmatter.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `.agents/scripts/validate-bugfix-evidence.sh`, `.agents/scripts/validate-review-evidence.sh`
- `.agents/skills/create-tasks/SKILL.md`, `.agents/skills/review/SKILL.md`, `.agents/skills/execute-all-tasks/SKILL.md`
