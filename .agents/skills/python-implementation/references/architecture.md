# Arquitetura Python

<!-- TL;DR
Especificidades de arquitetura Python: DI por construtor/factory e Depends, layouts para API, worker, monolito modular e monorepo, e adaptacao a projetos existentes e a pequeno/medio/grande porte.
Keywords: arquitetura, di, depends, monorepo, pacotes, camadas, layout, legado
Load complete when: tarefa envolve estrutura de projeto, organizacao de modulos, injecao de dependencia, packaging ou decisao de fronteiras de camada em Python.
-->

Principios gerais de arquitetura, DI e sinais de excesso estao em `shared-architecture.md` (agent-governance). Este arquivo cobre apenas especificidades Python.

## DI em Python
- Preferir factory functions e construtores. Usar `dependency-injector` ou `FastAPI Depends` apenas quando justificado.
- Depender de `Protocol`/ABC nas fronteiras de IO, nao de classes concretas (R6).
- Nao usar import de modulo com efeito colateral (IO ao importar) como wiring — montar dependencias no entrypoint (`main.py`/`asgi.py`) ou via container explicito.

## Estrutura de Diretorios

### Projeto novo — layouts recomendados

#### API HTTP
```
src/
  domain/<aggregate>/         # entidades, value objects, regras
  application/<usecase>/      # orquestracao, interfaces de porta
  infra/<adapter>/            # repositories, clients, messaging
  api/                        # routers, DTOs, middlewares
  main.py                     # composition root (wiring explicito)
```

#### Worker / Consumer
```
src/
  domain/
  application/
  infra/
  workers/                    # consumers, job handlers
```

#### Monolito modular
```
src/
  <module>/                   # cada modulo isola seu dominio
    domain/
    application/
    infra/
    api/
  shared/                     # tipos e utilitarios genuinamente compartilhados
  main.py
```
- Cada modulo expoe uma fronteira publica (`__init__.py` enxuto) e esconde o interno.
- Comunicacao entre modulos via interface de aplicacao ou evento, nunca import direto de `infra` alheio.

#### Monorepo (multi-package)
```
packages/
  <app>/                      # aplicacoes deployaveis (api, worker)
    pyproject.toml
  <lib>/                      # libs compartilhadas
    pyproject.toml
pyproject.toml                # workspace (uv/hatch/poetry) ou metapacote
```
- Usar workspace do gerenciador (uv, hatch, poetry) e validar apenas os pacotes afetados (R7/Etapa 5).
- Cada pacote tem `pyproject.toml` proprio; libs nao dependem de apps (sem ciclo).

### Regras Python
- `src/` contem codigo de aplicacao; `tests/` contem testes (preferir layout `src/`).
- Evitar `utils/` ou `helpers/` que misturem responsabilidades (preferir modulos nomeados pelo dominio).
- `__init__.py` apenas quando necessario para o import / fronteira de pacote.
- Profundidade maxima recomendada: `src/<camada>/<modulo>/`.

## Projetos existentes (legado)
- Mapear primeiro a arquitetura real antes de propor camadas novas; preservar o estilo dominante.
- Introduzir camada/abstracao apenas quando houver fronteira consumidora real ou ponto de teste claro (R6).
- Adicionar type hints incrementais (por modulo) e habilitar `mypy`/`pyright` em modo gradual, sem reescrita em massa.

## Escala (pequeno / medio / grande)
- **Pequeno:** layout achatado (`src/` com `domain`/`infra`/`api`) e suficiente; nao impor monolito modular.
- **Medio:** adotar monolito modular quando houver >=3 modulos de dominio com fronteiras nitidas.
- **Grande / monorepo:** isolar apps e libs em pacotes com `pyproject.toml` proprios; validar e versionar por pacote.
