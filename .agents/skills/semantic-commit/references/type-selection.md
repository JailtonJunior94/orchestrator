# Regras de Seleção de Tipo

Use estas regras apenas quando a classificação não estiver evidente no diff ou na descrição do usuário.

## Prioridade por Evidência
1. Comportamento observado ou contrato alterado.
2. Objetivo declarado pelo usuário, se compatível com a evidência.
3. Arquivos alterados.
4. Ferramenta ou linguagem envolvida.

## Matriz de Decisão

### `feat`
Use quando a mudança introduz capacidade nova percebida por usuário, API, fluxo ou integração.
Sinais comuns:
- novo endpoint, comando, tela, campo funcional ou automação de produto
- expansão de contrato compatível
- novo comportamento habilitado por flag

Não use quando a mudança apenas corrige algo que já deveria funcionar.

### `fix`
Use quando a mudança corrige comportamento incorreto, regressão, bug de integração, erro de validação ou resultado inesperado.
Sinais comuns:
- correção de condição, cálculo, query, tratamento de erro ou fluxo quebrado
- ajuste para alinhar implementação ao comportamento esperado já existente

Não use quando o comportamento novo não existia antes.

### `refactor`
Use quando a estrutura interna muda sem alterar comportamento externo esperado.
Sinais comuns:
- extração de funções, classes ou módulos
- renomeação interna
- simplificação de fluxo
- reorganização de camadas

Se a refatoração também melhora desempenho de forma explicitamente intencional e relevante, considerar `perf`.

### `perf`
Use quando a motivação principal é desempenho.
Sinais comuns:
- redução clara de consultas, alocações, loops ou chamadas remotas
- cache, batching, lazy loading, debounce, memoization com objetivo de performance

Não use se a melhora de desempenho for apenas efeito colateral de um refactor.

### `docs`
Use quando a mudança principal está em documentação consumida por pessoas.
Sinais comuns:
- README, guias, exemplos, changelog, comentários de uso

Não use para comentários triviais em código misturados a mudança funcional dominante.

### `test`
Use quando a mudança principal adiciona, ajusta ou corrige testes.
Sinais comuns:
- novos cenários
- correção de fixtures
- melhoria de asserts ou cobertura

Não use quando testes só acompanham uma feature ou fix dominante.

### `chore`
Use para manutenção geral sem impacto funcional direto no produto nem foco em build/CI.
Sinais comuns:
- limpeza de repositório
- atualização de arquivos auxiliares
- ajustes de configuração local
- housekeeping operacional

### `build`
Use quando a mudança afeta build, empacotamento, dependências de build ou publicação de artefatos.
Sinais comuns:
- Dockerfile, bundler, package manager, lockfile por motivo de build
- configuração de compilação, release artifact, versionamento de build

### `ci`
Use quando a mudança afeta pipeline, workflow automatizado, verificação remota ou integração contínua.
Sinais comuns:
- GitHub Actions, Azure Pipelines, Jenkinsfile, validações automatizadas

### `style`
Use apenas para formatação sem efeito funcional.
Sinais comuns:
- lint fix automático
- whitespace
- ordenação cosmética

Não use quando houver qualquer efeito funcional misturado ao mesmo conjunto.

## Desempates
- Entre `feat` e `fix`, prefira `fix` se o comportamento já era esperado antes.
- Entre `refactor` e `perf`, prefira `perf` somente quando desempenho for o objetivo principal.
- Entre `chore` e `build`, prefira `build` se a mudança interferir no processo de build ou empacotamento.
- Entre `chore` e `ci`, prefira `ci` se a mudança ocorrer em pipeline ou automação remota.
- Entre `docs` e qualquer tipo funcional, prefira o tipo funcional quando a documentação apenas acompanha a implementação.
