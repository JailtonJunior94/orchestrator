# Especificação Técnica — Telemetria: Feedback Loop Acionável

**PRD:** `tasks/prd-telemetry-feedback/prd.md`
**Data:** 2026-04-20
**Status:** Pronta para implementação
**ADRs:** [ADR-001 — Separação Report vs Summary](adr-001-report-vs-summary.md)

---

## Resumo Executivo

A feature adiciona `ai-spec-harness telemetry report` como novo subcomando ao grupo `telemetry` existente. O subcomando lê `.agents/telemetry.log`, computa métricas acionáveis (skills mais usadas, taxa de referências por invocação, estimativa de tokens, alertas) e emite saída formatada em texto ou JSON.

A implementação cria `internal/telemetry/report.go` com função `Report(rootDir string, since time.Duration) (ReportData, error)` e os tipos de domínio associados. O comando CLI (`cmd/ai_spec_harness/telemetry.go`) adiciona o subcomando `report` com flags `--since` e `--format`. Nenhuma dependência externa nova é introduzida.

---

## Arquitetura do Sistema

### Visão Geral dos Componentes

```
cmd/ai_spec_harness/telemetry.go   ← modificado: adiciona telemetryReportCmd
internal/telemetry/
  telemetry.go                     ← existente, inalterado
  summary.go                       ← existente, inalterado (retrocompatibilidade)
  report.go                        ← novo: Report(), ReportData, SkillMetric, RefMetric
  report_test.go                   ← novo: testes table-driven
docs/
  telemetry-feedback-cycle.md      ← novo: documentação do ciclo
CLAUDE.md                          ← modificado: referência à documentação
```

**Fluxo de dados:**
```
.agents/telemetry.log
      ↓ (bufio.Scanner — resiliente a linhas malformadas)
internal/telemetry.Report()
      ↓ (ReportData struct)
cmd/telemetryReportCmd.RunE()
      ↓ (--format text → texto formatado via output.Printer)
      ↓ (--format json → encoding/json.Marshal → stdout)
```

---

## Design de Implementação

### Interfaces Chave

Sem nova interface pública. `Report` é uma função pura (entrada: rootDir + since; saída: ReportData + error). Isso é consistente com o padrão de `Summary` no mesmo pacote.

### Modelos de Dados

```go
// ReportData contém as métricas acionáveis derivadas do log de telemetria.
// Exportado para permitir serialização JSON e testes de asserção estruturada.
type ReportData struct {
    Period            string        `json:"period"`             // "all" ou duração formatada
    TotalInvocations  int           `json:"total_invocations"`  // linhas válidas no período
    Skills            []SkillMetric `json:"skills"`             // top 5 por contagem, desc
    Refs              []RefMetric   `json:"refs"`               // top 5 por contagem, desc
    EstimatedTokens   int           `json:"estimated_tokens"`   // totalRefLoads × tokensPerRefLoad
    RefsPerInvocation float64       `json:"refs_per_invocation"` // média de refs por invocação
    Alerts            []string      `json:"alerts"`             // avisos acionáveis
}

type SkillMetric struct {
    Name       string  `json:"name"`
    Count      int     `json:"count"`
    Percentage float64 `json:"percentage"` // Count / TotalInvocations × 100
}

type RefMetric struct {
    Name  string `json:"name"`
    Count int    `json:"count"`
}
```

### Assinatura da função principal

```go
// Report lê .agents/telemetry.log, aplica filtro de período e retorna métricas acionáveis.
// Linhas malformadas são ignoradas sem erro. Log ausente retorna ReportData zero-value.
func Report(rootDir string, since time.Duration) (ReportData, error)
```

### Lógica de Alertas (RF-02.4)

Regra única na v1: se uma skill foi invocada mas nenhuma referência foi carregada em nenhuma das suas invocações, adicionar alerta:
```
"skill 'X' invocada N vezes sem carregar nenhuma referência — possível bypass de governança"
```

### Adição no CLI (`cmd/ai_spec_harness/telemetry.go`)

```go
var (
    reportSince  string
    reportFormat string // "text" | "json"
)

var telemetryReportCmd = &cobra.Command{
    Use:   "report",
    Short: "Exibe relatório acionável de telemetria com métricas e alertas",
    RunE: func(cmd *cobra.Command, args []string) error {
        var since time.Duration
        if reportSince != "" {
            d, err := time.ParseDuration(reportSince)
            if err != nil {
                return fmt.Errorf("duração inválida %q: %w", reportSince, err)
            }
            since = d
        }
        data, err := telemetry.Report(".", since)
        if err != nil {
            return err
        }
        switch reportFormat {
        case "json":
            return json.NewEncoder(os.Stdout).Encode(data)
        default:
            printReport(p, data) // função auxiliar local, usa output.Printer
            return nil
        }
    },
}

func init() {
    telemetryReportCmd.Flags().StringVar(&reportSince, "since", "", "Filtrar por período (ex: 24h, 168h)")
    telemetryReportCmd.Flags().StringVar(&reportFormat, "format", "text", "Formato de saída: text ou json")
    telemetryCmd.AddCommand(telemetryReportCmd)
}
```

**Nota sobre `output.Printer`:** O `Printer` existente não suporta injeção no cmd layer (os comandos usam `fmt.Print` diretamente). Para manter consistência com os outros comandos existentes (`summary`, `log`), `printReport` usará `fmt.Fprintf(os.Stdout, ...)` diretamente, sem injetar `Printer`. Mudança estrutural de DI no cmd layer está fora de escopo.

### Formato de saída texto (`printReport`)

```
Relatório de Telemetria
Período: <all | últimas Xh>
Total de invocações: N

Skills Mais Usadas (top 5):
  1. <skill>  N  (XX.X%)
  ...

Referências Mais Carregadas (top 5):
  1. <ref>  N
  ...

Métricas:
  Refs por invocação (média): X.X
  Tokens estimados:           N (N refs × 570 tok/ref)

Alertas:
  ⚠ <mensagem>
```

---

## Pontos de Integração

Nenhuma integração externa. Toda leitura é do filesystem local (`.agents/telemetry.log`).

---

## Abordagem de Testes

### Testes Unitários (`internal/telemetry/report_test.go`)

Usar `t.TempDir()` e escrever `.agents/telemetry.log` com conteúdo controlado. Tabela obrigatória:

| Caso | Entrada | Expectativa |
|---|---|---|
| `log_ausente` | diretório sem log | `ReportData{}` zero-value, err=nil |
| `sem_dados_no_periodo` | log com entradas antigas, `since=1h` | `TotalInvocations=0`, err=nil |
| `agregacao_correta` | 3 entradas bugfix + 1 review, 2 refs bug.md, 1 schema.json | Skills top: bugfix=3 (75%), Skills: review=1 (25%), Refs: bug.md=2 |
| `alerta_skill_sem_ref` | `skill=foo` sem nenhum `ref=` | `Alerts` contém aviso de bypass para "foo" |
| `top5_trunca_lista` | 7 skills diferentes | `Skills` tem exatamente 5 entradas |
| `json_valido` | dados normais | `json.Unmarshal(json.Marshal(data))` sem erro |
| `linha_malformada_ignorada` | log com linhas inválidas intercaladas | não falha, conta apenas linhas válidas |

**Cobertura esperada de `report.go`:** ≥ 85% (sem IO de filesystem externo nos testes).

### Testes de Integração

Não necessários para esta feature: a lógica de IO é simples (`os.Open` + `bufio.Scanner`), já coberta por `t.TempDir()` nos testes unitários. Os critérios da template (fronteiras de IO críticas, incidente anterior) não se aplicam.

### Testes E2E

Não necessários. CLI testado indiretamente via `internal/telemetry/report_test.go`.

---

## Sequenciamento de Desenvolvimento

1. **`internal/telemetry/report.go`** — lógica pura, sem dependências de cobra/output. Primeiro porque é o núcleo testável.
2. **`internal/telemetry/report_test.go`** — testes table-driven, `t.TempDir()`. Validar cobertura.
3. **`cmd/ai_spec_harness/telemetry.go`** — adicionar `telemetryReportCmd`, flags `--since` e `--format`.
4. **`docs/telemetry-feedback-cycle.md`** — documentação do ciclo.
5. **`CLAUDE.md`** — adicionar referência ao doc.

### Dependências Técnicas

- `encoding/json` — stdlib, já disponível.
- `sort` — stdlib, já usado em `summary.go`.
- Nenhuma nova dependência em `go.mod`.

---

## Monitoramento e Observabilidade

- O próprio comando `report` é o mecanismo de observabilidade. Não há métricas a expor em formato Prometheus para esta feature.
- Saída `--format json` permite integração com `jq` e pipelines externos.

---

## Considerações Técnicas

### Decisões Chave

Ver [ADR-001 — Separação Report vs Summary](adr-001-report-vs-summary.md).

**Outras decisões inline:**

| Decisão | Escolha | Justificativa |
|---|---|---|
| Top N = 5 | Hard-coded | Sem evidência de necessidade de configurabilidade; YAGNI |
| Duração `7d` não suportada | Aceitar apenas `time.ParseDuration` | Evita parser custom; `168h` é documentado como equivalente |
| `Printer` não injetado no cmd | `fmt.Fprintf(os.Stdout)` | Consistente com os outros subcomandos existentes; DI no cmd layer é refactor fora de escopo |
| RF-02.5 (tendência) | Não implementado na v1 | Exigiria segunda passagem de parse para janela anterior; custo > benefício agora |

### Riscos Conhecidos

| Risco | Impacto | Mitigação |
|---|---|---|
| Log muito grande (>100k linhas) | Lentidão na leitura | `bufio.Scanner` é stream-based; sem carga total em memória |
| Linha com timestamp futuro | Pode escapar filtros `since` | Aceitar como válido; dados do futuro são improvável em uso normal |
| Regressão em `summary` existente | Quebra retrocompatibilidade | `summary.go` não é modificado |

### Conformidade com Padrões

- **R-GOV-001** — decisões documentadas neste spec e em ADR.
- **R-DDD-001** — `ReportData` é um VO simples; sem domínio rico necessário para contagens de log.
- Convenções do projeto: erros em PT-BR, `fmt.Errorf("contexto: %w", err)`, `t.TempDir()` em testes.

### Arquivos Relevantes e Dependentes

| Arquivo | Mudança |
|---|---|
| `internal/telemetry/telemetry.go` | Inalterado |
| `internal/telemetry/summary.go` | Inalterado |
| `internal/telemetry/report.go` | **Novo** |
| `internal/telemetry/report_test.go` | **Novo** |
| `cmd/ai_spec_harness/telemetry.go` | Modificado: adiciona `telemetryReportCmd` |
| `docs/telemetry-feedback-cycle.md` | **Novo** |
| `CLAUDE.md` | Modificado: referência ao doc |
