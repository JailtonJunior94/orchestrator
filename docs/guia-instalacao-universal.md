# Guia de Instalação Universal — ai-spec-harness

<!-- Gerado pela Tarefa 8.0 do PRD Fundação Portátil -->
<!-- Referências: ADR-016, ADR-017, ADR-018, ADR-019 -->

## Visão Geral

O `ai-spec-harness` instala-se em qualquer codebase em **menos de 30 segundos**, sem flags
obrigatórias. O instalador detecta automaticamente quais agentes de IA estão presentes no ambiente
(Claude, Codex, Gemini, Copilot) e materializa os assets de governança para cada um.

---

## Pré-requisitos

- macOS ou Linux (Ubuntu 20.04+)
- Go 1.26+ instalado (para build local) OU download do binário pré-compilado
- Pelo menos um agente de IA instalado no sistema:
  - **Claude Code**: `claude-agent-acp` ou `claude` no PATH
  - **Codex**: `codex-acp` ou `npx @zed-industries/codex-acp` disponível
  - **Gemini CLI**: `gemini` no PATH
  - **GitHub Copilot**: `copilot` ou `gh copilot` no PATH

---

## Instalação do Binário

### Via Homebrew (macOS/Linux recomendado)

```bash
brew install ai-spec-harness
```

### Via Go install

```bash
go install github.com/JailtonJunior94/ai-spec-harness/cmd/ai_spec_harness@latest
```

Após a instalação, confirme:

```bash
ai-spec-harness --version
```

---

## Bootstrap em um Novo Codebase

### Passo 1 — Instalar sem flags (auto-detecção)

A partir do diretório raiz do projeto:

```bash
ai-spec-harness install .
```

O instalador:
1. Detecta automaticamente os agentes presentes no sistema (`LookPath` + diretórios de config).
2. Materializa os assets de governança (skills, regras) para cada agente detectado.
3. Cria o manifesto `.ai_spec_harness.json` com checksums dos arquivos instalados.

Saída esperada (exemplo com Claude e Gemini detectados):

```
Agentes detectados: claude, gemini
Instalando skills para claude... OK
Instalando skills para gemini... OK
Manifesto atualizado: .ai_spec_harness.json
```

### Passo 2 — Verificar a instalação

```bash
ai-spec-harness verify .
```

Saída esperada — todos os itens `current`:

```
Resultado da verificacao:

  [claude]
    agent-governance                         current
    go-implementation                        current
  [gemini]
    agent-governance                         current
    go-implementation                        current

Resumo: 4 current, 0 missing, 0 drifted
```

### Passo 3 — Reexecutar (idempotência)

```bash
ai-spec-harness install .
ai-spec-harness verify .
```

Segunda execução converge para o mesmo estado — zero drift, 100% `current`.

---

## Escopo Global (`--global`)

Instala os assets de governança em `~/.aispec/` e nos diretórios globais por-agente
(`~/.claude/`, `~/.codex/`, `~/.gemini/`, `~/.config/github-copilot/`).

```bash
# Instalação global (afeta todos os projetos do usuário)
ai-spec-harness install --global

# Verificação global
ai-spec-harness verify --global
```

> **Nota:** o escopo global é opt-in. O comportamento default (sem `--global`) instala apenas
> no projeto corrente. Requer `$HOME` disponível; degrada com erro explícito em ambientes sem home.

---

## Seleção Manual de Ferramentas (`--tools`)

A flag `--tools` é **opcional**. Use-a para sobrescrever a detecção automática:

```bash
# Instalar apenas para Claude e Gemini (override da detecção)
ai-spec-harness install . --tools claude,gemini

# Instalar para todas as ferramentas suportadas
ai-spec-harness install . --tools all
```

Sem `--tools`, o instalador usa os agentes detectados automaticamente no ambiente.

---

## Modos de Instalação (`--mode`)

| Modo | Descrição | Default |
|------|-----------|---------|
| `symlink` | Cria links simbólicos para os assets embarcados (recomendado em Unix) | Sim |
| `copy` | Copia os arquivos (portátil; necessário quando symlinks não são suportados) | Não |

```bash
# Forçar cópia (ex.: sistemas de arquivos sem suporte a symlinks)
ai-spec-harness install . --mode copy
```

---

## Dry-run (Simulação)

Visualize o que seria criado sem executar:

```bash
ai-spec-harness install . --dry-run
```

---

## Atualização Após `git pull`

Após atualizar o harness ou o repositório de governança:

```bash
# Reinstalar (idempotente — converge sem duplicar)
ai-spec-harness install .

# Confirmar que nenhum arquivo driftou
ai-spec-harness verify .
```

Itens `drifted` indicam que o arquivo instalado divergiu do asset embarcado. Reinstale para
convergir.

---

## Estados de Verificação

| Estado | Significado |
|--------|-------------|
| `current` | Arquivo instalado corresponde ao asset embarcado |
| `missing` | Arquivo esperado não está presente |
| `drifted` | Arquivo presente mas com conteúdo diferente do asset embarcado |

---

## Casos de Borda

### Nenhum agente detectado

```
Nenhum agente detectado no ambiente. Use --tools para especificar explicitamente.
```

O instalador **não falha silenciosamente** — reporta claramente e retorna código de saída
apropriado.

### `$HOME` indisponível (CI sem home dir)

O escopo global degrada com erro explícito:

```
erro: $HOME não disponível (necessário para escopo global)
```

O escopo de projeto (`--global` ausente) permanece funcional.

### Subdiretório profundo

A partir de qualquer subdiretório do projeto, a configuração do workspace é encontrada via
upward-walk (veja [Hierarquia de Configuração](config-hierarchy.md)):

```bash
cd deep/nested/subdir
ai-spec-harness install .   # encontra a config do projeto via upward-walk
```

### Fallback launcher (binário direto ausente)

Quando o binário ACP direto não está no PATH, o harness tenta os launchers alternativos
configurados (ex.: `npx @zed-industries/codex-acp`). O fallback é **transparente** — o resultado
é idêntico ao do binário direto. Ver ADR-017.

---

## Integração em CI/CD

```yaml
# .github/workflows/install-governance.yml
- name: Install ai-spec-harness
  run: |
    go install github.com/JailtonJunior94/ai-spec-harness/cmd/ai_spec_harness@latest

- name: Bootstrap governance
  run: ai-spec-harness install . --tools claude,gemini --mode copy

- name: Verify governance
  run: ai-spec-harness verify .
```

---

## Referências

- [Hierarquia de Configuração](config-hierarchy.md) — precedência `flags > workspace > global > built-in`
- [ADR-016](../.specs/prd-fundacao-portatil/adr-016-config-hierarquico-universal.md) — Config hierárquico universal
- [ADR-017](../.specs/prd-fundacao-portatil/adr-017-fallback-launcher-chain.md) — Fallback launcher chain
- [ADR-018](../.specs/prd-fundacao-portatil/adr-018-runtimeconfig-retry-backpressure.md) — RuntimeConfig + retry/backpressure
- [ADR-019](../.specs/prd-fundacao-portatil/adr-019-instalador-portatil-detect-verify.md) — Instalador portátil
- [Troubleshooting](troubleshooting.md) — Problemas comuns
