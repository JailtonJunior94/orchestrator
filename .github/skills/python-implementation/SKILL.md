---
name: python-implementation
version: 1.4.0
category: language
prerequisites: [agent-governance]
description: Implementa alteracoes em codigo Python usando governanca base, regras estritas [HARD], convencoes de projeto e validacao proporcional com gates bloqueantes. Use quando a tarefa exigir adicionar, corrigir, refatorar ou validar codigo Python. Nao use para tarefas sem codigo Python.
---

# Implementacao Python

## Procedimentos

**Etapa 1: Carregar base obrigatoria**
1. Confirmar que o contrato de carga base definido em `AGENTS.md` foi cumprido.
2. Ler `references/architecture.md`.
3. Ler `pyproject.toml`, `setup.py` ou `requirements.txt` para identificar dependencias e versao de Python.
4. Executar `bash .agents/skills/agent-governance/scripts/detect-toolchain.sh` para descobrir comandos de fmt, test e lint.
5. Carregar as **Regras Estritas Obrigatorias (R0-R7)** desta skill. Sao `[HARD]` (bloqueantes de
   merge) salvo quando marcadas `[SOFT]`. Aplicam-se a todo codigo Python de dominio, aplicacao e
   infraestrutura produzido ou modificado, em qualquer camada.

## Regras Estritas Obrigatorias (R0-R7)

> Severidade padrao: toda violacao e `[HARD]` (bloqueante de merge) salvo marcacao explicita `[SOFT]`.
> As regras sao cumulativas e nao tem precedencia entre si. Em conflito com outra orientacao desta
> skill, prevalece a **restricao mais restritiva**. Verificar a versao de Python no projeto
> (`pyproject.toml`/`requires-python`) antes de aplicar recursos de linguagem; se a versao for
> anterior ao recurso, NAO usa-lo e registrar a omissao.

- **R0 — Type hints e checagem estatica `[HARD]`:** type hints obrigatorios em toda assinatura publica
  (funcoes, metodos, atributos de dataclass). O type checker do projeto (`mypy` ou `pyright`) nao pode
  introduzir erro novo. Proibido `# type: ignore` sem comentario de motivo na mesma linha. Preferir
  tipos modernos (`X | None`, `list[T]`) quando a versao do projeto permitir.
- **R1 — Excecoes especificas e encadeadas `[HARD]`:** proibido `except:` nu e `except Exception` sem
  re-`raise` ou tratamento real. Capturar a excecao mais especifica possivel e preservar o contexto com
  `raise NovoErro(...) from err`. Nunca engolir excecao silenciosamente (`except ...: pass` so com
  comentario justificando). Tratar o erro **uma unica vez**; mensagens em PT-BR.
- **R2 — Sem mutable default arguments `[HARD]`:** proibido `def f(x=[])` / `={}` / `=set()`. Usar
  `x: list[T] | None = None` e inicializar dentro da funcao. O mesmo vale para defaults de dataclass —
  usar `field(default_factory=...)`.
- **R3 — Value objects/DTOs imutaveis `[HARD]`:** usar `@dataclass(frozen=True)` (ou `attrs`/pydantic)
  para value objects e DTOs; invariantes em `__post_init__` ou validators. Nao usar dict cru como DTO
  em fronteira de dominio. Entidades com ciclo de vida gerenciado por ORM ficam fora desta regra.
- **R4 — Async correto `[HARD]`:** nao chamar codigo bloqueante (IO sincrono, `time.sleep`) dentro de
  corrotina — usar a variante async ou `run_in_executor`. Usar `asyncio.gather` para paralelismo
  independente. Sempre fechar recursos com `async with`/`with`; nao criar tasks orfas sem aguardar/cancelar.
- **R5 — Validacao na fronteira `[HARD]`:** todo input externo (HTTP, fila, env, arquivo) deve ser
  validado em runtime (pydantic, `dataclass` + validacao explicita ou equivalente do projeto) antes de
  ser tratado como confiavel. Nao confiar em type hints para dados externos — eles nao validam em runtime.
- **R6 — DI por construtor e fronteiras por Protocol/ABC `[HARD]`:** injetar dependencias via construtor
  ou parametro de funcao; em FastAPI usar `Depends()`. Depender de `Protocol`/ABC em fronteiras de IO.
  Nao retornar instancias ORM do repositorio — mapear para entidades de dominio.
- **R7 — Testes para todo comportamento `[HARD]`:** toda mudanca de comportamento exige teste pytest novo
  ou atualizado; fixtures isoladas, sem estado global compartilhado entre testes; codigo async testado
  com `pytest.mark.asyncio`/`anyio`. Mockar por fronteira (Protocol/ABC), nao por implementacao concreta.

**Patterns frequentes (inline — evitar carregar patterns.md para estes)**
- **Dependency Injection:** Preferir injecao via construtor ou parametros de funcao. Em FastAPI, usar `Depends()`. Depender de Protocol ou ABC em fronteiras de IO.
- **Repository:** Interface do repository deve expor operacoes de dominio, nao primitivas SQL. Nao retornar instancias ORM diretamente — mapear para entidades de dominio.
- **Dataclasses:** Preferir `dataclass` ou `attrs` para value objects e DTOs. Usar `frozen=True` para imutabilidade. Usar `__post_init__` para invariantes.

**Etapa 2: Selecionar apenas o contexto necessario**
1. Ler `references/conventions.md` quando a tarefa envolver estrutura de projeto, organizacao de modulos ou padroes de importacao.
2. Ler `references/testing.md` quando a tarefa envolver estrategia de testes, fixtures ou cobertura.
3. Ler `references/api.md` quando a tarefa envolver handlers HTTP, middlewares, DTOs, validacao de request ou serializacao.
4. Ler `references/patterns.md` **somente** quando a tarefa envolver strategy, composicao vs heranca ou organizacao de modulos nao cobertos inline. DI, Repository e Dataclasses ja estao definidos na secao "Patterns frequentes" acima e NAO devem motivar o carregamento deste arquivo — isso evita ~480 tokens redundantes.
5. Ler `references/concurrency.md` quando a tarefa envolver asyncio, threading, multiprocessing, controle de concorrencia ou paralelismo.
6. Ler `references/resilience.md` quando a tarefa envolver retries, circuit breakers, timeouts em chamadas externas, fallbacks ou health checks.
7. Ler `references/build.md` quando a tarefa envolver Dockerfile, pipeline de CI, packaging, gerenciamento de dependencias ou distribuicao.
8. Ler `references/examples-domain-flow.md` quando a tarefa precisar de esqueleto concreto de fluxo end-to-end (entidade, use case, handler, teste). Para tarefas menores, usar o esqueleto inline: `Entity/dataclass -> UseCase(deps) -> Router(use_case) -> test com pytest fixtures`, sem carregar o arquivo completo.
9. Ler `references/examples-testing.md` quando a tarefa precisar de exemplos de fixtures, parametrize, validacao de schemas ou assercoes async.
10. Ler `references/examples-infrastructure.md` quando a tarefa precisar de exemplo de graceful shutdown, paginacao cursor-based ou versionamento de API.
11. Ler `references/graceful-lifecycle.md` quando a tarefa envolver shutdown gracioso, signal handling (SIGTERM/SIGINT), drain de conexoes, cleanup de asyncio tasks ou encerramento de workers.
12. Ler `references/configuration.md` quando a tarefa envolver carregamento de configuracao, variaveis de ambiente, pydantic-settings ou inicializacao de dependencias.
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
3. Preferir type hints em funcoes publicas.
4. Respeitar o estilo existente do projeto.

**Etapa 4: Implementar**
1. Editar o codigo seguindo a versao Python declarada no projeto e as convencoes do contexto analisado.
2. Atualizar ou adicionar testes para toda mudanca de comportamento.
3. Adaptar exemplos ao contexto real em vez de replica-los literalmente.

**Etapa 5: Validar**
1. Seguir Etapa 4 de `.agents/skills/agent-governance/SKILL.md`.
2. Em Python, preferir `ruff` para lint e format quando disponivel; caso contrario, `black` + `flake8` ou o toolchain do projeto.
3. Executar o **Checklist de Validacao (R0-R7)** e reportar o resultado de cada gate. Qualquer item com
   resultado diferente do esperado e `[HARD]` — bloqueante de merge:
   - **Type check (R0/R5):** `mypy` ou `pyright` sem erro novo nos modulos afetados.
   - **Lint (R1/R2):** `ruff check` (ou `flake8`) limpo; grep por regressao:
     `grep -rnE 'except\s*:|except Exception\s*:|def .*=\s*(\[\]|\{\})' src/` deve vir vazio ou justificado.
   - **Testes (R7):** `pytest` direcionado aos modulos afetados; em monorepo, apenas os packages afetados.
   - **Format:** `ruff format --check` (ou `black --check`) sem diferencas.
   Se um comando nao existir no projeto, registrar a ausencia explicitamente em vez de inventar substituto.

## Tratamento de Erros
* Se nenhum arquivo de configuracao Python for encontrado, parar antes de assumir versao ou dependencias.
* Se o projeto usar monorepo, validar apenas os packages afetados pela mudanca.
* Se houver conflito entre esta skill e a governanca base, seguir a restricao mais segura e registrar a suposicao.
