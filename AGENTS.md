# AGENTS.md

Este arquivo adapta a convenção de `.claude/` para o uso no Codex neste repositório.

## Fonte de Verdade

- Regras e contexto vivem em `.claude/`.
- Este arquivo resume como o agente deve aplicar esse material.
- Em caso de conflito, seguir a precedência definida em `.claude/rules/governance.md`.

## Precedência

1. `.claude/rules/governance.md`
2. `.claude/rules/security.md`
3. `.claude/rules/architecture.md`
4. `.claude/rules/cli.md`
5. `.claude/rules/error-handling.md`
6. `.claude/rules/o11y.md`
7. `.claude/rules/ddd.md`
8. `.claude/rules/tests.md`
9. `.claude/rules/code-standards.md`
10. `tasks/prd-[feature]/prd.md` e `tasks/prd-[feature]/techspec.md`, quando existirem
11. Uber Go Style Guide PT-BR como baseline transversal

## Contexto do Projeto

- Produto: CLI de orquestração de agentes de IA.
- Linguagem: Go 1.22+.
- CLI: `cobra`.
- Logging: `log/slog`.
- Automação local: `task`.
- Release: `goreleaser`.
- Workflows: YAML.
- Persistência: filesystem + JSON.
- Providers V1: Claude CLI e Copilot CLI via subprocesso.

## Mapa de Diretórios

- `cmd/`: entrypoint da CLI.
- `internal/cli/`: comandos Cobra, flags e rendering.
- `internal/runtime/`: engine de execução e HITL.
- `internal/workflows/`: parser, validação e catálogo de workflows.
- `internal/providers/`: contratos e adapters de providers.
- `internal/state/`: persistência de runs e artefatos.
- `internal/platform/`: wrappers de OS, subprocesso, terminal, editor e filesystem.
- `pkg/`: tipos e utilitários realmente compartilhados.
- `.orq/`: artefatos e estado de execução.
- `tasks/prd-[feature]/`: PRD, techspec e tarefas de feature.

## Regras Operacionais

- Manter arquitetura de CLI, não backend HTTP.
- Respeitar dependências apontando para dentro: adapters/infrastructure dependem de application/domain.
- Não colocar regra de negócio em comandos Cobra.
- Não acoplar domínio a terminal, subprocesso, filesystem ou serialização.
- Usar `filepath` para paths e `exec.CommandContext` para subprocessos.
- Evitar shell inline quando a chamada direta for suficiente.
- Não hardcodar segredos, não logar segredos, não persistir segredos em `.orq/`.
- Não usar `panic` para erro recuperável.
- Fazer wrapping com `%w` e preservar `errors.Is` / `errors.As`.
- Usar `log/slog` para logging operacional.
- Não usar `fmt.Println` como mecanismo principal de logging operacional.
- Código, comentários de código e testes devem estar em inglês.
- Símbolos exportados devem ter godoc.
- Todo bug corrigido deve incluir teste de regressão.
- `go test ./...` é o gate mínimo de validação.

## Arquitetura Esperada

- Fluxo preferencial: CLI -> bootstrap/composição -> application -> domain.
- O runtime controla a execução do workflow; providers não escolhem o próximo step.
- `Run` deve ser o aggregate root natural da execução.
- Estados de run permitidos: `pending`, `running`, `paused`, `failed`, `completed`, `cancelled`.
- Estados de step permitidos: `pending`, `running`, `waiting_approval`, `approved`, `retrying`, `failed`, `skipped`.
- Ações HITL permitidas: `approve`, `edit`, `redo`, `exit`.

## Tooling Preferido

- Preferir `task` como interface de desenvolvimento local.
- Comandos usuais:
  - `task build`
  - `task test`
  - `task lint`
  - `task check`
  - `task fmt`
- Para release local: `goreleaser release --snapshot --clean`.

## Como o Codex Deve Usar `.claude`

### Regras

- Antes de mudanças relevantes, consultar as regras em `.claude/rules/` conforme o tipo de trabalho.
- Para mudanças arquiteturais, consultar ao menos `architecture.md`, `ddd.md` e `security.md`.
- Para mudanças de CLI/UX terminal, consultar ao menos `cli.md`, `error-handling.md` e `o11y.md`.
- Para mudanças em testes, consultar `tests.md`.

### Contexto

- Usar `.claude/context/stack.md`, `.claude/context/paths.md` e `.claude/context/tooling.md` como referência de stack, estrutura e comandos.

### Commands

Os arquivos em `.claude/commands/` devem ser tratados como playbooks textuais, não como automações nativas da ferramenta.

- Se o usuário pedir `run-task`, seguir `.claude/commands/run-task.md`.
- Se o usuário pedir `create-prd`, seguir `.claude/commands/create-prd.md`.
- Se o usuário pedir `create-techspec`, seguir `.claude/commands/create-techspec.md`.
- Se o usuário pedir `create-tasks`, seguir `.claude/commands/create-tasks.md`.
- Se o usuário pedir `run-refactor`, seguir `.claude/commands/run-refactor.md`.

Ao executar um playbook:

- Ler os arquivos de contexto exigidos pelo playbook.
- Produzir os artefatos no caminho esperado.
- Só marcar uma task como concluída quando o playbook exigir e houver evidência.

### Skills

Os arquivos em `.claude/skills/*/SKILL.md` devem ser tratados como personas e workflows reutilizáveis.

- `bugfix`: usar quando o usuário pedir correção de bug ou quando existir lista de bugs canônica.
- `refactor`: usar para refatoração segura e incremental.
- `reviewer`: usar para review técnico/funcional com foco em bugs, riscos, regressões e aderência às regras.
- `semantic-commit`: usar para sugerir mensagem Conventional Commit a partir do diff.

Se o usuário mencionar explicitamente uma skill da `.claude`, ler o `SKILL.md` correspondente e seguir o fluxo descrito.

## Convenções para Tasks

Quando atuar no contexto de `tasks/prd-[feature]/`:

- Ler `prd.md`, `techspec.md`, `tasks.md` e o arquivo da task alvo.
- Respeitar dependências entre tarefas.
- Registrar evidência objetiva de validação.
- Gerar relatórios nos caminhos esperados pelos playbooks e skills.

## Restrições Importantes

- Não executar ações destrutivas de git ou publicações remotas sem pedido explícito do usuário.
- Não usar `git commit`, `git push` ou `gh pr create` como parte automática do fluxo padrão.
- Não inventar estados fora dos enums canônicos.
- Não aprovar solução com lacuna crítica conhecida.

## Observação Sobre `.claude/settings.local.json`

- Esse arquivo expressa permissões e preferências do ambiente Claude.
- No Codex, ele serve apenas como referência do tipo de comando comum no projeto, não como fonte normativa de permissão.
