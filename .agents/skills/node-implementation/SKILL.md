---
name: node-implementation
version: 1.4.0
category: language
prerequisites: [agent-governance]
description: Implementa alteracoes em codigo Node/TypeScript usando governanca base, regras estritas [HARD], convencoes de projeto e validacao proporcional com gates bloqueantes. Use quando a tarefa exigir adicionar, corrigir, refatorar ou validar codigo Node.js ou TypeScript. Nao use para tarefas sem codigo Node/TypeScript.
---

# Implementacao Node/TypeScript

## Procedimentos

**Etapa 1: Carregar base obrigatoria**
1. Confirmar que o contrato de carga base definido em `AGENTS.md` foi cumprido.
2. Ler `references/architecture.md`.
3. Ler `package.json` para identificar dependencias, scripts e engine.
4. Ler `tsconfig.json` quando existir para identificar versao alvo e configuracao de tipos.
5. Executar `bash .agents/skills/agent-governance/scripts/detect-toolchain.sh` para descobrir comandos de fmt, test e lint.
6. Carregar as **Regras Estritas Obrigatorias (R0-R7)** desta skill. Sao `[HARD]` (bloqueantes de
   merge) salvo quando marcadas `[SOFT]`. Aplicam-se a todo codigo Node/TypeScript de dominio,
   aplicacao e infraestrutura produzido ou modificado, em qualquer camada.

## Regras Estritas Obrigatorias (R0-R7)

> Severidade padrao: toda violacao e `[HARD]` (bloqueante de merge) salvo marcacao explicita `[SOFT]`.
> As regras sao cumulativas e nao tem precedencia entre si. Em conflito com outra orientacao desta
> skill, prevalece a **restricao mais restritiva**. Verificar a versao alvo em `tsconfig.json` /
> `package.json` (`engines`) antes de aplicar recursos de linguagem; se a versao for anterior ao
> recurso, NAO usa-lo e registrar a omissao.

- **R0 — `strict` obrigatorio `[HARD]`:** `tsconfig.json` deve ter `"strict": true`. E proibido `any`
  implicito ou explicito sem comentario `// any justificado: <motivo>` na mesma linha. Preferir
  `unknown` + narrowing a `any`. Proibido `@ts-ignore`; usar `@ts-expect-error` com justificativa.
- **R1 — Sem `var`, sempre igualdade estrita `[HARD]`:** usar `const` por padrao e `let` apenas quando
  houver reatribuicao real; `var` e proibido. Usar `===`/`!==` — nunca `==`/`!=` (exceto `== null`
  para cobrir `null|undefined`, permitido).
- **R2 — Sem efeitos colaterais de log em producao `[HARD]`:** proibido `console.log`/`console.error`
  em codigo de producao; usar o logger estruturado do projeto. `console.*` so em scripts/CLI explicitos.
- **R3 — Erros tipados e encadeados `[HARD]`:** lancar subclasses de `Error` (nunca string/objeto cru);
  preservar a causa com `new Error(msg, { cause })`. Proibido `catch` vazio ou que apenas re-loga e
  engole; tratar o erro **uma unica vez**. Mensagens em PT-BR, sem `"failed to..."`.
- **R4 — Promises sem vazamento `[HARD]`:** proibida floating promise — toda Promise deve ser
  `await`-ada, retornada ou explicitamente tratada (`.catch`/`void`). Nao marcar `async` funcao sem
  `await`. Usar `Promise.all`/`allSettled` para paralelismo independente em vez de `await` sequencial.
  Habilitar `@typescript-eslint/no-floating-promises`.
- **R5 — Validacao na fronteira `[HARD]`:** todo input externo (HTTP, fila, env, arquivo) deve ser
  validado em runtime (zod, valibot, class-validator ou equivalente do projeto) antes de ser tratado
  como tipado. Nao confiar em cast (`as`) para dados nao confiaveis. DTOs explicitos, nunca `any`.
- **R6 — DI por construtor e fronteiras por interface `[HARD]`:** injetar dependencias via construtor
  ou factory; depender de interfaces/types em fronteiras de IO. Nao retornar entidades ORM do
  repositorio — mapear para entidades/DTOs de dominio. Container automatico so quando o projeto ja o adota.
- **R7 — Testes para todo comportamento `[HARD]`:** toda mudanca de comportamento exige teste novo ou
  atualizado; mockar por interface (nao por implementacao concreta); sem testes dependentes de ordem
  ou de estado global compartilhado. Async testado com `await`, nunca callback solto.

**Patterns frequentes (inline — evitar carregar patterns.md para estes)**
- **Factory Function:** Preferir factory functions sobre classes quando nao houver estado mutavel. Usar factory quando a construcao envolver validacao ou defaults complexos.
- **Dependency Injection:** Preferir injecao via construtor sobre containers automaticos, salvo quando o projeto ja adotar um container (tsyringe, inversify, NestJS IoC). Depender de interfaces/types em fronteiras de IO.
- **Repository:** Interface do repository deve expor operacoes de dominio, nao primitivas SQL. Nao retornar entidades ORM diretamente — mapear para entidades de dominio.

**Etapa 2: Selecionar apenas o contexto necessario**
1. Ler `references/conventions.md` quando a tarefa envolver estrutura de projeto, organizacao de modulos ou padroes de importacao.
2. Ler `references/testing.md` quando a tarefa envolver estrategia de testes, mocking ou cobertura.
3. Ler `references/api.md` quando a tarefa envolver handlers HTTP, middlewares, DTOs, validacao de request ou serializacao.
4. Ler `references/patterns.md` **somente** quando a tarefa envolver strategy, composicao vs heranca ou organizacao de modulos nao cobertos inline. Factory, DI e Repository ja estao definidos na secao "Patterns frequentes" acima e NAO devem motivar o carregamento deste arquivo — isso evita ~500 tokens redundantes.
5. Ler `references/concurrency.md` quando a tarefa envolver Promises, controle de concorrencia, worker threads, streams ou event loop.
6. Ler `references/resilience.md` quando a tarefa envolver retries, circuit breakers, timeouts em chamadas externas, fallbacks ou health checks.
7. Ler `references/build.md` quando a tarefa envolver Dockerfile, pipeline de CI, bundling, package manager ou empacotamento.
8. Ler `references/graceful-lifecycle.md` quando a tarefa envolver shutdown gracioso, signal handling (SIGTERM/SIGINT), drain de conexoes HTTP ou encerramento de workers e streams.
9. Ler `references/examples-domain-flow.md` quando a tarefa precisar de esqueleto concreto de fluxo end-to-end (entidade, use case, handler, teste). Para tarefas menores, usar o esqueleto inline: `Entity/VO -> UseCase(deps) -> Controller(useCase) -> test com jest/vitest mock`, sem carregar o arquivo completo.
10. Ler `references/examples-testing.md` quando a tarefa precisar de exemplos de parametrized tests, factory de mocks, validacao de DTOs ou assercoes async.
11. Ler `references/examples-infrastructure.md` quando a tarefa precisar de exemplo de graceful shutdown, paginacao cursor-based ou versionamento de API.
12. Ler `references/configuration.md` quando a tarefa envolver carregamento de configuracao, variaveis de ambiente ou inicializacao de dependencias.
13. Ler `../agent-governance/references/error-handling.md` quando a tarefa criar, propagar, encapsular ou apresentar erros.
14. Ler `references/persistence.md` quando a tarefa envolver repositories, transactions, migrations, queries ou connection management.
15. Ler `references/observability.md` quando a tarefa envolver logging, tracing, metricas ou health checks.
16. Ler `references/security.md` quando a tarefa envolver autenticacao, autorizacao, validacao de input, rate limiting, CORS ou tratamento de segredos.
17. Ler `references/messaging.md` quando a tarefa envolver producao ou consumo de mensagens, eventos, filas, topicos ou idempotencia de consumidores.

**Economia de contexto**
Classificar a complexidade da tarefa (trivial / standard / complex) conforme
`agent-governance/SKILL.md` antes de carregar referencias, e respeitar o teto correspondente:
- **trivial** (rename, typo, import, formatacao): nenhuma referencia — apenas esta SKILL.md.
- **standard** (metodo novo, fix local, refactor local): no maximo o TL;DR das 1-2 referencias
  diretamente ligadas a superficie alterada (o bloco `<!-- TL;DR -->` no topo de cada referencia).
- **complex** (feature, interface publica, migracao): carregar referencias completas sob demanda.
Se mais de 4 referencias forem necessarias para a mesma tarefa, priorizar as 3 mais criticas para o
escopo da mudanca e registrar as demais como contexto nao carregado. Carregar referencias adicionais
apenas se a implementacao revelar necessidade concreta.

**Etapa 3: Modelar a alteracao**
1. Identificar o menor conjunto seguro de mudancas que satisfaz a solicitacao.
2. Mapear o comportamento afetado, as dependencias envolvidas e o risco de regressao.
3. Preferir tipagem estrita em TypeScript; evitar `any` sem justificativa.
4. Respeitar o estilo existente do projeto (ESM vs CJS, semicolons, aspas).

**Etapa 4: Implementar**
1. Editar o codigo seguindo as convencoes do contexto analisado.
2. Atualizar ou adicionar testes para toda mudanca de comportamento.
3. Adaptar exemplos ao contexto real em vez de replica-los literalmente.

**Etapa 5: Validar**
1. Seguir Etapa 4 de `.agents/skills/agent-governance/SKILL.md`.
2. Em Node/TypeScript, preferir os scripts definidos em `package.json` (ex: `npm run lint`, `npm test`).
3. Executar o **Checklist de Validacao (R0-R7)** e reportar o resultado de cada gate. Qualquer item com
   resultado diferente do esperado e `[HARD]` — bloqueante de merge:
   - **Type check (R0/R5):** `npx tsc --noEmit` (ou o script equivalente) sem erro novo.
   - **Lint (R1/R2/R4):** `npm run lint` (eslint) com `no-floating-promises`, `no-explicit-any` e
     `eqeqeq` ativos; grep por regressao: `grep -rnE '\bvar |console\.(log|error)|@ts-ignore' src/`.
   - **Testes (R7):** `npm test` direcionado aos modulos afetados; em monorepo, apenas os workspaces afetados.
   - **Format:** `npm run format` / `prettier --check` (ou o formatter do projeto) sem diferencas.
   Se um comando nao existir no projeto, registrar a ausencia explicitamente em vez de inventar substituto.

## Tratamento de Erros
* Se `package.json` estiver ausente, parar antes de assumir dependencias ou runtime.
* Se o projeto usar monorepo (workspaces), validar apenas os workspaces afetados pela mudanca.
* Se houver conflito entre esta skill e a governanca base, seguir a restricao mais segura e registrar a suposicao.
