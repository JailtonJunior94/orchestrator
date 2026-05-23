# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Camada de configuração hierárquica (global + projeto) com upward-walk e precedência determinística
- **Data:** 2026-05-22
- **Status:** Proposta
- **Decisores:** Jailton Junior (owner), arquitetura ai-spec-harness
- **Relacionados:** [PRD](prd.md) RF-13/14/15/16/17; [techspec](techspec.md); referência Compozy `internal/config/home.go`; ADR-001 (go:embed)

## Contexto

Hoje `internal/config/runtime.go` carrega apenas configuração **de projeto** a partir de
`.claude/config.yaml` ou `.agents/config.yaml` na raiz do repositório (`LoadRuntime(repoRoot)`),
com 5 chaves (`tasks_root`, `prd_prefix`, `evidence_dir`, `coverage_threshold`,
`language_default`). Não existe:
- configuração **global** por usuário (defaults reutilizáveis entre projetos);
- **upward-walk** (a config só é encontrada se `repoRoot` já for a raiz);
- **precedência** explícita entre flags, projeto e defaults.

O par arquitetural Compozy resolve isso com `~/.compozy/config.toml` (global) + `.compozy/config.toml`
(workspace), upward-walk a partir do CWD e precedência `flags > workspace > global > built-in`. O PRD
exige paridade desse comportamento mantendo persistência **file-first** (sem banco).

## Decisão

Introduzir uma **camada de configuração em cascata** mantendo o tipo `config.Runtime` como modelo
único, resolvida por um novo `config.Resolver`:

1. **Fontes, em ordem crescente de precedência:**
   1. defaults built-in (`DefaultRuntime()`);
   2. config **global**: `~/.aispec/config.yaml` (resolvido via `os.UserHomeDir`, nunca hardcoded — R-SEC-001);
   3. config de **projeto**: arquivo mais próximo encontrado por **upward-walk** a partir do CWD,
      procurando `.aispec/config.yaml` → `.claude/config.yaml` → `.agents/config.yaml`;
   4. **overrides explícitos** (flags CLI), aplicados por último.
2. **Merge campo a campo** (não substituição de struct inteira): cada fonte só sobrescreve campos
   não-zero, preservando os já definidos por fontes de menor precedência.
3. **Upward-walk** caminha do CWD até a raiz do FS (ou até encontrar marcador de projeto: `.git/`,
   `.aispec/`, `.claude/` ou `.agents/`), parando no primeiro diretório com config de projeto.
4. **Compatibilidade:** sem config global e a partir da raiz do repo, o resultado é byte-idêntico
   ao `LoadRuntime` atual (RF-16). `LoadRuntime(repoRoot)` é preservado como wrapper fino sobre o
   Resolver para não quebrar chamadores existentes.
5. **Persistência inalterada:** evidências/memória continuam em arquivos versionáveis no projeto;
   `~/.aispec/` guarda **apenas configuração** (sem estado de execução, sem DB).

Formato **YAML** (não TOML do Compozy) para consistência com o `config.yaml` já existente no projeto.

## Alternativas Consideradas

- **Manter só config de projeto (status quo):** simples, mas falha RF-13/14/15 — sem reuso de
  defaults entre projetos e sem upward-walk.
- **Adotar TOML + seções por comando (paridade literal Compozy):** maior fidelidade, mas quebra o
  formato YAML atual, exige migração e introduz superfície maior que o PRD pede. Rejeitada por
  custo desproporcional nesta fase.
- **Variáveis de ambiente como camada global:** já existe projeção `EnvVars()`, mas env não é
  versionável nem descobrível; ruim para defaults persistentes do usuário. Rejeitada como fonte
  primária (mantida apenas como projeção de saída).

## Consequências

### Benefícios Esperados
- Defaults globais reutilizáveis entre projetos (multi-projeto sem fricção) — RF-13.
- Execução a partir de subdiretórios funciona via upward-walk — RF-14.
- Precedência previsível e testável — RF-15.
- Zero regressão para projetos atuais — RF-16.

### Trade-offs e Custos
- Nova superfície de leitura de FS (home dir) e lógica de merge — mais testes.
- Possível ambiguidade se múltiplos arquivos de projeto coexistirem; mitigado por ordem fixa de
  candidatos e parada no primeiro encontrado.

### Riscos e Mitigações
- **Risco:** home dir indisponível/CI sem `$HOME` → **Mitigação:** ausência de global é não-fatal
  (degrada para projeto+defaults); erro só se o arquivo existir e estiver malformado.
- **Risco:** upward-walk escapar do repositório → **Mitigação:** parar em marcador de projeto
  (`.git/`/`.aispec/`/`.claude/`/`.agents/`) e no boundary do FS.

## Plano de Implementação
1. Criar `config.Resolver` com `ResolveHomeConfigPath()` (via `os.UserHomeDir`) e `findProjectConfig(cwd)` (upward-walk).
2. Implementar merge campo-a-campo reusando `applyRuntimeDefaults`.
3. Reescrever `LoadRuntime` como wrapper sobre o Resolver (compat).
4. Cobrir com testes table-driven + FakeFileSystem (precedência, upward-walk, ausência de global).

## Monitoramento e Validação
- Teste determinístico de precedência (flags > projeto > global > default).
- Teste de upward-walk a partir de subdiretório profundo.
- Teste de regressão: sem global + raiz do repo == saída do `LoadRuntime` legado.

## Impacto em Documentação e Operação
- Atualizar `AGENTS.md`/`CLAUDE.md` (seção de config) e `docs/` com a hierarquia e a precedência.
- Documentar `~/.aispec/config.yaml` no guia de instalação.

## Revisão Futura
- Revisitar se/quando entrar o workspace registry multi-projeto (PRD futuro), que pode promover
  `~/.aispec/` a guardar índice de workspaces (ainda file-first).
