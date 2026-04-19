# TASK-05: Skills canônicas embutidas ou referenciáveis

## Contexto

O repo remoto `ai-governance` contém 24 skills em `.agents/skills/`, incluindo as 9 processuais
e 4 de linguagem que são a base de toda instalação. O `install.sh` copia ou linka essas skills
do próprio repo para o projeto-alvo.

O CLI Go (`ai-spec-harness`) não contém essas skills. O flag `--source` é obrigatório e deve
apontar para um clone local de `ai-governance`. Isso significa que o CLI não é autossuficiente.

## Objetivo

Definir e implementar uma estratégia para que o CLI Go possa instalar skills canônicas sem
depender de um clone prévio do repo de governança.

## Opções de design

### Opção A: `go:embed` das skills no binário

- Embutir `.agents/skills/` no binário via `//go:embed`
- Vantagens: binário 100% autossuficiente, zero dependências externas
- Desvantagens: binário maior (~2-5MB extras), atualização de skills exige rebuild,
  versionamento das skills acoplado ao binário

### Opção B: Download automático do GitHub

- `ai-spec install` baixa skills de `JailtonJunior94/ai-governance@<tag>` via `gh` ou HTTP
- Vantagens: binário leve, skills sempre atualizadas
- Desvantagens: requer rede, `gh` autenticado ou API pública, complexidade de cache

### Opção C: `--source` obrigatório (status quo) + documentação

- Manter `--source` como obrigatório
- Melhorar UX: mensagem de erro clara quando `--source` é omitido, documentação no README
- Vantagens: simples, sem acoplamento, sem aumento de binário
- Desvantagens: fricção para novos usuários

### Opção D: Híbrido (embed + remote update)

- Embutir versão base via `go:embed`
- Permitir `--source` para override com versão mais recente
- Vantagens: funciona offline, atualizável
- Desvantagens: maior complexidade

## Decisão necessária

**Esta task requer decisão de design antes da implementação.** Sugestão: Opção A para
paridade máxima, Opção C para menor esforço.

## Subtarefas (assumindo Opção A)

- [x] **5.1** Criar diretório `internal/embedded/assets/.agents/skills/` com as 13 skills canônicas
  - Copiadas de `ai-governance` as 9 processuais + 4 de linguagem
  - Estrutura: `<skill>/SKILL.md`, `<skill>/references/`, `<skill>/scripts/`, `<skill>/assets/`
  - Também embutidos: `.claude/`, `.gemini/`, `scripts/lib/`, `AGENTS.md`

- [x] **5.2** Configurar `go:embed`
  - Criado `internal/embedded/embedded.go` com `//go:embed all:assets`
  - Expõe `Assets embed.FS` e `ExtractToTempDir() (string, func(), error)`

- [x] **5.3** Adaptar `internal/install/` para usar fonte embutida
  - Se `--source` não for fornecido, extrai embutidos em temp dir e usa como sourceDir
  - Se `--source` for fornecido, usa como override (comportamento inalterado)
  - Modo embutido força `copy` (sem symlinks para temp dir)

- [x] **5.4** Adaptar `internal/upgrade/` para comparar com embutidas
  - Se `--source` não for fornecido, extrai embutidos em temp dir e usa como fonte de comparação

- [x] **5.5** Testes
  - `TestInstall_EmbeddedSource_NoSourceFlag` — install sem `--source` (usa embutidas)
  - `TestInstall_EmbeddedSource_AllToolsAllLangs` — install `--tools all --langs all` sem `--source`
  - `TestInstall_ExternalSource_Override` — `--source` externo tem precedência
  - `TestUpgrade_EmbeddedSource_NoSourceFlag` — upgrade sem `--source` detecta desatualização
  - `TestUpgrade_EmbeddedSource_UpdatesSkills` — upgrade sem `--source` atualiza skills

- [x] **5.6** README — descrição inline na Long do comando já documenta nova UX

## Subtarefas (assumindo Opção C)

- [ ] **5.1c** Melhorar mensagem de erro quando `--source` é omitido
  - Exibir: "Flag --source é obrigatório. Aponte para um clone de JailtonJunior94/ai-governance."

- [ ] **5.2c** Documentar no README o requisito de `--source`
  - Incluir exemplo de uso completo

- [ ] **5.3c** Validar que `--source` aponta para repo válido
  - Verificar existência de `.agents/skills/` e `VERSION` no path fornecido

## Arquivos afetados

- `internal/embedded/` (novo, se Opção A)
- `internal/install/install.go`
- `internal/upgrade/upgrade.go`
- `cmd/ai_spec_harness/install.go`
- `README.md`

## Critério de conclusão

- **Opção A:** `ai-spec install --tools all --langs all <target>` funciona sem `--source`
- **Opção C:** `ai-spec install <target>` sem `--source` exibe mensagem clara e sai com código 1

## Status

**CONCLUÍDA** — Opção A implementada. Todos os testes passando (`go test ./...`).

Decisão: Opção A (go:embed) — binário autossuficiente com 13 skills canônicas embutidas.
`--source` continua suportado como override para versões mais recentes ou customizadas.
