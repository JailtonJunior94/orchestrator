# Ciclo de Feedback de Telemetria

Este documento descreve como ativar, coletar, agregar e interpretar os dados de telemetria do harness para fechar o ciclo coleta → relatório → decisão.

---

## Ativação

A telemetria é opt-in. Para ativar, defina a variável de ambiente antes de usar o harness:

```bash
export GOVERNANCE_TELEMETRY=1
```

Sem essa variável, nenhum dado é escrito. Para ativar permanentemente no projeto, adicione ao `.envrc` ou ao script de setup do repositório.

---

## Cobertura por Tool

A partir de F1 (ADR-012 e ADR-013), os eventos de runtime ganham cardinalidade por ferramenta.
O subcomando `telemetry report` agrega por `tool`, permitindo comparar invocacoes de
Claude, Copilot e Codex no mesmo relatorio.

### Copilot ACP (ADR-012)

O evento `runtime_init` ganha cardinalidade `tool=copilot` quando o harness e invocado
com `--runtime=acp --tool=copilot`. O campo `launcher` distingue `binary` (binario
`copilot` local) de `npx` (fallback via npx). Os demais campos (`npm_version`,
`sdk_version`) sao preenchidos com metadados reais do `Spec` Copilot (nao constantes Claude).

Exemplo de linha no log com Copilot ACP:

```
2026-05-21T10:30:00Z skill=execute-task ref=security.md tool=copilot launcher=binary
```

### Codex ACP (ADR-013)

A partir de F1-Codex, o evento `runtime_init` tambem suporta `tool=codex` quando o
harness e invocado com `--runtime=acp --tool=codex`. Os campos registrados sao:

- `tool=codex` — identifica o runtime Codex ACP
- `launcher=binary|npx` — distingue `codex-acp` local de fallback `npx`
- `npm_version=0.14.0` — versao do adapter `@zed-industries/codex-acp`
- `sdk_version=v0.13.0` — versao do SDK ACP Go (go.mod)

Exemplo de linha no log com Codex ACP:

```
2026-05-21T10:30:00Z skill=execute-task ref=security.md tool=codex launcher=binary npm_version=0.14.0 sdk_version=v0.13.0
```

As invariantes de paridade multi-tool (ADR-008) cobrem `tool=codex` com os **mesmos
`kinds` de eventos** que cobrem Claude e Copilot (`runtime_init`, `tool_call`,
`session_end`, etc.). Nenhum novo `kind` de evento foi introduzido para Codex
(ADR-010 invariante preservada). Tool names Codex-nativos (`search_query`, `image_query`)
sao preservados ate F2-Codex implementar aliasing canonico.

### Gemini ACP (ADR-015)

A partir de F0-Gemini, o evento `runtime_init` suporta `tool=gemini` quando o harness e
invocado com `--runtime=acp --tool=gemini`. Os campos registrados sao:

- `tool=gemini` — identifica o runtime Gemini ACP
- `launcher=binary|npx` — distingue `gemini` local de fallback `npx @google/gemini-cli`
- `npm_version=0.43.0` — versao pinada do `@google/gemini-cli` (ADR-015 D-02)
- `sdk_version=v0.13.0` — versao do SDK ACP Go (go.mod, mesma de Claude/Codex/Copilot)

A partir de F4-Gemini, com `GOVERNANCE_TELEMETRY=1`, entradas adicionais Gemini-2026
sao acrescentadas ao log (aditivas — nao substituem entradas existentes):

- `gemini.cache_read=N` — tokens lidos do context cache Gemini (TTL configuravel)
- `gemini.effective_context=N` — tamanho real do contexto carregado na sessao
- `gemini.prompt_billed=N` — tokens efetivamente cobrados apos desconto de cache hit
- `gemini.thoughts=N` — tokens de reasoning interno Gemini 2.5 (pode ser zero por default)

As **invariantes de paridade multi-tool (ADR-008 e ADR-010) sao preservadas**: Gemini
emite os mesmos `kinds` de eventos que Claude/Copilot/Codex (`runtime_init`, `tool_call`,
`session_end`, `nested_agent`, etc.). Nenhum novo `kind` foi introduzido para Gemini.
Tool names Gemini-nativos sao normalizados via tabela `common` (F2-Gemini) sem alias
Gemini-especificos — Compozy confirma que os nomes emitidos pela CLI Gemini sao proximos
ao schema canonico.

Exemplo de linha no log com Gemini ACP:

```
2026-05-22T10:30:00Z skill=execute-task ref=security.md tool=gemini launcher=binary npm_version=0.43.0 sdk_version=v0.13.0
```

---

## Coleta

Cada invocação de skill que passa pela skill `agent-governance` registra uma linha em `.agents/telemetry.log`:

```
2026-04-20T10:30:00Z skill=bugfix ref=bug-schema.json
2026-04-20T10:31:00Z skill=review ref=security.md
2026-04-20T10:32:00Z skill=bugfix ref=testing.md
```

Formato: `<RFC3339> skill=<nome> ref=<nome>` — a linha `ref=` é omitida quando nenhuma referência foi carregada.

O arquivo é append-only e nunca truncado automaticamente.

---

## Agregação

Use o subcomando `telemetry report` para agregar os dados:

```bash
# Relatório completo (todo o histórico)
ai-spec-harness telemetry report

# Últimas 24 horas
ai-spec-harness telemetry report --since 24h

# Última semana
ai-spec-harness telemetry report --since 168h

# JSON estruturado (integrável com jq ou dashboards)
ai-spec-harness telemetry report --format json

# Filtrar e exportar
ai-spec-harness telemetry report --since 24h --format json | jq '.skills'
```

O subcomando legado `telemetry summary` ainda está disponível para contagens brutas.

---

## Relatório

Exemplo de saída do `telemetry report`:

```
Relatório de Telemetria
Período: últimas 24h
Total de invocações: 12

Skills Mais Usadas (top 5):
  1. bugfix                         5  (41.7%)
  2. review                         4  (33.3%)
  3. go-implementation              3  (25.0%)

Referências Mais Carregadas (top 5):
  1. testing.md                     6
  2. security.md                    4
  3. error-handling.md              2

Métricas:
  Refs por invocação (média): 1.0
  Tokens estimados:           6840 (12 refs × 570 tok/ref)

Alertas:
  ⚠ skill 'foo' invocada 2 vez(es) sem carregar nenhuma referência — possível bypass de governança
```

---

## Interpretação das Métricas

| Métrica | O que significa | Ação sugerida |
|---|---|---|
| **Skills mais usadas** | Quais skills geram mais valor operacional | Priorizar melhorias e documentação nas skills com maior adoção |
| **Refs mais carregadas** | Quais referências são mais consultadas | Verificar se referências muito acessadas estão atualizadas e bem estruturadas |
| **Taxa refs/invocação** | Média de arquivos carregados por chamada | Taxa alta (> 3) pode indicar skills "gordas" — candidatas a otimização de tokens |
| **Tokens estimados** | Custo operacional aproximado no período | Comparar entre períodos para avaliar tendência de consumo |
| **Alertas de bypass** | Skills invocadas sem carregar nenhuma referência | Investigar se o carregamento de governança está sendo pulado intencionalmente |

---

## Loop de Decisão

```
GOVERNANCE_TELEMETRY=1
        ↓
  uso normal de skills
        ↓
  .agents/telemetry.log acumula entradas
        ↓
  ai-spec-harness telemetry report --since 168h
        ↓
  análise de métricas:
    - skill mais usada → priorizar melhoria
    - taxa refs alta → candidata a otimização
    - alerta bypass → investigar governança
        ↓
  decisão de produto ou ajuste de skill
        ↓
  (novo ciclo)
```

---

## Limpeza do Log

O log não é rotacionado automaticamente. Para limpar manualmente:

```bash
# Ver tamanho
ls -lh .agents/telemetry.log

# Arquivar e reiniciar
mv .agents/telemetry.log .agents/telemetry-$(date +%Y%m%d).log.bak
```
