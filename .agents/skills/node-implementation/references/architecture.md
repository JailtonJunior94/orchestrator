# Arquitetura Node/TypeScript

<!-- TL;DR
Especificidades de arquitetura Node/TypeScript: DI por construtor/factory, layouts para API, worker, monolito modular e monorepo (workspaces), e adaptacao a projetos existentes e a pequeno/medio/grande porte.
Keywords: arquitetura, di, factory, monorepo, workspaces, camadas, layout, legado
Load complete when: tarefa envolve estrutura de projeto, organizacao de modulos, injecao de dependencia, monorepo ou decisao de fronteiras de camada em Node/TypeScript.
-->

Principios gerais de arquitetura, DI e sinais de excesso estao em `shared-architecture.md` (agent-governance). Este arquivo cobre apenas especificidades Node/TypeScript.

## DI em Node/TypeScript
- Preferir construtores e factory functions. Usar `tsyringe`, `inversify` ou NestJS modules apenas quando justificado.
- Depender de interfaces/types nas fronteiras de IO, nao de classes concretas (R6).
- Nao usar import com efeito colateral (modulo que executa IO ao ser importado) como mecanismo de wiring — montar dependencias explicitamente no entrypoint (`main.ts`/`index.ts`).

## Estrutura de Diretorios

### Projeto novo — layouts recomendados

#### API HTTP/gRPC
```
src/
  domain/<aggregate>/         # entidades, value objects, regras
  application/<usecase>/      # orquestracao, interfaces de porta
  infra/<adapter>/            # repositories, clients, messaging
  http/                       # controllers, DTOs, middlewares
  main.ts                     # composition root (wiring explicito)
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
    http/
  shared/                     # tipos e utilitarios genuinamente compartilhados
  main.ts
```
- Cada modulo expoe uma fronteira publica (index/barrel) e esconde o interno.
- Comunicacao entre modulos via interface de aplicacao ou evento, nunca import direto de `infra` alheio.

#### Monorepo (workspaces)
```
packages/
  <app>/                      # aplicacoes deployaveis (api, worker)
  <lib>/                      # libs compartilhadas versionadas
package.json                  # workspaces: ["packages/*"]
```
- Definir `workspaces` no `package.json` (npm/pnpm/yarn) e validar apenas os pacotes afetados (R7/Etapa 5).
- Cada pacote tem `package.json` e `tsconfig.json` proprios; usar `references` do TypeScript para build incremental.
- Nao criar dependencia ciclica entre pacotes — libs nao dependem de apps.

### Regras Node/TypeScript
- `src/` contem codigo; `test/` ou `__tests__/` contem testes.
- Evitar `utils/` ou `helpers/` que misturem responsabilidades (preferir modulos nomeados pelo dominio).
- Respeitar o sistema de modulos existente (ESM vs CJS) — nao misturar `require` e `import` no mesmo pacote.
- Profundidade maxima recomendada: `src/<camada>/<modulo>/`.

## Projetos existentes (legado)
- Mapear primeiro a arquitetura real antes de propor camadas novas; preservar o estilo dominante.
- Introduzir camada/abstracao apenas quando houver fronteira consumidora real ou ponto de teste claro (R6) — nao "modernizar" oportunisticamente.
- Em codigo JS sem tipos, adicionar tipagem incremental (`// @ts-check` + JSDoc ou migracao por arquivo) sem reescrever em massa.

## Escala (pequeno / medio / grande)
- **Pequeno:** layout achatado (`src/` com `domain`/`infra`/`http`) e suficiente; nao impor monolito modular.
- **Medio:** adotar monolito modular quando houver >=3 modulos de dominio com fronteiras nitidas.
- **Grande / monorepo:** isolar apps e libs em workspaces; validar e versionar por pacote; build incremental com project references.
